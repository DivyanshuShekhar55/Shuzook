package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

/*
we can create global vars for the meter instruments, but with the struct we can group them cleanly

otel would collect data for latencies and auto group them into buckets
so it would look like 2 reqs done in 50ms, 5 reqs in 120ms, ...
tools like grafana or click-stack would auto calculate the percentile values (p50, p90, etc)

otel also groups by request attributes (route, httpStatus, etc)
these attrubutes are added in the finishRequest() function
*/

type apiMetrics struct {
	requestsTotal metric.Int64Counter
	inflightNow   atomic.Int64 // in-progress requests
	inflightAsync metric.Int64ObservableUpDownCounter
	latencyMs     metric.Float64Histogram
}

func newAPIMetrics() (*apiMetrics, error) {
	meter := otel.Meter("shazam/cmd/api/metrics")
	m := &apiMetrics{}

	// initialise the various meter instruments we want to use
	var err error
	m.requestsTotal, err = meter.Int64Counter(
		"shazam.http.requests.total",
		metric.WithDescription("Counts how many metric demo requests we handled."),
	)
	if err != nil {
		return nil, err
	}

	m.inflightAsync, err = meter.Int64ObservableUpDownCounter(
		"shazam.http.requests.inflight",
		metric.WithDescription("Tracks how many requests are active right now."),
	)
	if err != nil {
		return nil, err
	}

	m.latencyMs, err = meter.Float64Histogram(
		"shazam.http.request.duration.ms",
		metric.WithDescription("Request duration in milliseconds for demo routes."),
	)
	if err != nil {
		return nil, err
	}

	// This callback runs whenever the metrics reader scrapes/collects data.
	// We publish the latest inflight value into the async up-down counter.
	_, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		observer.ObserveInt64(m.inflightAsync, m.inflightNow.Load())
		return nil
	}, m.inflightAsync)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *apiMetrics) registerMetricsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/metrics/counter", m.counterRoute)
	mux.HandleFunc("/metrics/async-updown", m.asyncUpDownRoute)
	mux.HandleFunc("/metrics/histogram", m.histogramRoute)
}

func (app *application) registerMetricsRoutes(mux *http.ServeMux) {
	if app.metrics == nil {
		return
	}
	app.metrics.registerMetricsRoutes(mux)
}

func (m *apiMetrics) counterRoute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	m.inflightNow.Add(1)
	defer m.finishRequest(r, "/metrics/counter", http.StatusOK, start)

	// Easy idea: counter goes up by 1 for each call.
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "counter route hit: one request was counted")
}

func (m *apiMetrics) asyncUpDownRoute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	m.inflightNow.Add(1)
	defer m.finishRequest(r, "/metrics/async-updown", http.StatusOK, start)

	// Hold the request a little for demonstration purpose
	time.Sleep(3500 * time.Millisecond)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "async up-down route done; inflight now=%d", m.inflightNow.Load())
}

func (m *apiMetrics) histogramRoute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	m.inflightNow.Add(1)
	defer m.finishRequest(r, "/metrics/histogram", http.StatusOK, start)

	// We simulate latency with a random sleep so histogram receives varied values.
	delayMs := 50 + rand.Intn(450)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "histogram recorded a request duration around %dms", delayMs)
}

func (m *apiMetrics) finishRequest(r *http.Request, route string, status int, start time.Time) {
	durationMs := float64(time.Since(start)) / float64(time.Millisecond)
	attrs := []attribute.KeyValue{
		attribute.String("route", route),
		attribute.String("method", r.Method),
		attribute.Int("http.status_code", status),
	}

	m.requestsTotal.Add(r.Context(), 1, metric.WithAttributes(attrs...))
	m.latencyMs.Record(r.Context(), durationMs, metric.WithAttributes(attrs...))
	m.inflightNow.Add(-1)
}

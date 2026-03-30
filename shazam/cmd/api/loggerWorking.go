package main

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)


/*
	for showing loggin we use two simple routes
	both would show info llike ISO timestamps, git commit hash, trace and span id, etc
	the first route shows a normal trace
	second route shows trace and span data of the span we created manually
*/
func (app *application) registerLoggerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/logger/auto", app.loggerAuto)
	mux.HandleFunc("/logger/child", app.loggerChild)
}

func (app *application) loggerAuto(w http.ResponseWriter, r *http.Request) {
	// This route relies on otelhttp middleware's automatic server span
	// so the span's ctx would simply be request's context
	app.logInfoCtx(
		r.Context(),
		"auto logger route hit",
		zap.String("route", "/logger/auto"),
	)

	// simulate some work with a sleep
	time.Sleep(150 * time.Millisecond)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "auto logger route completed")
}

func (app *application) loggerChild(w http.ResponseWriter, r *http.Request) {
	// create a span manually with tracer.Start()
	ctx, span := tracer.Start(r.Context(), "logger.child.work")
	defer span.End()

	// span's ctx would be the ctx we created above
	app.logInfoCtx(
		ctx,
		"child logger route hit",
		zap.String("route", "/logger/child"),
	)

	// simulate the work/computation here
	time.Sleep(200 * time.Millisecond)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "child logger route completed")
}


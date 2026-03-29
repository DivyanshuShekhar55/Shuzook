package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/DivyanshuShekhar55/Shuzook/internals/otel"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type application struct {
	config config
}

type config struct {
	addr string
	env  string
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	var cfg config

	flag.StringVar(&cfg.addr, "addr", envOrDefault("SHAZOOK_ADDR", ":8080"), "API server address")
	flag.StringVar(&cfg.env, "env", envOrDefault("SHAZOOK_ENV", "development"), "Environment name")
	flag.Parse()

	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app := &application{config: cfg}

	// Set up OpenTelemetry.
	otelShutdown, err := otel.SetupOTelSDK(ctx)
	if err != nil {
		return err
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Start HTTP server.
	srv := &http.Server{
		Addr:         cfg.addr,
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  30 * time.Second,
		Handler:      app.routes(),
	}
	srvErr := make(chan error, 1)
	go func() {
		log.Printf("starting %s server on %s", cfg.env, cfg.addr)
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption or startup/runtime errors.
	select {
	case err = <-srvErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(shutdownCtx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/auto", AutoSpan)
	mux.HandleFunc("/manual", SomeLogic)
	mux.HandleFunc("/child", ChildLogic)
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}

func envOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	return v
}

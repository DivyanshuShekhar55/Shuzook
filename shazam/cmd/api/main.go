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
	"go.uber.org/zap"
)

/*
Why we ignore tracer in the application struct ?
In logging : there is setup cost — file handles, buffers, the OTel bridge core — so you genuinely want to do it once and share it.
Other packages that need logging would receive the logger via dependency injection
same with metrics, create it once, pass around in packages

in tracerWorking.go (main package) : var tracer = otel.Tracer("myapp/main"), i.e., we put a name to tracers, which helps us identify the package the trace belongs to
So we create the "tracer" variable per package usually
otel.Tracer(...) is NOT creating a new tracer provider. The provider was already created once in otel.go and registered globally. `otel.Tracer(...)` just gets a lightweight named handle from that provider. Low cost action
*/
type application struct {
	config  config
	logger  *zap.Logger
	metrics *apiMetrics
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

	// Set up OpenTelemetry.
	otelShutdown, err := otel.SetupOTelSDK(ctx)
	if err != nil {
		return err
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	logger, err := newZapLogger()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, logger.Sync())
	}()

	app := &application{config: cfg, logger: logger}

	// init the metrics and attach to the application struct instance
	metrics, err := newAPIMetrics()
	if err != nil {
		return err
	}
	app.metrics = metrics

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
		app.logger.Info(
			"starting server",
			zap.String("env", cfg.env),
			zap.String("addr", cfg.addr),
		)
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
	app.registerTracingRoutes(mux)
	app.registerLoggerRoutes(mux)
	app.registerMetricsRoutes(mux)
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

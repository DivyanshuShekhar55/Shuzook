package main

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Set this at build time with:
// go build -ldflags "-X main.gitCommitHash=$(git rev-parse --short HEAD)"
var gitCommitHash = "dev"

func newZapLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json" // not required cause its already in json, but just for showing
	cfg.OutputPaths = []string{"stdout"} // swap with external db or stdout
	cfg.ErrorOutputPaths = []string{"stderr"} //swap with external db or stderr
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // show iso time instead of the normal millisecond one

	logger, err := cfg.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	commit := envOrDefault("GIT_COMMIT_HASH", gitCommitHash)
	if commit == "" {
		commit = "unknown"
	}

	return logger.With(zap.String("git_commit", commit)), nil
}


/*
	when otel creates a span it stores it in context, we pull some metadata out from that context
	so all logs automatically get there trace and span id
	the following function implements that
*/
func traceFieldsFromContext(ctx context.Context) []zap.Field {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}

	return []zap.Field{
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	}
}

/*
	the following function is used to show the trace and span data in the log in addition to already exisiting things
	take the context and extract the trace and span from it
*/
func (app *application) logInfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if app.logger == nil {
		return
	}

	allFields := append(fields, traceFieldsFromContext(ctx)...)
	app.logger.Info(msg, allFields...)
}

func (app *application) logErrorCtx(ctx context.Context, msg string, err error, fields ...zap.Field) {
	if app.logger == nil {
		return
	}

	allFields := append(fields, zap.Error(err))
	allFields = append(allFields, traceFieldsFromContext(ctx)...)
	app.logger.Error(msg, allFields...)
}

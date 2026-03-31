package main

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Set this at build time with:
// go build -ldflags "-X main.gitCommitHash=$(git rev-parse --short HEAD)"
var gitCommitHash = "dev"

func newZapLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)
	jsonCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), cfg.Level)
	otelCore := otelzap.NewCore(
		"github.com/DivyanshuShekhar55/Shuzook/cmd/api",
		otelzap.WithVersion("0.1.0"),
		otelzap.WithLoggerProvider(otellogglobal.GetLoggerProvider()),
		otelzap.WithAttributes(attribute.String("git.commit", envOrDefault("GIT_COMMIT_HASH", gitCommitHash))),
	)

	core := zapcore.NewTee(jsonCore, otelCore)
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

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
	allFields = append(allFields, zap.Any("context", ctx))
	app.logger.Info(msg, allFields...)
}

func (app *application) logErrorCtx(ctx context.Context, msg string, err error, fields ...zap.Field) {
	if app.logger == nil {
		return
	}

	allFields := append(fields, zap.Error(err))
	allFields = append(allFields, traceFieldsFromContext(ctx)...)
	allFields = append(allFields, zap.Any("context", ctx))
	app.logger.Error(msg, allFields...)
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Set this at build time with:
// go build -ldflags "-X main.gitCommitHash=$(git rev-parse --short HEAD)"
var gitCommitHash = "dev"

/*
zap is the logging library, and we can "inject" other libraries into it
this causes zap change logging behaviour
we start with a core setup, core is the engine running zap
we customise zap with our own fields, then plug-in the lumberjack package for writing logs to a log file
we write logic for log rotation with lumberjack
we plug-in otelzap bridge for changing logs from zap to otel standard
*/

func newZapLogger() (*zap.Logger, error) {

	// customise to zap
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// setup lumberjack
	logFile := envOrDefault("SHAZOOK_LOG_FILE", filepath.Join("logs", "shazam.log"))
	// MkdirAll: This is a "create if not exists" command, creates any parent folder if doesn't exist
	// 0o755: These are Unix permissions. It means the Owner can read/write/execute, while others can only read/execute. This is standard for log directories.
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	maxSizeMB := intFromEnv("SHAZOOK_LOG_MAX_SIZE_MB", 20)
	maxBackups := intFromEnv("SHAZOOK_LOG_MAX_BACKUPS", 7)
	maxAgeDays := intFromEnv("SHAZOOK_LOG_MAX_AGE_DAYS", 14)

	rotatingFile := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		Compress:   true, // compresses to gzip, reduces size by almost 90%
	}

	// the plug-in process
	encoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)
	jsonCore := zapcore.NewCore(encoder, zapcore.AddSync(rotatingFile), cfg.Level)
	otelCore := otelzap.NewCore(
		"github.com/DivyanshuShekhar55/Shuzook/cmd/api",
		otelzap.WithVersion("0.1.0"),
		otelzap.WithLoggerProvider(otellogglobal.GetLoggerProvider()),
		// custom fields to otel
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

func intFromEnv(key string, fallback int) int {
	v := envOrDefault(key, "")
	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
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

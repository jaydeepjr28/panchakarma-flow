package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

var defaultLogger *slog.Logger

func init() {
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(defaultLogger)
}

// Setup initializes the global structured logger with the specified level and format.
func Setup(levelStr string, isDevelopment bool) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if isDevelopment {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
	return defaultLogger
}

// WithContext returns a logger annotated with trace_id from the context.
func WithContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		return defaultLogger.With(slog.String("trace_id", traceID))
	}
	return defaultLogger
}

// ContextWithTraceID returns a new context containing the trace_id.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

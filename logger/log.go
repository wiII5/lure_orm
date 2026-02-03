package logger

import (
	"context"
	"log/slog"
)

// Logger provides query logging for lure_orm operations.
type Logger struct {
	cfg *Config
}

// New creates a new Logger with the given options.
func New(opts ...Option) *Logger {
	return &Logger{
		cfg: newConfig(opts...),
	}
}

// Read logs a read query if the log level allows it.
func (l *Logger) Read(ctx context.Context, format string, args ...any) {
	if l.cfg.level.allowRead() {
		l.log(ctx, slog.LevelInfo, format, args...)
	}
}

// Write logs a write query if the log level allows it.
func (l *Logger) Write(ctx context.Context, format string, args ...any) {
	if l.cfg.level.allowWrite() {
		l.log(ctx, slog.LevelInfo, format, args...)
	}
}

// Error logs an error.
func (l *Logger) Error(ctx context.Context, err error, format string, args ...any) {
	attrs := l.buildAttrs()
	attrs = append(attrs, slog.Any("error", err))
	slog.LogAttrs(ctx, slog.LevelError, fmt(format, args...), attrs...)
}

func (l *Logger) log(ctx context.Context, level slog.Level, format string, args ...any) {
	attrs := l.buildAttrs()
	slog.LogAttrs(ctx, level, fmt(format, args...), attrs...)
}

func (l *Logger) buildAttrs() []slog.Attr {
	attrs := make([]slog.Attr, 0, len(l.cfg.fields))
	for k, v := range l.cfg.fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

func fmt(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	// Simple formatting without importing fmt to keep it lightweight
	// For complex formatting, callers should pre-format their messages
	return format
}

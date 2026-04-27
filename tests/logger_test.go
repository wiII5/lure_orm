package tests

import (
	"context"
	"testing"

	"github.com/wiII5/lure_orm/logger"
)

func TestLoggerNew(t *testing.T) {
	log := logger.New()
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestLoggerWithOptions(t *testing.T) {
	log := logger.New(
		logger.WithLogLevel(logger.LogLevelRead),
		logger.WithFields(map[string]any{"service": "test"}),
	)
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestLogLevelNone(t *testing.T) {
	log := logger.New(logger.WithLogLevel(logger.LogLevelNone))
	ctx := context.Background()
	// These should not panic even with LogLevelNone
	log.Read(ctx, "test read query")
	log.Write(ctx, "test write query")
}

func TestLogLevelRead(t *testing.T) {
	log := logger.New(logger.WithLogLevel(logger.LogLevelRead))
	ctx := context.Background()
	// Should log read queries only
	log.Read(ctx, "SELECT * FROM Users")
	log.Write(ctx, "INSERT INTO Users") // Should be filtered out
}

func TestLogLevelWrite(t *testing.T) {
	log := logger.New(logger.WithLogLevel(logger.LogLevelWrite))
	ctx := context.Background()
	// Should log write queries only
	log.Read(ctx, "SELECT * FROM Users") // Should be filtered out
	log.Write(ctx, "INSERT INTO Users")
}

func TestLogLevelAll(t *testing.T) {
	log := logger.New(logger.WithLogLevel(logger.LogLevelAll))
	ctx := context.Background()
	// Should log both
	log.Read(ctx, "SELECT * FROM Users")
	log.Write(ctx, "INSERT INTO Users")
}

func TestLoggerError(t *testing.T) {
	log := logger.New()
	ctx := context.Background()
	err := context.DeadlineExceeded
	// Should not panic
	log.Error(ctx, err, "operation failed")
}

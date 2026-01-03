// Package logging provides implementations of the ports.Logger interface.
// It includes a NopLogger for disabled logging and a ConsoleLogger for
// structured console output in text or JSON format.
package logging

import (
	"context"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// NopLogger is a no-op logger that discards all messages.
// Useful when logging is disabled or as a default.
type NopLogger struct {
	level ports.Level
}

// NewNopLogger creates a new no-op logger.
func NewNopLogger() *NopLogger {
	return &NopLogger{level: ports.LevelInfo}
}

// Debug does nothing.
func (l *NopLogger) Debug(_ context.Context, _ string, _ ...ports.Field) {}

// Info does nothing.
func (l *NopLogger) Info(_ context.Context, _ string, _ ...ports.Field) {}

// Warn does nothing.
func (l *NopLogger) Warn(_ context.Context, _ string, _ ...ports.Field) {}

// Error does nothing.
func (l *NopLogger) Error(_ context.Context, _ string, _ ...ports.Field) {}

// With returns itself (no-op has no fields to add).
func (l *NopLogger) With(_ ...ports.Field) ports.Logger {
	return l
}

// Level returns the log level.
func (l *NopLogger) Level() ports.Level {
	return l.level
}

// SetLevel sets the log level.
func (l *NopLogger) SetLevel(level ports.Level) {
	l.level = level
}

// Ensure NopLogger implements Logger.
var _ ports.Logger = (*NopLogger)(nil)

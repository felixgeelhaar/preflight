package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// ConsoleLogger logs structured messages to the console.
type ConsoleLogger struct {
	mu           sync.Mutex
	out          io.Writer
	level        ports.Level
	fields       []ports.Field
	jsonFormat   bool
	includeTime  bool
	includeLevel bool
}

// ConsoleLoggerOption configures the console logger.
type ConsoleLoggerOption func(*ConsoleLogger)

// WithOutput sets the output writer (default: os.Stderr).
func WithOutput(w io.Writer) ConsoleLoggerOption {
	return func(l *ConsoleLogger) {
		l.out = w
	}
}

// WithLevel sets the minimum log level (default: Info).
func WithLevel(level ports.Level) ConsoleLoggerOption {
	return func(l *ConsoleLogger) {
		l.level = level
	}
}

// WithJSONFormat enables JSON output format.
func WithJSONFormat(enabled bool) ConsoleLoggerOption {
	return func(l *ConsoleLogger) {
		l.jsonFormat = enabled
	}
}

// WithTimestamp includes timestamp in log entries.
func WithTimestamp(enabled bool) ConsoleLoggerOption {
	return func(l *ConsoleLogger) {
		l.includeTime = enabled
	}
}

// WithLevelLabel includes level label in log entries.
func WithLevelLabel(enabled bool) ConsoleLoggerOption {
	return func(l *ConsoleLogger) {
		l.includeLevel = enabled
	}
}

// NewConsoleLogger creates a new console logger.
func NewConsoleLogger(opts ...ConsoleLoggerOption) *ConsoleLogger {
	l := &ConsoleLogger{
		out:          os.Stderr,
		level:        ports.LevelInfo,
		includeTime:  true,
		includeLevel: true,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Debug logs a debug message.
func (l *ConsoleLogger) Debug(ctx context.Context, msg string, fields ...ports.Field) {
	l.log(ctx, ports.LevelDebug, msg, fields)
}

// Info logs an informational message.
func (l *ConsoleLogger) Info(ctx context.Context, msg string, fields ...ports.Field) {
	l.log(ctx, ports.LevelInfo, msg, fields)
}

// Warn logs a warning message.
func (l *ConsoleLogger) Warn(ctx context.Context, msg string, fields ...ports.Field) {
	l.log(ctx, ports.LevelWarn, msg, fields)
}

// Error logs an error message.
func (l *ConsoleLogger) Error(ctx context.Context, msg string, fields ...ports.Field) {
	l.log(ctx, ports.LevelError, msg, fields)
}

// With returns a new logger with additional fields.
func (l *ConsoleLogger) With(fields ...ports.Field) ports.Logger {
	newFields := make([]ports.Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &ConsoleLogger{
		out:          l.out,
		level:        l.level,
		fields:       newFields,
		jsonFormat:   l.jsonFormat,
		includeTime:  l.includeTime,
		includeLevel: l.includeLevel,
	}
}

// Level returns the minimum log level.
func (l *ConsoleLogger) Level() ports.Level {
	return l.level
}

// SetLevel sets the minimum log level.
func (l *ConsoleLogger) SetLevel(level ports.Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log writes a log entry if the level is enabled.
func (l *ConsoleLogger) log(_ context.Context, level ports.Level, msg string, fields []ports.Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Combine base fields with call-specific fields
	allFields := make([]ports.Field, len(l.fields)+len(fields))
	copy(allFields, l.fields)
	copy(allFields[len(l.fields):], fields)

	if l.jsonFormat {
		l.writeJSON(level, msg, allFields)
	} else {
		l.writeText(level, msg, allFields)
	}
}

// writeJSON writes a JSON-formatted log entry.
func (l *ConsoleLogger) writeJSON(level ports.Level, msg string, fields []ports.Field) {
	entry := make(map[string]interface{})

	if l.includeTime {
		entry["time"] = time.Now().UTC().Format(time.RFC3339)
	}
	if l.includeLevel {
		entry["level"] = level.String()
	}
	entry["msg"] = msg

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintln(l.out, string(data))
}

// writeText writes a human-readable log entry.
func (l *ConsoleLogger) writeText(level ports.Level, msg string, fields []ports.Field) {
	var prefix string

	if l.includeTime {
		prefix = time.Now().Format("15:04:05") + " "
	}
	if l.includeLevel {
		prefix += fmt.Sprintf("[%s] ", level.String())
	}

	line := prefix + msg

	if len(fields) > 0 {
		line += " "
		for i, f := range fields {
			if i > 0 {
				line += " "
			}
			line += fmt.Sprintf("%s=%v", f.Key, f.Value)
		}
	}

	_, _ = fmt.Fprintln(l.out, line)
}

// Ensure ConsoleLogger implements Logger.
var _ ports.Logger = (*ConsoleLogger)(nil)

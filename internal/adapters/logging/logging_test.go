package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

func TestNopLogger_ImplementsInterface(_ *testing.T) {
	var _ ports.Logger = NewNopLogger()
}

func TestNopLogger_Methods(t *testing.T) {
	logger := NewNopLogger()
	ctx := context.Background()

	// All methods should be no-ops
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")

	// With should return itself
	withLogger := logger.With(ports.F("key", "value"))
	if withLogger != logger {
		t.Error("NopLogger.With should return itself")
	}
}

func TestNopLogger_Level(t *testing.T) {
	logger := NewNopLogger()

	if logger.Level() != ports.LevelInfo {
		t.Errorf("default level = %v, want %v", logger.Level(), ports.LevelInfo)
	}

	logger.SetLevel(ports.LevelDebug)
	if logger.Level() != ports.LevelDebug {
		t.Errorf("after SetLevel, level = %v, want %v", logger.Level(), ports.LevelDebug)
	}
}

func TestConsoleLogger_ImplementsInterface(_ *testing.T) {
	var _ ports.Logger = NewConsoleLogger()
}

func TestConsoleLogger_TextOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf),
		WithLevel(ports.LevelDebug),
		WithTimestamp(false),
		WithLevelLabel(true),
	)

	ctx := context.Background()
	logger.Info(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("output should contain [INFO], got %q", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain message, got %q", output)
	}
}

func TestConsoleLogger_TextOutput_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf),
		WithLevel(ports.LevelDebug),
		WithTimestamp(false),
		WithLevelLabel(false),
	)

	ctx := context.Background()
	logger.Info(ctx, "test", ports.F("key1", "value1"), ports.F("key2", 42))

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("output should contain key1=value1, got %q", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("output should contain key2=42, got %q", output)
	}
}

func TestConsoleLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf),
		WithLevel(ports.LevelDebug),
		WithJSONFormat(true),
		WithTimestamp(false),
		WithLevelLabel(true),
	)

	ctx := context.Background()
	logger.Info(ctx, "test message", ports.F("key", "value"))

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", entry["level"])
	}
	if entry["msg"] != "test message" {
		t.Errorf("msg = %v, want 'test message'", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Errorf("key = %v, want 'value'", entry["key"])
	}
}

func TestConsoleLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf),
		WithLevel(ports.LevelWarn),
		WithTimestamp(false),
	)

	ctx := context.Background()

	// Debug and Info should be filtered out
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	if buf.Len() > 0 {
		t.Errorf("Debug and Info should be filtered, got %q", buf.String())
	}

	// Warn and Error should pass through
	logger.Warn(ctx, "warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("Warn should not be filtered, got %q", buf.String())
	}

	buf.Reset()
	logger.Error(ctx, "error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Errorf("Error should not be filtered, got %q", buf.String())
	}
}

func TestConsoleLogger_With(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf),
		WithLevel(ports.LevelDebug),
		WithTimestamp(false),
		WithLevelLabel(false),
	)

	// Add base fields
	loggerWithFields := logger.With(ports.F("component", "test"))

	ctx := context.Background()
	loggerWithFields.Info(ctx, "message", ports.F("extra", "field"))

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Errorf("output should contain base field, got %q", output)
	}
	if !strings.Contains(output, "extra=field") {
		t.Errorf("output should contain extra field, got %q", output)
	}
}

func TestConsoleLogger_With_DoesNotModifyOriginal(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf1),
		WithLevel(ports.LevelDebug),
		WithTimestamp(false),
		WithLevelLabel(false),
	)

	// Create derived logger with additional field
	derived := logger.With(ports.F("derived", "yes"))
	derivedConsole := derived.(*ConsoleLogger)
	derivedConsole.out = &buf2

	ctx := context.Background()
	logger.Info(ctx, "original")
	derived.Info(ctx, "derived")

	if strings.Contains(buf1.String(), "derived=yes") {
		t.Error("original logger should not have derived field")
	}
	if !strings.Contains(buf2.String(), "derived=yes") {
		t.Error("derived logger should have derived field")
	}
}

func TestConsoleLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewConsoleLogger(
		WithOutput(&buf),
		WithLevel(ports.LevelError),
		WithTimestamp(false),
	)

	ctx := context.Background()

	// Info should be filtered
	logger.Info(ctx, "info message")
	if buf.Len() > 0 {
		t.Error("Info should be filtered at Error level")
	}

	// Change level to Debug
	logger.SetLevel(ports.LevelDebug)

	// Now Info should pass through
	logger.Info(ctx, "info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info should pass through at Debug level")
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    ports.Level
		expected string
	}{
		{ports.LevelDebug, "DEBUG"},
		{ports.LevelInfo, "INFO"},
		{ports.LevelWarn, "WARN"},
		{ports.LevelError, "ERROR"},
		{ports.Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoggerContext(t *testing.T) {
	logger := NewConsoleLogger()
	ctx := context.Background()

	// No logger in context
	if ports.LoggerFromContext(ctx) != nil {
		t.Error("LoggerFromContext should return nil for empty context")
	}

	// Add logger to context
	ctxWithLogger := ports.ContextWithLogger(ctx, logger)
	retrieved := ports.LoggerFromContext(ctxWithLogger)

	if retrieved == nil {
		t.Fatal("LoggerFromContext should return logger")
	}
	if retrieved != logger {
		t.Error("LoggerFromContext should return the same logger")
	}
}

func TestField(t *testing.T) {
	field := ports.F("key", "value")
	if field.Key != "key" {
		t.Errorf("Field.Key = %q, want 'key'", field.Key)
	}
	if field.Value != "value" {
		t.Errorf("Field.Value = %v, want 'value'", field.Value)
	}
}

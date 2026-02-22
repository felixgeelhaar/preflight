package ports

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{
			name:     "debug level",
			level:    LevelDebug,
			expected: "DEBUG",
		},
		{
			name:     "info level",
			level:    LevelInfo,
			expected: "INFO",
		},
		{
			name:     "warn level",
			level:    LevelWarn,
			expected: "WARN",
		},
		{
			name:     "error level",
			level:    LevelError,
			expected: "ERROR",
		},
		{
			name:     "unknown level",
			level:    Level(99),
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		value   interface{}
		wantKey string
		wantVal interface{}
	}{
		{
			name:    "string value",
			key:     "operation",
			value:   "install",
			wantKey: "operation",
			wantVal: "install",
		},
		{
			name:    "int value",
			key:     "count",
			value:   42,
			wantKey: "count",
			wantVal: 42,
		},
		{
			name:    "nil value",
			key:     "error",
			value:   nil,
			wantKey: "error",
			wantVal: nil,
		},
		{
			name:    "bool value",
			key:     "dry_run",
			value:   true,
			wantKey: "dry_run",
			wantVal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field := F(tt.key, tt.value)

			assert.Equal(t, tt.wantKey, field.Key)
			assert.Equal(t, tt.wantVal, field.Value)
		})
	}
}

// stubLogger is a minimal Logger implementation for context round-trip tests.
type stubLogger struct {
	level Level
}

func (s *stubLogger) Debug(_ context.Context, _ string, _ ...Field) {}
func (s *stubLogger) Info(_ context.Context, _ string, _ ...Field)  {}
func (s *stubLogger) Warn(_ context.Context, _ string, _ ...Field)  {}
func (s *stubLogger) Error(_ context.Context, _ string, _ ...Field) {}
func (s *stubLogger) With(_ ...Field) Logger                        { return s }
func (s *stubLogger) Level() Level                                  { return s.level }
func (s *stubLogger) SetLevel(level Level)                          { s.level = level }

func TestContextWithLogger_RoundTrip(t *testing.T) {
	t.Parallel()

	logger := &stubLogger{level: LevelInfo}
	ctx := ContextWithLogger(context.Background(), logger)

	got := LoggerFromContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, logger, got)
}

func TestLoggerFromContext_ReturnsNilWhenAbsent(t *testing.T) {
	t.Parallel()

	got := LoggerFromContext(context.Background())
	assert.Nil(t, got)
}

func TestLoggerFromContext_ReturnsNilForWrongType(t *testing.T) {
	t.Parallel()

	// Store a non-Logger value at the loggerKey to verify type assertion
	ctx := context.WithValue(context.Background(), loggerKey{}, "not-a-logger")

	got := LoggerFromContext(ctx)
	assert.Nil(t, got)
}

func TestContextWithLogger_OverwritesPrevious(t *testing.T) {
	t.Parallel()

	first := &stubLogger{level: LevelDebug}
	second := &stubLogger{level: LevelError}

	ctx := ContextWithLogger(context.Background(), first)
	ctx = ContextWithLogger(ctx, second)

	got := LoggerFromContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, second, got)
	assert.Equal(t, LevelError, got.Level())
}

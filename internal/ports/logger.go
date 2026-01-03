package ports

import "context"

// Level represents the severity of a log message.
type Level int

const (
	// LevelDebug is for verbose debugging information.
	LevelDebug Level = iota
	// LevelInfo is for general operational information.
	LevelInfo
	// LevelWarn is for potentially problematic situations.
	LevelWarn
	// LevelError is for error conditions.
	LevelError
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Field represents a structured logging field.
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger defines the interface for structured logging.
// Implementations can log to console, files, or external services.
type Logger interface {
	// Debug logs a debug message with optional structured fields.
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs an informational message with optional structured fields.
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs a warning message with optional structured fields.
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs an error message with optional structured fields.
	Error(ctx context.Context, msg string, fields ...Field)

	// With returns a new Logger with the given fields added to every log entry.
	With(fields ...Field) Logger

	// Level returns the minimum log level.
	Level() Level

	// SetLevel sets the minimum log level.
	SetLevel(level Level)
}

// LoggerFromContext retrieves a Logger from the context.
// Returns nil if no logger is present.
func LoggerFromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return logger
	}
	return nil
}

// ContextWithLogger returns a new context with the logger attached.
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// loggerKey is the context key for Logger.
type loggerKey struct{}

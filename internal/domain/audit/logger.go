package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger defines the interface for audit logging.
type Logger interface {
	// Log records an audit event
	Log(ctx context.Context, event Event) error

	// Query retrieves events matching the filter
	Query(ctx context.Context, filter QueryFilter) ([]Event, error)

	// Close releases any resources
	Close() error
}

// FileLogger implements Logger with file-based storage.
type FileLogger struct {
	mu       sync.Mutex
	dir      string
	maxSize  int64
	maxAge   time.Duration
	file     *os.File
	encoder  *json.Encoder
	size     int64
	rotation int
}

// FileLoggerConfig configures the file logger.
type FileLoggerConfig struct {
	// Dir is the directory for log files
	Dir string

	// MaxSize is the maximum size of a log file before rotation (default: 10MB)
	MaxSize int64

	// MaxAge is how long to keep old log files (default: 90 days)
	MaxAge time.Duration

	// MaxRotations is the number of rotated files to keep (default: 10)
	MaxRotations int
}

// DefaultFileLoggerConfig returns sensible defaults.
func DefaultFileLoggerConfig() FileLoggerConfig {
	home, _ := os.UserHomeDir()
	return FileLoggerConfig{
		Dir:          filepath.Join(home, ".preflight", "audit"),
		MaxSize:      10 * 1024 * 1024, // 10 MB
		MaxAge:       90 * 24 * time.Hour,
		MaxRotations: 10,
	}
}

// NewFileLogger creates a new file-based logger.
func NewFileLogger(config FileLoggerConfig) (*FileLogger, error) {
	if err := os.MkdirAll(config.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	logger := &FileLogger{
		dir:      config.Dir,
		maxSize:  config.MaxSize,
		maxAge:   config.MaxAge,
		rotation: config.MaxRotations,
	}

	if err := logger.openOrCreate(); err != nil {
		return nil, err
	}

	return logger, nil
}

// Log records an audit event.
func (l *FileLogger) Log(_ context.Context, event Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rotation is needed
	if l.size >= l.maxSize {
		if err := l.rotate(); err != nil {
			return fmt.Errorf("failed to rotate log: %w", err)
		}
	}

	// Encode event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write with newline
	data = append(data, '\n')
	n, err := l.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	l.size += int64(n)
	return nil
}

// Query retrieves events matching the filter.
func (l *FileLogger) Query(_ context.Context, filter QueryFilter) ([]Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var events []Event

	// List all log files
	files, err := l.listLogFiles()
	if err != nil {
		return nil, err
	}

	// Read from newest to oldest
	for i := len(files) - 1; i >= 0; i-- {
		fileEvents, err := l.readLogFile(files[i])
		if err != nil {
			continue // Skip corrupt files
		}

		for _, event := range fileEvents {
			if filter.Matches(event) {
				events = append(events, event)
				if filter.Limit > 0 && len(events) >= filter.Limit {
					return events, nil
				}
			}
		}
	}

	return events, nil
}

// Close releases resources.
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Cleanup removes old log files.
func (l *FileLogger) Cleanup() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	files, err := l.listLogFiles()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-l.maxAge)
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(f)
		}
	}

	return nil
}

// openOrCreate opens the current log file or creates a new one.
func (l *FileLogger) openOrCreate() error {
	path := l.currentLogPath()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}

	// Get current size
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to stat audit log: %w", err)
	}

	l.file = file
	l.encoder = json.NewEncoder(file)
	l.size = info.Size()

	return nil
}

// rotate rotates the current log file.
func (l *FileLogger) rotate() error {
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return err
		}
		l.file = nil
	}

	// Rename current log with timestamp
	current := l.currentLogPath()
	rotated := filepath.Join(l.dir, fmt.Sprintf("audit-%s.jsonl", time.Now().Format("20060102-150405")))
	if err := os.Rename(current, rotated); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove old rotated files if over limit
	if err := l.pruneRotated(); err != nil {
		return err
	}

	// Open new log file
	return l.openOrCreate()
}

// pruneRotated removes old rotated files beyond the limit.
func (l *FileLogger) pruneRotated() error {
	files, err := l.listLogFiles()
	if err != nil {
		return err
	}

	// Remove oldest files if over rotation limit
	if len(files) > l.rotation {
		for _, f := range files[:len(files)-l.rotation] {
			if f != l.currentLogPath() {
				_ = os.Remove(f)
			}
		}
	}

	return nil
}

// currentLogPath returns the path to the current log file.
func (l *FileLogger) currentLogPath() string {
	return filepath.Join(l.dir, "audit.jsonl")
}

// listLogFiles returns all log files sorted by modification time.
func (l *FileLogger) listLogFiles() ([]string, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, filepath.Join(l.dir, entry.Name()))
		}
	}

	return files, nil
}

// readLogFile reads all events from a log file.
func (l *FileLogger) readLogFile(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var events []Event
	decoder := json.NewDecoder(file)

	for {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed lines
		}
		events = append(events, event)
	}

	return events, nil
}

// MemoryLogger implements Logger with in-memory storage (for testing).
type MemoryLogger struct {
	mu     sync.RWMutex
	events []Event
}

// NewMemoryLogger creates a new in-memory logger.
func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{
		events: make([]Event, 0),
	}
}

// Log records an audit event.
func (l *MemoryLogger) Log(_ context.Context, event Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
	return nil
}

// Query retrieves events matching the filter.
func (l *MemoryLogger) Query(_ context.Context, filter QueryFilter) ([]Event, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []Event
	for _, event := range l.events {
		if filter.Matches(event) {
			result = append(result, event)
			if filter.Limit > 0 && len(result) >= filter.Limit {
				break
			}
		}
	}

	return result, nil
}

// Close is a no-op for memory logger.
func (l *MemoryLogger) Close() error {
	return nil
}

// Events returns all logged events (for testing).
func (l *MemoryLogger) Events() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Event, len(l.events))
	copy(result, l.events)
	return result
}

// Clear removes all logged events (for testing).
func (l *MemoryLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = l.events[:0]
}

// NullLogger discards all events (for disabled logging).
type NullLogger struct{}

// NewNullLogger creates a new null logger.
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Log discards the event.
func (l *NullLogger) Log(_ context.Context, _ Event) error {
	return nil
}

// Query returns empty results.
func (l *NullLogger) Query(_ context.Context, _ QueryFilter) ([]Event, error) {
	return nil, nil
}

// Close is a no-op.
func (l *NullLogger) Close() error {
	return nil
}

// Ensure implementations satisfy Logger interface.
var (
	_ Logger = (*FileLogger)(nil)
	_ Logger = (*MemoryLogger)(nil)
	_ Logger = (*NullLogger)(nil)
)

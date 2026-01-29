package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogLevel represents log severity levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// StructuredLog represents a single structured log entry
type StructuredLog struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      LogLevel               `json:"level"`
	Message    string                 `json:"message"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	Service    string                 `json:"service"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// StructuredLogger provides trace-correlated structured logging
type StructuredLogger struct {
	service    string
	minLevel   LogLevel
	mu         sync.Mutex
	buffer     []StructuredLog
	bufferSize int
	exportCh   chan []StructuredLog
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(service string, minLevel LogLevel) *StructuredLogger {
	if minLevel == "" {
		minLevel = LogLevelInfo
	}
	logger := &StructuredLogger{
		service:    service,
		minLevel:   minLevel,
		buffer:     make([]StructuredLog, 0, 100),
		bufferSize: 100,
		exportCh:   make(chan []StructuredLog, 10),
	}
	go logger.exportLoop()
	return logger
}

// Log creates a structured log entry with optional trace context
func (l *StructuredLogger) Log(ctx context.Context, level LogLevel, msg string, attrs map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := StructuredLog{
		Timestamp:  time.Now().UTC(),
		Level:      level,
		Message:    msg,
		Service:    l.service,
		Attributes: attrs,
	}

	// Extract trace context if available
	if tc := GetTraceContext(ctx); tc != nil {
		entry.TraceID = tc.TraceID
		entry.SpanID = tc.SpanID
	}

	// Output to stdout as JSON
	data, err := json.Marshal(entry)
	if err == nil {
		fmt.Fprintln(os.Stdout, string(data))
	}

	// Buffer for batch export
	l.mu.Lock()
	l.buffer = append(l.buffer, entry)
	if len(l.buffer) >= l.bufferSize {
		batch := make([]StructuredLog, len(l.buffer))
		copy(batch, l.buffer)
		l.buffer = l.buffer[:0]
		l.mu.Unlock()
		select {
		case l.exportCh <- batch:
		default:
		}
	} else {
		l.mu.Unlock()
	}
}

// Debug logs at DEBUG level
func (l *StructuredLogger) Debug(ctx context.Context, msg string, attrs map[string]interface{}) {
	l.Log(ctx, LogLevelDebug, msg, attrs)
}

// Info logs at INFO level
func (l *StructuredLogger) Info(ctx context.Context, msg string, attrs map[string]interface{}) {
	l.Log(ctx, LogLevelInfo, msg, attrs)
}

// Warn logs at WARN level
func (l *StructuredLogger) Warn(ctx context.Context, msg string, attrs map[string]interface{}) {
	l.Log(ctx, LogLevelWarn, msg, attrs)
}

// Error logs at ERROR level
func (l *StructuredLogger) Error(ctx context.Context, msg string, attrs map[string]interface{}) {
	l.Log(ctx, LogLevelError, msg, attrs)
}

// Flush exports any buffered logs
func (l *StructuredLogger) Flush() {
	l.mu.Lock()
	if len(l.buffer) > 0 {
		batch := make([]StructuredLog, len(l.buffer))
		copy(batch, l.buffer)
		l.buffer = l.buffer[:0]
		l.mu.Unlock()
		select {
		case l.exportCh <- batch:
		default:
		}
	} else {
		l.mu.Unlock()
	}
}

func (l *StructuredLogger) shouldLog(level LogLevel) bool {
	return levelRank(level) >= levelRank(l.minLevel)
}

func levelRank(level LogLevel) int {
	switch level {
	case LogLevelDebug:
		return 0
	case LogLevelInfo:
		return 1
	case LogLevelWarn:
		return 2
	case LogLevelError:
		return 3
	default:
		return 0
	}
}

func (l *StructuredLogger) exportLoop() {
	for batch := range l.exportCh {
		// Logs are already written to stdout; this loop handles batch processing
		_ = batch
	}
}

package utils

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
)

// newTestableLogger creates a Logger that writes to the given buffer
// so tests can capture and inspect output.
func newTestableLogger(service string, level int, format string, buf *bytes.Buffer) *Logger {
	return &Logger{
		logger:   log.New(buf, "", 0),
		service:  service,
		minLevel: level,
		format:   format,
	}
}

// --- parseLogLevel ---

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"DEBUG", 0},
		{"debug", 0},
		{"INFO", 1},
		{"info", 1},
		{"WARN", 2},
		{"warn", 2},
		{"ERROR", 3},
		{"error", 3},
		{"unknown", 1}, // default to INFO
		{"", 1},        // default to INFO
		{"TRACE", 1},   // unknown → default
		{"Info", 1},    // mixed case
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// --- NewLogger constructor ---

func TestNewLogger_Defaults(t *testing.T) {
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FORMAT")

	l := NewLogger("test-service")
	if l.service != "test-service" {
		t.Errorf("service = %q, want %q", l.service, "test-service")
	}
	if l.minLevel != 1 { // INFO default
		t.Errorf("minLevel = %d, want 1 (INFO)", l.minLevel)
	}
	if l.format != "json" {
		t.Errorf("format = %q, want %q", l.format, "json")
	}
}

func TestNewLogger_EnvOverrides(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	l := NewLogger("my-svc")
	if l.minLevel != 0 {
		t.Errorf("minLevel = %d, want 0 (DEBUG)", l.minLevel)
	}
	if l.format != "text" {
		t.Errorf("format = %q, want %q", l.format, "text")
	}
}

func TestNewLogger_EnvLevelError(t *testing.T) {
	t.Setenv("LOG_LEVEL", "ERROR")
	t.Setenv("LOG_FORMAT", "json")

	l := NewLogger("svc")
	if l.minLevel != 3 {
		t.Errorf("minLevel = %d, want 3 (ERROR)", l.minLevel)
	}
}

// --- Log level filtering ---

func TestLoggerLevelFiltering_InfoSkipsDebug(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 1, "json", &buf) // INFO level

	l.Debug("should be skipped", nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output for DEBUG at INFO level, got: %s", buf.String())
	}
}

func TestLoggerLevelFiltering_InfoAllowsInfo(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 1, "json", &buf)

	l.Info("hello", nil)
	if buf.Len() == 0 {
		t.Error("expected output for INFO at INFO level")
	}
}

func TestLoggerLevelFiltering_InfoAllowsWarnAndError(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 1, "json", &buf)

	l.Warn("warning", nil)
	if buf.Len() == 0 {
		t.Error("expected output for WARN at INFO level")
	}

	buf.Reset()
	l.Error("error", nil)
	if buf.Len() == 0 {
		t.Error("expected output for ERROR at INFO level")
	}
}

func TestLoggerLevelFiltering_ErrorSkipsLower(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 3, "json", &buf) // ERROR level

	l.Debug("skip", nil)
	l.Info("skip", nil)
	l.Warn("skip", nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output for DEBUG/INFO/WARN at ERROR level, got: %s", buf.String())
	}

	l.Error("allowed", nil)
	if buf.Len() == 0 {
		t.Error("expected output for ERROR at ERROR level")
	}
}

func TestLoggerLevelFiltering_DebugAllowsAll(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf) // DEBUG level

	l.Debug("d", nil)
	l.Info("i", nil)
	l.Warn("w", nil)
	l.Error("e", nil)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 log lines, got %d: %v", len(lines), lines)
	}
}

// --- JSON output format ---

func TestLoggerJSONOutput_BasicFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("webhook-api", 0, "json", &buf)

	l.Info("server started", nil)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, buf.String())
	}
	if entry.Level != "INFO" {
		t.Errorf("level = %q, want INFO", entry.Level)
	}
	if entry.Message != "server started" {
		t.Errorf("message = %q, want %q", entry.Message, "server started")
	}
	if entry.Service != "webhook-api" {
		t.Errorf("service = %q, want %q", entry.Service, "webhook-api")
	}
	if entry.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestLoggerJSONOutput_WithFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	fields := map[string]interface{}{
		"port": 8080,
		"env":  "production",
	}
	l.Info("listening", fields)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Fields == nil {
		t.Fatal("fields should not be nil")
	}
	if entry.Fields["env"] != "production" {
		t.Errorf("fields[env] = %v, want production", entry.Fields["env"])
	}
}

func TestLoggerJSONOutput_FieldExtraction(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	fields := map[string]interface{}{
		"request_id":  "req-123",
		"tenant_id":   "tenant-abc",
		"delivery_id": "del-456",
		"endpoint_id": "ep-789",
		"extra":       "value",
	}
	l.Info("processing", fields)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.RequestID != "req-123" {
		t.Errorf("request_id = %q, want %q", entry.RequestID, "req-123")
	}
	if entry.TenantID != "tenant-abc" {
		t.Errorf("tenant_id = %q, want %q", entry.TenantID, "tenant-abc")
	}
	if entry.DeliveryID != "del-456" {
		t.Errorf("delivery_id = %q, want %q", entry.DeliveryID, "del-456")
	}
	if entry.EndpointID != "ep-789" {
		t.Errorf("endpoint_id = %q, want %q", entry.EndpointID, "ep-789")
	}
}

func TestLoggerJSONOutput_NilFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	l.Info("msg", nil)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.RequestID != "" {
		t.Errorf("request_id should be empty, got %q", entry.RequestID)
	}
}

// --- Text output format ---

func TestLoggerTextOutput_BasicFormat(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("webhook-api", 0, "text", &buf)

	l.Info("hello world", nil)

	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("text output missing [INFO]: %s", out)
	}
	if !strings.Contains(out, "webhook-api") {
		t.Errorf("text output missing service name: %s", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("text output missing message: %s", out)
	}
}

func TestLoggerTextOutput_WithFieldExtraction(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "text", &buf)

	fields := map[string]interface{}{
		"request_id":  "req-1",
		"tenant_id":   "t-2",
		"delivery_id": "d-3",
		"endpoint_id": "e-4",
	}
	l.Info("msg", fields)

	out := buf.String()
	if !strings.Contains(out, "request_id=req-1") {
		t.Errorf("missing request_id in text output: %s", out)
	}
	if !strings.Contains(out, "tenant_id=t-2") {
		t.Errorf("missing tenant_id in text output: %s", out)
	}
	if !strings.Contains(out, "delivery_id=d-3") {
		t.Errorf("missing delivery_id in text output: %s", out)
	}
	if !strings.Contains(out, "endpoint_id=e-4") {
		t.Errorf("missing endpoint_id in text output: %s", out)
	}
}

func TestLoggerTextOutput_ExtraFieldsExcludeExtracted(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "text", &buf)

	fields := map[string]interface{}{
		"request_id": "req-1",
		"custom_key": "custom_val",
	}
	l.Info("msg", fields)

	out := buf.String()
	// request_id appears as top-level, not duplicated
	if !strings.Contains(out, "request_id=req-1") {
		t.Errorf("missing request_id: %s", out)
	}
	if !strings.Contains(out, "custom_key=custom_val") {
		t.Errorf("missing custom_key: %s", out)
	}
	// Count occurrences — request_id should appear exactly once (as top-level)
	if strings.Count(out, "request_id") != 1 {
		t.Errorf("request_id should appear once, output: %s", out)
	}
}

func TestLoggerTextOutput_CorrelationID(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "text", &buf)

	l.InfoWithCorrelation("corr msg", "corr-abc", nil)

	out := buf.String()
	if !strings.Contains(out, "correlation_id=corr-abc") {
		t.Errorf("missing correlation_id in text output: %s", out)
	}
}

// --- Correlation ID methods ---

func TestInfoWithCorrelation(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	l.InfoWithCorrelation("correlated", "corr-123", map[string]interface{}{"key": "val"})

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Level != "INFO" {
		t.Errorf("level = %q, want INFO", entry.Level)
	}
	if entry.CorrelationID != "corr-123" {
		t.Errorf("correlation_id = %q, want %q", entry.CorrelationID, "corr-123")
	}
	if entry.Message != "correlated" {
		t.Errorf("message = %q, want %q", entry.Message, "correlated")
	}
}

func TestErrorWithCorrelation(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	l.ErrorWithCorrelation("fail", "corr-err", nil)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", entry.Level)
	}
	if entry.CorrelationID != "corr-err" {
		t.Errorf("correlation_id = %q, want %q", entry.CorrelationID, "corr-err")
	}
}

func TestWarnWithCorrelation(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	l.WarnWithCorrelation("warning", "corr-w", nil)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Level != "WARN" {
		t.Errorf("level = %q, want WARN", entry.Level)
	}
	if entry.CorrelationID != "corr-w" {
		t.Errorf("correlation_id = %q, want %q", entry.CorrelationID, "corr-w")
	}
}

func TestDebugWithCorrelation(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	l.DebugWithCorrelation("trace", "corr-d", nil)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Level != "DEBUG" {
		t.Errorf("level = %q, want DEBUG", entry.Level)
	}
	if entry.CorrelationID != "corr-d" {
		t.Errorf("correlation_id = %q, want %q", entry.CorrelationID, "corr-d")
	}
}

func TestCorrelationWithFieldExtraction(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	fields := map[string]interface{}{
		"request_id":  "req-x",
		"tenant_id":   "ten-y",
		"delivery_id": "del-z",
		"endpoint_id": "ep-w",
	}
	l.InfoWithCorrelation("msg", "corr-id", fields)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.CorrelationID != "corr-id" {
		t.Errorf("correlation_id = %q, want corr-id", entry.CorrelationID)
	}
	if entry.RequestID != "req-x" {
		t.Errorf("request_id = %q, want req-x", entry.RequestID)
	}
	if entry.TenantID != "ten-y" {
		t.Errorf("tenant_id = %q, want ten-y", entry.TenantID)
	}
	if entry.DeliveryID != "del-z" {
		t.Errorf("delivery_id = %q, want del-z", entry.DeliveryID)
	}
	if entry.EndpointID != "ep-w" {
		t.Errorf("endpoint_id = %q, want ep-w", entry.EndpointID)
	}
}

// --- Correlation level filtering ---

func TestCorrelationLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 2, "json", &buf) // WARN level

	l.DebugWithCorrelation("skip", "c1", nil)
	l.InfoWithCorrelation("skip", "c2", nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output for DEBUG/INFO correlation at WARN level, got: %s", buf.String())
	}

	l.WarnWithCorrelation("yes", "c3", nil)
	if buf.Len() == 0 {
		t.Error("expected output for WARN correlation at WARN level")
	}

	buf.Reset()
	l.ErrorWithCorrelation("yes", "c4", nil)
	if buf.Len() == 0 {
		t.Error("expected output for ERROR correlation at WARN level")
	}
}

// --- WithCorrelationID ---

func TestWithCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	child := l.WithCorrelationID("child-corr")
	if child.service != l.service {
		t.Errorf("child service = %q, want %q", child.service, l.service)
	}
	if child.minLevel != l.minLevel {
		t.Errorf("child minLevel = %d, want %d", child.minLevel, l.minLevel)
	}
	if child.format != l.format {
		t.Errorf("child format = %q, want %q", child.format, l.format)
	}
	// Child should share the same underlying logger writer
	child.Info("child msg", nil)
	if buf.Len() == 0 {
		t.Error("child logger should write to the same writer")
	}
}

// --- Each log method produces correct level ---

func TestLogMethods_CorrectLevels(t *testing.T) {
	methods := []struct {
		name     string
		logFn    func(*Logger, *bytes.Buffer)
		expected string
	}{
		{"Info", func(l *Logger, _ *bytes.Buffer) { l.Info("m", nil) }, "INFO"},
		{"Warn", func(l *Logger, _ *bytes.Buffer) { l.Warn("m", nil) }, "WARN"},
		{"Error", func(l *Logger, _ *bytes.Buffer) { l.Error("m", nil) }, "ERROR"},
		{"Debug", func(l *Logger, _ *bytes.Buffer) { l.Debug("m", nil) }, "DEBUG"},
	}

	for _, tt := range methods {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := newTestableLogger("svc", 0, "json", &buf)
			tt.logFn(l, &buf)

			var entry LogEntry
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("parse error: %v\nraw: %s", err, buf.String())
			}
			if entry.Level != tt.expected {
				t.Errorf("level = %q, want %q", entry.Level, tt.expected)
			}
		})
	}
}

// --- shouldLog ---

func TestShouldLog(t *testing.T) {
	l := &Logger{minLevel: 2} // WARN

	tests := []struct {
		level    string
		expected bool
	}{
		{"DEBUG", false},
		{"INFO", false},
		{"WARN", true},
		{"ERROR", true},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if got := l.shouldLog(tt.level); got != tt.expected {
				t.Errorf("shouldLog(%q) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

// --- JSON omitempty behavior ---

func TestLoggerJSON_OmitsEmptyOptionalFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	l.Info("simple", nil)

	raw := buf.String()
	// These optional fields should be omitted when empty
	if strings.Contains(raw, `"request_id"`) {
		t.Errorf("empty request_id should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"correlation_id"`) {
		t.Errorf("empty correlation_id should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"tenant_id"`) {
		t.Errorf("empty tenant_id should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"delivery_id"`) {
		t.Errorf("empty delivery_id should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"endpoint_id"`) {
		t.Errorf("empty endpoint_id should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"fields"`) {
		t.Errorf("nil fields should be omitted: %s", raw)
	}
}

// --- Non-string field values are not extracted ---

func TestLoggerFieldExtraction_NonStringIgnored(t *testing.T) {
	var buf bytes.Buffer
	l := newTestableLogger("svc", 0, "json", &buf)

	fields := map[string]interface{}{
		"request_id": 12345, // not a string
		"tenant_id":  true,  // not a string
	}
	l.Info("msg", fields)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// Non-string values should not be extracted to top-level
	if entry.RequestID != "" {
		t.Errorf("request_id should be empty for non-string, got %q", entry.RequestID)
	}
	if entry.TenantID != "" {
		t.Errorf("tenant_id should be empty for non-string, got %q", entry.TenantID)
	}
}

// --- Text format for each level ---

func TestLoggerTextOutput_AllLevels(t *testing.T) {
	levels := []struct {
		name  string
		logFn func(*Logger)
		tag   string
	}{
		{"Info", func(l *Logger) { l.Info("m", nil) }, "[INFO]"},
		{"Warn", func(l *Logger) { l.Warn("m", nil) }, "[WARN]"},
		{"Error", func(l *Logger) { l.Error("m", nil) }, "[ERROR]"},
		{"Debug", func(l *Logger) { l.Debug("m", nil) }, "[DEBUG]"},
	}
	for _, tt := range levels {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := newTestableLogger("svc", 0, "text", &buf)
			tt.logFn(l)
			if !strings.Contains(buf.String(), tt.tag) {
				t.Errorf("text output missing %s: %s", tt.tag, buf.String())
			}
		})
	}
}

package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Logger struct {
	logger   *log.Logger
	service  string
	minLevel int
	format   string // "json" or "text"
}

type LogEntry struct {
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Timestamp     time.Time              `json:"timestamp"`
	Service       string                 `json:"service"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	DeliveryID    string                 `json:"delivery_id,omitempty"`
	EndpointID    string                 `json:"endpoint_id,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

var levelValues = map[string]int{
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
}

func parseLogLevel(level string) int {
	if v, ok := levelValues[strings.ToUpper(level)]; ok {
		return v
	}
	return 1 // default to INFO
}

func NewLogger(service string) *Logger {
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "json")
	return &Logger{
		logger:   log.New(os.Stdout, "", 0),
		service:  service,
		minLevel: parseLogLevel(logLevel),
		format:   strings.ToLower(logFormat),
	}
}

func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	return &Logger{
		logger:   l.logger,
		service:  l.service,
		minLevel: l.minLevel,
		format:   l.format,
	}
}

func (l *Logger) InfoWithCorrelation(message string, correlationID string, fields map[string]interface{}) {
	l.logWithCorrelation("INFO", message, correlationID, fields)
}

func (l *Logger) ErrorWithCorrelation(message string, correlationID string, fields map[string]interface{}) {
	l.logWithCorrelation("ERROR", message, correlationID, fields)
}

func (l *Logger) WarnWithCorrelation(message string, correlationID string, fields map[string]interface{}) {
	l.logWithCorrelation("WARN", message, correlationID, fields)
}

func (l *Logger) DebugWithCorrelation(message string, correlationID string, fields map[string]interface{}) {
	l.logWithCorrelation("DEBUG", message, correlationID, fields)
}

func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log("INFO", message, fields)
}

func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log("ERROR", message, fields)
}

func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log("WARN", message, fields)
}

func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log("DEBUG", message, fields)
}

func (l *Logger) shouldLog(level string) bool {
	return parseLogLevel(level) >= l.minLevel
}

func (l *Logger) formatText(entry LogEntry) string {
	msg := fmt.Sprintf("%s [%s] %s: %s",
		entry.Timestamp.Format(time.RFC3339), entry.Level, l.service, entry.Message)
	if entry.RequestID != "" {
		msg += fmt.Sprintf(" request_id=%s", entry.RequestID)
	}
	if entry.CorrelationID != "" {
		msg += fmt.Sprintf(" correlation_id=%s", entry.CorrelationID)
	}
	if entry.TenantID != "" {
		msg += fmt.Sprintf(" tenant_id=%s", entry.TenantID)
	}
	if entry.DeliveryID != "" {
		msg += fmt.Sprintf(" delivery_id=%s", entry.DeliveryID)
	}
	if entry.EndpointID != "" {
		msg += fmt.Sprintf(" endpoint_id=%s", entry.EndpointID)
	}
	if entry.Fields != nil {
		for k, v := range entry.Fields {
			// Skip fields already extracted as top-level
			switch k {
			case "request_id", "tenant_id", "delivery_id", "endpoint_id":
				continue
			}
			msg += fmt.Sprintf(" %s=%v", k, v)
		}
	}
	return msg
}

func (l *Logger) log(level, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Service:   l.service,
		Fields:    fields,
	}

	// Extract common fields from the fields map
	if fields != nil {
		if requestID, ok := fields["request_id"].(string); ok {
			entry.RequestID = requestID
		}
		if tenantID, ok := fields["tenant_id"].(string); ok {
			entry.TenantID = tenantID
		}
		if deliveryID, ok := fields["delivery_id"].(string); ok {
			entry.DeliveryID = deliveryID
		}
		if endpointID, ok := fields["endpoint_id"].(string); ok {
			entry.EndpointID = endpointID
		}
	}

	if l.format == "text" {
		l.logger.Println(l.formatText(entry))
		return
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}

func (l *Logger) logWithCorrelation(level, message, correlationID string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Level:         level,
		Message:       message,
		Timestamp:     time.Now().UTC(),
		Service:       l.service,
		CorrelationID: correlationID,
		Fields:        fields,
	}

	// Extract common fields from the fields map
	if fields != nil {
		if requestID, ok := fields["request_id"].(string); ok {
			entry.RequestID = requestID
		}
		if tenantID, ok := fields["tenant_id"].(string); ok {
			entry.TenantID = tenantID
		}
		if deliveryID, ok := fields["delivery_id"].(string); ok {
			entry.DeliveryID = deliveryID
		}
		if endpointID, ok := fields["endpoint_id"].(string); ok {
			entry.EndpointID = endpointID
		}
	}

	if l.format == "text" {
		l.logger.Println(l.formatText(entry))
		return
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}

package utils

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct {
	logger *log.Logger
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

func NewLogger(service string) *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	return &Logger{
		logger: l.logger,
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

func (l *Logger) log(level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC(),
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

	jsonData, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}

func (l *Logger) logWithCorrelation(level, message, correlationID string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:         level,
		Message:       message,
		Timestamp:     time.Now().UTC(),
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

	jsonData, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}
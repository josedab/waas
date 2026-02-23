package standardwebhooks

import (
	"encoding/json"
	"time"
)

// Standard Webhooks header constants per spec
const (
	HeaderWebhookID        = "webhook-id"
	HeaderWebhookTimestamp = "webhook-timestamp"
	HeaderWebhookSignature = "webhook-signature"
)

// CloudEvents header constants per CNCF CloudEvents v1.0
const (
	HeaderCEID          = "ce-id"
	HeaderCESource      = "ce-source"
	HeaderCESpecVersion = "ce-specversion"
	HeaderCEType        = "ce-type"
	HeaderCETime        = "ce-time"
	HeaderCESubject     = "ce-subject"
	HeaderContentType   = "Content-Type"
)

// Content mode constants
const (
	ContentModeStructured = "structured"
	ContentModeBinary     = "binary"
)

// Format constants for auto-detection
const (
	FormatStandardWebhooks = "standard_webhooks"
	FormatCloudEvents      = "cloudevents"
	FormatCustom           = "custom"
)

// CloudEvents specversion
const CloudEventsSpecVersion = "1.0"

// StandardWebhooksEnvelope is the envelope format per the Standard Webhooks spec.
type StandardWebhooksEnvelope struct {
	WebhookID        string          `json:"webhook_id"`
	WebhookTimestamp int64           `json:"webhook_timestamp"`
	EventType        string          `json:"event_type"`
	Data             json.RawMessage `json:"data"`
}

// CloudEvent is the structured CloudEvents v1.0 envelope.
type CloudEvent struct {
	SpecVersion     string                 `json:"specversion"`
	ID              string                 `json:"id"`
	Source          string                 `json:"source"`
	Type            string                 `json:"type"`
	Time            *time.Time             `json:"time,omitempty"`
	Subject         string                 `json:"subject,omitempty"`
	DataContentType string                 `json:"datacontenttype,omitempty"`
	Data            json.RawMessage        `json:"data,omitempty"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`
}

// DetectedFormat holds the result of auto-detecting a webhook format.
type DetectedFormat struct {
	Format      string  `json:"format"`
	ContentMode string  `json:"content_mode,omitempty"`
	Confidence  float64 `json:"confidence"`
}

// ConversionResult holds the output of a format conversion.
type ConversionResult struct {
	Payload     json.RawMessage   `json:"payload"`
	Headers     map[string]string `json:"headers"`
	Format      string            `json:"format"`
	ContentMode string            `json:"content_mode,omitempty"`
}

// ConformanceResult holds the result of a conformance test.
type ConformanceResult struct {
	Format  string             `json:"format"`
	Passed  int                `json:"passed"`
	Failed  int                `json:"failed"`
	Total   int                `json:"total"`
	Results []ConformanceCheck `json:"results"`
}

// ConformanceCheck is a single conformance check.
type ConformanceCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message,omitempty"`
}

// Request DTOs

// DetectFormatRequest asks the engine to detect a webhook's format.
type DetectFormatRequest struct {
	Headers map[string]string `json:"headers" binding:"required"`
	Payload json.RawMessage   `json:"payload" binding:"required"`
}

// ConvertRequest asks for format conversion.
type ConvertRequest struct {
	Headers      map[string]string `json:"headers" binding:"required"`
	Payload      json.RawMessage   `json:"payload" binding:"required"`
	TargetFormat string            `json:"target_format" binding:"required"`
	Source       string            `json:"source,omitempty"`
	EventType    string            `json:"event_type,omitempty"`
}

// SignRequest asks the engine to sign a payload per Standard Webhooks spec.
type SignRequest struct {
	WebhookID string          `json:"webhook_id" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
	Secret    string          `json:"secret" binding:"required"`
}

// SignResponse returns the signed headers.
type SignResponse struct {
	Headers map[string]string `json:"headers"`
}

// VerifyRequest asks the engine to verify a Standard Webhooks signature.
type VerifyRequest struct {
	Headers   map[string]string `json:"headers" binding:"required"`
	Payload   json.RawMessage   `json:"payload" binding:"required"`
	Secret    string            `json:"secret" binding:"required"`
	Tolerance int               `json:"tolerance_seconds,omitempty"`
}

// VerifyResponse returns the verification result.
type VerifyResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
}

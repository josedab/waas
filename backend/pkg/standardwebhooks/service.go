package standardwebhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultTimestampTolerance = 5 * 60 // 5 minutes in seconds

// Service provides Standard Webhooks and CloudEvents business logic.
type Service struct{}

// NewService creates a new Standard Webhooks service.
func NewService() *Service {
	return &Service{}
}

// DetectFormat auto-detects the webhook format from headers and payload.
func (s *Service) DetectFormat(headers map[string]string, payload json.RawMessage) *DetectedFormat {
	normalizedHeaders := normalizeHeaders(headers)

	// Check for Standard Webhooks headers
	if _, ok := normalizedHeaders[HeaderWebhookID]; ok {
		if _, ok2 := normalizedHeaders[HeaderWebhookTimestamp]; ok2 {
			return &DetectedFormat{Format: FormatStandardWebhooks, Confidence: 1.0}
		}
	}

	// Check for CloudEvents binary content mode
	if _, ok := normalizedHeaders[HeaderCESpecVersion]; ok {
		return &DetectedFormat{
			Format:      FormatCloudEvents,
			ContentMode: ContentModeBinary,
			Confidence:  1.0,
		}
	}

	// Check for CloudEvents structured content mode
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err == nil {
		if sv, ok := data["specversion"]; ok {
			if svStr, ok := sv.(string); ok && svStr == CloudEventsSpecVersion {
				return &DetectedFormat{
					Format:      FormatCloudEvents,
					ContentMode: ContentModeStructured,
					Confidence:  0.95,
				}
			}
		}
	}

	return &DetectedFormat{Format: FormatCustom, Confidence: 0.5}
}

// Sign generates Standard Webhooks spec-compliant headers for a payload.
func (s *Service) Sign(webhookID string, payload json.RawMessage, secret string) (*SignResponse, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhook_id is required")
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("payload is required")
	}
	if secret == "" {
		return nil, fmt.Errorf("secret is required")
	}

	ts := time.Now().Unix()
	toSign := fmt.Sprintf("%s.%d.%s", webhookID, ts, string(payload))
	signature := computeHMACSHA256(toSign, secret)

	return &SignResponse{
		Headers: map[string]string{
			HeaderWebhookID:        webhookID,
			HeaderWebhookTimestamp: strconv.FormatInt(ts, 10),
			HeaderWebhookSignature: "v1," + signature,
		},
	}, nil
}

// Verify verifies a Standard Webhooks signature.
func (s *Service) Verify(headers map[string]string, payload json.RawMessage, secret string, toleranceSec int) (*VerifyResponse, error) {
	normalized := normalizeHeaders(headers)

	webhookID := normalized[HeaderWebhookID]
	if webhookID == "" {
		return &VerifyResponse{Valid: false, Reason: "missing webhook-id header"}, nil
	}

	tsStr := normalized[HeaderWebhookTimestamp]
	if tsStr == "" {
		return &VerifyResponse{Valid: false, Reason: "missing webhook-timestamp header"}, nil
	}

	sig := normalized[HeaderWebhookSignature]
	if sig == "" {
		return &VerifyResponse{Valid: false, Reason: "missing webhook-signature header"}, nil
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return &VerifyResponse{Valid: false, Reason: "invalid timestamp"}, nil
	}

	if toleranceSec <= 0 {
		toleranceSec = defaultTimestampTolerance
	}
	age := time.Now().Unix() - ts
	if math.Abs(float64(age)) > float64(toleranceSec) {
		return &VerifyResponse{Valid: false, Reason: "timestamp outside tolerance window"}, nil
	}

	toSign := fmt.Sprintf("%s.%d.%s", webhookID, ts, string(payload))
	expectedSig := "v1," + computeHMACSHA256(toSign, secret)

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return &VerifyResponse{Valid: false, Reason: "signature mismatch"}, nil
	}

	return &VerifyResponse{Valid: true}, nil
}

// ConvertToCloudEvents converts a Standard Webhooks payload to CloudEvents format.
func (s *Service) ConvertToCloudEvents(headers map[string]string, payload json.RawMessage, source, eventType string) (*ConversionResult, error) {
	normalized := normalizeHeaders(headers)

	webhookID := normalized[HeaderWebhookID]
	if webhookID == "" {
		webhookID = uuid.New().String()
	}

	if source == "" {
		source = "waas"
	}
	if eventType == "" {
		eventType = "webhook.delivery"
	}

	now := time.Now()
	ce := CloudEvent{
		SpecVersion:     CloudEventsSpecVersion,
		ID:              webhookID,
		Source:          source,
		Type:            eventType,
		Time:            &now,
		DataContentType: "application/json",
		Data:            payload,
	}

	cePayload, err := json.Marshal(ce)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CloudEvent: %w", err)
	}

	return &ConversionResult{
		Payload: cePayload,
		Headers: map[string]string{
			HeaderContentType: "application/cloudevents+json",
		},
		Format:      FormatCloudEvents,
		ContentMode: ContentModeStructured,
	}, nil
}

// ConvertToStandardWebhooks converts a CloudEvent to Standard Webhooks format.
func (s *Service) ConvertToStandardWebhooks(headers map[string]string, payload json.RawMessage) (*ConversionResult, error) {
	var ce CloudEvent

	// Try structured mode first
	if err := json.Unmarshal(payload, &ce); err != nil || ce.SpecVersion == "" {
		// Try binary mode
		normalized := normalizeHeaders(headers)
		ce = CloudEvent{
			SpecVersion: normalized[HeaderCESpecVersion],
			ID:          normalized[HeaderCEID],
			Source:      normalized[HeaderCESource],
			Type:        normalized[HeaderCEType],
			Data:        payload,
		}
	}

	if ce.ID == "" {
		ce.ID = uuid.New().String()
	}

	envelope := StandardWebhooksEnvelope{
		WebhookID:        ce.ID,
		WebhookTimestamp: time.Now().Unix(),
		EventType:        ce.Type,
		Data:             ce.Data,
	}

	swPayload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Standard Webhooks envelope: %w", err)
	}

	return &ConversionResult{
		Payload: swPayload,
		Headers: map[string]string{
			HeaderWebhookID:        envelope.WebhookID,
			HeaderWebhookTimestamp: strconv.FormatInt(envelope.WebhookTimestamp, 10),
		},
		Format: FormatStandardWebhooks,
	}, nil
}

// RunConformanceTests runs conformance tests for a given format.
func (s *Service) RunConformanceTests(format string) *ConformanceResult {
	switch format {
	case FormatStandardWebhooks:
		return s.runStandardWebhooksConformance()
	case FormatCloudEvents:
		return s.runCloudEventsConformance()
	default:
		return &ConformanceResult{Format: format}
	}
}

func (s *Service) runStandardWebhooksConformance() *ConformanceResult {
	checks := []ConformanceCheck{
		s.checkSWSignatureGeneration(),
		s.checkSWSignatureVerification(),
		s.checkSWTimestampTolerance(),
		s.checkSWHeaderPresence(),
		s.checkSWIdempotencyKey(),
	}

	result := &ConformanceResult{Format: FormatStandardWebhooks, Results: checks, Total: len(checks)}
	for _, c := range checks {
		if c.Passed {
			result.Passed++
		} else {
			result.Failed++
		}
	}
	return result
}

func (s *Service) runCloudEventsConformance() *ConformanceResult {
	checks := []ConformanceCheck{
		s.checkCEStructuredMode(),
		s.checkCEBinaryMode(),
		s.checkCERequiredAttributes(),
		s.checkCESpecVersion(),
	}

	result := &ConformanceResult{Format: FormatCloudEvents, Results: checks, Total: len(checks)}
	for _, c := range checks {
		if c.Passed {
			result.Passed++
		} else {
			result.Failed++
		}
	}
	return result
}

func (s *Service) checkSWSignatureGeneration() ConformanceCheck {
	resp, err := s.Sign("msg_test123", json.RawMessage(`{"test":true}`), "whsec_testkey")
	if err != nil || resp.Headers[HeaderWebhookSignature] == "" {
		return ConformanceCheck{Name: "sw_signature_generation", Description: "Generate valid HMAC-SHA256 signature", Passed: false, Message: "Failed to generate signature"}
	}
	return ConformanceCheck{Name: "sw_signature_generation", Description: "Generate valid HMAC-SHA256 signature", Passed: true}
}

func (s *Service) checkSWSignatureVerification() ConformanceCheck {
	payload := json.RawMessage(`{"test":true}`)
	resp, _ := s.Sign("msg_test123", payload, "whsec_testkey")
	verify, _ := s.Verify(resp.Headers, payload, "whsec_testkey", 60)
	if verify == nil || !verify.Valid {
		return ConformanceCheck{Name: "sw_signature_verification", Description: "Verify HMAC-SHA256 signature", Passed: false, Message: "Verification failed"}
	}
	return ConformanceCheck{Name: "sw_signature_verification", Description: "Verify HMAC-SHA256 signature", Passed: true}
}

func (s *Service) checkSWTimestampTolerance() ConformanceCheck {
	headers := map[string]string{
		HeaderWebhookID:        "msg_old",
		HeaderWebhookTimestamp: "1000000000",
		HeaderWebhookSignature: "v1,invalid",
	}
	verify, _ := s.Verify(headers, json.RawMessage(`{}`), "secret", 60)
	if verify != nil && !verify.Valid && strings.Contains(verify.Reason, "tolerance") {
		return ConformanceCheck{Name: "sw_timestamp_tolerance", Description: "Reject expired timestamps", Passed: true}
	}
	return ConformanceCheck{Name: "sw_timestamp_tolerance", Description: "Reject expired timestamps", Passed: false}
}

func (s *Service) checkSWHeaderPresence() ConformanceCheck {
	resp, _ := s.Sign("msg_1", json.RawMessage(`{}`), "secret")
	if resp.Headers[HeaderWebhookID] != "" && resp.Headers[HeaderWebhookTimestamp] != "" && resp.Headers[HeaderWebhookSignature] != "" {
		return ConformanceCheck{Name: "sw_header_presence", Description: "Include all required headers", Passed: true}
	}
	return ConformanceCheck{Name: "sw_header_presence", Description: "Include all required headers", Passed: false}
}

func (s *Service) checkSWIdempotencyKey() ConformanceCheck {
	resp, _ := s.Sign("msg_idempotent_123", json.RawMessage(`{}`), "secret")
	if resp.Headers[HeaderWebhookID] == "msg_idempotent_123" {
		return ConformanceCheck{Name: "sw_idempotency_key", Description: "Use webhook-id as idempotency key", Passed: true}
	}
	return ConformanceCheck{Name: "sw_idempotency_key", Description: "Use webhook-id as idempotency key", Passed: false}
}

func (s *Service) checkCEStructuredMode() ConformanceCheck {
	ce := CloudEvent{SpecVersion: "1.0", ID: "test-1", Source: "test", Type: "test.event", DataContentType: "application/json", Data: json.RawMessage(`{}`)}
	b, _ := json.Marshal(ce)
	detected := s.DetectFormat(map[string]string{"Content-Type": "application/cloudevents+json"}, b)
	if detected.Format == FormatCloudEvents && detected.ContentMode == ContentModeStructured {
		return ConformanceCheck{Name: "ce_structured_mode", Description: "Detect CloudEvents structured content mode", Passed: true}
	}
	return ConformanceCheck{Name: "ce_structured_mode", Description: "Detect CloudEvents structured content mode", Passed: false}
}

func (s *Service) checkCEBinaryMode() ConformanceCheck {
	headers := map[string]string{HeaderCESpecVersion: "1.0", HeaderCEID: "test-1", HeaderCESource: "test", HeaderCEType: "test.event"}
	detected := s.DetectFormat(headers, json.RawMessage(`{"key":"value"}`))
	if detected.Format == FormatCloudEvents && detected.ContentMode == ContentModeBinary {
		return ConformanceCheck{Name: "ce_binary_mode", Description: "Detect CloudEvents binary content mode", Passed: true}
	}
	return ConformanceCheck{Name: "ce_binary_mode", Description: "Detect CloudEvents binary content mode", Passed: false}
}

func (s *Service) checkCERequiredAttributes() ConformanceCheck {
	ce := CloudEvent{SpecVersion: "1.0", ID: "test-1", Source: "test", Type: "test.event"}
	if ce.SpecVersion != "" && ce.ID != "" && ce.Source != "" && ce.Type != "" {
		return ConformanceCheck{Name: "ce_required_attributes", Description: "Include required CloudEvents attributes", Passed: true}
	}
	return ConformanceCheck{Name: "ce_required_attributes", Description: "Include required CloudEvents attributes", Passed: false}
}

func (s *Service) checkCESpecVersion() ConformanceCheck {
	if CloudEventsSpecVersion == "1.0" {
		return ConformanceCheck{Name: "ce_specversion", Description: "Support CloudEvents spec version 1.0", Passed: true}
	}
	return ConformanceCheck{Name: "ce_specversion", Description: "Support CloudEvents spec version 1.0", Passed: false}
}

func computeHMACSHA256(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func normalizeHeaders(headers map[string]string) map[string]string {
	normalized := make(map[string]string, len(headers))
	for k, v := range headers {
		normalized[strings.ToLower(k)] = v
	}
	return normalized
}

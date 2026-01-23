package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// TimingBreakdown provides detailed HTTP timing information for a delivery attempt
type TimingBreakdown struct {
	DeliveryID    string  `json:"delivery_id"`
	DNSLookupMs   float64 `json:"dns_lookup_ms"`
	TCPConnectMs  float64 `json:"tcp_connect_ms"`
	TLSHandshakeMs float64 `json:"tls_handshake_ms"`
	TimeToFirstByteMs float64 `json:"time_to_first_byte_ms"`
	ContentTransferMs float64 `json:"content_transfer_ms"`
	TotalMs       float64 `json:"total_ms"`
	Timestamp     time.Time `json:"timestamp"`
}

// CurlExport represents an exported curl command
type CurlExport struct {
	DeliveryID string `json:"delivery_id"`
	Command    string `json:"command"`
	Verbose    string `json:"verbose_command"`
}

// LiveInspectorEvent represents a real-time webhook event for the inspector
type LiveInspectorEvent struct {
	Type       string      `json:"type"`
	DeliveryID string      `json:"delivery_id"`
	EndpointID string      `json:"endpoint_id"`
	Status     string      `json:"status"`
	StatusCode int         `json:"status_code,omitempty"`
	DurationMs float64     `json:"duration_ms,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Error      string      `json:"error,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

// DeliveryInspection provides full request/response details for debugging
type DeliveryInspection struct {
	DeliveryID string              `json:"delivery_id"`
	Request    InspectionRequest   `json:"request"`
	Response   InspectionResponse  `json:"response"`
	Timing     TimingBreakdown     `json:"timing"`
	Retries    []RetryInfo         `json:"retries"`
	Curl       CurlExport          `json:"curl"`
}

// InspectionRequest contains the full outgoing request details
type InspectionRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	BodySize int64            `json:"body_size"`
}

// InspectionResponse contains the full response details
type InspectionResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	BodySize   int64             `json:"body_size"`
}

// RetryInfo contains information about a retry attempt
type RetryInfo struct {
	Attempt    int       `json:"attempt"`
	StatusCode int       `json:"status_code"`
	Error      string    `json:"error,omitempty"`
	DurationMs float64   `json:"duration_ms"`
	Timestamp  time.Time `json:"timestamp"`
}

// SearchFilter provides filtering for delivery search
type SearchFilter struct {
	TenantID    string     `json:"tenant_id"`
	EndpointID  string     `json:"endpoint_id,omitempty"`
	Status      string     `json:"status,omitempty"`
	StatusCode  int        `json:"status_code,omitempty"`
	MinDuration float64    `json:"min_duration_ms,omitempty"`
	MaxDuration float64    `json:"max_duration_ms,omitempty"`
	Since       *time.Time `json:"since,omitempty"`
	Until       *time.Time `json:"until,omitempty"`
	PayloadContains string `json:"payload_contains,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// InspectorEventType constants
const (
	EventTypeDeliveryStarted   = "delivery.started"
	EventTypeDeliveryCompleted = "delivery.completed"
	EventTypeDeliveryFailed    = "delivery.failed"
	EventTypeDeliveryRetrying  = "delivery.retrying"
	EventTypeEndpointDown      = "endpoint.down"
	EventTypeEndpointRecovered = "endpoint.recovered"
)

// GenerateCurlCommand generates a curl command from delivery request details
func GenerateCurlCommand(req InspectionRequest) CurlExport {
	var parts []string
	parts = append(parts, "curl")

	if req.Method != "" && req.Method != http.MethodGet {
		parts = append(parts, "-X", req.Method)
	}

	// Sort headers for deterministic output
	headerKeys := make([]string, 0, len(req.Headers))
	for k := range req.Headers {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)

	for _, k := range headerKeys {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, req.Headers[k]))
	}

	if req.Body != "" {
		escaped := strings.ReplaceAll(req.Body, "'", "'\\''")
		parts = append(parts, "-d", fmt.Sprintf("'%s'", escaped))
	}

	parts = append(parts, fmt.Sprintf("'%s'", req.URL))

	verboseParts := make([]string, len(parts))
	copy(verboseParts, parts)
	verboseParts = append(verboseParts[:1], append([]string{"-v", "--trace-time"}, verboseParts[1:]...)...)

	return CurlExport{
		Command: strings.Join(parts, " \\\n  "),
		Verbose: strings.Join(verboseParts, " \\\n  "),
	}
}

// ExtractTimingFromTrace computes timing breakdown from a trace's stages
func ExtractTimingFromTrace(trace *DeliveryTrace) TimingBreakdown {
	tb := TimingBreakdown{
		DeliveryID: trace.DeliveryID,
		TotalMs:    float64(trace.TotalDurMs),
		Timestamp:  trace.CreatedAt,
	}

	for _, stage := range trace.Stages {
		switch stage.Name {
		case "dns_lookup":
			tb.DNSLookupMs = float64(stage.DurationMs)
		case "tcp_connect":
			tb.TCPConnectMs = float64(stage.DurationMs)
		case "tls_handshake":
			tb.TLSHandshakeMs = float64(stage.DurationMs)
		case StageDelivery:
			tb.TimeToFirstByteMs = float64(stage.DurationMs)
		case StageResponse:
			tb.ContentTransferMs = float64(stage.DurationMs)
		}
	}

	return tb
}

// MatchesFilter checks if a delivery trace matches the given search filter
func MatchesFilter(trace *DeliveryTrace, filter SearchFilter) bool {
	if filter.EndpointID != "" && trace.EndpointID != filter.EndpointID {
		return false
	}
	if filter.Status != "" && trace.FinalStatus != filter.Status {
		return false
	}
	if filter.Since != nil && trace.CreatedAt.Before(*filter.Since) {
		return false
	}
	if filter.Until != nil && trace.CreatedAt.After(*filter.Until) {
		return false
	}
	if filter.MinDuration > 0 && float64(trace.TotalDurMs) < filter.MinDuration {
		return false
	}
	if filter.MaxDuration > 0 && float64(trace.TotalDurMs) > filter.MaxDuration {
		return false
	}
	if filter.PayloadContains != "" {
		found := false
		for _, stage := range trace.Stages {
			if strings.Contains(stage.Input, filter.PayloadContains) ||
				strings.Contains(stage.Output, filter.PayloadContains) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// BuildInspection creates a full delivery inspection from a trace
func BuildInspection(trace *DeliveryTrace) DeliveryInspection {
	inspection := DeliveryInspection{
		DeliveryID: trace.DeliveryID,
		Timing:     ExtractTimingFromTrace(trace),
	}

	for _, stage := range trace.Stages {
		switch stage.Name {
		case StageReceived:
			var headers map[string]string
			if stage.Metadata != nil {
				headers = stage.Metadata
			}
			var body string
			if stage.Input != "" {
				body = stage.Input
			}
			inspection.Request = InspectionRequest{
				Method:  "POST",
				Headers: headers,
				Body:    body,
				BodySize: int64(len(body)),
			}
			if url, ok := stage.Metadata["url"]; ok {
				inspection.Request.URL = url
			}
		case StageResponse:
			statusCode := 0
			if sc, ok := stage.Metadata["status_code"]; ok {
				fmt.Sscanf(sc, "%d", &statusCode)
			}
			inspection.Response = InspectionResponse{
				StatusCode: statusCode,
				Body:       stage.Output,
				BodySize:   int64(len(stage.Output)),
			}
		case StageRetry:
			retry := RetryInfo{
				Error:      stage.Error,
				DurationMs: float64(stage.DurationMs),
				Timestamp:  stage.Timestamp,
			}
			if attempt, ok := stage.Metadata["attempt"]; ok {
				fmt.Sscanf(attempt, "%d", &retry.Attempt)
			}
			if sc, ok := stage.Metadata["status_code"]; ok {
				fmt.Sscanf(sc, "%d", &retry.StatusCode)
			}
			inspection.Retries = append(inspection.Retries, retry)
		}
	}

	inspection.Curl = GenerateCurlCommand(inspection.Request)
	inspection.Curl.DeliveryID = trace.DeliveryID

	return inspection
}

// FormatAsJSON helper to pretty-print a value as JSON for inspection
func FormatAsJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// Package ai provides AI-powered debugging capabilities for webhook failures
package ai

import (
	"encoding/json"
	"time"
)

// ErrorCategory represents a classified error type
type ErrorCategory string

const (
	CategoryNetwork      ErrorCategory = "network"
	CategoryTimeout      ErrorCategory = "timeout"
	CategoryAuth         ErrorCategory = "authentication"
	CategoryRateLimit    ErrorCategory = "rate_limit"
	CategoryServerError  ErrorCategory = "server_error"
	CategoryClientError  ErrorCategory = "client_error"
	CategoryPayload      ErrorCategory = "payload"
	CategoryCertificate  ErrorCategory = "certificate"
	CategoryDNS          ErrorCategory = "dns"
	CategoryUnknown      ErrorCategory = "unknown"
)

// ErrorClassification represents a classified webhook error
type ErrorClassification struct {
	Category       ErrorCategory `json:"category"`
	Confidence     float64       `json:"confidence"`
	SubCategory    string        `json:"sub_category,omitempty"`
	IsRetryable    bool          `json:"is_retryable"`
	SuggestedDelay int           `json:"suggested_delay_seconds,omitempty"`
}

// DeliveryContext contains context about a failed webhook delivery
type DeliveryContext struct {
	DeliveryID     string            `json:"delivery_id"`
	EndpointID     string            `json:"endpoint_id"`
	TenantID       string            `json:"tenant_id"`
	URL            string            `json:"url"`
	HTTPMethod     string            `json:"http_method"`
	HTTPStatus     *int              `json:"http_status,omitempty"`
	ErrorMessage   string            `json:"error_message"`
	ResponseBody   string            `json:"response_body,omitempty"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	PayloadPreview string            `json:"payload_preview,omitempty"`
	AttemptNumber  int               `json:"attempt_number"`
	Latency        int64             `json:"latency_ms"`
	Timestamp      time.Time         `json:"timestamp"`
}

// DebugAnalysis represents the AI-generated analysis of a webhook failure
type DebugAnalysis struct {
	ID               string              `json:"id"`
	DeliveryID       string              `json:"delivery_id"`
	Classification   ErrorClassification `json:"classification"`
	RootCause        string              `json:"root_cause"`
	Explanation      string              `json:"explanation"`
	Suggestions      []Suggestion        `json:"suggestions"`
	TransformFix     *TransformFix       `json:"transform_fix,omitempty"`
	SimilarIssues    []SimilarIssue      `json:"similar_issues,omitempty"`
	ConfidenceScore  float64             `json:"confidence_score"`
	ProcessingTimeMs int64               `json:"processing_time_ms"`
	CreatedAt        time.Time           `json:"created_at"`
}

// Suggestion represents a fix suggestion
type Suggestion struct {
	Priority    int               `json:"priority"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Action      SuggestionAction  `json:"action"`
	CodeSnippet string            `json:"code_snippet,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// SuggestionAction represents the type of suggested action
type SuggestionAction string

const (
	ActionRetry            SuggestionAction = "retry"
	ActionUpdateHeaders    SuggestionAction = "update_headers"
	ActionUpdatePayload    SuggestionAction = "update_payload"
	ActionUpdateEndpoint   SuggestionAction = "update_endpoint"
	ActionAddTransform     SuggestionAction = "add_transform"
	ActionContactSupport   SuggestionAction = "contact_support"
	ActionCheckCertificate SuggestionAction = "check_certificate"
	ActionCheckDNS         SuggestionAction = "check_dns"
)

// TransformFix represents a suggested transformation script fix
type TransformFix struct {
	Description    string `json:"description"`
	Script         string `json:"script"`
	InputExample   string `json:"input_example,omitempty"`
	OutputExample  string `json:"output_example,omitempty"`
	Confidence     float64 `json:"confidence"`
}

// SimilarIssue represents a previously seen similar issue
type SimilarIssue struct {
	DeliveryID   string    `json:"delivery_id"`
	Similarity   float64   `json:"similarity"`
	Resolution   string    `json:"resolution,omitempty"`
	ResolvedAt   time.Time `json:"resolved_at,omitempty"`
}

// AnalysisRequest represents a request for AI analysis
type AnalysisRequest struct {
	DeliveryID       string `json:"delivery_id" binding:"required"`
	IncludeSimilar   bool   `json:"include_similar"`
	GenerateTransform bool   `json:"generate_transform"`
}

// BatchAnalysisRequest represents a batch analysis request
type BatchAnalysisRequest struct {
	DeliveryIDs      []string `json:"delivery_ids" binding:"required,min=1,max=50"`
	IncludeSimilar   bool     `json:"include_similar"`
	GenerateTransform bool     `json:"generate_transform"`
}

// BatchAnalysisResponse represents batch analysis results
type BatchAnalysisResponse struct {
	Analyses   []DebugAnalysis `json:"analyses"`
	Summary    AnalysisSummary `json:"summary"`
	TotalCount int             `json:"total_count"`
	FailedCount int            `json:"failed_count"`
}

// AnalysisSummary summarizes patterns across multiple failures
type AnalysisSummary struct {
	TopCategories     []CategoryCount   `json:"top_categories"`
	CommonRootCauses  []string          `json:"common_root_causes"`
	RecommendedAction string            `json:"recommended_action"`
	AffectedEndpoints []string          `json:"affected_endpoints"`
}

// CategoryCount represents error category frequency
type CategoryCount struct {
	Category ErrorCategory `json:"category"`
	Count    int           `json:"count"`
	Percent  float64       `json:"percent"`
}

// TransformGenerateRequest requests transformation script generation
type TransformGenerateRequest struct {
	Description   string          `json:"description" binding:"required"`
	InputExample  json.RawMessage `json:"input_example" binding:"required"`
	OutputExample json.RawMessage `json:"output_example" binding:"required"`
}

// TransformGenerateResponse contains the generated transformation
type TransformGenerateResponse struct {
	Script        string  `json:"script"`
	Explanation   string  `json:"explanation"`
	Confidence    float64 `json:"confidence"`
	TestResult    string  `json:"test_result,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

// ErrorPattern represents a learned error pattern
type ErrorPattern struct {
	ID            string            `json:"id" db:"id"`
	TenantID      string            `json:"tenant_id" db:"tenant_id"`
	Pattern       string            `json:"pattern" db:"pattern"`
	Category      ErrorCategory     `json:"category" db:"category"`
	Frequency     int               `json:"frequency" db:"frequency"`
	LastSeen      time.Time         `json:"last_seen" db:"last_seen"`
	Resolution    string            `json:"resolution,omitempty" db:"resolution"`
	Metadata      json.RawMessage   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
}

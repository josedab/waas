package dlq

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Root-cause category constants
const (
	RootCauseEndpointDown     = "endpoint_down"
	RootCauseTimeout          = "timeout"
	RootCauseRateLimit        = "rate_limit"
	RootCauseAuthFailure      = "auth_failure"
	RootCausePayloadRejected  = "payload_rejected"
	RootCauseDNSFailure       = "dns_failure"
	RootCauseTLSError         = "tls_error"
	RootCauseServerError      = "server_error"
	RootCauseUnknown          = "unknown"
)

// Severity constants for root-cause analysis
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// RootCauseAnalysis represents the result of analyzing a DLQ entry.
type RootCauseAnalysis struct {
	ID             string              `json:"id"`
	EntryID        string              `json:"entry_id"`
	TenantID       string              `json:"tenant_id"`
	EndpointID     string              `json:"endpoint_id"`
	Category       string              `json:"category"`
	Severity       string              `json:"severity"`
	Summary        string              `json:"summary"`
	Details        string              `json:"details"`
	Confidence     float64             `json:"confidence"`
	Suggestions    []RemediationAction `json:"suggestions"`
	RelatedEntries []string            `json:"related_entries,omitempty"`
	Patterns       []FailurePattern    `json:"patterns,omitempty"`
	AnalyzedAt     time.Time           `json:"analyzed_at"`
}

// RemediationAction suggests how to fix a root cause.
type RemediationAction struct {
	Action      string `json:"action"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	AutoFix     bool   `json:"auto_fix"`
}

// FailurePattern describes a pattern observed across failures.
type FailurePattern struct {
	Pattern     string  `json:"pattern"`
	Occurrences int     `json:"occurrences"`
	TimeSpan    string  `json:"time_span"`
	Confidence  float64 `json:"confidence"`
}

// EndpointHealthSummary aggregates failure data per endpoint.
type EndpointHealthSummary struct {
	EndpointID        string             `json:"endpoint_id"`
	TotalFailures     int                `json:"total_failures"`
	FailuresByCategory map[string]int    `json:"failures_by_category"`
	AvgResponseTimeMs float64            `json:"avg_response_time_ms"`
	LastFailureAt     *time.Time         `json:"last_failure_at,omitempty"`
	HealthScore       float64            `json:"health_score"`
	TopRootCauses     []RootCauseSummary `json:"top_root_causes"`
}

// RootCauseSummary is a compact root cause description.
type RootCauseSummary struct {
	Category   string  `json:"category"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// SmartRetryRecommendation suggests optimal retry parameters.
type SmartRetryRecommendation struct {
	EntryID         string `json:"entry_id"`
	ShouldRetry     bool   `json:"should_retry"`
	Reason          string `json:"reason"`
	RecommendedDelay string `json:"recommended_delay"`
	SuccessProbability float64 `json:"success_probability"`
}

// AnalyzeRootCause performs root-cause analysis on a DLQ entry.
func (s *Service) AnalyzeRootCause(ctx context.Context, entryID string) (*RootCauseAnalysis, error) {
	s.mu.RLock()
	entry, exists := s.entries[entryID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("DLQ entry %s not found", entryID)
	}

	analysis := &RootCauseAnalysis{
		ID:         uuid.New().String(),
		EntryID:    entryID,
		TenantID:   entry.TenantID,
		EndpointID: entry.EndpointID,
		AnalyzedAt: time.Now(),
	}

	// Determine root cause from error type and HTTP status
	analysis.Category, analysis.Confidence = classifyRootCause(entry)
	analysis.Severity = determineSeverity(entry, analysis.Category)
	analysis.Summary = generateSummary(analysis.Category, entry)
	analysis.Details = generateDetails(entry)
	analysis.Suggestions = generateSuggestions(analysis.Category, entry)

	// Find related entries with same endpoint/error pattern
	analysis.RelatedEntries = s.findRelatedEntries(entry)
	analysis.Patterns = s.detectPatterns(entry)

	return analysis, nil
}

// AnalyzeEndpointHealth aggregates root-cause data for an endpoint.
func (s *Service) AnalyzeEndpointHealth(ctx context.Context, tenantID, endpointID string) (*EndpointHealthSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &EndpointHealthSummary{
		EndpointID:         endpointID,
		FailuresByCategory: make(map[string]int),
	}

	var totalResponseTime float64
	var responseCount int

	for _, entry := range s.entries {
		if entry == nil || entry.TenantID != tenantID || entry.EndpointID != endpointID {
			continue
		}
		summary.TotalFailures++

		category, _ := classifyRootCause(entry)
		summary.FailuresByCategory[category]++

		if len(entry.AllAttempts) > 0 {
			lastAttempt := entry.AllAttempts[len(entry.AllAttempts)-1]
			totalResponseTime += float64(lastAttempt.DurationMs)
			responseCount++
			t := lastAttempt.AttemptedAt
			if summary.LastFailureAt == nil || t.After(*summary.LastFailureAt) {
				summary.LastFailureAt = &t
			}
		}
	}

	if responseCount > 0 {
		summary.AvgResponseTimeMs = totalResponseTime / float64(responseCount)
	}

	// Calculate health score (0-100, higher is healthier)
	if summary.TotalFailures == 0 {
		summary.HealthScore = 100
	} else if summary.TotalFailures < 5 {
		summary.HealthScore = 80
	} else if summary.TotalFailures < 20 {
		summary.HealthScore = 50
	} else if summary.TotalFailures < 100 {
		summary.HealthScore = 20
	} else {
		summary.HealthScore = 5
	}

	// Build top root causes
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range summary.FailuresByCategory {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

	for _, item := range sorted {
		pct := 0.0
		if summary.TotalFailures > 0 {
			pct = float64(item.v) / float64(summary.TotalFailures) * 100
		}
		summary.TopRootCauses = append(summary.TopRootCauses, RootCauseSummary{
			Category:   item.k,
			Count:      item.v,
			Percentage: pct,
		})
	}

	return summary, nil
}

// GetSmartRetryRecommendation provides a retry recommendation for a DLQ entry.
func (s *Service) GetSmartRetryRecommendation(ctx context.Context, entryID string) (*SmartRetryRecommendation, error) {
	s.mu.RLock()
	entry, exists := s.entries[entryID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("DLQ entry %s not found", entryID)
	}

	rec := &SmartRetryRecommendation{EntryID: entryID}

	category, _ := classifyRootCause(entry)

	switch category {
	case RootCauseEndpointDown:
		rec.ShouldRetry = true
		rec.Reason = "Endpoint was down; may have recovered"
		rec.RecommendedDelay = "5m"
		rec.SuccessProbability = 0.4
	case RootCauseTimeout:
		rec.ShouldRetry = true
		rec.Reason = "Request timed out; retry with longer timeout"
		rec.RecommendedDelay = "2m"
		rec.SuccessProbability = 0.6
	case RootCauseRateLimit:
		rec.ShouldRetry = true
		rec.Reason = "Rate limited; retry after backoff"
		rec.RecommendedDelay = "10m"
		rec.SuccessProbability = 0.8
	case RootCauseAuthFailure:
		rec.ShouldRetry = false
		rec.Reason = "Authentication failure; credentials need updating"
		rec.SuccessProbability = 0.05
	case RootCausePayloadRejected:
		rec.ShouldRetry = false
		rec.Reason = "Payload rejected by receiver; fix payload format"
		rec.SuccessProbability = 0.05
	case RootCauseTLSError:
		rec.ShouldRetry = false
		rec.Reason = "TLS certificate error; certificate needs renewal"
		rec.SuccessProbability = 0.1
	default:
		rec.ShouldRetry = true
		rec.Reason = "Unknown error; retry may succeed"
		rec.RecommendedDelay = "5m"
		rec.SuccessProbability = 0.3
	}

	// Reduce probability based on retry count
	if entry.RetryCount > 3 {
		rec.SuccessProbability *= 0.5
	}

	return rec, nil
}

func classifyRootCause(entry *DLQEntry) (string, float64) {
	errorType := strings.ToLower(entry.ErrorType)

	if strings.Contains(errorType, "timeout") {
		return RootCauseTimeout, 0.9
	}
	if strings.Contains(errorType, "dns") {
		return RootCauseDNSFailure, 0.95
	}
	if strings.Contains(errorType, "tls") || strings.Contains(errorType, "certificate") {
		return RootCauseTLSError, 0.9
	}

	if entry.FinalHTTPStatus != nil {
		status := *entry.FinalHTTPStatus
		switch {
		case status == 401 || status == 403:
			return RootCauseAuthFailure, 0.95
		case status == 429:
			return RootCauseRateLimit, 0.95
		case status == 400 || status == 422:
			return RootCausePayloadRejected, 0.85
		case status >= 500:
			return RootCauseServerError, 0.8
		case status == 0:
			return RootCauseEndpointDown, 0.7
		}
	}

	if strings.Contains(errorType, "connection") || strings.Contains(errorType, "refused") {
		return RootCauseEndpointDown, 0.85
	}

	return RootCauseUnknown, 0.3
}

func determineSeverity(entry *DLQEntry, category string) string {
	switch category {
	case RootCauseEndpointDown, RootCauseDNSFailure:
		return SeverityCritical
	case RootCauseAuthFailure, RootCauseTLSError:
		return SeverityHigh
	case RootCauseRateLimit, RootCauseServerError:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func generateSummary(category string, entry *DLQEntry) string {
	switch category {
	case RootCauseEndpointDown:
		return fmt.Sprintf("Endpoint %s appears to be down or unreachable", entry.EndpointID)
	case RootCauseTimeout:
		return fmt.Sprintf("Requests to endpoint %s are timing out", entry.EndpointID)
	case RootCauseRateLimit:
		return fmt.Sprintf("Endpoint %s is rate-limiting webhook deliveries", entry.EndpointID)
	case RootCauseAuthFailure:
		return fmt.Sprintf("Authentication failed for endpoint %s", entry.EndpointID)
	case RootCausePayloadRejected:
		return fmt.Sprintf("Payload was rejected by endpoint %s", entry.EndpointID)
	case RootCauseDNSFailure:
		return fmt.Sprintf("DNS resolution failed for endpoint %s", entry.EndpointID)
	case RootCauseTLSError:
		return fmt.Sprintf("TLS/SSL error when connecting to endpoint %s", entry.EndpointID)
	case RootCauseServerError:
		return fmt.Sprintf("Server error (5xx) from endpoint %s", entry.EndpointID)
	default:
		return fmt.Sprintf("Unknown failure for delivery to endpoint %s", entry.EndpointID)
	}
}

func generateDetails(entry *DLQEntry) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Total attempts: %d/%d", entry.RetryCount, entry.MaxRetries))

	if entry.FinalHTTPStatus != nil {
		parts = append(parts, fmt.Sprintf("Final HTTP status: %d", *entry.FinalHTTPStatus))
	}

	if entry.FinalResponseBody != nil && *entry.FinalResponseBody != "" {
		body := *entry.FinalResponseBody
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		parts = append(parts, fmt.Sprintf("Response: %s", body))
	}

	if len(entry.AllAttempts) > 0 {
		last := entry.AllAttempts[len(entry.AllAttempts)-1]
		parts = append(parts, fmt.Sprintf("Last attempt duration: %dms", last.DurationMs))
	}

	return strings.Join(parts, "; ")
}

func generateSuggestions(category string, entry *DLQEntry) []RemediationAction {
	var suggestions []RemediationAction

	switch category {
	case RootCauseEndpointDown:
		suggestions = append(suggestions,
			RemediationAction{Action: "check_endpoint", Description: "Verify the endpoint URL is correct and the server is running", Priority: 1, AutoFix: false},
			RemediationAction{Action: "retry_with_backoff", Description: "Retry delivery with exponential backoff", Priority: 2, AutoFix: true},
		)
	case RootCauseTimeout:
		suggestions = append(suggestions,
			RemediationAction{Action: "increase_timeout", Description: "Increase the delivery timeout for this endpoint", Priority: 1, AutoFix: true},
			RemediationAction{Action: "check_payload_size", Description: "Check if the payload size is causing slow processing", Priority: 2, AutoFix: false},
		)
	case RootCauseRateLimit:
		suggestions = append(suggestions,
			RemediationAction{Action: "reduce_rate", Description: "Reduce delivery rate to this endpoint", Priority: 1, AutoFix: true},
			RemediationAction{Action: "retry_delayed", Description: "Retry after a longer delay", Priority: 2, AutoFix: true},
		)
	case RootCauseAuthFailure:
		suggestions = append(suggestions,
			RemediationAction{Action: "update_credentials", Description: "Update webhook signing secret or API key", Priority: 1, AutoFix: false},
			RemediationAction{Action: "verify_signature", Description: "Verify signature algorithm matches endpoint expectations", Priority: 2, AutoFix: false},
		)
	case RootCausePayloadRejected:
		suggestions = append(suggestions,
			RemediationAction{Action: "validate_schema", Description: "Validate the payload against the endpoint's expected schema", Priority: 1, AutoFix: false},
			RemediationAction{Action: "check_content_type", Description: "Verify Content-Type header matches expected format", Priority: 2, AutoFix: false},
		)
	default:
		suggestions = append(suggestions,
			RemediationAction{Action: "manual_review", Description: "Review the delivery logs and error details", Priority: 1, AutoFix: false},
			RemediationAction{Action: "retry", Description: "Attempt manual retry", Priority: 2, AutoFix: true},
		)
	}

	return suggestions
}

func (s *Service) findRelatedEntries(entry *DLQEntry) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var related []string
	for _, other := range s.entries {
		if other.ID == entry.ID {
			continue
		}
		if other.EndpointID == entry.EndpointID && other.ErrorType == entry.ErrorType {
			related = append(related, other.ID)
			if len(related) >= 10 {
				break
			}
		}
	}
	return related
}

func (s *Service) detectPatterns(entry *DLQEntry) []FailurePattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	patternCounts := make(map[string]int)
	for _, other := range s.entries {
		if other.EndpointID == entry.EndpointID {
			key := other.ErrorType
			if other.FinalHTTPStatus != nil {
				key = fmt.Sprintf("%s:%d", other.ErrorType, *other.FinalHTTPStatus)
			}
			patternCounts[key]++
		}
	}

	var patterns []FailurePattern
	for pattern, count := range patternCounts {
		if count >= 2 {
			patterns = append(patterns, FailurePattern{
				Pattern:     pattern,
				Occurrences: count,
				TimeSpan:    "24h",
				Confidence:  float64(count) / float64(len(s.entries)),
			})
		}
	}

	return patterns
}

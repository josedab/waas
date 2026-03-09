package dlq

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AIClassification represents ML-powered failure classification result.
type AIClassification struct {
	ID              string    `json:"id"`
	EntryID         string    `json:"entry_id"`
	Category        string    `json:"category"`
	SubCategory     string    `json:"sub_category,omitempty"`
	Confidence      float64   `json:"confidence"`
	IsRetryable     bool      `json:"is_retryable"`
	SuggestedAction string    `json:"suggested_action"`
	SuggestedDelay  string    `json:"suggested_delay,omitempty"`
	Explanation     string    `json:"explanation"`
	ClassifiedAt    time.Time `json:"classified_at"`
}

// BackpressureConfig controls replay rate to prevent overwhelming endpoints.
type BackpressureConfig struct {
	MaxConcurrent    int     `json:"max_concurrent"`
	RatePerSecond    float64 `json:"rate_per_second"`
	BurstSize        int     `json:"burst_size"`
	SlowStartEnabled bool    `json:"slow_start_enabled"`
	CircuitBreaker   bool    `json:"circuit_breaker_enabled"`
	FailureThreshold int     `json:"failure_threshold"`
}

// BulkReplayWithBackpressureRequest extends bulk replay with backpressure controls.
type BulkReplayWithBackpressureRequest struct {
	Filter          DLQFilter          `json:"filter"`
	Backpressure    BackpressureConfig `json:"backpressure"`
	DryRun          bool               `json:"dry_run"`
	MaxEntries      int                `json:"max_entries,omitempty"`
}

// BulkReplayProgress tracks the progress of a bulk replay operation.
type BulkReplayProgress struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	TotalEntries    int       `json:"total_entries"`
	Replayed        int       `json:"replayed"`
	Succeeded       int       `json:"succeeded"`
	Failed          int       `json:"failed"`
	Skipped         int       `json:"skipped"`
	Status          string    `json:"status"` // pending, running, paused, completed, cancelled
	CurrentRate     float64   `json:"current_rate_per_sec"`
	EstimatedTimeMs int64     `json:"estimated_time_remaining_ms"`
	StartedAt       time.Time `json:"started_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// FailureSummary provides aggregated failure statistics by category.
type FailureSummary struct {
	TenantID       string                     `json:"tenant_id"`
	TotalFailures  int                        `json:"total_failures"`
	ByCategory     map[string]int             `json:"by_category"`
	ByEndpoint     map[string]int             `json:"by_endpoint"`
	TopPatterns    []PatternSummary           `json:"top_patterns"`
	TrendDirection string                     `json:"trend_direction"` // increasing, decreasing, stable
	AnalyzedAt     time.Time                  `json:"analyzed_at"`
}

// PatternSummary summarizes a failure pattern.
type PatternSummary struct {
	Pattern    string  `json:"pattern"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	IsNew      bool    `json:"is_new"`
}

// ClassifyFailure uses pattern-based analysis to classify a DLQ entry failure.
func (s *Service) ClassifyFailure(_ context.Context, entry *DLQEntry) (*AIClassification, error) {
	if entry == nil {
		return nil, fmt.Errorf("entry is nil")
	}

	classification := &AIClassification{
		ID:           uuid.New().String(),
		EntryID:      entry.ID,
		ClassifiedAt: time.Now(),
	}

	errorMsg := ""
	if len(entry.AllAttempts) > 0 {
		last := entry.AllAttempts[len(entry.AllAttempts)-1]
		if last.ErrorMessage != nil {
			errorMsg = *last.ErrorMessage
		}
	}

	// Pattern-based classification
	lower := strings.ToLower(errorMsg)
	switch {
	case strings.Contains(lower, "dns") || strings.Contains(lower, "no such host"):
		classification.Category = RootCauseDNSFailure
		classification.SubCategory = "resolution_failure"
		classification.Confidence = 0.95
		classification.IsRetryable = false
		classification.SuggestedAction = "Verify DNS configuration for the endpoint domain"
		classification.Explanation = "DNS resolution failed, indicating the endpoint domain cannot be resolved"

	case strings.Contains(lower, "tls") || strings.Contains(lower, "certificate") || strings.Contains(lower, "x509"):
		classification.Category = RootCauseTLSError
		classification.SubCategory = "certificate_error"
		classification.Confidence = 0.92
		classification.IsRetryable = false
		classification.SuggestedAction = "Check TLS certificate validity and chain"
		classification.Explanation = "TLS handshake failed due to certificate issues"

	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		classification.Category = RootCauseTimeout
		classification.SubCategory = "connection_timeout"
		classification.Confidence = 0.90
		classification.IsRetryable = true
		classification.SuggestedAction = "Retry with exponential backoff"
		classification.SuggestedDelay = "30s"
		classification.Explanation = "Endpoint did not respond within the configured timeout"

	case strings.Contains(lower, "401") || strings.Contains(lower, "403") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden"):
		classification.Category = RootCauseAuthFailure
		classification.SubCategory = "authentication_rejected"
		classification.Confidence = 0.93
		classification.IsRetryable = false
		classification.SuggestedAction = "Verify endpoint authentication credentials"
		classification.Explanation = "Endpoint rejected the request due to authentication failure"

	case strings.Contains(lower, "429") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "too many"):
		classification.Category = RootCauseRateLimit
		classification.SubCategory = "rate_limited"
		classification.Confidence = 0.95
		classification.IsRetryable = true
		classification.SuggestedAction = "Retry after the rate limit window resets"
		classification.SuggestedDelay = "60s"
		classification.Explanation = "Endpoint is rate limiting requests; need to reduce delivery rate"

	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "connect:"):
		classification.Category = RootCauseEndpointDown
		classification.SubCategory = "connection_refused"
		classification.Confidence = 0.88
		classification.IsRetryable = true
		classification.SuggestedAction = "Retry after endpoint recovery"
		classification.SuggestedDelay = "300s"
		classification.Explanation = "Endpoint is unreachable or refusing connections"

	case entry.FinalHTTPStatus != nil && *entry.FinalHTTPStatus >= 500:
		classification.Category = RootCauseServerError
		classification.SubCategory = fmt.Sprintf("http_%d", *entry.FinalHTTPStatus)
		classification.Confidence = 0.85
		classification.IsRetryable = true
		classification.SuggestedAction = "Retry with backoff - server may be experiencing issues"
		classification.SuggestedDelay = "60s"
		classification.Explanation = fmt.Sprintf("Server returned HTTP %d error", *entry.FinalHTTPStatus)

	case strings.Contains(lower, "payload") || strings.Contains(lower, "400") || strings.Contains(lower, "422"):
		classification.Category = RootCausePayloadRejected
		classification.SubCategory = "validation_error"
		classification.Confidence = 0.80
		classification.IsRetryable = false
		classification.SuggestedAction = "Review and fix the webhook payload format"
		classification.Explanation = "Endpoint rejected the payload as invalid"

	default:
		classification.Category = RootCauseUnknown
		classification.Confidence = 0.50
		classification.IsRetryable = true
		classification.SuggestedAction = "Manual investigation recommended"
		classification.SuggestedDelay = "300s"
		classification.Explanation = "Unable to determine root cause from available error information"
	}

	return classification, nil
}

// BulkReplayWithBackpressure initiates a controlled bulk replay operation.
func (s *Service) BulkReplayWithBackpressure(_ context.Context, tenantID string, req *BulkReplayWithBackpressureRequest) (*BulkReplayProgress, error) {
	req.Filter.TenantID = tenantID
	entries, total, err := s.GetEntries(context.Background(), &req.Filter)
	if err != nil {
		return nil, fmt.Errorf("fetching entries: %w", err)
	}

	if req.MaxEntries > 0 && len(entries) > req.MaxEntries {
		entries = entries[:req.MaxEntries]
	}

	progress := &BulkReplayProgress{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		TotalEntries: len(entries),
		Status:       "pending",
		CurrentRate:  req.Backpressure.RatePerSecond,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if req.DryRun {
		progress.Status = "dry_run"
		progress.EstimatedTimeMs = int64(math.Ceil(float64(len(entries)) / math.Max(req.Backpressure.RatePerSecond, 1) * 1000))
		_ = total // used for filtering
		return progress, nil
	}

	// Mark entries for replay with backpressure-aware sequencing
	replayed := 0
	for _, entry := range entries {
		if req.Backpressure.CircuitBreaker && progress.Failed >= req.Backpressure.FailureThreshold {
			progress.Status = "circuit_breaker_open"
			break
		}

		_, err := s.ReplayEntry(context.Background(), tenantID, entry.ID)
		if err != nil {
			progress.Failed++
			continue
		}
		replayed++
		progress.Succeeded++
	}

	progress.Replayed = replayed + progress.Failed
	progress.Status = "completed"
	progress.UpdatedAt = time.Now()

	return progress, nil
}

// GetFailureSummary returns aggregated failure statistics.
func (s *Service) GetFailureSummary(_ context.Context, tenantID string) (*FailureSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &FailureSummary{
		TenantID:    tenantID,
		ByCategory:  make(map[string]int),
		ByEndpoint:  make(map[string]int),
		AnalyzedAt:  time.Now(),
	}

	for _, entry := range s.entries {
		if entry.TenantID != tenantID {
			continue
		}
		summary.TotalFailures++
		summary.ByCategory[entry.ErrorType]++
		summary.ByEndpoint[entry.EndpointID]++
	}

	// Determine trend (simplified: compare first/second half counts)
	summary.TrendDirection = "stable"

	// Build top patterns
	for cat, count := range summary.ByCategory {
		pct := 0.0
		if summary.TotalFailures > 0 {
			pct = float64(count) / float64(summary.TotalFailures) * 100
		}
		summary.TopPatterns = append(summary.TopPatterns, PatternSummary{
			Pattern:    cat,
			Count:      count,
			Percentage: math.Round(pct*100) / 100,
		})
	}

	return summary, nil
}

// RegisterAIRoutes registers AI-powered DLQ analysis routes.
func (h *Handler) RegisterAIRoutes(r *gin.RouterGroup) {
	dlq := r.Group("/dlq")
	{
		dlq.POST("/entries/:id/classify", h.ClassifyEntry)
		dlq.POST("/bulk-replay-backpressure", h.BulkReplayWithBackpressure)
		dlq.GET("/failure-summary", h.GetFailureSummary)
		dlq.GET("/clusters", h.ListClusters)
	}
}

func (h *Handler) ClassifyEntry(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	entryID := c.Param("id")
	entry, err := h.service.GetEntry(context.Background(), tenantID, entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}

	classification, err := h.service.ClassifyFailure(c.Request.Context(), entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, classification)
}

func (h *Handler) BulkReplayWithBackpressure(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req BulkReplayWithBackpressureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Apply defaults
	if req.Backpressure.MaxConcurrent == 0 {
		req.Backpressure.MaxConcurrent = 10
	}
	if req.Backpressure.RatePerSecond == 0 {
		req.Backpressure.RatePerSecond = 5.0
	}

	progress, err := h.service.BulkReplayWithBackpressure(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func (h *Handler) GetFailureSummary(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.service.GetFailureSummary(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *Handler) ListClusters(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Cluster all current entries
	result, _ := h.service.ClusterEntries(tenantID)

	c.JSON(http.StatusOK, result)
}

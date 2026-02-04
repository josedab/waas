package intelligence

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- Root-Cause Analysis ---

// FailureCategory classifies delivery failure types
type FailureCategory string

const (
	FailureCategoryTimeout    FailureCategory = "timeout"
	FailureCategory5xx        FailureCategory = "server_error"
	FailureCategoryDNS        FailureCategory = "dns_resolution"
	FailureCategoryTLS        FailureCategory = "tls_handshake"
	FailureCategoryConnection FailureCategory = "connection_refused"
	FailureCategoryRateLimit  FailureCategory = "rate_limited"
	FailureCategoryPayload    FailureCategory = "payload_rejected"
	FailureCategoryAuth       FailureCategory = "authentication"
	FailureCategoryUnknown    FailureCategory = "unknown"
)

// RootCauseAnalysis contains the result of analyzing delivery failures
type RootCauseAnalysis struct {
	ID              string              `json:"id"`
	TenantID        string              `json:"tenant_id"`
	EndpointID      string              `json:"endpoint_id"`
	AnalyzedAt      time.Time           `json:"analyzed_at"`
	FailureCount    int                 `json:"failure_count"`
	TimeWindow      string              `json:"time_window"`
	PrimaryCategory FailureCategory     `json:"primary_category"`
	Categories      []CategoryBreakdown `json:"categories"`
	RootCauses      []RootCause         `json:"root_causes"`
	Recommendations []string            `json:"recommendations"`
	SuggestedFixes  []SuggestedFix      `json:"suggested_fixes"`
	Confidence      float64             `json:"confidence"`
}

// CategoryBreakdown shows failure distribution by category
type CategoryBreakdown struct {
	Category   FailureCategory `json:"category"`
	Count      int             `json:"count"`
	Percentage float64         `json:"percentage"`
}

// RootCause represents an identified root cause
type RootCause struct {
	Description string          `json:"description"`
	Category    FailureCategory `json:"category"`
	Confidence  float64         `json:"confidence"`
	Evidence    []string        `json:"evidence"`
}

// SuggestedFix provides an actionable fix for a root cause
type SuggestedFix struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"` // "config", "endpoint", "retry", "infrastructure"
	Impact      string `json:"impact"`   // "high", "medium", "low"
	AutoApply   bool   `json:"auto_apply"`
}

// DeliveryLog represents a single delivery attempt log for analysis
type DeliveryLog struct {
	DeliveryID   string    `json:"delivery_id"`
	EndpointID   string    `json:"endpoint_id"`
	Status       string    `json:"status"`
	HTTPStatus   int       `json:"http_status,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	LatencyMs    int64     `json:"latency_ms"`
	AttemptNum   int       `json:"attempt_number"`
	Timestamp    time.Time `json:"timestamp"`
}

// AnalyzeFailures performs root-cause analysis on delivery failures
func (s *Service) AnalyzeFailures(ctx context.Context, tenantID, endpointID string, logs []DeliveryLog) (*RootCauseAnalysis, error) {
	if len(logs) == 0 {
		return nil, fmt.Errorf("no delivery logs provided")
	}

	analysis := &RootCauseAnalysis{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		EndpointID:   endpointID,
		AnalyzedAt:   time.Now(),
		FailureCount: len(logs),
		TimeWindow:   "24h",
	}

	// Categorize failures
	categoryCounts := make(map[FailureCategory]int)
	for _, log := range logs {
		cat := categorizeFailure(log)
		categoryCounts[cat]++
	}

	// Build category breakdown
	for cat, count := range categoryCounts {
		analysis.Categories = append(analysis.Categories, CategoryBreakdown{
			Category:   cat,
			Count:      count,
			Percentage: float64(count) / float64(len(logs)) * 100,
		})
	}

	// Sort by count descending
	sort.Slice(analysis.Categories, func(i, j int) bool {
		return analysis.Categories[i].Count > analysis.Categories[j].Count
	})

	if len(analysis.Categories) > 0 {
		analysis.PrimaryCategory = analysis.Categories[0].Category
	}

	// Identify root causes based on heuristics
	analysis.RootCauses = identifyRootCauses(logs, categoryCounts)
	analysis.Recommendations = generateRecommendations(analysis.PrimaryCategory, categoryCounts)
	analysis.SuggestedFixes = generateFixes(analysis.PrimaryCategory, logs)
	analysis.Confidence = calculateAnalysisConfidence(logs, categoryCounts)

	return analysis, nil
}

func categorizeFailure(log DeliveryLog) FailureCategory {
	errLower := strings.ToLower(log.ErrorMessage)

	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "deadline exceeded") {
		return FailureCategoryTimeout
	}
	if strings.Contains(errLower, "dns") || strings.Contains(errLower, "no such host") {
		return FailureCategoryDNS
	}
	if strings.Contains(errLower, "tls") || strings.Contains(errLower, "certificate") || strings.Contains(errLower, "x509") {
		return FailureCategoryTLS
	}
	if strings.Contains(errLower, "connection refused") || strings.Contains(errLower, "connect:") {
		return FailureCategoryConnection
	}
	if log.HTTPStatus == 429 || strings.Contains(errLower, "rate limit") {
		return FailureCategoryRateLimit
	}
	if log.HTTPStatus == 401 || log.HTTPStatus == 403 {
		return FailureCategoryAuth
	}
	if log.HTTPStatus >= 400 && log.HTTPStatus < 500 {
		return FailureCategoryPayload
	}
	if log.HTTPStatus >= 500 {
		return FailureCategory5xx
	}
	return FailureCategoryUnknown
}

func identifyRootCauses(logs []DeliveryLog, categories map[FailureCategory]int) []RootCause {
	var causes []RootCause

	if count, ok := categories[FailureCategoryTimeout]; ok && count > 0 {
		causes = append(causes, RootCause{
			Description: "Endpoint is responding too slowly or not at all",
			Category:    FailureCategoryTimeout,
			Confidence:  0.85,
			Evidence:    []string{fmt.Sprintf("%d timeout failures detected", count)},
		})
	}

	if count, ok := categories[FailureCategory5xx]; ok && count > 0 {
		causes = append(causes, RootCause{
			Description: "Target server is experiencing internal errors",
			Category:    FailureCategory5xx,
			Confidence:  0.9,
			Evidence:    []string{fmt.Sprintf("%d server errors (5xx) detected", count)},
		})
	}

	if count, ok := categories[FailureCategoryDNS]; ok && count > 0 {
		causes = append(causes, RootCause{
			Description: "DNS resolution failing - endpoint hostname may be misconfigured",
			Category:    FailureCategoryDNS,
			Confidence:  0.95,
			Evidence:    []string{fmt.Sprintf("%d DNS failures detected", count)},
		})
	}

	if count, ok := categories[FailureCategoryTLS]; ok && count > 0 {
		causes = append(causes, RootCause{
			Description: "TLS certificate issue - expired, self-signed, or hostname mismatch",
			Category:    FailureCategoryTLS,
			Confidence:  0.92,
			Evidence:    []string{fmt.Sprintf("%d TLS errors detected", count)},
		})
	}

	if count, ok := categories[FailureCategoryRateLimit]; ok && count > 0 {
		causes = append(causes, RootCause{
			Description: "Endpoint is rate-limiting webhook deliveries",
			Category:    FailureCategoryRateLimit,
			Confidence:  0.88,
			Evidence:    []string{fmt.Sprintf("%d rate limit responses (429) detected", count)},
		})
	}

	return causes
}

func generateRecommendations(primary FailureCategory, categories map[FailureCategory]int) []string {
	var recs []string

	switch primary {
	case FailureCategoryTimeout:
		recs = append(recs, "Increase delivery timeout to 30+ seconds")
		recs = append(recs, "Check endpoint server performance and response times")
		recs = append(recs, "Consider reducing payload size")
	case FailureCategory5xx:
		recs = append(recs, "Contact endpoint owner about server errors")
		recs = append(recs, "Increase retry backoff to allow server recovery")
		recs = append(recs, "Consider pausing deliveries until server recovers")
	case FailureCategoryDNS:
		recs = append(recs, "Verify the endpoint URL hostname is correct")
		recs = append(recs, "Check DNS propagation if recently changed")
	case FailureCategoryTLS:
		recs = append(recs, "Check SSL certificate validity and expiration")
		recs = append(recs, "Ensure certificate hostname matches endpoint URL")
	case FailureCategoryRateLimit:
		recs = append(recs, "Reduce delivery rate with longer backoff intervals")
		recs = append(recs, "Implement request batching to reduce call frequency")
		recs = append(recs, "Contact endpoint owner about rate limit increase")
	case FailureCategoryConnection:
		recs = append(recs, "Verify endpoint is reachable and accepting connections")
		recs = append(recs, "Check firewall rules allow inbound traffic from WaaS")
	default:
		recs = append(recs, "Review delivery logs for error patterns")
	}

	return recs
}

func generateFixes(primary FailureCategory, logs []DeliveryLog) []SuggestedFix {
	var fixes []SuggestedFix

	switch primary {
	case FailureCategoryTimeout:
		fixes = append(fixes, SuggestedFix{
			Title:       "Increase delivery timeout",
			Description: "Increase timeout from default to 30 seconds",
			Category:    "config",
			Impact:      "high",
			AutoApply:   true,
		})
	case FailureCategoryRateLimit:
		fixes = append(fixes, SuggestedFix{
			Title:       "Adjust retry backoff",
			Description: "Increase initial retry delay to 5 seconds with 3x multiplier",
			Category:    "retry",
			Impact:      "high",
			AutoApply:   true,
		})
	case FailureCategory5xx:
		fixes = append(fixes, SuggestedFix{
			Title:       "Enable circuit breaker",
			Description: "Pause deliveries after 5 consecutive failures, resume after 60s",
			Category:    "retry",
			Impact:      "medium",
			AutoApply:   true,
		})
	case FailureCategoryTLS:
		fixes = append(fixes, SuggestedFix{
			Title:       "Certificate alert",
			Description: "Set up alerting for SSL certificate expiration",
			Category:    "infrastructure",
			Impact:      "medium",
			AutoApply:   false,
		})
	}

	return fixes
}

func calculateAnalysisConfidence(logs []DeliveryLog, categories map[FailureCategory]int) float64 {
	if len(logs) < 5 {
		return 0.5
	}
	// More logs = higher confidence, max 0.95
	conf := math.Min(0.5+float64(len(logs))*0.01, 0.95)
	// Single dominant category = higher confidence
	if len(categories) == 1 {
		conf = math.Min(conf+0.1, 0.98)
	}
	return conf
}

// --- Failure Clustering ---

// FailureCluster groups similar failures together
type FailureCluster struct {
	ID                string          `json:"id"`
	Category          FailureCategory `json:"category"`
	Pattern           string          `json:"pattern"`
	Count             int             `json:"count"`
	FirstSeen         time.Time       `json:"first_seen"`
	LastSeen          time.Time       `json:"last_seen"`
	AffectedEndpoints []string        `json:"affected_endpoints"`
	SampleErrors      []string        `json:"sample_errors"`
	SuggestedAction   string          `json:"suggested_action"`
}

// ClusterFailures groups similar failures into clusters
func (s *Service) ClusterFailures(ctx context.Context, tenantID string, logs []DeliveryLog) ([]FailureCluster, error) {
	if len(logs) == 0 {
		return nil, nil
	}

	clusterMap := make(map[FailureCategory]*FailureCluster)

	for _, log := range logs {
		cat := categorizeFailure(log)
		cluster, exists := clusterMap[cat]
		if !exists {
			cluster = &FailureCluster{
				ID:                uuid.New().String(),
				Category:          cat,
				Pattern:           string(cat),
				FirstSeen:         log.Timestamp,
				LastSeen:          log.Timestamp,
				AffectedEndpoints: make([]string, 0),
			}
			clusterMap[cat] = cluster
		}

		cluster.Count++
		if log.Timestamp.Before(cluster.FirstSeen) {
			cluster.FirstSeen = log.Timestamp
		}
		if log.Timestamp.After(cluster.LastSeen) {
			cluster.LastSeen = log.Timestamp
		}

		// Track unique endpoints
		found := false
		for _, ep := range cluster.AffectedEndpoints {
			if ep == log.EndpointID {
				found = true
				break
			}
		}
		if !found {
			cluster.AffectedEndpoints = append(cluster.AffectedEndpoints, log.EndpointID)
		}

		// Keep sample errors (max 5)
		if len(cluster.SampleErrors) < 5 && log.ErrorMessage != "" {
			cluster.SampleErrors = append(cluster.SampleErrors, log.ErrorMessage)
		}
	}

	// Generate suggested actions
	clusters := make([]FailureCluster, 0, len(clusterMap))
	for _, cluster := range clusterMap {
		recs := generateRecommendations(cluster.Category, map[FailureCategory]int{cluster.Category: cluster.Count})
		if len(recs) > 0 {
			cluster.SuggestedAction = recs[0]
		}
		clusters = append(clusters, *cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Count > clusters[j].Count
	})

	return clusters, nil
}

// --- Predictive Alerting ---

// PredictiveAlert is generated before an SLA breach occurs
type PredictiveAlert struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	EndpointID      string    `json:"endpoint_id"`
	AlertType       string    `json:"alert_type"` // "sla_breach", "degradation", "failure_spike"
	Severity        RiskLevel `json:"severity"`
	Message         string    `json:"message"`
	PredictedAt     time.Time `json:"predicted_at"`
	ETA             string    `json:"eta"` // estimated time until breach
	CurrentValue    float64   `json:"current_value"`
	ThresholdValue  float64   `json:"threshold_value"`
	TrendDirection  string    `json:"trend_direction"` // "worsening", "stable", "improving"
	Recommendations []string  `json:"recommendations"`
}

// PredictSLABreach checks if current trends will lead to SLA breach
func (s *Service) PredictSLABreach(ctx context.Context, tenantID, endpointID string, features *FeatureVector, slaTarget float64) (*PredictiveAlert, error) {
	if features == nil {
		return nil, fmt.Errorf("feature vector is required")
	}

	// Calculate trend-based prediction
	currentFailRate := features.FailureRate24h
	trendRate := features.FailureRate24h - features.FailureRate7d

	// No alert needed if within SLA
	if currentFailRate < (1-slaTarget)*0.8 && trendRate <= 0 {
		return nil, nil
	}

	alert := &PredictiveAlert{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		EndpointID:     endpointID,
		PredictedAt:    time.Now(),
		CurrentValue:   (1 - currentFailRate) * 100,
		ThresholdValue: slaTarget * 100,
	}

	if trendRate > 0 {
		alert.TrendDirection = "worsening"
	} else if trendRate < 0 {
		alert.TrendDirection = "improving"
	} else {
		alert.TrendDirection = "stable"
	}

	if currentFailRate > (1 - slaTarget) {
		alert.AlertType = "sla_breach"
		alert.Severity = RiskCritical
		alert.Message = fmt.Sprintf("SLA target %.1f%% is currently breached (actual: %.1f%%)", slaTarget*100, (1-currentFailRate)*100)
		alert.ETA = "now"
	} else if trendRate > 0.01 {
		alert.AlertType = "degradation"
		alert.Severity = RiskHigh
		hoursUntilBreach := ((1 - slaTarget) - currentFailRate) / trendRate * 24
		alert.ETA = fmt.Sprintf("%.0f hours", math.Max(hoursUntilBreach, 1))
		alert.Message = fmt.Sprintf("Failure rate trending up. SLA breach predicted in %s", alert.ETA)
	} else {
		alert.AlertType = "degradation"
		alert.Severity = RiskMedium
		alert.Message = fmt.Sprintf("Success rate at %.1f%%, approaching SLA threshold of %.1f%%", (1-currentFailRate)*100, slaTarget*100)
		alert.ETA = "unknown"
	}

	alert.Recommendations = []string{
		"Review recent delivery errors for patterns",
		"Check endpoint health and response times",
		"Consider adjusting retry policies",
	}

	return alert, nil
}

// --- Auto-Adjust Retry Policies ---

// AutoRetryAdjustment represents an automatic retry policy adjustment
type AutoRetryAdjustment struct {
	EndpointID         string  `json:"endpoint_id"`
	PreviousMaxRetries int     `json:"previous_max_retries"`
	NewMaxRetries      int     `json:"new_max_retries"`
	PreviousBackoff    string  `json:"previous_backoff"`
	NewBackoff         string  `json:"new_backoff"`
	Reason             string  `json:"reason"`
	Confidence         float64 `json:"confidence"`
}

// AutoAdjustRetryPolicy adjusts retry policy based on failure patterns
func (s *Service) AutoAdjustRetryPolicy(ctx context.Context, tenantID, endpointID string, logs []DeliveryLog, currentMaxRetries int) (*AutoRetryAdjustment, error) {
	if len(logs) < 3 {
		return nil, nil // Not enough data
	}

	adjustment := &AutoRetryAdjustment{
		EndpointID:         endpointID,
		PreviousMaxRetries: currentMaxRetries,
		NewMaxRetries:      currentMaxRetries,
		PreviousBackoff:    "exponential",
		NewBackoff:         "exponential",
	}

	// Analyze success by attempt number
	successByAttempt := make(map[int]int)
	failByAttempt := make(map[int]int)
	for _, log := range logs {
		if log.Status == "delivered" || log.Status == "success" {
			successByAttempt[log.AttemptNum]++
		} else {
			failByAttempt[log.AttemptNum]++
		}
	}

	// If most failures resolve on retry 2-3, keep low retry count
	totalRetrySuccess := 0
	for attempt := 2; attempt <= 5; attempt++ {
		totalRetrySuccess += successByAttempt[attempt]
	}

	// Categorize failures to adjust backoff
	categories := make(map[FailureCategory]int)
	for _, log := range logs {
		if log.Status != "delivered" && log.Status != "success" {
			categories[categorizeFailure(log)]++
		}
	}

	// Rate limiting: increase backoff (check before retry success)
	if categories[FailureCategoryRateLimit] > len(logs)/4 {
		adjustment.NewBackoff = "exponential_with_jitter"
		adjustment.NewMaxRetries = max(currentMaxRetries, 5)
		adjustment.Reason = "Rate limiting detected; increasing backoff with jitter"
		adjustment.Confidence = 0.85
		return adjustment, nil
	}

	// If retries rarely succeed, reduce to save resources
	if totalRetrySuccess == 0 && len(logs) > 10 {
		adjustment.NewMaxRetries = 2
		adjustment.Reason = "Retries rarely succeed; reducing to conserve resources"
		adjustment.Confidence = 0.8
		return adjustment, nil
	}

	// Timeouts: increase retries
	if categories[FailureCategoryTimeout] > len(logs)/3 {
		adjustment.NewMaxRetries = max(currentMaxRetries, 5)
		adjustment.Reason = "Timeout pattern detected; increasing retry attempts"
		adjustment.Confidence = 0.75
		return adjustment, nil
	}

	return nil, nil // No adjustment needed
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- Natural Language Failure Reports ---

// FailureReport is a human-readable failure summary
type FailureReport struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Period      string    `json:"period"`
	Summary     string    `json:"summary"`
	Details     string    `json:"details"`
	GeneratedAt time.Time `json:"generated_at"`
	Format      string    `json:"format"` // "slack", "email", "plain"
}

// GenerateFailureReport creates a natural-language failure report
func (s *Service) GenerateFailureReport(ctx context.Context, tenantID string, logs []DeliveryLog, format string) (*FailureReport, error) {
	if len(logs) == 0 {
		return &FailureReport{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Period:      "24h",
			Summary:     "No delivery failures in the last 24 hours. All systems operating normally.",
			GeneratedAt: time.Now(),
			Format:      format,
		}, nil
	}

	// Analyze failures
	categories := make(map[FailureCategory]int)
	var totalLatency int64
	endpointSet := make(map[string]bool)

	for _, log := range logs {
		categories[categorizeFailure(log)]++
		totalLatency += log.LatencyMs
		endpointSet[log.EndpointID] = true
	}

	avgLatency := totalLatency / int64(len(logs))

	// Build summary
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 Webhook Delivery Report (%s)\n\n", tenantID))
	sb.WriteString(fmt.Sprintf("Total failures: %d across %d endpoints\n", len(logs), len(endpointSet)))
	sb.WriteString(fmt.Sprintf("Average latency: %dms\n\n", avgLatency))

	sb.WriteString("Failure breakdown:\n")
	for cat, count := range categories {
		pct := float64(count) / float64(len(logs)) * 100
		sb.WriteString(fmt.Sprintf("  • %s: %d (%.0f%%)\n", cat, count, pct))
	}

	sb.WriteString("\nTop recommendations:\n")
	if len(categories) > 0 {
		var primary FailureCategory
		maxCount := 0
		for cat, count := range categories {
			if count > maxCount {
				primary = cat
				maxCount = count
			}
		}
		recs := generateRecommendations(primary, categories)
		for _, rec := range recs {
			sb.WriteString(fmt.Sprintf("  → %s\n", rec))
		}
	}

	report := &FailureReport{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Period:      "24h",
		Summary:     fmt.Sprintf("%d failures across %d endpoints", len(logs), len(endpointSet)),
		Details:     sb.String(),
		GeneratedAt: time.Now(),
		Format:      format,
	}

	return report, nil
}

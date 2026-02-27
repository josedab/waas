package dlq

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FailureCluster represents a group of DLQ entries with similar failure patterns
type FailureCluster struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Category     string            `json:"category"`
	Pattern      string            `json:"pattern"`
	Severity     string            `json:"severity"`
	EntryCount   int               `json:"entry_count"`
	EntryIDs     []string          `json:"entry_ids"`
	EndpointIDs  []string          `json:"endpoint_ids"`
	FirstSeenAt  time.Time         `json:"first_seen_at"`
	LastSeenAt   time.Time         `json:"last_seen_at"`
	RootCause    *ClusterRootCause `json:"root_cause"`
	ReplayStatus string            `json:"replay_status"` // pending, replaying, completed
	CreatedAt    time.Time         `json:"created_at"`
}

// ClusterRootCause provides detailed root-cause analysis for a cluster
type ClusterRootCause struct {
	Category         string              `json:"category"`
	ConfidenceScore  float64             `json:"confidence_score"` // 0.0 - 1.0
	Summary          string              `json:"summary"`
	TechnicalDetails string              `json:"technical_details"`
	ImpactAssessment string              `json:"impact_assessment"`
	Remediations     []RemediationAction `json:"remediations"`
	SimilarIncidents int                 `json:"similar_incidents"`
	EstimatedTTR     string              `json:"estimated_ttr"` // time to resolve
}

// ClusteringResult contains the result of a clustering operation
type ClusteringResult struct {
	TenantID     string            `json:"tenant_id"`
	TotalEntries int               `json:"total_entries"`
	Clusters     []*FailureCluster `json:"clusters"`
	Unclustered  int               `json:"unclustered"`
	ClusteredAt  time.Time         `json:"clustered_at"`
}

// BulkReplayClusterRequest requests bulk replay for a cluster
type BulkReplayClusterRequest struct {
	ClusterID string `json:"cluster_id" binding:"required"`
	RateLimit int    `json:"rate_limit,omitempty"`
	DryRun    bool   `json:"dry_run"`
}

// BulkReplayClusterResult contains the result of a cluster replay
type BulkReplayClusterResult struct {
	ClusterID    string `json:"cluster_id"`
	TotalEntries int    `json:"total_entries"`
	Replayed     int    `json:"replayed"`
	Failed       int    `json:"failed"`
	Skipped      int    `json:"skipped"`
	DryRun       bool   `json:"dry_run"`
}

// ClusterEntries groups DLQ entries into failure clusters
func (s *Service) ClusterEntries(tenantID string) (*ClusteringResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &ClusteringResult{
		TenantID:    tenantID,
		ClusteredAt: time.Now(),
	}

	// Collect entries for tenant
	var tenantEntries []*DLQEntry
	for _, entry := range s.entries {
		if entry.TenantID == tenantID {
			tenantEntries = append(tenantEntries, entry)
		}
	}
	result.TotalEntries = len(tenantEntries)

	// Group by failure signature (category + error pattern)
	clusterMap := make(map[string]*FailureCluster)
	for _, entry := range tenantEntries {
		category, _ := classifyRootCause(entry)
		sig := buildClusterSignature(category, entry)

		cluster, exists := clusterMap[sig]
		if !exists {
			cluster = &FailureCluster{
				ID:           uuid.New().String(),
				TenantID:     tenantID,
				Category:     category,
				Pattern:      sig,
				FirstSeenAt:  entry.CreatedAt,
				LastSeenAt:   entry.CreatedAt,
				CreatedAt:    time.Now(),
				ReplayStatus: "pending",
			}
			clusterMap[sig] = cluster
		}

		cluster.EntryCount++
		cluster.EntryIDs = append(cluster.EntryIDs, entry.ID)

		if !containsStr(cluster.EndpointIDs, entry.EndpointID) {
			cluster.EndpointIDs = append(cluster.EndpointIDs, entry.EndpointID)
		}

		if entry.CreatedAt.Before(cluster.FirstSeenAt) {
			cluster.FirstSeenAt = entry.CreatedAt
		}
		if entry.CreatedAt.After(cluster.LastSeenAt) {
			cluster.LastSeenAt = entry.CreatedAt
		}
	}

	// Analyze each cluster's root cause
	for _, cluster := range clusterMap {
		cluster.Severity = clusterSeverity(cluster)
		cluster.RootCause = analyzeClusterRootCause(cluster)
		result.Clusters = append(result.Clusters, cluster)
	}

	// Sort by entry count (largest clusters first)
	sort.Slice(result.Clusters, func(i, j int) bool {
		return result.Clusters[i].EntryCount > result.Clusters[j].EntryCount
	})

	return result, nil
}

func buildClusterSignature(category string, entry *DLQEntry) string {
	parts := []string{category}

	if entry.FinalHTTPStatus != nil && *entry.FinalHTTPStatus > 0 {
		statusClass := (*entry.FinalHTTPStatus / 100) * 100
		parts = append(parts, fmt.Sprintf("http_%d", statusClass))
	}

	errNorm := normalizeError(entry.ErrorType)
	if errNorm != "" {
		parts = append(parts, errNorm)
	}

	return strings.Join(parts, ":")
}

func normalizeError(errMsg string) string {
	msg := strings.ToLower(errMsg)

	patterns := map[string]string{
		"connection refused": "conn_refused",
		"connection reset":   "conn_reset",
		"timeout":            "timeout",
		"no such host":       "dns_failure",
		"tls":                "tls_error",
		"certificate":        "tls_cert",
		"eof":                "conn_closed",
		"rate limit":         "rate_limited",
		"too many requests":  "rate_limited",
		"unauthorized":       "auth_failure",
		"forbidden":          "auth_failure",
	}

	for pattern, normalized := range patterns {
		if strings.Contains(msg, pattern) {
			return normalized
		}
	}

	return ""
}

func clusterSeverity(cluster *FailureCluster) string {
	if cluster.EntryCount >= 100 || len(cluster.EndpointIDs) >= 10 {
		return SeverityCritical
	}
	if cluster.EntryCount >= 50 || len(cluster.EndpointIDs) >= 5 {
		return SeverityHigh
	}
	if cluster.EntryCount >= 10 {
		return SeverityMedium
	}
	return SeverityLow
}

func analyzeClusterRootCause(cluster *FailureCluster) *ClusterRootCause {
	rca := &ClusterRootCause{
		Category:         cluster.Category,
		SimilarIncidents: cluster.EntryCount,
	}

	// Compute confidence based on cluster size and pattern consistency
	if cluster.EntryCount >= 10 {
		rca.ConfidenceScore = math.Min(0.95, 0.7+float64(cluster.EntryCount)*0.005)
	} else {
		rca.ConfidenceScore = 0.5 + float64(cluster.EntryCount)*0.05
	}

	switch cluster.Category {
	case RootCauseEndpointDown:
		rca.Summary = fmt.Sprintf("Endpoint unreachable - %d deliveries affected across %d endpoint(s)",
			cluster.EntryCount, len(cluster.EndpointIDs))
		rca.TechnicalDetails = "The target endpoint is refusing connections or not responding. This typically indicates the endpoint server is down, misconfigured, or the URL has changed."
		rca.ImpactAssessment = fmt.Sprintf("%d webhook deliveries failed. Affected endpoints: %s",
			cluster.EntryCount, strings.Join(cluster.EndpointIDs, ", "))
		rca.EstimatedTTR = "depends on endpoint owner"
		rca.Remediations = []RemediationAction{
			{Action: "verify_endpoint", Description: "Verify the endpoint URL is correct and reachable", Priority: 1, AutoFix: false},
			{Action: "check_firewall", Description: "Check if firewall rules are blocking the connection", Priority: 2, AutoFix: false},
			{Action: "retry_batch", Description: "Retry all failed deliveries once endpoint is restored", Priority: 3, AutoFix: true},
		}

	case RootCauseTimeout:
		rca.Summary = fmt.Sprintf("Request timeout - %d deliveries timed out", cluster.EntryCount)
		rca.TechnicalDetails = "Deliveries are timing out before receiving a response. This may indicate the endpoint is slow, overloaded, or performing heavy processing synchronously."
		rca.ImpactAssessment = fmt.Sprintf("%d deliveries timed out", cluster.EntryCount)
		rca.EstimatedTTR = "minutes to hours"
		rca.Remediations = []RemediationAction{
			{Action: "increase_timeout", Description: "Increase the delivery timeout threshold", Priority: 1, AutoFix: true},
			{Action: "optimize_endpoint", Description: "Optimize the endpoint to respond within the timeout", Priority: 2, AutoFix: false},
			{Action: "async_processing", Description: "Return 200 immediately and process asynchronously", Priority: 3, AutoFix: false},
		}

	case RootCauseRateLimit:
		rca.Summary = fmt.Sprintf("Rate limited - %d deliveries throttled", cluster.EntryCount)
		rca.TechnicalDetails = "The target endpoint is returning rate limit responses (429 Too Many Requests)."
		rca.ImpactAssessment = fmt.Sprintf("%d deliveries rate limited", cluster.EntryCount)
		rca.EstimatedTTR = "automatic after backoff"
		rca.Remediations = []RemediationAction{
			{Action: "apply_backoff", Description: "Apply exponential backoff between retries", Priority: 1, AutoFix: true},
			{Action: "reduce_rate", Description: "Reduce webhook delivery rate to the endpoint", Priority: 2, AutoFix: true},
			{Action: "contact_owner", Description: "Ask endpoint owner to increase rate limits", Priority: 3, AutoFix: false},
		}

	case RootCauseAuthFailure:
		rca.Summary = fmt.Sprintf("Authentication failure - %d deliveries rejected", cluster.EntryCount)
		rca.TechnicalDetails = "The endpoint is returning 401/403 responses indicating invalid or expired credentials."
		rca.ImpactAssessment = fmt.Sprintf("%d deliveries rejected due to auth failure", cluster.EntryCount)
		rca.EstimatedTTR = "minutes"
		rca.Remediations = []RemediationAction{
			{Action: "rotate_secret", Description: "Rotate the webhook signing secret", Priority: 1, AutoFix: false},
			{Action: "verify_credentials", Description: "Verify the authentication credentials are current", Priority: 2, AutoFix: false},
		}

	case RootCauseDNSFailure:
		rca.Summary = fmt.Sprintf("DNS resolution failure - %d deliveries affected", cluster.EntryCount)
		rca.TechnicalDetails = "Unable to resolve the endpoint hostname via DNS."
		rca.ImpactAssessment = fmt.Sprintf("%d deliveries failed due to DNS resolution", cluster.EntryCount)
		rca.EstimatedTTR = "minutes to hours"
		rca.Remediations = []RemediationAction{
			{Action: "verify_dns", Description: "Verify the endpoint hostname resolves correctly", Priority: 1, AutoFix: false},
			{Action: "check_domain", Description: "Check if the domain registration is active", Priority: 2, AutoFix: false},
		}

	case RootCauseTLSError:
		rca.Summary = fmt.Sprintf("TLS/SSL error - %d deliveries affected", cluster.EntryCount)
		rca.TechnicalDetails = "TLS handshake failed. Certificate may be expired, self-signed, or using unsupported protocols."
		rca.ImpactAssessment = fmt.Sprintf("%d deliveries failed due to TLS errors", cluster.EntryCount)
		rca.EstimatedTTR = "minutes"
		rca.Remediations = []RemediationAction{
			{Action: "renew_cert", Description: "Renew the SSL/TLS certificate", Priority: 1, AutoFix: false},
			{Action: "check_chain", Description: "Verify the full certificate chain is valid", Priority: 2, AutoFix: false},
		}

	case RootCauseServerError:
		rca.Summary = fmt.Sprintf("Server error (5xx) - %d deliveries failed", cluster.EntryCount)
		rca.TechnicalDetails = "The endpoint is returning 5xx server error responses."
		rca.ImpactAssessment = fmt.Sprintf("%d deliveries received server errors", cluster.EntryCount)
		rca.EstimatedTTR = "varies"
		rca.Remediations = []RemediationAction{
			{Action: "retry_with_backoff", Description: "Retry with exponential backoff", Priority: 1, AutoFix: true},
			{Action: "check_logs", Description: "Check endpoint server logs for errors", Priority: 2, AutoFix: false},
		}

	default:
		rca.Summary = fmt.Sprintf("Unknown failure - %d entries", cluster.EntryCount)
		rca.TechnicalDetails = "The failure pattern could not be automatically classified."
		rca.ConfidenceScore = 0.3
		rca.Remediations = []RemediationAction{
			{Action: "manual_review", Description: "Manually review the failed deliveries", Priority: 1, AutoFix: false},
		}
	}

	return rca
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

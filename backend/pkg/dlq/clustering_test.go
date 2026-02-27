package dlq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClusterEntries(t *testing.T) {
	svc := NewService()

	// Add entries with different failure patterns
	now := time.Now()
	httpStatus429 := 429
	entries := []*DLQEntry{
		{ID: "1", TenantID: "t1", EndpointID: "ep1", ErrorType: "connection refused", CreatedAt: now},
		{ID: "2", TenantID: "t1", EndpointID: "ep1", ErrorType: "connection refused", CreatedAt: now},
		{ID: "3", TenantID: "t1", EndpointID: "ep2", ErrorType: "connection refused", CreatedAt: now},
		{ID: "4", TenantID: "t1", EndpointID: "ep1", ErrorType: "timeout exceeded", CreatedAt: now},
		{ID: "5", TenantID: "t1", EndpointID: "ep1", ErrorType: "timeout exceeded", CreatedAt: now},
		{ID: "6", TenantID: "t1", EndpointID: "ep3", ErrorType: "429 too many requests", FinalHTTPStatus: &httpStatus429, CreatedAt: now},
	}

	for _, e := range entries {
		svc.mu.Lock()
		svc.entries[e.ID] = e
		svc.mu.Unlock()
	}

	result, err := svc.ClusterEntries("t1")
	assert.NoError(t, err)
	assert.Equal(t, 6, result.TotalEntries)
	assert.GreaterOrEqual(t, len(result.Clusters), 2, "should have at least 2 clusters")

	// Largest cluster should be connection refused (3 entries)
	assert.Equal(t, 3, result.Clusters[0].EntryCount)
}

func TestBuildClusterSignature(t *testing.T) {
	httpStatus504 := 504
	httpStatus429 := 429
	tests := []struct {
		name         string
		category     string
		entry        *DLQEntry
		wantContains string
	}{
		{
			name:         "connection refused",
			category:     RootCauseEndpointDown,
			entry:        &DLQEntry{ErrorType: "connection refused"},
			wantContains: "conn_refused",
		},
		{
			name:         "timeout with 500",
			category:     RootCauseTimeout,
			entry:        &DLQEntry{ErrorType: "timeout exceeded", FinalHTTPStatus: &httpStatus504},
			wantContains: "timeout",
		},
		{
			name:         "rate limit 429",
			category:     RootCauseRateLimit,
			entry:        &DLQEntry{ErrorType: "rate limit exceeded", FinalHTTPStatus: &httpStatus429},
			wantContains: "rate_limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := buildClusterSignature(tt.category, tt.entry)
			assert.Contains(t, sig, tt.wantContains)
		})
	}
}

func TestNormalizeError(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"connection refused", "conn_refused"},
		{"Connection Reset by peer", "conn_reset"},
		{"dial tcp: i/o timeout", "timeout"},
		{"no such host: api.example.com", "dns_failure"},
		{"tls: handshake failure", "tls_error"},
		{"x509: certificate expired", "tls_cert"},
		{"unknown error", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeError(tt.input))
		})
	}
}

func TestClusterSeverity(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		endpoints int
		want      string
	}{
		{"critical by count", 100, 1, SeverityCritical},
		{"critical by endpoints", 5, 10, SeverityCritical},
		{"high", 50, 3, SeverityHigh},
		{"medium", 10, 1, SeverityMedium},
		{"low", 3, 1, SeverityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &FailureCluster{
				EntryCount:  tt.count,
				EndpointIDs: make([]string, tt.endpoints),
			}
			assert.Equal(t, tt.want, clusterSeverity(cluster))
		})
	}
}

func TestAnalyzeClusterRootCause(t *testing.T) {
	categories := []string{
		RootCauseEndpointDown, RootCauseTimeout, RootCauseRateLimit,
		RootCauseAuthFailure, RootCauseDNSFailure, RootCauseTLSError,
		RootCauseServerError, RootCauseUnknown,
	}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			cluster := &FailureCluster{
				Category:    cat,
				EntryCount:  25,
				EndpointIDs: []string{"ep-1"},
			}

			rca := analyzeClusterRootCause(cluster)
			assert.NotEmpty(t, rca.Summary)
			assert.NotEmpty(t, rca.Remediations)
			assert.Greater(t, rca.ConfidenceScore, 0.0)
			assert.Equal(t, cat, rca.Category)
		})
	}
}

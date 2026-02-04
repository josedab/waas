package intelligence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeFailures(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	logs := []DeliveryLog{
		{DeliveryID: "d1", EndpointID: "e1", Status: "failed", HTTPStatus: 503, ErrorMessage: "server error", LatencyMs: 100, AttemptNum: 1, Timestamp: time.Now()},
		{DeliveryID: "d2", EndpointID: "e1", Status: "failed", HTTPStatus: 503, ErrorMessage: "server error", LatencyMs: 200, AttemptNum: 1, Timestamp: time.Now()},
		{DeliveryID: "d3", EndpointID: "e1", Status: "failed", HTTPStatus: 0, ErrorMessage: "timeout exceeded", LatencyMs: 30000, AttemptNum: 1, Timestamp: time.Now()},
	}

	analysis, err := svc.AnalyzeFailures(ctx, "tenant-1", "e1", logs)
	require.NoError(t, err)
	assert.Equal(t, 3, analysis.FailureCount)
	assert.True(t, len(analysis.Categories) > 0)
	assert.True(t, len(analysis.RootCauses) > 0)
	assert.True(t, len(analysis.Recommendations) > 0)
}

func TestAnalyzeFailures_Empty(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.AnalyzeFailures(context.Background(), "t", "e", nil)
	assert.Error(t, err)
}

func TestCategorizeFailure(t *testing.T) {
	tests := []struct {
		log      DeliveryLog
		expected FailureCategory
	}{
		{DeliveryLog{ErrorMessage: "connection timeout"}, FailureCategoryTimeout},
		{DeliveryLog{HTTPStatus: 503}, FailureCategory5xx},
		{DeliveryLog{ErrorMessage: "no such host"}, FailureCategoryDNS},
		{DeliveryLog{ErrorMessage: "tls handshake failure"}, FailureCategoryTLS},
		{DeliveryLog{ErrorMessage: "connection refused"}, FailureCategoryConnection},
		{DeliveryLog{HTTPStatus: 429}, FailureCategoryRateLimit},
		{DeliveryLog{HTTPStatus: 401}, FailureCategoryAuth},
		{DeliveryLog{HTTPStatus: 400}, FailureCategoryPayload},
		{DeliveryLog{}, FailureCategoryUnknown},
	}

	for _, tt := range tests {
		result := categorizeFailure(tt.log)
		assert.Equal(t, tt.expected, result, "for error: %s, status: %d", tt.log.ErrorMessage, tt.log.HTTPStatus)
	}
}

func TestClusterFailures(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	logs := []DeliveryLog{
		{EndpointID: "e1", ErrorMessage: "timeout", Timestamp: time.Now()},
		{EndpointID: "e1", ErrorMessage: "timeout", Timestamp: time.Now()},
		{EndpointID: "e2", ErrorMessage: "timeout", Timestamp: time.Now()},
		{EndpointID: "e2", HTTPStatus: 503, Timestamp: time.Now()},
	}

	clusters, err := svc.ClusterFailures(ctx, "t1", logs)
	require.NoError(t, err)
	assert.True(t, len(clusters) >= 2)
	assert.Equal(t, 3, clusters[0].Count) // timeout cluster
}

func TestPredictSLABreach(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	t.Run("no alert when healthy", func(t *testing.T) {
		features := &FeatureVector{FailureRate24h: 0.001, FailureRate7d: 0.001}
		alert, err := svc.PredictSLABreach(ctx, "t1", "e1", features, 0.99)
		require.NoError(t, err)
		assert.Nil(t, alert)
	})

	t.Run("alert when breached", func(t *testing.T) {
		features := &FeatureVector{FailureRate24h: 0.05, FailureRate7d: 0.02}
		alert, err := svc.PredictSLABreach(ctx, "t1", "e1", features, 0.99)
		require.NoError(t, err)
		require.NotNil(t, alert)
		assert.Equal(t, "sla_breach", alert.AlertType)
		assert.Equal(t, RiskCritical, alert.Severity)
	})
}

func TestAutoAdjustRetryPolicy(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	t.Run("too few logs", func(t *testing.T) {
		adj, err := svc.AutoAdjustRetryPolicy(ctx, "t1", "e1", []DeliveryLog{{}, {}}, 3)
		require.NoError(t, err)
		assert.Nil(t, adj)
	})

	t.Run("rate limited adjusts backoff", func(t *testing.T) {
		logs := make([]DeliveryLog, 20)
		for i := range logs {
			logs[i] = DeliveryLog{Status: "failed", HTTPStatus: 429, AttemptNum: 1}
		}
		adj, err := svc.AutoAdjustRetryPolicy(ctx, "t1", "e1", logs, 3)
		require.NoError(t, err)
		require.NotNil(t, adj)
		assert.Equal(t, "exponential_with_jitter", adj.NewBackoff)
	})
}

func TestGenerateFailureReport(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	t.Run("no failures", func(t *testing.T) {
		report, err := svc.GenerateFailureReport(ctx, "t1", nil, "plain")
		require.NoError(t, err)
		assert.Contains(t, report.Summary, "No delivery failures")
	})

	t.Run("with failures", func(t *testing.T) {
		logs := []DeliveryLog{
			{EndpointID: "e1", Status: "failed", HTTPStatus: 503, ErrorMessage: "server error", LatencyMs: 100},
			{EndpointID: "e2", Status: "failed", ErrorMessage: "timeout", LatencyMs: 30000},
		}
		report, err := svc.GenerateFailureReport(ctx, "t1", logs, "slack")
		require.NoError(t, err)
		assert.Contains(t, report.Details, "Total failures: 2")
		assert.Equal(t, "slack", report.Format)
	})
}

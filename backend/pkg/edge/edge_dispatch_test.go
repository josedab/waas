package edge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEdgeDispatchConfig(t *testing.T) {
	svc := NewService(nil, DefaultServiceConfig())

	config, err := svc.CreateEdgeDispatchConfig(context.Background(), "tenant-edge-1", &CreateEdgeDispatchConfigRequest{
		Strategy:         DispatchStrategyGeo,
		PreferredRegions: []string{"us-east-1", "eu-west-1"},
		MaxLatencyMs:     200,
		EnableFailover:   true,
		FailoverRegions:  []string{"us-west-2"},
	})
	require.NoError(t, err)
	assert.Equal(t, DispatchStrategyGeo, config.Strategy)
	assert.True(t, config.Enabled)
	assert.Equal(t, 200, config.MaxLatencyMs)
}

func TestCreateEdgeDispatchConfigInvalidStrategy(t *testing.T) {
	svc := NewService(nil, DefaultServiceConfig())

	_, err := svc.CreateEdgeDispatchConfig(context.Background(), "t1", &CreateEdgeDispatchConfigRequest{
		Strategy: "invalid",
	})
	assert.Error(t, err)
}

func TestDispatchWebhook(t *testing.T) {
	svc := NewService(nil, DefaultServiceConfig())

	result, err := svc.DispatchWebhook(context.Background(), "tenant-edge-2", &DispatchWebhookRequest{
		WebhookID:   "wh-1",
		EndpointURL: "https://api.example.com/webhook",
		ReceiverLat: 51.5, // London
		ReceiverLon: -0.1,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.NodeID)
	assert.NotEmpty(t, result.Region)
	assert.True(t, result.Success)
	assert.Equal(t, 200, result.StatusCode)
}

func TestDispatchWebhookValidation(t *testing.T) {
	svc := NewService(nil, DefaultServiceConfig())

	_, err := svc.DispatchWebhook(context.Background(), "t1", &DispatchWebhookRequest{})
	assert.Error(t, err)
}

func TestDispatchWebhookWithConfig(t *testing.T) {
	svc := NewService(nil, DefaultServiceConfig())

	_, _ = svc.CreateEdgeDispatchConfig(context.Background(), "tenant-edge-3", &CreateEdgeDispatchConfigRequest{
		Strategy:         DispatchStrategyGeo,
		PreferredRegions: []string{"eu-west-1"},
	})

	result, err := svc.DispatchWebhook(context.Background(), "tenant-edge-3", &DispatchWebhookRequest{
		WebhookID:   "wh-2",
		EndpointURL: "https://api.example.com/webhook",
	})
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", result.Region)
}

func TestGetEdgeNetworkOverview(t *testing.T) {
	svc := NewService(nil, DefaultServiceConfig())

	overview, err := svc.GetEdgeNetworkOverview(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, 8, overview.TotalNodes)
	assert.Equal(t, 8, overview.ActiveNodes)
	assert.NotEmpty(t, overview.Regions)
	assert.Equal(t, float64(100), overview.HealthScore)
}

func TestHaversine(t *testing.T) {
	// New York to London ≈ 5570 km
	dist := haversine(40.7, -74.0, 51.5, -0.1)
	assert.InDelta(t, 5570, dist, 100)
}

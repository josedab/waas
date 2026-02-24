package smartlimit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAdaptiveConfig(t *testing.T) {
	svc := NewService(nil, nil)
	svc.resetAdaptiveState()

	config, err := svc.CreateAdaptiveConfig(context.Background(), "tenant-1", &CreateRecvAdaptiveConfigRequest{
		EndpointID:        "ep-1",
		Strategy:          StrategyAIMD,
		BaseRatePerSecond: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "ep-1", config.EndpointID)
	assert.Equal(t, float64(100), config.BaseRatePerSecond)
	assert.Equal(t, float64(100), config.CurrentRate)
	assert.Equal(t, StrategyAIMD, config.Strategy)
	assert.True(t, config.Enabled)
}

func TestCreateAdaptiveConfigValidation(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.CreateAdaptiveConfig(context.Background(), "t1", &CreateRecvAdaptiveConfigRequest{})
	assert.Error(t, err)

	_, err = svc.CreateAdaptiveConfig(context.Background(), "t1", &CreateRecvAdaptiveConfigRequest{
		EndpointID: "ep-1", BaseRatePerSecond: -1,
	})
	assert.Error(t, err)
}

func TestRecordDeliveryResultAdaptive(t *testing.T) {
	svc := NewService(nil, nil)

	// Create config first
	_, _ = svc.CreateAdaptiveConfig(context.Background(), "tenant-1", &CreateRecvAdaptiveConfigRequest{
		EndpointID: "ep-adapt-1", BaseRatePerSecond: 100,
	})

	// Record successful deliveries
	for i := 0; i < 10; i++ {
		health, err := svc.RecordRecvDeliveryResult(context.Background(), "tenant-1", &RecvDeliveryResultRequest{
			EndpointID:     "ep-adapt-1",
			StatusCode:     200,
			ResponseTimeMs: 50,
			Success:        true,
		})
		require.NoError(t, err)
		assert.Equal(t, HealthStatusHealthy, health.Status)
	}

	// Check rate increased
	config, err := svc.GetAdaptiveConfig(context.Background(), "tenant-1", "ep-adapt-1")
	require.NoError(t, err)
	assert.Greater(t, config.CurrentRate, float64(100))
}

func TestRecordDeliveryResultDegrades(t *testing.T) {
	svc := NewService(nil, nil)

	_, _ = svc.CreateAdaptiveConfig(context.Background(), "tenant-1", &CreateRecvAdaptiveConfigRequest{
		EndpointID: "ep-adapt-2", BaseRatePerSecond: 100,
	})

	// Record mixed results (low success rate)
	for i := 0; i < 10; i++ {
		success := i < 3 // Only 30% success
		_, _ = svc.RecordRecvDeliveryResult(context.Background(), "tenant-1", &RecvDeliveryResultRequest{
			EndpointID:     "ep-adapt-2",
			StatusCode:     200,
			ResponseTimeMs: 100,
			Success:        success,
		})
	}

	config, err := svc.GetAdaptiveConfig(context.Background(), "tenant-1", "ep-adapt-2")
	require.NoError(t, err)
	assert.Less(t, config.CurrentRate, float64(100))
}

func TestGetReceiverHealth(t *testing.T) {
	svc := NewService(nil, nil)

	health, err := svc.GetReceiverHealth(context.Background(), "tenant-1", "unknown-ep")
	require.NoError(t, err)
	assert.Equal(t, HealthStatusUnknown, health.Status)
}

func TestGetAdaptiveStats(t *testing.T) {
	svc := NewService(nil, nil)

	stats, err := svc.GetAdaptiveStats(context.Background(), "tenant-1", "ep-1")
	require.NoError(t, err)
	assert.Equal(t, "ep-1", stats.EndpointID)
}

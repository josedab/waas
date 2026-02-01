package multicloud_test

import (
	"testing"

	"github.com/josedab/waas/pkg/multicloud"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationClusterCRUD(t *testing.T) {
	t.Skip("Integration test - requires database")

	service := multicloud.NewFederationService(nil, nil, multicloud.DefaultFederationConfig())
	require.NotNil(t, service)
}

func TestFederationConfigDefaults(t *testing.T) {
	config := multicloud.DefaultFederationConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 30, config.HealthCheckIntervalSec)
	assert.Equal(t, 3, config.FailoverThreshold)
	assert.True(t, config.EnableAutoFailover)
}

func TestClusterStatusConstants(t *testing.T) {
	statuses := []multicloud.ClusterStatus{
		multicloud.ClusterHealthy,
		multicloud.ClusterDegraded,
		multicloud.ClusterUnhealthy,
		multicloud.ClusterDraining,
		multicloud.ClusterOffline,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}

func TestRoutingStrategyConstants(t *testing.T) {
	strategies := []multicloud.RoutingStrategy{
		multicloud.RoutingLatency,
		multicloud.RoutingGeo,
		multicloud.RoutingRoundRobin,
		multicloud.RoutingWeighted,
		multicloud.RoutingFailover,
	}

	for _, strategy := range strategies {
		assert.NotEmpty(t, string(strategy))
	}
}

func TestFailoverModeConstants(t *testing.T) {
	modes := []multicloud.FailoverMode{
		multicloud.FailoverAutomatic,
		multicloud.FailoverManual,
		multicloud.FailoverNone,
	}

	for _, mode := range modes {
		assert.NotEmpty(t, string(mode))
	}
}

func TestProviderConstants(t *testing.T) {
	providers := []multicloud.Provider{
		multicloud.ProviderAWSEventBridge,
		multicloud.ProviderGCPPubSub,
		multicloud.ProviderAzureEventGrid,
		multicloud.ProviderCustom,
	}

	for _, provider := range providers {
		assert.NotEmpty(t, string(provider))
	}
}

func TestFederationClusterStructure(t *testing.T) {
	cluster := &multicloud.FederationCluster{
		ID:       "cluster-123",
		TenantID: "tenant-456",
		Name:     "us-west-primary",
		Provider: multicloud.ProviderAWSEventBridge,
		Region:   "us-west-2",
		Endpoint: "https://waas-us-west.example.com",
	}

	assert.Equal(t, "cluster-123", cluster.ID)
	assert.Equal(t, "us-west-primary", cluster.Name)
	assert.Equal(t, multicloud.ProviderAWSEventBridge, cluster.Provider)
}

func TestFederationRouteStructure(t *testing.T) {
	route := &multicloud.FederationRoute{
		ID:       "route-123",
		TenantID: "tenant-456",
		Name:     "US traffic routing",
		Strategy: multicloud.RoutingGeo,
		Active:   true,
	}

	assert.Equal(t, "route-123", route.ID)
	assert.Equal(t, multicloud.RoutingGeo, route.Strategy)
	assert.True(t, route.Active)
}

func TestClusterCapacityStructure(t *testing.T) {
	capacity := multicloud.ClusterCapacity{
		MaxRPS:            10000,
		CurrentRPS:        5000,
		MaxConcurrency:    1000,
		CPUUtilization:    0.75,
		MemoryUtilization: 0.60,
		Available:         true,
	}

	assert.Equal(t, 10000, capacity.MaxRPS)
	assert.Equal(t, 5000, capacity.CurrentRPS)
	assert.True(t, capacity.Available)
}

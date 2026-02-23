package dataplane

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvisionPlane_Dedicated(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	plane, err := svc.ProvisionPlane(context.Background(), &ProvisionPlaneRequest{
		TenantID:  "tenant-enterprise",
		PlaneType: PlaneTypeDedicated,
		Region:    "eu-west-1",
	})

	require.NoError(t, err)
	assert.Equal(t, PlaneTypeDedicated, plane.PlaneType)
	assert.Equal(t, "eu-west-1", plane.Region)
	assert.Contains(t, plane.DBSchema, "tenant_")
	assert.Contains(t, plane.RedisNamespace, "waas:")
}

func TestProvisionPlane_InvalidType(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.ProvisionPlane(context.Background(), &ProvisionPlaneRequest{
		TenantID:  "tenant-1",
		PlaneType: "premium",
	})

	assert.Error(t, err)
}

func TestProvisionPlane_DefaultRegion(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	plane, err := svc.ProvisionPlane(context.Background(), &ProvisionPlaneRequest{
		TenantID:  "tenant-2",
		PlaneType: PlaneTypeShared,
	})

	require.NoError(t, err)
	assert.Equal(t, "us-east-1", plane.Region)
}

func TestProvisionPlane_DefaultConfig(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	plane, err := svc.ProvisionPlane(context.Background(), &ProvisionPlaneRequest{
		TenantID:  "tenant-3",
		PlaneType: PlaneTypeIsolated,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, plane.Config)
}

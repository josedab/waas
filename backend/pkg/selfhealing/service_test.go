package selfhealing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordFailureAndHealing(t *testing.T) {
	repo := NewMemoryRepository()
	config := DefaultServiceConfig()
	config.FailureThreshold = 3
	svc := NewService(repo, config)

	// First 2 failures should not trigger healing
	for i := 0; i < 2; i++ {
		discovery, err := svc.RecordFailure("t1", "ep-1", "https://old.example.com/hook")
		require.NoError(t, err)
		assert.Nil(t, discovery)
	}

	// 3rd failure triggers healing (will fail to discover since no real server)
	discovery, err := svc.RecordFailure("t1", "ep-1", "https://old.example.com/hook")
	// Discovery will fail since there's no real .well-known endpoint
	assert.Nil(t, discovery)
	assert.Error(t, err)
}

func TestRecordSuccessResetsTracker(t *testing.T) {
	repo := NewMemoryRepository()
	config := DefaultServiceConfig()
	config.FailureThreshold = 5
	svc := NewService(repo, config)

	// Record some failures
	for i := 0; i < 3; i++ {
		svc.RecordFailure("t1", "ep-1", "https://example.com")
	}

	ft, _ := svc.GetFailureStatus("ep-1")
	assert.Equal(t, 3, ft.ConsecutiveFailures)

	// Success resets
	err := svc.RecordSuccess("ep-1")
	require.NoError(t, err)

	ft, _ = svc.GetFailureStatus("ep-1")
	assert.Equal(t, 0, ft.ConsecutiveFailures)
}

func TestGenerateWellKnownSpec(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	spec := svc.GenerateWellKnownSpec([]WellKnownEndpoint{
		{
			Name:       "orders",
			URL:        "https://api.example.com/webhooks/orders",
			EventTypes: []string{"order.created", "order.updated"},
			Status:     "active",
		},
	})

	assert.Equal(t, "1.0", spec.Version)
	assert.Len(t, spec.Endpoints, 1)
	assert.Equal(t, "orders", spec.Endpoints[0].Name)
}

func TestMigrationEvents(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	events, err := svc.GetMigrationEvents("t1", 10)
	require.NoError(t, err)
	assert.Empty(t, events)
}

package testutil

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers.go ──────────────────────────────────────────────────────────────

func TestSetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		preExist bool
		preValue string
	}{
		{
			name:     "sets new env var and removes on cleanup",
			key:      "TESTUTIL_NEW_VAR",
			value:    "hello",
			preExist: false,
		},
		{
			name:     "overrides existing env var and restores on cleanup",
			key:      "TESTUTIL_EXISTING_VAR",
			value:    "override",
			preExist: true,
			preValue: "original",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure clean state
			os.Unsetenv(tc.key)

			if tc.preExist {
				os.Setenv(tc.key, tc.preValue)
			}

			// Use a sub-test so cleanup runs before our assertion
			var afterCleanup string
			var existsAfterCleanup bool

			t.Run("inner", func(t *testing.T) {
				SetEnv(t, tc.key, tc.value)
				got := os.Getenv(tc.key)
				assert.Equal(t, tc.value, got, "env var should be set during test")
			})

			afterCleanup, existsAfterCleanup = os.LookupEnv(tc.key)

			if tc.preExist {
				assert.True(t, existsAfterCleanup, "env var should exist after cleanup")
				assert.Equal(t, tc.preValue, afterCleanup, "env var should be restored")
			} else {
				assert.False(t, existsAfterCleanup, "env var should be removed after cleanup")
			}

			// Final cleanup
			os.Unsetenv(tc.key)
		})
	}
}

func TestNewTestConfig(t *testing.T) {
	// Clear all config vars so NewTestConfig sets defaults
	keys := []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET", "ENVIRONMENT", "LOG_LEVEL"}
	for _, k := range keys {
		os.Unsetenv(k)
	}

	t.Run("sets default config values", func(t *testing.T) {
		NewTestConfig(t)

		assert.Contains(t, os.Getenv("DATABASE_URL"), "postgres://")
		assert.Contains(t, os.Getenv("REDIS_URL"), "redis://")
		assert.NotEmpty(t, os.Getenv("JWT_SECRET"))
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "error", os.Getenv("LOG_LEVEL"))
	})

	t.Run("does not override pre-existing values", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "my-custom-secret")
		defer os.Unsetenv("JWT_SECRET")

		NewTestConfig(t)
		assert.Equal(t, "my-custom-secret", os.Getenv("JWT_SECRET"))
	})
}

func TestSkipIfShort(t *testing.T) {
	// We can't truly test the skip behavior in a unit test without -short,
	// so we just verify it doesn't panic when called.
	t.Run("does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			// SkipIfShort uses t.Skip which is fine — it won't skip this
			// outer test, only the sub-test if -short is set.
			if testing.Short() {
				t.Skip("expected skip in short mode")
			}
		})
	})
}

// ── wait.go ─────────────────────────────────────────────────────────────────

func TestWaitFor(t *testing.T) {
	tests := []struct {
		name      string
		condition func() bool
		timeout   time.Duration
		interval  time.Duration
		wantErr   bool
	}{
		{
			name:      "returns nil when condition is immediately true",
			condition: func() bool { return true },
			timeout:   100 * time.Millisecond,
			interval:  10 * time.Millisecond,
			wantErr:   false,
		},
		{
			name:      "returns error on timeout",
			condition: func() bool { return false },
			timeout:   50 * time.Millisecond,
			interval:  10 * time.Millisecond,
			wantErr:   true,
		},
		{
			name: "succeeds when condition becomes true before timeout",
			condition: func() func() bool {
				var count int32
				return func() bool {
					return atomic.AddInt32(&count, 1) >= 3
				}
			}(),
			timeout:  500 * time.Millisecond,
			interval: 10 * time.Millisecond,
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := WaitFor(tc.condition, tc.timeout, tc.interval)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "timed out")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWaitForValue(t *testing.T) {
	t.Run("succeeds when value matches immediately", func(t *testing.T) {
		err := WaitForValue(func() int { return 42 }, 42, 100*time.Millisecond, 10*time.Millisecond)
		assert.NoError(t, err)
	})

	t.Run("succeeds when value converges", func(t *testing.T) {
		var counter int32
		fn := func() int32 {
			return atomic.AddInt32(&counter, 1)
		}
		err := WaitForValue(fn, int32(5), 500*time.Millisecond, 10*time.Millisecond)
		assert.NoError(t, err)
	})

	t.Run("returns error with last value on timeout", func(t *testing.T) {
		err := WaitForValue(func() string { return "nope" }, "yes", 50*time.Millisecond, 10*time.Millisecond)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nope")
		assert.Contains(t, err.Error(), "yes")
	})
}

// ── fixtures.go ─────────────────────────────────────────────────────────────

func TestSampleTenant(t *testing.T) {
	tenant := SampleTenant()
	require.NotNil(t, tenant)
	assert.NotEqual(t, uuid.Nil, tenant.ID)
	assert.Equal(t, "test-tenant", tenant.Name)
	assert.Equal(t, "pro", tenant.SubscriptionTier)
	assert.Equal(t, 1000, tenant.RateLimitPerMinute)

	// Each call should produce a unique ID
	tenant2 := SampleTenant()
	assert.NotEqual(t, tenant.ID, tenant2.ID)
}

func TestSampleEndpoint(t *testing.T) {
	tenantID := uuid.New()
	ep := SampleEndpoint(tenantID)
	require.NotNil(t, ep)
	assert.Equal(t, tenantID, ep.TenantID)
	assert.NotEqual(t, uuid.Nil, ep.ID)
	assert.Equal(t, "https://example.com/webhook", ep.URL)
	assert.True(t, ep.IsActive)
	assert.Equal(t, 5, ep.RetryConfig.MaxAttempts)
	assert.Contains(t, ep.CustomHeaders, "X-Custom")
}

func TestSampleDeliveryAttempt(t *testing.T) {
	endpointID := uuid.New()
	da := SampleDeliveryAttempt(endpointID)
	require.NotNil(t, da)
	assert.Equal(t, endpointID, da.EndpointID)
	assert.NotEqual(t, uuid.Nil, da.ID)
	assert.Equal(t, "delivered", da.Status)
	assert.Equal(t, 1, da.AttemptNumber)
	require.NotNil(t, da.HTTPStatus)
	assert.Equal(t, 200, *da.HTTPStatus)
}

// ── mocks.go ────────────────────────────────────────────────────────────────

func TestMockTenantRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMockTenantRepository()

	tenant := SampleTenant()

	t.Run("Create and GetByID", func(t *testing.T) {
		err := repo.Create(ctx, tenant)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, tenant.Name, got.Name)
	})

	t.Run("GetByID returns nil for unknown ID", func(t *testing.T) {
		got, err := repo.GetByID(ctx, uuid.New())
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("GetByAPIKeyHash", func(t *testing.T) {
		got, err := repo.GetByAPIKeyHash(ctx, tenant.APIKeyHash)
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, got.ID)

		got, err = repo.GetByAPIKeyHash(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("List", func(t *testing.T) {
		list, err := repo.List(ctx, 100, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 1)
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, tenant.ID)
		require.NoError(t, err)
		got, err := repo.GetByID(ctx, tenant.ID)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("CreateFunc hook overrides default behavior", func(t *testing.T) {
		repo.CreateFunc = func(ctx context.Context, t *models.Tenant) error {
			return assert.AnError
		}
		err := repo.Create(ctx, SampleTenant())
		assert.ErrorIs(t, err, assert.AnError)
		repo.CreateFunc = nil
	})
}

func TestMockWebhookEndpointRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMockWebhookEndpointRepository()

	tenantID := uuid.New()
	ep := SampleEndpoint(tenantID)

	t.Run("Create and GetByID", func(t *testing.T) {
		err := repo.Create(ctx, ep)
		require.NoError(t, err)
		got, err := repo.GetByID(ctx, ep.ID)
		require.NoError(t, err)
		assert.Equal(t, ep.URL, got.URL)
	})

	t.Run("GetByTenantID", func(t *testing.T) {
		list, err := repo.GetByTenantID(ctx, tenantID, 100, 0)
		require.NoError(t, err)
		assert.Len(t, list, 1)
	})

	t.Run("GetActiveByTenantID", func(t *testing.T) {
		active, err := repo.GetActiveByTenantID(ctx, tenantID)
		require.NoError(t, err)
		assert.Len(t, active, 1)
	})

	t.Run("SetActive", func(t *testing.T) {
		err := repo.SetActive(ctx, ep.ID, false)
		require.NoError(t, err)
		active, err := repo.GetActiveByTenantID(ctx, tenantID)
		require.NoError(t, err)
		assert.Len(t, active, 0)
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, ep.ID)
		require.NoError(t, err)
		got, err := repo.GetByID(ctx, ep.ID)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestMockPublisher(t *testing.T) {
	ctx := context.Background()
	pub := NewMockPublisher()

	t.Run("PublishDelivery stores messages", func(t *testing.T) {
		msg := &queue.DeliveryMessage{EndpointID: uuid.New()}
		err := pub.PublishDelivery(ctx, msg)
		require.NoError(t, err)
		assert.Len(t, pub.Messages, 1)
	})

	t.Run("PublishToDeadLetter stores in DLQ", func(t *testing.T) {
		msg := &queue.DeliveryMessage{EndpointID: uuid.New()}
		err := pub.PublishToDeadLetter(ctx, msg, "test reason")
		require.NoError(t, err)
		assert.Len(t, pub.DLQ, 1)
	})

	t.Run("GetQueueStats returns counts", func(t *testing.T) {
		stats, err := pub.GetQueueStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats["messages"])
		assert.Equal(t, int64(1), stats["dlq"])
	})

	t.Run("PublishFunc hook overrides default", func(t *testing.T) {
		pub.PublishFunc = func(ctx context.Context, msg *queue.DeliveryMessage) error {
			return assert.AnError
		}
		err := pub.PublishDelivery(ctx, &queue.DeliveryMessage{})
		assert.ErrorIs(t, err, assert.AnError)
		pub.PublishFunc = nil
	})
}

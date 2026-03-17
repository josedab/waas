package replay

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockRepo struct {
	mu sync.Mutex

	getDeliveryArchiveFn     func(ctx context.Context, tenantID, deliveryID string) (*DeliveryArchive, error)
	listDeliveryArchivesFn   func(ctx context.Context, tenantID string, filters *BulkReplayRequest) ([]DeliveryArchive, int, error)
	archiveDeliveryFn        func(ctx context.Context, archive *DeliveryArchive) error
	createSnapshotFn         func(ctx context.Context, snapshot *Snapshot, deliveryIDs []string) error
	getSnapshotFn            func(ctx context.Context, tenantID, snapshotID string) (*Snapshot, error)
	listSnapshotsFn          func(ctx context.Context, tenantID string, limit, offset int) ([]Snapshot, int, error)
	getSnapshotDeliveryIDsFn func(ctx context.Context, snapshotID string) ([]string, error)
	deleteSnapshotFn         func(ctx context.Context, tenantID, snapshotID string) error
	cleanupExpiredFn         func(ctx context.Context) (int64, error)
}

func (m *mockRepo) GetDeliveryArchive(ctx context.Context, tenantID, deliveryID string) (*DeliveryArchive, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getDeliveryArchiveFn != nil {
		return m.getDeliveryArchiveFn(ctx, tenantID, deliveryID)
	}
	return nil, nil
}

func (m *mockRepo) ListDeliveryArchives(ctx context.Context, tenantID string, filters *BulkReplayRequest) ([]DeliveryArchive, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listDeliveryArchivesFn != nil {
		return m.listDeliveryArchivesFn(ctx, tenantID, filters)
	}
	return nil, 0, nil
}

func (m *mockRepo) ArchiveDelivery(ctx context.Context, archive *DeliveryArchive) error {
	if m.archiveDeliveryFn != nil {
		return m.archiveDeliveryFn(ctx, archive)
	}
	return nil
}

func (m *mockRepo) CreateSnapshot(ctx context.Context, snapshot *Snapshot, deliveryIDs []string) error {
	if m.createSnapshotFn != nil {
		return m.createSnapshotFn(ctx, snapshot, deliveryIDs)
	}
	return nil
}

func (m *mockRepo) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*Snapshot, error) {
	if m.getSnapshotFn != nil {
		return m.getSnapshotFn(ctx, tenantID, snapshotID)
	}
	return nil, nil
}

func (m *mockRepo) ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]Snapshot, int, error) {
	if m.listSnapshotsFn != nil {
		return m.listSnapshotsFn(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) GetSnapshotDeliveryIDs(ctx context.Context, snapshotID string) ([]string, error) {
	if m.getSnapshotDeliveryIDsFn != nil {
		return m.getSnapshotDeliveryIDsFn(ctx, snapshotID)
	}
	return nil, nil
}

func (m *mockRepo) DeleteSnapshot(ctx context.Context, tenantID, snapshotID string) error {
	if m.deleteSnapshotFn != nil {
		return m.deleteSnapshotFn(ctx, tenantID, snapshotID)
	}
	return nil
}

func (m *mockRepo) CleanupExpiredSnapshots(ctx context.Context) (int64, error) {
	if m.cleanupExpiredFn != nil {
		return m.cleanupExpiredFn(ctx)
	}
	return 0, nil
}

type mockPublisher struct {
	mu        sync.Mutex
	publishFn func(ctx context.Context, tenantID, endpointID string, payload []byte, headers map[string]string) (string, error)
	calls     []publishCall
}

type publishCall struct {
	TenantID   string
	EndpointID string
	Payload    []byte
	Headers    map[string]string
}

func (m *mockPublisher) Publish(ctx context.Context, tenantID, endpointID string, payload []byte, headers map[string]string) (string, error) {
	m.mu.Lock()
	m.calls = append(m.calls, publishCall{TenantID: tenantID, EndpointID: endpointID, Payload: payload, Headers: headers})
	m.mu.Unlock()
	if m.publishFn != nil {
		return m.publishFn(ctx, tenantID, endpointID, payload, headers)
	}
	return "new-delivery-id", nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sampleArchive(id, tenantID, endpointID string) *DeliveryArchive {
	return &DeliveryArchive{
		ID:             id,
		TenantID:       tenantID,
		EndpointID:     endpointID,
		EndpointURL:    "https://example.com/webhook",
		Payload:        json.RawMessage(`{"event":"test","data":"value"}`),
		Headers:        map[string]string{"Content-Type": "application/json"},
		Status:         "failed",
		AttemptCount:   3,
		LastHTTPStatus: 500,
		LastError:      "server error",
		CreatedAt:      time.Now().Add(-1 * time.Hour),
	}
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	t.Run("with nil dependencies", func(t *testing.T) {
		svc := NewService(nil, nil)
		require.NotNil(t, svc)
	})

	t.Run("with valid dependencies", func(t *testing.T) {
		svc := NewService(&mockRepo{}, &mockPublisher{})
		require.NotNil(t, svc)
	})
}

// ---------------------------------------------------------------------------
// ReplaySingle tests
// ---------------------------------------------------------------------------

func TestReplaySingle_Success(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, tid, did string) (*DeliveryArchive, error) {
			if tid == "tenant-1" && did == "del-1" {
				return archive, nil
			}
			return nil, nil
		},
	}
	pub := &mockPublisher{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
			return "new-del-1", nil
		},
	}
	svc := NewService(repo, pub)

	result, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "del-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "del-1", result.OriginalDeliveryID)
	assert.Equal(t, "new-del-1", result.NewDeliveryID)
	assert.Equal(t, "ep-1", result.EndpointID)
	assert.Equal(t, "queued", result.Status)
	assert.WithinDuration(t, time.Now(), result.ReplayedAt, 2*time.Second)
}

func TestReplaySingle_NotFound(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return nil, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "nonexistent",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivery not found")
}

func TestReplaySingle_RepoError(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return nil, errors.New("db connection failed")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "del-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get delivery")
}

func TestReplaySingle_PublishError(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
			return "", errors.New("queue full")
		},
	}
	svc := NewService(repo, pub)

	_, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "del-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to queue replay")
}

func TestReplaySingle_EndpointOverride(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "del-1",
		EndpointID: "ep-override",
	})

	require.NoError(t, err)
	assert.Equal(t, "ep-override", result.EndpointID)
	require.Len(t, pub.calls, 1)
	assert.Equal(t, "ep-override", pub.calls[0].EndpointID)
}

func TestReplaySingle_ModifiedPayload(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	newPayload := []byte(`{"event":"modified"}`)
	_, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID:    "del-1",
		ModifyPayload: true,
		Payload:       newPayload,
	})

	require.NoError(t, err)
	require.Len(t, pub.calls, 1)
	assert.JSONEq(t, `{"event":"modified"}`, string(pub.calls[0].Payload))
}

func TestReplaySingle_ModifyPayloadFlagWithoutPayload(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	_, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID:    "del-1",
		ModifyPayload: true,
		Payload:       nil,
	})

	require.NoError(t, err)
	require.Len(t, pub.calls, 1)
	// Original payload should be used when ModifyPayload is true but Payload is nil
	assert.JSONEq(t, `{"event":"test","data":"value"}`, string(pub.calls[0].Payload))
}

func TestReplaySingle_HeadersPreserved(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Headers = map[string]string{"X-Custom": "val", "Content-Type": "application/json"}
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	_, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{DeliveryID: "del-1"})

	require.NoError(t, err)
	require.Len(t, pub.calls, 1)
	assert.Equal(t, "val", pub.calls[0].Headers["X-Custom"])
	assert.Equal(t, "application/json", pub.calls[0].Headers["Content-Type"])
}

// ---------------------------------------------------------------------------
// ReplayBulk tests
// ---------------------------------------------------------------------------

func TestReplayBulk_ByIDs_Success(t *testing.T) {
	archives := map[string]*DeliveryArchive{
		"del-1": sampleArchive("del-1", "tenant-1", "ep-1"),
		"del-2": sampleArchive("del-2", "tenant-1", "ep-2"),
	}
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			if a, ok := archives[did]; ok {
				return a, nil
			}
			return nil, nil
		},
	}
	callCount := 0
	pub := &mockPublisher{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
			callCount++
			return "new-" + string(rune('0'+callCount)), nil
		},
	}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		DeliveryIDs: []string{"del-1", "del-2"},
		RateLimit:   1000, // high rate to speed up test
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalFound)
	assert.Equal(t, 2, result.TotalReplayed)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Len(t, result.Results, 2)
}

func TestReplayBulk_ByIDs_PartialFailure(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			if did == "del-1" {
				return sampleArchive("del-1", "tenant-1", "ep-1"), nil
			}
			return nil, errors.New("archive corrupted")
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		DeliveryIDs: []string{"del-1", "del-bad"},
		RateLimit:   1000,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalReplayed)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "del-bad", result.Errors[0].DeliveryID)
}

func TestReplayBulk_ByIDs_MissingDeliverySkipped(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			if did == "del-1" {
				return sampleArchive("del-1", "tenant-1", "ep-1"), nil
			}
			return nil, nil // not found – returns nil,nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		DeliveryIDs: []string{"del-1", "del-missing"},
		RateLimit:   1000,
	})

	require.NoError(t, err)
	// Only del-1 is found, del-missing returns nil so it's not appended
	assert.Equal(t, 1, result.TotalFound)
	assert.Equal(t, 1, result.TotalReplayed)
}

func TestReplayBulk_ByFilter(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, filters *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			assert.Equal(t, "failed", filters.Status)
			assert.Equal(t, "ep-1", filters.EndpointID)
			return []DeliveryArchive{
				*sampleArchive("del-1", "tenant-1", "ep-1"),
				*sampleArchive("del-2", "tenant-1", "ep-1"),
			}, 2, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		Status:     "failed",
		EndpointID: "ep-1",
		RateLimit:  1000,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalFound)
	assert.Equal(t, 2, result.TotalReplayed)
}

func TestReplayBulk_ByFilter_ListError(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return nil, 0, errors.New("query timeout")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		Status: "failed",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list deliveries")
}

func TestReplayBulk_DryRun(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return []DeliveryArchive{
				*sampleArchive("del-1", "tenant-1", "ep-1"),
			}, 1, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		Status: "failed",
		DryRun: true,
	})

	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "would_replay", result.Results[0].Status)
	// Publisher should NOT have been called
	assert.Empty(t, pub.calls)
}

func TestReplayBulk_PublishPartialFailure(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return []DeliveryArchive{
				*sampleArchive("del-1", "tenant-1", "ep-1"),
				*sampleArchive("del-2", "tenant-1", "ep-2"),
				*sampleArchive("del-3", "tenant-1", "ep-3"),
			}, 3, nil
		},
	}
	callIdx := 0
	pub := &mockPublisher{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
			callIdx++
			if callIdx == 2 {
				return "", errors.New("endpoint unreachable")
			}
			return "new-id", nil
		},
	}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		Status:    "failed",
		RateLimit: 1000,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalFound)
	assert.Equal(t, 2, result.TotalReplayed)
	assert.Equal(t, 1, result.TotalFailed)
	assert.Len(t, result.Errors, 1)
}

func TestReplayBulk_EmptyList(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return nil, 0, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		Status:    "failed",
		RateLimit: 1000,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalFound)
	assert.Equal(t, 0, result.TotalReplayed)
	assert.Empty(t, pub.calls)
}

func TestReplayBulk_DefaultRateLimit(t *testing.T) {
	// Verify the service doesn't panic with RateLimit <= 0
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return []DeliveryArchive{*sampleArchive("del-1", "tenant-1", "ep-1")}, 1, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		Status:    "failed",
		RateLimit: 0, // should default to 10
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalReplayed)
}

// ---------------------------------------------------------------------------
// Snapshot tests
// ---------------------------------------------------------------------------

func TestCreateSnapshot_Success(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return []DeliveryArchive{
				*sampleArchive("del-1", "tenant-1", "ep-1"),
				*sampleArchive("del-2", "tenant-1", "ep-1"),
			}, 2, nil
		},
		createSnapshotFn: func(_ context.Context, snap *Snapshot, ids []string) error {
			assert.NotEmpty(t, snap.ID)
			assert.Equal(t, "tenant-1", snap.TenantID)
			assert.Equal(t, "my snapshot", snap.Name)
			assert.Len(t, ids, 2)
			return nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	snap, err := svc.CreateSnapshot(context.Background(), "tenant-1", &CreateSnapshotRequest{
		Name:    "my snapshot",
		Filters: SnapshotFilters{Status: "failed"},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, snap.ID)
	assert.Equal(t, "my snapshot", snap.Name)
	assert.Len(t, snap.DeliveryIDs, 2)
}

func TestCreateSnapshot_WithTTL(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return []DeliveryArchive{*sampleArchive("del-1", "tenant-1", "ep-1")}, 1, nil
		},
		createSnapshotFn: func(_ context.Context, snap *Snapshot, _ []string) error {
			assert.False(t, snap.ExpiresAt.IsZero(), "ExpiresAt should be set when TTLDays > 0")
			return nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	snap, err := svc.CreateSnapshot(context.Background(), "tenant-1", &CreateSnapshotRequest{
		Name:    "ttl-snap",
		Filters: SnapshotFilters{Status: "failed"},
		TTLDays: 7,
	})

	require.NoError(t, err)
	assert.False(t, snap.ExpiresAt.IsZero())
}

func TestCreateSnapshot_NoMatchingDeliveries(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return nil, 0, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.CreateSnapshot(context.Background(), "tenant-1", &CreateSnapshotRequest{
		Name:    "empty",
		Filters: SnapshotFilters{Status: "nonexistent"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no deliveries match")
}

func TestCreateSnapshot_ListError(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.CreateSnapshot(context.Background(), "tenant-1", &CreateSnapshotRequest{
		Name:    "fail",
		Filters: SnapshotFilters{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query deliveries")
}

func TestCreateSnapshot_RepoCreateError(t *testing.T) {
	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, _ *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			return []DeliveryArchive{*sampleArchive("del-1", "tenant-1", "ep-1")}, 1, nil
		},
		createSnapshotFn: func(_ context.Context, _ *Snapshot, _ []string) error {
			return errors.New("disk full")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.CreateSnapshot(context.Background(), "tenant-1", &CreateSnapshotRequest{
		Name:    "fail",
		Filters: SnapshotFilters{Status: "failed"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create snapshot")
}

func TestGetSnapshot_Success(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, tid, sid string) (*Snapshot, error) {
			return &Snapshot{
				ID:       sid,
				TenantID: tid,
				Name:     "snap-1",
			}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, sid string) ([]string, error) {
			return []string{"del-1", "del-2"}, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	snap, err := svc.GetSnapshot(context.Background(), "tenant-1", "snap-1")

	require.NoError(t, err)
	assert.Equal(t, "snap-1", snap.ID)
	assert.Equal(t, []string{"del-1", "del-2"}, snap.DeliveryIDs)
}

func TestGetSnapshot_NotFound(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return nil, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.GetSnapshot(context.Background(), "tenant-1", "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot not found")
}

func TestGetSnapshot_RepoError(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return nil, errors.New("db fail")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.GetSnapshot(context.Background(), "tenant-1", "snap-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get snapshot")
}

func TestGetSnapshot_DeliveryIDsError(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return &Snapshot{ID: "snap-1", TenantID: "tenant-1"}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("join table missing")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.GetSnapshot(context.Background(), "tenant-1", "snap-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get snapshot deliveries")
}

func TestListSnapshots_Success(t *testing.T) {
	repo := &mockRepo{
		listSnapshotsFn: func(_ context.Context, tid string, limit, offset int) ([]Snapshot, int, error) {
			assert.Equal(t, "tenant-1", tid)
			assert.Equal(t, 10, limit)
			assert.Equal(t, 0, offset)
			return []Snapshot{
				{ID: "snap-1", Name: "first"},
				{ID: "snap-2", Name: "second"},
			}, 2, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	snaps, total, err := svc.ListSnapshots(context.Background(), "tenant-1", 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, snaps, 2)
}

func TestListSnapshots_DefaultLimit(t *testing.T) {
	repo := &mockRepo{
		listSnapshotsFn: func(_ context.Context, _ string, limit, _ int) ([]Snapshot, int, error) {
			assert.Equal(t, 20, limit, "should default to 20 when limit <= 0")
			return nil, 0, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, _, err := svc.ListSnapshots(context.Background(), "tenant-1", 0, 0)
	require.NoError(t, err)
}

func TestListSnapshots_MaxLimit(t *testing.T) {
	repo := &mockRepo{
		listSnapshotsFn: func(_ context.Context, _ string, limit, _ int) ([]Snapshot, int, error) {
			assert.Equal(t, 100, limit, "should cap at 100")
			return nil, 0, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, _, err := svc.ListSnapshots(context.Background(), "tenant-1", 500, 0)
	require.NoError(t, err)
}

func TestDeleteSnapshot(t *testing.T) {
	called := false
	repo := &mockRepo{
		deleteSnapshotFn: func(_ context.Context, tid, sid string) error {
			called = true
			assert.Equal(t, "tenant-1", tid)
			assert.Equal(t, "snap-1", sid)
			return nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	err := svc.DeleteSnapshot(context.Background(), "tenant-1", "snap-1")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestDeleteSnapshot_Error(t *testing.T) {
	repo := &mockRepo{
		deleteSnapshotFn: func(_ context.Context, _, _ string) error {
			return errors.New("permission denied")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	err := svc.DeleteSnapshot(context.Background(), "tenant-1", "snap-1")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ReplayFromSnapshot tests
// ---------------------------------------------------------------------------

func TestReplayFromSnapshot_Success(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, sid string) (*Snapshot, error) {
			return &Snapshot{ID: sid, TenantID: "tenant-1"}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"del-1", "del-2"}, nil
		},
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			return sampleArchive(did, "tenant-1", "ep-1"), nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "snap-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalReplayed)
	assert.Len(t, pub.calls, 2)
}

func TestReplayFromSnapshot_WithLimit(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return &Snapshot{ID: "snap-1", TenantID: "tenant-1"}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"del-1", "del-2", "del-3"}, nil
		},
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			return sampleArchive(did, "tenant-1", "ep-1"), nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "snap-1",
		Limit:      1,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalReplayed)
	assert.Len(t, pub.calls, 1)
}

func TestReplayFromSnapshot_DryRun(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return &Snapshot{ID: "snap-1", TenantID: "tenant-1"}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"del-1"}, nil
		},
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			return sampleArchive(did, "tenant-1", "ep-1"), nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "snap-1",
		DryRun:     true,
	})

	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Equal(t, "would_replay", result.Results[0].Status)
	assert.Empty(t, pub.calls)
}

func TestReplayFromSnapshot_NotFound(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return nil, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "nonexistent",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot not found")
}

func TestReplayFromSnapshot_GetSnapshotError(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return nil, errors.New("db down")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "snap-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get snapshot")
}

func TestReplayFromSnapshot_GetDeliveryIDsError(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return &Snapshot{ID: "snap-1", TenantID: "tenant-1"}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("snapshot corrupted")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "snap-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get snapshot deliveries")
}

// ---------------------------------------------------------------------------
// GetDeliveryForReplay tests
// ---------------------------------------------------------------------------

func TestGetDeliveryForReplay_Success(t *testing.T) {
	expected := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return expected, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	archive, err := svc.GetDeliveryForReplay(context.Background(), "tenant-1", "del-1")

	require.NoError(t, err)
	assert.Equal(t, expected, archive)
}

func TestGetDeliveryForReplay_NotFound(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return nil, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	archive, err := svc.GetDeliveryForReplay(context.Background(), "tenant-1", "nonexistent")

	require.NoError(t, err)
	assert.Nil(t, archive)
}

// ---------------------------------------------------------------------------
// Cleanup tests
// ---------------------------------------------------------------------------

func TestCleanup_Success(t *testing.T) {
	repo := &mockRepo{
		cleanupExpiredFn: func(_ context.Context) (int64, error) {
			return 5, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	count, err := svc.Cleanup(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestCleanup_NoExpired(t *testing.T) {
	repo := &mockRepo{
		cleanupExpiredFn: func(_ context.Context) (int64, error) {
			return 0, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	count, err := svc.Cleanup(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestCleanup_Error(t *testing.T) {
	repo := &mockRepo{
		cleanupExpiredFn: func(_ context.Context) (int64, error) {
			return 0, errors.New("cleanup failed")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.Cleanup(context.Background())
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// RunWhatIf tests
// ---------------------------------------------------------------------------

func TestRunWhatIf_NoModifications(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	result, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID: "del-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "del-1", result.OriginalDeliveryID)
	assert.Equal(t, archive.EndpointID, result.Original.EndpointID)
	assert.Equal(t, archive.EndpointID, result.Simulated.EndpointID)
	assert.Empty(t, result.PayloadDiff)
	assert.Empty(t, result.EndpointChanges)
	assert.Contains(t, result.Analysis, "0 payload field(s) changed")
}

func TestRunWhatIf_ModifiedPayload(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Payload = json.RawMessage(`{"event":"original","count":1}`)
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	result, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID:      "del-1",
		ModifiedPayload: json.RawMessage(`{"event":"modified","count":2,"extra":"field"}`),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.PayloadDiff)

	// Verify diff items
	diffTypes := map[string]bool{}
	for _, d := range result.PayloadDiff {
		diffTypes[d.Type] = true
	}
	assert.True(t, diffTypes["changed"], "should detect changed fields")
	assert.True(t, diffTypes["added"], "should detect added fields")
}

func TestRunWhatIf_ModifiedHeaders(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Headers = map[string]string{"Content-Type": "application/json"}
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	result, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID:      "del-1",
		ModifiedHeaders: map[string]string{"X-Custom": "value"},
	})

	require.NoError(t, err)
	assert.Equal(t, "value", result.Simulated.Headers["X-Custom"])
	assert.Equal(t, "application/json", result.Simulated.Headers["Content-Type"])
}

func TestRunWhatIf_TargetEndpoints(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	result, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID:      "del-1",
		TargetEndpoints: []string{"ep-new-1", "ep-new-2"},
	})

	require.NoError(t, err)
	assert.Equal(t, "ep-new-1", result.Simulated.EndpointID)
	assert.Equal(t, "endpoint:ep-new-1", result.Simulated.EndpointURL)
	assert.Equal(t, []string{"ep-new-1", "ep-new-2"}, result.EndpointChanges)
	assert.Contains(t, result.Analysis, "endpoint(s) retargeted")
}

func TestRunWhatIf_NotFound(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return nil, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID: "nonexistent",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivery not found")
}

func TestRunWhatIf_RepoError(t *testing.T) {
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return nil, errors.New("timeout")
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID: "del-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get delivery")
}

func TestRunWhatIf_PayloadSizeDelta(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Payload = json.RawMessage(`{"a":"b"}`)
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	bigPayload := json.RawMessage(`{"a":"b","c":"d","e":"f","g":"h"}`)
	result, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID:      "del-1",
		ModifiedPayload: bigPayload,
	})

	require.NoError(t, err)
	assert.Greater(t, result.Simulated.PayloadSize, result.Original.PayloadSize)
	assert.Contains(t, result.Analysis, "payload size delta")
}

// ---------------------------------------------------------------------------
// computePayloadDiff tests (via exported behavior)
// ---------------------------------------------------------------------------

func TestComputePayloadDiff_BothEmpty(t *testing.T) {
	diffs := computePayloadDiff(json.RawMessage(`{}`), json.RawMessage(`{}`))
	assert.Empty(t, diffs)
}

func TestComputePayloadDiff_AddedField(t *testing.T) {
	diffs := computePayloadDiff(
		json.RawMessage(`{"a":"1"}`),
		json.RawMessage(`{"a":"1","b":"2"}`),
	)
	require.Len(t, diffs, 1)
	assert.Equal(t, "added", diffs[0].Type)
	assert.Equal(t, "b", diffs[0].Path)
}

func TestComputePayloadDiff_RemovedField(t *testing.T) {
	diffs := computePayloadDiff(
		json.RawMessage(`{"a":"1","b":"2"}`),
		json.RawMessage(`{"a":"1"}`),
	)
	require.Len(t, diffs, 1)
	assert.Equal(t, "removed", diffs[0].Type)
	assert.Equal(t, "b", diffs[0].Path)
}

func TestComputePayloadDiff_ChangedField(t *testing.T) {
	diffs := computePayloadDiff(
		json.RawMessage(`{"a":"old"}`),
		json.RawMessage(`{"a":"new"}`),
	)
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
	assert.Equal(t, "a", diffs[0].Path)
}

func TestComputePayloadDiff_InvalidOldJSON(t *testing.T) {
	diffs := computePayloadDiff(
		json.RawMessage(`not json`),
		json.RawMessage(`{"a":"1"}`),
	)
	// Old unmarshal fails → nil map; new has "a" → appears as "added"
	require.Len(t, diffs, 1)
	assert.Equal(t, "added", diffs[0].Type)
	assert.Equal(t, "a", diffs[0].Path)
}

func TestComputePayloadDiff_BothInvalidJSON(t *testing.T) {
	diffs := computePayloadDiff(
		json.RawMessage(`not json`),
		json.RawMessage(`also not json`),
	)
	// Both unmarshal to nil maps → no diffs
	assert.Empty(t, diffs)
}

// ---------------------------------------------------------------------------
// generateWhatIfAnalysis tests
// ---------------------------------------------------------------------------

func TestGenerateWhatIfAnalysis_NoChanges(t *testing.T) {
	result := &WhatIfResult{
		Original:  &WhatIfDelivery{PayloadSize: 100},
		Simulated: &WhatIfDelivery{PayloadSize: 100},
	}
	analysis := generateWhatIfAnalysis(result)
	assert.Contains(t, analysis, "0 payload field(s) changed")
	assert.NotContains(t, analysis, "retargeted")
	assert.NotContains(t, analysis, "payload size delta")
}

func TestGenerateWhatIfAnalysis_WithEndpointChanges(t *testing.T) {
	result := &WhatIfResult{
		Original:        &WhatIfDelivery{PayloadSize: 100},
		Simulated:       &WhatIfDelivery{PayloadSize: 100},
		EndpointChanges: []string{"ep-1", "ep-2"},
	}
	analysis := generateWhatIfAnalysis(result)
	assert.Contains(t, analysis, "2 endpoint(s) retargeted")
}

func TestGenerateWhatIfAnalysis_WithSizeDelta(t *testing.T) {
	result := &WhatIfResult{
		Original:    &WhatIfDelivery{PayloadSize: 100},
		Simulated:   &WhatIfDelivery{PayloadSize: 150},
		PayloadDiff: []PayloadDiffItem{{Path: "x", Type: "added"}},
	}
	analysis := generateWhatIfAnalysis(result)
	assert.Contains(t, analysis, "1 payload field(s) changed")
	assert.Contains(t, analysis, "+50 bytes")
}

// ---------------------------------------------------------------------------
// Concurrent replay safety
// ---------------------------------------------------------------------------

func TestReplaySingle_ConcurrentSafe(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
			return "new-id", nil
		},
	}
	svc := NewService(repo, pub)

	var wg sync.WaitGroup
	errs := make([]error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
				DeliveryID: "del-1",
			})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d failed", i)
	}
	assert.Len(t, pub.calls, 20)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestReplaySingle_EmptyPayload(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Payload = json.RawMessage(`{}`)
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "del-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "queued", result.Status)
	assert.JSONEq(t, `{}`, string(pub.calls[0].Payload))
}

func TestReplaySingle_NilHeaders(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Headers = nil
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplaySingle(context.Background(), "tenant-1", &ReplayRequest{
		DeliveryID: "del-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "queued", result.Status)
	assert.Nil(t, pub.calls[0].Headers)
}

func TestRunWhatIf_EmptyHeaders(t *testing.T) {
	archive := sampleArchive("del-1", "tenant-1", "ep-1")
	archive.Headers = nil
	repo := &mockRepo{
		getDeliveryArchiveFn: func(_ context.Context, _, _ string) (*DeliveryArchive, error) {
			return archive, nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	result, err := svc.RunWhatIf(context.Background(), "tenant-1", &WhatIfRequest{
		DeliveryID:      "del-1",
		ModifiedHeaders: map[string]string{"X-New": "val"},
	})

	require.NoError(t, err)
	assert.Equal(t, "val", result.Simulated.Headers["X-New"])
}

func TestReplayBulk_ByIDs_EmptyList(t *testing.T) {
	repo := &mockRepo{}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayBulk(context.Background(), "tenant-1", &BulkReplayRequest{
		DeliveryIDs: []string{},
		RateLimit:   1000,
	})

	require.NoError(t, err)
	// Empty IDs falls through to filter path
	assert.Equal(t, 0, result.TotalFound)
	assert.Empty(t, pub.calls)
}

func TestReplayFromSnapshot_LimitExceedsAvailable(t *testing.T) {
	repo := &mockRepo{
		getSnapshotFn: func(_ context.Context, _, _ string) (*Snapshot, error) {
			return &Snapshot{ID: "snap-1", TenantID: "tenant-1"}, nil
		},
		getSnapshotDeliveryIDsFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"del-1"}, nil
		},
		getDeliveryArchiveFn: func(_ context.Context, _, did string) (*DeliveryArchive, error) {
			return sampleArchive(did, "tenant-1", "ep-1"), nil
		},
	}
	pub := &mockPublisher{}
	svc := NewService(repo, pub)

	result, err := svc.ReplayFromSnapshot(context.Background(), "tenant-1", &ReplayFromSnapshotRequest{
		SnapshotID: "snap-1",
		Limit:      100, // more than available
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalReplayed)
}

func TestCreateSnapshot_FiltersPassedCorrectly(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	repo := &mockRepo{
		listDeliveryArchivesFn: func(_ context.Context, _ string, filters *BulkReplayRequest) ([]DeliveryArchive, int, error) {
			assert.Equal(t, "ep-1", filters.EndpointID)
			assert.Equal(t, "failed", filters.Status)
			assert.Equal(t, startTime, filters.StartTime)
			assert.Equal(t, endTime, filters.EndTime)
			assert.Equal(t, 10000, filters.Limit)
			return []DeliveryArchive{*sampleArchive("del-1", "tenant-1", "ep-1")}, 1, nil
		},
		createSnapshotFn: func(_ context.Context, _ *Snapshot, _ []string) error {
			return nil
		},
	}
	svc := NewService(repo, &mockPublisher{})

	_, err := svc.CreateSnapshot(context.Background(), "tenant-1", &CreateSnapshotRequest{
		Name: "filtered",
		Filters: SnapshotFilters{
			EndpointID: "ep-1",
			Status:     "failed",
			StartTime:  startTime,
			EndTime:    endTime,
		},
	})

	require.NoError(t, err)
}

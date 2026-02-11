package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock ReplayRepository ---
type MockReplayRepo struct {
	mock.Mock
}

func (m *MockReplayRepo) ArchiveEvent(ctx context.Context, event *models.EventArchive) error {
	return m.Called(ctx, event).Error(0)
}
func (m *MockReplayRepo) GetArchivedEvent(ctx context.Context, id uuid.UUID) (*models.EventArchive, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.EventArchive), args.Error(1)
}
func (m *MockReplayRepo) SearchEvents(ctx context.Context, tenantID uuid.UUID, req *models.EventSearchRequest) ([]*models.EventArchive, int, error) {
	args := m.Called(ctx, tenantID, req)
	return args.Get(0).([]*models.EventArchive), args.Int(1), args.Error(2)
}
func (m *MockReplayRepo) GetEventsByTimeRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit, offset int) ([]*models.EventArchive, error) {
	args := m.Called(ctx, tenantID, start, end, limit, offset)
	return args.Get(0).([]*models.EventArchive), args.Error(1)
}
func (m *MockReplayRepo) CountEventsByTimeRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter *models.ReplayFilterCriteria) (int, error) {
	args := m.Called(ctx, tenantID, start, end, filter)
	return args.Int(0), args.Error(1)
}
func (m *MockReplayRepo) DeleteOldEvents(ctx context.Context, tenantID uuid.UUID, before time.Time) (int64, error) {
	args := m.Called(ctx, tenantID, before)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockReplayRepo) CreateReplayJob(ctx context.Context, job *models.ReplayJob) error {
	return m.Called(ctx, job).Error(0)
}
func (m *MockReplayRepo) GetReplayJob(ctx context.Context, id uuid.UUID) (*models.ReplayJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ReplayJob), args.Error(1)
}
func (m *MockReplayRepo) GetReplayJobs(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]*models.ReplayJob, error) {
	args := m.Called(ctx, tenantID, status, limit, offset)
	return args.Get(0).([]*models.ReplayJob), args.Error(1)
}
func (m *MockReplayRepo) UpdateReplayJob(ctx context.Context, job *models.ReplayJob) error {
	return m.Called(ctx, job).Error(0)
}
func (m *MockReplayRepo) UpdateReplayJobProgress(ctx context.Context, id uuid.UUID, processed, successful, failed int) error {
	return m.Called(ctx, id, processed, successful, failed).Error(0)
}
func (m *MockReplayRepo) UpdateReplayJobStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error {
	return m.Called(ctx, id, status, errorMsg).Error(0)
}
func (m *MockReplayRepo) CreateReplayJobEvents(ctx context.Context, events []*models.ReplayJobEvent) error {
	return m.Called(ctx, events).Error(0)
}
func (m *MockReplayRepo) GetReplayJobEvents(ctx context.Context, jobID uuid.UUID, status string, limit, offset int) ([]*models.ReplayJobEvent, error) {
	args := m.Called(ctx, jobID, status, limit, offset)
	return args.Get(0).([]*models.ReplayJobEvent), args.Error(1)
}
func (m *MockReplayRepo) UpdateReplayJobEvent(ctx context.Context, event *models.ReplayJobEvent) error {
	return m.Called(ctx, event).Error(0)
}
func (m *MockReplayRepo) GetPendingReplayEvents(ctx context.Context, jobID uuid.UUID, batchSize int) ([]*models.ReplayJobEvent, error) {
	args := m.Called(ctx, jobID, batchSize)
	return args.Get(0).([]*models.ReplayJobEvent), args.Error(1)
}
func (m *MockReplayRepo) CreateSnapshot(ctx context.Context, snapshot *models.ReplaySnapshot) error {
	return m.Called(ctx, snapshot).Error(0)
}
func (m *MockReplayRepo) GetSnapshot(ctx context.Context, id uuid.UUID) (*models.ReplaySnapshot, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ReplaySnapshot), args.Error(1)
}
func (m *MockReplayRepo) GetSnapshots(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplaySnapshot, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.ReplaySnapshot), args.Error(1)
}
func (m *MockReplayRepo) DeleteSnapshot(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockReplayRepo) CreateComparison(ctx context.Context, comparison *models.ReplayComparison) error {
	return m.Called(ctx, comparison).Error(0)
}
func (m *MockReplayRepo) GetComparison(ctx context.Context, id uuid.UUID) (*models.ReplayComparison, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ReplayComparison), args.Error(1)
}
func (m *MockReplayRepo) UpdateComparison(ctx context.Context, comparison *models.ReplayComparison) error {
	return m.Called(ctx, comparison).Error(0)
}

// --- Replay Service Tests ---

func TestReplayService_CreateReplayJob_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	tenantID := uuid.New()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	req := &models.CreateReplayJobRequest{
		Name:           "Test Replay",
		TimeRangeStart: start,
		TimeRangeEnd:   end,
	}

	repo.On("CountEventsByTimeRange", mock.Anything, tenantID, start, end, mock.Anything).Return(100, nil)
	repo.On("CreateReplayJob", mock.Anything, mock.AnythingOfType("*models.ReplayJob")).Return(nil)

	job, err := svc.CreateReplayJob(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, job.TenantID)
	assert.Equal(t, "Test Replay", job.Name)
	assert.Equal(t, 100, job.TotalEvents)
}

func TestReplayService_CreateReplayJob_InvalidTimeRange(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	_, err := svc.CreateReplayJob(context.Background(), uuid.New(), &models.CreateReplayJobRequest{
		Name:           "Bad Range",
		TimeRangeStart: time.Now(),
		TimeRangeEnd:   time.Now().Add(-1 * time.Hour),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time_range_end must be after time_range_start")
}

func TestReplayService_CreateReplayJob_NoEvents(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	repo.On("CountEventsByTimeRange", mock.Anything, mock.Anything, start, end, mock.Anything).Return(0, nil)

	_, err := svc.CreateReplayJob(context.Background(), uuid.New(), &models.CreateReplayJobRequest{
		TimeRangeStart: start,
		TimeRangeEnd:   end,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no events found")
}

func TestReplayService_CreateReplayJob_CountError(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	repo.On("CountEventsByTimeRange", mock.Anything, mock.Anything, start, end, mock.Anything).Return(0, fmt.Errorf("db error"))

	_, err := svc.CreateReplayJob(context.Background(), uuid.New(), &models.CreateReplayJobRequest{
		TimeRangeStart: start,
		TimeRangeEnd:   end,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count events")
}

func TestReplayService_StartReplayJob_TenantIsolation(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	jobID := uuid.New()
	ownerTenantID := uuid.New()
	otherTenantID := uuid.New()

	repo.On("GetReplayJob", mock.Anything, jobID).Return(&models.ReplayJob{
		ID:       jobID,
		TenantID: ownerTenantID,
		Status:   models.ReplayStatusPending,
	}, nil)

	err := svc.StartReplayJob(context.Background(), otherTenantID, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestReplayService_StartReplayJob_WrongStatus(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	tenantID := uuid.New()
	jobID := uuid.New()
	repo.On("GetReplayJob", mock.Anything, jobID).Return(&models.ReplayJob{
		ID:       jobID,
		TenantID: tenantID,
		Status:   models.ReplayStatusCompleted,
	}, nil)

	err := svc.StartReplayJob(context.Background(), tenantID, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be started in status")
}

func TestReplayService_PauseReplayJob_NotRunning(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	tenantID := uuid.New()
	jobID := uuid.New()
	repo.On("GetReplayJob", mock.Anything, jobID).Return(&models.ReplayJob{
		ID:       jobID,
		TenantID: tenantID,
		Status:   models.ReplayStatusPending,
	}, nil)

	err := svc.PauseReplayJob(context.Background(), tenantID, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only pause running jobs")
}

func TestReplayService_CancelReplayJob_TenantIsolation(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	jobID := uuid.New()
	ownerTenantID := uuid.New()

	repo.On("GetReplayJob", mock.Anything, jobID).Return(&models.ReplayJob{
		TenantID: ownerTenantID,
	}, nil)

	err := svc.CancelReplayJob(context.Background(), uuid.New(), jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestReplayService_GetReplayJobProgress(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	tenantID := uuid.New()
	jobID := uuid.New()
	started := time.Now().Add(-10 * time.Minute)

	repo.On("GetReplayJob", mock.Anything, jobID).Return(&models.ReplayJob{
		ID:               jobID,
		TenantID:         tenantID,
		Status:           models.ReplayStatusRunning,
		TotalEvents:      1000,
		ProcessedEvents:  500,
		SuccessfulEvents: 480,
		FailedEvents:     20,
		StartedAt:        &started,
	}, nil)

	progress, err := svc.GetReplayJobProgress(context.Background(), tenantID, jobID)
	require.NoError(t, err)
	assert.Equal(t, jobID, progress.JobID)
	assert.Equal(t, 50.0, progress.ProgressPercent)
	assert.Equal(t, 1000, progress.TotalEvents)
	assert.Equal(t, 500, progress.ProcessedEvents)
	assert.NotEmpty(t, progress.EstimatedTimeRemaining)
}

func TestReplayService_CreateSnapshot(t *testing.T) {
	t.Parallel()
	repo := &MockReplayRepo{}
	logger := utils.NewLogger("test")
	svc := NewReplayService(repo, nil, logger)

	tenantID := uuid.New()
	snapshotTime := time.Now()

	repo.On("CountEventsByTimeRange", mock.Anything, tenantID, mock.Anything, snapshotTime, mock.Anything).Return(500, nil)
	repo.On("CreateSnapshot", mock.Anything, mock.AnythingOfType("*models.ReplaySnapshot")).Return(nil)

	snapshot, err := svc.CreateSnapshot(context.Background(), tenantID, &models.CreateSnapshotRequest{
		Name:          "Test Snapshot",
		SnapshotTime:  snapshotTime,
		ExpiresInDays: 30,
	})
	require.NoError(t, err)
	assert.Equal(t, "Test Snapshot", snapshot.Name)
	assert.Equal(t, 500, snapshot.EventCount)
	assert.NotNil(t, snapshot.ExpiresAt)
}

func TestReplayService_MatchesFilter_EventTypes(t *testing.T) {
	t.Parallel()
	svc := NewReplayService(nil, nil, utils.NewLogger("test"))

	event := &models.EventArchive{EventType: "order.created"}
	filter := &models.ReplayFilterCriteria{
		EventTypes: []string{"order.created", "order.updated"},
	}
	assert.True(t, svc.matchesFilter(event, filter))

	filter.EventTypes = []string{"payment.processed"}
	assert.False(t, svc.matchesFilter(event, filter))
}

func TestReplayService_MatchesFilter_NilFilter(t *testing.T) {
	t.Parallel()
	svc := NewReplayService(nil, nil, utils.NewLogger("test"))

	event := &models.EventArchive{EventType: "test"}
	assert.True(t, svc.matchesFilter(event, nil))
}

func TestReplayService_MatchesFilter_ExcludeHashes(t *testing.T) {
	t.Parallel()
	svc := NewReplayService(nil, nil, utils.NewLogger("test"))

	event := &models.EventArchive{PayloadHash: "abc123"}
	filter := &models.ReplayFilterCriteria{
		ExcludeHashes: []string{"abc123"},
	}
	assert.False(t, svc.matchesFilter(event, filter))
}

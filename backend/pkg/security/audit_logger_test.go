package security

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockAuditRepository implements repository.AuditRepository for testing
type mockAuditRepository struct {
	mock.Mock
}

func (m *mockAuditRepository) LogEvent(ctx context.Context, event *repository.AuditEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockAuditRepository) GetAuditLogs(ctx context.Context, filter repository.AuditFilter) ([]*repository.AuditEvent, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*repository.AuditEvent), args.Error(1)
}

func (m *mockAuditRepository) GetAuditLogsByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.AuditFilter) ([]*repository.AuditEvent, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*repository.AuditEvent), args.Error(1)
}

func TestAuditLogger_LogEvent_SetsDefaults(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	repo.On("LogEvent", ctx, mock.MatchedBy(func(e *repository.AuditEvent) bool {
		return e.ID != uuid.Nil && !e.Timestamp.IsZero()
	})).Return(nil)

	event := &repository.AuditEvent{
		Action:   "test.action",
		Resource: "test",
	}
	err := logger.LogEvent(ctx, event)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.False(t, event.Timestamp.IsZero())
	repo.AssertExpectations(t)
}

func TestAuditLogger_LogEvent_PreservesExistingID(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	existingID := uuid.New()
	existingTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	repo.On("LogEvent", ctx, mock.Anything).Return(nil)

	event := &repository.AuditEvent{
		ID:        existingID,
		Timestamp: existingTime,
		Action:    "test.action",
		Resource:  "test",
	}
	err := logger.LogEvent(ctx, event)
	require.NoError(t, err)
	assert.Equal(t, existingID, event.ID)
	assert.Equal(t, existingTime, event.Timestamp)
}

func TestAuditLogger_LogEvent_RepoError(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	repo.On("LogEvent", ctx, mock.Anything).Return(errors.New("db error"))

	err := logger.LogEvent(ctx, &repository.AuditEvent{Action: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestAuditLogger_ConcurrentWrites(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	repo.On("LogEvent", ctx, mock.Anything).Return(nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := &repository.AuditEvent{
				Action:   ActionAuthLogin,
				Resource: "authentication",
			}
			_ = logger.LogEvent(ctx, event)
		}(i)
	}
	wg.Wait()

	repo.AssertNumberOfCalls(t, "LogEvent", 50)
}

func TestAuditLogger_LogTenantAction(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	repo.On("LogEvent", ctx, mock.MatchedBy(func(e *repository.AuditEvent) bool {
		return e.Action == ActionTenantCreate && e.Resource == "tenant" && e.Success
	})).Return(nil)

	tenantID := uuid.New()
	err := logger.LogTenantAction(ctx, tenantID, nil, ActionTenantCreate, "res-1",
		map[string]interface{}{"name": "test"}, "127.0.0.1", "test-agent", true, nil)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAuditLogger_LogAuthAction_Failure(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	repo.On("LogEvent", ctx, mock.MatchedBy(func(e *repository.AuditEvent) bool {
		return e.Action == ActionAuthLoginFailed && !e.Success && e.ErrorMsg != nil
	})).Return(nil)

	errMsg := "invalid credentials"
	err := logger.LogAuthAction(ctx, nil, nil, ActionAuthLoginFailed,
		nil, "10.0.0.1", "browser", false, &errMsg)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAuditLogger_GetAuditLogs(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	expected := []*repository.AuditEvent{{Action: "test"}}
	repo.On("GetAuditLogs", ctx, mock.Anything).Return(expected, nil)

	logs, err := logger.GetAuditLogs(ctx, repository.AuditFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestAuditLogger_GetTenantAuditLogs(t *testing.T) {
	t.Parallel()
	repo := &mockAuditRepository{}
	logger := NewAuditLogger(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	expected := []*repository.AuditEvent{{Action: "test"}}
	repo.On("GetAuditLogsByTenant", ctx, tenantID, mock.Anything).Return(expected, nil)

	logs, err := logger.GetTenantAuditLogs(ctx, tenantID, repository.AuditFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

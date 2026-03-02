package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock BidirectionalSyncRepository ---
type MockBidirectionalSyncRepo struct {
	mock.Mock
}

func (m *MockBidirectionalSyncRepo) CreateConfig(ctx context.Context, config *models.WebhookSyncConfig) error {
	return m.Called(ctx, config).Error(0)
}
func (m *MockBidirectionalSyncRepo) GetConfig(ctx context.Context, id uuid.UUID) (*models.WebhookSyncConfig, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.WebhookSyncConfig), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetConfigsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookSyncConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.WebhookSyncConfig), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) UpdateConfig(ctx context.Context, config *models.WebhookSyncConfig) error {
	return m.Called(ctx, config).Error(0)
}
func (m *MockBidirectionalSyncRepo) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockBidirectionalSyncRepo) CreateTransaction(ctx context.Context, tx *models.SyncTransaction) error {
	return m.Called(ctx, tx).Error(0)
}
func (m *MockBidirectionalSyncRepo) GetTransaction(ctx context.Context, id uuid.UUID) (*models.SyncTransaction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.SyncTransaction), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetTransactionByCorrelation(ctx context.Context, correlationID string) (*models.SyncTransaction, error) {
	args := m.Called(ctx, correlationID)
	return args.Get(0).(*models.SyncTransaction), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetTransactionsByConfig(ctx context.Context, configID uuid.UUID, limit int) ([]*models.SyncTransaction, error) {
	args := m.Called(ctx, configID, limit)
	return args.Get(0).([]*models.SyncTransaction), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetPendingTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncTransaction, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.SyncTransaction), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetTimedOutTransactions(ctx context.Context) ([]*models.SyncTransaction, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.SyncTransaction), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) UpdateTransactionState(ctx context.Context, id uuid.UUID, state string, errorMessage string) error {
	return m.Called(ctx, id, state, errorMessage).Error(0)
}
func (m *MockBidirectionalSyncRepo) CompleteTransaction(ctx context.Context, id uuid.UUID, responsePayload map[string]interface{}) error {
	return m.Called(ctx, id, responsePayload).Error(0)
}
func (m *MockBidirectionalSyncRepo) IncrementTransactionRetry(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockBidirectionalSyncRepo) CountTransactionsToday(ctx context.Context, tenantID uuid.UUID, state string) (int, error) {
	args := m.Called(ctx, tenantID, state)
	return args.Int(0), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) CreateStateRecord(ctx context.Context, record *models.SyncStateRecord) error {
	return m.Called(ctx, record).Error(0)
}
func (m *MockBidirectionalSyncRepo) GetStateRecord(ctx context.Context, id uuid.UUID) (*models.SyncStateRecord, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.SyncStateRecord), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetStateRecordByResource(ctx context.Context, configID uuid.UUID, resourceType, resourceID string) (*models.SyncStateRecord, error) {
	args := m.Called(ctx, configID, resourceType, resourceID)
	return args.Get(0).(*models.SyncStateRecord), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetStateRecordsByConfig(ctx context.Context, configID uuid.UUID) ([]*models.SyncStateRecord, error) {
	args := m.Called(ctx, configID)
	return args.Get(0).([]*models.SyncStateRecord), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetConflictedRecords(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncStateRecord, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.SyncStateRecord), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) UpdateLocalState(ctx context.Context, id uuid.UUID, state map[string]interface{}) error {
	return m.Called(ctx, id, state).Error(0)
}
func (m *MockBidirectionalSyncRepo) UpdateRemoteState(ctx context.Context, id uuid.UUID, state map[string]interface{}) error {
	return m.Called(ctx, id, state).Error(0)
}
func (m *MockBidirectionalSyncRepo) SetConflict(ctx context.Context, id uuid.UUID, conflictData map[string]interface{}) error {
	return m.Called(ctx, id, conflictData).Error(0)
}
func (m *MockBidirectionalSyncRepo) ResolveConflict(ctx context.Context, id uuid.UUID, resolvedState map[string]interface{}) error {
	return m.Called(ctx, id, resolvedState).Error(0)
}
func (m *MockBidirectionalSyncRepo) CountActiveConflicts(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) CreateAcknowledgment(ctx context.Context, ack *models.SyncAcknowledgment) error {
	return m.Called(ctx, ack).Error(0)
}
func (m *MockBidirectionalSyncRepo) GetAcknowledgment(ctx context.Context, id uuid.UUID) (*models.SyncAcknowledgment, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.SyncAcknowledgment), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetAcknowledgmentByCorrelation(ctx context.Context, correlationID string) (*models.SyncAcknowledgment, error) {
	args := m.Called(ctx, correlationID)
	return args.Get(0).(*models.SyncAcknowledgment), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) GetPendingAcknowledgments(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncAcknowledgment, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.SyncAcknowledgment), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) MarkAcknowledged(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockBidirectionalSyncRepo) CountPendingAcks(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockBidirectionalSyncRepo) CreateConflictHistory(ctx context.Context, history *models.SyncConflictHistory) error {
	return m.Called(ctx, history).Error(0)
}
func (m *MockBidirectionalSyncRepo) GetConflictHistoryByRecord(ctx context.Context, stateRecordID uuid.UUID) ([]*models.SyncConflictHistory, error) {
	args := m.Called(ctx, stateRecordID)
	return args.Get(0).([]*models.SyncConflictHistory), args.Error(1)
}

// --- Bidirectional Sync Service Tests ---

func TestBidirectionalSyncService_CreateConfig_ValidRequestResponse(t *testing.T) {
	t.Parallel()
	repo := &MockBidirectionalSyncRepo{}
	logger := utils.NewLogger("test")
	svc := NewBidirectionalSyncService(repo, logger)

	tenantID := uuid.New()
	repo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*models.WebhookSyncConfig")).Return(nil)

	config, err := svc.CreateConfig(context.Background(), tenantID, &models.CreateSyncConfigRequest{
		Name:     "Order Sync",
		SyncMode: models.SyncModeRequestResponse,
	})
	require.NoError(t, err)
	assert.Equal(t, tenantID, config.TenantID)
	assert.Equal(t, models.SyncModeRequestResponse, config.SyncMode)
	assert.True(t, config.Enabled)
}

func TestBidirectionalSyncService_CreateConfig_InvalidSyncMode(t *testing.T) {
	t.Parallel()
	repo := &MockBidirectionalSyncRepo{}
	logger := utils.NewLogger("test")
	svc := NewBidirectionalSyncService(repo, logger)

	_, err := svc.CreateConfig(context.Background(), uuid.New(), &models.CreateSyncConfigRequest{
		Name:     "Bad Config",
		SyncMode: "invalid_mode",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sync mode")
}

func TestBidirectionalSyncService_CreateConfig_DefaultTimeout(t *testing.T) {
	t.Parallel()
	repo := &MockBidirectionalSyncRepo{}
	logger := utils.NewLogger("test")
	svc := NewBidirectionalSyncService(repo, logger)

	tenantID := uuid.New()
	repo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*models.WebhookSyncConfig")).Return(nil)

	config, err := svc.CreateConfig(context.Background(), tenantID, &models.CreateSyncConfigRequest{
		Name:     "Default Timeout Config",
		SyncMode: models.SyncModeEventAcknowledgment,
	})
	require.NoError(t, err)
	assert.Equal(t, 30, config.TimeoutSeconds)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestBidirectionalSyncService_GetConfigs(t *testing.T) {
	t.Parallel()
	repo := &MockBidirectionalSyncRepo{}
	logger := utils.NewLogger("test")
	svc := NewBidirectionalSyncService(repo, logger)

	tenantID := uuid.New()
	configs := []*models.WebhookSyncConfig{
		{TenantID: tenantID, Name: "Config 1", SyncMode: models.SyncModeRequestResponse},
		{TenantID: tenantID, Name: "Config 2", SyncMode: models.SyncModeStateSync},
	}
	repo.On("GetConfigsByTenant", mock.Anything, tenantID).Return(configs, nil)

	result, err := svc.GetConfigs(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestBidirectionalSyncService_CreateConfig_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockBidirectionalSyncRepo{}
	logger := utils.NewLogger("test")
	svc := NewBidirectionalSyncService(repo, logger)

	repo.On("CreateConfig", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))

	_, err := svc.CreateConfig(context.Background(), uuid.New(), &models.CreateSyncConfigRequest{
		Name:     "Fail Config",
		SyncMode: models.SyncModeRequestResponse,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config")
}

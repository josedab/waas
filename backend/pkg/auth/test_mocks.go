package auth

import (
	"context"
	"github.com/josedab/waas/pkg/models"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockQuotaRepository is a mock implementation of QuotaRepository
type MockQuotaRepository struct {
	mock.Mock
}

func (m *MockQuotaRepository) GetOrCreateQuotaUsage(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error) {
	args := m.Called(ctx, tenantID, month)
	return args.Get(0).(*models.QuotaUsage), args.Error(1)
}

func (m *MockQuotaRepository) UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	args := m.Called(ctx, usage)
	return args.Error(0)
}

func (m *MockQuotaRepository) IncrementUsage(ctx context.Context, tenantID uuid.UUID, success bool) error {
	args := m.Called(ctx, tenantID, success)
	return args.Error(0)
}

func (m *MockQuotaRepository) GetQuotaUsageByTenant(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error) {
	args := m.Called(ctx, tenantID, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QuotaUsage), args.Error(1)
}

func (m *MockQuotaRepository) CreateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockQuotaRepository) GetBillingRecord(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) (*models.BillingRecord, error) {
	args := m.Called(ctx, tenantID, billingPeriod)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BillingRecord), args.Error(1)
}

func (m *MockQuotaRepository) UpdateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockQuotaRepository) GetBillingHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BillingRecord, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]*models.BillingRecord), args.Error(1)
}

func (m *MockQuotaRepository) CreateQuotaNotification(ctx context.Context, notification *models.QuotaNotification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockQuotaRepository) GetPendingNotifications(ctx context.Context, tenantID uuid.UUID) ([]*models.QuotaNotification, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.QuotaNotification), args.Error(1)
}

func (m *MockQuotaRepository) MarkNotificationSent(ctx context.Context, notificationID uuid.UUID) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockQuotaRepository) GetNotificationHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.QuotaNotification, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]*models.QuotaNotification), args.Error(1)
}

// MockTenantRepository is a mock implementation of TenantRepository
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Tenant, error) {
	args := m.Called(ctx, apiKeyHash)
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantRepository) List(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.Tenant), args.Error(1)
}

// minInt returns the smaller of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

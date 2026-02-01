package repository

import (
	"context"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QuotaRepositoryTestSuite struct {
	suite.Suite
	repo QuotaRepository
	ctx  context.Context
}

func (suite *QuotaRepositoryTestSuite) SetupTest() {
	suite.ctx = context.Background()
	// In a real test, you'd set up a test database connection
	// For now, we'll use a mock or in-memory database
	// suite.repo = NewQuotaRepository(testDB)
}

func TestQuotaUsage_GetUsagePercentage(t *testing.T) {
	usage := &models.QuotaUsage{
		RequestCount: 750,
	}

	percentage := usage.GetUsagePercentage(1000)
	assert.Equal(t, 75.0, percentage)

	// Test edge case with zero quota
	percentage = usage.GetUsagePercentage(0)
	assert.Equal(t, 0.0, percentage)
}

func TestQuotaUsage_IsOverQuota(t *testing.T) {
	usage := &models.QuotaUsage{
		RequestCount: 1200,
	}

	assert.True(t, usage.IsOverQuota(1000))
	assert.False(t, usage.IsOverQuota(1500))
}

func TestQuotaUsage_ShouldNotify(t *testing.T) {
	usage := &models.QuotaUsage{
		RequestCount: 850,
	}

	quota := 1000

	// Should notify at 80% threshold
	assert.True(t, usage.ShouldNotify(quota, 80))
	assert.True(t, usage.ShouldNotify(quota, 85))

	// Should not notify at 90% threshold (usage is 85%)
	assert.False(t, usage.ShouldNotify(quota, 90))
}

func TestGetSubscriptionTiers(t *testing.T) {
	tiers := models.GetSubscriptionTiers()

	assert.NotEmpty(t, tiers)
	assert.Contains(t, tiers, "free")
	assert.Contains(t, tiers, "starter")
	assert.Contains(t, tiers, "professional")
	assert.Contains(t, tiers, "enterprise")

	// Test free tier configuration
	freeTier := tiers["free"]
	assert.Equal(t, "free", freeTier.Name)
	assert.Equal(t, 1000, freeTier.MonthlyQuota)
	assert.Equal(t, 10, freeTier.RateLimitPerMinute)
	assert.Equal(t, 5, freeTier.MaxEndpoints)
	assert.Equal(t, 100, freeTier.BurstAllowance)

	// Test enterprise tier configuration
	enterpriseTier := tiers["enterprise"]
	assert.Equal(t, "enterprise", enterpriseTier.Name)
	assert.Equal(t, 1000000, enterpriseTier.MonthlyQuota)
	assert.Equal(t, 2000, enterpriseTier.RateLimitPerMinute)
	assert.Equal(t, -1, enterpriseTier.MaxEndpoints) // Unlimited
	assert.Equal(t, 100000, enterpriseTier.BurstAllowance)
}

func TestGetTierConfig(t *testing.T) {
	// Test valid tier
	tier, exists := models.GetTierConfig("starter")
	assert.True(t, exists)
	assert.Equal(t, "starter", tier.Name)
	assert.Equal(t, 10000, tier.MonthlyQuota)

	// Test invalid tier
	_, exists = models.GetTierConfig("invalid")
	assert.False(t, exists)
}

func TestBillingRecord_Validation(t *testing.T) {
	record := &models.BillingRecord{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		BillingPeriod:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseRequests:    5000,
		OverageRequests: 1000,
		BaseAmount:      0,
		OverageAmount:   1000,
		TotalAmount:     1000,
		Status:          "pending",
	}

	assert.Equal(t, 5000, record.BaseRequests)
	assert.Equal(t, 1000, record.OverageRequests)
	assert.Equal(t, 1000, record.TotalAmount)
	assert.Equal(t, "pending", record.Status)
}

func TestQuotaNotification_Types(t *testing.T) {
	notification := &models.QuotaNotification{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Type:       "warning",
		Threshold:  80,
		UsageCount: 800,
		QuotaLimit: 1000,
		Sent:       false,
	}

	assert.Equal(t, "warning", notification.Type)
	assert.Equal(t, 80, notification.Threshold)
	assert.False(t, notification.Sent)
	assert.Nil(t, notification.SentAt)
}

// Mock repository implementation for testing
type mockQuotaRepository struct {
	quotaUsage    map[string]*models.QuotaUsage
	billingRecords map[string]*models.BillingRecord
	notifications  map[string][]*models.QuotaNotification
}

func newMockQuotaRepository() *mockQuotaRepository {
	return &mockQuotaRepository{
		quotaUsage:     make(map[string]*models.QuotaUsage),
		billingRecords: make(map[string]*models.BillingRecord),
		notifications:  make(map[string][]*models.QuotaNotification),
	}
}

func (m *mockQuotaRepository) GetOrCreateQuotaUsage(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error) {
	key := tenantID.String() + month.Format("2006-01")
	if usage, exists := m.quotaUsage[key]; exists {
		return usage, nil
	}

	usage := &models.QuotaUsage{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Month:        month,
		RequestCount: 0,
		SuccessCount: 0,
		FailureCount: 0,
		OverageCount: 0,
		CreatedAt:    time.Now(),
	}
	m.quotaUsage[key] = usage
	return usage, nil
}

func (m *mockQuotaRepository) UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	key := usage.TenantID.String() + usage.Month.Format("2006-01")
	m.quotaUsage[key] = usage
	return nil
}

func (m *mockQuotaRepository) IncrementUsage(ctx context.Context, tenantID uuid.UUID, success bool) error {
	now := time.Now()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	usage, _ := m.GetOrCreateQuotaUsage(ctx, tenantID, month)
	usage.RequestCount++
	if success {
		usage.SuccessCount++
	} else {
		usage.FailureCount++
	}
	
	return m.UpdateQuotaUsage(ctx, usage)
}

func (m *mockQuotaRepository) GetQuotaUsageByTenant(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error) {
	key := tenantID.String() + month.Format("2006-01")
	if usage, exists := m.quotaUsage[key]; exists {
		return usage, nil
	}
	return nil, assert.AnError
}

func (m *mockQuotaRepository) CreateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	key := record.TenantID.String() + record.BillingPeriod.Format("2006-01")
	m.billingRecords[key] = record
	return nil
}

func (m *mockQuotaRepository) GetBillingRecord(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) (*models.BillingRecord, error) {
	key := tenantID.String() + billingPeriod.Format("2006-01")
	if record, exists := m.billingRecords[key]; exists {
		return record, nil
	}
	return nil, assert.AnError
}

func (m *mockQuotaRepository) UpdateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	key := record.TenantID.String() + record.BillingPeriod.Format("2006-01")
	m.billingRecords[key] = record
	return nil
}

func (m *mockQuotaRepository) GetBillingHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BillingRecord, error) {
	var records []*models.BillingRecord
	for _, record := range m.billingRecords {
		if record.TenantID == tenantID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (m *mockQuotaRepository) CreateQuotaNotification(ctx context.Context, notification *models.QuotaNotification) error {
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	key := notification.TenantID.String()
	m.notifications[key] = append(m.notifications[key], notification)
	return nil
}

func (m *mockQuotaRepository) GetPendingNotifications(ctx context.Context, tenantID uuid.UUID) ([]*models.QuotaNotification, error) {
	key := tenantID.String()
	var pending []*models.QuotaNotification
	for _, notification := range m.notifications[key] {
		if !notification.Sent {
			pending = append(pending, notification)
		}
	}
	return pending, nil
}

func (m *mockQuotaRepository) MarkNotificationSent(ctx context.Context, notificationID uuid.UUID) error {
	for _, notifications := range m.notifications {
		for _, notification := range notifications {
			if notification.ID == notificationID {
				notification.Sent = true
				now := time.Now()
				notification.SentAt = &now
				return nil
			}
		}
	}
	return assert.AnError
}

func (m *mockQuotaRepository) GetNotificationHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.QuotaNotification, error) {
	key := tenantID.String()
	return m.notifications[key], nil
}

func TestMockQuotaRepository_IncrementUsage(t *testing.T) {
	repo := newMockQuotaRepository()
	tenantID := uuid.New()

	// Test incrementing usage
	err := repo.IncrementUsage(context.Background(), tenantID, true)
	assert.NoError(t, err)

	err = repo.IncrementUsage(context.Background(), tenantID, false)
	assert.NoError(t, err)

	// Verify usage was incremented
	now := time.Now()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	usage, err := repo.GetQuotaUsageByTenant(context.Background(), tenantID, month)
	
	assert.NoError(t, err)
	assert.Equal(t, 2, usage.RequestCount)
	assert.Equal(t, 1, usage.SuccessCount)
	assert.Equal(t, 1, usage.FailureCount)
}

func TestMockQuotaRepository_BillingOperations(t *testing.T) {
	repo := newMockQuotaRepository()
	tenantID := uuid.New()
	billingPeriod := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	record := &models.BillingRecord{
		TenantID:        tenantID,
		BillingPeriod:   billingPeriod,
		BaseRequests:    5000,
		OverageRequests: 0,
		TotalAmount:     0,
		Status:          "pending",
	}

	// Test creating billing record
	err := repo.CreateBillingRecord(context.Background(), record)
	assert.NoError(t, err)

	// Test getting billing record
	retrieved, err := repo.GetBillingRecord(context.Background(), tenantID, billingPeriod)
	assert.NoError(t, err)
	assert.Equal(t, record.TenantID, retrieved.TenantID)
	assert.Equal(t, record.BaseRequests, retrieved.BaseRequests)

	// Test updating billing record
	retrieved.Status = "processed"
	err = repo.UpdateBillingRecord(context.Background(), retrieved)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetBillingRecord(context.Background(), tenantID, billingPeriod)
	assert.NoError(t, err)
	assert.Equal(t, "processed", updated.Status)
}

func TestQuotaRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(QuotaRepositoryTestSuite))
}
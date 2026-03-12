package repository

import (
	"context"
	"github.com/josedab/waas/pkg/models"
	"time"

	"github.com/google/uuid"
)

// TenantRepository defines the interface for tenant data operations
type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Tenant, error)
	FindByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.Tenant, error)
}

// WebhookEndpointRepository defines the interface for webhook endpoint data operations
type WebhookEndpointRepository interface {
	Create(ctx context.Context, endpoint *models.WebhookEndpoint) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error)
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error)
	GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error)
	Update(ctx context.Context, endpoint *models.WebhookEndpoint) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
	UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error
}

// DeliveryHistoryFilters defines filters for delivery history queries
type DeliveryHistoryFilters struct {
	EndpointIDs []uuid.UUID `json:"endpoint_ids,omitempty"`
	Statuses    []string    `json:"statuses,omitempty"`
	StartDate   time.Time   `json:"start_date,omitempty"`
	EndDate     time.Time   `json:"end_date,omitempty"`
}

// DeliveryAttemptRepository defines the interface for delivery attempt data operations
type DeliveryAttemptRepository interface {
	Create(ctx context.Context, attempt *models.DeliveryAttempt) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryAttempt, error)
	GetByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.DeliveryAttempt, error)
	GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.DeliveryAttempt, error)
	GetPendingDeliveries(ctx context.Context, limit int) ([]*models.DeliveryAttempt, error)
	Update(ctx context.Context, attempt *models.DeliveryAttempt) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetDeliveryHistory(ctx context.Context, endpointID uuid.UUID, statuses []string, limit, offset int) ([]*models.DeliveryAttempt, error)
	GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error)
	GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error)
}

// QuotaRepository defines the interface for quota and billing data operations
type QuotaRepository interface {
	// Quota Usage operations
	GetOrCreateQuotaUsage(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error)
	UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error
	IncrementUsage(ctx context.Context, tenantID uuid.UUID, success bool) error
	GetQuotaUsageByTenant(ctx context.Context, tenantID uuid.UUID, month time.Time) (*models.QuotaUsage, error)

	// Billing operations
	CreateBillingRecord(ctx context.Context, record *models.BillingRecord) error
	GetBillingRecord(ctx context.Context, tenantID uuid.UUID, billingPeriod time.Time) (*models.BillingRecord, error)
	UpdateBillingRecord(ctx context.Context, record *models.BillingRecord) error
	GetBillingHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BillingRecord, error)

	// Notification operations
	CreateQuotaNotification(ctx context.Context, notification *models.QuotaNotification) error
	GetPendingNotifications(ctx context.Context, tenantID uuid.UUID) ([]*models.QuotaNotification, error)
	MarkNotificationSent(ctx context.Context, notificationID uuid.UUID) error
	GetNotificationHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.QuotaNotification, error)
}

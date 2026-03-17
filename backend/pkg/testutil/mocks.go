package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
)

// Compile-time interface checks.
var (
	_ repository.TenantRepository          = (*MockTenantRepository)(nil)
	_ repository.WebhookEndpointRepository = (*MockWebhookEndpointRepository)(nil)
	_ repository.DeliveryAttemptRepository = (*MockDeliveryAttemptRepository)(nil)
	_ queue.PublisherInterface             = (*MockPublisher)(nil)
)

// ──────────────────────────────────────────────
// MockTenantRepository
// ──────────────────────────────────────────────

// MockTenantRepository is an in-memory implementation of repository.TenantRepository.
type MockTenantRepository struct {
	mu      sync.RWMutex
	tenants map[uuid.UUID]*models.Tenant

	// Optional hooks for custom behavior in tests.
	CreateFunc       func(ctx context.Context, tenant *models.Tenant) error
	GetByIDFunc      func(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	FindByAPIKeyFunc func(ctx context.Context, apiKey string) (*models.Tenant, error)
}

func NewMockTenantRepository() *MockTenantRepository {
	return &MockTenantRepository{tenants: make(map[uuid.UUID]*models.Tenant)}
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, tenant)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tenants[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *MockTenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tenants {
		if t.APIKeyHash == apiKeyHash {
			return t, nil
		}
	}
	return nil, nil
}

func (m *MockTenantRepository) FindByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	if m.FindByAPIKeyFunc != nil {
		return m.FindByAPIKeyFunc(ctx, apiKey)
	}
	return nil, nil
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tenants, id)
	return nil
}

func (m *MockTenantRepository) List(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result, nil
}

// ──────────────────────────────────────────────
// MockWebhookEndpointRepository
// ──────────────────────────────────────────────

type MockWebhookEndpointRepository struct {
	mu        sync.RWMutex
	endpoints map[uuid.UUID]*models.WebhookEndpoint
}

func NewMockWebhookEndpointRepository() *MockWebhookEndpointRepository {
	return &MockWebhookEndpointRepository{endpoints: make(map[uuid.UUID]*models.WebhookEndpoint)}
}

func (m *MockWebhookEndpointRepository) Create(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endpoints[endpoint.ID] = endpoint
	return nil
}

func (m *MockWebhookEndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ep, ok := m.endpoints[id]
	if !ok {
		return nil, nil
	}
	return ep, nil
}

func (m *MockWebhookEndpointRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.WebhookEndpoint
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			result = append(result, ep)
		}
	}
	return result, nil
}

func (m *MockWebhookEndpointRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockWebhookEndpointRepository) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.WebhookEndpoint
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID && ep.IsActive {
			result = append(result, ep)
		}
	}
	return result, nil
}

func (m *MockWebhookEndpointRepository) Update(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endpoints[endpoint.ID] = endpoint
	return nil
}

func (m *MockWebhookEndpointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.endpoints, id)
	return nil
}

func (m *MockWebhookEndpointRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ep, ok := m.endpoints[id]; ok {
		ep.IsActive = active
	}
	return nil
}

func (m *MockWebhookEndpointRepository) UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error {
	return m.SetActive(ctx, id, active)
}

// ──────────────────────────────────────────────
// MockDeliveryAttemptRepository
// ──────────────────────────────────────────────

type MockDeliveryAttemptRepository struct {
	mu       sync.RWMutex
	attempts map[uuid.UUID]*models.DeliveryAttempt
}

func NewMockDeliveryAttemptRepository() *MockDeliveryAttemptRepository {
	return &MockDeliveryAttemptRepository{attempts: make(map[uuid.UUID]*models.DeliveryAttempt)}
}

func (m *MockDeliveryAttemptRepository) Create(ctx context.Context, attempt *models.DeliveryAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *MockDeliveryAttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryAttempt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.attempts[id]
	if !ok {
		return nil, nil
	}
	return a, nil
}

func (m *MockDeliveryAttemptRepository) GetByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.DeliveryAttempt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.DeliveryAttempt
	for _, a := range m.attempts {
		if a.EndpointID == endpointID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *MockDeliveryAttemptRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.DeliveryAttempt
	for _, a := range m.attempts {
		if a.Status == status {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *MockDeliveryAttemptRepository) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.DeliveryAttempt, error) {
	return m.GetByStatus(ctx, "pending", limit, 0)
}

func (m *MockDeliveryAttemptRepository) Update(ctx context.Context, attempt *models.DeliveryAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *MockDeliveryAttemptRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.attempts[id]; ok {
		a.Status = status
	}
	return nil
}

func (m *MockDeliveryAttemptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.attempts, id)
	return nil
}

func (m *MockDeliveryAttemptRepository) GetDeliveryHistory(ctx context.Context, endpointID uuid.UUID, statuses []string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	return m.GetByEndpointID(ctx, endpointID, limit, offset)
}

func (m *MockDeliveryAttemptRepository) GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters repository.DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.DeliveryAttempt
	for _, a := range m.attempts {
		result = append(result, a)
	}
	return result, len(result), nil
}

func (m *MockDeliveryAttemptRepository) GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error) {
	return m.GetByEndpointID(ctx, deliveryID, 100, 0)
}

func (m *MockDeliveryAttemptRepository) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

// ──────────────────────────────────────────────
// MockPublisher
// ──────────────────────────────────────────────

// MockPublisher is an in-memory implementation of queue.PublisherInterface.
type MockPublisher struct {
	mu       sync.Mutex
	Messages []*queue.DeliveryMessage
	DLQ      []*queue.DeliveryMessage

	PublishFunc func(ctx context.Context, message *queue.DeliveryMessage) error
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

func (m *MockPublisher) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, message)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, message)
	return nil
}

func (m *MockPublisher) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	return m.PublishDelivery(ctx, message)
}

func (m *MockPublisher) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DLQ = append(m.DLQ, message)
	return nil
}

func (m *MockPublisher) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.Messages)), nil
}

func (m *MockPublisher) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]int64{
		"messages": int64(len(m.Messages)),
		"dlq":      int64(len(m.DLQ)),
	}, nil
}

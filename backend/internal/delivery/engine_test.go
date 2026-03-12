package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWebhookRepository is a mock implementation of WebhookEndpointRepository
type MockWebhookRepository struct {
	mock.Mock
}

func (m *MockWebhookRepository) Create(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockWebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookRepository) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookRepository) Update(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockWebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWebhookRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockWebhookRepository) UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockWebhookRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

// MockDeliveryRepository is a mock implementation of DeliveryAttemptRepository
type MockDeliveryRepository struct {
	mock.Mock
}

func (m *MockDeliveryRepository) Create(ctx context.Context, attempt *models.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *MockDeliveryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryAttempt, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryRepository) GetByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, endpointID, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryRepository) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryRepository) Update(ctx context.Context, attempt *models.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *MockDeliveryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDeliveryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDeliveryRepository) GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, deliveryID, tenantID)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryRepository) GetDeliveryHistory(ctx context.Context, endpointID uuid.UUID, statuses []string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, endpointID, statuses, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryRepository) GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters repository.DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error) {
	args := m.Called(ctx, tenantID, filters, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Int(1), args.Error(2)
}

// createTestEngine creates a delivery engine with mocked dependencies for testing
func createTestEngine() (*DeliveryEngine, *MockWebhookRepository, *MockDeliveryRepository) {
	mockWebhookRepo := &MockWebhookRepository{}
	mockDeliveryRepo := &MockDeliveryRepository{}
	logger := utils.NewLogger("test")

	engine := &DeliveryEngine{
		webhookRepo:   mockWebhookRepo,
		deliveryRepo:  mockDeliveryRepo,
		httpClient:    createHTTPClient(getDeliveryConfig()),
		healthMonitor: NewEndpointHealthMonitor(mockWebhookRepo, logger),
		logger:        logger,
	}

	return engine, mockWebhookRepo, mockDeliveryRepo
}

func TestDeliveryEngine_HandleDelivery_Success(t *testing.T) {
	// Create test server that returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "received"}`))
	}))
	defer server.Close()

	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	deliveryID := uuid.New()

	// Mock webhook endpoint
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      server.URL,
		IsActive: true,
		CustomHeaders: map[string]string{
			"X-Custom-Header": "test-value",
		},
	}

	// Mock delivery message
	payload := json.RawMessage(`{"event": "test", "data": {"id": 123}}`)
	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		Payload:       payload,
		Headers:       map[string]string{"X-Event-Type": "test"},
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
		Signature:     "sha256=test-signature",
	}

	// Set up mocks
	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	// Execute
	result, err := engine.HandleDelivery(context.Background(), message)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusSuccess, result.Status)
	assert.Equal(t, deliveryID, result.DeliveryID)
	assert.Equal(t, 200, *result.HTTPStatus)
	assert.NotNil(t, result.DeliveredAt)

	mockWebhookRepo.AssertExpectations(t)
	mockDeliveryRepo.AssertExpectations(t)
}

func TestDeliveryEngine_HandleDelivery_HTTPError(t *testing.T) {
	// Create test server that returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	deliveryID := uuid.New()

	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      server.URL,
		IsActive: true,
	}

	payload := json.RawMessage(`{"event": "test"}`)
	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		Payload:       payload,
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	result, err := engine.HandleDelivery(context.Background(), message)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusRetrying, result.Status) // Should retry on 500 error
	assert.Equal(t, 500, *result.HTTPStatus)
	assert.Contains(t, *result.ErrorMessage, "HTTP 500")

	mockWebhookRepo.AssertExpectations(t)
	mockDeliveryRepo.AssertExpectations(t)
}

func TestDeliveryEngine_HandleDelivery_InactiveEndpoint(t *testing.T) {
	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	deliveryID := uuid.New()

	// Inactive endpoint
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://example.com/webhook",
		IsActive: false,
	}

	payload := json.RawMessage(`{"event": "test"}`)
	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		Payload:       payload,
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	// For inactive endpoints, we still create and update the delivery attempt record
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	result, err := engine.HandleDelivery(context.Background(), message)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusFailed, result.Status)
	assert.Contains(t, *result.ErrorMessage, "endpoint is inactive")

	mockWebhookRepo.AssertExpectations(t)
	mockDeliveryRepo.AssertExpectations(t)
}

func TestDeliveryEngine_HandleDelivery_EndpointNotFound(t *testing.T) {
	engine, mockWebhookRepo, _ := createTestEngine()

	endpointID := uuid.New()
	deliveryID := uuid.New()

	payload := json.RawMessage(`{"event": "test"}`)
	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		Payload:       payload,
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return((*models.WebhookEndpoint)(nil), fmt.Errorf("endpoint not found"))

	result, err := engine.HandleDelivery(context.Background(), message)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusFailed, result.Status)
	assert.Contains(t, *result.ErrorMessage, "endpoint not found")

	mockWebhookRepo.AssertExpectations(t)
}

func TestDeliveryEngine_HandleDelivery_NetworkError(t *testing.T) {
	// Create a server that hangs to trigger timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the client timeout to trigger timeout error
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()
	// Set a very short timeout to trigger timeout quickly
	engine.httpClient.Timeout = 100 * time.Millisecond

	endpointID := uuid.New()
	deliveryID := uuid.New()

	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      server.URL,
		IsActive: true,
	}

	payload := json.RawMessage(`{"event": "test"}`)
	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		Payload:       payload,
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	result, err := engine.HandleDelivery(context.Background(), message)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusRetrying, result.Status) // Timeout errors should be retryable
	assert.Contains(t, *result.ErrorMessage, "request failed")

	mockWebhookRepo.AssertExpectations(t)
	mockDeliveryRepo.AssertExpectations(t)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "no such host error",
			err:      fmt.Errorf("dial tcp: lookup invalid-host: no such host"),
			expected: true,
		},
		{
			name:     "connection refused error",
			err:      fmt.Errorf("dial tcp 127.0.0.1:8080: connect: connection refused"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{408, true}, // Request Timeout
		{410, false},
		{429, true}, // Too Many Requests
		{500, true}, // Internal Server Error
		{502, true}, // Bad Gateway
		{503, true}, // Service Unavailable
		{504, true}, // Gateway Timeout
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := isRetryableStatusCode(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateHTTPClient(t *testing.T) {
	config := DeliveryConfig{
		RequestTimeout:        30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := createHTTPClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, config.RequestTimeout, client.Timeout)

	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok)
	assert.Equal(t, config.MaxIdleConns, transport.MaxIdleConns)
	assert.Equal(t, config.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	assert.Equal(t, config.IdleConnTimeout, transport.IdleConnTimeout)
	assert.Equal(t, config.TLSHandshakeTimeout, transport.TLSHandshakeTimeout)
	assert.Equal(t, config.ResponseHeaderTimeout, transport.ResponseHeaderTimeout)
}

// --- Benchmarks ---

func BenchmarkHandleDelivery_Success(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      server.URL,
		IsActive: true,
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	payload := json.RawMessage(`{"event":"bench","data":{"id":1}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := &queue.DeliveryMessage{
			DeliveryID:    uuid.New(),
			EndpointID:    endpointID,
			Payload:       payload,
			AttemptNumber: 1,
			MaxAttempts:   3,
			ScheduledAt:   time.Now(),
		}
		_, _ = engine.HandleDelivery(context.Background(), message)
	}
}

func BenchmarkIsRetryableStatusCode(b *testing.B) {
	codes := []int{200, 400, 408, 429, 500, 502, 503, 504}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isRetryableStatusCode(codes[i%len(codes)])
	}
}

func BenchmarkCreateHTTPClient(b *testing.B) {
	config := getDeliveryConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createHTTPClient(config)
	}
}

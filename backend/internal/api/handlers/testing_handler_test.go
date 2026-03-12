package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock repositories
type MockWebhookRepo struct {
	mock.Mock
}

func (m *MockWebhookRepo) Create(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockWebhookRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookRepo) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookRepo) Update(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockWebhookRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWebhookRepo) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockWebhookRepo) UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockWebhookRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

type MockDeliveryAttemptRepo struct {
	mock.Mock
}

func (m *MockDeliveryAttemptRepo) Create(ctx context.Context, attempt *models.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryAttempt, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepo) GetByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, endpointID, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepo) GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepo) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepo) Update(ctx context.Context, attempt *models.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepo) GetDeliveryHistory(ctx context.Context, endpointID uuid.UUID, statuses []string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, endpointID, statuses, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepo) GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters repository.DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error) {
	args := m.Called(ctx, tenantID, filters, limit, offset)
	return args.Get(0).([]*models.DeliveryAttempt), args.Int(1), args.Error(2)
}

func (m *MockDeliveryAttemptRepo) GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error) {
	args := m.Called(ctx, deliveryID, tenantID)
	return args.Get(0).([]*models.DeliveryAttempt), args.Error(1)
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockPublisher) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	args := m.Called(ctx, message, delay)
	return args.Error(0)
}

func (m *MockPublisher) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	args := m.Called(ctx, message, reason)
	return args.Error(0)
}

func (m *MockPublisher) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	args := m.Called(ctx, queueName)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPublisher) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]int64), args.Error(1)
}

func TestTestingHandler_TestWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := new(MockWebhookRepo)
	mockDeliveryRepo := new(MockDeliveryAttemptRepo)
	mockPublisher := new(MockPublisher)
	logger := utils.NewTestLogger()

	handler := NewTestingHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)
	handler.SetURLValidator(&noopURLValidator{})
	handler.SetHTTPClientFactory(func(timeout time.Duration) *http.Client {
		return &http.Client{Timeout: timeout}
	})
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "received"}`))
	}))
	defer testServer.Close()

	// Create test request
	testReq := TestWebhookRequest{
		URL:     testServer.URL,
		Payload: json.RawMessage(`{"test": "data"}`),
		Headers: map[string]string{
			"X-Custom": "value",
		},
		Method:  "POST",
		Timeout: 10,
	}

	reqBody, err := json.Marshal(testReq)
	require.NoError(t, err)

	// Setup Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/webhooks/test", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", uuid.New())

	// Execute handler
	handler.TestWebhook(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, response.TestID)
	assert.Equal(t, testServer.URL, response.URL)
	assert.Equal(t, "success", response.Status)
	assert.NotNil(t, response.HTTPStatus)
	assert.Equal(t, 200, *response.HTTPStatus)
	assert.NotNil(t, response.Latency)
	assert.NotEmpty(t, response.RequestID)
}

func TestTestingHandler_TestWebhookInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := new(MockWebhookRepo)
	mockDeliveryRepo := new(MockDeliveryAttemptRepo)
	mockPublisher := new(MockPublisher)
	logger := utils.NewTestLogger()

	handler := NewTestingHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)
	// Use the default URL validator (not noop) so invalid URLs are properly rejected

	// Create test request with invalid URL
	testReq := TestWebhookRequest{
		URL:     "invalid-url",
		Payload: json.RawMessage(`{"test": "data"}`),
	}

	reqBody, err := json.Marshal(testReq)
	require.NoError(t, err)

	// Setup Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/webhooks/test", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", uuid.New())

	// Execute handler
	handler.TestWebhook(c)

	// Verify response - invalid URL should be rejected by the URL validator
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_URL", response.Code)
}

func TestTestingHandler_CreateTestEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := new(MockWebhookRepo)
	mockDeliveryRepo := new(MockDeliveryAttemptRepo)
	mockPublisher := new(MockPublisher)
	logger := utils.NewTestLogger()

	handler := NewTestingHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)
	handler.SetURLValidator(&noopURLValidator{})

	// Create test request
	testReq := CreateTestEndpointRequest{
		Name:        "Test Endpoint",
		Description: "For testing",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
		TTL: 3600,
	}

	reqBody, err := json.Marshal(testReq)
	require.NoError(t, err)

	// Setup Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/webhooks/test/endpoints", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", uuid.New())

	// Execute handler
	handler.CreateTestEndpoint(c)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code)

	var response TestEndpointResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Contains(t, response.URL, response.ID.String())
	assert.Equal(t, "Test Endpoint", response.Name)
	assert.Equal(t, "For testing", response.Description)
	assert.Equal(t, "Bearer token", response.Headers["Authorization"])
	assert.True(t, response.ExpiresAt.After(response.CreatedAt))
}

func TestTestingHandler_InspectDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := new(MockWebhookRepo)
	mockDeliveryRepo := new(MockDeliveryAttemptRepo)
	mockPublisher := new(MockPublisher)
	logger := utils.NewTestLogger()

	handler := NewTestingHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)
	handler.SetURLValidator(&noopURLValidator{})

	// Test data
	tenantID := uuid.New()
	deliveryID := uuid.New()
	endpointID := uuid.New()

	endpoint := &models.WebhookEndpoint{
		ID:         endpointID,
		TenantID:   tenantID,
		URL:        "https://example.com/webhook",
		SecretHash: "secret-hash",
		IsActive:   true,
		CustomHeaders: map[string]string{
			"X-Custom": "value",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	attempts := []*models.DeliveryAttempt{
		{
			ID:            deliveryID,
			EndpointID:    endpointID,
			PayloadHash:   "abc123",
			PayloadSize:   100,
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			ResponseBody:  &[]string{"OK"}[0],
			AttemptNumber: 1,
			ScheduledAt:   time.Now().Add(-5 * time.Minute),
			DeliveredAt:   &[]time.Time{time.Now().Add(-4 * time.Minute)}[0],
			CreatedAt:     time.Now().Add(-5 * time.Minute),
		},
	}

	// Setup mock expectations
	mockDeliveryRepo.On("GetDeliveryAttemptsByDeliveryID", mock.Anything, deliveryID, tenantID).Return(attempts, nil)
	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)

	// Setup Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/webhooks/deliveries/"+deliveryID.String()+"/inspect", nil)
	c.Set("tenant_id", tenantID)
	c.Params = []gin.Param{{Key: "id", Value: deliveryID.String()}}

	// Execute handler
	handler.InspectDelivery(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response DeliveryInspectionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, deliveryID, response.DeliveryID)
	assert.Equal(t, endpointID, response.EndpointID)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, 1, response.AttemptNumber)

	// Verify request details
	assert.NotNil(t, response.Request)
	assert.Equal(t, endpoint.URL, response.Request.URL)
	assert.Equal(t, "POST", response.Request.Method)

	// Verify response details
	assert.NotNil(t, response.Response)
	assert.Equal(t, 200, response.Response.HTTPStatus)
	assert.Equal(t, "OK", response.Response.Body)

	// Verify timeline
	assert.NotEmpty(t, response.Timeline)

	// Verify mock calls
	mockDeliveryRepo.AssertExpectations(t)
	mockWebhookRepo.AssertExpectations(t)
}

func TestTestingHandler_GetDeliveryLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := new(MockWebhookRepo)
	mockDeliveryRepo := new(MockDeliveryAttemptRepo)
	mockPublisher := new(MockPublisher)
	logger := utils.NewTestLogger()

	handler := NewTestingHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)
	handler.SetURLValidator(&noopURLValidator{})

	// Test data
	tenantID := uuid.New()
	deliveryID := uuid.New()
	endpointID := uuid.New()

	attempts := []*models.DeliveryAttempt{
		{
			ID:            deliveryID,
			EndpointID:    endpointID,
			PayloadHash:   "abc123",
			PayloadSize:   100,
			Status:        "failed",
			HTTPStatus:    &[]int{500}[0],
			ErrorMessage:  &[]string{"Internal Server Error"}[0],
			AttemptNumber: 1,
			ScheduledAt:   time.Now().Add(-10 * time.Minute),
			DeliveredAt:   &[]time.Time{time.Now().Add(-9 * time.Minute)}[0],
			CreatedAt:     time.Now().Add(-10 * time.Minute),
		},
		{
			ID:            uuid.New(),
			EndpointID:    endpointID,
			PayloadHash:   "abc123",
			PayloadSize:   100,
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			ResponseBody:  &[]string{"OK"}[0],
			AttemptNumber: 2,
			ScheduledAt:   time.Now().Add(-5 * time.Minute),
			DeliveredAt:   &[]time.Time{time.Now().Add(-4 * time.Minute)}[0],
			CreatedAt:     time.Now().Add(-5 * time.Minute),
		},
	}

	// Setup mock expectations
	mockDeliveryRepo.On("GetDeliveryAttemptsByDeliveryID", mock.Anything, deliveryID, tenantID).Return(attempts, nil)

	// Setup Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/webhooks/deliveries/"+deliveryID.String()+"/logs", nil)
	c.Set("tenant_id", tenantID)
	c.Params = []gin.Param{{Key: "id", Value: deliveryID.String()}}

	// Execute handler
	handler.GetDeliveryLogs(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, deliveryID.String(), response["delivery_id"])
	assert.Equal(t, float64(2), response["total_attempts"])

	logs, ok := response["logs"].([]interface{})
	require.True(t, ok)
	assert.Len(t, logs, 2)

	// Verify mock calls
	mockDeliveryRepo.AssertExpectations(t)
}

func TestTestingHandler_BuildErrorDetails(t *testing.T) {
	// Setup handler
	mockWebhookRepo := new(MockWebhookRepo)
	mockDeliveryRepo := new(MockDeliveryAttemptRepo)
	mockPublisher := new(MockPublisher)
	logger := utils.NewTestLogger()

	handler := NewTestingHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)
	handler.SetURLValidator(&noopURLValidator{})

	// Test timeout error
	timeoutAttempt := &models.DeliveryAttempt{
		Status:       "failed",
		ErrorMessage: &[]string{"HTTP request failed: timeout"}[0],
	}

	errorDetails := handler.buildErrorDetails(timeoutAttempt, 3)
	assert.Equal(t, "timeout", errorDetails.ErrorType)
	assert.Equal(t, "HTTP request failed: timeout", errorDetails.ErrorMessage)
	assert.Equal(t, 3, errorDetails.RetryCount)
	assert.Contains(t, errorDetails.Suggestions[0], "timeout")

	// Test HTTP 404 error
	notFoundAttempt := &models.DeliveryAttempt{
		Status:     "failed",
		HTTPStatus: &[]int{404}[0],
	}

	errorDetails = handler.buildErrorDetails(notFoundAttempt, 1)
	assert.Equal(t, "not_found", errorDetails.ErrorType)
	assert.Equal(t, 404, *errorDetails.HTTPStatus)
	assert.Contains(t, errorDetails.Suggestions[0], "URL path")

	// Test connection error
	connectionAttempt := &models.DeliveryAttempt{
		Status:       "failed",
		ErrorMessage: &[]string{"connection refused"}[0],
	}

	errorDetails = handler.buildErrorDetails(connectionAttempt, 2)
	assert.Equal(t, "connection", errorDetails.ErrorType)
	assert.Contains(t, errorDetails.Suggestions[0], "accessible")
}

func TestTestEndpointHandler_ReceiveTestWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := utils.NewTestLogger()
	handler := NewTestEndpointHandler(logger)

	endpointID := uuid.New()
	payload := `{"event": "test", "data": {"id": 123}}`

	// Setup Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/test/"+endpointID.String(), strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Webhook-ID", "test-123")
	c.Params = []gin.Param{{Key: "endpoint_id", Value: endpointID.String()}}

	// Execute handler
	handler.ReceiveTestWebhook(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Webhook received successfully", response["message"])
	assert.NotEmpty(t, response["receive_id"])
	assert.NotEmpty(t, response["timestamp"])
}

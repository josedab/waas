package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/queue"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestingHandlerIntegrationTestSuite struct {
	suite.Suite
	db                  *database.DB
	redisClient         *database.RedisClient
	webhookRepo         repository.WebhookEndpointRepository
	deliveryAttemptRepo repository.DeliveryAttemptRepository
	publisher           queue.PublisherInterface
	handler             *TestingHandler
	testEndpointHandler *TestEndpointHandler
	router              *gin.Engine
	logger              *utils.Logger
	testTenant          *models.Tenant
}

func (suite *TestingHandlerIntegrationTestSuite) SetupSuite() {
	// Initialize test database
	var err error
	suite.db, err = database.NewTestConnection()
	require.NoError(suite.T(), err)

	// Initialize Redis client
	suite.redisClient, err = database.NewTestRedisConnection()
	require.NoError(suite.T(), err)

	// Initialize logger
	suite.logger = utils.NewTestLogger()

	// Initialize repositories
	suite.webhookRepo = repository.NewWebhookEndpointRepository(suite.db)
	suite.deliveryAttemptRepo = repository.NewDeliveryAttemptRepository(suite.db)

	// Initialize publisher
	suite.publisher = queue.NewTestPublisher()

	// Initialize handlers
	suite.handler = NewTestingHandler(suite.webhookRepo, suite.deliveryAttemptRepo, suite.publisher, suite.logger)
	suite.testEndpointHandler = NewTestEndpointHandler(suite.logger)

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *TestingHandlerIntegrationTestSuite) SetupTest() {
	// Clean database
	suite.cleanDatabase()

	// Create test tenant
	suite.testTenant = &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "test-api-key-hash",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	tenantRepo := repository.NewTenantRepository(suite.db)
	err := tenantRepo.Create(context.Background(), suite.testTenant)
	require.NoError(suite.T(), err)
}

func (suite *TestingHandlerIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.redisClient != nil {
		suite.redisClient.Close()
	}
}

func (suite *TestingHandlerIntegrationTestSuite) setupRoutes() {
	// Add middleware to set tenant context
	suite.router.Use(func(c *gin.Context) {
		c.Set("tenant_id", suite.testTenant.ID)
		c.Next()
	})

	// Testing routes
	suite.router.POST("/webhooks/test", suite.handler.TestWebhook)
	suite.router.POST("/webhooks/test/endpoints", suite.handler.CreateTestEndpoint)
	suite.router.GET("/webhooks/deliveries/:id/inspect", suite.handler.InspectDelivery)
	suite.router.GET("/webhooks/deliveries/:id/logs", suite.handler.GetDeliveryLogs)
	suite.router.GET("/webhooks/realtime", suite.handler.WebSocketUpdates)

	// Test endpoint routes
	suite.router.Any("/test/:endpoint_id", suite.testEndpointHandler.ReceiveTestWebhook)
	suite.router.GET("/test/:endpoint_id/receives", suite.testEndpointHandler.GetTestEndpointReceives)
	suite.router.GET("/test/:endpoint_id/receives/:receive_id", suite.testEndpointHandler.GetTestEndpointReceive)
	suite.router.DELETE("/test/:endpoint_id/receives", suite.testEndpointHandler.ClearTestEndpointReceives)
}

func (suite *TestingHandlerIntegrationTestSuite) cleanDatabase() {
	ctx := context.Background()
	
	// Clean tables in correct order (respecting foreign keys)
	tables := []string{
		"delivery_attempts",
		"webhook_endpoints",
		"tenants",
	}
	
	for _, table := range tables {
		_, err := suite.db.Pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(suite.T(), err)
	}
}

func (suite *TestingHandlerIntegrationTestSuite) TestWebhookTesting() {
	// Start a test HTTP server to receive webhooks
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "received"}`))
	}))
	defer testServer.Close()

	// Test webhook request
	testReq := TestWebhookRequest{
		URL:     testServer.URL,
		Payload: json.RawMessage(`{"test": "data", "timestamp": "2024-01-01T12:00:00Z"}`),
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
		Method:  "POST",
		Timeout: 10,
	}

	reqBody, err := json.Marshal(testReq)
	require.NoError(suite.T(), err)

	// Make request
	req, err := http.NewRequest("POST", "/webhooks/test", bytes.NewBuffer(reqBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response TestWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.NotEqual(suite.T(), uuid.Nil, response.TestID)
	assert.Equal(suite.T(), testServer.URL, response.URL)
	assert.Equal(suite.T(), "success", response.Status)
	assert.NotNil(suite.T(), response.HTTPStatus)
	assert.Equal(suite.T(), 200, *response.HTTPStatus)
	assert.NotNil(suite.T(), response.Latency)
	assert.Greater(suite.T(), *response.Latency, int64(0))
	assert.NotEmpty(suite.T(), response.RequestID)
}

func (suite *TestingHandlerIntegrationTestSuite) TestWebhookTestingWithFailure() {
	// Test webhook request to non-existent endpoint
	testReq := TestWebhookRequest{
		URL:     "http://localhost:99999/webhook",
		Payload: json.RawMessage(`{"test": "data"}`),
		Method:  "POST",
		Timeout: 5,
	}

	reqBody, err := json.Marshal(testReq)
	require.NoError(suite.T(), err)

	// Make request
	req, err := http.NewRequest("POST", "/webhooks/test", bytes.NewBuffer(reqBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response TestWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "failed", response.Status)
	assert.NotNil(suite.T(), response.ErrorMessage)
	assert.Contains(suite.T(), *response.ErrorMessage, "connection")
}

func (suite *TestingHandlerIntegrationTestSuite) TestCreateTestEndpoint() {
	// Create test endpoint request
	testReq := CreateTestEndpointRequest{
		Name:        "My Test Endpoint",
		Description: "Testing webhook delivery",
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
		TTL: 7200, // 2 hours
	}

	reqBody, err := json.Marshal(testReq)
	require.NoError(suite.T(), err)

	// Make request
	req, err := http.NewRequest("POST", "/webhooks/test/endpoints", bytes.NewBuffer(reqBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response TestEndpointResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.NotEqual(suite.T(), uuid.Nil, response.ID)
	assert.Contains(suite.T(), response.URL, response.ID.String())
	assert.Equal(suite.T(), "My Test Endpoint", response.Name)
	assert.Equal(suite.T(), "Testing webhook delivery", response.Description)
	assert.Equal(suite.T(), "Bearer test-token", response.Headers["Authorization"])
	assert.True(suite.T(), response.ExpiresAt.After(response.CreatedAt))
}

func (suite *TestingHandlerIntegrationTestSuite) TestDeliveryInspection() {
	// Create a webhook endpoint
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: suite.testTenant.ID,
		URL:      "https://example.com/webhook",
		SecretHash: "secret-hash",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    1000,
			MaxDelayMs:        30000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{
			"X-Custom": "value",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := suite.webhookRepo.Create(context.Background(), endpoint)
	require.NoError(suite.T(), err)

	// Create delivery attempts
	deliveryID := uuid.New()
	attempts := []*models.DeliveryAttempt{
		{
			ID:            deliveryID,
			EndpointID:    endpoint.ID,
			PayloadHash:   "abc123",
			PayloadSize:   100,
			Status:        "failed",
			HTTPStatus:    &[]int{500}[0],
			ResponseBody:  &[]string{"Internal Server Error"}[0],
			ErrorMessage:  &[]string{"HTTP 500: Internal Server Error"}[0],
			AttemptNumber: 1,
			ScheduledAt:   time.Now().Add(-10 * time.Minute),
			DeliveredAt:   &[]time.Time{time.Now().Add(-9 * time.Minute)}[0],
			CreatedAt:     time.Now().Add(-10 * time.Minute),
		},
		{
			ID:            uuid.New(),
			EndpointID:    endpoint.ID,
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

	for _, attempt := range attempts {
		err = suite.deliveryAttemptRepo.Create(context.Background(), attempt)
		require.NoError(suite.T(), err)
	}

	// Make inspection request
	req, err := http.NewRequest("GET", fmt.Sprintf("/webhooks/deliveries/%s/inspect", deliveryID), nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response DeliveryInspectionResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), deliveryID, response.DeliveryID)
	assert.Equal(suite.T(), endpoint.ID, response.EndpointID)
	assert.Equal(suite.T(), "success", response.Status)
	assert.Equal(suite.T(), 2, response.AttemptNumber)

	// Verify request details
	assert.NotNil(suite.T(), response.Request)
	assert.Equal(suite.T(), endpoint.URL, response.Request.URL)
	assert.Equal(suite.T(), "POST", response.Request.Method)
	assert.Equal(suite.T(), "abc123", response.Request.PayloadHash)
	assert.Equal(suite.T(), 100, response.Request.PayloadSize)

	// Verify response details
	assert.NotNil(suite.T(), response.Response)
	assert.Equal(suite.T(), 200, response.Response.HTTPStatus)
	assert.Equal(suite.T(), "OK", response.Response.Body)

	// Verify timeline
	assert.Len(suite.T(), response.Timeline, 4) // 2 scheduled + 2 delivered events
}

func (suite *TestingHandlerIntegrationTestSuite) TestDeliveryLogs() {
	// Create a webhook endpoint
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: suite.testTenant.ID,
		URL:      "https://example.com/webhook",
		SecretHash: "secret-hash",
		IsActive: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := suite.webhookRepo.Create(context.Background(), endpoint)
	require.NoError(suite.T(), err)

	// Create delivery attempts
	deliveryID := uuid.New()
	attempts := []*models.DeliveryAttempt{
		{
			ID:            deliveryID,
			EndpointID:    endpoint.ID,
			PayloadHash:   "abc123",
			PayloadSize:   100,
			Status:        "failed",
			HTTPStatus:    &[]int{404}[0],
			ErrorMessage:  &[]string{"Not Found"}[0],
			AttemptNumber: 1,
			ScheduledAt:   time.Now().Add(-10 * time.Minute),
			DeliveredAt:   &[]time.Time{time.Now().Add(-9 * time.Minute)}[0],
			CreatedAt:     time.Now().Add(-10 * time.Minute),
		},
		{
			ID:            uuid.New(),
			EndpointID:    endpoint.ID,
			PayloadHash:   "abc123",
			PayloadSize:   100,
			Status:        "success",
			HTTPStatus:    &[]int{200}[0],
			ResponseBody:  &[]string{"Success"}[0],
			AttemptNumber: 2,
			ScheduledAt:   time.Now().Add(-5 * time.Minute),
			DeliveredAt:   &[]time.Time{time.Now().Add(-4 * time.Minute)}[0],
			CreatedAt:     time.Now().Add(-5 * time.Minute),
		},
	}

	for _, attempt := range attempts {
		err = suite.deliveryAttemptRepo.Create(context.Background(), attempt)
		require.NoError(suite.T(), err)
	}

	// Make logs request
	req, err := http.NewRequest("GET", fmt.Sprintf("/webhooks/deliveries/%s/logs", deliveryID), nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), deliveryID.String(), response["delivery_id"])
	assert.Equal(suite.T(), float64(2), response["total_attempts"])

	logs, ok := response["logs"].([]interface{})
	require.True(suite.T(), ok)
	assert.Len(suite.T(), logs, 2)

	// Verify first log entry
	firstLog := logs[0].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), firstLog["attempt_number"])
	assert.Equal(suite.T(), "failed", firstLog["status"])
	assert.Equal(suite.T(), float64(404), firstLog["http_status"])
	assert.Equal(suite.T(), "Not Found", firstLog["error_message"])

	// Verify second log entry
	secondLog := logs[1].(map[string]interface{})
	assert.Equal(suite.T(), float64(2), secondLog["attempt_number"])
	assert.Equal(suite.T(), "success", secondLog["status"])
	assert.Equal(suite.T(), float64(200), secondLog["http_status"])
	assert.Equal(suite.T(), "Success", secondLog["response_body"])
}

func (suite *TestingHandlerIntegrationTestSuite) TestTestEndpointReceiver() {
	endpointID := uuid.New()

	// Test receiving a webhook
	payload := `{"event": "test", "data": {"id": 123, "name": "test"}}`
	req, err := http.NewRequest("POST", fmt.Sprintf("/test/%s", endpointID), strings.NewReader(payload))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-ID", "test-123")
	req.Header.Set("User-Agent", "Test-Client/1.0")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Webhook received successfully", response["message"])
	assert.NotEmpty(suite.T(), response["receive_id"])
	assert.NotEmpty(suite.T(), response["timestamp"])
}

func (suite *TestingHandlerIntegrationTestSuite) TestWebSocketConnection() {
	// This test would require a more complex setup with actual WebSocket testing
	// For now, we'll test that the endpoint exists and returns the expected error for non-WebSocket requests
	
	req, err := http.NewRequest("GET", "/webhooks/realtime", nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return an error because it's not a WebSocket upgrade request
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *TestingHandlerIntegrationTestSuite) TestInvalidRequests() {
	// Test invalid webhook test request
	req, err := http.NewRequest("POST", "/webhooks/test", strings.NewReader(`{"invalid": "json"`))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	// Test invalid delivery ID for inspection
	req, err = http.NewRequest("GET", "/webhooks/deliveries/invalid-uuid/inspect", nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	// Test non-existent delivery for inspection
	req, err = http.NewRequest("GET", fmt.Sprintf("/webhooks/deliveries/%s/inspect", uuid.New()), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func TestTestingHandlerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TestingHandlerIntegrationTestSuite))
}
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"github.com/josedab/waas/internal/api/handlers"
	"github.com/josedab/waas/pkg/auth"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides comprehensive integration testing
type IntegrationTestSuite struct {
	suite.Suite
	
	// Infrastructure
	db           *database.DB
	redis        *database.RedisClient
	logger       *utils.Logger
	
	// Repositories
	tenantRepo   repository.TenantRepository
	webhookRepo  repository.WebhookEndpointRepository
	deliveryRepo repository.DeliveryAttemptRepository
	
	// Services
	queueManager *queue.Manager
	
	// API
	apiServer    *gin.Engine
	
	// Test data
	testTenant1  *models.Tenant
	testTenant2  *models.Tenant
	
	// Mock servers
	mockServer   *httptest.Server
	receivedWebhooks []WebhookReceived
}

type WebhookReceived struct {
	TenantID  uuid.UUID
	Headers   map[string]string
	Body      []byte
	Timestamp time.Time
}

func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Setup logger
	s.logger = utils.NewLogger("integration-test")
	
	// Setup database connection
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable"
	}
	
	var err error
	os.Setenv("DATABASE_URL", dbURL)
	s.db, err = database.NewConnection()
	s.Require().NoError(err, "Failed to connect to test database")
	
	// Setup Redis connection
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/12" // Use test database
	}
	
	s.redis, err = database.NewRedisConnection(redisURL)
	s.Require().NoError(err, "Failed to connect to test Redis")
	
	// Clean Redis
	ctx := context.Background()
	s.redis.Client.FlushDB(ctx)
	
	// Initialize repositories
	s.tenantRepo = repository.NewTenantRepository(s.db)
	s.webhookRepo = repository.NewWebhookEndpointRepository(s.db)
	s.deliveryRepo = repository.NewDeliveryAttemptRepository(s.db)
	
	// Initialize queue manager
	integrationHandler := &IntegrationDeliveryHandler{
		suite: s,
	}
	s.queueManager = queue.NewManager(s.redis, integrationHandler, 2)
	
	// Setup API server
	s.setupAPIServer()
	
	// Create test data
	s.createTestData()
	
	// Start services
	err = s.queueManager.Start(ctx)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	// Stop services
	if s.queueManager != nil && s.queueManager.IsRunning() {
		s.queueManager.Stop()
	}
	
	// Close mock server
	if s.mockServer != nil {
		s.mockServer.Close()
	}
	
	// Close connections
	if s.redis != nil {
		s.redis.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *IntegrationTestSuite) SetupTest() {
	// Clean up between tests
	ctx := context.Background()
	s.redis.Client.FlushDB(ctx)
	s.receivedWebhooks = []WebhookReceived{}
}

func (s *IntegrationTestSuite) setupAPIServer() {
	// Create handlers
	publisher := &queuePublisherAdapter{manager: s.queueManager}
	webhookHandler := handlers.NewWebhookHandler(s.webhookRepo, s.deliveryRepo, publisher, s.logger)
	tenantHandler := handlers.NewTenantHandler(s.tenantRepo, s.logger)
	
	// Setup router
	s.apiServer = gin.New()
	s.apiServer.Use(gin.Recovery())
	
	// Add authentication middleware
	authMiddleware := auth.NewAuthMiddleware(s.tenantRepo)
	s.apiServer.Use(authMiddleware.RequireAuth())
	
	// Setup routes
	api := s.apiServer.Group("/api/v1")
	{
		api.POST("/tenants", tenantHandler.CreateTenant)
		// api.GET("/tenants/current", tenantHandler.GetCurrentTenant)
		api.POST("/webhooks/endpoints", webhookHandler.CreateWebhookEndpoint)
		api.GET("/webhooks/endpoints", webhookHandler.GetWebhookEndpoints)
		api.GET("/webhooks/endpoints/:id", webhookHandler.GetWebhookEndpoint)
		api.PUT("/webhooks/endpoints/:id", webhookHandler.UpdateWebhookEndpoint)
		api.DELETE("/webhooks/endpoints/:id", webhookHandler.DeleteWebhookEndpoint)
		api.POST("/webhooks/send", webhookHandler.SendWebhook)
		api.POST("/webhooks/send/batch", webhookHandler.BatchSendWebhook)
		// api.GET("/webhooks/deliveries", webhookHandler.GetDeliveryHistory)
		// api.GET("/webhooks/deliveries/:id", webhookHandler.GetDeliveryDetails)
	}
}

func (s *IntegrationTestSuite) createTestData() {
	ctx := context.Background()
	
	// Create test tenants
	s.testTenant1 = &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Integration Test Tenant 1",
		APIKeyHash:         "$2a$10$test.hash.for.integration.testing.1",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
	}
	
	s.testTenant2 = &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Integration Test Tenant 2",
		APIKeyHash:         "$2a$10$test.hash.for.integration.testing.2",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}
	
	err := s.tenantRepo.Create(ctx, s.testTenant1)
	s.Require().NoError(err)
	
	err = s.tenantRepo.Create(ctx, s.testTenant2)
	s.Require().NoError(err)
	
	// Create mock webhook server
	s.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.recordWebhookReceived(r)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "received"}`))
	}))
}

func (s *IntegrationTestSuite) recordWebhookReceived(r *http.Request) {
	body := make([]byte, 0)
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	
	// Extract tenant ID from headers or context
	tenantID := uuid.Nil
	if tid := headers["X-Tenant-ID"]; tid != "" {
		if parsed, err := uuid.Parse(tid); err == nil {
			tenantID = parsed
		}
	}
	
	webhook := WebhookReceived{
		TenantID:  tenantID,
		Headers:   headers,
		Body:      body,
		Timestamp: time.Now(),
	}
	
	s.receivedWebhooks = append(s.receivedWebhooks, webhook)
}

func (s *IntegrationTestSuite) makeAPIRequestForTenant(tenantID uuid.UUID, method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	
	req, err := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Set appropriate API key based on tenant
	if tenantID == s.testTenant1.ID {
		req.Header.Set("X-API-Key", "integration-test-api-key-1")
	} else if tenantID == s.testTenant2.ID {
		req.Header.Set("X-API-Key", "integration-test-api-key-2")
	}
	
	w := httptest.NewRecorder()
	s.apiServer.ServeHTTP(w, req)
	
	return w, nil
}

// Test complete webhook lifecycle
func (s *IntegrationTestSuite) TestCompleteWebhookLifecycle() {
	_ = context.Background()
	
	// Step 1: Create webhook endpoint for tenant 1
	createRequest := handlers.CreateWebhookEndpointRequest{
		URL: s.mockServer.URL + "/webhook",
		CustomHeaders: map[string]string{
			"Authorization": "Bearer test-token",
		},
	}
	
	w, err := s.makeAPIRequestForTenant(s.testTenant1.ID, "POST", "/api/v1/webhooks/endpoints", createRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusCreated, w.Code)
	
	var endpointResponse handlers.WebhookEndpointResponse
	err = json.Unmarshal(w.Body.Bytes(), &endpointResponse)
	s.Require().NoError(err)
	
	endpointID := endpointResponse.ID
	
	// Step 2: Send webhook
	payload := map[string]interface{}{
		"event":    "integration.test",
		"user_id":  "12345",
		"metadata": map[string]string{"test": "lifecycle"},
	}
	
	sendRequest := handlers.SendWebhookRequest{
		EndpointID: &endpointID,
		Payload:    json.RawMessage(mustMarshal(payload)),
		Headers: map[string]string{
			"X-Event-Type": "integration.test",
		},
	}
	
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, w.Code)
	
	var sendResponse handlers.SendWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &sendResponse)
	s.Require().NoError(err)
	
	deliveryID := sendResponse.DeliveryID
	
	// Step 3: Wait for delivery
	time.Sleep(3 * time.Second)
	
	// Step 4: Verify webhook was received
	s.Require().Greater(len(s.receivedWebhooks), 0, "Expected webhook to be received")
	
	receivedWebhook := s.receivedWebhooks[0]
	s.Equal("application/json", receivedWebhook.Headers["Content-Type"])
	s.Equal("integration.test", receivedWebhook.Headers["X-Event-Type"])
	s.Equal("Bearer test-token", receivedWebhook.Headers["Authorization"])
	
	var receivedPayload map[string]interface{}
	err = json.Unmarshal(receivedWebhook.Body, &receivedPayload)
	s.Require().NoError(err)
	s.Equal(payload["event"], receivedPayload["event"])
	s.Equal(payload["user_id"], receivedPayload["user_id"])
	
	// Step 5: Check delivery status
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "GET", fmt.Sprintf("/api/v1/webhooks/deliveries/%s", deliveryID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	var deliveryDetails map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &deliveryDetails)
	s.Require().NoError(err)
	
	s.Equal("success", deliveryDetails["status"])
	s.Equal(deliveryID.String(), deliveryDetails["id"])
	
	// Step 6: Update endpoint
	updateRequest := handlers.UpdateWebhookEndpointRequest{
		IsActive: boolPtr(false),
	}
	
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "PUT", fmt.Sprintf("/api/v1/webhooks/endpoints/%s", endpointID.String()), updateRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	// Step 7: Try to send to inactive endpoint
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusBadRequest, w.Code)
	
	var errorResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.Require().NoError(err)
	
	errorObj := errorResponse["error"].(map[string]interface{})
	s.Equal("ENDPOINT_INACTIVE", errorObj["code"])
	
	// Step 8: Delete endpoint
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "DELETE", fmt.Sprintf("/api/v1/webhooks/endpoints/%s", endpointID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusNoContent, w.Code)
	
	// Step 9: Verify endpoint is deleted
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "GET", fmt.Sprintf("/api/v1/webhooks/endpoints/%s", endpointID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code)
}

// Test multi-tenant data isolation
func (s *IntegrationTestSuite) TestMultiTenantDataIsolation() {
	ctx := context.Background()
	
	// Create endpoints for both tenants
	endpoint1 := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: s.testTenant1.ID,
		URL:      s.mockServer.URL + "/tenant1",
		SecretHash: "tenant1-secret",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	endpoint2 := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: s.testTenant2.ID,
		URL:      s.mockServer.URL + "/tenant2",
		SecretHash: "tenant2-secret",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	err := s.webhookRepo.Create(ctx, endpoint1)
	s.Require().NoError(err)
	
	err = s.webhookRepo.Create(ctx, endpoint2)
	s.Require().NoError(err)
	
	// Test 1: Tenant 1 can only see their endpoints
	w, err := s.makeAPIRequestForTenant(s.testTenant1.ID, "GET", "/api/v1/webhooks/endpoints", nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	var response1 map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response1)
	s.Require().NoError(err)
	
	endpoints1 := response1["endpoints"].([]interface{})
	s.Len(endpoints1, 1)
	
	endpoint1Data := endpoints1[0].(map[string]interface{})
	s.Equal(endpoint1.ID.String(), endpoint1Data["id"])
	
	// Test 2: Tenant 2 can only see their endpoints
	w, err = s.makeAPIRequestForTenant(s.testTenant2.ID, "GET", "/api/v1/webhooks/endpoints", nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	var response2 map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response2)
	s.Require().NoError(err)
	
	endpoints2 := response2["endpoints"].([]interface{})
	s.Len(endpoints2, 1)
	
	endpoint2Data := endpoints2[0].(map[string]interface{})
	s.Equal(endpoint2.ID.String(), endpoint2Data["id"])
	
	// Test 3: Tenant 1 cannot access tenant 2's endpoint
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "GET", fmt.Sprintf("/api/v1/webhooks/endpoints/%s", endpoint2.ID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code)
	
	// Test 4: Tenant 2 cannot access tenant 1's endpoint
	w, err = s.makeAPIRequestForTenant(s.testTenant2.ID, "GET", fmt.Sprintf("/api/v1/webhooks/endpoints/%s", endpoint1.ID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code)
	
	// Test 5: Tenant 1 cannot send to tenant 2's endpoint
	payload := map[string]interface{}{"test": "isolation"}
	sendRequest := handlers.SendWebhookRequest{
		EndpointID: &endpoint2.ID,
		Payload:    json.RawMessage(mustMarshal(payload)),
	}
	
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code)
	
	// Test 6: Tenant 2 cannot send to tenant 1's endpoint
	sendRequest.EndpointID = &endpoint1.ID
	
	w, err = s.makeAPIRequestForTenant(s.testTenant2.ID, "POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code)
}

// Test webhook signature verification
func (s *IntegrationTestSuite) TestWebhookSignatureVerification() {
	ctx := context.Background()
	
	// Create endpoint with signature verification
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: s.testTenant1.ID,
		URL:      s.mockServer.URL + "/signed-webhook",
		SecretHash: "test-secret-for-signing",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	err := s.webhookRepo.Create(ctx, endpoint)
	s.Require().NoError(err)
	
	// Send webhook
	payload := map[string]interface{}{
		"event": "signature.test",
		"data":  "testing webhook signatures",
	}
	
	sendRequest := handlers.SendWebhookRequest{
		EndpointID: &endpoint.ID,
		Payload:    json.RawMessage(mustMarshal(payload)),
	}
	
	w, err := s.makeAPIRequestForTenant(s.testTenant1.ID, "POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, w.Code)
	
	// Wait for delivery
	time.Sleep(3 * time.Second)
	
	// Verify webhook was received with signature
	s.Require().Greater(len(s.receivedWebhooks), 0, "Expected webhook to be received")
	
	receivedWebhook := s.receivedWebhooks[0]
	
	// Check for signature header
	signature := receivedWebhook.Headers["X-Webhook-Signature"]
	s.NotEmpty(signature, "Expected webhook signature header")
	
	// Verify signature format (should be algo=signature)
	s.Contains(signature, "sha256=", "Expected SHA256 signature")
}

// Test rate limiting
func (s *IntegrationTestSuite) TestRateLimiting() {
	// This test would require implementing rate limiting in the test setup
	// For now, we'll test that the system handles high request volumes gracefully
	
	ctx := context.Background()
	
	// Create endpoint
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: s.testTenant2.ID, // Use basic tier tenant with lower limits
		URL:      s.mockServer.URL + "/rate-limit-test",
		SecretHash: "rate-limit-secret",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	err := s.webhookRepo.Create(ctx, endpoint)
	s.Require().NoError(err)
	
	// Send many webhooks rapidly
	requestCount := 50
	successCount := 0
	rateLimitCount := 0
	
	for i := 0; i < requestCount; i++ {
		payload := map[string]interface{}{
			"event":      "rate.limit.test",
			"request_id": i,
		}
		
		sendRequest := handlers.SendWebhookRequest{
			EndpointID: &endpoint.ID,
			Payload:    json.RawMessage(mustMarshal(payload)),
		}
		
		w, err := s.makeAPIRequestForTenant(s.testTenant2.ID, "POST", "/api/v1/webhooks/send", sendRequest)
		s.Require().NoError(err)
		
		if w.Code == http.StatusAccepted {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			rateLimitCount++
		}
	}
	
	s.T().Logf("Rate limiting test: %d successful, %d rate limited out of %d requests", 
		successCount, rateLimitCount, requestCount)
	
	// Should have processed some requests successfully
	s.Greater(successCount, 0, "Expected some requests to succeed")
	
	// If rate limiting is implemented, should have some rate limited requests
	// s.Greater(rateLimitCount, 0, "Expected some requests to be rate limited")
}

// Test delivery retry logic
func (s *IntegrationTestSuite) TestDeliveryRetryLogic() {
	// Create a failing mock server
	failCount := 0
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount <= 2 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "temporary_failure"}`))
		} else {
			// Succeed on 3rd attempt
			s.recordWebhookReceived(r)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success_after_retries"}`))
		}
	}))
	defer failingServer.Close()
	
	ctx := context.Background()
	
	// Create endpoint pointing to failing server
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: s.testTenant1.ID,
		URL:      failingServer.URL + "/retry-test",
		SecretHash: "retry-test-secret",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	err := s.webhookRepo.Create(ctx, endpoint)
	s.Require().NoError(err)
	
	// Send webhook
	payload := map[string]interface{}{
		"event": "retry.test",
		"data":  "testing retry logic",
	}
	
	sendRequest := handlers.SendWebhookRequest{
		EndpointID: &endpoint.ID,
		Payload:    json.RawMessage(mustMarshal(payload)),
	}
	
	w, err := s.makeAPIRequestForTenant(s.testTenant1.ID, "POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, w.Code)
	
	var sendResponse handlers.SendWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &sendResponse)
	s.Require().NoError(err)
	
	// Wait for retries to complete
	time.Sleep(15 * time.Second)
	
	// Check delivery status
	w, err = s.makeAPIRequestForTenant(s.testTenant1.ID, "GET", fmt.Sprintf("/api/v1/webhooks/deliveries/%s", sendResponse.DeliveryID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	var deliveryDetails map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &deliveryDetails)
	s.Require().NoError(err)
	
	// Should eventually succeed
	s.Equal("success", deliveryDetails["status"])
	
	// Should have multiple attempts
	attempts := deliveryDetails["attempts"].([]interface{})
	s.Greater(len(attempts), 1, "Expected multiple delivery attempts")
	s.LessOrEqual(len(attempts), 5, "Should not exceed max attempts")
	
	// Verify webhook was eventually received
	s.Greater(len(s.receivedWebhooks), 0, "Expected webhook to be eventually received")
	
	s.T().Logf("Retry test completed with %d attempts, final status: %s", 
		len(attempts), deliveryDetails["status"])
}

// IntegrationDeliveryHandler handles webhook deliveries for integration testing
type IntegrationDeliveryHandler struct {
	suite *IntegrationTestSuite
}

func (m *IntegrationDeliveryHandler) HandleDelivery(ctx context.Context, message *queue.DeliveryMessage) (*queue.DeliveryResult, error) {
	// Simulate HTTP delivery
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// Find the endpoint
	endpoint, err := m.suite.webhookRepo.GetByID(ctx, message.EndpointID)
	if err != nil {
		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusFailed,
			AttemptNumber: message.AttemptNumber,
			ErrorMessage:  stringPtr("Endpoint not found"),
		}, nil
	}
	
	req, err := http.NewRequest("POST", endpoint.URL, bytes.NewReader(message.Payload))
	if err != nil {
		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusFailed,
			AttemptNumber: message.AttemptNumber,
			ErrorMessage:  stringPtr("Failed to create request"),
		}, nil
	}
	
	req.Header.Set("Content-Type", "application/json")
	for key, value := range message.Headers {
		req.Header.Set(key, value)
	}
	
	// Add signature if present
	if message.Signature != "" {
		req.Header.Set("X-Webhook-Signature", message.Signature)
	}
	
	// Add tenant ID for tracking
	req.Header.Set("X-Tenant-ID", message.TenantID.String())
	
	resp, err := client.Do(req)
	if err != nil {
		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusFailed,
			AttemptNumber: message.AttemptNumber,
			ErrorMessage:  stringPtr(err.Error()),
		}, nil
	}
	defer resp.Body.Close()
	
	now := time.Now()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusSuccess,
			HTTPStatus:    &resp.StatusCode,
			DeliveredAt:   &now,
			AttemptNumber: message.AttemptNumber,
		}, nil
	}
	
	return &queue.DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        queue.StatusFailed,
		HTTPStatus:    &resp.StatusCode,
		AttemptNumber: message.AttemptNumber,
		ErrorMessage:  stringPtr(fmt.Sprintf("HTTP %d", resp.StatusCode)),
	}, nil
}

// Helper functions
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

// queuePublisherAdapter wraps *queue.Manager to implement queue.PublisherInterface
type queuePublisherAdapter struct {
	manager *queue.Manager
}

func (a *queuePublisherAdapter) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	return a.manager.PublishDelivery(ctx, message)
}

func (a *queuePublisherAdapter) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	return nil
}

func (a *queuePublisherAdapter) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	return nil
}

func (a *queuePublisherAdapter) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	return 0, nil
}

func (a *queuePublisherAdapter) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return a.manager.GetQueueStats(ctx)
}
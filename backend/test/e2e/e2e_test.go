package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"webhook-platform/internal/api/handlers"
	"webhook-platform/internal/delivery"
	"webhook-platform/pkg/auth"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/queue"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite provides end-to-end integration testing for the webhook platform
type E2ETestSuite struct {
	suite.Suite
	
	// Infrastructure
	db           *database.DB
	redis        *database.RedisClient
	logger       *utils.Logger
	
	// Repositories
	tenantRepo         repository.TenantRepository
	webhookRepo        repository.WebhookEndpointRepository
	deliveryRepo       repository.DeliveryAttemptRepository
	
	// Services
	queueManager       *queue.Manager
	deliveryEngine     *delivery.DeliveryEngine
	
	// API
	apiServer          *gin.Engine
	
	// Test data
	testTenant         *models.Tenant
	testEndpoints      []*models.WebhookEndpoint
	
	// Mock webhook servers
	webhookServers     []*httptest.Server
	webhookServerMutex sync.RWMutex
	receivedWebhooks   []WebhookReceived
}

type WebhookReceived struct {
	URL       string
	Headers   map[string]string
	Body      []byte
	Timestamp time.Time
}

func TestE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}
	
	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Setup logger
	s.logger = utils.NewLogger("e2e-test")
	
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
		redisURL = "redis://localhost:6379/15" // Use test database
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
	mockHandler := &MockDeliveryHandler{
		suite: s,
	}
	s.queueManager = queue.NewManager(s.redis, mockHandler, 2)
	
	// Initialize delivery engine
	engine, err := delivery.NewEngine()
	if err != nil {
		panic(err)
	}
	s.deliveryEngine = engine
	
	// Setup API server
	s.setupAPIServer()
	
	// Create test data
	s.createTestData()
}

func (s *E2ETestSuite) TearDownSuite() {
	// Stop services
	if s.queueManager != nil && s.queueManager.IsRunning() {
		s.queueManager.Stop()
	}
	
	// Close webhook servers
	s.webhookServerMutex.Lock()
	for _, server := range s.webhookServers {
		server.Close()
	}
	s.webhookServerMutex.Unlock()
	
	// Close connections
	if s.redis != nil {
		s.redis.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *E2ETestSuite) SetupTest() {
	// Clean up between tests
	ctx := context.Background()
	s.redis.Client.FlushDB(ctx)
	s.receivedWebhooks = []WebhookReceived{}
}

func (s *E2ETestSuite) setupAPIServer() {
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
		// Tenant routes
		api.POST("/tenants", tenantHandler.CreateTenant)
		// api.GET("/tenants/current", tenantHandler.GetCurrentTenant) // Method does not exist
		
		// Webhook endpoint routes
		api.POST("/webhooks/endpoints", webhookHandler.CreateWebhookEndpoint)
		api.GET("/webhooks/endpoints", webhookHandler.GetWebhookEndpoints)
		api.GET("/webhooks/endpoints/:id", webhookHandler.GetWebhookEndpoint)
		api.PUT("/webhooks/endpoints/:id", webhookHandler.UpdateWebhookEndpoint)
		api.DELETE("/webhooks/endpoints/:id", webhookHandler.DeleteWebhookEndpoint)
		
		// Webhook sending routes
		api.POST("/webhooks/send", webhookHandler.SendWebhook)
		api.POST("/webhooks/send/batch", webhookHandler.BatchSendWebhook)
		
		// Delivery monitoring routes
		// api.GET("/webhooks/deliveries", webhookHandler.GetDeliveryHistory) // Method does not exist
		// api.GET("/webhooks/deliveries/:id", webhookHandler.GetDeliveryDetails) // Method does not exist
	}
}

func (s *E2ETestSuite) createTestData() {
	ctx := context.Background()
	
	// Create test tenant
	s.testTenant = &models.Tenant{
		ID:                 uuid.New(),
		Name:               "E2E Test Tenant",
		APIKeyHash:         "$2a$10$test.hash.for.e2e.testing",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 1000,
		MonthlyQuota:       100000,
	}
	
	err := s.tenantRepo.Create(ctx, s.testTenant)
	s.Require().NoError(err)
	
	// Create mock webhook servers
	s.createMockWebhookServers()
	
	// Create test webhook endpoints
	s.createTestWebhookEndpoints()
}

func (s *E2ETestSuite) createMockWebhookServers() {
	// Create successful webhook server
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.recordWebhookReceived(r)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "received"}`))
	}))
	
	// Create slow webhook server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.recordWebhookReceived(r)
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "slow_received"}`))
	}))
	
	// Create failing webhook server
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.recordWebhookReceived(r)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal_error"}`))
	}))
	
	// Create intermittent webhook server
	intermittentCount := 0
	intermittentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.recordWebhookReceived(r)
		intermittentCount++
		if intermittentCount%3 == 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success_after_retries"}`))
		} else {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error": "temporary_failure"}`))
		}
	}))
	
	s.webhookServerMutex.Lock()
	s.webhookServers = []*httptest.Server{successServer, slowServer, failServer, intermittentServer}
	s.webhookServerMutex.Unlock()
}

func (s *E2ETestSuite) createTestWebhookEndpoints() {
	ctx := context.Background()
	
	s.webhookServerMutex.RLock()
	defer s.webhookServerMutex.RUnlock()
	
	endpoints := []*models.WebhookEndpoint{
		{
			ID:       uuid.New(),
			TenantID: s.testTenant.ID,
			URL:      s.webhookServers[0].URL + "/webhook", // Success server
			SecretHash: "success-secret-hash",
			IsActive: true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       3,
				InitialDelayMs:    100,
				MaxDelayMs:        5000,
				BackoffMultiplier: 2,
			},
		},
		{
			ID:       uuid.New(),
			TenantID: s.testTenant.ID,
			URL:      s.webhookServers[1].URL + "/webhook", // Slow server
			SecretHash: "slow-secret-hash",
			IsActive: true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       2,
				InitialDelayMs:    50,
				MaxDelayMs:        1000,
				BackoffMultiplier: 2,
			},
		},
		{
			ID:       uuid.New(),
			TenantID: s.testTenant.ID,
			URL:      s.webhookServers[2].URL + "/webhook", // Fail server
			SecretHash: "fail-secret-hash",
			IsActive: true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       3,
				InitialDelayMs:    100,
				MaxDelayMs:        2000,
				BackoffMultiplier: 2,
			},
		},
		{
			ID:       uuid.New(),
			TenantID: s.testTenant.ID,
			URL:      s.webhookServers[3].URL + "/webhook", // Intermittent server
			SecretHash: "intermittent-secret-hash",
			IsActive: true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    100,
				MaxDelayMs:        3000,
				BackoffMultiplier: 2,
			},
		},
	}
	
	for _, endpoint := range endpoints {
		err := s.webhookRepo.Create(ctx, endpoint)
		s.Require().NoError(err)
	}
	
	s.testEndpoints = endpoints
}

func (s *E2ETestSuite) recordWebhookReceived(r *http.Request) {
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
	
	webhook := WebhookReceived{
		URL:       r.URL.String(),
		Headers:   headers,
		Body:      body,
		Timestamp: time.Now(),
	}
	
	s.webhookServerMutex.Lock()
	s.receivedWebhooks = append(s.receivedWebhooks, webhook)
	s.webhookServerMutex.Unlock()
}

func (s *E2ETestSuite) makeAPIRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
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
	req.Header.Set("X-API-Key", "test-api-key") // Mock API key
	
	w := httptest.NewRecorder()
	s.apiServer.ServeHTTP(w, req)
	
	return w, nil
}

// Test complete webhook delivery workflow
func (s *E2ETestSuite) TestCompleteWebhookDeliveryWorkflow() {
	ctx := context.Background()
	
	// Start queue manager
	err := s.queueManager.Start(ctx)
	s.Require().NoError(err)
	defer s.queueManager.Stop()
	
	// Test payload
	payload := map[string]interface{}{
		"event":    "user.created",
		"user_id":  "12345",
		"email":    "test@example.com",
		"metadata": map[string]string{"source": "e2e_test"},
	}
	
	// Send webhook to successful endpoint
	sendRequest := handlers.SendWebhookRequest{
		EndpointID: &s.testEndpoints[0].ID,
		Payload:    json.RawMessage(mustMarshal(payload)),
		Headers: map[string]string{
			"X-Event-Type": "user.created",
			"X-Source":     "e2e-test",
		},
	}
	
	w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, w.Code)
	
	var response handlers.SendWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	s.NotEmpty(response.DeliveryID)
	s.Equal(s.testEndpoints[0].ID, response.EndpointID)
	
	// Wait for delivery processing
	time.Sleep(3 * time.Second)
	
	// Verify webhook was received
	s.webhookServerMutex.RLock()
	s.Require().Greater(len(s.receivedWebhooks), 0, "Expected webhook to be received")
	
	receivedWebhook := s.receivedWebhooks[0]
	s.Contains(receivedWebhook.URL, "/webhook")
	s.Equal("application/json", receivedWebhook.Headers["Content-Type"])
	s.Equal("user.created", receivedWebhook.Headers["X-Event-Type"])
	s.Equal("e2e-test", receivedWebhook.Headers["X-Source"])
	
	var receivedPayload map[string]interface{}
	err = json.Unmarshal(receivedWebhook.Body, &receivedPayload)
	s.Require().NoError(err)
	s.Equal(payload["event"], receivedPayload["event"])
	s.Equal(payload["user_id"], receivedPayload["user_id"])
	s.webhookServerMutex.RUnlock()
	
	// Verify delivery history
	w, err = s.makeAPIRequest("GET", "/api/v1/webhooks/deliveries", nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	var historyResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &historyResponse)
	s.Require().NoError(err)
	
	deliveries := historyResponse["deliveries"].([]interface{})
	s.Greater(len(deliveries), 0)
	
	delivery := deliveries[0].(map[string]interface{})
	s.Equal(response.DeliveryID.String(), delivery["id"])
	s.Equal("success", delivery["status"])
}

// Test multi-tenant isolation
func (s *E2ETestSuite) TestMultiTenantIsolation() {
	ctx := context.Background()
	
	// Create second tenant
	tenant2 := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "E2E Test Tenant 2",
		APIKeyHash:         "$2a$10$test.hash.for.e2e.testing.2",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}
	
	err := s.tenantRepo.Create(ctx, tenant2)
	s.Require().NoError(err)
	
	// Create endpoint for second tenant
	endpoint2 := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenant2.ID,
		URL:      s.webhookServers[0].URL + "/webhook2",
		SecretHash: "tenant2-secret-hash",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	err = s.webhookRepo.Create(ctx, endpoint2)
	s.Require().NoError(err)
	
	// Test that tenant 1 cannot access tenant 2's endpoints
	w, err := s.makeAPIRequest("GET", "/api/v1/webhooks/endpoints", nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	
	endpoints := response["endpoints"].([]interface{})
	
	// Should only see tenant 1's endpoints
	for _, ep := range endpoints {
		endpoint := ep.(map[string]interface{})
		endpointID := endpoint["id"].(string)
		
		// Verify this endpoint belongs to tenant 1
		found := false
		for _, testEndpoint := range s.testEndpoints {
			if testEndpoint.ID.String() == endpointID {
				found = true
				break
			}
		}
		s.True(found, "Found endpoint that doesn't belong to current tenant")
	}
	
	// Test that tenant 1 cannot access tenant 2's endpoint directly
	w, err = s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/webhooks/endpoints/%s", endpoint2.ID.String()), nil)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code) // Should not find endpoint from different tenant
	
	// Test that tenant 1 cannot send to tenant 2's endpoint
	sendRequest := handlers.SendWebhookRequest{
		EndpointID: &endpoint2.ID,
		Payload:    json.RawMessage(`{"test": "isolation"}`),
	}
	
	w, err = s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusNotFound, w.Code) // Should not find endpoint from different tenant
}

// Test performance and load scenarios
func (s *E2ETestSuite) TestPerformanceAndLoad() {
	ctx := context.Background()
	
	// Start queue manager
	err := s.queueManager.Start(ctx)
	s.Require().NoError(err)
	defer s.queueManager.Stop()
	
	// Test concurrent webhook sending
	concurrency := 10
	webhooksPerGoroutine := 5
	totalWebhooks := concurrency * webhooksPerGoroutine
	
	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32
	
	startTime := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < webhooksPerGoroutine; j++ {
				payload := map[string]interface{}{
					"event":        "load.test",
					"goroutine_id": goroutineID,
					"webhook_id":   j,
					"timestamp":    time.Now().Unix(),
				}
				
				sendRequest := handlers.SendWebhookRequest{
					EndpointID: &s.testEndpoints[0].ID, // Use successful endpoint
					Payload:    json.RawMessage(mustMarshal(payload)),
					Headers: map[string]string{
						"X-Event-Type":   "load.test",
						"X-Goroutine-ID": fmt.Sprintf("%d", goroutineID),
					},
				}
				
				w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
				if err != nil || w.Code != http.StatusAccepted {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// Verify all requests were processed
	s.Equal(int32(totalWebhooks), successCount+errorCount)
	s.Equal(int32(0), errorCount, "Expected no errors during load test")
	
	// Wait for webhook processing
	time.Sleep(5 * time.Second)
	
	// Verify webhooks were received
	s.webhookServerMutex.RLock()
	receivedCount := len(s.receivedWebhooks)
	s.webhookServerMutex.RUnlock()
	
	s.GreaterOrEqual(receivedCount, totalWebhooks/2, "Expected at least half of webhooks to be received")
	
	// Performance assertions
	s.Less(duration, 30*time.Second, "Load test took too long")
	
	throughput := float64(totalWebhooks) / duration.Seconds()
	s.Greater(throughput, 1.0, "Throughput too low: %.2f webhooks/second", throughput)
	
	s.T().Logf("Load test completed: %d webhooks in %v (%.2f webhooks/second)", 
		totalWebhooks, duration, throughput)
}

// Test chaos engineering scenarios
func (s *E2ETestSuite) TestChaosEngineeringScenarios() {
	ctx := context.Background()
	
	// Start queue manager
	err := s.queueManager.Start(ctx)
	s.Require().NoError(err)
	defer s.queueManager.Stop()
	
	// Test 1: Webhook endpoint failures and retries
	s.Run("WebhookEndpointFailuresAndRetries", func() {
		payload := map[string]interface{}{
			"event": "chaos.test.failure",
			"data":  "testing failure scenarios",
		}
		
		// Send to failing endpoint
		sendRequest := handlers.SendWebhookRequest{
			EndpointID: &s.testEndpoints[2].ID, // Failing server
			Payload:    json.RawMessage(mustMarshal(payload)),
		}
		
		w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
		s.Require().NoError(err)
		s.Equal(http.StatusAccepted, w.Code)
		
		var response handlers.SendWebhookResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		s.Require().NoError(err)
		
		// Wait for retries to complete
		time.Sleep(10 * time.Second)
		
		// Verify delivery attempts were made
		w, err = s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/webhooks/deliveries/%s", response.DeliveryID.String()), nil)
		s.Require().NoError(err)
		s.Equal(http.StatusOK, w.Code)
		
		var deliveryDetails map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deliveryDetails)
		s.Require().NoError(err)
		
		s.Equal("failed", deliveryDetails["status"])
		
		attempts := deliveryDetails["attempts"].([]interface{})
		s.Greater(len(attempts), 1, "Expected multiple delivery attempts")
		s.LessOrEqual(len(attempts), 3, "Expected no more than max attempts")
	})
	
	// Test 2: Intermittent failures (eventual success)
	s.Run("IntermittentFailures", func() {
		payload := map[string]interface{}{
			"event": "chaos.test.intermittent",
			"data":  "testing intermittent failures",
		}
		
		// Send to intermittent endpoint
		sendRequest := handlers.SendWebhookRequest{
			EndpointID: &s.testEndpoints[3].ID, // Intermittent server
			Payload:    json.RawMessage(mustMarshal(payload)),
		}
		
		w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
		s.Require().NoError(err)
		s.Equal(http.StatusAccepted, w.Code)
		
		var response handlers.SendWebhookResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		s.Require().NoError(err)
		
		// Wait for retries and eventual success
		time.Sleep(15 * time.Second)
		
		// Verify eventual success
		w, err = s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/webhooks/deliveries/%s", response.DeliveryID.String()), nil)
		s.Require().NoError(err)
		s.Equal(http.StatusOK, w.Code)
		
		var deliveryDetails map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deliveryDetails)
		s.Require().NoError(err)
		
		// Should eventually succeed after retries
		s.Equal("success", deliveryDetails["status"])
		
		attempts := deliveryDetails["attempts"].([]interface{})
		s.Greater(len(attempts), 1, "Expected multiple attempts before success")
	})
	
	// Test 3: Queue overflow simulation
	s.Run("QueueOverflowSimulation", func() {
		// Send many webhooks rapidly to test queue handling
		overflowCount := 100
		
		for i := 0; i < overflowCount; i++ {
			payload := map[string]interface{}{
				"event":      "chaos.test.overflow",
				"message_id": i,
			}
			
			sendRequest := handlers.SendWebhookRequest{
				EndpointID: &s.testEndpoints[0].ID,
				Payload:    json.RawMessage(mustMarshal(payload)),
			}
			
			w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
			s.Require().NoError(err)
			s.Equal(http.StatusAccepted, w.Code)
		}
		
		// Check queue stats
		stats, err := s.queueManager.GetQueueStats(ctx)
		s.Require().NoError(err)
		
		totalQueued := stats[queue.DeliveryQueue] + stats[queue.RetryQueue] + stats[queue.ProcessingQueue]
		s.Greater(totalQueued, int64(0), "Expected messages in queues")
		
		// Wait for processing
		time.Sleep(10 * time.Second)
		
		// Verify queue is being processed
		newStats, err := s.queueManager.GetQueueStats(ctx)
		s.Require().NoError(err)
		
		newTotalQueued := newStats[queue.DeliveryQueue] + newStats[queue.RetryQueue] + newStats[queue.ProcessingQueue]
		s.LessOrEqual(newTotalQueued, totalQueued, "Expected queue to be processed")
	})
	
	// Test 4: Database connection issues simulation
	s.Run("DatabaseConnectionIssues", func() {
		// This test would require more sophisticated database connection mocking
		// For now, we'll test that the system handles repository errors gracefully
		
		// Try to create an endpoint with invalid data to trigger database error
		invalidEndpoint := handlers.CreateWebhookEndpointRequest{
			URL: "invalid-url-format",
		}
		
		w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/endpoints", invalidEndpoint)
		s.Require().NoError(err)
		s.Equal(http.StatusBadRequest, w.Code)
		
		var errorResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
		s.Require().NoError(err)
		
		errorObj := errorResponse["error"].(map[string]interface{})
		s.Equal("INVALID_URL", errorObj["code"])
	})
}

// Test batch webhook sending
func (s *E2ETestSuite) TestBatchWebhookSending() {
	ctx := context.Background()
	
	// Start queue manager
	err := s.queueManager.Start(ctx)
	s.Require().NoError(err)
	defer s.queueManager.Stop()
	
	payload := map[string]interface{}{
		"event": "batch.test",
		"data":  "testing batch webhook sending",
	}
	
	// Send to multiple endpoints
	endpointIDs := []uuid.UUID{
		s.testEndpoints[0].ID, // Success
		s.testEndpoints[1].ID, // Slow
		s.testEndpoints[2].ID, // Fail
	}
	
	batchRequest := handlers.BatchSendWebhookRequest{
		EndpointIDs: endpointIDs,
		Payload:     json.RawMessage(mustMarshal(payload)),
		Headers: map[string]string{
			"X-Event-Type": "batch.test",
		},
	}
	
	w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send/batch", batchRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, w.Code)
	
	var response handlers.BatchSendWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	
	s.Equal(3, response.Total)
	s.Equal(3, response.Queued)
	s.Equal(0, response.Failed)
	s.Len(response.Deliveries, 3)
	
	// Wait for processing
	time.Sleep(8 * time.Second)
	
	// Verify deliveries
	for _, delivery := range response.Deliveries {
		w, err := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/webhooks/deliveries/%s", delivery.DeliveryID.String()), nil)
		s.Require().NoError(err)
		s.Equal(http.StatusOK, w.Code)
		
		var deliveryDetails map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deliveryDetails)
		s.Require().NoError(err)
		
		// Status depends on which endpoint it was sent to
		status := deliveryDetails["status"].(string)
		s.Contains([]string{"success", "failed"}, status)
	}
}

// queuePublisherAdapter wraps queue.Manager to implement queue.PublisherInterface
type queuePublisherAdapter struct {
	manager *queue.Manager
}

func (a *queuePublisherAdapter) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	return a.manager.PublishDelivery(ctx, message)
}

func (a *queuePublisherAdapter) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	return a.manager.PublishDelivery(ctx, message)
}

func (a *queuePublisherAdapter) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	return nil
}

func (a *queuePublisherAdapter) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	stats, err := a.manager.GetQueueStats(ctx)
	if err != nil {
		return 0, err
	}
	return stats[queueName], nil
}

func (a *queuePublisherAdapter) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return a.manager.GetQueueStats(ctx)
}

// MockDeliveryHandler handles webhook deliveries for testing
type MockDeliveryHandler struct {
	suite *E2ETestSuite
}

func (m *MockDeliveryHandler) HandleDelivery(ctx context.Context, message *queue.DeliveryMessage) (*queue.DeliveryResult, error) {
	// This would normally be handled by the delivery engine
	// For E2E tests, we'll simulate the HTTP delivery
	
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
	
	// Simulate HTTP request
	client := &http.Client{
		Timeout: 10 * time.Second,
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
	
	// Add headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range message.Headers {
		req.Header.Set(key, value)
	}
	
	// Add signature if present
	if message.Signature != "" {
		req.Header.Set("X-Webhook-Signature", message.Signature)
	}
	
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

func stringPtr(s string) *string {
	return &s
}
package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"webhook-platform/internal/api/handlers"
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

// LoadTestSuite provides performance and load testing for the webhook platform
type LoadTestSuite struct {
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
	testTenant   *models.Tenant
	testEndpoint *models.WebhookEndpoint
	
	// Mock webhook server
	webhookServer *httptest.Server
	requestCount  int64
}

func TestLoadTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}
	
	// Only run if explicitly requested
	if os.Getenv("RUN_LOAD_TESTS") != "true" {
		t.Skip("Load tests disabled. Set RUN_LOAD_TESTS=true to enable")
	}
	
	suite.Run(t, new(LoadTestSuite))
}

func (s *LoadTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Setup logger
	s.logger = utils.NewLogger("load-test")
	
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
		redisURL = "redis://localhost:6379/14" // Use test database
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
	mockHandler := &LoadTestDeliveryHandler{
		suite: s,
	}
	s.queueManager = queue.NewManager(s.redis, mockHandler, 4) // More workers for load testing
	
	// Setup API server
	s.setupAPIServer()
	
	// Create test data
	s.createTestData()
	
	// Start services
	err = s.queueManager.Start(ctx)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) TearDownSuite() {
	// Stop services
	if s.queueManager != nil && s.queueManager.IsRunning() {
		s.queueManager.Stop()
	}
	
	// Close webhook server
	if s.webhookServer != nil {
		s.webhookServer.Close()
	}
	
	// Close connections
	if s.redis != nil {
		s.redis.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *LoadTestSuite) SetupTest() {
	// Reset counters
	atomic.StoreInt64(&s.requestCount, 0)
	
	// Clean up between tests
	ctx := context.Background()
	s.redis.Client.FlushDB(ctx)
}

func (s *LoadTestSuite) setupAPIServer() {
	// Create handlers
	webhookHandler := handlers.NewWebhookHandler(s.webhookRepo, s.deliveryRepo, &queuePublisherAdapter{manager: s.queueManager}, s.logger)
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
		api.POST("/webhooks/endpoints", webhookHandler.CreateWebhookEndpoint)
		api.POST("/webhooks/send", webhookHandler.SendWebhook)
		api.POST("/webhooks/send/batch", webhookHandler.BatchSendWebhook)
		// api.GET("/webhooks/deliveries", webhookHandler.GetDeliveryHistory)
	}
}

func (s *LoadTestSuite) createTestData() {
	ctx := context.Background()
	
	// Create test tenant
	s.testTenant = &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Load Test Tenant",
		APIKeyHash:         "$2a$10$test.hash.for.load.testing",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 10000, // High limit for load testing
		MonthlyQuota:       1000000,
	}
	
	err := s.tenantRepo.Create(ctx, s.testTenant)
	s.Require().NoError(err)
	
	// Create mock webhook server
	s.webhookServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.requestCount, 1)
		
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "received"}`))
	}))
	
	// Create test webhook endpoint
	s.testEndpoint = &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: s.testTenant.ID,
		URL:      s.webhookServer.URL + "/webhook",
		SecretHash: "load-test-secret-hash",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    100,
			MaxDelayMs:        5000,
			BackoffMultiplier: 2,
		},
	}
	
	err = s.webhookRepo.Create(ctx, s.testEndpoint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) makeAPIRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
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
	req.Header.Set("X-API-Key", "load-test-api-key")
	
	w := httptest.NewRecorder()
	s.apiServer.ServeHTTP(w, req)
	
	return w, nil
}

// Test high-throughput webhook sending
func (s *LoadTestSuite) TestHighThroughputWebhookSending() {
	testCases := []struct {
		name         string
		concurrency  int
		webhooksEach int
		maxDuration  time.Duration
		minThroughput float64
	}{
		{
			name:         "Low Load",
			concurrency:  5,
			webhooksEach: 20,
			maxDuration:  30 * time.Second,
			minThroughput: 10.0,
		},
		{
			name:         "Medium Load",
			concurrency:  10,
			webhooksEach: 50,
			maxDuration:  60 * time.Second,
			minThroughput: 20.0,
		},
		{
			name:         "High Load",
			concurrency:  20,
			webhooksEach: 100,
			maxDuration:  120 * time.Second,
			minThroughput: 30.0,
		},
	}
	
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.runThroughputTest(tc.concurrency, tc.webhooksEach, tc.maxDuration, tc.minThroughput)
		})
	}
}

func (s *LoadTestSuite) runThroughputTest(concurrency, webhooksEach int, maxDuration time.Duration, minThroughput float64) {
	totalWebhooks := concurrency * webhooksEach
	
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	
	startTime := time.Now()
	
	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < webhooksEach; j++ {
				payload := map[string]interface{}{
					"event":        "load.test.throughput",
					"goroutine_id": goroutineID,
					"webhook_id":   j,
					"timestamp":    time.Now().Unix(),
				}
				
				sendRequest := handlers.SendWebhookRequest{
					EndpointID: &s.testEndpoint.ID,
					Payload:    json.RawMessage(mustMarshal(payload)),
					Headers: map[string]string{
						"X-Event-Type":   "load.test.throughput",
						"X-Goroutine-ID": fmt.Sprintf("%d", goroutineID),
					},
				}
				
				w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
				if err != nil || w.Code != http.StatusAccepted {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// Verify results
	s.Equal(int64(totalWebhooks), successCount+errorCount)
	s.Equal(int64(0), errorCount, "Expected no errors during throughput test")
	s.Less(duration, maxDuration, "Test took too long")
	
	throughput := float64(totalWebhooks) / duration.Seconds()
	s.Greater(throughput, minThroughput, "Throughput too low: %.2f webhooks/second", throughput)
	
	s.T().Logf("%s completed: %d webhooks in %v (%.2f webhooks/second)", 
		s.T().Name(), totalWebhooks, duration, throughput)
	
	// Wait for webhook processing and verify delivery
	time.Sleep(10 * time.Second)
	
	receivedCount := atomic.LoadInt64(&s.requestCount)
	deliveryRate := float64(receivedCount) / float64(totalWebhooks) * 100
	
	s.Greater(deliveryRate, 80.0, "Delivery rate too low: %.2f%%", deliveryRate)
	s.T().Logf("Delivery rate: %.2f%% (%d/%d)", deliveryRate, receivedCount, totalWebhooks)
}

// Test batch webhook performance
func (s *LoadTestSuite) TestBatchWebhookPerformance() {
	batchSizes := []int{10, 50, 100, 200}
	
	for _, batchSize := range batchSizes {
		s.Run(fmt.Sprintf("BatchSize_%d", batchSize), func() {
			s.runBatchPerformanceTest(batchSize)
		})
	}
}

func (s *LoadTestSuite) runBatchPerformanceTest(batchSize int) {
	// Create multiple endpoints for batch testing
	endpoints := make([]uuid.UUID, batchSize)
	ctx := context.Background()
	
	for i := 0; i < batchSize; i++ {
		endpoint := &models.WebhookEndpoint{
			ID:       uuid.New(),
			TenantID: s.testTenant.ID,
			URL:      s.webhookServer.URL + fmt.Sprintf("/webhook-%d", i),
			SecretHash: fmt.Sprintf("batch-test-secret-%d", i),
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
		
		endpoints[i] = endpoint.ID
	}
	
	payload := map[string]interface{}{
		"event":      "batch.performance.test",
		"batch_size": batchSize,
		"timestamp":  time.Now().Unix(),
	}
	
	batchRequest := handlers.BatchSendWebhookRequest{
		EndpointIDs: endpoints,
		Payload:     json.RawMessage(mustMarshal(payload)),
		Headers: map[string]string{
			"X-Event-Type": "batch.performance.test",
			"X-Batch-Size": fmt.Sprintf("%d", batchSize),
		},
	}
	
	startTime := time.Now()
	
	w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send/batch", batchRequest)
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, w.Code)
	
	duration := time.Since(startTime)
	
	var response handlers.BatchSendWebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)
	
	s.Equal(batchSize, response.Total)
	s.Equal(batchSize, response.Queued)
	s.Equal(0, response.Failed)
	
	// Performance assertions
	maxExpectedDuration := time.Duration(batchSize) * 10 * time.Millisecond // 10ms per endpoint max
	s.Less(duration, maxExpectedDuration, "Batch processing took too long for size %d", batchSize)
	
	throughput := float64(batchSize) / duration.Seconds()
	minThroughput := 100.0 // At least 100 endpoints per second
	s.Greater(throughput, minThroughput, "Batch throughput too low: %.2f endpoints/second", throughput)
	
	s.T().Logf("Batch size %d processed in %v (%.2f endpoints/second)", 
		batchSize, duration, throughput)
}

// Test queue processing performance
func (s *LoadTestSuite) TestQueueProcessingPerformance() {
	ctx := context.Background()
	
	// Fill queue with messages
	messageCount := 1000
	
	for i := 0; i < messageCount; i++ {
		payload := map[string]interface{}{
			"event":      "queue.performance.test",
			"message_id": i,
		}
		
		sendRequest := handlers.SendWebhookRequest{
			EndpointID: &s.testEndpoint.ID,
			Payload:    json.RawMessage(mustMarshal(payload)),
		}
		
		w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
		s.Require().NoError(err)
		s.Equal(http.StatusAccepted, w.Code)
	}
	
	// Measure queue processing time
	startTime := time.Now()
	initialStats, err := s.queueManager.GetQueueStats(ctx)
	s.Require().NoError(err)
	
	initialQueueDepth := initialStats[queue.DeliveryQueue]
	s.Greater(initialQueueDepth, int64(messageCount/2), "Expected significant queue depth")
	
	// Wait for queue to be processed
	for {
		stats, err := s.queueManager.GetQueueStats(ctx)
		s.Require().NoError(err)
		
		currentDepth := stats[queue.DeliveryQueue] + stats[queue.ProcessingQueue]
		if currentDepth == 0 {
			break
		}
		
		time.Sleep(100 * time.Millisecond)
		
		// Timeout after 2 minutes
		if time.Since(startTime) > 2*time.Minute {
			s.Fail("Queue processing took too long")
			break
		}
	}
	
	processingDuration := time.Since(startTime)
	processingRate := float64(messageCount) / processingDuration.Seconds()
	
	s.T().Logf("Processed %d messages in %v (%.2f messages/second)", 
		messageCount, processingDuration, processingRate)
	
	// Performance assertions
	minProcessingRate := 50.0 // At least 50 messages per second
	s.Greater(processingRate, minProcessingRate, "Queue processing rate too low: %.2f messages/second", processingRate)
	
	// Verify delivery success rate
	receivedCount := atomic.LoadInt64(&s.requestCount)
	successRate := float64(receivedCount) / float64(messageCount) * 100
	s.Greater(successRate, 95.0, "Success rate too low: %.2f%%", successRate)
}

// Test memory usage under load
func (s *LoadTestSuite) TestMemoryUsageUnderLoad() {
	// This test monitors memory usage during high load
	// In a real implementation, you would use runtime.MemStats
	
	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStatsBefore)
	
	// Generate load
	concurrency := 20
	webhooksEach := 100
	totalWebhooks := concurrency * webhooksEach
	
	var wg sync.WaitGroup
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < webhooksEach; j++ {
				payload := map[string]interface{}{
					"event":        "memory.test",
					"goroutine_id": goroutineID,
					"webhook_id":   j,
					"data":         make([]byte, 1024), // 1KB payload
				}
				
				sendRequest := handlers.SendWebhookRequest{
					EndpointID: &s.testEndpoint.ID,
					Payload:    json.RawMessage(mustMarshal(payload)),
				}
				
				s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Wait for processing
	time.Sleep(10 * time.Second)
	
	runtime.GC()
	runtime.ReadMemStats(&memStatsAfter)
	
	// Calculate memory usage
	memoryIncrease := memStatsAfter.Alloc - memStatsBefore.Alloc
	memoryPerWebhook := float64(memoryIncrease) / float64(totalWebhooks)
	
	s.T().Logf("Memory usage: %d bytes total, %.2f bytes per webhook", 
		memoryIncrease, memoryPerWebhook)
	
	// Memory usage should be reasonable (less than 10KB per webhook)
	maxMemoryPerWebhook := 10 * 1024 // 10KB
	s.Less(memoryPerWebhook, float64(maxMemoryPerWebhook), 
		"Memory usage per webhook too high: %.2f bytes", memoryPerWebhook)
}

// Test API response times under load
func (s *LoadTestSuite) TestAPIResponseTimesUnderLoad() {
	concurrency := 10
	requestsEach := 50
	
	var wg sync.WaitGroup
	var totalDuration int64
	var requestCount int64
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < requestsEach; j++ {
				payload := map[string]interface{}{
					"event": "response.time.test",
					"data":  "testing API response times",
				}
				
				sendRequest := handlers.SendWebhookRequest{
					EndpointID: &s.testEndpoint.ID,
					Payload:    json.RawMessage(mustMarshal(payload)),
				}
				
				start := time.Now()
				w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
				duration := time.Since(start)
				
				if err == nil && w.Code == http.StatusAccepted {
					atomic.AddInt64(&totalDuration, int64(duration))
					atomic.AddInt64(&requestCount, 1)
				}
			}
		}()
	}
	
	wg.Wait()
	
	avgResponseTime := time.Duration(atomic.LoadInt64(&totalDuration) / atomic.LoadInt64(&requestCount))
	
	s.T().Logf("Average API response time: %v", avgResponseTime)
	
	// Response time should be reasonable (less than 100ms)
	maxResponseTime := 100 * time.Millisecond
	s.Less(avgResponseTime, maxResponseTime, "Average response time too high: %v", avgResponseTime)
}

// queuePublisherAdapter wraps *queue.Manager to satisfy queue.PublisherInterface
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

// LoadTestDeliveryHandler handles webhook deliveries for load testing
type LoadTestDeliveryHandler struct {
	suite *LoadTestSuite
}

func (m *LoadTestDeliveryHandler) HandleDelivery(ctx context.Context, message *queue.DeliveryMessage) (*queue.DeliveryResult, error) {
	// Simulate HTTP delivery with minimal overhead for load testing
	client := &http.Client{
		Timeout: 5 * time.Second,
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
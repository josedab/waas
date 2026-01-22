package chaos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
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

// ChaosTestSuite provides chaos engineering tests for the webhook platform
type ChaosTestSuite struct {
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
	testEndpoints []*models.WebhookEndpoint
	
	// Chaos servers
	chaosServers []*ChaosWebhookServer
	
	// Metrics
	totalRequests    int64
	successfulRequests int64
	failedRequests   int64
}

// queuePublisherAdapter wraps queue.Manager to implement queue.PublisherInterface
type queuePublisherAdapter struct {
	manager *queue.Manager
}

func (a *queuePublisherAdapter) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	return a.manager.PublishDelivery(ctx, message)
}

func (a *queuePublisherAdapter) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	// Delegate to regular publish for test purposes
	return a.manager.PublishDelivery(ctx, message)
}

func (a *queuePublisherAdapter) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	// No-op for chaos tests
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

// ChaosWebhookServer simulates various failure scenarios
type ChaosWebhookServer struct {
	server          *httptest.Server
	failureRate     float64
	latencyMin      time.Duration
	latencyMax      time.Duration
	intermittentDown bool
	downProbability float64
	requestCount    int64
}

func TestChaosTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos tests in short mode")
	}
	
	// Only run if explicitly requested
	if os.Getenv("RUN_CHAOS_TESTS") != "true" {
		t.Skip("Chaos tests disabled. Set RUN_CHAOS_TESTS=true to enable")
	}
	
	suite.Run(t, new(ChaosTestSuite))
}

func (s *ChaosTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Setup logger
	s.logger = utils.NewLogger("chaos-test")
	
	// Setup database connection
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable"
	}
	os.Setenv("DATABASE_URL", dbURL)
	
	var err error
	s.db, err = database.NewConnection()
	s.Require().NoError(err, "Failed to connect to test database")
	
	// Setup Redis connection
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/13" // Use test database
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
	chaosHandler := &ChaosDeliveryHandler{
		suite: s,
	}
	s.queueManager = queue.NewManager(s.redis, chaosHandler, 3)
	
	// Setup API server
	s.setupAPIServer()
	
	// Create test data
	s.createTestData()
	
	// Start services
	err = s.queueManager.Start(ctx)
	s.Require().NoError(err)
}

func (s *ChaosTestSuite) TearDownSuite() {
	// Stop services
	if s.queueManager != nil && s.queueManager.IsRunning() {
		s.queueManager.Stop()
	}
	
	// Close chaos servers
	for _, server := range s.chaosServers {
		server.server.Close()
	}
	
	// Close connections
	if s.redis != nil {
		s.redis.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *ChaosTestSuite) SetupTest() {
	// Reset metrics
	atomic.StoreInt64(&s.totalRequests, 0)
	atomic.StoreInt64(&s.successfulRequests, 0)
	atomic.StoreInt64(&s.failedRequests, 0)
	
	// Reset server counters
	for _, server := range s.chaosServers {
		atomic.StoreInt64(&server.requestCount, 0)
	}
	
	// Clean up between tests
	ctx := context.Background()
	s.redis.Client.FlushDB(ctx)
}

func (s *ChaosTestSuite) setupAPIServer() {
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
		api.POST("/webhooks/endpoints", webhookHandler.CreateWebhookEndpoint)
		api.POST("/webhooks/send", webhookHandler.SendWebhook)
		api.POST("/webhooks/send/batch", webhookHandler.BatchSendWebhook)
		// GetDeliveryHistory and GetDeliveryDetails are not implemented on WebhookHandler
		// api.GET("/webhooks/deliveries", webhookHandler.GetDeliveryHistory)
		// api.GET("/webhooks/deliveries/:id", webhookHandler.GetDeliveryDetails)
	}
}

func (s *ChaosTestSuite) createTestData() {
	ctx := context.Background()
	
	// Create test tenant
	s.testTenant = &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Chaos Test Tenant",
		APIKeyHash:         "$2a$10$test.hash.for.chaos.testing",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 5000,
		MonthlyQuota:       500000,
	}
	
	err := s.tenantRepo.Create(ctx, s.testTenant)
	s.Require().NoError(err)
	
	// Create chaos webhook servers
	s.createChaosServers()
	
	// Create test webhook endpoints
	s.createTestWebhookEndpoints()
}

func (s *ChaosTestSuite) createChaosServers() {
	// Server 1: High failure rate
	server1 := &ChaosWebhookServer{
		failureRate: 0.7, // 70% failure rate
		latencyMin:  50 * time.Millisecond,
		latencyMax:  200 * time.Millisecond,
	}
	server1.server = httptest.NewServer(http.HandlerFunc(server1.handleRequest))
	
	// Server 2: Intermittent downtime
	server2 := &ChaosWebhookServer{
		failureRate:      0.2, // 20% failure rate when up
		latencyMin:       10 * time.Millisecond,
		latencyMax:       100 * time.Millisecond,
		intermittentDown: true,
		downProbability:  0.3, // 30% chance of being down
	}
	server2.server = httptest.NewServer(http.HandlerFunc(server2.handleRequest))
	
	// Server 3: Slow responses
	server3 := &ChaosWebhookServer{
		failureRate: 0.1, // 10% failure rate
		latencyMin:  1 * time.Second,
		latencyMax:  5 * time.Second,
	}
	server3.server = httptest.NewServer(http.HandlerFunc(server3.handleRequest))
	
	// Server 4: Random chaos
	server4 := &ChaosWebhookServer{
		failureRate:      0.4, // 40% failure rate
		latencyMin:       10 * time.Millisecond,
		latencyMax:       2 * time.Second,
		intermittentDown: true,
		downProbability:  0.2, // 20% chance of being down
	}
	server4.server = httptest.NewServer(http.HandlerFunc(server4.handleRequest))
	
	s.chaosServers = []*ChaosWebhookServer{server1, server2, server3, server4}
}

func (cs *ChaosWebhookServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&cs.requestCount, 1)
	
	// Check if server is down (intermittent downtime)
	if cs.intermittentDown && rand.Float64() < cs.downProbability {
		// Simulate server being down
		time.Sleep(10 * time.Second) // Long timeout
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	
	// Simulate variable latency
	latency := cs.latencyMin + time.Duration(rand.Float64()*float64(cs.latencyMax-cs.latencyMin))
	time.Sleep(latency)
	
	// Simulate failures
	if rand.Float64() < cs.failureRate {
		// Random failure status codes
		statusCodes := []int{500, 502, 503, 504, 408, 429}
		statusCode := statusCodes[rand.Intn(len(statusCodes))]
		w.WriteHeader(statusCode)
		w.Write([]byte(fmt.Sprintf(`{"error": "chaos_failure", "status": %d}`, statusCode)))
		return
	}
	
	// Success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "chaos_success", "received_at": "` + time.Now().Format(time.RFC3339) + `"}`))
}

func (s *ChaosTestSuite) createTestWebhookEndpoints() {
	ctx := context.Background()
	
	for i, server := range s.chaosServers {
		endpoint := &models.WebhookEndpoint{
			ID:       uuid.New(),
			TenantID: s.testTenant.ID,
			URL:      server.server.URL + fmt.Sprintf("/chaos-webhook-%d", i),
			SecretHash: fmt.Sprintf("chaos-secret-hash-%d", i),
			IsActive: true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    100,
				MaxDelayMs:        10000,
				BackoffMultiplier: 2,
			},
		}
		
		err := s.webhookRepo.Create(ctx, endpoint)
		s.Require().NoError(err)
		
		s.testEndpoints = append(s.testEndpoints, endpoint)
	}
}

func (s *ChaosTestSuite) makeAPIRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
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
	req.Header.Set("X-API-Key", "chaos-test-api-key")
	
	w := httptest.NewRecorder()
	s.apiServer.ServeHTTP(w, req)
	
	return w, nil
}

// Test system resilience under chaos conditions
func (s *ChaosTestSuite) TestSystemResilienceUnderChaos() {
	testDuration := 2 * time.Minute
	webhookInterval := 500 * time.Millisecond
	
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Start continuous webhook sending
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.continuousWebhookSending(ctx, webhookInterval)
	}()
	
	// Start chaos injection
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.injectChaos(ctx)
	}()
	
	// Monitor system health
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.monitorSystemHealth(ctx)
	}()
	
	wg.Wait()
	
	// Analyze results
	s.analyzeResilienceResults()
}

func (s *ChaosTestSuite) continuousWebhookSending(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send webhook to random endpoint
			endpointIndex := rand.Intn(len(s.testEndpoints))
			endpoint := s.testEndpoints[endpointIndex]
			
			payload := map[string]interface{}{
				"event":      "chaos.resilience.test",
				"timestamp":  time.Now().Unix(),
				"endpoint":   endpointIndex,
				"request_id": uuid.New().String(),
			}
			
			sendRequest := handlers.SendWebhookRequest{
				EndpointID: &endpoint.ID,
				Payload:    json.RawMessage(mustMarshal(payload)),
				Headers: map[string]string{
					"X-Event-Type": "chaos.resilience.test",
					"X-Endpoint":   fmt.Sprintf("%d", endpointIndex),
				},
			}
			
			atomic.AddInt64(&s.totalRequests, 1)
			
			w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
			if err != nil || w.Code != http.StatusAccepted {
				atomic.AddInt64(&s.failedRequests, 1)
			} else {
				atomic.AddInt64(&s.successfulRequests, 1)
			}
		}
	}
}

func (s *ChaosTestSuite) injectChaos(ctx context.Context) {
	// Periodically inject different types of chaos
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	chaosActions := []func(){
		s.simulateRedisFailure,
		s.simulateHighLoad,
		s.simulateNetworkPartition,
	}
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Randomly select and execute chaos action
			action := chaosActions[rand.Intn(len(chaosActions))]
			action()
		}
	}
}

func (s *ChaosTestSuite) simulateRedisFailure() {
	s.logger.Info("Injecting Redis failure simulation", nil)
	
	// Simulate Redis connection issues by flushing and temporarily blocking
	ctx := context.Background()
	s.redis.Client.FlushDB(ctx)
	
	// Brief pause to simulate connection issues
	time.Sleep(2 * time.Second)
}

func (s *ChaosTestSuite) simulateHighLoad() {
	s.logger.Info("Injecting high load simulation", nil)
	
	// Send burst of webhooks
	burstSize := 50
	var wg sync.WaitGroup
	
	for i := 0; i < burstSize; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			payload := map[string]interface{}{
				"event":    "chaos.high.load",
				"burst_id": id,
			}
			
			endpointIndex := rand.Intn(len(s.testEndpoints))
			endpoint := s.testEndpoints[endpointIndex]
			
			sendRequest := handlers.SendWebhookRequest{
				EndpointID: &endpoint.ID,
				Payload:    json.RawMessage(mustMarshal(payload)),
			}
			
			s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
		}(i)
	}
	
	wg.Wait()
}

func (s *ChaosTestSuite) simulateNetworkPartition() {
	s.logger.Info("Injecting network partition simulation", nil)
	
	// This would typically involve network manipulation
	// For testing, we'll simulate by temporarily making servers unresponsive
	for _, server := range s.chaosServers {
		server.intermittentDown = true
		server.downProbability = 0.8 // 80% chance of being down
	}
	
	// Restore after brief period
	time.Sleep(5 * time.Second)
	
	for _, server := range s.chaosServers {
		server.downProbability = 0.2 // Back to normal
	}
}

func (s *ChaosTestSuite) monitorSystemHealth(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check queue stats
			stats, err := s.queueManager.GetQueueStats(ctx)
			if err != nil {
				s.logger.Error("Failed to get queue stats", map[string]interface{}{"error": err.Error()})
				continue
			}
			
			totalQueued := stats[queue.DeliveryQueue] + stats[queue.RetryQueue] + stats[queue.ProcessingQueue]
			dlqCount := stats[queue.DeadLetterQueue]
			
			s.logger.Info("System health check", map[string]interface{}{
				"total_queued":        totalQueued,
				"dlq_count":           dlqCount,
				"total_requests":      atomic.LoadInt64(&s.totalRequests),
				"successful_requests": atomic.LoadInt64(&s.successfulRequests),
				"failed_requests":     atomic.LoadInt64(&s.failedRequests),
			})
			
			// Alert if queue is growing too large
			if totalQueued > 1000 {
				s.logger.Warn("Queue depth is high", map[string]interface{}{"depth": totalQueued})
			}
		}
	}
}

func (s *ChaosTestSuite) analyzeResilienceResults() {
	totalRequests := atomic.LoadInt64(&s.totalRequests)
	successfulRequests := atomic.LoadInt64(&s.successfulRequests)
	failedRequests := atomic.LoadInt64(&s.failedRequests)
	
	s.T().Logf("Chaos resilience test results:")
	s.T().Logf("  Total requests: %d", totalRequests)
	s.T().Logf("  Successful requests: %d", successfulRequests)
	s.T().Logf("  Failed requests: %d", failedRequests)
	
	if totalRequests > 0 {
		successRate := float64(successfulRequests) / float64(totalRequests) * 100
		s.T().Logf("  Success rate: %.2f%%", successRate)
		
		// System should maintain at least 80% success rate even under chaos
		s.Greater(successRate, 80.0, "System success rate too low under chaos conditions")
	}
	
	// Check that system didn't completely fail
	s.Greater(totalRequests, int64(10), "Too few requests processed")
	s.Greater(successfulRequests, int64(5), "Too few successful requests")
	
	// Verify server request distribution
	for i, server := range s.chaosServers {
		requestCount := atomic.LoadInt64(&server.requestCount)
		s.T().Logf("  Chaos server %d received %d requests", i, requestCount)
	}
}

// Test failure recovery scenarios
func (s *ChaosTestSuite) TestFailureRecoveryScenarios() {
	scenarios := []struct {
		name        string
		setupChaos  func()
		testAction  func() error
		verifyChaos func() bool
	}{
		{
			name: "High Failure Rate Recovery",
			setupChaos: func() {
				for _, server := range s.chaosServers {
					server.failureRate = 0.9 // 90% failure rate
				}
			},
			testAction: func() error {
				return s.sendTestWebhooks(20)
			},
			verifyChaos: func() bool {
				// Verify that retries eventually succeed for some webhooks
				return s.verifyEventualSuccess()
			},
		},
		{
			name: "Timeout Recovery",
			setupChaos: func() {
				for _, server := range s.chaosServers {
					server.latencyMin = 10 * time.Second
					server.latencyMax = 15 * time.Second
				}
			},
			testAction: func() error {
				return s.sendTestWebhooks(10)
			},
			verifyChaos: func() bool {
				// Verify that timeouts are handled gracefully
				return s.verifyTimeoutHandling()
			},
		},
		{
			name: "Intermittent Downtime Recovery",
			setupChaos: func() {
				for _, server := range s.chaosServers {
					server.intermittentDown = true
					server.downProbability = 0.7 // 70% chance of being down
				}
			},
			testAction: func() error {
				return s.sendTestWebhooks(15)
			},
			verifyChaos: func() bool {
				// Verify that system handles downtime gracefully
				return s.verifyDowntimeHandling()
			},
		},
	}
	
	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			// Setup chaos conditions
			scenario.setupChaos()
			
			// Execute test action
			err := scenario.testAction()
			s.NoError(err, "Test action should not fail")
			
			// Wait for processing and retries
			time.Sleep(30 * time.Second)
			
			// Verify chaos handling
			handled := scenario.verifyChaos()
			s.True(handled, "System should handle chaos scenario gracefully")
			
			// Reset chaos conditions
			s.resetChaosConditions()
		})
	}
}

func (s *ChaosTestSuite) sendTestWebhooks(count int) error {
	for i := 0; i < count; i++ {
		payload := map[string]interface{}{
			"event":      "chaos.recovery.test",
			"webhook_id": i,
			"timestamp":  time.Now().Unix(),
		}
		
		endpointIndex := rand.Intn(len(s.testEndpoints))
		endpoint := s.testEndpoints[endpointIndex]
		
		sendRequest := handlers.SendWebhookRequest{
			EndpointID: &endpoint.ID,
			Payload:    json.RawMessage(mustMarshal(payload)),
		}
		
		w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
		if err != nil {
			return err
		}
		
		if w.Code != http.StatusAccepted {
			return fmt.Errorf("unexpected status code: %d", w.Code)
		}
	}
	
	return nil
}

func (s *ChaosTestSuite) verifyEventualSuccess() bool {
	// Check delivery history for eventual successes
	w, err := s.makeAPIRequest("GET", "/api/v1/webhooks/deliveries?limit=100", nil)
	if err != nil {
		return false
	}
	
	if w.Code != http.StatusOK {
		return false
	}
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		return false
	}
	
	deliveries := response["deliveries"].([]interface{})
	successCount := 0
	
	for _, delivery := range deliveries {
		d := delivery.(map[string]interface{})
		if d["status"].(string) == "success" {
			successCount++
		}
	}
	
	// At least some deliveries should eventually succeed
	return successCount > 0
}

func (s *ChaosTestSuite) verifyTimeoutHandling() bool {
	// Verify that timeout errors are properly recorded
	w, err := s.makeAPIRequest("GET", "/api/v1/webhooks/deliveries?limit=100", nil)
	if err != nil {
		return false
	}
	
	if w.Code != http.StatusOK {
		return false
	}
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		return false
	}
	
	deliveries := response["deliveries"].([]interface{})
	timeoutCount := 0
	
	for _, delivery := range deliveries {
		d := delivery.(map[string]interface{})
		if d["status"].(string) == "failed" {
			// Check if error message indicates timeout
			if errorMsg, exists := d["error_message"]; exists {
				if errorMsg != nil && fmt.Sprintf("%v", errorMsg) != "" {
					timeoutCount++
				}
			}
		}
	}
	
	// Should have some timeout-related failures
	return timeoutCount > 0
}

func (s *ChaosTestSuite) verifyDowntimeHandling() bool {
	// Verify that system continues to function despite downtime
	ctx := context.Background()
	stats, err := s.queueManager.GetQueueStats(ctx)
	if err != nil {
		return false
	}
	
	// System should still be processing messages
	totalQueued := stats[queue.DeliveryQueue] + stats[queue.RetryQueue] + stats[queue.ProcessingQueue]
	
	// Queue should not be completely stalled
	return totalQueued < 1000 // Reasonable queue depth
}

func (s *ChaosTestSuite) resetChaosConditions() {
	for _, server := range s.chaosServers {
		server.failureRate = 0.2
		server.latencyMin = 50 * time.Millisecond
		server.latencyMax = 200 * time.Millisecond
		server.intermittentDown = false
		server.downProbability = 0.1
	}
}

// Test queue overflow and backpressure
func (s *ChaosTestSuite) TestQueueOverflowAndBackpressure() {
	// Fill queue rapidly
	overflowCount := 500
	
	for i := 0; i < overflowCount; i++ {
		payload := map[string]interface{}{
			"event":      "chaos.overflow.test",
			"message_id": i,
		}
		
		endpointIndex := rand.Intn(len(s.testEndpoints))
		endpoint := s.testEndpoints[endpointIndex]
		
		sendRequest := handlers.SendWebhookRequest{
			EndpointID: &endpoint.ID,
			Payload:    json.RawMessage(mustMarshal(payload)),
		}
		
		w, err := s.makeAPIRequest("POST", "/api/v1/webhooks/send", sendRequest)
		s.NoError(err)
		s.Equal(http.StatusAccepted, w.Code)
	}
	
	// Check queue stats
	ctx := context.Background()
	stats, err := s.queueManager.GetQueueStats(ctx)
	s.NoError(err)
	
	totalQueued := stats[queue.DeliveryQueue] + stats[queue.RetryQueue] + stats[queue.ProcessingQueue]
	s.Greater(totalQueued, int64(overflowCount/2), "Expected significant queue depth")
	
	// Wait for processing
	time.Sleep(60 * time.Second)
	
	// Verify queue is being processed
	newStats, err := s.queueManager.GetQueueStats(ctx)
	s.NoError(err)
	
	newTotalQueued := newStats[queue.DeliveryQueue] + newStats[queue.RetryQueue] + newStats[queue.ProcessingQueue]
	s.Less(newTotalQueued, totalQueued, "Queue should be processing messages")
	
	s.T().Logf("Queue overflow test: %d -> %d messages", totalQueued, newTotalQueued)
}

// ChaosDeliveryHandler handles webhook deliveries for chaos testing
type ChaosDeliveryHandler struct {
	suite *ChaosTestSuite
}

func (m *ChaosDeliveryHandler) HandleDelivery(ctx context.Context, message *queue.DeliveryMessage) (*queue.DeliveryResult, error) {
	// Simulate HTTP delivery with chaos conditions
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
package queue

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"webhook-platform/pkg/database"
)

// MockMessageHandler implements MessageHandler for testing
type MockMessageHandler struct {
	results map[string]*DeliveryResult
	delay   time.Duration
}

func NewMockMessageHandler() *MockMessageHandler {
	return &MockMessageHandler{
		results: make(map[string]*DeliveryResult),
	}
}

func (m *MockMessageHandler) HandleDelivery(ctx context.Context, message *DeliveryMessage) (*DeliveryResult, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	deliveryID := message.DeliveryID.String()
	if result, exists := m.results[deliveryID]; exists {
		return result, nil
	}

	// Default success result
	now := time.Now()
	return &DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        StatusSuccess,
		HTTPStatus:    &[]int{200}[0],
		DeliveredAt:   &now,
		AttemptNumber: message.AttemptNumber,
	}, nil
}

func (m *MockMessageHandler) SetResult(deliveryID string, result *DeliveryResult) {
	m.results[deliveryID] = result
}

func (m *MockMessageHandler) SetDelay(delay time.Duration) {
	m.delay = delay
}

func setupTestRedis(t *testing.T) *database.RedisClient {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/1" // Use database 1 for tests
	}

	redis, err := database.NewRedisConnection(redisURL)
	require.NoError(t, err)

	// Clean up test database
	ctx := context.Background()
	redis.Client.FlushDB(ctx)

	return redis
}

func TestPublisher_PublishDelivery(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	publisher := NewPublisher(redis)
	ctx := context.Background()

	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data"}`),
		Headers:       map[string]string{"Content-Type": "application/json"},
		AttemptNumber: 1,
		ScheduledAt:   time.Now(),
		Signature:     "test-signature",
		MaxAttempts:   3,
	}

	err := publisher.PublishDelivery(ctx, message)
	assert.NoError(t, err)

	// Verify message is in queue
	length, err := publisher.GetQueueLength(ctx, DeliveryQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), length)

	// Verify we can retrieve the message
	result, err := redis.Client.RPop(ctx, DeliveryQueue).Result()
	assert.NoError(t, err)

	var retrievedMessage DeliveryMessage
	err = retrievedMessage.FromJSON([]byte(result))
	assert.NoError(t, err)
	assert.Equal(t, message.DeliveryID, retrievedMessage.DeliveryID)
}

func TestPublisher_PublishDelayedDelivery(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	publisher := NewPublisher(redis)
	ctx := context.Background()

	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data"}`),
		AttemptNumber: 2,
		MaxAttempts:   3,
	}

	delay := 5 * time.Second
	err := publisher.PublishDelayedDelivery(ctx, message, delay)
	assert.NoError(t, err)

	// Verify message is in retry queue
	length, err := publisher.GetQueueLength(ctx, RetryQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), length)
}

func TestPublisher_PublishToDeadLetter(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	publisher := NewPublisher(redis)
	ctx := context.Background()

	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		AttemptNumber: 3,
		MaxAttempts:   3,
	}

	reason := "Max attempts exceeded"
	err := publisher.PublishToDeadLetter(ctx, message, reason)
	assert.NoError(t, err)

	// Verify message is in dead letter queue
	length, err := publisher.GetQueueLength(ctx, DeadLetterQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), length)
}

func TestConsumer_ProcessMessage(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	handler := NewMockMessageHandler()
	consumer := NewConsumer(redis, handler, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start consumer
	err := consumer.Start(ctx)
	require.NoError(t, err)
	defer consumer.Stop()

	// Publish a test message
	publisher := NewPublisher(redis)
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
	}

	err = publisher.PublishDelivery(ctx, message)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Verify queue is empty (message was processed)
	length, err := publisher.GetQueueLength(ctx, DeliveryQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), length)
}

func TestConsumer_RetryLogic(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	handler := NewMockMessageHandler()
	consumer := NewConsumer(redis, handler, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set up handler to fail first attempt
	deliveryID := uuid.New()
	errorMsg := "Temporary failure"
	handler.SetResult(deliveryID.String(), &DeliveryResult{
		DeliveryID:    deliveryID,
		Status:        StatusFailed,
		ErrorMessage:  &errorMsg,
		AttemptNumber: 1,
	})

	// Start consumer
	err := consumer.Start(ctx)
	require.NoError(t, err)
	defer consumer.Stop()

	// Publish a test message
	publisher := NewPublisher(redis)
	message := &DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
	}

	err = publisher.PublishDelivery(ctx, message)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Verify message was moved to retry queue
	retryLength, err := publisher.GetQueueLength(ctx, RetryQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), retryLength)
}

func TestConsumer_DeadLetterQueue(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	handler := NewMockMessageHandler()
	consumer := NewConsumer(redis, handler, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set up handler to always fail
	deliveryID := uuid.New()
	errorMsg := "Permanent failure"
	handler.SetResult(deliveryID.String(), &DeliveryResult{
		DeliveryID:    deliveryID,
		Status:        StatusFailed,
		ErrorMessage:  &errorMsg,
		AttemptNumber: 3, // Max attempts
	})

	// Start consumer
	err := consumer.Start(ctx)
	require.NoError(t, err)
	defer consumer.Stop()

	// Publish a test message that has already reached max attempts
	publisher := NewPublisher(redis)
	message := &DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data"}`),
		AttemptNumber: 3, // Max attempts
		MaxAttempts:   3,
	}

	err = publisher.PublishDelivery(ctx, message)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Verify message was moved to dead letter queue
	dlqLength, err := publisher.GetQueueLength(ctx, DeadLetterQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), dlqLength)
}

func TestRetryProcessor(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	publisher := NewPublisher(redis)
	retryProcessor := NewRetryProcessor(redis, publisher)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add a message to retry queue with past timestamp (ready for retry)
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data"}`),
		AttemptNumber: 2,
		MaxAttempts:   3,
	}

	// Schedule for immediate retry (past timestamp)
	err := publisher.PublishDelayedDelivery(ctx, message, -1*time.Second)
	require.NoError(t, err)

	// Process ready messages
	err = retryProcessor.processReadyMessages(ctx)
	assert.NoError(t, err)

	// Verify message was moved back to delivery queue
	deliveryLength, err := publisher.GetQueueLength(ctx, DeliveryQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deliveryLength)

	// Verify message was removed from retry queue
	retryLength, err := publisher.GetQueueLength(ctx, RetryQueue)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), retryLength)
}

func TestManager_Integration(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	handler := NewMockMessageHandler()
	manager := NewManager(redis, handler, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test health check
	err := manager.HealthCheck(ctx)
	assert.NoError(t, err)

	// Start manager
	err = manager.Start(ctx)
	require.NoError(t, err)
	assert.True(t, manager.IsRunning())
	defer manager.Stop()

	// Publish multiple messages
	for i := 0; i < 5; i++ {
		message := &DeliveryMessage{
			DeliveryID:    uuid.New(),
			EndpointID:    uuid.New(),
			TenantID:      uuid.New(),
			Payload:       json.RawMessage(`{"test": "data"}`),
			AttemptNumber: 1,
			MaxAttempts:   3,
		}

		err = manager.PublishDelivery(ctx, message)
		require.NoError(t, err)
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Check queue stats
	stats, err := manager.GetQueueStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats[DeliveryQueue]) // All messages should be processed
}

func TestQueueStats(t *testing.T) {
	redis := setupTestRedis(t)
	defer redis.Close()

	publisher := NewPublisher(redis)
	ctx := context.Background()

	// Add messages to different queues
	message := &DeliveryMessage{
		DeliveryID: uuid.New(),
		EndpointID: uuid.New(),
		TenantID:   uuid.New(),
	}

	// Add to delivery queue
	err := publisher.PublishDelivery(ctx, message)
	require.NoError(t, err)

	// Add to retry queue
	err = publisher.PublishDelayedDelivery(ctx, message, time.Hour)
	require.NoError(t, err)

	// Add to dead letter queue
	err = publisher.PublishToDeadLetter(ctx, message, "test")
	require.NoError(t, err)

	// Get stats
	stats, err := publisher.GetQueueStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stats[DeliveryQueue])
	assert.Equal(t, int64(1), stats[RetryQueue])
	assert.Equal(t, int64(1), stats[DeadLetterQueue])
}
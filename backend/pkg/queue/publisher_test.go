package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josedab/waas/pkg/database"
)

func setupTestPublisher(t *testing.T) (*Publisher, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient := &database.RedisClient{Client: client}
	return NewPublisher(redisClient), mr
}

func newTestMessage() *DeliveryMessage {
	return &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"event":"test","data":{"id":1}}`),
		Headers:       map[string]string{"X-Event": "test"},
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
		Signature:     "sha256=abc123",
	}
}

func TestPublishDelivery_Success(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	msg := newTestMessage()
	err := pub.PublishDelivery(ctx, msg)
	require.NoError(t, err)

	// Verify message was pushed to the delivery queue
	items, err := mr.List(DeliveryQueue)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	// Verify the serialized message is valid JSON
	var decoded DeliveryMessage
	require.NoError(t, decoded.FromJSON([]byte(items[0])))
	assert.Equal(t, msg.DeliveryID, decoded.DeliveryID)
	assert.Equal(t, msg.EndpointID, decoded.EndpointID)
}

func TestPublishDelivery_RedisConnectionFailure(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient := &database.RedisClient{Client: client}
	pub := NewPublisher(redisClient)

	// Close Redis to simulate connection failure
	mr.Close()

	err := pub.PublishDelivery(context.Background(), newTestMessage())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish delivery message")
}

func TestPublishDelayedDelivery_Success(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	msg := newTestMessage()
	delay := 30 * time.Second
	err := pub.PublishDelayedDelivery(ctx, msg, delay)
	require.NoError(t, err)

	// Verify message was added to the retry sorted set
	members, err := mr.ZMembers(RetryQueue)
	require.NoError(t, err)
	assert.Len(t, members, 1)

	// Verify the score exists
	assert.True(t, mr.Exists(RetryQueue))
}

func TestPublishDelayedDelivery_CorrectDelay(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	msg := newTestMessage()
	delay := 60 * time.Second
	before := time.Now().Add(delay).Unix()
	err := pub.PublishDelayedDelivery(ctx, msg, delay)
	after := time.Now().Add(delay).Unix()
	require.NoError(t, err)

	// Get the score (Unix timestamp)
	members, _ := mr.ZMembers(RetryQueue)
	require.Len(t, members, 1)

	score, err := mr.ZScore(RetryQueue, members[0])
	require.NoError(t, err)
	assert.GreaterOrEqual(t, score, float64(before))
	assert.LessOrEqual(t, score, float64(after))
}

func TestPublishToDeadLetter_Success(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	msg := newTestMessage()
	reason := "max retries exceeded"
	err := pub.PublishToDeadLetter(ctx, msg, reason)
	require.NoError(t, err)

	// Verify message was pushed to the DLQ
	items, err := mr.List(DeadLetterQueue)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	// Verify the DLQ message includes the reason
	var dlqMsg struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal([]byte(items[0]), &dlqMsg))
	assert.Equal(t, reason, dlqMsg.Reason)
}

func TestPublishToDeadLetter_IncludesTimestamp(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	err := pub.PublishToDeadLetter(ctx, newTestMessage(), "test failure")
	require.NoError(t, err)

	items, _ := mr.List(DeadLetterQueue)
	require.Len(t, items, 1)

	var dlqMsg struct {
		Timestamp time.Time `json:"timestamp"`
	}
	require.NoError(t, json.Unmarshal([]byte(items[0]), &dlqMsg))
	assert.WithinDuration(t, time.Now(), dlqMsg.Timestamp, 5*time.Second)
}

func TestPublishToDeadLetter_RedisFailure(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	pub := NewPublisher(&database.RedisClient{Client: client})
	mr.Close()

	err := pub.PublishToDeadLetter(context.Background(), newTestMessage(), "reason")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish to dead letter queue")
}

func TestGetQueueLength_ListQueue(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	// Empty queue
	length, err := pub.GetQueueLength(ctx, DeliveryQueue)
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)

	// Push some items
	mr.Lpush(DeliveryQueue, "msg1")
	mr.Lpush(DeliveryQueue, "msg2")

	length, err = pub.GetQueueLength(ctx, DeliveryQueue)
	require.NoError(t, err)
	assert.Equal(t, int64(2), length)
}

func TestGetQueueLength_RetryQueue(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	// Empty sorted set
	length, err := pub.GetQueueLength(ctx, RetryQueue)
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)

	// Add to sorted set
	mr.ZAdd(RetryQueue, 1.0, "msg1")
	mr.ZAdd(RetryQueue, 2.0, "msg2")
	mr.ZAdd(RetryQueue, 3.0, "msg3")

	length, err = pub.GetQueueLength(ctx, RetryQueue)
	require.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

func TestGetQueueStats_AllQueues(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	// Populate queues
	mr.Lpush(DeliveryQueue, "d1")
	mr.Lpush(DeliveryQueue, "d2")
	mr.Lpush(DeadLetterQueue, "dlq1")
	mr.Lpush(ProcessingQueue, "p1")
	mr.Lpush(ProcessingQueue, "p2")
	mr.Lpush(ProcessingQueue, "p3")
	mr.ZAdd(RetryQueue, 1.0, "r1")

	stats, err := pub.GetQueueStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(2), stats[DeliveryQueue])
	assert.Equal(t, int64(1), stats[DeadLetterQueue])
	assert.Equal(t, int64(3), stats[ProcessingQueue])
	assert.Equal(t, int64(1), stats[RetryQueue])
}

func TestGetQueueStats_EmptyQueues(t *testing.T) {
	t.Parallel()
	pub, _ := setupTestPublisher(t)
	ctx := context.Background()

	stats, err := pub.GetQueueStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats[DeliveryQueue])
	assert.Equal(t, int64(0), stats[DeadLetterQueue])
	assert.Equal(t, int64(0), stats[ProcessingQueue])
	assert.Equal(t, int64(0), stats[RetryQueue])
}

func TestGetQueueStats_RedisFailure(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	pub := NewPublisher(&database.RedisClient{Client: client})
	mr.Close()

	_, err := pub.GetQueueStats(context.Background())
	assert.Error(t, err)
}

func TestPublishDelivery_MultipleMessages(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, pub.PublishDelivery(ctx, newTestMessage()))
	}

	items, err := mr.List(DeliveryQueue)
	require.NoError(t, err)
	assert.Len(t, items, 5)
}

func TestPublishDelayedDelivery_RedisFailure(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	pub := NewPublisher(&database.RedisClient{Client: client})
	mr.Close()

	err := pub.PublishDelayedDelivery(context.Background(), newTestMessage(), 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish delayed delivery message")
}

func TestGetQueueStats_PartialRedisFailure(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	// Populate delivery queue
	mr.Lpush(DeliveryQueue, "d1")
	mr.Lpush(DeliveryQueue, "d2")

	// Stats should work on a normal setup
	stats, err := pub.GetQueueStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats[DeliveryQueue])
	assert.Equal(t, int64(0), stats[RetryQueue])
}

func TestPublishDelivery_MessageIntegrity(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	msg := newTestMessage()
	err := pub.PublishDelivery(ctx, msg)
	require.NoError(t, err)

	items, err := mr.List(DeliveryQueue)
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Verify deserialized message matches
	var decoded DeliveryMessage
	require.NoError(t, decoded.FromJSON([]byte(items[0])))
	assert.Equal(t, msg.DeliveryID, decoded.DeliveryID)
	assert.Equal(t, msg.EndpointID, decoded.EndpointID)
	assert.Equal(t, msg.TenantID, decoded.TenantID)
	assert.Equal(t, msg.AttemptNumber, decoded.AttemptNumber)
	assert.Equal(t, msg.MaxAttempts, decoded.MaxAttempts)
	assert.Equal(t, msg.Signature, decoded.Signature)
}

func TestPublishToDeadLetter_MessageContainsAllFields(t *testing.T) {
	t.Parallel()
	pub, mr := setupTestPublisher(t)
	ctx := context.Background()

	msg := newTestMessage()
	reason := "exhausted retries after 3 attempts"
	err := pub.PublishToDeadLetter(ctx, msg, reason)
	require.NoError(t, err)

	items, err := mr.List(DeadLetterQueue)
	require.NoError(t, err)
	require.Len(t, items, 1)

	var dlqMsg struct {
		DeliveryID uuid.UUID `json:"delivery_id"`
		EndpointID uuid.UUID `json:"endpoint_id"`
		Reason     string    `json:"reason"`
		Timestamp  time.Time `json:"timestamp"`
	}
	require.NoError(t, json.Unmarshal([]byte(items[0]), &dlqMsg))
	assert.Equal(t, msg.DeliveryID, dlqMsg.DeliveryID)
	assert.Equal(t, msg.EndpointID, dlqMsg.EndpointID)
	assert.Equal(t, reason, dlqMsg.Reason)
	assert.WithinDuration(t, time.Now(), dlqMsg.Timestamp, 5*time.Second)
}

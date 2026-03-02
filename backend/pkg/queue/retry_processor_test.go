package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josedab/waas/pkg/database"
)

func setupTestRetryProcessor(t *testing.T) (*RetryProcessor, *Publisher, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient := &database.RedisClient{Client: client}
	pub := NewPublisher(redisClient)
	rp := NewRetryProcessor(redisClient, pub)
	return rp, pub, mr
}

func addRetryMessage(t *testing.T, mr *miniredis.Miniredis, msg *DeliveryMessage, score float64) {
	t.Helper()
	data, err := msg.ToJSON()
	require.NoError(t, err)
	mr.ZAdd(RetryQueue, score, string(data))
}

func TestRetryProcessor_StartStop(t *testing.T) {
	t.Parallel()
	rp, _, _ := setupTestRetryProcessor(t)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		rp.Start(ctx)
		close(done)
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context should stop the processor
	cancel()
	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("RetryProcessor did not stop within timeout")
	}
}

func TestRetryProcessor_StopChannel(t *testing.T) {
	t.Parallel()
	rp, _, _ := setupTestRetryProcessor(t)

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		rp.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Stop via channel
	rp.Stop()
	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("RetryProcessor did not stop via Stop()")
	}
}

func TestRetryProcessor_ProcessReadyMessages(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	msg := newTestMessage()
	pastScore := float64(time.Now().Add(-10 * time.Second).Unix())
	addRetryMessage(t, mr, msg, pastScore)

	// processReadyMessages now uses strconv.FormatFloat for proper ZSET queries
	err := rp.processReadyMessages(ctx)
	require.NoError(t, err)

	// Verify message was moved
	items, err := mr.List(DeliveryQueue)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	members, _ := mr.ZMembers(RetryQueue)
	assert.Empty(t, members)
}

func TestRetryProcessor_NoReadyMessages(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	// Add a message with a future score (not ready)
	msg := newTestMessage()
	futureScore := float64(time.Now().Add(1 * time.Hour).Unix())
	addRetryMessage(t, mr, msg, futureScore)

	// processReadyMessages should not panic and future messages stay in retry queue
	assert.NotPanics(t, func() {
		err := rp.processReadyMessages(ctx)
		require.NoError(t, err)
	})

	// Retry queue should still have the message regardless
	members, _ := mr.ZMembers(RetryQueue)
	assert.Len(t, members, 1)
}

func TestRetryProcessor_InvalidMessageInQueue(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	// Add invalid JSON to the retry queue
	pastScore := float64(time.Now().Add(-10 * time.Second).Unix())
	mr.ZAdd(RetryQueue, pastScore, "invalid-json-data")

	// processReadyMessages should not panic; invalid JSON is handled gracefully
	assert.NotPanics(t, func() {
		err := rp.processReadyMessages(ctx)
		require.NoError(t, err)
	})
}

func TestRetryProcessor_GetReadyCount(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	// Empty queue
	// GetReadyCount now uses strconv.FormatFloat — works correctly with miniredis
	count, err := rp.GetReadyCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add past messages (ready)
	pastScore := float64(time.Now().Add(-10 * time.Second).Unix())
	addRetryMessage(t, mr, newTestMessage(), pastScore)
	addRetryMessage(t, mr, newTestMessage(), pastScore-1)

	count, err = rp.GetReadyCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestRetryProcessor_GetPendingRetries_Empty(t *testing.T) {
	t.Parallel()
	rp, _, _ := setupTestRetryProcessor(t)
	ctx := context.Background()

	retries, err := rp.GetPendingRetries(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, retries)
}

func TestRetryProcessor_GetPendingRetries_WithItems(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	msg1 := newTestMessage()
	msg2 := newTestMessage()
	score1 := float64(time.Now().Add(1 * time.Minute).Unix())
	score2 := float64(time.Now().Add(2 * time.Minute).Unix())
	addRetryMessage(t, mr, msg1, score1)
	addRetryMessage(t, mr, msg2, score2)

	retries, err := rp.GetPendingRetries(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, retries, 2)

	// Verify fields
	assert.Equal(t, msg1.DeliveryID.String(), retries[0].DeliveryID)
	assert.Equal(t, msg1.EndpointID.String(), retries[0].EndpointID)
	assert.Equal(t, msg1.AttemptNumber, retries[0].AttemptNumber)
}

func TestRetryProcessor_GetPendingRetries_SkipsInvalid(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	// Add valid and invalid messages
	addRetryMessage(t, mr, newTestMessage(), float64(time.Now().Unix()))
	mr.ZAdd(RetryQueue, float64(time.Now().Unix()+1), "invalid-json")

	retries, err := rp.GetPendingRetries(ctx, 10)
	require.NoError(t, err)
	// Only valid message should be returned
	assert.Len(t, retries, 1)
}

func TestRetryProcessor_GetPendingRetries_LimitRespected(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		addRetryMessage(t, mr, newTestMessage(), float64(time.Now().Unix()+int64(i)))
	}

	retries, err := rp.GetPendingRetries(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, retries, 3)
}

func TestRetryProcessor_MultipleReadyMessages(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	pastScore := float64(time.Now().Add(-10 * time.Second).Unix())
	for i := 0; i < 3; i++ {
		addRetryMessage(t, mr, newTestMessage(), pastScore-float64(i))
	}

	err := rp.processReadyMessages(ctx)
	require.NoError(t, err)

	items, _ := mr.List(DeliveryQueue)
	assert.Len(t, items, 3)

	members, _ := mr.ZMembers(RetryQueue)
	assert.Empty(t, members)
}

func TestRetryProcessor_EmptyQueue(t *testing.T) {
	t.Parallel()
	rp, _, _ := setupTestRetryProcessor(t)

	// processReadyMessages on empty queue should not panic
	assert.NotPanics(t, func() {
		rp.processReadyMessages(context.Background())
	})
}

func TestNewRetryProcessor(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient := &database.RedisClient{Client: client}
	pub := NewPublisher(redisClient)
	rp := NewRetryProcessor(redisClient, pub)

	assert.NotNil(t, rp)
	assert.NotNil(t, rp.stopCh)
}

// Test that invalid JSON in retry queue doesn't cause a panic
func TestRetryProcessor_InvalidJSON_NoCrash(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)

	pastScore := float64(time.Now().Add(-10 * time.Second).Unix())
	mr.ZAdd(RetryQueue, pastScore, "{not valid json")
	mr.ZAdd(RetryQueue, pastScore-1, "")

	// Add a valid message too
	validMsg := newTestMessage()
	data, _ := json.Marshal(validMsg)
	mr.ZAdd(RetryQueue, pastScore-2, string(data))

	// Should not panic regardless of internal Redis handling
	assert.NotPanics(t, func() {
		rp.processReadyMessages(context.Background())
	})
}

func TestRetryProcessor_ExactBoundaryTimestamp(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	// Add a message with score equal to "now" — should be included (Max uses <=)
	now := float64(time.Now().Unix())
	msg := newTestMessage()
	addRetryMessage(t, mr, msg, now)

	err := rp.processReadyMessages(ctx)
	require.NoError(t, err)

	items, err := mr.List(DeliveryQueue)
	require.NoError(t, err)
	assert.Len(t, items, 1, "message at exact boundary should be processed")

	members, _ := mr.ZMembers(RetryQueue)
	assert.Empty(t, members)
}

func TestRetryProcessor_FloatTimestampScores(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	// Use fractional scores — strconv.FormatFloat truncates to integer
	pastScore := float64(time.Now().Add(-5*time.Second).Unix()) + 0.999
	addRetryMessage(t, mr, newTestMessage(), pastScore)

	err := rp.processReadyMessages(ctx)
	require.NoError(t, err)

	items, err := mr.List(DeliveryQueue)
	require.NoError(t, err)
	assert.Len(t, items, 1, "message with fractional score in the past should be processed")
}

func TestRetryProcessor_GetReadyCount_WithBoundary(t *testing.T) {
	t.Parallel()
	rp, _, mr := setupTestRetryProcessor(t)
	ctx := context.Background()

	now := float64(time.Now().Unix())
	// One message at boundary, one in the past, one in the future
	addRetryMessage(t, mr, newTestMessage(), now)
	addRetryMessage(t, mr, newTestMessage(), now-10)
	addRetryMessage(t, mr, newTestMessage(), now+3600)

	count, err := rp.GetReadyCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "boundary + past messages should be ready, future should not")
}

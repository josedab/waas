package queue

import (
	"context"
	"time"

	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/utils"
	"github.com/redis/go-redis/v9"
)

// RetryProcessor handles processing delayed messages from the retry queue
type RetryProcessor struct {
	redis     *database.RedisClient
	publisher *Publisher
	stopCh    chan struct{}
	logger    *utils.Logger
}

// NewRetryProcessor creates a new retry processor
func NewRetryProcessor(redisClient *database.RedisClient, publisher *Publisher) *RetryProcessor {
	return &RetryProcessor{
		redis:     redisClient,
		publisher: publisher,
		stopCh:    make(chan struct{}),
		logger:    utils.NewLogger("queue-retry"),
	}
}

// Start begins processing delayed messages
func (rp *RetryProcessor) Start(ctx context.Context) {
	rp.logger.Info("Starting retry processor", nil)
	defer rp.logger.Info("Retry processor stopped", nil)

	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-rp.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := rp.processReadyMessages(ctx); err != nil {
				rp.logger.Error("Retry processor error", map[string]interface{}{"error": err.Error()})
			}
		}
	}
}

// Stop stops the retry processor
func (rp *RetryProcessor) Stop() {
	close(rp.stopCh)
}

// processReadyMessages moves messages that are ready for retry back to the main queue
func (rp *RetryProcessor) processReadyMessages(ctx context.Context) error {
	now := float64(time.Now().Unix())

	// Get messages that are ready for processing (score <= now)
	messages, err := rp.redis.Client.ZRangeByScoreWithScores(ctx, RetryQueue, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   string(rune(int(now))),
		Count: 100, // Process up to 100 messages at a time
	}).Result()

	if err != nil {
		return err
	}

	if len(messages) == 0 {
		return nil
	}

	rp.logger.Info("Processing ready retry messages", map[string]interface{}{"count": len(messages)})

	// Process each ready message
	for _, msg := range messages {
		messageData := msg.Member.(string)

		// Parse the message to validate it
		var deliveryMessage DeliveryMessage
		if err := deliveryMessage.FromJSON([]byte(messageData)); err != nil {
			rp.logger.Warn("Invalid message in retry queue, removing", map[string]interface{}{"error": err.Error()})
			// Remove invalid message
			rp.redis.Client.ZRem(ctx, RetryQueue, messageData)
			continue
		}

		// Move message back to main delivery queue
		pipe := rp.redis.Client.TxPipeline()
		pipe.ZRem(ctx, RetryQueue, messageData)
		pipe.LPush(ctx, DeliveryQueue, messageData)

		if _, err := pipe.Exec(ctx); err != nil {
			rp.logger.Error("Failed to move retry message to delivery queue", map[string]interface{}{"error": err.Error()})
			continue
		}

		rp.logger.Info("Moved delivery back to main queue for retry", map[string]interface{}{"delivery_id": deliveryMessage.DeliveryID.String(), "attempt": deliveryMessage.AttemptNumber})
	}

	return nil
}

// GetReadyCount returns the number of messages ready for retry
func (rp *RetryProcessor) GetReadyCount(ctx context.Context) (int64, error) {
	now := float64(time.Now().Unix())
	return rp.redis.Client.ZCount(ctx, RetryQueue, "-inf", string(rune(int(now)))).Result()
}

// GetPendingRetries returns information about pending retries
func (rp *RetryProcessor) GetPendingRetries(ctx context.Context, limit int64) ([]PendingRetry, error) {
	messages, err := rp.redis.Client.ZRangeByScoreWithScores(ctx, RetryQueue, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   "+inf",
		Count: limit,
	}).Result()

	if err != nil {
		return nil, err
	}

	var retries []PendingRetry
	for _, msg := range messages {
		messageData := msg.Member.(string)
		retryTime := time.Unix(int64(msg.Score), 0)

		var deliveryMessage DeliveryMessage
		if err := deliveryMessage.FromJSON([]byte(messageData)); err != nil {
			continue // Skip invalid messages
		}

		retries = append(retries, PendingRetry{
			DeliveryID:    deliveryMessage.DeliveryID.String(),
			EndpointID:    deliveryMessage.EndpointID.String(),
			AttemptNumber: deliveryMessage.AttemptNumber,
			RetryAt:       retryTime,
		})
	}

	return retries, nil
}

// PendingRetry represents a pending retry message
type PendingRetry struct {
	DeliveryID    string    `json:"delivery_id"`
	EndpointID    string    `json:"endpoint_id"`
	AttemptNumber int       `json:"attempt_number"`
	RetryAt       time.Time `json:"retry_at"`
}

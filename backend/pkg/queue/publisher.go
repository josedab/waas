package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"webhook-platform/pkg/database"
)

// PublisherInterface defines the interface for publishing messages to queues
type PublisherInterface interface {
	PublishDelivery(ctx context.Context, message *DeliveryMessage) error
	PublishDelayedDelivery(ctx context.Context, message *DeliveryMessage, delay time.Duration) error
	PublishToDeadLetter(ctx context.Context, message *DeliveryMessage, reason string) error
	GetQueueLength(ctx context.Context, queueName string) (int64, error)
	GetQueueStats(ctx context.Context) (map[string]int64, error)
}

// Publisher handles publishing messages to Redis queues
type Publisher struct {
	redis *database.RedisClient
}

// NewPublisher creates a new queue publisher
func NewPublisher(redisClient *database.RedisClient) *Publisher {
	return &Publisher{
		redis: redisClient,
	}
}

// PublishDelivery publishes a delivery message to the main delivery queue
func (p *Publisher) PublishDelivery(ctx context.Context, message *DeliveryMessage) error {
	data, err := message.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize delivery message: %w", err)
	}

	// Use LPUSH to add to the left of the list (FIFO when using RPOP)
	err = p.redis.Client.LPush(ctx, DeliveryQueue, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish delivery message: %w", err)
	}

	return nil
}

// PublishDelayedDelivery publishes a delivery message with a delay (for retries)
func (p *Publisher) PublishDelayedDelivery(ctx context.Context, message *DeliveryMessage, delay time.Duration) error {
	data, err := message.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize delayed delivery message: %w", err)
	}

	// Use sorted set with score as Unix timestamp for delayed processing
	score := float64(time.Now().Add(delay).Unix())
	err = p.redis.Client.ZAdd(ctx, RetryQueue, redis.Z{
		Score:  score,
		Member: data,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to publish delayed delivery message: %w", err)
	}

	return nil
}

// PublishToDeadLetter publishes a message to the dead letter queue
func (p *Publisher) PublishToDeadLetter(ctx context.Context, message *DeliveryMessage, reason string) error {
	// Add metadata about why it went to DLQ
	dlqMessage := struct {
		*DeliveryMessage
		Reason    string    `json:"reason"`
		Timestamp time.Time `json:"timestamp"`
	}{
		DeliveryMessage: message,
		Reason:         reason,
		Timestamp:      time.Now(),
	}

	data, err := json.Marshal(dlqMessage)
	if err != nil {
		return fmt.Errorf("failed to serialize DLQ message: %w", err)
	}

	err = p.redis.Client.LPush(ctx, DeadLetterQueue, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish to dead letter queue: %w", err)
	}

	return nil
}

// GetQueueLength returns the length of a specific queue
func (p *Publisher) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	if queueName == RetryQueue {
		// For sorted sets, use ZCARD
		return p.redis.Client.ZCard(ctx, queueName).Result()
	}
	// For lists, use LLEN
	return p.redis.Client.LLen(ctx, queueName).Result()
}

// GetQueueStats returns statistics for all queues
func (p *Publisher) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)
	
	queues := []string{DeliveryQueue, DeadLetterQueue, ProcessingQueue}
	for _, queue := range queues {
		length, err := p.redis.Client.LLen(ctx, queue).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get length for queue %s: %w", queue, err)
		}
		stats[queue] = length
	}

	// Handle retry queue separately (sorted set)
	retryLength, err := p.redis.Client.ZCard(ctx, RetryQueue).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get length for retry queue: %w", err)
	}
	stats[RetryQueue] = retryLength

	return stats, nil
}
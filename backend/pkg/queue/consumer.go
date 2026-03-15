package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/utils"
	"github.com/redis/go-redis/v9"
)

// MessageHandler defines the interface for handling delivery messages
type MessageHandler interface {
	HandleDelivery(ctx context.Context, message *DeliveryMessage) (*DeliveryResult, error)
}

// Consumer handles consuming messages from Redis queues
type Consumer struct {
	redis          *database.RedisClient
	publisher      *Publisher
	handler        MessageHandler
	workers        int
	stopCh         chan struct{}
	wg             sync.WaitGroup
	retryProcessor *RetryProcessor
	logger         *utils.Logger
}

// NewConsumer creates a new queue consumer
func NewConsumer(redisClient *database.RedisClient, handler MessageHandler, workers int) *Consumer {
	publisher := NewPublisher(redisClient)
	logger := utils.NewLogger("queue-consumer")
	return &Consumer{
		redis:          redisClient,
		publisher:      publisher,
		handler:        handler,
		workers:        workers,
		stopCh:         make(chan struct{}),
		retryProcessor: NewRetryProcessor(redisClient, publisher),
		logger:         logger,
	}
}

// Start begins consuming messages with the specified number of workers
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting consumer", map[string]interface{}{"workers": c.workers})

	// Start retry processor
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.retryProcessor.Start(ctx)
	}()

	// Start worker goroutines
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go func(workerID int) {
			defer c.wg.Done()
			c.worker(ctx, workerID)
		}(i)
	}

	return nil
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	c.logger.Info("Stopping consumer", nil)
	close(c.stopCh)
	c.retryProcessor.Stop()
	c.wg.Wait()
	c.logger.Info("Consumer stopped", nil)
}

// worker processes messages from the delivery queue
func (c *Consumer) worker(ctx context.Context, workerID int) {
	c.logger.Info("Worker started", map[string]interface{}{"worker_id": workerID})
	defer c.logger.Info("Worker stopped", map[string]interface{}{"worker_id": workerID})

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			if err := c.safeProcessNextMessage(ctx, workerID); err != nil {
				c.logger.Error("Worker error", map[string]interface{}{"worker_id": workerID, "error": err.Error()})
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// safeProcessNextMessage wraps processNextMessage with panic recovery to
// prevent a single bad message from killing a worker goroutine permanently.
func (c *Consumer) safeProcessNextMessage(ctx context.Context, workerID int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered in worker %d: %v", workerID, r)
			c.logger.Error("Worker panic recovered", map[string]interface{}{
				"worker_id": workerID,
				"panic":     fmt.Sprintf("%v", r),
			})
		}
	}()
	return c.processNextMessage(ctx, workerID)
}

// processNextMessage processes the next available message
func (c *Consumer) processNextMessage(ctx context.Context, workerID int) error {
	// Use BRPOPLPUSH for reliable message processing
	// This atomically moves message from delivery queue to processing queue
	result, err := c.redis.Client.BRPopLPush(ctx, DeliveryQueue, ProcessingQueue, 1*time.Second).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// No message available, this is normal
			return nil
		}
		return fmt.Errorf("failed to pop message: %w", err)
	}

	// Parse the message
	var message DeliveryMessage
	if err := message.FromJSON([]byte(result)); err != nil {
		c.logger.Error("Failed to parse message; routing to DLQ", map[string]interface{}{
			"worker_id": workerID,
			"error":     err.Error(),
		})
		// Route unparseable message to DLQ for audit trail
		c.routeMalformedToDLQ(ctx, result, err)
		// Remove from processing queue
		if remErr := c.redis.Client.LRem(ctx, ProcessingQueue, 1, result).Err(); remErr != nil {
			c.logger.Error("Failed to remove malformed message from processing queue", map[string]interface{}{
				"worker_id": workerID,
				"error":     remErr.Error(),
			})
		}
		return nil
	}

	c.logger.Info("Processing delivery", map[string]interface{}{"worker_id": workerID, "delivery_id": message.DeliveryID.String(), "attempt": message.AttemptNumber})

	// Process the message
	deliveryResult, err := c.handler.HandleDelivery(ctx, &message)
	if err != nil {
		c.logger.Error("Handler error for delivery", map[string]interface{}{"worker_id": workerID, "delivery_id": message.DeliveryID.String(), "error": err.Error()})
		errMsg := err.Error()
		deliveryResult = &DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        StatusFailed,
			ErrorMessage:  &errMsg,
			AttemptNumber: message.AttemptNumber,
		}
	}

	// Handle the result
	if err := c.handleDeliveryResult(ctx, &message, deliveryResult); err != nil {
		c.logger.Error("Failed to handle delivery result", map[string]interface{}{"worker_id": workerID, "error": err.Error()})
	}

	// Remove message from processing queue
	if err := c.redis.Client.LRem(ctx, ProcessingQueue, 1, result).Err(); err != nil {
		c.logger.Error("Failed to remove message from processing queue", map[string]interface{}{
			"worker_id":   workerID,
			"delivery_id": message.DeliveryID.String(),
			"error":       err.Error(),
		})
	}

	return nil
}

// handleDeliveryResult processes the delivery result and decides next action
func (c *Consumer) handleDeliveryResult(ctx context.Context, message *DeliveryMessage, result *DeliveryResult) error {
	switch result.Status {
	case StatusSuccess:
		c.logger.Info("Delivery succeeded", map[string]interface{}{"delivery_id": message.DeliveryID.String(), "attempt": message.AttemptNumber})
		return nil

	case StatusFailed:
		if message.AttemptNumber >= message.MaxAttempts {
			// Max attempts reached, send to dead letter queue
			reason := "Max retry attempts exceeded"
			if result.ErrorMessage != nil {
				reason = fmt.Sprintf("Max retry attempts exceeded. Last error: %s", *result.ErrorMessage)
			}
			c.logger.Error("Delivery failed permanently", map[string]interface{}{"delivery_id": message.DeliveryID.String(), "attempts": message.AttemptNumber})
			return c.publisher.PublishToDeadLetter(ctx, message, reason)
		}

		// Schedule retry
		return c.scheduleRetry(ctx, message, result)

	case StatusRetrying:
		// Explicit retry request
		return c.scheduleRetry(ctx, message, result)

	default:
		return fmt.Errorf("unknown delivery status: %s", result.Status)
	}
}

// scheduleRetry schedules a message for retry with exponential backoff
func (c *Consumer) scheduleRetry(ctx context.Context, message *DeliveryMessage, result *DeliveryResult) error {
	message.AttemptNumber++

	// Calculate retry delay using exponential backoff
	delay := c.calculateRetryDelay(message.AttemptNumber)

	c.logger.Info("Scheduling retry", map[string]interface{}{"delivery_id": message.DeliveryID.String(), "attempt": message.AttemptNumber, "delay": delay.String()})

	return c.publisher.PublishDelayedDelivery(ctx, message, delay)
}

// calculateRetryDelay calculates the delay for retry using exponential backoff with jitter
func (c *Consumer) calculateRetryDelay(attemptNumber int) time.Duration {
	// Base delay: 1 second
	baseDelay := time.Second

	// Cap attempt number to prevent bit shift overflow (1<<63 would overflow int64)
	const maxShift = 18 // 2^18 seconds ~ 3 days, well above maxDelay
	shift := attemptNumber - 1
	if shift < 0 {
		shift = 0
	}
	if shift > maxShift {
		shift = maxShift
	}

	// Exponential backoff: 2^(attempt-1) * baseDelay
	delay := time.Duration(1<<uint(shift)) * baseDelay

	// Cap at 5 minutes
	maxDelay := 5 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter (±25%)
	jitter := time.Duration(float64(delay) * 0.25)
	jitterMultiplier := 2*rand.Float64() - 1
	delay = delay + time.Duration(float64(jitter)*jitterMultiplier)

	// Ensure delay doesn't exceed max after jitter
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// routeMalformedToDLQ sends an unparseable raw message to the dead-letter queue
// so it can be inspected later.
func (c *Consumer) routeMalformedToDLQ(ctx context.Context, rawMessage string, parseErr error) {
	entry := map[string]interface{}{
		"raw_message": rawMessage,
		"reason":      fmt.Sprintf("JSON parsing failed: %s", parseErr.Error()),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		c.logger.Error("Failed to marshal malformed DLQ entry", map[string]interface{}{"error": err.Error()})
		return
	}
	if err := c.redis.Client.LPush(ctx, DeadLetterQueue, data).Err(); err != nil {
		c.logger.Error("Failed to push malformed message to DLQ", map[string]interface{}{"error": err.Error()})
	}
}

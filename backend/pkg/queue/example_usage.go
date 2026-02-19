package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/utils"
)

// ExampleDeliveryHandler demonstrates how to implement a MessageHandler
type ExampleDeliveryHandler struct {
	// Add any dependencies like HTTP client, database, etc.
	logger *utils.Logger
}

// HandleDelivery implements the MessageHandler interface
func (h *ExampleDeliveryHandler) HandleDelivery(ctx context.Context, message *DeliveryMessage) (*DeliveryResult, error) {
	h.logger.Info("Processing delivery", map[string]interface{}{
		"delivery_id": message.DeliveryID.String(),
		"endpoint_id": message.EndpointID.String(),
		"attempt":     message.AttemptNumber,
	})

	// Here you would implement the actual webhook delivery logic:
	// 1. Fetch endpoint details from database
	// 2. Make HTTP request to the webhook URL
	// 3. Handle response and errors

	// For this example, we'll simulate different scenarios

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Example: simulate failure on first attempt, success on retry
	if message.AttemptNumber == 1 {
		errorMsg := "Simulated temporary failure"
		return &DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        StatusFailed,
			ErrorMessage:  &errorMsg,
			AttemptNumber: message.AttemptNumber,
		}, nil
	}

	// Simulate successful delivery
	now := time.Now()
	httpStatus := 200
	responseBody := `{"received": true}`

	return &DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        StatusSuccess,
		HTTPStatus:    &httpStatus,
		ResponseBody:  &responseBody,
		DeliveredAt:   &now,
		AttemptNumber: message.AttemptNumber,
	}, nil
}

// ExampleUsage demonstrates how to use the queue system
func ExampleUsage() {
	// 1. Set up Redis connection
	redisURL := "redis://localhost:6379/0"
	redis, err := database.NewRedisConnection(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	logger := utils.NewLogger("queue-example")

	// 2. Create message handler
	handler := &ExampleDeliveryHandler{logger: logger}

	// 3. Create queue manager with 3 worker goroutines
	manager := NewManager(redis, handler, 3)

	// 4. Start the queue processing
	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}
	defer manager.Stop()

	// 5. Publish some webhook delivery messages
	for i := 0; i < 5; i++ {
		message := &DeliveryMessage{
			DeliveryID:    uuid.New(),
			EndpointID:    uuid.New(),
			TenantID:      uuid.New(),
			Payload:       json.RawMessage(fmt.Sprintf(`{"event": "test", "id": %d}`, i)),
			Headers:       map[string]string{"Content-Type": "application/json"},
			AttemptNumber: 1,
			MaxAttempts:   3,
			ScheduledAt:   time.Now(),
			Signature:     "sha256=example-signature",
		}

		if err := manager.PublishDelivery(ctx, message); err != nil {
			logger.Error("Failed to publish message", map[string]interface{}{"error": err.Error()})
		} else {
			logger.Info("Published delivery", map[string]interface{}{"delivery_id": message.DeliveryID.String()})
		}
	}

	// 6. Monitor queue statistics
	time.Sleep(2 * time.Second) // Give some time for processing

	stats, err := manager.GetQueueStats(ctx)
	if err != nil {
		logger.Error("Failed to get queue stats", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Info("Queue statistics", nil)
		for queue, count := range stats {
			logger.Info("Queue stat", map[string]interface{}{"queue": queue, "count": count})
		}
	}

	// 7. Check pending retries
	retries, err := manager.GetPendingRetries(ctx, 10)
	if err != nil {
		logger.Error("Failed to get pending retries", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Info("Pending retries", map[string]interface{}{"count": len(retries)})
		for _, retry := range retries {
			logger.Info("Pending retry", map[string]interface{}{
				"delivery_id": retry.DeliveryID,
				"retry_at":    retry.RetryAt.Format(time.RFC3339),
			})
		}
	}

	// 8. Health check
	if err := manager.HealthCheck(ctx); err != nil {
		logger.Error("Health check failed", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Info("Queue system is healthy", nil)
	}
}

// ExamplePublisherOnly demonstrates using just the publisher for enqueueing messages
func ExamplePublisherOnly() {
	redisURL := "redis://localhost:6379/0"
	redis, err := database.NewRedisConnection(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	publisher := NewPublisher(redis)
	logger := utils.NewLogger("queue-example")
	ctx := context.Background()

	// Publish immediate delivery
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"event": "user.created", "user_id": 123}`),
		Headers:       map[string]string{"Content-Type": "application/json"},
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	if err := publisher.PublishDelivery(ctx, message); err != nil {
		logger.Error("Failed to publish delivery", map[string]interface{}{"error": err.Error()})
	}

	// Publish delayed delivery (for retry)
	if err := publisher.PublishDelayedDelivery(ctx, message, 5*time.Minute); err != nil {
		logger.Error("Failed to publish delayed delivery", map[string]interface{}{"error": err.Error()})
	}

	// Send to dead letter queue
	if err := publisher.PublishToDeadLetter(ctx, message, "Max retries exceeded"); err != nil {
		logger.Error("Failed to publish to DLQ", map[string]interface{}{"error": err.Error()})
	}

	// Get queue statistics
	stats, err := publisher.GetQueueStats(ctx)
	if err != nil {
		logger.Error("Failed to get stats", map[string]interface{}{"error": err.Error()})
	} else {
		for queue, count := range stats {
			logger.Info("Queue stat", map[string]interface{}{"queue": queue, "count": count})
		}
	}
}

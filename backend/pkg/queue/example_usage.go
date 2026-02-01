package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/database"
)

// ExampleDeliveryHandler demonstrates how to implement a MessageHandler
type ExampleDeliveryHandler struct {
	// Add any dependencies like HTTP client, database, etc.
}

// HandleDelivery implements the MessageHandler interface
func (h *ExampleDeliveryHandler) HandleDelivery(ctx context.Context, message *DeliveryMessage) (*DeliveryResult, error) {
	log.Printf("Processing delivery %s to endpoint %s (attempt %d)", 
		message.DeliveryID, message.EndpointID, message.AttemptNumber)

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

	// 2. Create message handler
	handler := &ExampleDeliveryHandler{}

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
			log.Printf("Failed to publish message: %v", err)
		} else {
			log.Printf("Published delivery %s", message.DeliveryID)
		}
	}

	// 6. Monitor queue statistics
	time.Sleep(2 * time.Second) // Give some time for processing
	
	stats, err := manager.GetQueueStats(ctx)
	if err != nil {
		log.Printf("Failed to get queue stats: %v", err)
	} else {
		log.Printf("Queue statistics:")
		for queue, count := range stats {
			log.Printf("  %s: %d messages", queue, count)
		}
	}

	// 7. Check pending retries
	retries, err := manager.GetPendingRetries(ctx, 10)
	if err != nil {
		log.Printf("Failed to get pending retries: %v", err)
	} else {
		log.Printf("Pending retries: %d", len(retries))
		for _, retry := range retries {
			log.Printf("  Delivery %s scheduled for retry at %s", 
				retry.DeliveryID, retry.RetryAt.Format(time.RFC3339))
		}
	}

	// 8. Health check
	if err := manager.HealthCheck(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		log.Println("Queue system is healthy")
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
		log.Printf("Failed to publish delivery: %v", err)
	}

	// Publish delayed delivery (for retry)
	if err := publisher.PublishDelayedDelivery(ctx, message, 5*time.Minute); err != nil {
		log.Printf("Failed to publish delayed delivery: %v", err)
	}

	// Send to dead letter queue
	if err := publisher.PublishToDeadLetter(ctx, message, "Max retries exceeded"); err != nil {
		log.Printf("Failed to publish to DLQ: %v", err)
	}

	// Get queue statistics
	stats, err := publisher.GetQueueStats(ctx)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		for queue, count := range stats {
			log.Printf("%s: %d messages", queue, count)
		}
	}
}
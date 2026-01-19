package queue

import (
	"context"
	"time"
)

// TestPublisher is a mock publisher for testing
type TestPublisher struct {
	messages []interface{}
}

// NewTestPublisher creates a new test publisher
func NewTestPublisher() *TestPublisher {
	return &TestPublisher{
		messages: make([]interface{}, 0),
	}
}

// PublishDelivery publishes a delivery message (mock implementation)
func (p *TestPublisher) PublishDelivery(ctx context.Context, message *DeliveryMessage) error {
	p.messages = append(p.messages, message)
	return nil
}

// PublishDelayedDelivery publishes a delayed delivery message (mock implementation)
func (p *TestPublisher) PublishDelayedDelivery(ctx context.Context, message *DeliveryMessage, delay time.Duration) error {
	p.messages = append(p.messages, message)
	return nil
}

// PublishToDeadLetter publishes to dead letter queue (mock implementation)
func (p *TestPublisher) PublishToDeadLetter(ctx context.Context, message *DeliveryMessage, reason string) error {
	p.messages = append(p.messages, message)
	return nil
}

// GetQueueLength returns mock queue length
func (p *TestPublisher) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	return int64(len(p.messages)), nil
}

// GetQueueStats returns mock queue statistics
func (p *TestPublisher) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return map[string]int64{
		"delivery": int64(len(p.messages)),
		"retry":    0,
		"dlq":      0,
	}, nil
}

// GetMessages returns all published messages for testing
func (p *TestPublisher) GetMessages() []interface{} {
	return p.messages
}

// ClearMessages clears all published messages
func (p *TestPublisher) ClearMessages() {
	p.messages = make([]interface{}, 0)
}

// Reset clears all published messages (alias for ClearMessages)
func (p *TestPublisher) Reset() {
	p.ClearMessages()
}

// GetDeliveryMessages returns all delivery messages for testing
func (p *TestPublisher) GetDeliveryMessages() []DeliveryMessage {
	var deliveryMessages []DeliveryMessage
	for _, msg := range p.messages {
		if dm, ok := msg.(*DeliveryMessage); ok {
			deliveryMessages = append(deliveryMessages, *dm)
		}
	}
	return deliveryMessages
}
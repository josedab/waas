package queue

import (
	"context"
	"fmt"
	"sync"

	"github.com/josedab/waas/pkg/database"
)

// Manager provides a high-level interface for queue operations
type Manager struct {
	redis     *database.RedisClient
	publisher *Publisher
	consumer  *Consumer
	mu        sync.RWMutex
	running   bool
}

// NewManager creates a new queue manager
func NewManager(redisClient *database.RedisClient, handler MessageHandler, workers int) *Manager {
	publisher := NewPublisher(redisClient)
	consumer := NewConsumer(redisClient, handler, workers)
	
	return &Manager{
		redis:     redisClient,
		publisher: publisher,
		consumer:  consumer,
	}
}

// Start starts the queue processing
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		return fmt.Errorf("queue manager is already running")
	}
	
	if err := m.consumer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}
	
	m.running = true
	return nil
}

// Stop stops the queue processing
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.running {
		return nil
	}
	
	m.consumer.Stop()
	m.running = false
	return nil
}

// IsRunning returns whether the queue manager is running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// PublishDelivery publishes a delivery message
func (m *Manager) PublishDelivery(ctx context.Context, message *DeliveryMessage) error {
	return m.publisher.PublishDelivery(ctx, message)
}

// GetQueueStats returns statistics for all queues
func (m *Manager) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return m.publisher.GetQueueStats(ctx)
}

// GetPendingRetries returns information about pending retries
func (m *Manager) GetPendingRetries(ctx context.Context, limit int64) ([]PendingRetry, error) {
	return m.consumer.retryProcessor.GetPendingRetries(ctx, limit)
}

// HealthCheck performs a health check on the queue system
func (m *Manager) HealthCheck(ctx context.Context) error {
	// Check Redis connection
	if err := m.redis.HealthCheck(ctx); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	
	// Check if we can get queue stats
	_, err := m.GetQueueStats(ctx)
	if err != nil {
		return fmt.Errorf("queue stats check failed: %w", err)
	}
	
	return nil
}

// PurgeQueue removes all messages from a specific queue (for testing/admin)
func (m *Manager) PurgeQueue(ctx context.Context, queueName string) error {
	if queueName == RetryQueue {
		// For sorted sets, use DEL
		return m.redis.Client.Del(ctx, queueName).Err()
	}
	// For lists, use DEL
	return m.redis.Client.Del(ctx, queueName).Err()
}

// PurgeAllQueues removes all messages from all queues (for testing/admin)
func (m *Manager) PurgeAllQueues(ctx context.Context) error {
	queues := []string{DeliveryQueue, DeadLetterQueue, RetryQueue, ProcessingQueue}
	for _, queue := range queues {
		if err := m.PurgeQueue(ctx, queue); err != nil {
			return fmt.Errorf("failed to purge queue %s: %w", queue, err)
		}
	}
	return nil
}
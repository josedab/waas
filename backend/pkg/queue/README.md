# Queue Package

This package provides a Redis-based message queue system for webhook delivery with the following features:

- **Reliable message processing** with Redis BRPOPLPUSH for atomic operations
- **Retry logic** with exponential backoff and jitter
- **Dead letter queue** for failed messages
- **Concurrent processing** with configurable worker goroutines
- **Delayed message processing** for retries using Redis sorted sets
- **Queue monitoring** and statistics

## Architecture

The queue system consists of several components:

- **Publisher**: Publishes messages to queues
- **Consumer**: Processes messages with worker goroutines
- **RetryProcessor**: Handles delayed retry messages
- **Manager**: High-level interface that coordinates all components

## Queue Types

1. **Delivery Queue** (`webhook:delivery`): Main queue for immediate processing
2. **Retry Queue** (`webhook:retry`): Sorted set for delayed retry messages
3. **Processing Queue** (`webhook:processing`): Temporary queue for messages being processed
4. **Dead Letter Queue** (`webhook:dlq`): Failed messages that exceeded max retries

## Usage

### Basic Setup

```go
import (
    "webhook-platform/pkg/database"
    "webhook-platform/pkg/queue"
)

// Connect to Redis
redis, err := database.NewRedisConnection("redis://localhost:6379/0")
if err != nil {
    log.Fatal(err)
}

// Implement message handler
type MyHandler struct{}

func (h *MyHandler) HandleDelivery(ctx context.Context, message *queue.DeliveryMessage) (*queue.DeliveryResult, error) {
    // Your webhook delivery logic here
    // Return success/failure result
}

// Create and start queue manager
handler := &MyHandler{}
manager := queue.NewManager(redis, handler, 3) // 3 workers

ctx := context.Background()
manager.Start(ctx)
defer manager.Stop()
```

### Publishing Messages

```go
message := &queue.DeliveryMessage{
    DeliveryID:    uuid.New(),
    EndpointID:    uuid.New(),
    TenantID:      uuid.New(),
    Payload:       json.RawMessage(`{"event": "user.created"}`),
    Headers:       map[string]string{"Content-Type": "application/json"},
    AttemptNumber: 1,
    MaxAttempts:   3,
    ScheduledAt:   time.Now(),
    Signature:     "sha256=...",
}

err := manager.PublishDelivery(ctx, message)
```

### Monitoring

```go
// Get queue statistics
stats, err := manager.GetQueueStats(ctx)
for queue, count := range stats {
    fmt.Printf("%s: %d messages\n", queue, count)
}

// Get pending retries
retries, err := manager.GetPendingRetries(ctx, 10)
for _, retry := range retries {
    fmt.Printf("Delivery %s retrying at %s\n", retry.DeliveryID, retry.RetryAt)
}

// Health check
err := manager.HealthCheck(ctx)
```

## Message Handler Interface

Implement the `MessageHandler` interface to process webhook deliveries:

```go
type MessageHandler interface {
    HandleDelivery(ctx context.Context, message *DeliveryMessage) (*DeliveryResult, error)
}
```

Return appropriate `DeliveryResult` status:
- `StatusSuccess`: Message processed successfully
- `StatusFailed`: Message failed, will retry if attempts remaining
- `StatusRetrying`: Explicit retry request

## Retry Logic

The system implements exponential backoff with jitter:
- Base delay: 1 second
- Exponential multiplier: 2^(attempt-1)
- Maximum delay: 5 minutes
- Jitter: ±25% to prevent thundering herd

## Error Handling

- **Temporary failures**: Automatically retried with exponential backoff
- **Permanent failures**: Sent to dead letter queue after max attempts
- **Invalid messages**: Logged and discarded
- **Processing failures**: Messages returned to queue for retry

## Configuration

Key configuration options:
- **Workers**: Number of concurrent processing goroutines
- **Max Attempts**: Maximum retry attempts per message (default: 3)
- **Redis URL**: Redis connection string
- **Queue Names**: Customizable queue prefixes

## Testing

The package includes comprehensive tests:
- **Unit tests**: Core logic without Redis dependency
- **Integration tests**: Full workflow with Redis (requires running Redis)

Run unit tests:
```bash
go test ./pkg/queue -v -run "^Test.*(?:Message|Consumer|Retry|Delivery).*(?:JSON|Validation|Delay|Statuses)$"
```

Run integration tests (requires Redis):
```bash
go test ./pkg/queue -v
```

## Performance Considerations

- Use connection pooling for Redis
- Configure appropriate number of workers based on load
- Monitor queue depths to detect bottlenecks
- Use Redis clustering for high availability
- Consider message size limits for large payloads

## Security

- Messages may contain sensitive webhook payloads
- Use Redis AUTH and TLS in production
- Consider encrypting message payloads at rest
- Implement proper access controls for queue management

## Dependencies

- `github.com/redis/go-redis/v9`: Redis client
- `github.com/google/uuid`: UUID generation
- `github.com/stretchr/testify`: Testing framework
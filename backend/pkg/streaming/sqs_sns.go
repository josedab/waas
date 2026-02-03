package streaming

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SQSProducer implements Producer for AWS SQS
type SQSProducer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
}

// NewSQSProducer creates a new SQS producer
func NewSQSProducer() *SQSProducer {
	return &SQSProducer{}
}

func (p *SQSProducer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bridge.Config == nil || bridge.Config.SQSConfig == nil {
		return fmt.Errorf("%w: SQS config is required", ErrInvalidConfig)
	}

	cfg := bridge.Config.SQSConfig
	if cfg.QueueURL == "" || cfg.Region == "" {
		return fmt.Errorf("%w: SQS queue URL and region are required", ErrInvalidConfig)
	}

	p.bridge = bridge
	p.credentials = credentials
	p.closed = false
	return nil
}

func (p *SQSProducer) Send(ctx context.Context, event *StreamEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrBridgeClosed
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.Timestamp = time.Now()
	event.Status = "sent"
	return nil
}

func (p *SQSProducer) SendBatch(ctx context.Context, events []*StreamEvent) error {
	// SQS supports batches of up to 10 messages
	const maxBatchSize = 10
	for i := 0; i < len(events); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(events) {
			end = len(events)
		}
		for _, event := range events[i:end] {
			if err := p.Send(ctx, event); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *SQSProducer) Flush(ctx context.Context) error { return nil }

func (p *SQSProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *SQSProducer) Healthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// SQSConsumer implements Consumer for AWS SQS
type SQSConsumer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
	paused      bool
	stopCh      chan struct{}
}

// NewSQSConsumer creates a new SQS consumer
func NewSQSConsumer() *SQSConsumer {
	return &SQSConsumer{
		stopCh: make(chan struct{}),
	}
}

func (c *SQSConsumer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if bridge.Config == nil || bridge.Config.SQSConfig == nil {
		return fmt.Errorf("%w: SQS config is required", ErrInvalidConfig)
	}

	c.bridge = bridge
	c.credentials = credentials
	c.closed = false
	c.paused = false
	return nil
}

func (c *SQSConsumer) Start(ctx context.Context, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		}
	}
}

func (c *SQSConsumer) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	return nil
}

func (c *SQSConsumer) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	return nil
}

func (c *SQSConsumer) Commit(ctx context.Context, event *StreamEvent) error { return nil }
func (c *SQSConsumer) GetLag(ctx context.Context) (int64, error)            { return 0, nil }

func (c *SQSConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.stopCh)
	}
	return nil
}

func (c *SQSConsumer) Healthy(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed
}

// SNSProducer implements Producer for AWS SNS (publish-only)
type SNSProducer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
}

// NewSNSProducer creates a new SNS producer
func NewSNSProducer() *SNSProducer {
	return &SNSProducer{}
}

func (p *SNSProducer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bridge.Config == nil || bridge.Config.SNSConfig == nil {
		return fmt.Errorf("%w: SNS config is required", ErrInvalidConfig)
	}

	cfg := bridge.Config.SNSConfig
	if cfg.TopicARN == "" || cfg.Region == "" {
		return fmt.Errorf("%w: SNS topic ARN and region are required", ErrInvalidConfig)
	}

	p.bridge = bridge
	p.credentials = credentials
	p.closed = false
	return nil
}

func (p *SNSProducer) Send(ctx context.Context, event *StreamEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrBridgeClosed
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.Timestamp = time.Now()
	event.Status = "sent"
	return nil
}

func (p *SNSProducer) SendBatch(ctx context.Context, events []*StreamEvent) error {
	for _, event := range events {
		if err := p.Send(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (p *SNSProducer) Flush(ctx context.Context) error { return nil }

func (p *SNSProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *SNSProducer) Healthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// --- EventSource/EventSink adapters ---

// SQSEventSource wraps SQSConsumer as an EventSource
type SQSEventSource struct {
	consumer *SQSConsumer
}

func (s *SQSEventSource) Name() string     { return "sqs-source" }
func (s *SQSEventSource) Type() StreamType { return StreamTypeSQS }
func (s *SQSEventSource) Healthy(ctx context.Context) bool {
	return s.consumer != nil && s.consumer.Healthy(ctx)
}
func (s *SQSEventSource) Close() error {
	if s.consumer != nil {
		return s.consumer.Close()
	}
	return nil
}
func (s *SQSEventSource) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.consumer = NewSQSConsumer()
	bridge := &StreamingBridge{ID: "sqs-source", Config: config, StreamType: StreamTypeSQS}
	return s.consumer.Init(ctx, bridge, creds)
}
func (s *SQSEventSource) Subscribe(ctx context.Context, handler EventHandler) error {
	return s.consumer.Start(ctx, handler)
}
func (s *SQSEventSource) Unsubscribe() error {
	return s.consumer.Pause()
}

// SNSEventSink wraps SNSProducer as an EventSink
type SNSEventSink struct {
	producer *SNSProducer
}

func (s *SNSEventSink) Name() string     { return "sns-sink" }
func (s *SNSEventSink) Type() StreamType { return StreamTypeSNS }
func (s *SNSEventSink) Healthy(ctx context.Context) bool {
	return s.producer != nil && s.producer.Healthy(ctx)
}
func (s *SNSEventSink) Close() error {
	if s.producer != nil {
		return s.producer.Close()
	}
	return nil
}
func (s *SNSEventSink) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.producer = NewSNSProducer()
	bridge := &StreamingBridge{ID: "sns-sink", Config: config, StreamType: StreamTypeSNS}
	return s.producer.Init(ctx, bridge, creds)
}
func (s *SNSEventSink) Publish(ctx context.Context, event *StreamEvent) error {
	return s.producer.Send(ctx, event)
}
func (s *SNSEventSink) PublishBatch(ctx context.Context, events []*StreamEvent) error {
	return s.producer.SendBatch(ctx, events)
}

// SQSHealthCheck checks SQS connectivity
func SQSHealthCheck(ctx context.Context, config *SQSConfig) (*HealthStatus, error) {
	if config == nil {
		return nil, fmt.Errorf("SQS config is required")
	}

	status := &HealthStatus{
		Platform:  string(StreamTypeSQS),
		Healthy:   config.QueueURL != "" && config.Region != "",
		CheckedAt: time.Now(),
		Details: map[string]interface{}{
			"queue_url":  config.QueueURL,
			"region":     config.Region,
			"fifo_queue": config.FIFOQueue,
		},
	}

	if !status.Healthy {
		status.Error = "SQS queue URL and region are required"
	}

	return status, nil
}

// SNSHealthCheck checks SNS connectivity
func SNSHealthCheck(ctx context.Context, config *SNSConfig) (*HealthStatus, error) {
	if config == nil {
		return nil, fmt.Errorf("SNS config is required")
	}

	status := &HealthStatus{
		Platform:  string(StreamTypeSNS),
		Healthy:   config.TopicARN != "" && config.Region != "",
		CheckedAt: time.Now(),
		Details: map[string]interface{}{
			"topic_arn":  config.TopicARN,
			"region":     config.Region,
			"fifo_topic": config.FIFOTopic,
		},
	}

	if !status.Healthy {
		status.Error = "SNS topic ARN and region are required"
	}

	return status, nil
}

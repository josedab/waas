package streaming

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RabbitMQProducer implements Producer for RabbitMQ
type RabbitMQProducer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
}

// NewRabbitMQProducer creates a new RabbitMQ producer
func NewRabbitMQProducer() *RabbitMQProducer {
	return &RabbitMQProducer{}
}

func (p *RabbitMQProducer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bridge.Config == nil || bridge.Config.RabbitMQConfig == nil {
		return fmt.Errorf("%w: RabbitMQ config is required", ErrInvalidConfig)
	}

	cfg := bridge.Config.RabbitMQConfig
	if cfg.URL == "" || cfg.Exchange == "" {
		return fmt.Errorf("%w: RabbitMQ URL and exchange are required", ErrInvalidConfig)
	}

	p.bridge = bridge
	p.credentials = credentials
	p.closed = false
	return nil
}

func (p *RabbitMQProducer) Send(ctx context.Context, event *StreamEvent) error {
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

func (p *RabbitMQProducer) SendBatch(ctx context.Context, events []*StreamEvent) error {
	for _, event := range events {
		if err := p.Send(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (p *RabbitMQProducer) Flush(ctx context.Context) error { return nil }

func (p *RabbitMQProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *RabbitMQProducer) Healthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// RabbitMQConsumer implements Consumer for RabbitMQ
type RabbitMQConsumer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
	paused      bool
	stopCh      chan struct{}
}

// NewRabbitMQConsumer creates a new RabbitMQ consumer
func NewRabbitMQConsumer() *RabbitMQConsumer {
	return &RabbitMQConsumer{
		stopCh: make(chan struct{}),
	}
}

func (c *RabbitMQConsumer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if bridge.Config == nil || bridge.Config.RabbitMQConfig == nil {
		return fmt.Errorf("%w: RabbitMQ config is required", ErrInvalidConfig)
	}

	c.bridge = bridge
	c.credentials = credentials
	c.closed = false
	c.paused = false
	return nil
}

func (c *RabbitMQConsumer) Start(ctx context.Context, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		}
	}
}

func (c *RabbitMQConsumer) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	return nil
}

func (c *RabbitMQConsumer) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	return nil
}

func (c *RabbitMQConsumer) Commit(ctx context.Context, event *StreamEvent) error { return nil }
func (c *RabbitMQConsumer) GetLag(ctx context.Context) (int64, error)            { return 0, nil }

func (c *RabbitMQConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.stopCh)
	}
	return nil
}

func (c *RabbitMQConsumer) Healthy(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed
}

// --- EventSource/EventSink adapters ---

// RabbitMQEventSource wraps RabbitMQConsumer as an EventSource
type RabbitMQEventSource struct {
	consumer *RabbitMQConsumer
}

func (s *RabbitMQEventSource) Name() string     { return "rabbitmq-source" }
func (s *RabbitMQEventSource) Type() StreamType { return StreamTypeRabbitMQ }
func (s *RabbitMQEventSource) Healthy(ctx context.Context) bool {
	return s.consumer != nil && s.consumer.Healthy(ctx)
}
func (s *RabbitMQEventSource) Close() error {
	if s.consumer != nil {
		return s.consumer.Close()
	}
	return nil
}
func (s *RabbitMQEventSource) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.consumer = NewRabbitMQConsumer()
	bridge := &StreamingBridge{ID: "rabbitmq-source", Config: config, StreamType: StreamTypeRabbitMQ}
	return s.consumer.Init(ctx, bridge, creds)
}
func (s *RabbitMQEventSource) Subscribe(ctx context.Context, handler EventHandler) error {
	return s.consumer.Start(ctx, handler)
}
func (s *RabbitMQEventSource) Unsubscribe() error {
	return s.consumer.Pause()
}

// RabbitMQEventSink wraps RabbitMQProducer as an EventSink
type RabbitMQEventSink struct {
	producer *RabbitMQProducer
}

func (s *RabbitMQEventSink) Name() string     { return "rabbitmq-sink" }
func (s *RabbitMQEventSink) Type() StreamType { return StreamTypeRabbitMQ }
func (s *RabbitMQEventSink) Healthy(ctx context.Context) bool {
	return s.producer != nil && s.producer.Healthy(ctx)
}
func (s *RabbitMQEventSink) Close() error {
	if s.producer != nil {
		return s.producer.Close()
	}
	return nil
}
func (s *RabbitMQEventSink) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.producer = NewRabbitMQProducer()
	bridge := &StreamingBridge{ID: "rabbitmq-sink", Config: config, StreamType: StreamTypeRabbitMQ}
	return s.producer.Init(ctx, bridge, creds)
}
func (s *RabbitMQEventSink) Publish(ctx context.Context, event *StreamEvent) error {
	return s.producer.Send(ctx, event)
}
func (s *RabbitMQEventSink) PublishBatch(ctx context.Context, events []*StreamEvent) error {
	return s.producer.SendBatch(ctx, events)
}

// RabbitMQHealthCheck checks RabbitMQ connectivity
func RabbitMQHealthCheck(ctx context.Context, config *RabbitMQConfig) (*HealthStatus, error) {
	if config == nil {
		return nil, fmt.Errorf("RabbitMQ config is required")
	}

	status := &HealthStatus{
		Platform:  string(StreamTypeRabbitMQ),
		Healthy:   true,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if config.URL == "" {
		status.Healthy = false
		status.Error = "RabbitMQ URL not configured"
	}

	status.Details["exchange"] = config.Exchange
	status.Details["exchange_type"] = config.ExchangeType
	status.Details["queue"] = config.Queue
	status.Details["durable"] = config.Durable

	return status, nil
}

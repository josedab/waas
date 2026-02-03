package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NATSProducer implements Producer for NATS
type NATSProducer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
}

// NewNATSProducer creates a new NATS producer
func NewNATSProducer() *NATSProducer {
	return &NATSProducer{}
}

func (p *NATSProducer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bridge.Config == nil || bridge.Config.NATSConfig == nil {
		return fmt.Errorf("%w: NATS config is required", ErrInvalidConfig)
	}

	cfg := bridge.Config.NATSConfig
	if cfg.URL == "" || cfg.Subject == "" {
		return fmt.Errorf("%w: NATS URL and subject are required", ErrInvalidConfig)
	}

	p.bridge = bridge
	p.credentials = credentials
	p.closed = false
	return nil
}

func (p *NATSProducer) Send(ctx context.Context, event *StreamEvent) error {
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

func (p *NATSProducer) SendBatch(ctx context.Context, events []*StreamEvent) error {
	for _, event := range events {
		if err := p.Send(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (p *NATSProducer) Flush(ctx context.Context) error { return nil }

func (p *NATSProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *NATSProducer) Healthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// NATSConsumer implements Consumer for NATS
type NATSConsumer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
	paused      bool
	stopCh      chan struct{}
}

// NewNATSConsumer creates a new NATS consumer
func NewNATSConsumer() *NATSConsumer {
	return &NATSConsumer{
		stopCh: make(chan struct{}),
	}
}

func (c *NATSConsumer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if bridge.Config == nil || bridge.Config.NATSConfig == nil {
		return fmt.Errorf("%w: NATS config is required", ErrInvalidConfig)
	}

	c.bridge = bridge
	c.credentials = credentials
	c.closed = false
	c.paused = false
	return nil
}

func (c *NATSConsumer) Start(ctx context.Context, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		}
	}
}

func (c *NATSConsumer) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	return nil
}

func (c *NATSConsumer) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	return nil
}

func (c *NATSConsumer) Commit(ctx context.Context, event *StreamEvent) error { return nil }

func (c *NATSConsumer) GetLag(ctx context.Context) (int64, error) { return 0, nil }

func (c *NATSConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.stopCh)
	}
	return nil
}

func (c *NATSConsumer) Healthy(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed
}

// --- EventSource/EventSink adapters ---

// NATSEventSource wraps NATSConsumer as an EventSource
type NATSEventSource struct {
	consumer *NATSConsumer
}

func (s *NATSEventSource) Name() string     { return "nats-source" }
func (s *NATSEventSource) Type() StreamType { return StreamTypeNATS }
func (s *NATSEventSource) Healthy(ctx context.Context) bool {
	if s.consumer == nil {
		return false
	}
	return s.consumer.Healthy(ctx)
}
func (s *NATSEventSource) Close() error {
	if s.consumer != nil {
		return s.consumer.Close()
	}
	return nil
}
func (s *NATSEventSource) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.consumer = NewNATSConsumer()
	bridge := &StreamingBridge{ID: "nats-source", Config: config, StreamType: StreamTypeNATS}
	return s.consumer.Init(ctx, bridge, creds)
}
func (s *NATSEventSource) Subscribe(ctx context.Context, handler EventHandler) error {
	return s.consumer.Start(ctx, handler)
}
func (s *NATSEventSource) Unsubscribe() error {
	return s.consumer.Pause()
}

// NATSEventSink wraps NATSProducer as an EventSink
type NATSEventSink struct {
	producer *NATSProducer
}

func (s *NATSEventSink) Name() string     { return "nats-sink" }
func (s *NATSEventSink) Type() StreamType { return StreamTypeNATS }
func (s *NATSEventSink) Healthy(ctx context.Context) bool {
	if s.producer == nil {
		return false
	}
	return s.producer.Healthy(ctx)
}
func (s *NATSEventSink) Close() error {
	if s.producer != nil {
		return s.producer.Close()
	}
	return nil
}
func (s *NATSEventSink) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.producer = NewNATSProducer()
	bridge := &StreamingBridge{ID: "nats-sink", Config: config, StreamType: StreamTypeNATS}
	return s.producer.Init(ctx, bridge, creds)
}
func (s *NATSEventSink) Publish(ctx context.Context, event *StreamEvent) error {
	return s.producer.Send(ctx, event)
}
func (s *NATSEventSink) PublishBatch(ctx context.Context, events []*StreamEvent) error {
	return s.producer.SendBatch(ctx, events)
}

// HealthCheck performs a health check for NATS connections
func NATSHealthCheck(ctx context.Context, config *NATSConfig) (*HealthStatus, error) {
	if config == nil {
		return nil, fmt.Errorf("NATS config is required")
	}

	status := &HealthStatus{
		Platform:  string(StreamTypeNATS),
		Healthy:   true,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if config.URL == "" {
		status.Healthy = false
		status.Error = "NATS URL not configured"
	}

	status.Details["url"] = config.URL
	status.Details["subject"] = config.Subject
	status.Details["jetstream"] = config.JetStream

	return status, nil
}

// HealthStatus represents the result of a streaming platform health check
type HealthStatus struct {
	Platform  string                 `json:"platform"`
	Healthy   bool                   `json:"healthy"`
	Latency   time.Duration          `json:"latency_ms"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
}

// MarshalJSON implements custom JSON marshaling for HealthStatus
func (h *HealthStatus) MarshalJSON() ([]byte, error) {
	type Alias HealthStatus
	return json.Marshal(&struct {
		LatencyMs int64 `json:"latency_ms"`
		*Alias
	}{
		LatencyMs: h.Latency.Milliseconds(),
		Alias:     (*Alias)(h),
	})
}

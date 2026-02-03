package streaming

import (
	"context"
	"encoding/json"
)

// EventSource defines a generic interface for consuming events from external systems.
// Implementations wrap platform-specific consumers (Kafka, NATS, RabbitMQ, SQS).
type EventSource interface {
	// Name returns the event source identifier
	Name() string
	// Type returns the streaming platform type
	Type() StreamType
	// Connect establishes the connection to the event source
	Connect(ctx context.Context, config *BridgeConfig, credentials map[string]string) error
	// Subscribe starts receiving events, calling handler for each one
	Subscribe(ctx context.Context, handler EventHandler) error
	// Unsubscribe stops receiving events
	Unsubscribe() error
	// Healthy checks if the connection is alive
	Healthy(ctx context.Context) bool
	// Close shuts down the event source
	Close() error
}

// EventSink defines a generic interface for producing events to external systems.
// Implementations wrap platform-specific producers (Kafka, NATS, RabbitMQ, SNS).
type EventSink interface {
	// Name returns the event sink identifier
	Name() string
	// Type returns the streaming platform type
	Type() StreamType
	// Connect establishes the connection to the event sink
	Connect(ctx context.Context, config *BridgeConfig, credentials map[string]string) error
	// Publish sends a single event to the sink
	Publish(ctx context.Context, event *StreamEvent) error
	// PublishBatch sends multiple events to the sink
	PublishBatch(ctx context.Context, events []*StreamEvent) error
	// Healthy checks if the connection is alive
	Healthy(ctx context.Context) bool
	// Close shuts down the event sink
	Close() error
}

// EventSourceFactory creates EventSource instances by platform type
func EventSourceFactory(streamType StreamType) (EventSource, error) {
	switch streamType {
	case StreamTypeKafka:
		return &KafkaEventSource{}, nil
	case StreamTypeNATS:
		return &NATSEventSource{}, nil
	case StreamTypeRabbitMQ:
		return &RabbitMQEventSource{}, nil
	case StreamTypeSQS:
		return &SQSEventSource{}, nil
	default:
		return nil, ErrUnsupportedPlatform
	}
}

// EventSinkFactory creates EventSink instances by platform type
func EventSinkFactory(streamType StreamType) (EventSink, error) {
	switch streamType {
	case StreamTypeKafka:
		return &KafkaEventSink{}, nil
	case StreamTypeNATS:
		return &NATSEventSink{}, nil
	case StreamTypeRabbitMQ:
		return &RabbitMQEventSink{}, nil
	case StreamTypeSNS:
		return &SNSEventSink{}, nil
	default:
		return nil, ErrUnsupportedPlatform
	}
}

// --- Kafka EventSource/EventSink adapters (delegate to existing KafkaProducer/Consumer) ---

// KafkaEventSource wraps the existing Kafka consumer as an EventSource
type KafkaEventSource struct {
	consumer *KafkaConsumer
	bridge   *StreamingBridge
}

func (s *KafkaEventSource) Name() string     { return "kafka-source" }
func (s *KafkaEventSource) Type() StreamType { return StreamTypeKafka }
func (s *KafkaEventSource) Healthy(ctx context.Context) bool {
	if s.consumer == nil {
		return false
	}
	return s.consumer.Healthy(ctx)
}
func (s *KafkaEventSource) Close() error {
	if s.consumer != nil {
		return s.consumer.Close()
	}
	return nil
}
func (s *KafkaEventSource) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.consumer = NewKafkaConsumer()
	s.bridge = &StreamingBridge{ID: "kafka-source", Config: config, StreamType: StreamTypeKafka}
	return s.consumer.Init(ctx, s.bridge, creds)
}
func (s *KafkaEventSource) Subscribe(ctx context.Context, handler EventHandler) error {
	return s.consumer.Start(ctx, handler)
}
func (s *KafkaEventSource) Unsubscribe() error {
	return s.consumer.Pause()
}

// KafkaEventSink wraps the existing Kafka producer as an EventSink
type KafkaEventSink struct {
	producer *KafkaProducer
	bridge   *StreamingBridge
}

func (s *KafkaEventSink) Name() string     { return "kafka-sink" }
func (s *KafkaEventSink) Type() StreamType { return StreamTypeKafka }
func (s *KafkaEventSink) Healthy(ctx context.Context) bool {
	if s.producer == nil {
		return false
	}
	return s.producer.Healthy(ctx)
}
func (s *KafkaEventSink) Close() error {
	if s.producer != nil {
		return s.producer.Close()
	}
	return nil
}
func (s *KafkaEventSink) Connect(ctx context.Context, config *BridgeConfig, creds map[string]string) error {
	s.producer = NewKafkaProducer()
	s.bridge = &StreamingBridge{ID: "kafka-sink", Config: config, StreamType: StreamTypeKafka}
	return s.producer.Init(ctx, s.bridge, creds)
}
func (s *KafkaEventSink) Publish(ctx context.Context, event *StreamEvent) error {
	return s.producer.Send(ctx, event)
}
func (s *KafkaEventSink) PublishBatch(ctx context.Context, events []*StreamEvent) error {
	return s.producer.SendBatch(ctx, events)
}

// DeliveryTargetAdapter adapts an EventSink for use as a webhook delivery target.
// This allows streaming platforms to be used alongside HTTP as delivery mechanisms.
type DeliveryTargetAdapter struct {
	Sink     EventSink
	BridgeID string
	TenantID string
}

// Deliver sends a webhook payload to the streaming sink
func (a *DeliveryTargetAdapter) Deliver(ctx context.Context, payload json.RawMessage, headers map[string]string) error {
	event := &StreamEvent{
		BridgeID: a.BridgeID,
		TenantID: a.TenantID,
		Value:    payload,
		Headers:  headers,
		Status:   "delivering",
	}
	return a.Sink.Publish(ctx, event)
}

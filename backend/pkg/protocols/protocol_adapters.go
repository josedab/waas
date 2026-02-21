package protocols

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ProtocolAdapter provides a consistent interface for protocol handling
// with built-in observability and unified delivery configuration.
type ProtocolAdapter interface {
	// Name returns the protocol name
	Name() Protocol

	// Deliver performs a delivery and records observability metrics
	Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResult, error)

	// ValidateConfig validates protocol-specific config
	ValidateConfig(config *DeliveryConfig) error

	// Close releases resources
	Close() error
}

// DeliveryResult extends DeliveryResponse with protocol-specific metadata
type DeliveryResult struct {
	Response     *DeliveryResponse      `json:"response"`
	Protocol     Protocol               `json:"protocol"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	DeliveredAt  time.Time              `json:"delivered_at"`
	DurationMs   int64                  `json:"duration_ms"`
}

// UnifiedDeliveryConfig allows selecting protocol per endpoint with fallback
type UnifiedDeliveryConfig struct {
	EndpointID       string            `json:"endpoint_id"`
	PrimaryProtocol  Protocol          `json:"primary_protocol"`
	FallbackProtocol Protocol          `json:"fallback_protocol,omitempty"`
	Configs          map[Protocol]*DeliveryConfig `json:"configs"`
}

// ProtocolMetrics tracks per-protocol delivery metrics
type ProtocolMetrics struct {
	DeliveryCount   int64   `json:"delivery_count"`
	SuccessCount    int64   `json:"success_count"`
	ErrorCount      int64   `json:"error_count"`
	TotalLatencyMs  int64   `json:"total_latency_ms"`
}

// AverageLatencyMs returns the average delivery latency
func (m *ProtocolMetrics) AverageLatencyMs() float64 {
	count := atomic.LoadInt64(&m.DeliveryCount)
	if count == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&m.TotalLatencyMs)) / float64(count)
}

// ErrorRate returns the fraction of deliveries that resulted in errors
func (m *ProtocolMetrics) ErrorRate() float64 {
	count := atomic.LoadInt64(&m.DeliveryCount)
	if count == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&m.ErrorCount)) / float64(count)
}

// ProtocolObserver collects delivery metrics per protocol
type ProtocolObserver struct {
	mu      sync.RWMutex
	metrics map[Protocol]*ProtocolMetrics
}

// NewProtocolObserver creates a new ProtocolObserver
func NewProtocolObserver() *ProtocolObserver {
	return &ProtocolObserver{
		metrics: make(map[Protocol]*ProtocolMetrics),
	}
}

// Record records a delivery outcome for a given protocol
func (o *ProtocolObserver) Record(protocol Protocol, durationMs int64, success bool) {
	o.mu.Lock()
	m, ok := o.metrics[protocol]
	if !ok {
		m = &ProtocolMetrics{}
		o.metrics[protocol] = m
	}
	o.mu.Unlock()

	atomic.AddInt64(&m.DeliveryCount, 1)
	atomic.AddInt64(&m.TotalLatencyMs, durationMs)
	if success {
		atomic.AddInt64(&m.SuccessCount, 1)
	} else {
		atomic.AddInt64(&m.ErrorCount, 1)
	}
}

// GetMetrics returns a snapshot of metrics for a protocol
func (o *ProtocolObserver) GetMetrics(protocol Protocol) *ProtocolMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()

	m, ok := o.metrics[protocol]
	if !ok {
		return &ProtocolMetrics{}
	}
	return &ProtocolMetrics{
		DeliveryCount:  atomic.LoadInt64(&m.DeliveryCount),
		SuccessCount:   atomic.LoadInt64(&m.SuccessCount),
		ErrorCount:     atomic.LoadInt64(&m.ErrorCount),
		TotalLatencyMs: atomic.LoadInt64(&m.TotalLatencyMs),
	}
}

// AllMetrics returns a snapshot of metrics for all observed protocols
func (o *ProtocolObserver) AllMetrics() map[Protocol]*ProtocolMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[Protocol]*ProtocolMetrics, len(o.metrics))
	for p, m := range o.metrics {
		result[p] = &ProtocolMetrics{
			DeliveryCount:  atomic.LoadInt64(&m.DeliveryCount),
			SuccessCount:   atomic.LoadInt64(&m.SuccessCount),
			ErrorCount:     atomic.LoadInt64(&m.ErrorCount),
			TotalLatencyMs: atomic.LoadInt64(&m.TotalLatencyMs),
		}
	}
	return result
}

// delivererAdapter wraps a Deliverer to implement ProtocolAdapter
type delivererAdapter struct {
	deliverer Deliverer
	observer  *ProtocolObserver
}

// NewProtocolAdapter wraps a Deliverer into a ProtocolAdapter with observability
func NewProtocolAdapter(deliverer Deliverer, observer *ProtocolObserver) ProtocolAdapter {
	return &delivererAdapter{
		deliverer: deliverer,
		observer:  observer,
	}
}

func (a *delivererAdapter) Name() Protocol {
	return a.deliverer.Protocol()
}

func (a *delivererAdapter) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResult, error) {
	resp, err := a.deliverer.Deliver(ctx, config, request)
	if err != nil {
		if a.observer != nil {
			a.observer.Record(a.deliverer.Protocol(), 0, false)
		}
		return nil, err
	}

	durationMs := resp.Duration.Milliseconds()

	if a.observer != nil {
		a.observer.Record(a.deliverer.Protocol(), durationMs, resp.Success)
	}

	return &DeliveryResult{
		Response:    resp,
		Protocol:    a.deliverer.Protocol(),
		Metadata:    resp.ProtocolInfo,
		DeliveredAt: time.Now(),
		DurationMs:  durationMs,
	}, nil
}

func (a *delivererAdapter) ValidateConfig(config *DeliveryConfig) error {
	return a.deliverer.Validate(config)
}

func (a *delivererAdapter) Close() error {
	return a.deliverer.Close()
}

// UnifiedDeliverer delivers using the UnifiedDeliveryConfig with primary/fallback
type UnifiedDeliverer struct {
	registry *Registry
	observer *ProtocolObserver
}

// NewUnifiedDeliverer creates a UnifiedDeliverer backed by a Registry
func NewUnifiedDeliverer(registry *Registry, observer *ProtocolObserver) *UnifiedDeliverer {
	return &UnifiedDeliverer{
		registry: registry,
		observer: observer,
	}
}

// Deliver attempts delivery using the primary protocol, falling back if configured
func (u *UnifiedDeliverer) Deliver(ctx context.Context, udc *UnifiedDeliveryConfig, request *DeliveryRequest) (*DeliveryResult, error) {
	config, ok := udc.Configs[udc.PrimaryProtocol]
	if !ok {
		return nil, fmt.Errorf("no config for primary protocol: %s", udc.PrimaryProtocol)
	}

	deliverer, err := u.registry.Get(udc.PrimaryProtocol)
	if err != nil {
		return nil, fmt.Errorf("primary protocol not registered: %w", err)
	}

	adapter := NewProtocolAdapter(deliverer, u.observer)
	result, err := adapter.Deliver(ctx, config, request)
	if err == nil && result.Response.Success {
		return result, nil
	}

	// Attempt fallback
	if udc.FallbackProtocol != "" {
		fbConfig, fbOK := udc.Configs[udc.FallbackProtocol]
		if fbOK {
			fbDeliverer, fbErr := u.registry.Get(udc.FallbackProtocol)
			if fbErr == nil {
				fbAdapter := NewProtocolAdapter(fbDeliverer, u.observer)
				fbResult, fbErr := fbAdapter.Deliver(ctx, fbConfig, request)
				if fbErr == nil {
					fbResult.Metadata["fallback"] = true
					return fbResult, nil
				}
			}
		}
	}

	// Return primary result even if not successful
	if result != nil {
		return result, nil
	}
	return nil, err
}

// NewDelivererForProtocol is a factory that creates a Deliverer for a given protocol
func NewDelivererForProtocol(protocol Protocol) (Deliverer, error) {
	switch protocol {
	case ProtocolHTTP, ProtocolHTTPS:
		return NewHTTPDeliverer(), nil
	case ProtocolGRPC, ProtocolGRPCS:
		return NewGRPCDeliverer(), nil
	case ProtocolWebSocket:
		return NewWebSocketDeliverer(), nil
	case ProtocolMQTT:
		return NewMQTTDeliverer(), nil
	case ProtocolGraphQL:
		return NewGraphQLDeliverer(), nil
	case ProtocolSMTP:
		return NewSMTPDeliverer(), nil
	case ProtocolKafka:
		return NewKafkaDeliverer(), nil
	case ProtocolSNS:
		return NewSNSDeliverer(), nil
	case ProtocolSQS:
		return NewSQSDeliverer(), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

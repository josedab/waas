package protocols

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages protocol deliverers
type Registry struct {
	deliverers map[Protocol]Deliverer
	mu         sync.RWMutex
}

// NewRegistry creates a new protocol registry
func NewRegistry() *Registry {
	return &Registry{
		deliverers: make(map[Protocol]Deliverer),
	}
}

// Register registers a deliverer for a protocol
func (r *Registry) Register(protocol Protocol, deliverer Deliverer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deliverers[protocol] = deliverer
}

// Get retrieves a deliverer for a protocol
func (r *Registry) Get(protocol Protocol) (Deliverer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deliverer, exists := r.deliverers[protocol]
	if !exists {
		return nil, fmt.Errorf("no deliverer registered for protocol: %s", protocol)
	}
	return deliverer, nil
}

// Supported returns list of supported protocols
func (r *Registry) Supported() []Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]Protocol, 0, len(r.deliverers))
	for p := range r.deliverers {
		protocols = append(protocols, p)
	}
	return protocols
}

// Deliver performs a delivery using the appropriate protocol deliverer
func (r *Registry) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	deliverer, err := r.Get(config.Protocol)
	if err != nil {
		return nil, err
	}
	return deliverer.Deliver(ctx, config, request)
}

// ValidateConfig validates a delivery config using the appropriate deliverer
func (r *Registry) ValidateConfig(config *DeliveryConfig) error {
	deliverer, err := r.Get(config.Protocol)
	if err != nil {
		return err
	}
	return deliverer.Validate(config)
}

// Close closes all registered deliverers
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for _, deliverer := range r.deliverers {
		if err := deliverer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing deliverers: %v", errs)
	}
	return nil
}

// DefaultRegistry creates a registry with all default deliverers
func DefaultRegistry() *Registry {
	registry := NewRegistry()

	// Register HTTP deliverer
	registry.Register(ProtocolHTTP, NewHTTPDeliverer())
	registry.Register(ProtocolHTTPS, NewHTTPDeliverer())

	// Register gRPC deliverer
	registry.Register(ProtocolGRPC, NewGRPCDeliverer())
	registry.Register(ProtocolGRPCS, NewGRPCDeliverer())

	// Register WebSocket deliverer
	registry.Register(ProtocolWebSocket, NewWebSocketDeliverer())

	// Register MQTT deliverer
	registry.Register(ProtocolMQTT, NewMQTTDeliverer())

	return registry
}

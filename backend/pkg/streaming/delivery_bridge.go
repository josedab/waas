package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeliveryBridge connects an EventSource to the webhook delivery pipeline.
// It receives events from a streaming platform and forwards them as webhook deliveries.
type DeliveryBridge struct {
	bridgeID   string
	tenantID   string
	source     EventSource
	publisher  WebhookPublisher
	config     *DeliveryBridgeConfig
	metrics    *DeliveryBridgeMetrics
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

// WebhookPublisher publishes events to the webhook delivery queue
type WebhookPublisher interface {
	Publish(ctx context.Context, tenantID, endpointID string, payload []byte, headers map[string]string) (string, error)
}

// DeliveryBridgeConfig configures the delivery bridge
type DeliveryBridgeConfig struct {
	TargetEndpointIDs []string          `json:"target_endpoint_ids"`
	EventTypeMapping  map[string]string `json:"event_type_mapping,omitempty"`
	MaxConcurrency    int               `json:"max_concurrency"`
	RetryOnFailure    bool              `json:"retry_on_failure"`
	MaxRetries        int               `json:"max_retries"`
	FilterRules       []FilterRule      `json:"filter_rules,omitempty"`
}

// DefaultDeliveryBridgeConfig returns sensible defaults
func DefaultDeliveryBridgeConfig() *DeliveryBridgeConfig {
	return &DeliveryBridgeConfig{
		MaxConcurrency: 10,
		RetryOnFailure: true,
		MaxRetries:     3,
	}
}

// DeliveryBridgeMetrics tracks bridge delivery metrics
type DeliveryBridgeMetrics struct {
	EventsReceived  int64     `json:"events_received"`
	EventsDelivered int64     `json:"events_delivered"`
	EventsFiltered  int64     `json:"events_filtered"`
	EventsFailed    int64     `json:"events_failed"`
	AvgLatencyMs    float64   `json:"avg_latency_ms"`
	LastEventAt     time.Time `json:"last_event_at"`
	mu              sync.RWMutex
}

// NewDeliveryBridge creates a new delivery bridge
func NewDeliveryBridge(bridgeID, tenantID string, source EventSource, publisher WebhookPublisher, config *DeliveryBridgeConfig) *DeliveryBridge {
	if config == nil {
		config = DefaultDeliveryBridgeConfig()
	}
	return &DeliveryBridge{
		bridgeID:  bridgeID,
		tenantID:  tenantID,
		source:    source,
		publisher: publisher,
		config:    config,
		metrics:   &DeliveryBridgeMetrics{},
	}
}

// Start begins consuming from the EventSource and delivering to webhook endpoints
func (db *DeliveryBridge) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	db.mu.Lock()
	db.cancelFunc = cancel
	db.mu.Unlock()

	return db.source.Subscribe(ctx, db.handleEvent)
}

// Stop halts the delivery bridge
func (db *DeliveryBridge) Stop() error {
	db.mu.RLock()
	cancel := db.cancelFunc
	db.mu.RUnlock()

	if cancel != nil {
		cancel()
	}
	return db.source.Unsubscribe()
}

// GetMetrics returns current bridge metrics
func (db *DeliveryBridge) GetMetrics() DeliveryBridgeMetrics {
	db.metrics.mu.RLock()
	defer db.metrics.mu.RUnlock()
	return *db.metrics
}

func (db *DeliveryBridge) handleEvent(ctx context.Context, event *StreamEvent) error {
	db.metrics.mu.Lock()
	db.metrics.EventsReceived++
	db.metrics.LastEventAt = time.Now()
	db.metrics.mu.Unlock()

	startTime := time.Now()

	// Apply filter rules
	if len(db.config.FilterRules) > 0 && !db.matchesFilters(event) {
		db.metrics.mu.Lock()
		db.metrics.EventsFiltered++
		db.metrics.mu.Unlock()
		return nil
	}

	// Build headers from event
	headers := make(map[string]string)
	for k, v := range event.Headers {
		headers[k] = v
	}
	headers["X-Stream-Bridge-ID"] = db.bridgeID
	headers["X-Stream-Event-ID"] = event.ID
	headers["X-Stream-Source"] = string(db.source.Type())

	// Map event type if configured
	eventType := event.Key
	if mapped, ok := db.config.EventTypeMapping[eventType]; ok {
		eventType = mapped
	}
	if eventType != "" {
		headers["X-Event-Type"] = eventType
	}

	// Deliver to each target endpoint
	var lastErr error
	for _, endpointID := range db.config.TargetEndpointIDs {
		_, err := db.publisher.Publish(ctx, db.tenantID, endpointID, event.Value, headers)
		if err != nil {
			lastErr = err
			db.metrics.mu.Lock()
			db.metrics.EventsFailed++
			db.metrics.mu.Unlock()
		} else {
			db.metrics.mu.Lock()
			db.metrics.EventsDelivered++
			db.metrics.mu.Unlock()
		}
	}

	// Update latency
	latency := float64(time.Since(startTime).Milliseconds())
	db.metrics.mu.Lock()
	total := db.metrics.EventsDelivered + db.metrics.EventsFailed
	if total > 0 {
		db.metrics.AvgLatencyMs = (db.metrics.AvgLatencyMs*float64(total-1) + latency) / float64(total)
	}
	db.metrics.mu.Unlock()

	return lastErr
}

func (db *DeliveryBridge) matchesFilters(event *StreamEvent) bool {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Value, &data); err != nil {
		return true // Can't parse — let it through
	}

	for _, rule := range db.config.FilterRules {
		val, ok := data[rule.Field]
		if !ok {
			if rule.Operator == "exists" && !rule.Negate {
				return false
			}
			continue
		}

		valStr := fmt.Sprintf("%v", val)
		ruleValStr := fmt.Sprintf("%v", rule.Value)
		matched := false

		switch rule.Operator {
		case "eq":
			matched = valStr == ruleValStr
		case "neq":
			matched = valStr != ruleValStr
		case "contains":
			matched = len(valStr) > 0 && len(ruleValStr) > 0 && contains(valStr, ruleValStr)
		case "exists":
			matched = ok
		}

		if rule.Negate {
			matched = !matched
		}
		if !matched {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MultiProtocolHealthChecker checks health of all active bridges
type MultiProtocolHealthChecker struct {
	sources map[string]EventSource
	sinks   map[string]EventSink
	mu      sync.RWMutex
}

// NewMultiProtocolHealthChecker creates a new health checker
func NewMultiProtocolHealthChecker() *MultiProtocolHealthChecker {
	return &MultiProtocolHealthChecker{
		sources: make(map[string]EventSource),
		sinks:   make(map[string]EventSink),
	}
}

// RegisterSource registers a source for health monitoring
func (hc *MultiProtocolHealthChecker) RegisterSource(id string, source EventSource) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.sources[id] = source
}

// RegisterSink registers a sink for health monitoring
func (hc *MultiProtocolHealthChecker) RegisterSink(id string, sink EventSink) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.sinks[id] = sink
}

// BridgeHealthStatus represents the health of a single streaming connection
type BridgeHealthStatus struct {
	ID        string     `json:"id"`
	Type      StreamType `json:"type"`
	Role      string     `json:"role"` // source or sink
	Healthy   bool       `json:"healthy"`
	CheckedAt time.Time  `json:"checked_at"`
}

// CheckAll checks health of all registered sources and sinks
func (hc *MultiProtocolHealthChecker) CheckAll(ctx context.Context) []BridgeHealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var results []BridgeHealthStatus
	now := time.Now()

	for id, source := range hc.sources {
		results = append(results, BridgeHealthStatus{
			ID:        id,
			Type:      source.Type(),
			Role:      "source",
			Healthy:   source.Healthy(ctx),
			CheckedAt: now,
		})
	}

	for id, sink := range hc.sinks {
		results = append(results, BridgeHealthStatus{
			ID:        id,
			Type:      sink.Type(),
			Role:      "sink",
			Healthy:   sink.Healthy(ctx),
			CheckedAt: now,
		})
	}

	return results
}

// OutboundWebhookBridge captures webhook deliveries and forwards them to an EventSink.
// This enables using streaming platforms as delivery targets alongside HTTP.
type OutboundWebhookBridge struct {
	bridgeID string
	tenantID string
	sink     EventSink
	mu       sync.RWMutex
	metrics  struct {
		EventsSent   int64
		EventsFailed int64
		mu           sync.Mutex
	}
}

// NewOutboundWebhookBridge creates a new outbound bridge
func NewOutboundWebhookBridge(bridgeID, tenantID string, sink EventSink) *OutboundWebhookBridge {
	return &OutboundWebhookBridge{
		bridgeID: bridgeID,
		tenantID: tenantID,
		sink:     sink,
	}
}

// ForwardToStream sends a webhook delivery payload to the configured streaming sink
func (ob *OutboundWebhookBridge) ForwardToStream(ctx context.Context, endpointID string, payload json.RawMessage, headers map[string]string, eventType string) error {
	event := &StreamEvent{
		ID:       uuid.New().String(),
		BridgeID: ob.bridgeID,
		TenantID: ob.tenantID,
		Key:      eventType,
		Value:    payload,
		Headers:  headers,
		Status:   "delivering",
	}

	if event.Headers == nil {
		event.Headers = make(map[string]string)
	}
	event.Headers["X-Endpoint-ID"] = endpointID
	event.Headers["X-Bridge-ID"] = ob.bridgeID

	err := ob.sink.Publish(ctx, event)
	ob.metrics.mu.Lock()
	if err != nil {
		ob.metrics.EventsFailed++
	} else {
		ob.metrics.EventsSent++
	}
	ob.metrics.mu.Unlock()

	return err
}

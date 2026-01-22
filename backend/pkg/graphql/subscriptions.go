package graphql

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Subscriber represents a WebSocket subscription
type Subscriber struct {
	ID       string
	TenantID string
	Channel  string
	Filter   map[string]interface{}
	Messages chan []byte
	Done     chan struct{}
}

// SubscriptionManager manages GraphQL subscriptions
type SubscriptionManager struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]*Subscriber // channel -> id -> subscriber
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[string]map[string]*Subscriber),
	}
}

// Subscribe creates a new subscription
func (m *SubscriptionManager) Subscribe(ctx context.Context, tenantID, channel string, filter map[string]interface{}) *Subscriber {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub := &Subscriber{
		ID:       generateID(),
		TenantID: tenantID,
		Channel:  channel,
		Filter:   filter,
		Messages: make(chan []byte, 100),
		Done:     make(chan struct{}),
	}

	if m.subscribers[channel] == nil {
		m.subscribers[channel] = make(map[string]*Subscriber)
	}
	m.subscribers[channel][sub.ID] = sub

	return sub
}

// Unsubscribe removes a subscription
func (m *SubscriptionManager) Unsubscribe(sub *Subscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subs, ok := m.subscribers[sub.Channel]; ok {
		delete(subs, sub.ID)
		if len(subs) == 0 {
			delete(m.subscribers, sub.Channel)
		}
	}

	close(sub.Done)
}

// Publish sends a message to all matching subscribers
func (m *SubscriptionManager) Publish(channel string, tenantID string, data interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs, ok := m.subscribers[channel]
	if !ok {
		return
	}

	message, err := json.Marshal(data)
	if err != nil {
		return
	}

	for _, sub := range subs {
		if sub.TenantID != tenantID {
			continue
		}

		// Check filter match
		if !m.matchesFilter(data, sub.Filter) {
			continue
		}

		select {
		case sub.Messages <- message:
		default:
			// Channel full, skip
		}
	}
}

func (m *SubscriptionManager) matchesFilter(data interface{}, filter map[string]interface{}) bool {
	if filter == nil {
		return true
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		// Try to convert via JSON
		jsonBytes, _ := json.Marshal(data)
		json.Unmarshal(jsonBytes, &dataMap)
	}

	for key, filterValue := range filter {
		if dataValue, exists := dataMap[key]; exists {
			if dataValue != filterValue {
				return false
			}
		}
	}

	return true
}

// Subscription channels
const (
	ChannelDeliveryUpdated = "delivery_updated"
	ChannelAnomalyDetected = "anomaly_detected"
	ChannelMetricsUpdated  = "metrics_updated"
)

// DeliveryUpdateEvent represents a delivery update subscription event
type DeliveryUpdateEvent struct {
	ID             string    `json:"id"`
	EndpointID     string    `json:"endpointId"`
	Status         string    `json:"status"`
	AttemptCount   int       `json:"attemptCount"`
	LastHTTPStatus int       `json:"lastHttpStatus,omitempty"`
	LastError      string    `json:"lastError,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// AnomalyDetectedEvent represents an anomaly detection subscription event
type AnomalyDetectedEvent struct {
	ID            string    `json:"id"`
	MetricType    string    `json:"metricType"`
	Severity      string    `json:"severity"`
	CurrentValue  float64   `json:"currentValue"`
	ExpectedValue float64   `json:"expectedValue"`
	DeviationPct  float64   `json:"deviationPct"`
	Description   string    `json:"description"`
	DetectedAt    time.Time `json:"detectedAt"`
}

// MetricsUpdateEvent represents a metrics update subscription event
type MetricsUpdateEvent struct {
	TotalDeliveries       int     `json:"totalDeliveries"`
	SuccessfulDeliveries  int     `json:"successfulDeliveries"`
	FailedDeliveries      int     `json:"failedDeliveries"`
	SuccessRate           float64 `json:"successRate"`
	AvgLatencyMs          float64 `json:"avgLatencyMs"`
	DeliveryRatePerMinute float64 `json:"deliveryRatePerMinute"`
	Timestamp             time.Time `json:"timestamp"`
}

// Resolver provides GraphQL query/mutation resolution
type Resolver struct {
	subscriptions *SubscriptionManager
	// Add service dependencies here
}

// NewResolver creates a new GraphQL resolver
func NewResolver() *Resolver {
	return &Resolver{
		subscriptions: NewSubscriptionManager(),
	}
}

// GetSubscriptionManager returns the subscription manager
func (r *Resolver) GetSubscriptionManager() *SubscriptionManager {
	return r.subscriptions
}

// NotifyDeliveryUpdate notifies subscribers of a delivery update
func (r *Resolver) NotifyDeliveryUpdate(tenantID string, event *DeliveryUpdateEvent) {
	r.subscriptions.Publish(ChannelDeliveryUpdated, tenantID, event)
}

// NotifyAnomalyDetected notifies subscribers of an anomaly
func (r *Resolver) NotifyAnomalyDetected(tenantID string, event *AnomalyDetectedEvent) {
	r.subscriptions.Publish(ChannelAnomalyDetected, tenantID, event)
}

// NotifyMetricsUpdate notifies subscribers of metrics updates
func (r *Resolver) NotifyMetricsUpdate(tenantID string, event *MetricsUpdateEvent) {
	r.subscriptions.Publish(ChannelMetricsUpdated, tenantID, event)
}

func generateID() string {
	// Simple ID generation - in production use UUID
	return time.Now().Format("20060102150405.000000000")
}

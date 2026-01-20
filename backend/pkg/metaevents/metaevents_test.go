package metaevents

import (
	"context"
	"testing"
	"time"
)

func TestEventTypes(t *testing.T) {
	types := []EventType{
		EventDeliveryAttempted,
		EventDeliverySucceeded,
		EventDeliveryFailed,
		EventDeliveryRetrying,
		EventDeliveryExhausted,
		EventEndpointCreated,
		EventEndpointUpdated,
		EventEndpointDeleted,
		EventEndpointDisabled,
		EventEndpointEnabled,
		EventEndpointHealthy,
		EventEndpointUnhealthy,
		EventEndpointRecovered,
		EventThresholdError,
		EventThresholdLatency,
		EventThresholdVolume,
		EventAnomalyDetected,
		EventSecurityViolation,
		EventAPIKeyRotated,
	}
	
	for _, eventType := range types {
		if eventType == "" {
			t.Error("expected non-empty event type")
		}
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()
	
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	
	if policy.MaxRetries <= 0 {
		t.Error("expected positive max retries")
	}
	
	if policy.InitialInterval <= 0 {
		t.Error("expected positive initial interval")
	}
	
	if policy.Multiplier <= 1 {
		t.Error("expected multiplier greater than 1")
	}
}

func TestMetaEvent(t *testing.T) {
	event := &MetaEvent{
		ID:         "event-1",
		TenantID:   "tenant-1",
		Type:       EventDeliverySucceeded,
		Source:     "delivery",
		SourceID:   "delivery-123",
		Data: map[string]interface{}{
			"webhook_id":   "wh-1",
			"endpoint_id":  "ep-1",
			"status_code":  200,
			"latency_ms":   50,
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}
	
	if event.Type != EventDeliverySucceeded {
		t.Errorf("expected type delivery.succeeded, got %s", event.Type)
	}
	
	if event.Source != "delivery" {
		t.Errorf("expected source 'delivery', got %s", event.Source)
	}
}

func TestSubscription(t *testing.T) {
	sub := &Subscription{
		ID:         "sub-1",
		TenantID:   "tenant-1",
		Name:       "My Subscription",
		URL:        "https://example.com/webhook",
		EventTypes: []EventType{EventDeliverySucceeded, EventDeliveryFailed},
		Filters: &EventFilter{
			EndpointIDs: []string{"ep-1", "ep-2"},
			Severities:  []string{"high", "critical"},
		},
		IsActive:    true,
		Headers:     Headers{"X-Custom": "value"},
		RetryPolicy: DefaultRetryPolicy(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if sub.Name != "My Subscription" {
		t.Errorf("expected name 'My Subscription', got %s", sub.Name)
	}
	
	if len(sub.EventTypes) != 2 {
		t.Errorf("expected 2 event types, got %d", len(sub.EventTypes))
	}
	
	if !sub.IsActive {
		t.Error("expected subscription to be active")
	}
}

func TestEventFilter(t *testing.T) {
	filter := &EventFilter{
		EndpointIDs: []string{"ep-1", "ep-2", "ep-3"},
		Severities:  []string{"warning", "error", "critical"},
		Sources:     []string{"delivery", "endpoint"},
	}
	
	if len(filter.EndpointIDs) != 3 {
		t.Errorf("expected 3 endpoint IDs, got %d", len(filter.EndpointIDs))
	}
	
	if len(filter.Severities) != 3 {
		t.Errorf("expected 3 severities, got %d", len(filter.Severities))
	}
}

func TestDelivery(t *testing.T) {
	delivery := &Delivery{
		ID:             "del-1",
		SubscriptionID: "sub-1",
		EventID:        "event-1",
		TenantID:       "tenant-1",
		Status:         "success",
		Attempt:        1,
		StatusCode:     200,
		LatencyMs:      45,
		CreatedAt:      time.Now(),
	}
	
	if delivery.Status != "success" {
		t.Errorf("expected status 'success', got %s", delivery.Status)
	}
	
	if delivery.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", delivery.StatusCode)
	}
}

func TestDeliveryPayload(t *testing.T) {
	payload := &DeliveryPayload{
		ID:         "event-1",
		Type:       EventDeliveryFailed,
		Source:     "delivery",
		SourceID:   "delivery-123",
		TenantID:   "tenant-1",
		Data: map[string]interface{}{
			"error_message": "connection timeout",
		},
		OccurredAt: time.Now(),
		Timestamp:  time.Now(),
	}
	
	if payload.Type != EventDeliveryFailed {
		t.Errorf("expected type delivery.failed, got %s", payload.Type)
	}
}

func TestDeliveryData(t *testing.T) {
	data := &DeliveryData{
		WebhookID:    "wh-1",
		EndpointID:   "ep-1",
		EndpointURL:  "https://example.com/webhook",
		Attempt:      2,
		StatusCode:   503,
		LatencyMs:    5000,
		ErrorMessage: "service unavailable",
		ErrorType:    "http_error",
	}
	
	if data.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", data.StatusCode)
	}
	
	if data.ErrorType != "http_error" {
		t.Errorf("expected error type 'http_error', got %s", data.ErrorType)
	}
}

func TestEndpointData(t *testing.T) {
	data := &EndpointData{
		EndpointID:  "ep-1",
		EndpointURL: "https://example.com/webhook",
		Name:        "Production Webhook",
		OldStatus:   "active",
		NewStatus:   "disabled",
	}
	
	if data.OldStatus != "active" {
		t.Errorf("expected old status 'active', got %s", data.OldStatus)
	}
	
	if data.NewStatus != "disabled" {
		t.Errorf("expected new status 'disabled', got %s", data.NewStatus)
	}
}

func TestThresholdData(t *testing.T) {
	data := &ThresholdData{
		Metric:       "error_rate",
		Value:        0.15,
		Threshold:    0.10,
		EndpointID:   "ep-1",
		WindowPeriod: "5m",
	}
	
	if data.Value <= data.Threshold {
		t.Error("expected value to exceed threshold")
	}
	
	if data.WindowPeriod != "5m" {
		t.Errorf("expected window '5m', got %s", data.WindowPeriod)
	}
}

func TestAnomalyData(t *testing.T) {
	data := &AnomalyData{
		Metric:     "latency_p99",
		Expected:   100.0,
		Actual:     500.0,
		Deviation:  4.0,
		EndpointID: "ep-1",
		DetectedAt: time.Now(),
	}
	
	if data.Deviation <= 0 {
		t.Error("expected positive deviation")
	}
	
	if data.Actual <= data.Expected {
		t.Error("expected actual to exceed expected")
	}
}

func TestCreateSubscriptionRequest(t *testing.T) {
	req := &CreateSubscriptionRequest{
		Name:       "My Subscription",
		URL:        "https://example.com/webhook",
		EventTypes: []EventType{EventDeliverySucceeded, EventDeliveryFailed},
		Filters: &EventFilter{
			EndpointIDs: []string{"ep-1"},
		},
		Headers: Headers{"Authorization": "Bearer token"},
	}
	
	if req.Name == "" {
		t.Error("expected name to be set")
	}
	
	if len(req.EventTypes) != 2 {
		t.Errorf("expected 2 event types, got %d", len(req.EventTypes))
	}
}

func TestUpdateSubscriptionRequest(t *testing.T) {
	name := "Updated Name"
	url := "https://new.example.com/webhook"
	isActive := false
	
	req := &UpdateSubscriptionRequest{
		Name:       &name,
		URL:        &url,
		EventTypes: []EventType{EventEndpointHealthy},
		IsActive:   &isActive,
	}
	
	if *req.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", *req.Name)
	}
	
	if *req.IsActive {
		t.Error("expected is_active to be false")
	}
}

func TestServiceWithMockRepo(t *testing.T) {
	mockRepo := &mockMetaEventsRepository{}
	// Emitter can be nil for basic tests
	service := NewService(mockRepo, nil)
	
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	
	ctx := context.Background()
	
	// Test creating a subscription
	req := &CreateSubscriptionRequest{
		Name:       "Test Subscription",
		URL:        "https://example.com/webhook",
		EventTypes: []EventType{EventDeliverySucceeded},
	}
	
	sub, err := service.CreateSubscription(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}
	
	if sub.Name != "Test Subscription" {
		t.Errorf("expected name 'Test Subscription', got %s", sub.Name)
	}
}

// Mock repository for testing
type mockMetaEventsRepository struct {
	subscriptions map[string]*Subscription
	events        map[string]*MetaEvent
	deliveries    map[string]*Delivery
}

func (m *mockMetaEventsRepository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	if m.subscriptions == nil {
		m.subscriptions = make(map[string]*Subscription)
	}
	m.subscriptions[sub.ID] = sub
	return nil
}

func (m *mockMetaEventsRepository) GetSubscription(ctx context.Context, tenantID, subID string) (*Subscription, error) {
	if m.subscriptions == nil {
		return nil, nil
	}
	s, ok := m.subscriptions[subID]
	if !ok || s.TenantID != tenantID {
		return nil, nil
	}
	return s, nil
}

func (m *mockMetaEventsRepository) ListSubscriptions(ctx context.Context, tenantID string, limit, offset int) ([]Subscription, int, error) {
	var subs []Subscription
	for _, s := range m.subscriptions {
		if s.TenantID == tenantID {
			subs = append(subs, *s)
		}
	}
	return subs, len(subs), nil
}

func (m *mockMetaEventsRepository) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	if m.subscriptions == nil {
		m.subscriptions = make(map[string]*Subscription)
	}
	m.subscriptions[sub.ID] = sub
	return nil
}

func (m *mockMetaEventsRepository) DeleteSubscription(ctx context.Context, tenantID, subID string) error {
	delete(m.subscriptions, subID)
	return nil
}

func (m *mockMetaEventsRepository) GetSubscriptionsByEventType(ctx context.Context, tenantID string, eventType EventType) ([]Subscription, error) {
	var subs []Subscription
	for _, s := range m.subscriptions {
		if s.TenantID == tenantID {
			for _, et := range s.EventTypes {
				if et == eventType {
					subs = append(subs, *s)
					break
				}
			}
		}
	}
	return subs, nil
}

func (m *mockMetaEventsRepository) CreateEvent(ctx context.Context, event *MetaEvent) error {
	if m.events == nil {
		m.events = make(map[string]*MetaEvent)
	}
	m.events[event.ID] = event
	return nil
}

func (m *mockMetaEventsRepository) GetEvent(ctx context.Context, tenantID, eventID string) (*MetaEvent, error) {
	if m.events == nil {
		return nil, nil
	}
	e, ok := m.events[eventID]
	if !ok || e.TenantID != tenantID {
		return nil, nil
	}
	return e, nil
}

func (m *mockMetaEventsRepository) ListEvents(ctx context.Context, tenantID string, eventType *EventType, limit, offset int) ([]MetaEvent, int, error) {
	var events []MetaEvent
	for _, e := range m.events {
		if e.TenantID == tenantID {
			if eventType == nil || e.Type == *eventType {
				events = append(events, *e)
			}
		}
	}
	return events, len(events), nil
}

func (m *mockMetaEventsRepository) CreateDelivery(ctx context.Context, delivery *Delivery) error {
	if m.deliveries == nil {
		m.deliveries = make(map[string]*Delivery)
	}
	m.deliveries[delivery.ID] = delivery
	return nil
}

func (m *mockMetaEventsRepository) ListDeliveries(ctx context.Context, tenantID, subID string, limit, offset int) ([]Delivery, int, error) {
	var dels []Delivery
	for _, d := range m.deliveries {
		if d.TenantID == tenantID && (subID == "" || d.SubscriptionID == subID) {
			dels = append(dels, *d)
		}
	}
	return dels, len(dels), nil
}

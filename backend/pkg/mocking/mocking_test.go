package mocking

import (
	"context"
	"testing"
	"time"
)

func TestFakerTypes(t *testing.T) {
	types := GetAvailableFakerTypes()
	
	if len(types) == 0 {
		t.Error("expected non-empty faker types list")
	}
	
	// Verify some essential types are present
	essentialTypes := []FakerType{
		FakerUUID,
		FakerEmail,
		FakerName,
		FakerNumber,
		FakerBoolean,
		FakerTimestamp,
	}
	
	for _, expected := range essentialTypes {
		found := false
		for _, ft := range types {
			if ft == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected faker type %s not found", expected)
		}
	}
}

func TestMockEndpoint(t *testing.T) {
	endpoint := &MockEndpoint{
		ID:          "mock-1",
		TenantID:    "tenant-1",
		Name:        "Test Mock Endpoint",
		Description: "For testing purposes",
		URL:         "https://example.com/webhook",
		EventType:   "order.created",
		Template: &PayloadTemplate{
			Type: "faker",
			Content: map[string]interface{}{
				"event":      "order.created",
				"order_id":   "{{uuid}}",
				"amount":     "{{price}}",
				"created_at": "{{timestamp}}",
			},
			Fields: []TemplateField{
				{Path: "order_id", Type: "string", Faker: "uuid"},
				{Path: "amount", Type: "number", Faker: "price"},
			},
		},
		Schedule: &MockSchedule{
			Type:     "interval",
			Interval: "5m",
			MaxRuns:  100,
		},
		Settings: MockSettings{
			Headers:   map[string]string{"Content-Type": "application/json"},
			Signature: true,
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if endpoint.Name != "Test Mock Endpoint" {
		t.Errorf("expected name 'Test Mock Endpoint', got %s", endpoint.Name)
	}
	
	if endpoint.Template == nil {
		t.Error("expected template to be set")
	}
	
	if endpoint.Schedule == nil {
		t.Error("expected schedule to be set")
	}
}

func TestPayloadTemplate(t *testing.T) {
	template := &PayloadTemplate{
		Type: "template",
		Content: map[string]interface{}{
			"event": "user.created",
			"user": map[string]interface{}{
				"id":    "{{uuid}}",
				"email": "{{email}}",
				"name":  "{{name}}",
			},
		},
		Fields: []TemplateField{
			{Path: "user.id", Type: "string", Faker: "uuid"},
			{Path: "user.email", Type: "string", Faker: "email"},
			{Path: "user.name", Type: "string", Faker: "name"},
		},
	}
	
	if template.Type != "template" {
		t.Errorf("expected type 'template', got %s", template.Type)
	}
	
	if len(template.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(template.Fields))
	}
}

func TestTemplateField(t *testing.T) {
	field := TemplateField{
		Path:  "price",
		Type:  "number",
		Faker: "price",
		Options: FieldOptions{
			Min:    0,
			Max:    1000,
			Format: "%.2f",
		},
	}
	
	if field.Path != "price" {
		t.Errorf("expected path 'price', got %s", field.Path)
	}
	
	if field.Options.Max != 1000 {
		t.Errorf("expected max 1000, got %f", field.Options.Max)
	}
}

func TestFieldOptions(t *testing.T) {
	options := FieldOptions{
		Min:      1,
		Max:      100,
		Length:   10,
		Format:   "YYYY-MM-DD",
		Choices:  []string{"active", "pending", "completed"},
		Nullable: true,
		NullProb: 0.1,
	}
	
	if len(options.Choices) != 3 {
		t.Errorf("expected 3 choices, got %d", len(options.Choices))
	}
	
	if options.NullProb != 0.1 {
		t.Errorf("expected null prob 0.1, got %f", options.NullProb)
	}
}

func TestMockSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule MockSchedule
		valid    bool
	}{
		{
			name: "once schedule",
			schedule: MockSchedule{
				Type: "once",
			},
			valid: true,
		},
		{
			name: "interval schedule",
			schedule: MockSchedule{
				Type:     "interval",
				Interval: "10m",
				MaxRuns:  50,
			},
			valid: true,
		},
		{
			name: "cron schedule",
			schedule: MockSchedule{
				Type: "cron",
				Cron: "0 */5 * * * *",
			},
			valid: true,
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.schedule.Type == "" {
				t.Error("expected schedule type to be set")
			}
		})
	}
}

func TestMockSettings(t *testing.T) {
	settings := MockSettings{
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"X-Custom-Header": "value",
		},
		DelayMs:       100,
		BatchSize:     10,
		BatchInterval: "1s",
		Signature:     true,
		SignatureKey:  "secret-key",
	}
	
	if len(settings.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(settings.Headers))
	}
	
	if settings.DelayMs != 100 {
		t.Errorf("expected delay 100ms, got %d", settings.DelayMs)
	}
	
	if !settings.Signature {
		t.Error("expected signature to be enabled")
	}
}

func TestMockDelivery(t *testing.T) {
	sentAt := time.Now()
	delivery := &MockDelivery{
		ID:         "delivery-1",
		EndpointID: "mock-1",
		TenantID:   "tenant-1",
		Payload: map[string]interface{}{
			"event":    "order.created",
			"order_id": "ord-123",
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Status:       "success",
		StatusCode:   200,
		ResponseBody: `{"status": "received"}`,
		LatencyMs:    45,
		SentAt:       &sentAt,
		CreatedAt:    time.Now(),
	}
	
	if delivery.Status != "success" {
		t.Errorf("expected status 'success', got %s", delivery.Status)
	}
	
	if delivery.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", delivery.StatusCode)
	}
}

func TestMockTemplate(t *testing.T) {
	template := &MockTemplate{
		ID:          "template-1",
		TenantID:    "tenant-1",
		Name:        "Order Created",
		Description: "Template for order.created events",
		EventType:   "order.created",
		Category:    "ecommerce",
		Template: PayloadTemplate{
			Type: "faker",
			Content: map[string]interface{}{
				"order_id": "{{uuid}}",
			},
		},
		Examples: []map[string]interface{}{
			{"order_id": "ord-123", "amount": 99.99},
		},
		IsPublic:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if template.Category != "ecommerce" {
		t.Errorf("expected category 'ecommerce', got %s", template.Category)
	}
	
	if !template.IsPublic {
		t.Error("expected template to be public")
	}
}

func TestCreateMockEndpointRequest(t *testing.T) {
	req := &CreateMockEndpointRequest{
		Name:        "New Mock",
		Description: "Testing mock",
		URL:         "https://example.com/webhook",
		EventType:   "user.created",
		Template: &PayloadTemplate{
			Type: "static",
			Content: map[string]interface{}{
				"user_id": "test-123",
			},
		},
		Settings: MockSettings{
			Signature: true,
		},
	}
	
	if req.Name == "" {
		t.Error("expected name to be set")
	}
	
	if req.URL == "" {
		t.Error("expected URL to be set")
	}
}

func TestTriggerMockRequest(t *testing.T) {
	req := &TriggerMockRequest{
		Count:    5,
		Payload: map[string]interface{}{
			"custom_field": "override_value",
		},
		DelayMs:  100,
		Interval: "1s",
	}
	
	if req.Count != 5 {
		t.Errorf("expected count 5, got %d", req.Count)
	}
	
	if req.Interval != "1s" {
		t.Errorf("expected interval '1s', got %s", req.Interval)
	}
}

func TestCreateTemplateRequest(t *testing.T) {
	req := &CreateTemplateRequest{
		Name:        "Payment Template",
		Description: "Template for payment events",
		EventType:   "payment.completed",
		Category:    "payments",
		Template: PayloadTemplate{
			Type: "faker",
			Fields: []TemplateField{
				{Path: "payment_id", Faker: "uuid"},
				{Path: "amount", Faker: "price"},
			},
		},
		IsPublic: false,
	}
	
	if req.Name == "" {
		t.Error("expected name to be set")
	}
	
	if req.Category != "payments" {
		t.Errorf("expected category 'payments', got %s", req.Category)
	}
}

func TestServiceWithMockRepo(t *testing.T) {
	mockRepo := &mockMockingRepository{}
	service := NewService(mockRepo)
	
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	
	ctx := context.Background()
	
	// Test creating a mock endpoint
	req := &CreateMockEndpointRequest{
		Name:      "Test Mock",
		URL:       "https://example.com/webhook",
		EventType: "test.event",
	}
	
	endpoint, err := service.CreateMockEndpoint(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if endpoint == nil {
		t.Fatal("expected non-nil endpoint")
	}
	
	if endpoint.Name != "Test Mock" {
		t.Errorf("expected name 'Test Mock', got %s", endpoint.Name)
	}
}

// Mock repository for testing
type mockMockingRepository struct {
	endpoints  map[string]*MockEndpoint
	deliveries map[string]*MockDelivery
	templates  map[string]*MockTemplate
}

func (m *mockMockingRepository) CreateMockEndpoint(ctx context.Context, endpoint *MockEndpoint) error {
	if m.endpoints == nil {
		m.endpoints = make(map[string]*MockEndpoint)
	}
	m.endpoints[endpoint.ID] = endpoint
	return nil
}

func (m *mockMockingRepository) GetMockEndpoint(ctx context.Context, tenantID, endpointID string) (*MockEndpoint, error) {
	if m.endpoints == nil {
		return nil, nil
	}
	e, ok := m.endpoints[endpointID]
	if !ok || e.TenantID != tenantID {
		return nil, nil
	}
	return e, nil
}

func (m *mockMockingRepository) ListMockEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]MockEndpoint, int, error) {
	var endpoints []MockEndpoint
	for _, e := range m.endpoints {
		if e.TenantID == tenantID {
			endpoints = append(endpoints, *e)
		}
	}
	return endpoints, len(endpoints), nil
}

func (m *mockMockingRepository) UpdateMockEndpoint(ctx context.Context, endpoint *MockEndpoint) error {
	if m.endpoints == nil {
		m.endpoints = make(map[string]*MockEndpoint)
	}
	m.endpoints[endpoint.ID] = endpoint
	return nil
}

func (m *mockMockingRepository) DeleteMockEndpoint(ctx context.Context, tenantID, endpointID string) error {
	delete(m.endpoints, endpointID)
	return nil
}

func (m *mockMockingRepository) CreateMockDelivery(ctx context.Context, delivery *MockDelivery) error {
	if m.deliveries == nil {
		m.deliveries = make(map[string]*MockDelivery)
	}
	m.deliveries[delivery.ID] = delivery
	return nil
}

func (m *mockMockingRepository) ListMockDeliveries(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]MockDelivery, int, error) {
	var deliveries []MockDelivery
	for _, d := range m.deliveries {
		if d.TenantID == tenantID && (endpointID == "" || d.EndpointID == endpointID) {
			deliveries = append(deliveries, *d)
		}
	}
	return deliveries, len(deliveries), nil
}

func (m *mockMockingRepository) CreateTemplate(ctx context.Context, template *MockTemplate) error {
	if m.templates == nil {
		m.templates = make(map[string]*MockTemplate)
	}
	m.templates[template.ID] = template
	return nil
}

func (m *mockMockingRepository) GetTemplate(ctx context.Context, tenantID, templateID string) (*MockTemplate, error) {
	if m.templates == nil {
		return nil, nil
	}
	t, ok := m.templates[templateID]
	if !ok {
		return nil, nil
	}
	if t.TenantID != tenantID && !t.IsPublic {
		return nil, nil
	}
	return t, nil
}

func (m *mockMockingRepository) ListTemplates(ctx context.Context, tenantID string, includePublic bool, limit, offset int) ([]MockTemplate, int, error) {
	var templates []MockTemplate
	for _, t := range m.templates {
		if t.TenantID == tenantID || (includePublic && t.IsPublic) {
			templates = append(templates, *t)
		}
	}
	return templates, len(templates), nil
}

func (m *mockMockingRepository) DeleteTemplate(ctx context.Context, tenantID, templateID string) error {
	delete(m.templates, templateID)
	return nil
}

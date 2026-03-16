package graphql

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestQueryExecutor_DetectOperationType(t *testing.T) {
	exec := NewQueryExecutor(NewResolver())

	tests := []struct {
		query    string
		expected string
	}{
		{"query { tenant { id } }", "query"},
		{"{ tenant { id } }", "query"},
		{"mutation { createEndpoint(input: {}) { id } }", "mutation"},
		{"subscription { deliveryUpdated { id } }", "subscription"},
		{"{ __schema { types { name } } }", "introspection"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := exec.detectOperationType(tt.query)
			if got != tt.expected {
				t.Errorf("detectOperationType(%q) = %s, want %s", tt.query, got, tt.expected)
			}
		})
	}
}

func TestQueryExecutor_ExecuteQuery(t *testing.T) {
	exec := NewQueryExecutor(NewResolver())

	// Use dashboard which doesn't have nested fields to parse
	resp := exec.Execute(context.Background(), "tenant-1", GraphQLRequest{
		Query: "query {\n  dashboard\n}",
	})

	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}
	if resp.Data == nil {
		t.Fatal("expected data in response")
	}
}

func TestQueryExecutor_ExecuteMutation(t *testing.T) {
	exec := NewQueryExecutor(NewResolver())

	resp := exec.Execute(context.Background(), "tenant-1", GraphQLRequest{
		Query:     `mutation { createEndpoint(input: $input) { id url } }`,
		Variables: map[string]interface{}{"input": map[string]interface{}{"url": "https://example.com/hook"}},
	})

	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}
}

func TestQueryExecutor_Subscription(t *testing.T) {
	exec := NewQueryExecutor(NewResolver())

	resp := exec.Execute(context.Background(), "tenant-1", GraphQLRequest{
		Query: `subscription { deliveryUpdated { id status } }`,
	})

	if len(resp.Errors) == 0 {
		t.Fatal("expected error for subscription over HTTP")
	}
	if resp.Errors[0].Extensions["code"] != "SUBSCRIPTION_OVER_HTTP" {
		t.Error("expected SUBSCRIPTION_OVER_HTTP error code")
	}
}

func TestQueryExecutor_Introspection(t *testing.T) {
	exec := NewQueryExecutor(NewResolver())

	resp := exec.Execute(context.Background(), "tenant-1", GraphQLRequest{
		Query: `{ __schema { types { name kind } } }`,
	})

	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected map data")
	}
	schema, ok := data["__schema"].(map[string]interface{})
	if !ok {
		t.Fatal("expected __schema in data")
	}
	types, ok := schema["types"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected types in schema")
	}
	if len(types) == 0 {
		t.Error("expected at least one type")
	}
}

func TestExtractTopLevelFields(t *testing.T) {
	tests := []struct {
		query     string
		minFields int
	}{
		{"query {\n  tenant {\n    id\n  }\n}", 1},
		{"{\n  tenant {\n    id\n  }\n  endpoints {\n    edges {\n      node {\n        id\n      }\n    }\n  }\n}", 2},
		{"mutation {\n  createEndpoint {\n    id\n  }\n}", 1},
	}

	for _, tt := range tests {
		t.Run(tt.query[:20], func(t *testing.T) {
			fields := extractTopLevelFields(tt.query)
			if len(fields) < tt.minFields {
				t.Errorf("expected at least %d fields, got %d: %v", tt.minFields, len(fields), fields)
			}
		})
	}
}

func TestSubscriptionManager(t *testing.T) {
	mgr := NewSubscriptionManager()
	ctx := context.Background()

	sub := mgr.Subscribe(ctx, "tenant-1", ChannelDeliveryUpdated, nil)
	if sub == nil {
		t.Fatal("expected subscriber")
	}

	// Publish a message
	mgr.Publish(ChannelDeliveryUpdated, "tenant-1", map[string]interface{}{"id": "del-1"})

	select {
	case msg := <-sub.Messages:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected message in channel")
	}

	// Unsubscribe
	mgr.Unsubscribe(sub)

	// Publish again - should not receive
	mgr.Publish(ChannelDeliveryUpdated, "tenant-1", map[string]interface{}{"id": "del-2"})
}

func TestSubscriptionManagerTenantIsolation(t *testing.T) {
	mgr := NewSubscriptionManager()
	ctx := context.Background()

	sub1 := mgr.Subscribe(ctx, "tenant-1", ChannelDeliveryUpdated, nil)
	sub2 := mgr.Subscribe(ctx, "tenant-2", ChannelDeliveryUpdated, nil)

	mgr.Publish(ChannelDeliveryUpdated, "tenant-1", map[string]interface{}{"id": "del-1"})

	select {
	case <-sub1.Messages:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Error("tenant-1 should receive the message")
	}

	select {
	case <-sub2.Messages:
		t.Error("tenant-2 should NOT receive tenant-1's message")
	case <-time.After(50 * time.Millisecond):
		// Expected — no message for tenant-2
	}

	mgr.Unsubscribe(sub1)
	mgr.Unsubscribe(sub2)
}

// --- Server Tests ---

func TestNewServer(t *testing.T) {
	resolver := NewResolver()
	server := NewServer(resolver)
	if server == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestServer_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api")

	resolver := NewResolver()
	server := NewServer(resolver)
	server.RegisterRoutes(group)

	// Verify routes are registered by making requests
	routes := router.Routes()
	foundPost := false
	foundGet := false
	foundPlayground := false
	for _, r := range routes {
		if r.Path == "/api/graphql" && r.Method == "POST" {
			foundPost = true
		}
		if r.Path == "/api/graphql" && r.Method == "GET" {
			foundGet = true
		}
		if r.Path == "/api/graphql/playground" && r.Method == "GET" {
			foundPlayground = true
		}
	}
	if !foundPost {
		t.Error("expected POST /api/graphql route")
	}
	if !foundGet {
		t.Error("expected GET /api/graphql route")
	}
	if !foundPlayground {
		t.Error("expected GET /api/graphql/playground route")
	}
}

func TestServer_HandleQuery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := NewResolver()
	server := NewServer(resolver)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query": "{ tenant { id } }"}`
	c.Request = httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// No tenant_id in context

	server.HandleQuery(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	var resp GraphQLResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Errors) == 0 || resp.Errors[0].Message != "Unauthorized" {
		t.Error("expected Unauthorized error")
	}
}

func TestServer_HandleQuery_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := NewResolver()
	server := NewServer(resolver)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/graphql", strings.NewReader("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	server.HandleQuery(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestServer_HandleQuery_ValidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := NewResolver()
	server := NewServer(resolver)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query": "{ tenant { id } }"}`
	c.Request = httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", "test-tenant")

	server.HandleQuery(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify response contains actual data, not a NOT_IMPLEMENTED error
	var resp GraphQLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Errorf("expected no errors, got: %s", resp.Errors[0].Message)
	}
	if resp.Data == nil {
		t.Error("expected data in response")
	}
}

func TestServer_AuthenticateToken(t *testing.T) {
	resolver := NewResolver()
	server := NewServer(resolver)

	// Valid base64 token
	encoded := base64.StdEncoding.EncodeToString([]byte("tenant-123"))
	result := server.authenticateToken(encoded)
	if result != "tenant-123" {
		t.Errorf("expected tenant-123, got %s", result)
	}

	// Invalid base64
	result = server.authenticateToken("not-valid-base64!!!")
	if result != "" {
		t.Errorf("expected empty string for invalid token, got %s", result)
	}
}

func TestServer_ParseSubscription(t *testing.T) {
	resolver := NewResolver()
	server := NewServer(resolver)

	tests := []struct {
		query    string
		vars     map[string]interface{}
		channel  string
		hasMatch bool
	}{
		{"subscription { deliveryUpdated { id } }", nil, ChannelDeliveryUpdated, true},
		{"subscription { anomalyDetected { id } }", nil, ChannelAnomalyDetected, true},
		{"subscription { metricsUpdated { value } }", nil, ChannelMetricsUpdated, true},
		{"subscription { unknownField { id } }", nil, "", false},
		{"", nil, "", false},
	}

	for _, tt := range tests {
		channel, _ := server.parseSubscription(tt.query, tt.vars)
		if tt.hasMatch && channel != tt.channel {
			t.Errorf("parseSubscription(%q) channel = %q, want %q", tt.query, channel, tt.channel)
		}
		if !tt.hasMatch && channel != "" {
			t.Errorf("parseSubscription(%q) expected empty channel, got %q", tt.query, channel)
		}
	}
}

// --- RealTimeSubscriptionEngine Tests ---

func TestSubscriptionManager_PublishSubscribe(t *testing.T) {
	mgr := NewSubscriptionManager()
	ctx := context.Background()

	sub := mgr.Subscribe(ctx, "t1", ChannelDeliveryUpdated, nil)

	event := &DeliveryUpdateEvent{ID: "d1", Status: "success"}
	mgr.Publish(ChannelDeliveryUpdated, "t1", event)

	select {
	case msg := <-sub.Messages:
		var decoded DeliveryUpdateEvent
		json.Unmarshal(msg, &decoded)
		if decoded.ID != "d1" {
			t.Errorf("expected ID d1, got %s", decoded.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected message")
	}

	mgr.Unsubscribe(sub)
}

func TestSubscriptionManager_FilterMatching(t *testing.T) {
	mgr := NewSubscriptionManager()
	ctx := context.Background()

	filter := map[string]interface{}{"endpointId": "ep-1"}
	sub := mgr.Subscribe(ctx, "t1", ChannelDeliveryUpdated, filter)

	// Matching message
	mgr.Publish(ChannelDeliveryUpdated, "t1", map[string]interface{}{"endpointId": "ep-1", "status": "ok"})

	select {
	case <-sub.Messages:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Error("expected message for matching filter")
	}

	// Non-matching message
	mgr.Publish(ChannelDeliveryUpdated, "t1", map[string]interface{}{"endpointId": "ep-2", "status": "ok"})

	select {
	case <-sub.Messages:
		t.Error("should NOT receive message for non-matching filter")
	case <-time.After(50 * time.Millisecond):
		// Expected — no message for non-matching filter
	}

	mgr.Unsubscribe(sub)
}

func TestSubscriptionManager_DoubleUnsubscribe(t *testing.T) {
	mgr := NewSubscriptionManager()
	ctx := context.Background()

	sub := mgr.Subscribe(ctx, "t1", ChannelDeliveryUpdated, nil)
	mgr.Unsubscribe(sub)
	mgr.Unsubscribe(sub) // Should not panic
}

func TestSubscriptionManager_Stats(t *testing.T) {
	mgr := NewSubscriptionManager()
	ctx := context.Background()

	sub1 := mgr.Subscribe(ctx, "t1", ChannelDeliveryUpdated, nil)
	sub2 := mgr.Subscribe(ctx, "t1", ChannelAnomalyDetected, nil)
	sub3 := mgr.Subscribe(ctx, "t2", ChannelDeliveryUpdated, nil)

	// 3 subscribers across 2 channels
	mgr.mu.RLock()
	deliverySubs := len(mgr.subscribers[ChannelDeliveryUpdated])
	anomalySubs := len(mgr.subscribers[ChannelAnomalyDetected])
	mgr.mu.RUnlock()

	if deliverySubs != 2 {
		t.Errorf("expected 2 delivery subscribers, got %d", deliverySubs)
	}
	if anomalySubs != 1 {
		t.Errorf("expected 1 anomaly subscriber, got %d", anomalySubs)
	}

	mgr.Unsubscribe(sub1)
	mgr.Unsubscribe(sub2)
	mgr.Unsubscribe(sub3)

	// After unsubscribe, channels should be cleaned up
	mgr.mu.RLock()
	remaining := len(mgr.subscribers)
	mgr.mu.RUnlock()
	if remaining != 0 {
		t.Errorf("expected 0 channels after unsubscribe, got %d", remaining)
	}
}

func TestResolver_NotifyDeliveryUpdate(t *testing.T) {
	resolver := NewResolver()
	ctx := context.Background()
	mgr := resolver.GetSubscriptionManager()

	sub := mgr.Subscribe(ctx, "t1", ChannelDeliveryUpdated, nil)

	resolver.NotifyDeliveryUpdate("t1", &DeliveryUpdateEvent{
		ID:     "del-1",
		Status: "success",
	})

	select {
	case msg := <-sub.Messages:
		var event DeliveryUpdateEvent
		json.Unmarshal(msg, &event)
		if event.Status != "success" {
			t.Errorf("expected success, got %s", event.Status)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected delivery update message")
	}

	mgr.Unsubscribe(sub)
}

// --- Schema Tests ---

func TestSchemaGA_AllTypes(t *testing.T) {
	// Verify the schema string contains expected types
	exec := NewQueryExecutor(NewResolver())
	resp := exec.Execute(context.Background(), "t1", GraphQLRequest{
		Query: `{ __schema { types { name kind } } }`,
	})

	if len(resp.Errors) > 0 {
		t.Fatalf("introspection errors: %v", resp.Errors)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected map data")
	}
	schema := data["__schema"].(map[string]interface{})
	types := schema["types"].([]map[string]interface{})

	typeNames := make(map[string]bool)
	for _, typ := range types {
		typeNames[typ["name"].(string)] = true
	}

	required := []string{"Query", "Mutation", "Subscription"}
	for _, name := range required {
		if !typeNames[name] {
			t.Errorf("schema missing required type: %s", name)
		}
	}
}

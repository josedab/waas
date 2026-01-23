package graphql

import (
	"context"
	"testing"
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
		query   string
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
	default:
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
	default:
		t.Error("tenant-1 should receive the message")
	}

	select {
	case <-sub2.Messages:
		t.Error("tenant-2 should NOT receive tenant-1's message")
	default:
		// Expected
	}

	mgr.Unsubscribe(sub1)
	mgr.Unsubscribe(sub2)
}

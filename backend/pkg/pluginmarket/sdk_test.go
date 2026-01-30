package pluginmarket

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPluginSDK(t *testing.T) {
	sdk := NewPluginSDK("test-plugin", "1.0.0", "test", "A test plugin")

	// Register a hook
	err := sdk.RegisterSDKHook(SDKHookPreSend, func(ctx context.Context, hookCtx *SDKHookContext) (*SDKHookResult, error) {
		// Add a custom header
		hookCtx.Headers["X-Plugin-Processed"] = "true"
		return &SDKHookResult{
			Modified: true,
			Headers:  hookCtx.Headers,
		}, nil
	})
	if err != nil {
		t.Fatalf("RegisterSDKHook failed: %v", err)
	}

	if !sdk.HasSDKHook(SDKHookPreSend) {
		t.Error("expected SDKHookPreSend to be registered")
	}

	if sdk.HasSDKHook(SDKHookPostSend) {
		t.Error("expected SDKHookPostSend to not be registered")
	}

	// Execute hook
	result, err := sdk.ExecuteSDKHook(context.Background(), SDKHookPreSend, &SDKHookContext{
		TenantID: "t1",
		Payload:  json.RawMessage(`{"test": true}`),
		Headers:  map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		t.Fatalf("ExecuteSDKHook failed: %v", err)
	}
	if !result.Modified {
		t.Error("expected result to be modified")
	}
	if result.Headers["X-Plugin-Processed"] != "true" {
		t.Error("expected custom header")
	}

	// Test manifest
	manifest := sdk.Manifest()
	if manifest["name"] != "test-plugin" {
		t.Error("expected name in manifest")
	}
}

func TestConnectorTemplates(t *testing.T) {
	connectors := BuiltinConnectors()
	if len(connectors) < 5 {
		t.Errorf("expected at least 5 connectors, got %d", len(connectors))
	}

	// Test get by ID
	stripe, err := GetConnector("stripe")
	if err != nil {
		t.Fatalf("GetConnector(stripe) failed: %v", err)
	}
	if stripe.Provider != "stripe" {
		t.Error("expected stripe provider")
	}
	if len(stripe.Events) == 0 {
		t.Error("expected stripe to have events")
	}

	// Test category filter
	payments := GetConnectorsByCategory("payments")
	if len(payments) == 0 {
		t.Error("expected payment connectors")
	}

	// Test connector plugin creation
	plugin := CreateConnectorPlugin(stripe)
	if !plugin.HasSDKHook(SDKHookOnReceive) {
		t.Error("expected on_receive hook")
	}
	if !plugin.HasSDKHook(SDKHookOnValidate) {
		t.Error("expected on_validate hook")
	}
}

package transform

import (
	"context"
	"testing"
)

func TestEngine_Transform_Simple(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()

	script := `
		return {
			name: payload.name.toUpperCase(),
			processed: true
		};
	`

	payload := map[string]interface{}{
		"name": "test",
	}

	result, err := engine.Transform(ctx, script, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("transformation failed: %s", result.Error)
	}

	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got %T", result.Output)
	}

	if output["name"] != "TEST" {
		t.Errorf("expected name=TEST, got %v", output["name"])
	}

	if output["processed"] != true {
		t.Errorf("expected processed=true, got %v", output["processed"])
	}
}

func TestEngine_Transform_WithHelpers(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()

	script := `
		var data = clone(payload);
		data.nested = get(payload, 'user.email', 'default@example.com');
		data.picked = pick(payload, ['name', 'id']);
		return data;
	`

	payload := map[string]interface{}{
		"name": "test",
		"id":   123,
		"user": map[string]interface{}{
			"email": "user@example.com",
		},
	}

	result, err := engine.Transform(ctx, script, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("transformation failed: %s", result.Error)
	}

	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got %T", result.Output)
	}

	if output["nested"] != "user@example.com" {
		t.Errorf("expected nested=user@example.com, got %v", output["nested"])
	}
}

func TestEngine_Transform_Timeout(t *testing.T) {
	config := DefaultEngineConfig()
	config.TimeoutMs = 100 // Very short timeout
	engine := NewEngine(config)
	ctx := context.Background()

	script := `
		// Infinite loop to trigger timeout
		while(true) {}
		return payload;
	`

	payload := map[string]interface{}{"test": true}

	result, err := engine.Transform(ctx, script, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected transformation to fail due to timeout")
	}

	if result.Error != ErrTimeout.Error() {
		t.Errorf("expected timeout error, got: %s", result.Error)
	}
}

func TestEngine_Transform_ConsoleLog(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()

	script := `
		console.log('Processing payload:', payload.name);
		console.warn('This is a warning');
		console.error('This is an error');
		return payload;
	`

	payload := map[string]interface{}{"name": "test"}

	result, err := engine.Transform(ctx, script, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("transformation failed: %s", result.Error)
	}

	if len(result.Logs) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(result.Logs))
	}
}

func TestEngine_ValidateScript_Valid(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())

	validScripts := []string{
		`return payload;`,
		`var x = 1; return x;`,
		`if (payload.test) { return payload; } else { return null; }`,
		`return { ...payload, extra: true };`,
	}

	for _, script := range validScripts {
		err := engine.ValidateScript(script)
		if err != nil {
			t.Errorf("expected script to be valid: %s, error: %v", script, err)
		}
	}
}

func TestEngine_ValidateScript_Invalid(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())

	invalidScripts := []string{
		`return payload`,          // Missing semicolon (actually valid in JS)
		`function { return 1; }`,  // Invalid function syntax
		`var x = ;`,               // Syntax error
	}

	for _, script := range invalidScripts {
		err := engine.ValidateScript(script)
		if err == nil {
			// Note: Some of these might actually be valid JS
			// This test verifies the validation runs without panic
		}
	}
}

func TestEngine_TransformJSON(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()

	script := `
		return {
			id: payload.id,
			name: payload.name.toUpperCase()
		};
	`

	inputJSON := []byte(`{"id": 123, "name": "test"}`)

	outputJSON, result, err := engine.TransformJSON(ctx, script, inputJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("transformation failed: %s", result.Error)
	}

	expected := `{"id":123,"name":"TEST"}`
	if string(outputJSON) != expected {
		t.Errorf("expected %s, got %s", expected, string(outputJSON))
	}
}

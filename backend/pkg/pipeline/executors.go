package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// --- Transform Stage Executor ---

type transformExecutor struct{}

func (e *transformExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg TransformConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid transform config: %w", err)
	}

	if cfg.Script == "" {
		return input, nil
	}

	result, err := evalJS(cfg.Script, input)
	if err != nil {
		return nil, fmt.Errorf("transform script failed: %w", err)
	}

	return result, nil
}

// --- Validate Stage Executor ---

type validateExecutor struct{}

func (e *validateExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg ValidateConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid validate config: %w", err)
	}

	if len(cfg.Schema) == 0 {
		return input, nil // No schema = pass through
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, fmt.Errorf("payload is not valid JSON object: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(cfg.Schema, &schema); err != nil {
		return nil, fmt.Errorf("schema is not valid JSON: %w", err)
	}

	violations := validatePayloadSchema(payload, schema, cfg.Strictness)

	if len(violations) > 0 {
		rejectOn := cfg.RejectOn
		if rejectOn == "" {
			rejectOn = "error"
		}

		hasErrors := false
		for _, v := range violations {
			if v.Severity == "error" {
				hasErrors = true
				break
			}
		}

		if rejectOn == "error" && hasErrors {
			violationsJSON, _ := json.Marshal(violations)
			return nil, fmt.Errorf("validation failed: %s", string(violationsJSON))
		}
		if rejectOn == "warning" && len(violations) > 0 {
			violationsJSON, _ := json.Marshal(violations)
			return nil, fmt.Errorf("validation warnings: %s", string(violationsJSON))
		}
	}

	return input, nil
}

type validationViolation struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

func validatePayloadSchema(payload, schema map[string]interface{}, strictness string) []validationViolation {
	var violations []validationViolation

	props, _ := schema["properties"].(map[string]interface{})
	if props == nil {
		return violations
	}

	required, _ := schema["required"].([]interface{})
	requiredSet := make(map[string]bool)
	for _, r := range required {
		if s, ok := r.(string); ok {
			requiredSet[s] = true
		}
	}

	for name := range requiredSet {
		if _, exists := payload[name]; !exists {
			violations = append(violations, validationViolation{
				Path:     name,
				Message:  fmt.Sprintf("required field '%s' is missing", name),
				Severity: "error",
			})
		}
	}

	for name, value := range payload {
		propDef, exists := props[name]
		if !exists && strictness == "strict" {
			violations = append(violations, validationViolation{
				Path:     name,
				Message:  fmt.Sprintf("unexpected field '%s'", name),
				Severity: "warning",
			})
			continue
		}

		if propMap, ok := propDef.(map[string]interface{}); ok {
			if expectedType, ok := propMap["type"].(string); ok {
				if !typeMatches(value, expectedType) {
					violations = append(violations, validationViolation{
						Path:     name,
						Message:  fmt.Sprintf("expected type '%s' for field '%s'", expectedType, name),
						Severity: "error",
					})
				}
			}
		}
	}

	return violations
}

func typeMatches(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number", "integer":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	}
	return true
}

// --- Filter Stage Executor ---

type filterExecutor struct{}

func (e *filterExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg FilterConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid filter config: %w", err)
	}

	if cfg.Condition == "" {
		return input, nil
	}

	result, err := evalJSBool(cfg.Condition, input)
	if err != nil {
		return nil, fmt.Errorf("filter condition failed: %w", err)
	}

	if !result {
		onReject := cfg.OnReject
		if onReject == "" {
			onReject = "drop"
		}
		return nil, fmt.Errorf("filtered out (action: %s)", onReject)
	}

	return input, nil
}

// --- Enrich Stage Executor ---

type enrichExecutor struct{}

func (e *enrichExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg EnrichConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid enrich config: %w", err)
	}

	if cfg.Script == "" {
		return input, nil
	}

	return evalJS(cfg.Script, input)
}

// --- Route Stage Executor ---

type routeExecutor struct{}

func (e *routeExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg RouteConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid route config: %w", err)
	}

	if len(cfg.Rules) == 0 {
		return input, nil
	}

	// Evaluate rules and annotate payload with routing info
	var payload map[string]interface{}
	if err := json.Unmarshal(input, &payload); err != nil {
		return input, nil
	}

	matchedEndpoints := make([]string, 0)
	for _, rule := range cfg.Rules {
		if rule.Condition == "" {
			matchedEndpoints = append(matchedEndpoints, rule.EndpointIDs...)
			continue
		}

		matches, err := evalJSBool(rule.Condition, input)
		if err == nil && matches {
			matchedEndpoints = append(matchedEndpoints, rule.EndpointIDs...)
		}
	}

	// Add routing metadata to payload
	payload["_pipeline_routed_endpoints"] = matchedEndpoints
	result, _ := json.Marshal(payload)
	return result, nil
}

// --- Fan-Out Stage Executor ---

type fanOutExecutor struct {
	service *Service
}

func (e *fanOutExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg FanOutConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid fan-out config: %w", err)
	}

	// Check if route stage already determined endpoints
	var payload map[string]interface{}
	if err := json.Unmarshal(input, &payload); err == nil {
		if routed, ok := payload["_pipeline_routed_endpoints"].([]interface{}); ok && len(routed) > 0 {
			for _, ep := range routed {
				if epStr, ok := ep.(string); ok {
					cfg.EndpointIDs = append(cfg.EndpointIDs, epStr)
				}
			}
			delete(payload, "_pipeline_routed_endpoints")
			input, _ = json.Marshal(payload)
		}
	}

	if len(cfg.EndpointIDs) == 0 {
		return input, nil
	}

	// Record fan-out results
	results := make([]map[string]interface{}, 0, len(cfg.EndpointIDs))

	if e.service != nil && e.service.deliverFunc != nil {
		for _, epID := range cfg.EndpointIDs {
			err := e.service.deliverFunc(ctx, epID, input, nil)
			result := map[string]interface{}{
				"endpoint_id": epID,
				"success":     err == nil,
			}
			if err != nil {
				result["error"] = err.Error()
			}
			results = append(results, result)
		}
	} else {
		for _, epID := range cfg.EndpointIDs {
			results = append(results, map[string]interface{}{
				"endpoint_id": epID,
				"success":     true,
				"note":        "delivery function not configured",
			})
		}
	}

	output := map[string]interface{}{
		"fan_out_results": results,
		"original":        json.RawMessage(input),
	}
	result, _ := json.Marshal(output)
	return result, nil
}

// --- Deliver Stage Executor ---

type deliverExecutor struct {
	service *Service
}

func (e *deliverExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg DeliverConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid deliver config: %w", err)
	}

	if e.service == nil || e.service.deliverFunc == nil {
		return input, nil // Pass-through if no delivery function configured
	}

	endpointID := cfg.EndpointID
	if endpointID == "" {
		// Try to extract from payload metadata
		var payload map[string]interface{}
		if err := json.Unmarshal(input, &payload); err == nil {
			if ep, ok := payload["_endpoint_id"].(string); ok {
				endpointID = ep
			}
		}
	}

	if endpointID == "" {
		return input, nil // No target endpoint, pass through
	}

	if err := e.service.deliverFunc(ctx, endpointID, input, cfg.CustomHeaders); err != nil {
		return nil, fmt.Errorf("delivery failed: %w", err)
	}

	result, _ := json.Marshal(map[string]interface{}{
		"delivered":   true,
		"endpoint_id": endpointID,
	})
	return result, nil
}

// --- Delay Stage Executor ---

type delayExecutor struct{}

func (e *delayExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	var cfg DelayConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid delay config: %w", err)
	}

	if cfg.DurationMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(cfg.DurationMs) * time.Millisecond):
		}
	}

	return input, nil
}

// --- Log Stage Executor ---

type logExecutor struct{}

func (e *logExecutor) Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	// Log stage is a pass-through that records the current state
	return input, nil
}

// --- JS evaluation helpers ---

func evaluateCondition(condition string, payload json.RawMessage) (bool, error) {
	return evalJSBool(condition, payload)
}

func evalJS(script string, input json.RawMessage) (json.RawMessage, error) {
	vm := goja.New()

	// Parse input and make it available as 'payload'
	var payloadObj interface{}
	if err := json.Unmarshal(input, &payloadObj); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	if err := vm.Set("payload", payloadObj); err != nil {
		return nil, fmt.Errorf("failed to set payload: %w", err)
	}

	// Wrap script to return a value
	wrappedScript := fmt.Sprintf(`(function() { %s })()`, script)
	val, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// If the script returned undefined/null, return original input
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return input, nil
	}

	// Export result to Go and marshal back to JSON
	result := val.Export()
	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize script output: %w", err)
	}

	return output, nil
}

func evalJSBool(script string, input json.RawMessage) (bool, error) {
	vm := goja.New()

	var payloadObj interface{}
	if err := json.Unmarshal(input, &payloadObj); err != nil {
		return false, fmt.Errorf("invalid JSON input: %w", err)
	}
	if err := vm.Set("payload", payloadObj); err != nil {
		return false, fmt.Errorf("failed to set payload: %w", err)
	}

	wrappedScript := fmt.Sprintf(`(function() { %s })()`, script)
	val, err := vm.RunString(wrappedScript)
	if err != nil {
		return false, fmt.Errorf("condition evaluation failed: %w", err)
	}

	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return false, nil
	}

	return val.ToBoolean(), nil
}

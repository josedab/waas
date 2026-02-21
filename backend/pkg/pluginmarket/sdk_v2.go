package pluginmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// PluginHookTypeV2 represents hook points in the webhook lifecycle
type PluginHookTypeV2 string

const (
	HookBeforeDelivery PluginHookTypeV2 = "before_delivery"
	HookAfterDelivery  PluginHookTypeV2 = "after_delivery"
	HookOnFailureV2    PluginHookTypeV2 = "on_failure"
	HookTransformV2    PluginHookTypeV2 = "transform"
	HookValidateV2     PluginHookTypeV2 = "validate"
)

// PluginManifest describes a plugin's metadata and capabilities
type PluginManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Permissions  []string          `json:"permissions,omitempty"`
	Hooks        []PluginHookTypeV2 `json:"hooks"`
	ConfigSchema json.RawMessage   `json:"config_schema,omitempty"`
}

// SandboxConfig defines resource limits for sandboxed plugin execution
type SandboxConfig struct {
	TimeoutMs    int `json:"timeout_ms"`
	MaxMemoryKB  int `json:"max_memory_kb"`
}

// DefaultSandboxConfig returns sensible defaults
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		TimeoutMs:   5000,
		MaxMemoryKB: 65536,
	}
}

// SandboxResult captures the outcome of a sandboxed execution
type SandboxResult struct {
	Output     map[string]interface{} `json:"output,omitempty"`
	DurationMs int64                  `json:"duration_ms"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
}

// PluginSandbox manages sandboxed JavaScript execution via goja
type PluginSandbox struct {
	mu     sync.Mutex
	config SandboxConfig
}

// NewPluginSandbox creates a new sandbox with the given configuration
func NewPluginSandbox(config SandboxConfig) *PluginSandbox {
	return &PluginSandbox{config: config}
}

// Execute runs JavaScript source inside a sandboxed goja VM with timeout and error isolation
func (s *PluginSandbox) Execute(ctx context.Context, source string, input map[string]interface{}) (*SandboxResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vm := goja.New()

	// Inject input as a global variable
	if err := vm.Set("input", input); err != nil {
		return nil, fmt.Errorf("failed to set input: %w", err)
	}

	timeout := time.Duration(s.config.TimeoutMs) * time.Millisecond
	start := time.Now()

	// Set up timeout via interrupt
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt("execution timeout exceeded")
	})
	defer timer.Stop()

	val, err := vm.RunString(source)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return &SandboxResult{
			Success:    false,
			Error:      err.Error(),
			DurationMs: duration,
		}, nil
	}

	// Extract result
	output := make(map[string]interface{})
	if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
		exported := val.Export()
		if m, ok := exported.(map[string]interface{}); ok {
			output = m
		} else {
			output["result"] = exported
		}
	}

	return &SandboxResult{
		Output:     output,
		DurationMs: duration,
		Success:    true,
	}, nil
}

// MockWebhookEvent creates a mock webhook event for plugin testing
type MockWebhookEvent struct {
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
	Headers   map[string]string      `json:"headers"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewMockWebhookEvent creates a mock event with defaults
func NewMockWebhookEvent(eventType string, payload map[string]interface{}) *MockWebhookEvent {
	return &MockWebhookEvent{
		EventType: eventType,
		Payload:   payload,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Timestamp: time.Now(),
	}
}

// PluginTestRunner provides local testing of plugins with mock webhook events
type PluginTestRunner struct {
	sandbox  *PluginSandbox
	manifest *PluginManifest
}

// NewPluginTestRunner creates a test runner for a given manifest
func NewPluginTestRunner(manifest *PluginManifest, config SandboxConfig) *PluginTestRunner {
	return &PluginTestRunner{
		sandbox:  NewPluginSandbox(config),
		manifest: manifest,
	}
}

// RunHook executes a JavaScript source against a mock event for a given hook type
func (r *PluginTestRunner) RunHook(ctx context.Context, hook PluginHookTypeV2, source string, event *MockWebhookEvent) (*SandboxResult, error) {
	if !r.supportsHook(hook) {
		return nil, fmt.Errorf("manifest does not declare hook %q", hook)
	}

	input := map[string]interface{}{
		"event_type": event.EventType,
		"payload":    event.Payload,
		"headers":    event.Headers,
		"timestamp":  event.Timestamp.Format(time.RFC3339),
	}
	return r.sandbox.Execute(ctx, source, input)
}

func (r *PluginTestRunner) supportsHook(hook PluginHookTypeV2) bool {
	for _, h := range r.manifest.Hooks {
		if h == hook {
			return true
		}
	}
	return false
}

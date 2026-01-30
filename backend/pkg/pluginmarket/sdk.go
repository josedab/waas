package pluginmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PluginSDK provides the framework for building WaaS plugins
type PluginSDK struct {
	Name        string
	Version     string
	Author      string
	Description string
	hooks       map[SDKHookType]SDKHookFunc
	config      map[string]interface{}
}

// SDKHookType represents the type of webhook lifecycle hook
type SDKHookType string

const (
	SDKHookPreSend       SDKHookType = "pre_send"
	SDKHookPostSend      SDKHookType = "post_send"
	SDKHookOnFailure     SDKHookType = "on_failure"
	SDKHookOnRetry       SDKHookType = "on_retry"
	SDKHookPreTransform  SDKHookType = "pre_transform"
	SDKHookPostTransform SDKHookType = "post_transform"
	SDKHookOnReceive     SDKHookType = "on_receive"
	SDKHookOnValidate    SDKHookType = "on_validate"
)

// SDKHookContext provides context to hook functions
type SDKHookContext struct {
	TenantID   string                 `json:"tenant_id"`
	EndpointID string                 `json:"endpoint_id"`
	EventType  string                 `json:"event_type"`
	Payload    json.RawMessage        `json:"payload"`
	Headers    map[string]string      `json:"headers"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
}

// SDKHookResult represents the output of a hook execution
type SDKHookResult struct {
	Modified bool                   `json:"modified"`
	Payload  json.RawMessage        `json:"payload,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Skip     bool                   `json:"skip"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SDKHookFunc is the function signature for plugin hooks
type SDKHookFunc func(ctx context.Context, hookCtx *SDKHookContext) (*SDKHookResult, error)

// NewPluginSDK creates a new Plugin SDK instance
func NewPluginSDK(name, version, author, description string) *PluginSDK {
	return &PluginSDK{
		Name:        name,
		Version:     version,
		Author:      author,
		Description: description,
		hooks:       make(map[SDKHookType]SDKHookFunc),
		config:      make(map[string]interface{}),
	}
}

// RegisterSDKHook registers a hook function for a specific lifecycle event
func (sdk *PluginSDK) RegisterSDKHook(hookType SDKHookType, fn SDKHookFunc) error {
	if fn == nil {
		return fmt.Errorf("hook function cannot be nil")
	}
	sdk.hooks[hookType] = fn
	return nil
}

// ExecuteSDKHook runs a registered hook and returns the result
func (sdk *PluginSDK) ExecuteSDKHook(ctx context.Context, hookType SDKHookType, hookCtx *SDKHookContext) (*SDKHookResult, error) {
	fn, ok := sdk.hooks[hookType]
	if !ok {
		return &SDKHookResult{Modified: false}, nil
	}
	return fn(ctx, hookCtx)
}

// HasSDKHook checks if a hook is registered
func (sdk *PluginSDK) HasSDKHook(hookType SDKHookType) bool {
	_, ok := sdk.hooks[hookType]
	return ok
}

// GetRegisteredSDKHooks returns all registered hook types
func (sdk *PluginSDK) GetRegisteredSDKHooks() []SDKHookType {
	hooks := make([]SDKHookType, 0, len(sdk.hooks))
	for h := range sdk.hooks {
		hooks = append(hooks, h)
	}
	return hooks
}

// SetConfig sets a configuration value
func (sdk *PluginSDK) SetConfig(key string, value interface{}) {
	sdk.config[key] = value
}

// GetConfig retrieves a configuration value
func (sdk *PluginSDK) GetConfig(key string) (interface{}, bool) {
	v, ok := sdk.config[key]
	return v, ok
}

// Manifest returns the plugin manifest for marketplace registration
func (sdk *PluginSDK) Manifest() map[string]interface{} {
	hooks := make([]string, 0, len(sdk.hooks))
	for h := range sdk.hooks {
		hooks = append(hooks, string(h))
	}

	return map[string]interface{}{
		"name":        sdk.Name,
		"version":     sdk.Version,
		"author":      sdk.Author,
		"description": sdk.Description,
		"hooks":       hooks,
		"config":      sdk.config,
	}
}

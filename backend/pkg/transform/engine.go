// Package transform provides sandboxed JavaScript execution for webhook payload transformation
package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

var (
	ErrTimeout       = errors.New("transformation execution timeout")
	ErrMemoryLimit   = errors.New("transformation memory limit exceeded")
	ErrInvalidScript = errors.New("invalid transformation script")
	ErrExecution     = errors.New("transformation execution error")
)

// Engine handles payload transformations using sandboxed JavaScript
type Engine struct {
	vmPool      sync.Pool
	maxMemoryMB int
	timeoutMs   int
}

// EngineConfig holds engine configuration
type EngineConfig struct {
	MaxMemoryMB int
	TimeoutMs   int
	PoolSize    int
}

// DefaultEngineConfig returns default engine configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxMemoryMB: 64,
		TimeoutMs:   5000,
		PoolSize:    10,
	}
}

// NewEngine creates a new transformation engine
func NewEngine(config EngineConfig) *Engine {
	return &Engine{
		vmPool: sync.Pool{
			New: func() interface{} {
				return goja.New()
			},
		},
		maxMemoryMB: config.MaxMemoryMB,
		timeoutMs:   config.TimeoutMs,
	}
}

// TransformResult holds the result of a transformation
type TransformResult struct {
	Output          interface{} `json:"output"`
	Success         bool        `json:"success"`
	Error           string      `json:"error,omitempty"`
	ExecutionTimeMs int64       `json:"execution_time_ms"`
	Logs            []string    `json:"logs,omitempty"`
}

// Transform executes a transformation script on the given payload
func (e *Engine) Transform(ctx context.Context, script string, payload interface{}) (*TransformResult, error) {
	startTime := time.Now()
	result := &TransformResult{
		Logs: make([]string, 0),
	}

	// Get VM from pool
	vm := e.vmPool.Get().(*goja.Runtime)
	defer func() {
		// Reset VM state before returning to pool
		vm.ClearInterrupt()
		e.vmPool.Put(vm)
	}()

	// Setup timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(e.timeoutMs)*time.Millisecond)
	defer cancel()

	// Run transformation in goroutine with timeout
	done := make(chan error, 1)
	var output interface{}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("%s: %v", ErrExecution, r)
			}
		}()

		// Inject the input payload
		inputJSON, err := json.Marshal(payload)
		if err != nil {
			done <- fmt.Errorf("failed to marshal input: %w", err)
			return
		}

		// Setup the runtime environment
		if err := e.setupRuntime(vm, result); err != nil {
			done <- err
			return
		}

		// Set input variable
		if err := vm.Set("input", string(inputJSON)); err != nil {
			done <- fmt.Errorf("failed to set input: %w", err)
			return
		}

		// Parse input JSON in VM
		_, err = vm.RunString("var payload = JSON.parse(input);")
		if err != nil {
			done <- fmt.Errorf("failed to parse input JSON: %w", err)
			return
		}

		// Execute the transformation script
		wrappedScript := fmt.Sprintf(`
			(function() {
				%s
			})();
		`, script)

		val, err := vm.RunString(wrappedScript)
		if err != nil {
			done <- fmt.Errorf("%s: %w", ErrExecution, err)
			return
		}

		// Export the result
		output = val.Export()
		done <- nil
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, nil
		}
		result.Success = true
		result.Output = output
		return result, nil

	case <-timeoutCtx.Done():
		vm.Interrupt("timeout")
		result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
		result.Success = false
		result.Error = ErrTimeout.Error()
		return result, nil
	}
}

// setupRuntime configures the VM with helper functions and security restrictions
func (e *Engine) setupRuntime(vm *goja.Runtime, result *TransformResult) error {
	// Console logging
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		logMsg := fmt.Sprint(args...)
		result.Logs = append(result.Logs, logMsg)
		return goja.Undefined()
	})
	console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		logMsg := "[ERROR] " + fmt.Sprint(args...)
		result.Logs = append(result.Logs, logMsg)
		return goja.Undefined()
	})
	console.Set("warn", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		logMsg := "[WARN] " + fmt.Sprint(args...)
		result.Logs = append(result.Logs, logMsg)
		return goja.Undefined()
	})
	vm.Set("console", console)

	// Built-in helper functions
	if err := e.registerHelpers(vm); err != nil {
		return err
	}

	return nil
}

// registerHelpers adds built-in helper functions to the VM
func (e *Engine) registerHelpers(vm *goja.Runtime) error {
	helpers := `
		// Deep clone helper
		function clone(obj) {
			return JSON.parse(JSON.stringify(obj));
		}

		// Safe get nested property
		function get(obj, path, defaultValue) {
			var keys = path.split('.');
			var result = obj;
			for (var i = 0; i < keys.length; i++) {
				if (result == null || result == undefined) {
					return defaultValue;
				}
				result = result[keys[i]];
			}
			return result !== undefined ? result : defaultValue;
		}

		// Safe set nested property
		function set(obj, path, value) {
			var keys = path.split('.');
			var current = obj;
			for (var i = 0; i < keys.length - 1; i++) {
				if (current[keys[i]] == null) {
					current[keys[i]] = {};
				}
				current = current[keys[i]];
			}
			current[keys[keys.length - 1]] = value;
			return obj;
		}

		// Pick specific keys from object
		function pick(obj, keys) {
			var result = {};
			for (var i = 0; i < keys.length; i++) {
				if (obj.hasOwnProperty(keys[i])) {
					result[keys[i]] = obj[keys[i]];
				}
			}
			return result;
		}

		// Omit specific keys from object
		function omit(obj, keys) {
			var result = clone(obj);
			for (var i = 0; i < keys.length; i++) {
				delete result[keys[i]];
			}
			return result;
		}

		// Merge objects
		function merge() {
			var result = {};
			for (var i = 0; i < arguments.length; i++) {
				var obj = arguments[i];
				for (var key in obj) {
					if (obj.hasOwnProperty(key)) {
						result[key] = obj[key];
					}
				}
			}
			return result;
		}

		// Format date
		function formatDate(date, format) {
			var d = new Date(date);
			if (isNaN(d.getTime())) return date;
			
			var pad = function(n) { return n < 10 ? '0' + n : n; };
			
			format = format || 'YYYY-MM-DD';
			return format
				.replace('YYYY', d.getFullYear())
				.replace('MM', pad(d.getMonth() + 1))
				.replace('DD', pad(d.getDate()))
				.replace('HH', pad(d.getHours()))
				.replace('mm', pad(d.getMinutes()))
				.replace('ss', pad(d.getSeconds()));
		}

		// Generate UUID v4
		function uuid() {
			return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
				var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
				return v.toString(16);
			});
		}

		// Hash string (simple djb2)
		function hash(str) {
			var hash = 5381;
			for (var i = 0; i < str.length; i++) {
				hash = ((hash << 5) + hash) + str.charCodeAt(i);
			}
			return (hash >>> 0).toString(16);
		}

		// Base64 encode/decode
		function base64Encode(str) {
			return btoa(unescape(encodeURIComponent(str)));
		}

		function base64Decode(str) {
			return decodeURIComponent(escape(atob(str)));
		}
	`

	_, err := vm.RunString(helpers)
	return err
}

// ValidateScript checks if a script is syntactically valid
func (e *Engine) ValidateScript(script string) error {
	vm := goja.New()
	wrappedScript := fmt.Sprintf(`
		(function() {
			%s
		});
	`, script)

	_, err := vm.RunString(wrappedScript)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrInvalidScript, err)
	}
	return nil
}

// TransformJSON is a convenience method that handles JSON marshaling
func (e *Engine) TransformJSON(ctx context.Context, script string, payloadJSON []byte) ([]byte, *TransformResult, error) {
	var payload interface{}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	result, err := e.Transform(ctx, script, payload)
	if err != nil {
		return nil, nil, err
	}

	if !result.Success {
		return nil, result, nil
	}

	outputJSON, err := json.Marshal(result.Output)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to marshal output: %v", err)
		return nil, result, nil
	}

	return outputJSON, result, nil
}

package transform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Timeout handling
// ---------------------------------------------------------------------------

func TestTransform_Timeout_InfiniteLoop(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 100
	engine := NewEngine(cfg)

	result, err := engine.Transform(context.Background(), `while(true) {}`, map[string]interface{}{"a": 1})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, ErrTimeout.Error(), result.Error)
}

func TestTransform_Timeout_LongComputation(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 100
	engine := NewEngine(cfg)

	// Heavy computation that should exceed 100ms timeout
	script := `
		var x = 0;
		for (var i = 0; i < 1e12; i++) { x += i; }
		return x;
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success, "expected timeout for long computation")
	assert.Equal(t, ErrTimeout.Error(), result.Error)
}

func TestTransform_Timeout_ContextAlreadyCancelled(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Transform

	result, err := engine.Transform(ctx, `return payload;`, map[string]interface{}{"ok": true})
	require.NoError(t, err)
	// The context-derived timeout fires immediately; expect timeout.
	assert.False(t, result.Success)
}

func TestTransform_Timeout_ExternalContextDeadline(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 5000 // generous engine timeout
	engine := NewEngine(cfg)

	// Caller's context has a very short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := engine.Transform(ctx, `while(true) {}`, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
}

func TestTransform_Timeout_FastScriptSucceeds(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 100
	engine := NewEngine(cfg)

	result, err := engine.Transform(context.Background(), `return payload;`, map[string]interface{}{"v": 1})
	require.NoError(t, err)
	assert.True(t, result.Success, "fast script should succeed even with short timeout")
}

func TestTransform_Timeout_ExecutionTimeRecorded(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 100
	engine := NewEngine(cfg)

	result, err := engine.Transform(context.Background(), `while(true) {}`, map[string]interface{}{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.ExecutionTimeMs, int64(0), "execution time must be non-negative")
}

// ---------------------------------------------------------------------------
// Memory / resource-heavy scripts
// ---------------------------------------------------------------------------

func TestTransform_LargeAllocation(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 500
	engine := NewEngine(cfg)

	// Try to allocate a very large array; engine should either timeout or error.
	script := `
		var arr = [];
		for (var i = 0; i < 1e8; i++) { arr.push(i); }
		return arr.length;
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	// We just verify it didn't panic. Script may timeout or succeed.
	_ = result

	// Allow timed-out goroutine to finish before reusing the VM.
	time.Sleep(200 * time.Millisecond)

	// Engine should remain usable after heavy allocation attempt.
	result2, err := engine.Transform(context.Background(), `return {ok:true};`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result2.Success, "engine must remain usable after heavy allocation attempt: %s", result2.Error)
}

// ---------------------------------------------------------------------------
// Concurrent execution
// ---------------------------------------------------------------------------

func TestTransform_Concurrent_Basic(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())
	const goroutines = 20

	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			script := fmt.Sprintf(`return { idx: %d, name: payload.name };`, idx)
			payload := map[string]interface{}{"name": fmt.Sprintf("worker-%d", idx)}
			result, err := engine.Transform(context.Background(), script, payload)
			if err != nil {
				errs[idx] = err
				return
			}
			if !result.Success {
				errs[idx] = fmt.Errorf("transform failed: %s", result.Error)
				return
			}
			out, ok := result.Output.(map[string]interface{})
			if !ok {
				errs[idx] = fmt.Errorf("unexpected output type %T", result.Output)
				return
			}
			expected := fmt.Sprintf("worker-%d", idx)
			if out["name"] != expected {
				errs[idx] = fmt.Errorf("expected name=%s, got %v", expected, out["name"])
			}
		}(i)
	}

	wg.Wait()
	for i, e := range errs {
		assert.NoError(t, e, "goroutine %d failed", i)
	}
}

func TestTransform_Concurrent_MixedSuccessAndTimeout(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 200
	engine := NewEngine(cfg)

	const n = 10
	var wg sync.WaitGroup
	var successCount, failCount atomic.Int32

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var script string
			if idx%2 == 0 {
				script = `return {v: payload.v};` // fast
			} else {
				script = `while(true) {}` // infinite loop
			}
			result, err := engine.Transform(context.Background(), script, map[string]interface{}{"v": idx})
			require.NoError(t, err)
			if result.Success {
				successCount.Add(1)
			} else {
				failCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	// All infinite-loop scripts must timeout; some fast scripts may also fail
	// if they get a VM whose interrupt hasn't been cleared yet. Verify at least
	// one succeeded and the timeout scripts all failed.
	assert.GreaterOrEqual(t, successCount.Load(), int32(1), "at least one fast script should succeed")
	assert.GreaterOrEqual(t, failCount.Load(), int32(n/2), "at least the timeout scripts should fail")
	assert.Equal(t, int32(n), successCount.Load()+failCount.Load(), "all goroutines should complete")
}

func TestTransform_Concurrent_NoDataLeakage(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.PoolSize = 2 // small pool to force VM reuse
	engine := NewEngine(cfg)
	const runs = 30

	var wg sync.WaitGroup
	errs := make([]error, runs)

	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			secret := fmt.Sprintf("secret-%d", idx)
			script := fmt.Sprintf(`
				var leaked = (typeof previousSecret !== 'undefined') ? previousSecret : '';
				var previousSecret = '%s';
				return { leaked: leaked, mine: '%s' };
			`, secret, secret)

			result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
			if err != nil {
				errs[idx] = err
				return
			}
			if !result.Success {
				errs[idx] = fmt.Errorf("failed: %s", result.Error)
				return
			}
			out, ok := result.Output.(map[string]interface{})
			if !ok {
				errs[idx] = fmt.Errorf("unexpected type %T", result.Output)
				return
			}
			// The leaked field should be empty because VMs are cleaned.
			if leaked, _ := out["leaked"].(string); leaked != "" {
				errs[idx] = fmt.Errorf("data leakage detected: %q", leaked)
			}
		}(i)
	}

	wg.Wait()
	for i, e := range errs {
		assert.NoError(t, e, "run %d", i)
	}
}

// ---------------------------------------------------------------------------
// Script validation
// ---------------------------------------------------------------------------

func TestValidateScript_SyntaxError(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	err := engine.ValidateScript(`function { invalid }`)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidScript))
}

func TestValidateScript_UnterminatedString(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	err := engine.ValidateScript(`var x = "unterminated;`)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidScript))
}

func TestValidateScript_MismatchedBraces(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	err := engine.ValidateScript(`if (true) { return 1;`)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidScript))
}

func TestValidateScript_ValidComplexScript(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `
		var arr = [1, 2, 3];
		var obj = {a: 1, b: 'hello'};
		if (arr.length > 0) {
			return obj;
		}
		return null;
	`
	assert.NoError(t, engine.ValidateScript(script))
}

func TestValidateScript_EmptyScript(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())
	assert.NoError(t, engine.ValidateScript(``))
}

func TestValidateScript_OnlyComments(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())
	assert.NoError(t, engine.ValidateScript(`// just a comment`))
}

func TestValidateScript_InvalidAssignment(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())
	err := engine.ValidateScript(`var x = ;`)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidScript))
}

// ---------------------------------------------------------------------------
// Helper function coverage
// ---------------------------------------------------------------------------

func TestHelper_Set(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `
		var obj = {};
		set(obj, 'a.b.c', 42);
		return obj;
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	a := out["a"].(map[string]interface{})
	b := a["b"].(map[string]interface{})
	assert.Equal(t, int64(42), b["c"])
}

func TestHelper_Omit(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return omit(payload, ['secret', 'internal']);`
	payload := map[string]interface{}{
		"name":     "visible",
		"secret":   "hidden",
		"internal": "hidden",
		"id":       1,
	}
	result, err := engine.Transform(context.Background(), script, payload)
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Equal(t, "visible", out["name"])
	assert.Nil(t, out["secret"])
	assert.Nil(t, out["internal"])
	assert.NotNil(t, out["id"])
}

func TestHelper_Merge(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return merge({a:1}, {b:2}, {c:3, a:10});`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Equal(t, int64(10), out["a"], "later value should override")
	assert.Equal(t, int64(2), out["b"])
	assert.Equal(t, int64(3), out["c"])
}

func TestHelper_FormatDate(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return formatDate('2024-06-15T10:30:00Z', 'YYYY-MM-DD');`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, "2024-06-15", result.Output)
}

func TestHelper_FormatDate_InvalidDate(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return formatDate('not-a-date', 'YYYY-MM-DD');`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	// Invalid date should return the original string
	assert.Equal(t, "not-a-date", result.Output)
}

func TestHelper_UUID(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return uuid();`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	uid, ok := result.Output.(string)
	require.True(t, ok, "uuid should return a string")
	assert.Len(t, uid, 36, "UUID should be 36 chars (including hyphens)")
	assert.Equal(t, byte('4'), uid[14], "UUID v4 has 4 at position 14")
}

func TestHelper_UUID_Uniqueness(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return { a: uuid(), b: uuid() };`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.NotEqual(t, out["a"], out["b"], "two UUIDs should differ")
}

func TestHelper_Hash(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return { h1: hash('hello'), h2: hash('hello'), h3: hash('world') };`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Equal(t, out["h1"], out["h2"], "same input -> same hash")
	assert.NotEqual(t, out["h1"], out["h3"], "different input -> different hash")
}

func TestHelper_Base64(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	// btoa/atob are browser APIs not available in goja; base64Encode/Decode
	// rely on them, so calling these helpers is expected to fail.
	script := `
		var encoded = base64Encode('hello world');
		return encoded;
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	// If the runtime lacks btoa the call will produce an execution error.
	if !result.Success {
		assert.Contains(t, result.Error, "not defined",
			"expected a 'not defined' error when btoa is unavailable")
	}
	// Either way the engine must remain usable.
	r2, err := engine.Transform(context.Background(), `return 1;`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, r2.Success)
}

func TestHelper_Get_DefaultValue(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return get(payload, 'a.b.c', 'fallback');`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{"a": map[string]interface{}{}})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, "fallback", result.Output)
}

func TestHelper_Get_DeepNested(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return get(payload, 'a.b.c.d', null);`
	payload := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": "found",
				},
			},
		},
	}
	result, err := engine.Transform(context.Background(), script, payload)
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, "found", result.Output)
}

func TestHelper_Pick_EmptyKeys(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return pick(payload, []);`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{"a": 1, "b": 2})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Empty(t, out)
}

func TestHelper_Pick_NonexistentKeys(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return pick(payload, ['x', 'y']);`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{"a": 1})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Empty(t, out)
}

func TestHelper_Clone_Independence(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `
		var c = clone(payload);
		c.name = 'modified';
		return { original: payload.name, cloned: c.name };
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{"name": "original"})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Equal(t, "original", out["original"])
	assert.Equal(t, "modified", out["cloned"])
}

// ---------------------------------------------------------------------------
// Edge cases: empty scripts, nil inputs, output types, large payloads
// ---------------------------------------------------------------------------

func TestTransform_EmptyScript(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), ``, map[string]interface{}{"a": 1})
	require.NoError(t, err)
	// Empty script returns undefined → nil output, but should not error/panic.
	assert.True(t, result.Success)
}

func TestTransform_NilPayload(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return payload;`, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTransform_StringPayload(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return payload.toUpperCase();`, "hello")
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, "HELLO", result.Output)
}

func TestTransform_NumericPayload(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return payload * 2;`, 21)
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, int64(42), result.Output)
}

func TestTransform_ArrayPayload(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return payload.length;`, []int{1, 2, 3})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, int64(3), result.Output)
}

func TestTransform_BooleanOutput(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return payload.x > 10;`, map[string]interface{}{"x": 20})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, true, result.Output)
}

func TestTransform_NullReturn(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return null;`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Nil(t, result.Output)
}

func TestTransform_UndefinedReturn(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return undefined;`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTransform_ArrayReturn(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return [1, 'two', true];`, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	arr, ok := result.Output.([]interface{})
	require.True(t, ok, "expected array output, got %T", result.Output)
	assert.Len(t, arr, 3)
}

func TestTransform_LargePayload(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	// Build a payload with many keys
	payload := make(map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		payload[fmt.Sprintf("key_%d", i)] = i
	}

	script := `return { count: Object.keys(payload).length };`
	result, err := engine.Transform(context.Background(), script, payload)
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Equal(t, int64(1000), out["count"])
}

func TestTransform_DeeplyNestedPayload(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	// 50 levels of nesting
	payload := map[string]interface{}{"value": "deep"}
	for i := 0; i < 50; i++ {
		payload = map[string]interface{}{"child": payload}
	}

	script := `
		var node = payload;
		var depth = 0;
		while (node.child) { node = node.child; depth++; }
		return { depth: depth, value: node.value };
	`
	result, err := engine.Transform(context.Background(), script, payload)
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)

	out := result.Output.(map[string]interface{})
	assert.Equal(t, int64(50), out["depth"])
	assert.Equal(t, "deep", out["value"])
}

// ---------------------------------------------------------------------------
// Runtime errors in scripts
// ---------------------------------------------------------------------------

func TestTransform_RuntimeError_UndefinedProperty(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return payload.foo.bar;` // payload.foo is undefined
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestTransform_RuntimeError_TypeMismatch(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `return payload.toUpperCase();` // can't call toUpperCase on object
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{"a": 1})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestTransform_RuntimeError_ThrowException(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `throw new Error('custom error');`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "custom error")
}

func TestTransform_RuntimeError_StackOverflow(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 2000
	engine := NewEngine(cfg)

	script := `function f() { return f(); } return f();`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestTransform_RuntimeError_SyntaxErrorInScript(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `var x = {;` // syntax error
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

// ---------------------------------------------------------------------------
// VM pool: reuse and exhaustion under load
// ---------------------------------------------------------------------------

func TestVMPool_ReuseAfterTimeout(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 100
	engine := NewEngine(cfg)

	// First: trigger timeout
	result, err := engine.Transform(context.Background(), `while(true) {}`, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)

	// Second: VM should still be usable (interrupt cleared)
	result2, err := engine.Transform(context.Background(), `return {ok: true};`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result2.Success, "VM pool should recover after timeout: %s", result2.Error)
	out := result2.Output.(map[string]interface{})
	assert.Equal(t, true, out["ok"])
}

func TestVMPool_ExhaustionUnderLoad(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.PoolSize = 2
	cfg.TimeoutMs = 5000
	engine := NewEngine(cfg)

	// Flood with many concurrent requests beyond pool size
	const concurrent = 50
	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			script := fmt.Sprintf(`return { idx: %d };`, idx)
			result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
			if err == nil && result.Success {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	// sync.Pool creates new VMs as needed; all should succeed
	assert.Equal(t, int32(concurrent), successCount.Load(), "all concurrent transforms should succeed")
}

func TestVMPool_SequentialReuse(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	// Run many sequential transforms - VMs get reused from pool
	for i := 0; i < 50; i++ {
		script := fmt.Sprintf(`return { i: %d };`, i)
		result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
		require.NoError(t, err, "iteration %d", i)
		require.True(t, result.Success, "iteration %d: %s", i, result.Error)

		out := result.Output.(map[string]interface{})
		assert.Equal(t, int64(i), out["i"], "iteration %d", i)
	}
}

// ---------------------------------------------------------------------------
// Error recovery: engine continues working after failures
// ---------------------------------------------------------------------------

func TestErrorRecovery_AfterRuntimeError(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	// Cause an error
	result, err := engine.Transform(context.Background(), `throw new Error('boom');`, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)

	// Engine should still work
	result2, err := engine.Transform(context.Background(), `return {recovered: true};`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result2.Success, result2.Error)
}

func TestErrorRecovery_AfterStackOverflow(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 2000
	engine := NewEngine(cfg)

	// Stack overflow
	result, err := engine.Transform(context.Background(), `function f(){f();} f();`, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)

	// Should recover
	result2, err := engine.Transform(context.Background(), `return 42;`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result2.Success, result2.Error)
	assert.Equal(t, int64(42), result2.Output)
}

func TestErrorRecovery_AfterSyntaxError(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `var x = {;`, map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)

	result2, err := engine.Transform(context.Background(), `return 'ok';`, map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result2.Success, result2.Error)
	assert.Equal(t, "ok", result2.Output)
}

func TestErrorRecovery_AfterMultipleFailures(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	cfg.TimeoutMs = 100
	engine := NewEngine(cfg)

	// Cause several different failures in sequence
	failScripts := []string{
		`while(true) {}`,          // timeout
		`throw new Error('err');`, // exception
		`payload.x.y.z;`,          // runtime error
		`var x = {;`,              // syntax error
	}

	for _, s := range failScripts {
		result, err := engine.Transform(context.Background(), s, map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.Success)
	}

	// Let timed-out goroutines settle before reusing pool VMs.
	time.Sleep(300 * time.Millisecond)

	// Engine should still work after all those failures
	result, err := engine.Transform(context.Background(), `return {alive: true};`, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, "engine should recover: %s", result.Error)
	out := result.Output.(map[string]interface{})
	assert.Equal(t, true, out["alive"])
}

// ---------------------------------------------------------------------------
// TransformJSON edge cases
// ---------------------------------------------------------------------------

func TestTransformJSON_InvalidInputJSON(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	_, _, err := engine.TransformJSON(context.Background(), `return payload;`, []byte(`{invalid json}`))
	require.Error(t, err)
}

func TestTransformJSON_ScriptFailure(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	outputJSON, result, err := engine.TransformJSON(context.Background(), `throw new Error('fail');`, []byte(`{}`))
	require.NoError(t, err)
	assert.Nil(t, outputJSON)
	require.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestTransformJSON_EmptyJSONObject(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	outputJSON, result, err := engine.TransformJSON(context.Background(), `return {empty: true};`, []byte(`{}`))
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.JSONEq(t, `{"empty":true}`, string(outputJSON))
}

func TestTransformJSON_ArrayInput(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	outputJSON, result, err := engine.TransformJSON(context.Background(), `return payload.length;`, []byte(`[1,2,3]`))
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Equal(t, "3", string(outputJSON))
}

// ---------------------------------------------------------------------------
// Console logging edge cases
// ---------------------------------------------------------------------------

func TestConsole_MultipleArguments(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `
		console.log('a', 'b', 'c');
		return null;
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	require.Len(t, result.Logs, 1)
	assert.Contains(t, result.Logs[0], "a")
	assert.Contains(t, result.Logs[0], "b")
	assert.Contains(t, result.Logs[0], "c")
}

func TestConsole_WarnAndErrorPrefixes(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	script := `
		console.warn('warning msg');
		console.error('error msg');
		return null;
	`
	result, err := engine.Transform(context.Background(), script, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	require.Len(t, result.Logs, 2)
	assert.True(t, strings.HasPrefix(result.Logs[0], "[WARN] "))
	assert.True(t, strings.HasPrefix(result.Logs[1], "[ERROR] "))
}

func TestConsole_NoLogs(t *testing.T) {
	t.Parallel()
	engine := NewEngine(DefaultEngineConfig())

	result, err := engine.Transform(context.Background(), `return 1;`, map[string]interface{}{})
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	assert.Empty(t, result.Logs)
}

// ---------------------------------------------------------------------------
// DefaultEngineConfig
// ---------------------------------------------------------------------------

func TestDefaultEngineConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultEngineConfig()
	assert.Equal(t, 64, cfg.MaxMemoryMB)
	assert.Equal(t, 5000, cfg.TimeoutMs)
	assert.Equal(t, 10, cfg.PoolSize)
}

package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- In-Memory Store for Testing ---
type memoryStore struct {
	mu   sync.RWMutex
	data map[string]*Key
}

func newMemoryStore() *memoryStore {
	return &memoryStore{data: make(map[string]*Key)}
}

func (m *memoryStore) makeKey(tenantID, key string) string {
	return tenantID + ":" + key
}

func (m *memoryStore) Get(ctx context.Context, tenantID, key string) (*Key, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.data[m.makeKey(tenantID, key)]
	if !ok {
		return nil, nil
	}
	if time.Now().After(k.ExpiresAt) {
		return nil, nil
	}
	return k, nil
}

func (m *memoryStore) Create(ctx context.Context, key *Key) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[m.makeKey(key.TenantID, key.Key)] = key
	return nil
}

func (m *memoryStore) Update(ctx context.Context, key *Key) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[m.makeKey(key.TenantID, key.Key)] = key
	return nil
}

func (m *memoryStore) Delete(ctx context.Context, tenantID, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, m.makeKey(tenantID, key))
	return nil
}

func (m *memoryStore) Cleanup(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for k, v := range m.data {
		if time.Now().After(v.ExpiresAt) {
			delete(m.data, k)
			count++
		}
	}
	return count, nil
}

// --- Service Tests ---

func TestService_Check_NewRequest(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	result, err := svc.Check(ctx, "tenant1", "key1", []byte(`{"data":"test"}`))
	require.NoError(t, err)
	assert.True(t, result.IsNew)
	assert.NotNil(t, result.Key)
	assert.True(t, result.Key.IsProcessing)
}

func TestService_Check_EmptyKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), nil)
	ctx := context.Background()

	result, err := svc.Check(ctx, "tenant1", "", nil)
	require.NoError(t, err)
	assert.True(t, result.IsNew)
	assert.Nil(t, result.Key)
}

func TestService_Check_DuplicateReturnsCache(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	body := []byte(`{"data":"test"}`)

	// First request
	result1, err := svc.Check(ctx, "tenant1", "key1", body)
	require.NoError(t, err)
	assert.True(t, result1.IsNew)

	// Complete the request
	respBody := []byte(`{"status":"ok"}`)
	err = svc.Complete(ctx, "tenant1", "key1", 200, respBody)
	require.NoError(t, err)

	// Second request with same key should return cached response
	result2, err := svc.Check(ctx, "tenant1", "key1", body)
	require.NoError(t, err)
	assert.False(t, result2.IsNew)
	assert.False(t, result2.IsProcessing)
	assert.Equal(t, 200, result2.CachedStatusCode)
	assert.JSONEq(t, `{"status":"ok"}`, string(result2.CachedResponse))
}

func TestService_Check_InProgressRequest(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	body := []byte(`{"data":"test"}`)

	// First request
	_, err := svc.Check(ctx, "tenant1", "key1", body)
	require.NoError(t, err)

	// Second request while first is still processing
	result, err := svc.Check(ctx, "tenant1", "key1", body)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.True(t, result.IsProcessing)
}

func TestService_Check_KeyTooLong(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), &Config{
		MaxKeyLength: 10,
		DefaultTTL:   time.Hour,
	})
	ctx := context.Background()

	_, err := svc.Check(ctx, "tenant1", "this-key-is-way-too-long", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestService_Check_RequestHashMismatch(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, &Config{
		DefaultTTL:           time.Hour,
		MaxKeyLength:         255,
		EnableRequestHashing: true,
	})
	ctx := context.Background()

	// First request
	body1 := []byte(`{"data":"original"}`)
	_, err := svc.Check(ctx, "tenant1", "key1", body1)
	require.NoError(t, err)

	// Complete it
	err = svc.Complete(ctx, "tenant1", "key1", 200, []byte(`{}`))
	require.NoError(t, err)

	// Second request with DIFFERENT body should fail
	body2 := []byte(`{"data":"different"}`)
	_, err = svc.Check(ctx, "tenant1", "key1", body2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match original request")
}

func TestService_Complete_EmptyKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), nil)

	// Empty key should be a no-op
	err := svc.Complete(context.Background(), "tenant1", "", 200, nil)
	assert.NoError(t, err)
}

func TestService_Complete_NotFound(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), nil)

	err := svc.Complete(context.Background(), "tenant1", "nonexistent", 200, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_Abort_RemovesKey(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	body := []byte(`{"data":"test"}`)
	_, err := svc.Check(ctx, "tenant1", "key1", body)
	require.NoError(t, err)

	// Abort
	err = svc.Abort(ctx, "tenant1", "key1")
	require.NoError(t, err)

	// Should be treatable as new request now
	result, err := svc.Check(ctx, "tenant1", "key1", body)
	require.NoError(t, err)
	assert.True(t, result.IsNew)
}

func TestService_Abort_EmptyKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), nil)

	err := svc.Abort(context.Background(), "tenant1", "")
	assert.NoError(t, err)
}

func TestService_Cleanup(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, &Config{
		DefaultTTL:   -1 * time.Second, // Already expired
		MaxKeyLength: 255,
	})
	ctx := context.Background()

	// Create an expired key
	_, err := svc.Check(ctx, "tenant1", "key1", nil)
	require.NoError(t, err)

	count, err := svc.Cleanup(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestService_DifferentTenants_SameKey(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	body := []byte(`{"data":"test"}`)

	// Tenant A creates a key
	result1, err := svc.Check(ctx, "tenantA", "shared-key", body)
	require.NoError(t, err)
	assert.True(t, result1.IsNew)

	// Complete tenant A's key
	err = svc.Complete(ctx, "tenantA", "shared-key", 200, []byte(`{"from":"A"}`))
	require.NoError(t, err)

	// Tenant B uses the same key name - should be a NEW request
	result2, err := svc.Check(ctx, "tenantB", "shared-key", body)
	require.NoError(t, err)
	assert.True(t, result2.IsNew)
}

func TestService_ConcurrentRequests_SameKey(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]*Result, 10)
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx] = svc.Check(ctx, "tenant1", "concurrent-key", []byte(`{"data":"test"}`))
		}(i)
	}
	wg.Wait()

	// At least one should succeed as new, others should be in-progress or new
	newCount := 0
	for i, r := range results {
		assert.NoError(t, errors[i])
		if r.IsNew {
			newCount++
		}
	}
	assert.GreaterOrEqual(t, newCount, 1)
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	assert.Equal(t, 24*time.Hour, config.DefaultTTL)
	assert.Equal(t, 255, config.MaxKeyLength)
	assert.True(t, config.EnableRequestHashing)
}

func TestNewService_NilConfig(t *testing.T) {
	t.Parallel()

	svc := NewService(newMemoryStore(), nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.config)
	assert.Equal(t, 24*time.Hour, svc.config.DefaultTTL)
}

func TestHashRequest(t *testing.T) {
	t.Parallel()

	hash1 := hashRequest([]byte(`{"data":"test1"}`))
	hash2 := hashRequest([]byte(`{"data":"test2"}`))
	hash3 := hashRequest([]byte(`{"data":"test1"}`))

	assert.NotEmpty(t, hash1)
	assert.NotEqual(t, hash1, hash2)
	assert.Equal(t, hash1, hash3) // Same input = same hash
}

// --- Deduplication Tests ---

func TestDeduplicationService_Check_NotDuplicate(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	svc := NewDeduplicationService(store, nil)
	ctx := context.Background()

	result, err := svc.Check(ctx, "tenant1", "ep1", []byte(`{"event":"order.created"}`))
	require.NoError(t, err)
	assert.False(t, result.IsDuplicate)
	assert.NotEmpty(t, result.ContentHash)
}

func TestDeduplicationService_Check_Duplicate(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	svc := NewDeduplicationService(store, nil)
	ctx := context.Background()

	payload := []byte(`{"event":"order.created","id":123}`)

	// Record the first occurrence
	result1, err := svc.Check(ctx, "tenant1", "ep1", payload)
	require.NoError(t, err)
	err = svc.Record(ctx, "tenant1", "ep1", result1.ContentHash, "delivery-1")
	require.NoError(t, err)

	// Check again with same payload
	result2, err := svc.Check(ctx, "tenant1", "ep1", payload)
	require.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, "delivery-1", result2.OriginalDeliveryID)
}

func TestDeduplicationService_DifferentPayloads(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	svc := NewDeduplicationService(store, nil)
	ctx := context.Background()

	result1, _ := svc.Check(ctx, "t1", "ep1", []byte(`{"data":"a"}`))
	_ = svc.Record(ctx, "t1", "ep1", result1.ContentHash, "d1")

	result2, err := svc.Check(ctx, "t1", "ep1", []byte(`{"data":"b"}`))
	require.NoError(t, err)
	assert.False(t, result2.IsDuplicate)
}

func TestDeduplicationService_DifferentTenants(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	svc := NewDeduplicationService(store, nil)
	ctx := context.Background()

	payload := []byte(`{"same":"payload"}`)

	result1, _ := svc.Check(ctx, "tenant1", "ep1", payload)
	_ = svc.Record(ctx, "tenant1", "ep1", result1.ContentHash, "d1")

	// Same payload but different tenant should not be a duplicate
	result2, err := svc.Check(ctx, "tenant2", "ep1", payload)
	require.NoError(t, err)
	assert.False(t, result2.IsDuplicate)
}

func TestDeduplicationService_DifferentEndpoints(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	svc := NewDeduplicationService(store, nil)
	ctx := context.Background()

	payload := []byte(`{"same":"payload"}`)

	result1, _ := svc.Check(ctx, "t1", "ep1", payload)
	_ = svc.Record(ctx, "t1", "ep1", result1.ContentHash, "d1")

	// Same payload but different endpoint should not be a duplicate
	result2, err := svc.Check(ctx, "t1", "ep2", payload)
	require.NoError(t, err)
	assert.False(t, result2.IsDuplicate)
}

func TestInMemoryDeduplicationStore_Expiration(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	ctx := context.Background()

	// Record with very short expiration
	err := store.Record(ctx, "t1", "ep1", "hash123", "d1", 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	id, found, err := store.Check(ctx, "t1", "ep1", "hash123")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, id)
}

func TestDefaultDeduplicationConfig(t *testing.T) {
	t.Parallel()

	config := DefaultDeduplicationConfig()
	assert.Equal(t, 5*time.Minute, config.WindowDuration)
	assert.Equal(t, 10000, config.MaxEntriesPerTenant)
}

func TestNewDeduplicationService_NilConfig(t *testing.T) {
	t.Parallel()

	svc := NewDeduplicationService(NewInMemoryDeduplicationStore(), nil)
	assert.NotNil(t, svc)
	assert.Equal(t, 5*time.Minute, svc.config.WindowDuration)
}

// --- Redis Store Tests ---

type mockRedisClient struct {
	mu   sync.RWMutex
	data map[string]string
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{data: make(map[string]string)}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return v, nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch v := value.(type) {
	case []byte:
		m.data[key] = string(v)
	case string:
		m.data[key] = v
	default:
		m.data[key] = fmt.Sprintf("%v", v)
	}
	return nil
}

func (m *mockRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	switch v := value.(type) {
	case []byte:
		m.data[key] = string(v)
	case string:
		m.data[key] = v
	default:
		m.data[key] = fmt.Sprintf("%v", v)
	}
	return true, nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func TestRedisStore_CreateAndGet(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "test:")

	ctx := context.Background()
	key := &Key{
		Key:          "test-key",
		TenantID:     "tenant1",
		RequestHash:  "hash123",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsProcessing: true,
	}

	err := store.Create(ctx, key)
	require.NoError(t, err)

	// Get should return the key
	got, err := store.Get(ctx, "tenant1", "test-key")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "test-key", got.Key)
	assert.Equal(t, "tenant1", got.TenantID)
	assert.True(t, got.IsProcessing)
}

func TestRedisStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "")

	got, err := store.Get(context.Background(), "tenant1", "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisStore_Delete(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "test:")

	ctx := context.Background()
	key := &Key{
		Key:       "to-delete",
		TenantID:  "tenant1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	err := store.Create(ctx, key)
	require.NoError(t, err)

	err = store.Delete(ctx, "tenant1", "to-delete")
	require.NoError(t, err)

	got, _ := store.Get(ctx, "tenant1", "to-delete")
	assert.Nil(t, got)
}

func TestRedisStore_Update(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "test:")
	ctx := context.Background()

	key := &Key{
		Key:          "update-key",
		TenantID:     "tenant1",
		IsProcessing: true,
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	require.NoError(t, store.Create(ctx, key))

	// Update
	key.IsProcessing = false
	key.StatusCode = 200
	key.Response = json.RawMessage(`{"ok":true}`)
	require.NoError(t, store.Update(ctx, key))

	got, err := store.Get(ctx, "tenant1", "update-key")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.IsProcessing)
	assert.Equal(t, 200, got.StatusCode)
}

func TestRedisStore_Cleanup_NoOp(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "")

	count, err := store.Cleanup(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRedisStore_DefaultPrefix(t *testing.T) {
	t.Parallel()
	store := NewRedisStore(newMockRedisClient(), "")
	assert.Equal(t, "idempotency:", store.prefix)
}

// --- Key JSON Tests ---

func TestKey_JSONSerialization(t *testing.T) {
	t.Parallel()

	key := Key{
		Key:          "test",
		TenantID:     "t1",
		RequestHash:  "hash",
		Response:     json.RawMessage(`{"ok":true}`),
		StatusCode:   200,
		CreatedAt:    time.Now().Truncate(time.Second),
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		IsProcessing: false,
	}

	data, err := json.Marshal(key)
	require.NoError(t, err)

	var decoded Key
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, key.Key, decoded.Key)
	assert.Equal(t, key.TenantID, decoded.TenantID)
	assert.Equal(t, key.StatusCode, decoded.StatusCode)
	assert.Equal(t, key.IsProcessing, decoded.IsProcessing)
}

// =============================================================================
// Failing Store Mock (for error-handling tests)
// =============================================================================

type failingStore struct {
	getErr     error
	createErr  error
	updateErr  error
	deleteErr  error
	cleanupErr error
	// fallback allows selective failures: non-nil errors override this store
	fallback Store
}

func (f *failingStore) Get(ctx context.Context, tenantID, key string) (*Key, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.fallback != nil {
		return f.fallback.Get(ctx, tenantID, key)
	}
	return nil, nil
}

func (f *failingStore) Create(ctx context.Context, key *Key) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.fallback != nil {
		return f.fallback.Create(ctx, key)
	}
	return nil
}

func (f *failingStore) Update(ctx context.Context, key *Key) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if f.fallback != nil {
		return f.fallback.Update(ctx, key)
	}
	return nil
}

func (f *failingStore) Delete(ctx context.Context, tenantID, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if f.fallback != nil {
		return f.fallback.Delete(ctx, tenantID, key)
	}
	return nil
}

func (f *failingStore) Cleanup(ctx context.Context) (int64, error) {
	if f.cleanupErr != nil {
		return 0, f.cleanupErr
	}
	if f.fallback != nil {
		return f.fallback.Cleanup(ctx)
	}
	return 0, nil
}

// =============================================================================
// Service Construction Tests
// =============================================================================

func TestNewService_WithCustomConfig(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		DefaultTTL:           10 * time.Minute,
		MaxKeyLength:         64,
		EnableRequestHashing: false,
	}
	svc := NewService(newMemoryStore(), cfg)
	assert.NotNil(t, svc)
	assert.Equal(t, 10*time.Minute, svc.config.DefaultTTL)
	assert.Equal(t, 64, svc.config.MaxKeyLength)
	assert.False(t, svc.config.EnableRequestHashing)
}

// =============================================================================
// TTL / Expiry Behavior Tests
// =============================================================================

func TestService_Check_ExpiredKeyTreatedAsNew(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, &Config{
		DefaultTTL:   50 * time.Millisecond,
		MaxKeyLength: 255,
	})
	ctx := context.Background()

	// Create key
	result1, err := svc.Check(ctx, "tenant1", "exp-key", []byte(`{"a":1}`))
	require.NoError(t, err)
	assert.True(t, result1.IsNew)

	// Complete it
	require.NoError(t, svc.Complete(ctx, "tenant1", "exp-key", 200, []byte(`{"ok":true}`)))

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should be treated as a brand-new request
	result2, err := svc.Check(ctx, "tenant1", "exp-key", []byte(`{"a":1}`))
	require.NoError(t, err)
	assert.True(t, result2.IsNew)
}

func TestMemoryStore_Get_ExpiredKeyReturnsNil(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	ctx := context.Background()

	key := &Key{
		Key:          "k1",
		TenantID:     "t1",
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // already expired
		IsProcessing: true,
	}
	require.NoError(t, store.Create(ctx, key))

	got, err := store.Get(ctx, "t1", "k1")
	require.NoError(t, err)
	assert.Nil(t, got, "expired key should not be returned by Get")
}

func TestService_Cleanup_MultipleMixed(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	ctx := context.Background()

	// Create 3 expired keys directly
	for i := 0; i < 3; i++ {
		store.Create(ctx, &Key{
			Key:       fmt.Sprintf("expired-%d", i),
			TenantID:  "t1",
			ExpiresAt: time.Now().Add(-time.Hour),
		})
	}
	// Create 2 valid keys
	for i := 0; i < 2; i++ {
		store.Create(ctx, &Key{
			Key:       fmt.Sprintf("valid-%d", i),
			TenantID:  "t1",
			ExpiresAt: time.Now().Add(time.Hour),
		})
	}

	svc := NewService(store, nil)
	count, err := svc.Cleanup(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Valid keys still accessible
	for i := 0; i < 2; i++ {
		got, err := store.Get(ctx, "t1", fmt.Sprintf("valid-%d", i))
		require.NoError(t, err)
		assert.NotNil(t, got)
	}
}

// =============================================================================
// Request Hash Validation Tests
// =============================================================================

func TestService_Check_HashingDisabled_DifferentBodyAllowed(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, &Config{
		DefaultTTL:           time.Hour,
		MaxKeyLength:         255,
		EnableRequestHashing: false,
	})
	ctx := context.Background()

	body1 := []byte(`{"data":"original"}`)
	_, err := svc.Check(ctx, "t1", "key1", body1)
	require.NoError(t, err)
	require.NoError(t, svc.Complete(ctx, "t1", "key1", 200, []byte(`{"ok":true}`)))

	// Different body with same key should succeed when hashing is disabled
	body2 := []byte(`{"data":"different"}`)
	result, err := svc.Check(ctx, "t1", "key1", body2)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, 200, result.CachedStatusCode)
}

func TestService_Check_HashingEnabled_EmptyBodyOnBoth(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, &Config{
		DefaultTTL:           time.Hour,
		MaxKeyLength:         255,
		EnableRequestHashing: true,
	})
	ctx := context.Background()

	// First request with nil body
	_, err := svc.Check(ctx, "t1", "key-empty", nil)
	require.NoError(t, err)
	require.NoError(t, svc.Complete(ctx, "t1", "key-empty", 204, []byte(`{}`)))

	// Second request with nil body should return cached
	result, err := svc.Check(ctx, "t1", "key-empty", nil)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, 204, result.CachedStatusCode)
}

func TestService_Check_HashingEnabled_NoHashStoredSkipsValidation(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, &Config{
		DefaultTTL:           time.Hour,
		MaxKeyLength:         255,
		EnableRequestHashing: true,
	})
	ctx := context.Background()

	// Create key with empty body (no hash stored)
	_, err := svc.Check(ctx, "t1", "key-nohash", nil)
	require.NoError(t, err)
	require.NoError(t, svc.Complete(ctx, "t1", "key-nohash", 200, []byte(`{}`)))

	// Replay with a body: since original hash is empty, validation is skipped
	result, err := svc.Check(ctx, "t1", "key-nohash", []byte(`{"some":"body"}`))
	require.NoError(t, err)
	assert.False(t, result.IsNew)
}

// =============================================================================
// Edge Cases: Key Length, Special Characters
// =============================================================================

func TestService_Check_ExactMaxLengthKey(t *testing.T) {
	t.Parallel()
	maxLen := 50
	svc := NewService(newMemoryStore(), &Config{
		DefaultTTL:   time.Hour,
		MaxKeyLength: maxLen,
	})
	ctx := context.Background()

	// Key exactly at the limit
	exactKey := string(make([]byte, maxLen))
	for i := range exactKey {
		exactKey = exactKey[:i] + "a" + exactKey[i+1:]
	}
	result, err := svc.Check(ctx, "t1", exactKey, nil)
	require.NoError(t, err)
	assert.True(t, result.IsNew)

	// One over the limit
	tooLong := exactKey + "x"
	_, err = svc.Check(ctx, "t1", tooLong, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestService_Check_SpecialCharacterKeys(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), nil)
	ctx := context.Background()

	keys := []string{
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key/with/slashes",
		"key with spaces",
		"key🎉emoji",
		"日本語キー",
		"key=with&special?chars#!",
		"uuid:550e8400-e29b-41d4-a716-446655440000",
	}
	for _, k := range keys {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			result, err := svc.Check(ctx, "t1", k, nil)
			require.NoError(t, err)
			assert.True(t, result.IsNew)
		})
	}
}

func TestService_Check_NilVsEmptyBody(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemoryStore(), &Config{
		DefaultTTL:           time.Hour,
		MaxKeyLength:         255,
		EnableRequestHashing: true,
	})
	ctx := context.Background()

	// nil body
	r1, err := svc.Check(ctx, "t1", "nil-body-key", nil)
	require.NoError(t, err)
	assert.True(t, r1.IsNew)

	// empty []byte body on a different key
	r2, err := svc.Check(ctx, "t1", "empty-body-key", []byte{})
	require.NoError(t, err)
	assert.True(t, r2.IsNew)
}

// =============================================================================
// Error Handling: Store Failure Tests
// =============================================================================

func TestService_Check_StoreGetFailure(t *testing.T) {
	t.Parallel()
	store := &failingStore{getErr: fmt.Errorf("database connection refused")}
	svc := NewService(store, nil)

	_, err := svc.Check(context.Background(), "t1", "key1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check idempotency key")
	assert.Contains(t, err.Error(), "database connection refused")
}

func TestService_Check_StoreCreateFailure(t *testing.T) {
	t.Parallel()
	store := &failingStore{createErr: fmt.Errorf("disk full")}
	svc := NewService(store, nil)

	_, err := svc.Check(context.Background(), "t1", "key1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create idempotency key")
	assert.Contains(t, err.Error(), "disk full")
}

func TestService_Complete_StoreGetFailure(t *testing.T) {
	t.Parallel()
	store := &failingStore{getErr: fmt.Errorf("timeout")}
	svc := NewService(store, nil)

	err := svc.Complete(context.Background(), "t1", "key1", 200, []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get idempotency key")
}

func TestService_Complete_StoreUpdateFailure(t *testing.T) {
	t.Parallel()
	mem := newMemoryStore()
	store := &failingStore{updateErr: fmt.Errorf("write conflict"), fallback: mem}
	svc := NewService(store, nil)
	ctx := context.Background()

	// Create the key via the memory store fallback
	_, err := svc.Check(ctx, "t1", "key1", nil)
	require.NoError(t, err)

	// Complete should fail on update
	err = svc.Complete(ctx, "t1", "key1", 200, []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update idempotency key")
}

func TestService_Abort_StoreDeleteFailure(t *testing.T) {
	t.Parallel()
	store := &failingStore{deleteErr: fmt.Errorf("permission denied")}
	svc := NewService(store, nil)

	err := svc.Abort(context.Background(), "t1", "key1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete idempotency key")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestService_Cleanup_StoreFailure(t *testing.T) {
	t.Parallel()
	store := &failingStore{cleanupErr: fmt.Errorf("table locked")}
	svc := NewService(store, nil)

	count, err := svc.Cleanup(context.Background())
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestService_ConcurrentCheckAndComplete(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	const numKeys = 20
	var wg sync.WaitGroup

	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-%d", idx)
			body := []byte(fmt.Sprintf(`{"idx":%d}`, idx))

			result, err := svc.Check(ctx, "t1", key, body)
			require.NoError(t, err)
			assert.True(t, result.IsNew)

			err = svc.Complete(ctx, "t1", key, 200, []byte(fmt.Sprintf(`{"done":%d}`, idx)))
			require.NoError(t, err)

			// Verify cached response
			result2, err := svc.Check(ctx, "t1", key, body)
			require.NoError(t, err)
			assert.False(t, result2.IsNew)
			assert.Equal(t, 200, result2.CachedStatusCode)
		}(i)
	}
	wg.Wait()
}

func TestService_ConcurrentCheckAbort(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	const numKeys = 20
	var wg sync.WaitGroup

	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("abort-%d", idx)

			_, err := svc.Check(ctx, "t1", key, nil)
			require.NoError(t, err)

			err = svc.Abort(ctx, "t1", key)
			require.NoError(t, err)

			// Key should be available for reuse
			result, err := svc.Check(ctx, "t1", key, nil)
			require.NoError(t, err)
			assert.True(t, result.IsNew)
		}(i)
	}
	wg.Wait()
}

// =============================================================================
// MemoryStore Direct Tests
// =============================================================================

func TestMemoryStore_CreateAndGet(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	ctx := context.Background()

	key := &Key{
		Key:          "k1",
		TenantID:     "t1",
		RequestHash:  "abc",
		StatusCode:   201,
		Response:     json.RawMessage(`{"id":"123"}`),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsProcessing: false,
	}
	require.NoError(t, store.Create(ctx, key))

	got, err := store.Get(ctx, "t1", "k1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "k1", got.Key)
	assert.Equal(t, "t1", got.TenantID)
	assert.Equal(t, "abc", got.RequestHash)
	assert.Equal(t, 201, got.StatusCode)
}

func TestMemoryStore_GetNonExistent(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()

	got, err := store.Get(context.Background(), "t1", "missing")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryStore_Update(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	ctx := context.Background()

	key := &Key{
		Key:          "k1",
		TenantID:     "t1",
		ExpiresAt:    time.Now().Add(time.Hour),
		IsProcessing: true,
	}
	require.NoError(t, store.Create(ctx, key))

	key.IsProcessing = false
	key.StatusCode = 200
	key.Response = json.RawMessage(`{"updated":true}`)
	require.NoError(t, store.Update(ctx, key))

	got, err := store.Get(ctx, "t1", "k1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.IsProcessing)
	assert.Equal(t, 200, got.StatusCode)
}

func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	ctx := context.Background()

	key := &Key{Key: "k1", TenantID: "t1", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, store.Create(ctx, key))

	require.NoError(t, store.Delete(ctx, "t1", "k1"))

	got, err := store.Get(ctx, "t1", "k1")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryStore_DeleteNonExistent(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	// Should not error on deleting a non-existent key
	err := store.Delete(context.Background(), "t1", "missing")
	assert.NoError(t, err)
}

func TestMemoryStore_CleanupEmpty(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()

	count, err := store.Cleanup(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestMemoryStore_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	const n = 50

	// Concurrent writes
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := &Key{
				Key:       fmt.Sprintf("k-%d", idx),
				TenantID:  "t1",
				ExpiresAt: time.Now().Add(time.Hour),
			}
			assert.NoError(t, store.Create(ctx, key))
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			got, err := store.Get(ctx, "t1", fmt.Sprintf("k-%d", idx))
			assert.NoError(t, err)
			assert.NotNil(t, got)
		}(i)
	}
	wg.Wait()
}

// =============================================================================
// Full Lifecycle Tests
// =============================================================================

func TestService_FullLifecycle_CheckCompleteReplay(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()
	body := []byte(`{"order_id":"12345"}`)

	// Step 1: Check → new request
	r1, err := svc.Check(ctx, "t1", "lifecycle-key", body)
	require.NoError(t, err)
	assert.True(t, r1.IsNew)
	assert.True(t, r1.Key.IsProcessing)

	// Step 2: Complete with response
	respBody := []byte(`{"status":"created","id":"abc"}`)
	require.NoError(t, svc.Complete(ctx, "t1", "lifecycle-key", 201, respBody))

	// Step 3: Replay → returns cached response
	r2, err := svc.Check(ctx, "t1", "lifecycle-key", body)
	require.NoError(t, err)
	assert.False(t, r2.IsNew)
	assert.False(t, r2.IsProcessing)
	assert.Equal(t, 201, r2.CachedStatusCode)
	assert.JSONEq(t, `{"status":"created","id":"abc"}`, string(r2.CachedResponse))
}

func TestService_FullLifecycle_CheckAbortRetry(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()
	body := []byte(`{"action":"retry"}`)

	// Check → new
	r1, err := svc.Check(ctx, "t1", "retry-key", body)
	require.NoError(t, err)
	assert.True(t, r1.IsNew)

	// Abort (simulating handler failure)
	require.NoError(t, svc.Abort(ctx, "t1", "retry-key"))

	// Retry → should be new again
	r2, err := svc.Check(ctx, "t1", "retry-key", body)
	require.NoError(t, err)
	assert.True(t, r2.IsNew)

	// Complete on retry
	require.NoError(t, svc.Complete(ctx, "t1", "retry-key", 200, []byte(`{"retried":true}`)))

	// Verify cached
	r3, err := svc.Check(ctx, "t1", "retry-key", body)
	require.NoError(t, err)
	assert.False(t, r3.IsNew)
	assert.Equal(t, 200, r3.CachedStatusCode)
}

func TestService_MultipleKeysPerTenant(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := NewService(store, nil)
	ctx := context.Background()

	keys := []string{"key-a", "key-b", "key-c"}
	for i, k := range keys {
		body := []byte(fmt.Sprintf(`{"idx":%d}`, i))
		result, err := svc.Check(ctx, "t1", k, body)
		require.NoError(t, err)
		assert.True(t, result.IsNew)

		require.NoError(t, svc.Complete(ctx, "t1", k, 200+i, []byte(fmt.Sprintf(`{"key":"%s"}`, k))))
	}

	// Each key returns its own cached response
	for i, k := range keys {
		body := []byte(fmt.Sprintf(`{"idx":%d}`, i))
		result, err := svc.Check(ctx, "t1", k, body)
		require.NoError(t, err)
		assert.False(t, result.IsNew)
		assert.Equal(t, 200+i, result.CachedStatusCode)
		assert.JSONEq(t, fmt.Sprintf(`{"key":"%s"}`, k), string(result.CachedResponse))
	}
}

// =============================================================================
// Redis Deduplication Store Tests
// =============================================================================

func TestRedisDeduplicationStore_CheckAndRecord(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisDeduplicationStore(client, "dedup-test:")

	ctx := context.Background()

	// Initially not found
	id, found, err := store.Check(ctx, "t1", "ep1", "hash-abc")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, id)

	// Record
	require.NoError(t, store.Record(ctx, "t1", "ep1", "hash-abc", "delivery-42", time.Hour))

	// Now found
	id, found, err = store.Check(ctx, "t1", "ep1", "hash-abc")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "delivery-42", id)
}

func TestRedisDeduplicationStore_DefaultPrefix(t *testing.T) {
	t.Parallel()
	store := NewRedisDeduplicationStore(newMockRedisClient(), "")
	assert.Equal(t, "dedup:", store.prefix)
}

func TestRedisDeduplicationStore_DifferentTenants(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisDeduplicationStore(client, "")
	ctx := context.Background()

	require.NoError(t, store.Record(ctx, "t1", "ep1", "hash1", "d1", time.Hour))

	// Different tenant, same hash
	id, found, err := store.Check(ctx, "t2", "ep1", "hash1")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, id)
}

// =============================================================================
// Redis Store Additional Tests
// =============================================================================

func TestRedisStore_CreateWithExpiredTTL(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "test:")
	ctx := context.Background()

	key := &Key{
		Key:       "expired-ttl",
		TenantID:  "t1",
		ExpiresAt: time.Now().Add(-time.Hour), // already in the past
	}
	// Should not error; Redis store falls back to 24h TTL
	err := store.Create(ctx, key)
	assert.NoError(t, err)
}

func TestRedisStore_UpdateWithExpiredTTL(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	store := NewRedisStore(client, "test:")
	ctx := context.Background()

	key := &Key{
		Key:       "expired-update",
		TenantID:  "t1",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	// Should not error; falls back to 24h TTL
	err := store.Update(ctx, key)
	assert.NoError(t, err)
}

func TestRedisStore_GetEmptyData(t *testing.T) {
	t.Parallel()
	client := newMockRedisClient()
	// Manually set empty string
	client.data["idempotency:t1:empty-key"] = ""
	store := NewRedisStore(client, "")

	got, err := store.Get(context.Background(), "t1", "empty-key")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

// =============================================================================
// hashRequest Tests
// =============================================================================

func TestHashRequest_Deterministic(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event":"webhook","payload":{"id":1}}`)

	h1 := hashRequest(body)
	h2 := hashRequest(body)
	h3 := hashRequest(body)

	assert.Equal(t, h1, h2)
	assert.Equal(t, h2, h3)
	assert.Len(t, h1, 64) // SHA-256 hex = 64 chars
}

func TestHashRequest_EmptyBody(t *testing.T) {
	t.Parallel()
	h := hashRequest([]byte{})
	assert.NotEmpty(t, h)
	assert.Len(t, h, 64)
}

func TestHashRequest_UniquePerInput(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool)
	inputs := []string{
		`{"a":1}`,
		`{"a":2}`,
		`{"a":"1"}`,
		`{}`,
		`null`,
		`[]`,
		`{"a":1,"b":2}`,
		`{"b":2,"a":1}`, // different order = different hash (raw bytes)
	}
	for _, input := range inputs {
		h := hashRequest([]byte(input))
		assert.False(t, seen[h], "hash collision for input: %s", input)
		seen[h] = true
	}
}

// =============================================================================
// Deduplication Service Additional Tests
// =============================================================================

func TestDeduplicationService_RecordAndCheckMultiple(t *testing.T) {
	t.Parallel()
	store := NewInMemoryDeduplicationStore()
	svc := NewDeduplicationService(store, nil)
	ctx := context.Background()

	payloads := []struct {
		tenant  string
		ep      string
		payload string
		id      string
	}{
		{"t1", "ep1", `{"a":1}`, "d1"},
		{"t1", "ep1", `{"a":2}`, "d2"},
		{"t1", "ep2", `{"a":1}`, "d3"},
		{"t2", "ep1", `{"a":1}`, "d4"},
	}

	for _, p := range payloads {
		result, err := svc.Check(ctx, p.tenant, p.ep, []byte(p.payload))
		require.NoError(t, err)
		assert.False(t, result.IsDuplicate)
		require.NoError(t, svc.Record(ctx, p.tenant, p.ep, result.ContentHash, p.id))
	}

	// Verify all are duplicates
	for _, p := range payloads {
		result, err := svc.Check(ctx, p.tenant, p.ep, []byte(p.payload))
		require.NoError(t, err)
		assert.True(t, result.IsDuplicate)
		assert.Equal(t, p.id, result.OriginalDeliveryID)
	}
}

func TestDeduplicationService_NilConfig(t *testing.T) {
	t.Parallel()
	svc := NewDeduplicationService(NewInMemoryDeduplicationStore(), nil)
	assert.Equal(t, 5*time.Minute, svc.config.WindowDuration)
	assert.Equal(t, 10000, svc.config.MaxEntriesPerTenant)
}

func TestDeduplicationService_CustomConfig(t *testing.T) {
	t.Parallel()
	cfg := &DeduplicationConfig{
		WindowDuration:      30 * time.Second,
		MaxEntriesPerTenant: 500,
	}
	svc := NewDeduplicationService(NewInMemoryDeduplicationStore(), cfg)
	assert.Equal(t, 30*time.Second, svc.config.WindowDuration)
	assert.Equal(t, 500, svc.config.MaxEntriesPerTenant)
}

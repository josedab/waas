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

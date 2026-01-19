package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// DeduplicationService provides content-based deduplication
type DeduplicationService struct {
	store  DeduplicationStore
	config *DeduplicationConfig
}

// DeduplicationConfig holds deduplication configuration
type DeduplicationConfig struct {
	// WindowDuration is the time window for duplicate detection
	WindowDuration time.Duration
	
	// MaxEntriesPerTenant limits memory usage per tenant
	MaxEntriesPerTenant int
}

// DefaultDeduplicationConfig returns default configuration
func DefaultDeduplicationConfig() *DeduplicationConfig {
	return &DeduplicationConfig{
		WindowDuration:      5 * time.Minute,
		MaxEntriesPerTenant: 10000,
	}
}

// DeduplicationStore defines the interface for deduplication storage
type DeduplicationStore interface {
	// Check checks if a content hash exists and returns the original delivery ID
	Check(ctx context.Context, tenantID, endpointID, contentHash string) (string, bool, error)
	
	// Record records a content hash with the delivery ID
	Record(ctx context.Context, tenantID, endpointID, contentHash, deliveryID string, expiration time.Duration) error
}

// NewDeduplicationService creates a new deduplication service
func NewDeduplicationService(store DeduplicationStore, config *DeduplicationConfig) *DeduplicationService {
	if config == nil {
		config = DefaultDeduplicationConfig()
	}
	return &DeduplicationService{
		store:  store,
		config: config,
	}
}

// DeduplicationResult holds the result of a deduplication check
type DeduplicationResult struct {
	IsDuplicate        bool
	OriginalDeliveryID string
	ContentHash        string
}

// Check checks if the payload is a duplicate
func (s *DeduplicationService) Check(ctx context.Context, tenantID, endpointID string, payload []byte) (*DeduplicationResult, error) {
	contentHash := s.hashContent(tenantID, endpointID, payload)
	
	originalID, isDuplicate, err := s.store.Check(ctx, tenantID, endpointID, contentHash)
	if err != nil {
		return nil, fmt.Errorf("deduplication check failed: %w", err)
	}
	
	return &DeduplicationResult{
		IsDuplicate:        isDuplicate,
		OriginalDeliveryID: originalID,
		ContentHash:        contentHash,
	}, nil
}

// Record records a delivery for future deduplication
func (s *DeduplicationService) Record(ctx context.Context, tenantID, endpointID, contentHash, deliveryID string) error {
	return s.store.Record(ctx, tenantID, endpointID, contentHash, deliveryID, s.config.WindowDuration)
}

func (s *DeduplicationService) hashContent(tenantID, endpointID string, payload []byte) string {
	h := sha256.New()
	h.Write([]byte(tenantID))
	h.Write([]byte(":"))
	h.Write([]byte(endpointID))
	h.Write([]byte(":"))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// InMemoryDeduplicationStore provides an in-memory deduplication store
type InMemoryDeduplicationStore struct {
	mu      sync.RWMutex
	entries map[string]*deduplicationEntry
}

type deduplicationEntry struct {
	deliveryID string
	expiresAt  time.Time
}

// NewInMemoryDeduplicationStore creates a new in-memory deduplication store
func NewInMemoryDeduplicationStore() *InMemoryDeduplicationStore {
	store := &InMemoryDeduplicationStore{
		entries: make(map[string]*deduplicationEntry),
	}
	go store.cleanup()
	return store
}

func (s *InMemoryDeduplicationStore) makeKey(tenantID, endpointID, contentHash string) string {
	return tenantID + ":" + endpointID + ":" + contentHash
}

// Check checks if a content hash exists
func (s *InMemoryDeduplicationStore) Check(ctx context.Context, tenantID, endpointID, contentHash string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	key := s.makeKey(tenantID, endpointID, contentHash)
	entry, exists := s.entries[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return "", false, nil
	}
	
	return entry.deliveryID, true, nil
}

// Record records a content hash
func (s *InMemoryDeduplicationStore) Record(ctx context.Context, tenantID, endpointID, contentHash, deliveryID string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := s.makeKey(tenantID, endpointID, contentHash)
	s.entries[key] = &deduplicationEntry{
		deliveryID: deliveryID,
		expiresAt:  time.Now().Add(expiration),
	}
	
	return nil
}

func (s *InMemoryDeduplicationStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.entries {
			if now.After(entry.expiresAt) {
				delete(s.entries, key)
			}
		}
		s.mu.Unlock()
	}
}

// RedisDeduplicationStore provides a Redis-backed deduplication store
type RedisDeduplicationStore struct {
	client RedisClient
	prefix string
}

// NewRedisDeduplicationStore creates a new Redis-backed deduplication store
func NewRedisDeduplicationStore(client RedisClient, prefix string) *RedisDeduplicationStore {
	if prefix == "" {
		prefix = "dedup:"
	}
	return &RedisDeduplicationStore{
		client: client,
		prefix: prefix,
	}
}

func (s *RedisDeduplicationStore) makeKey(tenantID, endpointID, contentHash string) string {
	return s.prefix + tenantID + ":" + endpointID + ":" + contentHash
}

// Check checks if a content hash exists
func (s *RedisDeduplicationStore) Check(ctx context.Context, tenantID, endpointID, contentHash string) (string, bool, error) {
	key := s.makeKey(tenantID, endpointID, contentHash)
	deliveryID, err := s.client.Get(ctx, key)
	if err != nil || deliveryID == "" {
		return "", false, nil
	}
	return deliveryID, true, nil
}

// Record records a content hash
func (s *RedisDeduplicationStore) Record(ctx context.Context, tenantID, endpointID, contentHash, deliveryID string, expiration time.Duration) error {
	key := s.makeKey(tenantID, endpointID, contentHash)
	return s.client.Set(ctx, key, deliveryID, expiration)
}

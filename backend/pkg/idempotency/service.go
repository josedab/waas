package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Key represents an idempotency key record
type Key struct {
	Key          string          `json:"key" db:"key"`
	TenantID     string          `json:"tenant_id" db:"tenant_id"`
	RequestHash  string          `json:"request_hash" db:"request_hash"`
	Response     json.RawMessage `json:"response" db:"response"`
	StatusCode   int             `json:"status_code" db:"status_code"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt    time.Time       `json:"expires_at" db:"expires_at"`
	IsProcessing bool            `json:"is_processing" db:"is_processing"`
}

// Store defines the interface for idempotency key storage
type Store interface {
	// Get retrieves an idempotency key record
	Get(ctx context.Context, tenantID, key string) (*Key, error)
	
	// Create creates a new idempotency key in processing state
	Create(ctx context.Context, key *Key) error
	
	// Update updates an idempotency key with the response
	Update(ctx context.Context, key *Key) error
	
	// Delete removes an idempotency key
	Delete(ctx context.Context, tenantID, key string) error
	
	// Cleanup removes expired keys
	Cleanup(ctx context.Context) (int64, error)
}

// Config holds idempotency configuration
type Config struct {
	// DefaultTTL is the default time-to-live for idempotency keys
	DefaultTTL time.Duration
	
	// MaxKeyLength is the maximum length of an idempotency key
	MaxKeyLength int
	
	// EnableRequestHashing enables request body hashing for validation
	EnableRequestHashing bool
}

// DefaultConfig returns default idempotency configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTTL:           24 * time.Hour,
		MaxKeyLength:         255,
		EnableRequestHashing: true,
	}
}

// Service provides idempotency functionality
type Service struct {
	store  Store
	config *Config
}

// NewService creates a new idempotency service
func NewService(store Store, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}
	return &Service{
		store:  store,
		config: config,
	}
}

// Result represents the result of an idempotency check
type Result struct {
	// IsNew indicates if this is a new request
	IsNew bool
	
	// IsProcessing indicates if a duplicate request is still being processed
	IsProcessing bool
	
	// CachedResponse is the cached response if available
	CachedResponse json.RawMessage
	
	// CachedStatusCode is the cached status code
	CachedStatusCode int
	
	// Key is the idempotency key record
	Key *Key
}

// Check checks if a request is idempotent and returns the cached response if available
func (s *Service) Check(ctx context.Context, tenantID, idempotencyKey string, requestBody []byte) (*Result, error) {
	if idempotencyKey == "" {
		return &Result{IsNew: true}, nil
	}

	if len(idempotencyKey) > s.config.MaxKeyLength {
		return nil, fmt.Errorf("idempotency key exceeds maximum length of %d", s.config.MaxKeyLength)
	}

	existing, err := s.store.Get(ctx, tenantID, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	if existing == nil {
		// New request - create key in processing state
		requestHash := ""
		if s.config.EnableRequestHashing && len(requestBody) > 0 {
			requestHash = hashRequest(requestBody)
		}

		key := &Key{
			Key:          idempotencyKey,
			TenantID:     tenantID,
			RequestHash:  requestHash,
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(s.config.DefaultTTL),
			IsProcessing: true,
		}

		if err := s.store.Create(ctx, key); err != nil {
			return nil, fmt.Errorf("failed to create idempotency key: %w", err)
		}

		return &Result{IsNew: true, Key: key}, nil
	}

	// Existing key found
	if existing.IsProcessing {
		return &Result{
			IsNew:        false,
			IsProcessing: true,
			Key:          existing,
		}, nil
	}

	// Validate request hash if enabled
	if s.config.EnableRequestHashing && existing.RequestHash != "" {
		currentHash := hashRequest(requestBody)
		if currentHash != existing.RequestHash {
			return nil, fmt.Errorf("request body does not match original request for idempotency key")
		}
	}

	return &Result{
		IsNew:            false,
		IsProcessing:     false,
		CachedResponse:   existing.Response,
		CachedStatusCode: existing.StatusCode,
		Key:              existing,
	}, nil
}

// Complete marks an idempotency key as completed with the response
func (s *Service) Complete(ctx context.Context, tenantID, idempotencyKey string, statusCode int, response []byte) error {
	if idempotencyKey == "" {
		return nil
	}

	key, err := s.store.Get(ctx, tenantID, idempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to get idempotency key: %w", err)
	}

	if key == nil {
		return fmt.Errorf("idempotency key not found")
	}

	key.StatusCode = statusCode
	key.Response = response
	key.IsProcessing = false

	if err := s.store.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to update idempotency key: %w", err)
	}

	return nil
}

// Abort removes an idempotency key when processing fails
func (s *Service) Abort(ctx context.Context, tenantID, idempotencyKey string) error {
	if idempotencyKey == "" {
		return nil
	}

	if err := s.store.Delete(ctx, tenantID, idempotencyKey); err != nil {
		return fmt.Errorf("failed to delete idempotency key: %w", err)
	}

	return nil
}

// Cleanup removes expired idempotency keys
func (s *Service) Cleanup(ctx context.Context) (int64, error) {
	return s.store.Cleanup(ctx)
}

func hashRequest(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

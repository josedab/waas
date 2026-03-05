package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostgresStore implements Store using PostgreSQL
type PostgresStore struct {
	db *sqlx.DB
}

// NewPostgresStore creates a new PostgreSQL-backed idempotency store
func NewPostgresStore(db *sqlx.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Get retrieves an idempotency key record
func (s *PostgresStore) Get(ctx context.Context, tenantID, key string) (*Key, error) {
	var record Key
	query := `
		SELECT key, tenant_id, request_hash, response, status_code, 
		       created_at, expires_at, is_processing
		FROM idempotency_keys
		WHERE tenant_id = $1 AND key = $2 AND expires_at > NOW()
	`

	err := s.db.GetContext(ctx, &record, query, tenantID, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// Create creates a new idempotency key in processing state
func (s *PostgresStore) Create(ctx context.Context, key *Key) error {
	query := `
		INSERT INTO idempotency_keys (key, tenant_id, request_hash, created_at, expires_at, is_processing)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, key) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		key.Key, key.TenantID, key.RequestHash,
		key.CreatedAt, key.ExpiresAt, key.IsProcessing)
	return err
}

// Update updates an idempotency key with the response
func (s *PostgresStore) Update(ctx context.Context, key *Key) error {
	query := `
		UPDATE idempotency_keys
		SET response = $1, status_code = $2, is_processing = $3
		WHERE tenant_id = $4 AND key = $5
	`

	_, err := s.db.ExecContext(ctx, query,
		key.Response, key.StatusCode, key.IsProcessing,
		key.TenantID, key.Key)
	return err
}

// Delete removes an idempotency key
func (s *PostgresStore) Delete(ctx context.Context, tenantID, key string) error {
	query := `DELETE FROM idempotency_keys WHERE tenant_id = $1 AND key = $2`
	_, err := s.db.ExecContext(ctx, query, tenantID, key)
	return err
}

// Cleanup removes expired keys
func (s *PostgresStore) Cleanup(ctx context.Context) (int64, error) {
	query := `DELETE FROM idempotency_keys WHERE expires_at < NOW()`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// RedisStore implements Store using Redis
type RedisStore struct {
	client RedisClient
	prefix string
}

// RedisClient defines the Redis client interface
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
}

// NewRedisStore creates a new Redis-backed idempotency store
func NewRedisStore(client RedisClient, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "idempotency:"
	}
	return &RedisStore{
		client: client,
		prefix: prefix,
	}
}

func (s *RedisStore) makeKey(tenantID, key string) string {
	return s.prefix + tenantID + ":" + key
}

// Get retrieves an idempotency key record
func (s *RedisStore) Get(ctx context.Context, tenantID, key string) (*Key, error) {
	redisKey := s.makeKey(tenantID, key)
	data, err := s.client.Get(ctx, redisKey)
	if err != nil {
		// Assume key not found for any error
		return nil, nil
	}
	if data == "" {
		return nil, nil
	}

	var record Key
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, err
	}

	return &record, nil
}

// Create creates a new idempotency key in processing state
func (s *RedisStore) Create(ctx context.Context, key *Key) error {
	redisKey := s.makeKey(key.TenantID, key.Key)
	data, err := json.Marshal(key)
	if err != nil {
		return err
	}

	ttl := time.Until(key.ExpiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	_, err = s.client.SetNX(ctx, redisKey, data, ttl)
	return err
}

// Update updates an idempotency key with the response
func (s *RedisStore) Update(ctx context.Context, key *Key) error {
	redisKey := s.makeKey(key.TenantID, key.Key)
	data, err := json.Marshal(key)
	if err != nil {
		return err
	}

	ttl := time.Until(key.ExpiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return s.client.Set(ctx, redisKey, data, ttl)
}

// Delete removes an idempotency key
func (s *RedisStore) Delete(ctx context.Context, tenantID, key string) error {
	redisKey := s.makeKey(tenantID, key)
	return s.client.Del(ctx, redisKey)
}

// Cleanup is a no-op for Redis as keys auto-expire
func (s *RedisStore) Cleanup(ctx context.Context) (int64, error) {
	// Redis keys auto-expire, no cleanup needed
	return 0, nil
}

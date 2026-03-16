package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connection pool configuration defaults
const (
	DefaultMaxConns           = 30
	DefaultMinConns           = 5
	DefaultMaxConnLifetime    = time.Hour
	DefaultMaxConnIdleTime    = 30 * time.Minute
	DefaultHealthCheckTimeout = 5 * time.Second

	// Test pool configuration defaults (smaller pool for test environments)
	TestMaxConns        = 5
	TestMinConns        = 1
	TestMaxConnLifetime = 10 * time.Minute
	TestMaxConnIdleTime = 5 * time.Minute
)

// DB wraps a pgxpool connection pool for PostgreSQL access.
type DB struct {
	Pool *pgxpool.Pool
}

// NewConnection creates a new database connection pool using the DATABASE_URL
// environment variable. It configures the pool with default settings and
// verifies connectivity before returning.
func NewConnection() (*DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool — honour env overrides for production tuning.
	config.MaxConns = int32(getEnvInt("DB_MAX_CONNS", int(DefaultMaxConns)))
	config.MinConns = int32(getEnvInt("DB_MIN_CONNS", int(DefaultMinConns)))
	config.MaxConnLifetime = getEnvDuration("DB_CONN_MAX_LIFETIME", DefaultMaxConnLifetime)
	config.MaxConnIdleTime = getEnvDuration("DB_CONN_MAX_IDLE_TIME", DefaultMaxConnIdleTime)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close releases all connections in the pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// GetStdDB returns a *sql.DB from the connection string for compatibility
// with components that need *sql.DB instead of pgxpool
func GetStdDB() (*sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	return sql.Open("pgx", databaseURL)
}

// HealthCheck performs a health check on the database connection
func (db *DB) HealthCheck() error {
	if db.Pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultHealthCheckTimeout)
	defer cancel()

	return db.Pool.Ping(ctx)
}

// NewTestConnection creates a database connection for testing
func NewTestConnection() (*DB, error) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("TEST_DATABASE_URL environment variable is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test database URL: %w", err)
	}

	// Configure connection pool for testing (smaller pool)
	config.MaxConns = TestMaxConns
	config.MinConns = TestMinConns
	config.MaxConnLifetime = TestMaxConnLifetime
	config.MaxConnIdleTime = TestMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping test database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// GetConnectionString returns the connection string for the database
func (db *DB) GetConnectionString() string {
	// This is a simplified version for testing
	// In production, you might want to store this or reconstruct it properly
	return os.Getenv("DATABASE_URL")
}

// getEnvInt reads an env var as an int, falling back to defaultVal.
func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

// getEnvDuration reads an env var as a Go duration string (e.g. "1h", "30m"),
// falling back to defaultVal.
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultVal
}

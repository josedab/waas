package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewConnection() (*DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	config.MaxConns = 30
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

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
		databaseURL = "postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable"
	}
	return sql.Open("pgx", databaseURL)
}

// HealthCheck performs a health check on the database connection
func (db *DB) HealthCheck() error {
	if db.Pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return db.Pool.Ping(ctx)
}

// NewTestConnection creates a database connection for testing
func NewTestConnection() (*DB, error) {
	// Use test database URL or fallback to default
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test database URL: %w", err)
	}

	// Configure connection pool for testing (smaller pool)
	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = time.Minute * 10
	config.MaxConnIdleTime = time.Minute * 5

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
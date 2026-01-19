package database

import (
	"context"
	"fmt"
	"os"
)

// NewTestRedisConnection creates a Redis connection for testing
func NewTestRedisConnection() (*RedisClient, error) {
	// Use test Redis URL if available, otherwise use regular Redis
	testRedisURL := os.Getenv("TEST_REDIS_URL")
	if testRedisURL == "" {
		testRedisURL = "redis://localhost:6379/1" // Use database 1 for testing
	}
	
	return NewRedisConnection(testRedisURL)
}

// CleanTestDatabase removes all data from test database tables
func CleanTestDatabase(db *DB) error {
	ctx := context.Background()
	
	// Clean tables in correct order (respecting foreign keys)
	tables := []string{
		"delivery_attempts",
		"webhook_endpoints", 
		"tenants",
	}
	
	for _, table := range tables {
		_, err := db.Pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clean table %s: %w", table, err)
		}
	}
	
	return nil
}
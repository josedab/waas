package testutil

import (
	"os"
	"testing"
)

// SetEnv sets an environment variable for the duration of a test and restores
// the original value during cleanup.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()
	prev, existed := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}

// NewTestConfig sets up minimal environment variables required by
// utils.LoadConfig and returns a cleanup function. Call in TestMain or
// individual tests that need a valid configuration.
func NewTestConfig(t *testing.T) {
	t.Helper()
	defaults := map[string]string{
		"DATABASE_URL": "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable",
		"REDIS_URL":    "redis://localhost:6379/10",
		"JWT_SECRET":   "test-secret-key-for-testing-only",
		"ENVIRONMENT":  "test",
		"LOG_LEVEL":    "error",
	}
	for k, v := range defaults {
		if os.Getenv(k) == "" {
			SetEnv(t, k, v)
		}
	}
}

// SkipIfShort calls t.Skip when the -short flag is set.
// Useful for tests that require external services.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}
}

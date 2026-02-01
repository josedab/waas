package repository

import (
	"os"
	"testing"
	"github.com/josedab/waas/pkg/database"
)

func setupTestDB(t *testing.T) *database.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database integration test")
	}
	db, err := database.NewTestConnection()
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return db
}

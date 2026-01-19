package database

import (
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// NewSQLxConnection creates a new sqlx database connection
// This is used by the new feature packages that use sqlx
func NewSQLxConnection() (*sqlx.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(5)

	return db, nil
}

// NewTestSQLxConnection creates a sqlx connection for testing
func NewTestSQLxConnection() (*sqlx.DB, error) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(1)

	return db, nil
}

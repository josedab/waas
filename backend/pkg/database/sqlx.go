package database

import (
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// NewSQLxConnection creates a new sqlx database connection.
// DATABASE_URL env var must be set; returns an error if missing.
func NewSQLxConnection() (*sqlx.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := sqlx.Connect("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(5)

	return db, nil
}

// NewTestSQLxConnection creates a sqlx connection for testing.
// TEST_DATABASE_URL env var must be set; returns an error if missing.
func NewTestSQLxConnection() (*sqlx.DB, error) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("TEST_DATABASE_URL environment variable is required")
	}

	db, err := sqlx.Connect("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(1)

	return db, nil
}

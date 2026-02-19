package database

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/josedab/waas/pkg/utils"
)

var logger = utils.NewLogger("database")

// RunMigrations applies all pending database migrations from the migrations/
// directory. Returns nil if migrations succeed or if no new migrations exist.
func RunMigrations(databaseURL string) error {
	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Database migrations completed successfully", nil)
	return nil
}

// RollbackMigrations reverts the specified number of migration steps.
func RollbackMigrations(databaseURL string, steps int) error {
	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	logger.Info("Rolled back migration steps successfully", map[string]interface{}{"steps": steps})
	return nil
}
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/josedab/waas/pkg/database"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type tenantRepository struct {
	db *database.DB
}

// NewTenantRepository creates a new tenant repository instance
func NewTenantRepository(db *database.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, api_key_hash, subscription_tier, rate_limit_per_minute, monthly_quota, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	now := time.Now()
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.APIKeyHash,
		tenant.SubscriptionTier,
		tenant.RateLimitPerMinute,
		tenant.MonthlyQuota,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	query := `
		SELECT id, name, api_key_hash, subscription_tier, rate_limit_per_minute, monthly_quota, created_at, updated_at
		FROM tenants WHERE id = $1`

	var tenant models.Tenant
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.APIKeyHash,
		&tenant.SubscriptionTier,
		&tenant.RateLimitPerMinute,
		&tenant.MonthlyQuota,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant: %w", apperrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return &tenant, nil
}

func (r *tenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Tenant, error) {
	query := `
		SELECT id, name, api_key_hash, subscription_tier, rate_limit_per_minute, monthly_quota, created_at, updated_at
		FROM tenants WHERE api_key_hash = $1`

	var tenant models.Tenant
	err := r.db.Pool.QueryRow(ctx, query, apiKeyHash).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.APIKeyHash,
		&tenant.SubscriptionTier,
		&tenant.RateLimitPerMinute,
		&tenant.MonthlyQuota,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant: %w", apperrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get tenant by API key: %w", err)
	}

	return &tenant, nil
}

func (r *tenantRepository) FindByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	// Get all tenants and validate API key against each hash
	// This is necessary because bcrypt hashes are one-way
	query := `
		SELECT id, name, api_key_hash, subscription_tier, rate_limit_per_minute, monthly_quota, created_at, updated_at
		FROM tenants`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tenant models.Tenant
		err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.APIKeyHash,
			&tenant.SubscriptionTier,
			&tenant.RateLimitPerMinute,
			&tenant.MonthlyQuota,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		// Validate API key against stored hash
		if validateAPIKey(apiKey, tenant.APIKeyHash) {
			return &tenant, nil
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant rows: %w", err)
	}

	return nil, fmt.Errorf("tenant: %w", apperrors.ErrNotFound)
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	query := `
		UPDATE tenants 
		SET name = $2, api_key_hash = $3, subscription_tier = $4, rate_limit_per_minute = $5, 
		    monthly_quota = $6, updated_at = $7
		WHERE id = $1`

	tenant.UpdatedAt = time.Now()

	result, err := r.db.Pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.APIKeyHash,
		tenant.SubscriptionTier,
		tenant.RateLimitPerMinute,
		tenant.MonthlyQuota,
		tenant.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tenants WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant: %w", apperrors.ErrNotFound)
	}

	return nil
}

func (r *tenantRepository) List(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	query := `
		SELECT id, name, api_key_hash, subscription_tier, rate_limit_per_minute, monthly_quota, created_at, updated_at
		FROM tenants 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		var tenant models.Tenant
		err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.APIKeyHash,
			&tenant.SubscriptionTier,
			&tenant.RateLimitPerMinute,
			&tenant.MonthlyQuota,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, &tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant rows: %w", err)
	}

	return tenants, nil
}

// validateAPIKey checks if the provided API key matches the stored hash
func validateAPIKey(apiKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
	return err == nil
}
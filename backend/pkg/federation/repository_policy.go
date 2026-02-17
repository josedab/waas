package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *PostgresRepository) SavePolicy(ctx context.Context, policy *FederationPolicy) error {
	allowedJSON, err := json.Marshal(policy.AllowedDomains)
	if err != nil {
		return fmt.Errorf("failed to marshal allowed domains: %w", err)
	}
	blockedJSON, err := json.Marshal(policy.BlockedDomains)
	if err != nil {
		return fmt.Errorf("failed to marshal blocked domains: %w", err)
	}

	query := `
		INSERT INTO federation_policies (
			id, tenant_id, enabled, auto_accept_trust, min_trust_level,
			allowed_domains, blocked_domains, require_encryption, allow_relay,
			max_subscriptions, rate_limit_per_member, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			auto_accept_trust = EXCLUDED.auto_accept_trust,
			min_trust_level = EXCLUDED.min_trust_level,
			allowed_domains = EXCLUDED.allowed_domains,
			blocked_domains = EXCLUDED.blocked_domains,
			require_encryption = EXCLUDED.require_encryption,
			allow_relay = EXCLUDED.allow_relay,
			max_subscriptions = EXCLUDED.max_subscriptions,
			rate_limit_per_member = EXCLUDED.rate_limit_per_member,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		policy.ID, policy.TenantID, policy.Enabled, policy.AutoAcceptTrust,
		policy.MinTrustLevel, allowedJSON, blockedJSON, policy.RequireEncryption,
		policy.AllowRelay, policy.MaxSubscriptions, policy.RateLimitPerMember,
		policy.CreatedAt, policy.UpdatedAt)

	return err
}

// GetPolicy retrieves federation policy
func (r *PostgresRepository) GetPolicy(ctx context.Context, tenantID string) (*FederationPolicy, error) {
	query := `
		SELECT id, tenant_id, enabled, auto_accept_trust, min_trust_level,
			   allowed_domains, blocked_domains, require_encryption, allow_relay,
			   max_subscriptions, rate_limit_per_member, created_at, updated_at
		FROM federation_policies
		WHERE tenant_id = $1`

	var p FederationPolicy
	var allowedJSON, blockedJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Enabled, &p.AutoAcceptTrust, &p.MinTrustLevel,
		&allowedJSON, &blockedJSON, &p.RequireEncryption, &p.AllowRelay,
		&p.MaxSubscriptions, &p.RateLimitPerMember, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(allowedJSON, &p.AllowedDomains); err != nil {
		r.logger.Warn("failed to unmarshal policy allowed domains", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}
	if err := json.Unmarshal(blockedJSON, &p.BlockedDomains); err != nil {
		r.logger.Warn("failed to unmarshal policy blocked domains", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
	}

	return &p, nil
}

// SaveKeys saves crypto keys
func (r *PostgresRepository) SaveKeys(ctx context.Context, keys *CryptoKeys) error {
	query := `
		INSERT INTO federation_keys (
			member_id, public_key, algorithm, key_id, created_at, expires_at, revoked
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (member_id) DO UPDATE SET
			public_key = EXCLUDED.public_key,
			algorithm = EXCLUDED.algorithm,
			key_id = EXCLUDED.key_id,
			expires_at = EXCLUDED.expires_at,
			revoked = EXCLUDED.revoked`

	_, err := r.db.ExecContext(ctx, query,
		keys.MemberID, keys.PublicKey, keys.Algorithm, keys.KeyID,
		keys.CreatedAt, keys.ExpiresAt, keys.Revoked)

	return err
}

// GetKeys retrieves keys for a member
func (r *PostgresRepository) GetKeys(ctx context.Context, memberID string) (*CryptoKeys, error) {
	query := `
		SELECT member_id, public_key, algorithm, key_id, created_at, expires_at, revoked
		FROM federation_keys
		WHERE member_id = $1 AND revoked = false`

	var k CryptoKeys
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, memberID).Scan(
		&k.MemberID, &k.PublicKey, &k.Algorithm, &k.KeyID,
		&k.CreatedAt, &expiresAt, &k.Revoked)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("keys not found")
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}

	return &k, nil
}

// GetKeyByID retrieves key by key ID
func (r *PostgresRepository) GetKeyByID(ctx context.Context, keyID string) (*CryptoKeys, error) {
	query := `
		SELECT member_id, public_key, algorithm, key_id, created_at, expires_at, revoked
		FROM federation_keys
		WHERE key_id = $1`

	var k CryptoKeys
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, keyID).Scan(
		&k.MemberID, &k.PublicKey, &k.Algorithm, &k.KeyID,
		&k.CreatedAt, &expiresAt, &k.Revoked)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found")
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}

	return &k, nil
}

// GetMetrics retrieves federation metrics
func (r *PostgresRepository) GetMetrics(ctx context.Context, tenantID string) (*FederationMetrics, error) {
	metrics := &FederationMetrics{TenantID: tenantID, UpdatedAt: time.Now()}

	// Count members
	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM federation_members WHERE tenant_id = $1", tenantID).
		Scan(&metrics.TotalMembers)

	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM federation_members WHERE tenant_id = $1 AND status = 'active'", tenantID).
		Scan(&metrics.ActiveMembers)

	// Count subscriptions
	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM federation_subscriptions WHERE tenant_id = $1", tenantID).
		Scan(&metrics.TotalSubscriptions)

	// Delivery stats
	r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'succeeded'),
			COUNT(*) FILTER (WHERE status = 'failed')
		FROM federation_deliveries
		WHERE tenant_id = $1`, tenantID).
		Scan(&metrics.TotalDeliveries, &metrics.SuccessfulDeliveries, &metrics.FailedDeliveries)

	// Average latency
	r.db.QueryRowContext(ctx,
		"SELECT COALESCE(AVG(latency), 0) FROM federation_deliveries WHERE tenant_id = $1 AND status = 'succeeded'", tenantID).
		Scan(&metrics.AverageLatency)

	return metrics, nil
}

// UpdateMetrics updates federation metrics (used for caching)
func (r *PostgresRepository) UpdateMetrics(ctx context.Context, metrics *FederationMetrics) error {
	// This would typically update a metrics cache table
	return nil
}

// GenerateMemberID generates a new member ID
func GenerateMemberID() string {
	return uuid.New().String()
}

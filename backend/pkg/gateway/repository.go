package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for gateway storage
type Repository interface {
	// Provider operations
	CreateProvider(ctx context.Context, provider *Provider) error
	GetProvider(ctx context.Context, tenantID, providerID string) (*Provider, error)
	ListProviders(ctx context.Context, tenantID string, limit, offset int) ([]Provider, int, error)
	UpdateProvider(ctx context.Context, provider *Provider) error
	DeleteProvider(ctx context.Context, tenantID, providerID string) error

	// Routing rule operations
	CreateRoutingRule(ctx context.Context, rule *RoutingRule) error
	GetRoutingRule(ctx context.Context, tenantID, ruleID string) (*RoutingRule, error)
	ListRoutingRules(ctx context.Context, tenantID, providerID string) ([]RoutingRule, error)
	UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error
	DeleteRoutingRule(ctx context.Context, tenantID, ruleID string) error

	// Inbound webhook operations
	SaveInboundWebhook(ctx context.Context, webhook *InboundWebhook) error
	GetInboundWebhook(ctx context.Context, tenantID, webhookID string) (*InboundWebhook, error)
	ListInboundWebhooks(ctx context.Context, tenantID string, providerID string, limit, offset int) ([]InboundWebhook, int, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL gateway repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateProvider creates a new provider
func (r *PostgresRepository) CreateProvider(ctx context.Context, provider *Provider) error {
	if provider.ID == "" {
		provider.ID = uuid.New().String()
	}
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	query := `
		INSERT INTO gateway_providers (id, tenant_id, name, type, description, is_active, signature_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.TenantID, provider.Name, provider.Type, provider.Description,
		provider.IsActive, provider.SignatureConfig, provider.CreatedAt, provider.UpdatedAt)
	return err
}

// GetProvider retrieves a provider by ID
func (r *PostgresRepository) GetProvider(ctx context.Context, tenantID, providerID string) (*Provider, error) {
	var provider Provider
	query := `SELECT * FROM gateway_providers WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &provider, query, tenantID, providerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &provider, err
}

// ListProviders lists all providers for a tenant
func (r *PostgresRepository) ListProviders(ctx context.Context, tenantID string, limit, offset int) ([]Provider, int, error) {
	var providers []Provider
	var total int

	countQuery := `SELECT COUNT(*) FROM gateway_providers WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `SELECT * FROM gateway_providers WHERE tenant_id = $1 ORDER BY name ASC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &providers, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

// UpdateProvider updates a provider
func (r *PostgresRepository) UpdateProvider(ctx context.Context, provider *Provider) error {
	provider.UpdatedAt = time.Now()
	query := `
		UPDATE gateway_providers 
		SET name = $1, description = $2, is_active = $3, signature_config = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7
	`
	_, err := r.db.ExecContext(ctx, query,
		provider.Name, provider.Description, provider.IsActive, provider.SignatureConfig,
		provider.UpdatedAt, provider.ID, provider.TenantID)
	return err
}

// DeleteProvider deletes a provider
func (r *PostgresRepository) DeleteProvider(ctx context.Context, tenantID, providerID string) error {
	query := `DELETE FROM gateway_providers WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, providerID)
	return err
}

// CreateRoutingRule creates a new routing rule
func (r *PostgresRepository) CreateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	query := `
		INSERT INTO gateway_routing_rules (id, tenant_id, provider_id, name, description, priority, is_active, conditions, destinations, transform, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		rule.ID, rule.TenantID, rule.ProviderID, rule.Name, rule.Description,
		rule.Priority, rule.IsActive, rule.Conditions, rule.Destinations, rule.Transform,
		rule.CreatedAt, rule.UpdatedAt)
	return err
}

// GetRoutingRule retrieves a routing rule by ID
func (r *PostgresRepository) GetRoutingRule(ctx context.Context, tenantID, ruleID string) (*RoutingRule, error) {
	var rule RoutingRule
	query := `SELECT * FROM gateway_routing_rules WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &rule, query, tenantID, ruleID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &rule, err
}

// ListRoutingRules lists routing rules for a provider
func (r *PostgresRepository) ListRoutingRules(ctx context.Context, tenantID, providerID string) ([]RoutingRule, error) {
	var rules []RoutingRule
	query := `SELECT * FROM gateway_routing_rules WHERE tenant_id = $1 AND provider_id = $2 ORDER BY priority ASC, created_at ASC`
	err := r.db.SelectContext(ctx, &rules, query, tenantID, providerID)
	return rules, err
}

// UpdateRoutingRule updates a routing rule
func (r *PostgresRepository) UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	rule.UpdatedAt = time.Now()
	query := `
		UPDATE gateway_routing_rules 
		SET name = $1, description = $2, priority = $3, is_active = $4, 
		    conditions = $5, destinations = $6, transform = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10
	`
	_, err := r.db.ExecContext(ctx, query,
		rule.Name, rule.Description, rule.Priority, rule.IsActive,
		rule.Conditions, rule.Destinations, rule.Transform, rule.UpdatedAt,
		rule.ID, rule.TenantID)
	return err
}

// DeleteRoutingRule deletes a routing rule
func (r *PostgresRepository) DeleteRoutingRule(ctx context.Context, tenantID, ruleID string) error {
	query := `DELETE FROM gateway_routing_rules WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, ruleID)
	return err
}

// SaveInboundWebhook saves an inbound webhook
func (r *PostgresRepository) SaveInboundWebhook(ctx context.Context, webhook *InboundWebhook) error {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}
	webhook.CreatedAt = time.Now()

	headersJSON, _ := json.Marshal(webhook.Headers)

	query := `
		INSERT INTO inbound_webhooks (id, tenant_id, provider_id, provider_type, event_type, payload, headers, raw_body, signature_valid, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		webhook.ID, webhook.TenantID, webhook.ProviderID, webhook.ProviderType,
		webhook.EventType, webhook.Payload, headersJSON, webhook.RawBody,
		webhook.SignatureValid, webhook.CreatedAt)
	return err
}

// GetInboundWebhook retrieves an inbound webhook by ID
func (r *PostgresRepository) GetInboundWebhook(ctx context.Context, tenantID, webhookID string) (*InboundWebhook, error) {
	var webhook InboundWebhook
	query := `SELECT * FROM inbound_webhooks WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &webhook, query, tenantID, webhookID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err == nil && webhook.HeadersJSON != nil {
		json.Unmarshal(webhook.HeadersJSON, &webhook.Headers)
	}
	return &webhook, err
}

// ListInboundWebhooks lists inbound webhooks
func (r *PostgresRepository) ListInboundWebhooks(ctx context.Context, tenantID string, providerID string, limit, offset int) ([]InboundWebhook, int, error) {
	var webhooks []InboundWebhook
	var total int

	countQuery := `SELECT COUNT(*) FROM inbound_webhooks WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if providerID != "" {
		countQuery += ` AND provider_id = $2`
		args = append(args, providerID)
	}

	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	query := `SELECT * FROM inbound_webhooks WHERE tenant_id = $1`
	if providerID != "" {
		query += ` AND provider_id = $2`
	}
	query += ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+len(args)+1)) + ` OFFSET $` + string(rune('0'+len(args)+2))
	args = append(args, limit, offset)

	if err := r.db.SelectContext(ctx, &webhooks, query, args...); err != nil {
		return nil, 0, err
	}

	for i := range webhooks {
		if webhooks[i].HeadersJSON != nil {
			json.Unmarshal(webhooks[i].HeadersJSON, &webhooks[i].Headers)
		}
	}

	return webhooks, total, nil
}

package pushbridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// SaveMapping saves a push mapping
func (r *PostgresRepository) SaveMapping(ctx context.Context, mapping *PushMapping) error {
	configJSON, _ := json.Marshal(mapping.Config)
	templateJSON, _ := json.Marshal(mapping.Template)
	targetingJSON, _ := json.Marshal(mapping.Targeting)

	query := `
		INSERT INTO push_mappings (
			id, tenant_id, name, description, webhook_id, event_type,
			enabled, config, template, targeting, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			webhook_id = EXCLUDED.webhook_id,
			event_type = EXCLUDED.event_type,
			enabled = EXCLUDED.enabled,
			config = EXCLUDED.config,
			template = EXCLUDED.template,
			targeting = EXCLUDED.targeting,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		mapping.ID, mapping.TenantID, mapping.Name, mapping.Description,
		mapping.WebhookID, mapping.EventType, mapping.Enabled,
		configJSON, templateJSON, targetingJSON,
		mapping.CreatedAt, mapping.UpdatedAt)

	return err
}

// GetMapping retrieves a mapping
func (r *PostgresRepository) GetMapping(ctx context.Context, tenantID, mappingID string) (*PushMapping, error) {
	query := `
		SELECT id, tenant_id, name, description, webhook_id, event_type,
			   enabled, config, template, targeting, created_at, updated_at
		FROM push_mappings
		WHERE tenant_id = $1 AND id = $2`

	var mapping PushMapping
	var configJSON, templateJSON, targetingJSON []byte
	var description, webhookID, eventType sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, mappingID).Scan(
		&mapping.ID, &mapping.TenantID, &mapping.Name, &description,
		&webhookID, &eventType, &mapping.Enabled,
		&configJSON, &templateJSON, &targetingJSON,
		&mapping.CreatedAt, &mapping.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mapping not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(configJSON, &mapping.Config)
	json.Unmarshal(templateJSON, &mapping.Template)
	json.Unmarshal(targetingJSON, &mapping.Targeting)

	if description.Valid {
		mapping.Description = description.String
	}
	if webhookID.Valid {
		mapping.WebhookID = webhookID.String
	}
	if eventType.Valid {
		mapping.EventType = eventType.String
	}

	return &mapping, nil
}

// ListMappings lists mappings
func (r *PostgresRepository) ListMappings(ctx context.Context, tenantID string) ([]PushMapping, error) {
	query := `
		SELECT id, tenant_id, name, description, webhook_id, event_type,
			   enabled, config, template, targeting, created_at, updated_at
		FROM push_mappings
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []PushMapping
	for rows.Next() {
		var mapping PushMapping
		var configJSON, templateJSON, targetingJSON []byte
		var description, webhookID, eventType sql.NullString

		err := rows.Scan(
			&mapping.ID, &mapping.TenantID, &mapping.Name, &description,
			&webhookID, &eventType, &mapping.Enabled,
			&configJSON, &templateJSON, &targetingJSON,
			&mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(configJSON, &mapping.Config)
		json.Unmarshal(templateJSON, &mapping.Template)
		json.Unmarshal(targetingJSON, &mapping.Targeting)

		if description.Valid {
			mapping.Description = description.String
		}
		if webhookID.Valid {
			mapping.WebhookID = webhookID.String
		}
		if eventType.Valid {
			mapping.EventType = eventType.String
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// GetMappingByWebhook gets mappings for a webhook
func (r *PostgresRepository) GetMappingByWebhook(ctx context.Context, tenantID, webhookID string) ([]PushMapping, error) {
	query := `
		SELECT id, tenant_id, name, description, webhook_id, event_type,
			   enabled, config, template, targeting, created_at, updated_at
		FROM push_mappings
		WHERE tenant_id = $1 AND webhook_id = $2 AND enabled = true`

	rows, err := r.db.QueryContext(ctx, query, tenantID, webhookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []PushMapping
	for rows.Next() {
		var mapping PushMapping
		var configJSON, templateJSON, targetingJSON []byte
		var description, wID, eventType sql.NullString

		err := rows.Scan(
			&mapping.ID, &mapping.TenantID, &mapping.Name, &description,
			&wID, &eventType, &mapping.Enabled,
			&configJSON, &templateJSON, &targetingJSON,
			&mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(configJSON, &mapping.Config)
		json.Unmarshal(templateJSON, &mapping.Template)
		json.Unmarshal(targetingJSON, &mapping.Targeting)

		if description.Valid {
			mapping.Description = description.String
		}
		if wID.Valid {
			mapping.WebhookID = wID.String
		}
		if eventType.Valid {
			mapping.EventType = eventType.String
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// DeleteMapping deletes a mapping
func (r *PostgresRepository) DeleteMapping(ctx context.Context, tenantID, mappingID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_mappings WHERE tenant_id = $1 AND id = $2",
		tenantID, mappingID)
	return err
}

// GenerateMappingID generates a new mapping ID
func GenerateMappingID() string {
	return uuid.New().String()
}

package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func (r *PostgresRepository) SaveCatalog(ctx context.Context, catalog *EventCatalog) error {
	eventsJSON, err := json.Marshal(catalog.EventTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog event types: %w", err)
	}

	query := `
		INSERT INTO federation_catalogs (
			id, tenant_id, member_id, name, description, event_types,
			version, public, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			event_types = EXCLUDED.event_types,
			version = EXCLUDED.version,
			public = EXCLUDED.public,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		catalog.ID, catalog.TenantID, catalog.MemberID, catalog.Name,
		catalog.Description, eventsJSON, catalog.Version, catalog.Public,
		catalog.CreatedAt, catalog.UpdatedAt)

	return err
}

// GetCatalog retrieves a catalog
func (r *PostgresRepository) GetCatalog(ctx context.Context, catalogID string) (*EventCatalog, error) {
	query := `
		SELECT id, tenant_id, member_id, name, description, event_types,
			   version, public, created_at, updated_at
		FROM federation_catalogs
		WHERE id = $1`

	var c EventCatalog
	var eventsJSON []byte

	err := r.db.QueryRowContext(ctx, query, catalogID).Scan(
		&c.ID, &c.TenantID, &c.MemberID, &c.Name, &c.Description,
		&eventsJSON, &c.Version, &c.Public, &c.CreatedAt, &c.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("catalog not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(eventsJSON, &c.EventTypes); err != nil {
		r.logger.Warn("failed to unmarshal catalog event types", map[string]interface{}{"catalog_id": catalogID, "error": err.Error()})
	}

	return &c, nil
}

// ListCatalogs lists catalogs
func (r *PostgresRepository) ListCatalogs(ctx context.Context, tenantID string, public bool) ([]EventCatalog, error) {
	query := `
		SELECT id, tenant_id, member_id, name, description, event_types,
			   version, public, created_at, updated_at
		FROM federation_catalogs
		WHERE tenant_id = $1`

	if public {
		query += " AND public = true"
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalogs []EventCatalog
	for rows.Next() {
		var c EventCatalog
		var eventsJSON []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.MemberID, &c.Name, &c.Description,
			&eventsJSON, &c.Version, &c.Public, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &c.EventTypes); err != nil {
			r.logger.Warn("failed to unmarshal catalog event types", map[string]interface{}{"catalog_id": c.ID, "error": err.Error()})
		}
		catalogs = append(catalogs, c)
	}

	return catalogs, nil
}

// ListPublicCatalogs lists all public catalogs
func (r *PostgresRepository) ListPublicCatalogs(ctx context.Context) ([]EventCatalog, error) {
	query := `
		SELECT id, tenant_id, member_id, name, description, event_types,
			   version, public, created_at, updated_at
		FROM federation_catalogs
		WHERE public = true
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalogs []EventCatalog
	for rows.Next() {
		var c EventCatalog
		var eventsJSON []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.MemberID, &c.Name, &c.Description,
			&eventsJSON, &c.Version, &c.Public, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &c.EventTypes); err != nil {
			r.logger.Warn("failed to unmarshal public catalog event types", map[string]interface{}{"catalog_id": c.ID, "error": err.Error()})
		}
		catalogs = append(catalogs, c)
	}

	return catalogs, nil
}

// SaveSubscription saves a subscription
func (r *PostgresRepository) SaveSubscription(ctx context.Context, sub *FederatedSubscription) error {
	eventsJSON, err := json.Marshal(sub.EventTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription event types: %w", err)
	}
	filterJSON, err := json.Marshal(sub.Filter)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription filter: %w", err)
	}
	deliveryJSON, err := json.Marshal(sub.DeliveryConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription delivery config: %w", err)
	}

	query := `
		INSERT INTO federation_subscriptions (
			id, tenant_id, source_member_id, target_member_id, catalog_id,
			event_types, filter, status, delivery_config, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			event_types = EXCLUDED.event_types,
			filter = EXCLUDED.filter,
			status = EXCLUDED.status,
			delivery_config = EXCLUDED.delivery_config,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.SourceMemberID, sub.TargetMemberID, sub.CatalogID,
		eventsJSON, filterJSON, sub.Status, deliveryJSON, sub.CreatedAt, sub.UpdatedAt)

	return err
}

// GetSubscription retrieves a subscription
func (r *PostgresRepository) GetSubscription(ctx context.Context, subID string) (*FederatedSubscription, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, catalog_id,
			   event_types, filter, status, delivery_config, created_at, updated_at
		FROM federation_subscriptions
		WHERE id = $1`

	var s FederatedSubscription
	var eventsJSON, filterJSON, deliveryJSON []byte

	err := r.db.QueryRowContext(ctx, query, subID).Scan(
		&s.ID, &s.TenantID, &s.SourceMemberID, &s.TargetMemberID, &s.CatalogID,
		&eventsJSON, &filterJSON, &s.Status, &deliveryJSON, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(eventsJSON, &s.EventTypes); err != nil {
		r.logger.Warn("failed to unmarshal subscription event types", map[string]interface{}{"subscription_id": subID, "error": err.Error()})
	}
	if err := json.Unmarshal(deliveryJSON, &s.DeliveryConfig); err != nil {
		r.logger.Warn("failed to unmarshal subscription delivery config", map[string]interface{}{"subscription_id": subID, "error": err.Error()})
	}
	if len(filterJSON) > 0 {
		var f EventFilter
		if err := json.Unmarshal(filterJSON, &f); err != nil {
			r.logger.Warn("failed to unmarshal subscription filter", map[string]interface{}{"subscription_id": subID, "error": err.Error()})
		} else {
			s.Filter = &f
		}
	}

	return &s, nil
}

// ListSubscriptions lists subscriptions
func (r *PostgresRepository) ListSubscriptions(ctx context.Context, tenantID string, status *SubStatus) ([]FederatedSubscription, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, catalog_id,
			   event_types, filter, status, delivery_config, created_at, updated_at
		FROM federation_subscriptions
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []FederatedSubscription
	for rows.Next() {
		var s FederatedSubscription
		var eventsJSON, filterJSON, deliveryJSON []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.SourceMemberID, &s.TargetMemberID, &s.CatalogID,
			&eventsJSON, &filterJSON, &s.Status, &deliveryJSON, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &s.EventTypes); err != nil {
			r.logger.Warn("failed to unmarshal subscription event types", map[string]interface{}{"subscription_id": s.ID, "error": err.Error()})
		}
		if err := json.Unmarshal(deliveryJSON, &s.DeliveryConfig); err != nil {
			r.logger.Warn("failed to unmarshal subscription delivery config", map[string]interface{}{"subscription_id": s.ID, "error": err.Error()})
		}
		if len(filterJSON) > 0 {
			var f EventFilter
			if err := json.Unmarshal(filterJSON, &f); err != nil {
				r.logger.Warn("failed to unmarshal subscription filter", map[string]interface{}{"subscription_id": s.ID, "error": err.Error()})
			} else {
				s.Filter = &f
			}
		}

		subs = append(subs, s)
	}

	return subs, nil
}

// ListSubscriptionsByMember lists subscriptions by member
func (r *PostgresRepository) ListSubscriptionsByMember(ctx context.Context, memberID string) ([]FederatedSubscription, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, catalog_id,
			   event_types, filter, status, delivery_config, created_at, updated_at
		FROM federation_subscriptions
		WHERE target_member_id = $1 AND status = 'active'`

	rows, err := r.db.QueryContext(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []FederatedSubscription
	for rows.Next() {
		var s FederatedSubscription
		var eventsJSON, filterJSON, deliveryJSON []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.SourceMemberID, &s.TargetMemberID, &s.CatalogID,
			&eventsJSON, &filterJSON, &s.Status, &deliveryJSON, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &s.EventTypes); err != nil {
			r.logger.Warn("failed to unmarshal subscription event types", map[string]interface{}{"subscription_id": s.ID, "error": err.Error()})
		}
		if err := json.Unmarshal(deliveryJSON, &s.DeliveryConfig); err != nil {
			r.logger.Warn("failed to unmarshal subscription delivery config", map[string]interface{}{"subscription_id": s.ID, "error": err.Error()})
		}
		if len(filterJSON) > 0 {
			var f EventFilter
			if err := json.Unmarshal(filterJSON, &f); err != nil {
				r.logger.Warn("failed to unmarshal subscription filter", map[string]interface{}{"subscription_id": s.ID, "error": err.Error()})
			} else {
				s.Filter = &f
			}
		}

		subs = append(subs, s)
	}

	return subs, nil
}

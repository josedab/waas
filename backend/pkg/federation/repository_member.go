package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

func (r *PostgresRepository) SaveMember(ctx context.Context, member *FederationMember) error {
	endpointsJSON, err := json.Marshal(member.Endpoints)
	if err != nil {
		return fmt.Errorf("failed to marshal member endpoints: %w", err)
	}
	capsJSON, err := json.Marshal(member.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal member capabilities: %w", err)
	}
	metaJSON, err := json.Marshal(member.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal member metadata: %w", err)
	}

	query := `
		INSERT INTO federation_members (
			id, tenant_id, organization_id, name, domain, status, public_key,
			endpoints, capabilities, trust_level, metadata,
			joined_at, last_seen_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			public_key = EXCLUDED.public_key,
			endpoints = EXCLUDED.endpoints,
			capabilities = EXCLUDED.capabilities,
			trust_level = EXCLUDED.trust_level,
			metadata = EXCLUDED.metadata,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		member.ID, member.TenantID, member.OrganizationID, member.Name, member.Domain,
		member.Status, member.PublicKey, endpointsJSON, capsJSON, member.TrustLevel,
		metaJSON, member.JoinedAt, member.LastSeenAt, member.CreatedAt, member.UpdatedAt)

	return err
}

// GetMember retrieves a member
func (r *PostgresRepository) GetMember(ctx context.Context, memberID string) (*FederationMember, error) {
	query := `
		SELECT id, tenant_id, organization_id, name, domain, status, public_key,
			   endpoints, capabilities, trust_level, metadata,
			   joined_at, last_seen_at, created_at, updated_at
		FROM federation_members
		WHERE id = $1`

	var m FederationMember
	var endpointsJSON, capsJSON, metaJSON []byte

	err := r.db.QueryRowContext(ctx, query, memberID).Scan(
		&m.ID, &m.TenantID, &m.OrganizationID, &m.Name, &m.Domain, &m.Status,
		&m.PublicKey, &endpointsJSON, &capsJSON, &m.TrustLevel, &metaJSON,
		&m.JoinedAt, &m.LastSeenAt, &m.CreatedAt, &m.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(endpointsJSON, &m.Endpoints); err != nil {
		r.logger.Warn("failed to unmarshal member endpoints", map[string]interface{}{"member_id": memberID, "error": err.Error()})
	}
	if err := json.Unmarshal(capsJSON, &m.Capabilities); err != nil {
		r.logger.Warn("failed to unmarshal member capabilities", map[string]interface{}{"member_id": memberID, "error": err.Error()})
	}
	if err := json.Unmarshal(metaJSON, &m.Metadata); err != nil {
		r.logger.Warn("failed to unmarshal member metadata", map[string]interface{}{"member_id": memberID, "error": err.Error()})
	}

	return &m, nil
}

// GetMemberByDomain retrieves member by domain
func (r *PostgresRepository) GetMemberByDomain(ctx context.Context, domain string) (*FederationMember, error) {
	query := `SELECT id FROM federation_members WHERE domain = $1`
	var memberID string
	err := r.db.QueryRowContext(ctx, query, domain).Scan(&memberID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}
	return r.GetMember(ctx, memberID)
}

// ListMembers lists members
func (r *PostgresRepository) ListMembers(ctx context.Context, tenantID string, status *MemberStatus) ([]FederationMember, error) {
	query := `
		SELECT id, tenant_id, organization_id, name, domain, status, public_key,
			   endpoints, capabilities, trust_level, metadata,
			   joined_at, last_seen_at, created_at, updated_at
		FROM federation_members
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []FederationMember
	for rows.Next() {
		var m FederationMember
		var endpointsJSON, capsJSON, metaJSON []byte

		err := rows.Scan(
			&m.ID, &m.TenantID, &m.OrganizationID, &m.Name, &m.Domain, &m.Status,
			&m.PublicKey, &endpointsJSON, &capsJSON, &m.TrustLevel, &metaJSON,
			&m.JoinedAt, &m.LastSeenAt, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(endpointsJSON, &m.Endpoints); err != nil {
			r.logger.Warn("failed to unmarshal member endpoints", map[string]interface{}{"member_id": m.ID, "error": err.Error()})
		}
		if err := json.Unmarshal(capsJSON, &m.Capabilities); err != nil {
			r.logger.Warn("failed to unmarshal member capabilities", map[string]interface{}{"member_id": m.ID, "error": err.Error()})
		}
		if err := json.Unmarshal(metaJSON, &m.Metadata); err != nil {
			r.logger.Warn("failed to unmarshal member metadata", map[string]interface{}{"member_id": m.ID, "error": err.Error()})
		}

		members = append(members, m)
	}

	return members, nil
}

// DeleteMember deletes a member
func (r *PostgresRepository) DeleteMember(ctx context.Context, memberID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM federation_members WHERE id = $1", memberID)
	return err
}

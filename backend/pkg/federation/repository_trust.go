package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (r *PostgresRepository) SaveTrustRelationship(ctx context.Context, trust *TrustRelationship) error {
	permsJSON, err := json.Marshal(trust.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal trust permissions: %w", err)
	}

	query := `
		INSERT INTO federation_trust (
			id, tenant_id, source_member_id, target_member_id, status,
			trust_level, permissions, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			trust_level = EXCLUDED.trust_level,
			permissions = EXCLUDED.permissions,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		trust.ID, trust.TenantID, trust.SourceMemberID, trust.TargetMemberID,
		trust.Status, trust.TrustLevel, permsJSON, trust.ExpiresAt,
		trust.CreatedAt, trust.UpdatedAt)

	return err
}

// GetTrustRelationship retrieves a trust relationship
func (r *PostgresRepository) GetTrustRelationship(ctx context.Context, trustID string) (*TrustRelationship, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, status,
			   trust_level, permissions, expires_at, created_at, updated_at
		FROM federation_trust
		WHERE id = $1`

	var t TrustRelationship
	var permsJSON []byte
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, trustID).Scan(
		&t.ID, &t.TenantID, &t.SourceMemberID, &t.TargetMemberID, &t.Status,
		&t.TrustLevel, &permsJSON, &expiresAt, &t.CreatedAt, &t.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("trust relationship not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(permsJSON, &t.Permissions); err != nil {
		r.logger.Warn("failed to unmarshal trust permissions", map[string]interface{}{"trust_id": trustID, "error": err.Error()})
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}

	return &t, nil
}

// GetTrustBetween gets trust between two members
func (r *PostgresRepository) GetTrustBetween(ctx context.Context, sourceID, targetID string) (*TrustRelationship, error) {
	query := `
		SELECT id FROM federation_trust
		WHERE source_member_id = $1 AND target_member_id = $2`

	var trustID string
	err := r.db.QueryRowContext(ctx, query, sourceID, targetID).Scan(&trustID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("trust relationship not found")
	}
	if err != nil {
		return nil, err
	}

	return r.GetTrustRelationship(ctx, trustID)
}

// ListTrustRelationships lists trust relationships
func (r *PostgresRepository) ListTrustRelationships(ctx context.Context, tenantID, memberID string) ([]TrustRelationship, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, status,
			   trust_level, permissions, expires_at, created_at, updated_at
		FROM federation_trust
		WHERE tenant_id = $1 AND (source_member_id = $2 OR target_member_id = $2)
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trusts []TrustRelationship
	for rows.Next() {
		var t TrustRelationship
		var permsJSON []byte
		var expiresAt sql.NullTime

		err := rows.Scan(
			&t.ID, &t.TenantID, &t.SourceMemberID, &t.TargetMemberID, &t.Status,
			&t.TrustLevel, &permsJSON, &expiresAt, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(permsJSON, &t.Permissions); err != nil {
			r.logger.Warn("failed to unmarshal trust permissions", map[string]interface{}{"trust_id": t.ID, "error": err.Error()})
		}
		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Time
		}

		trusts = append(trusts, t)
	}

	return trusts, nil
}

// SaveTrustRequest saves a trust request
func (r *PostgresRepository) SaveTrustRequest(ctx context.Context, req *TrustRequest) error {
	permsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal trust request permissions: %w", err)
	}

	query := `
		INSERT INTO federation_trust_requests (
			id, tenant_id, requester_id, target_member_id, requested_level,
			permissions, message, status, expires_at, responded_at, response, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			responded_at = EXCLUDED.responded_at,
			response = EXCLUDED.response`

	_, err = r.db.ExecContext(ctx, query,
		req.ID, req.TenantID, req.RequesterID, req.TargetMemberID, req.RequestedLevel,
		permsJSON, req.Message, req.Status, req.ExpiresAt, req.RespondedAt,
		req.Response, req.CreatedAt)

	return err
}

// GetTrustRequest retrieves a trust request
func (r *PostgresRepository) GetTrustRequest(ctx context.Context, reqID string) (*TrustRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, target_member_id, requested_level,
			   permissions, message, status, expires_at, responded_at, response, created_at
		FROM federation_trust_requests
		WHERE id = $1`

	var req TrustRequest
	var permsJSON []byte
	var expiresAt, respondedAt sql.NullTime
	var response sql.NullString

	err := r.db.QueryRowContext(ctx, query, reqID).Scan(
		&req.ID, &req.TenantID, &req.RequesterID, &req.TargetMemberID,
		&req.RequestedLevel, &permsJSON, &req.Message, &req.Status,
		&expiresAt, &respondedAt, &response, &req.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("trust request not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(permsJSON, &req.Permissions); err != nil {
		r.logger.Warn("failed to unmarshal trust request permissions", map[string]interface{}{"request_id": reqID, "error": err.Error()})
	}
	if expiresAt.Valid {
		req.ExpiresAt = &expiresAt.Time
	}
	if respondedAt.Valid {
		req.RespondedAt = &respondedAt.Time
	}
	if response.Valid {
		req.Response = response.String
	}

	return &req, nil
}

// ListTrustRequests lists trust requests
func (r *PostgresRepository) ListTrustRequests(ctx context.Context, tenantID string, status *TrustReqStatus) ([]TrustRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, target_member_id, requested_level,
			   permissions, message, status, expires_at, responded_at, response, created_at
		FROM federation_trust_requests
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

	var requests []TrustRequest
	for rows.Next() {
		var req TrustRequest
		var permsJSON []byte
		var expiresAt, respondedAt sql.NullTime
		var response sql.NullString

		err := rows.Scan(
			&req.ID, &req.TenantID, &req.RequesterID, &req.TargetMemberID,
			&req.RequestedLevel, &permsJSON, &req.Message, &req.Status,
			&expiresAt, &respondedAt, &response, &req.CreatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(permsJSON, &req.Permissions); err != nil {
			r.logger.Warn("failed to unmarshal trust request permissions", map[string]interface{}{"request_id": req.ID, "error": err.Error()})
		}
		if expiresAt.Valid {
			req.ExpiresAt = &expiresAt.Time
		}
		if respondedAt.Valid {
			req.RespondedAt = &respondedAt.Time
		}
		if response.Valid {
			req.Response = response.String
		}

		requests = append(requests, req)
	}

	return requests, nil
}

// UpdateTrustRequestStatus updates trust request status
func (r *PostgresRepository) UpdateTrustRequestStatus(ctx context.Context, reqID string, status TrustReqStatus, response string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE federation_trust_requests SET status = $1, responded_at = $2, response = $3 WHERE id = $4",
		status, now, response, reqID)
	return err
}

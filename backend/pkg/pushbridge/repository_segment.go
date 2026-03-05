package pushbridge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SaveSegment saves a segment
func (r *PostgresRepository) SaveSegment(ctx context.Context, segment *DeviceSegment) error {
	query := `
		INSERT INTO push_segments (id, tenant_id, name, description, query, device_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			query = EXCLUDED.query,
			device_count = EXCLUDED.device_count,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		segment.ID, segment.TenantID, segment.Name, segment.Description,
		segment.Query, segment.DeviceCount, segment.CreatedAt, segment.UpdatedAt)
	return err
}

// GetSegment retrieves a segment
func (r *PostgresRepository) GetSegment(ctx context.Context, tenantID, segmentID string) (*DeviceSegment, error) {
	query := `
		SELECT id, tenant_id, name, description, query, device_count, created_at, updated_at
		FROM push_segments
		WHERE tenant_id = $1 AND id = $2`

	var segment DeviceSegment
	var description sql.NullString
	err := r.db.QueryRowContext(ctx, query, tenantID, segmentID).Scan(
		&segment.ID, &segment.TenantID, &segment.Name, &description,
		&segment.Query, &segment.DeviceCount, &segment.CreatedAt, &segment.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("segment not found")
	}
	if description.Valid {
		segment.Description = description.String
	}
	return &segment, err
}

// ListSegments lists segments
func (r *PostgresRepository) ListSegments(ctx context.Context, tenantID string) ([]DeviceSegment, error) {
	query := `
		SELECT id, tenant_id, name, description, query, device_count, created_at, updated_at
		FROM push_segments
		WHERE tenant_id = $1
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []DeviceSegment
	for rows.Next() {
		var segment DeviceSegment
		var description sql.NullString
		if err := rows.Scan(&segment.ID, &segment.TenantID, &segment.Name, &description, &segment.Query, &segment.DeviceCount, &segment.CreatedAt, &segment.UpdatedAt); err != nil {
			continue
		}
		if description.Valid {
			segment.Description = description.String
		}
		segments = append(segments, segment)
	}

	return segments, nil
}

// DeleteSegment deletes a segment
func (r *PostgresRepository) DeleteSegment(ctx context.Context, tenantID, segmentID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_segments WHERE tenant_id = $1 AND id = $2",
		tenantID, segmentID)
	return err
}

// GetDevicesInSegment gets devices matching a segment
func (r *PostgresRepository) GetDevicesInSegment(ctx context.Context, tenantID, segmentID string) ([]PushDevice, error) {
	// This would execute the segment query - simplified for now
	return r.ListDevices(ctx, tenantID, nil)
}

package pushbridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// SaveDevice saves a push device
func (r *PostgresRepository) SaveDevice(ctx context.Context, device *PushDevice) error {
	deviceInfoJSON, err := json.Marshal(device.DeviceInfo)
	if err != nil {
		return fmt.Errorf("marshal device info: %w", err)
	}
	prefsJSON, err := json.Marshal(device.Preferences)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}
	tagsJSON, err := json.Marshal(device.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	metadataJSON, err := json.Marshal(device.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO push_devices (
			id, tenant_id, user_id, platform, push_token,
			device_info, status, preferences, tags, metadata,
			last_active_at, registered_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			push_token = EXCLUDED.push_token,
			device_info = EXCLUDED.device_info,
			status = EXCLUDED.status,
			preferences = EXCLUDED.preferences,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			last_active_at = EXCLUDED.last_active_at,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		device.ID, device.TenantID, device.UserID, device.Platform, device.PushToken,
		deviceInfoJSON, device.Status, prefsJSON, tagsJSON, metadataJSON,
		device.LastActiveAt, device.RegisteredAt, device.UpdatedAt)

	return err
}

// GetDevice retrieves a device
func (r *PostgresRepository) GetDevice(ctx context.Context, tenantID, deviceID string) (*PushDevice, error) {
	query := `
		SELECT id, tenant_id, user_id, platform, push_token,
			   device_info, status, preferences, tags, metadata,
			   last_active_at, registered_at, updated_at
		FROM push_devices
		WHERE tenant_id = $1 AND id = $2`

	return r.scanDevice(r.db.QueryRowContext(ctx, query, tenantID, deviceID))
}

// GetDeviceByToken retrieves a device by push token
func (r *PostgresRepository) GetDeviceByToken(ctx context.Context, tenantID, pushToken string) (*PushDevice, error) {
	query := `
		SELECT id, tenant_id, user_id, platform, push_token,
			   device_info, status, preferences, tags, metadata,
			   last_active_at, registered_at, updated_at
		FROM push_devices
		WHERE tenant_id = $1 AND push_token = $2`

	return r.scanDevice(r.db.QueryRowContext(ctx, query, tenantID, pushToken))
}

func (r *PostgresRepository) scanDevice(row *sql.Row) (*PushDevice, error) {
	var device PushDevice
	var deviceInfoJSON, prefsJSON, tagsJSON, metadataJSON []byte
	var userID sql.NullString
	var lastActiveAt sql.NullTime

	err := row.Scan(
		&device.ID, &device.TenantID, &userID, &device.Platform, &device.PushToken,
		&deviceInfoJSON, &device.Status, &prefsJSON, &tagsJSON, &metadataJSON,
		&lastActiveAt, &device.RegisteredAt, &device.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(deviceInfoJSON, &device.DeviceInfo)
	json.Unmarshal(prefsJSON, &device.Preferences)
	json.Unmarshal(tagsJSON, &device.Tags)
	json.Unmarshal(metadataJSON, &device.Metadata)

	if userID.Valid {
		device.UserID = userID.String
	}
	if lastActiveAt.Valid {
		device.LastActiveAt = &lastActiveAt.Time
	}

	return &device, nil
}

// ListDevices lists devices
func (r *PostgresRepository) ListDevices(ctx context.Context, tenantID string, filter *DeviceFilter) ([]PushDevice, error) {
	query := `
		SELECT id, tenant_id, user_id, platform, push_token,
			   device_info, status, preferences, tags, metadata,
			   last_active_at, registered_at, updated_at
		FROM push_devices
		WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if filter != nil {
		if filter.Platform != nil {
			query += fmt.Sprintf(" AND platform = $%d", argIdx)
			args = append(args, *filter.Platform)
			argIdx++
		}
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.UserID != nil {
			query += fmt.Sprintf(" AND user_id = $%d", argIdx)
			args = append(args, *filter.UserID)
			argIdx++
		}
		if len(filter.Tags) > 0 {
			query += fmt.Sprintf(" AND tags ?| $%d", argIdx)
			args = append(args, filter.Tags)
			argIdx++
		}
	}

	query += " ORDER BY registered_at DESC"

	limit := 50
	if filter != nil && filter.Limit > 0 {
		limit = filter.Limit
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filter != nil && filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []PushDevice
	for rows.Next() {
		var device PushDevice
		var deviceInfoJSON, prefsJSON, tagsJSON, metadataJSON []byte
		var userID sql.NullString
		var lastActiveAt sql.NullTime

		err := rows.Scan(
			&device.ID, &device.TenantID, &userID, &device.Platform, &device.PushToken,
			&deviceInfoJSON, &device.Status, &prefsJSON, &tagsJSON, &metadataJSON,
			&lastActiveAt, &device.RegisteredAt, &device.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(deviceInfoJSON, &device.DeviceInfo)
		json.Unmarshal(prefsJSON, &device.Preferences)
		json.Unmarshal(tagsJSON, &device.Tags)
		json.Unmarshal(metadataJSON, &device.Metadata)

		if userID.Valid {
			device.UserID = userID.String
		}
		if lastActiveAt.Valid {
			device.LastActiveAt = &lastActiveAt.Time
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// DeleteDevice deletes a device
func (r *PostgresRepository) DeleteDevice(ctx context.Context, tenantID, deviceID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_devices WHERE tenant_id = $1 AND id = $2",
		tenantID, deviceID)
	return err
}

// UpdateDeviceStatus updates device status
func (r *PostgresRepository) UpdateDeviceStatus(ctx context.Context, deviceID string, status DeviceStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE push_devices SET status = $1, updated_at = NOW() WHERE id = $2",
		status, deviceID)
	return err
}

// GenerateDeviceID generates a new device ID
func GenerateDeviceID() string {
	return uuid.New().String()
}

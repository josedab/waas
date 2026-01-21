package pushbridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines push bridge data access
type Repository interface {
	// Devices
	SaveDevice(ctx context.Context, device *PushDevice) error
	GetDevice(ctx context.Context, tenantID, deviceID string) (*PushDevice, error)
	GetDeviceByToken(ctx context.Context, tenantID, pushToken string) (*PushDevice, error)
	ListDevices(ctx context.Context, tenantID string, filter *DeviceFilter) ([]PushDevice, error)
	DeleteDevice(ctx context.Context, tenantID, deviceID string) error
	UpdateDeviceStatus(ctx context.Context, deviceID string, status DeviceStatus) error
	
	// Mappings
	SaveMapping(ctx context.Context, mapping *PushMapping) error
	GetMapping(ctx context.Context, tenantID, mappingID string) (*PushMapping, error)
	ListMappings(ctx context.Context, tenantID string) ([]PushMapping, error)
	GetMappingByWebhook(ctx context.Context, tenantID, webhookID string) ([]PushMapping, error)
	DeleteMapping(ctx context.Context, tenantID, mappingID string) error
	
	// Notifications
	SaveNotification(ctx context.Context, notif *PushNotification) error
	GetNotification(ctx context.Context, notifID string) (*PushNotification, error)
	ListNotifications(ctx context.Context, tenantID string, filter *NotificationFilter) ([]PushNotification, error)
	UpdateNotificationStatus(ctx context.Context, notifID string, status DeliveryStatus, response *ProviderResponse) error
	
	// Offline Queue
	QueueNotification(ctx context.Context, queue *OfflineQueue) error
	GetQueuedNotifications(ctx context.Context, deviceID string, limit int) ([]OfflineQueue, error)
	DeleteQueuedNotification(ctx context.Context, queueID string) error
	CleanExpiredQueue(ctx context.Context) error
	
	// Providers
	SaveProvider(ctx context.Context, provider *PushProviderConfig) error
	GetProvider(ctx context.Context, tenantID string, providerType ProviderType) (*PushProviderConfig, error)
	ListProviders(ctx context.Context, tenantID string) ([]PushProviderConfig, error)
	DeleteProvider(ctx context.Context, tenantID string, providerType ProviderType) error
	
	// Credentials
	SaveCredentials(ctx context.Context, creds *ProviderCredentials) error
	GetCredentials(ctx context.Context, tenantID, credID string) (*ProviderCredentials, error)
	ListCredentials(ctx context.Context, tenantID string, provider *Platform) ([]*ProviderCredentials, error)
	UpdateCredentials(ctx context.Context, creds *ProviderCredentials) error
	DeleteCredentials(ctx context.Context, tenantID, credID string) error
	
	// Stats
	GetStats(ctx context.Context, tenantID string, from, to time.Time) (*PushStats, error)
	IncrementStats(ctx context.Context, tenantID string, platform Platform, status DeliveryStatus) error
	
	// Segments
	SaveSegment(ctx context.Context, segment *DeviceSegment) error
	GetSegment(ctx context.Context, tenantID, segmentID string) (*DeviceSegment, error)
	ListSegments(ctx context.Context, tenantID string) ([]DeviceSegment, error)
	DeleteSegment(ctx context.Context, tenantID, segmentID string) error
	GetDevicesInSegment(ctx context.Context, tenantID, segmentID string) ([]PushDevice, error)
}

// DeviceFilter defines device list filters
type DeviceFilter struct {
	Platform  *Platform
	Status    *DeviceStatus
	UserID    *string
	Tags      []string
	Limit     int
	Offset    int
}

// NotificationFilter defines notification list filters
type NotificationFilter struct {
	DeviceID  *string
	MappingID *string
	Status    *DeliveryStatus
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveDevice saves a push device
func (r *PostgresRepository) SaveDevice(ctx context.Context, device *PushDevice) error {
	deviceInfoJSON, _ := json.Marshal(device.DeviceInfo)
	prefsJSON, _ := json.Marshal(device.Preferences)
	tagsJSON, _ := json.Marshal(device.Tags)
	metadataJSON, _ := json.Marshal(device.Metadata)

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

	_, err := r.db.ExecContext(ctx, query,
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

// SaveNotification saves a notification
func (r *PostgresRepository) SaveNotification(ctx context.Context, notif *PushNotification) error {
	responseJSON, _ := json.Marshal(notif.Response)

	query := `
		INSERT INTO push_notifications (
			id, tenant_id, mapping_id, webhook_id, platform, device_id,
			push_token, status, payload, response, attempts,
			last_attempt, delivered_at, opened_at, error, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			response = EXCLUDED.response,
			attempts = EXCLUDED.attempts,
			last_attempt = EXCLUDED.last_attempt,
			delivered_at = EXCLUDED.delivered_at,
			opened_at = EXCLUDED.opened_at,
			error = EXCLUDED.error`

	_, err := r.db.ExecContext(ctx, query,
		notif.ID, notif.TenantID, notif.MappingID, notif.WebhookID,
		notif.Platform, notif.DeviceID, notif.PushToken, notif.Status,
		notif.Payload, responseJSON, notif.Attempts,
		notif.LastAttempt, notif.DeliveredAt, notif.OpenedAt, notif.Error, notif.CreatedAt)

	return err
}

// GetNotification retrieves a notification
func (r *PostgresRepository) GetNotification(ctx context.Context, notifID string) (*PushNotification, error) {
	query := `
		SELECT id, tenant_id, mapping_id, webhook_id, platform, device_id,
			   push_token, status, payload, response, attempts,
			   last_attempt, delivered_at, opened_at, error, created_at
		FROM push_notifications
		WHERE id = $1`

	var notif PushNotification
	var responseJSON []byte
	var mappingID, webhookID, errMsg sql.NullString
	var lastAttempt, deliveredAt, openedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, notifID).Scan(
		&notif.ID, &notif.TenantID, &mappingID, &webhookID,
		&notif.Platform, &notif.DeviceID, &notif.PushToken, &notif.Status,
		&notif.Payload, &responseJSON, &notif.Attempts,
		&lastAttempt, &deliveredAt, &openedAt, &errMsg, &notif.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(responseJSON, &notif.Response)

	if mappingID.Valid {
		notif.MappingID = mappingID.String
	}
	if webhookID.Valid {
		notif.WebhookID = webhookID.String
	}
	if errMsg.Valid {
		notif.Error = errMsg.String
	}
	if lastAttempt.Valid {
		notif.LastAttempt = &lastAttempt.Time
	}
	if deliveredAt.Valid {
		notif.DeliveredAt = &deliveredAt.Time
	}
	if openedAt.Valid {
		notif.OpenedAt = &openedAt.Time
	}

	return &notif, nil
}

// ListNotifications lists notifications
func (r *PostgresRepository) ListNotifications(ctx context.Context, tenantID string, filter *NotificationFilter) ([]PushNotification, error) {
	query := `
		SELECT id, tenant_id, mapping_id, webhook_id, platform, device_id,
			   push_token, status, payload, response, attempts,
			   last_attempt, delivered_at, opened_at, error, created_at
		FROM push_notifications
		WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if filter != nil {
		if filter.DeviceID != nil {
			query += fmt.Sprintf(" AND device_id = $%d", argIdx)
			args = append(args, *filter.DeviceID)
			argIdx++
		}
		if filter.MappingID != nil {
			query += fmt.Sprintf(" AND mapping_id = $%d", argIdx)
			args = append(args, *filter.MappingID)
			argIdx++
		}
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.From != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
			args = append(args, *filter.From)
			argIdx++
		}
		if filter.To != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
			args = append(args, *filter.To)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"

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

	var notifications []PushNotification
	for rows.Next() {
		var notif PushNotification
		var responseJSON []byte
		var mappingID, webhookID, errMsg sql.NullString
		var lastAttempt, deliveredAt, openedAt sql.NullTime

		err := rows.Scan(
			&notif.ID, &notif.TenantID, &mappingID, &webhookID,
			&notif.Platform, &notif.DeviceID, &notif.PushToken, &notif.Status,
			&notif.Payload, &responseJSON, &notif.Attempts,
			&lastAttempt, &deliveredAt, &openedAt, &errMsg, &notif.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(responseJSON, &notif.Response)

		if mappingID.Valid {
			notif.MappingID = mappingID.String
		}
		if webhookID.Valid {
			notif.WebhookID = webhookID.String
		}
		if errMsg.Valid {
			notif.Error = errMsg.String
		}
		if lastAttempt.Valid {
			notif.LastAttempt = &lastAttempt.Time
		}
		if deliveredAt.Valid {
			notif.DeliveredAt = &deliveredAt.Time
		}
		if openedAt.Valid {
			notif.OpenedAt = &openedAt.Time
		}

		notifications = append(notifications, notif)
	}

	return notifications, nil
}

// UpdateNotificationStatus updates notification status
func (r *PostgresRepository) UpdateNotificationStatus(ctx context.Context, notifID string, status DeliveryStatus, response *ProviderResponse) error {
	responseJSON, _ := json.Marshal(response)

	var deliveredAt, openedAt *time.Time
	now := time.Now()
	if status == DeliveryDelivered {
		deliveredAt = &now
	} else if status == DeliveryOpened {
		openedAt = &now
	}

	query := `
		UPDATE push_notifications SET
			status = $1, response = $2, last_attempt = NOW(),
			delivered_at = COALESCE($3, delivered_at),
			opened_at = COALESCE($4, opened_at),
			attempts = attempts + 1
		WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, status, responseJSON, deliveredAt, openedAt, notifID)
	return err
}

// QueueNotification queues a notification
func (r *PostgresRepository) QueueNotification(ctx context.Context, queue *OfflineQueue) error {
	query := `
		INSERT INTO push_offline_queue (id, tenant_id, device_id, notification, priority, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		queue.ID, queue.TenantID, queue.DeviceID, queue.Notification,
		queue.Priority, queue.ExpiresAt, queue.CreatedAt)
	return err
}

// GetQueuedNotifications gets queued notifications
func (r *PostgresRepository) GetQueuedNotifications(ctx context.Context, deviceID string, limit int) ([]OfflineQueue, error) {
	query := `
		SELECT id, tenant_id, device_id, notification, priority, expires_at, created_at
		FROM push_offline_queue
		WHERE device_id = $1 AND expires_at > NOW()
		ORDER BY priority DESC, created_at ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queued []OfflineQueue
	for rows.Next() {
		var q OfflineQueue
		if err := rows.Scan(&q.ID, &q.TenantID, &q.DeviceID, &q.Notification, &q.Priority, &q.ExpiresAt, &q.CreatedAt); err != nil {
			continue
		}
		queued = append(queued, q)
	}

	return queued, nil
}

// DeleteQueuedNotification deletes a queued notification
func (r *PostgresRepository) DeleteQueuedNotification(ctx context.Context, queueID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM push_offline_queue WHERE id = $1", queueID)
	return err
}

// CleanExpiredQueue removes expired queue entries
func (r *PostgresRepository) CleanExpiredQueue(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM push_offline_queue WHERE expires_at <= NOW()")
	return err
}

// SaveProvider saves a provider config
func (r *PostgresRepository) SaveProvider(ctx context.Context, provider *PushProviderConfig) error {
	query := `
		INSERT INTO push_providers (id, tenant_id, provider, name, enabled, config, credentials, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, provider) DO UPDATE SET
			name = EXCLUDED.name,
			enabled = EXCLUDED.enabled,
			config = EXCLUDED.config,
			credentials = EXCLUDED.credentials,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.TenantID, provider.Provider, provider.Name,
		provider.Enabled, provider.Config, provider.Credentials,
		provider.CreatedAt, provider.UpdatedAt)
	return err
}

// GetProvider retrieves a provider
func (r *PostgresRepository) GetProvider(ctx context.Context, tenantID string, providerType ProviderType) (*PushProviderConfig, error) {
	query := `
		SELECT id, tenant_id, provider, name, enabled, config, credentials, created_at, updated_at
		FROM push_providers
		WHERE tenant_id = $1 AND provider = $2`

	var provider PushProviderConfig
	err := r.db.QueryRowContext(ctx, query, tenantID, providerType).Scan(
		&provider.ID, &provider.TenantID, &provider.Provider, &provider.Name,
		&provider.Enabled, &provider.Config, &provider.Credentials,
		&provider.CreatedAt, &provider.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("provider not found")
	}
	return &provider, err
}

// ListProviders lists providers
func (r *PostgresRepository) ListProviders(ctx context.Context, tenantID string) ([]PushProviderConfig, error) {
	query := `
		SELECT id, tenant_id, provider, name, enabled, config, created_at, updated_at
		FROM push_providers
		WHERE tenant_id = $1`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []PushProviderConfig
	for rows.Next() {
		var p PushProviderConfig
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Provider, &p.Name, &p.Enabled, &p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// DeleteProvider deletes a provider
func (r *PostgresRepository) DeleteProvider(ctx context.Context, tenantID string, providerType ProviderType) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_providers WHERE tenant_id = $1 AND provider = $2",
		tenantID, providerType)
	return err
}

// GetStats retrieves push statistics
func (r *PostgresRepository) GetStats(ctx context.Context, tenantID string, from, to time.Time) (*PushStats, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN status IN ('sent', 'delivered', 'opened') THEN 1 ELSE 0 END), 0) as sent,
			COALESCE(SUM(CASE WHEN status IN ('delivered', 'opened') THEN 1 ELSE 0 END), 0) as delivered,
			COALESCE(SUM(CASE WHEN status = 'opened' THEN 1 ELSE 0 END), 0) as opened,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status = 'dropped' THEN 1 ELSE 0 END), 0) as dropped
		FROM push_notifications
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3`

	stats := &PushStats{TenantID: tenantID}
	err := r.db.QueryRowContext(ctx, query, tenantID, from, to).Scan(
		&stats.TotalSent, &stats.TotalDelivered, &stats.TotalOpened,
		&stats.TotalFailed, &stats.TotalDropped)
	if err != nil {
		return nil, err
	}

	if stats.TotalSent > 0 {
		stats.DeliveryRate = float64(stats.TotalDelivered) / float64(stats.TotalSent) * 100
		stats.OpenRate = float64(stats.TotalOpened) / float64(stats.TotalDelivered) * 100
	}

	return stats, nil
}

// IncrementStats increments statistics
func (r *PostgresRepository) IncrementStats(ctx context.Context, tenantID string, platform Platform, status DeliveryStatus) error {
	// Stats are calculated from notifications table, no separate stats table needed
	return nil
}

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

	if err == sql.ErrNoRows {
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

// GenerateDeviceID generates a new device ID
func GenerateDeviceID() string {
	return uuid.New().String()
}

// GenerateMappingID generates a new mapping ID
func GenerateMappingID() string {
	return uuid.New().String()
}

// GenerateNotificationID generates a new notification ID
func GenerateNotificationID() string {
	return uuid.New().String()
}

// SaveCredentials saves provider credentials
func (r *PostgresRepository) SaveCredentials(ctx context.Context, creds *ProviderCredentials) error {
	if creds.ID == "" {
		creds.ID = uuid.New().String()
	}
	
	credsJSON, _ := json.Marshal(creds.Credentials)
	
	query := `
		INSERT INTO push_provider_credentials (
			id, tenant_id, provider, name, credentials, environment,
			is_default, status, last_used_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.ExecContext(ctx, query,
		creds.ID, creds.TenantID, creds.Provider, creds.Name, credsJSON,
		creds.Environment, creds.IsDefault, creds.Status, creds.LastUsedAt,
		creds.CreatedAt, creds.UpdatedAt)
	return err
}

// GetCredentials retrieves provider credentials by ID
func (r *PostgresRepository) GetCredentials(ctx context.Context, tenantID, credID string) (*ProviderCredentials, error) {
	query := `
		SELECT id, tenant_id, provider, name, credentials, environment,
			   is_default, status, last_used_at, created_at, updated_at
		FROM push_provider_credentials
		WHERE tenant_id = $1 AND id = $2`

	var creds ProviderCredentials
	var credsJSON []byte
	var lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, credID).Scan(
		&creds.ID, &creds.TenantID, &creds.Provider, &creds.Name, &credsJSON,
		&creds.Environment, &creds.IsDefault, &creds.Status, &lastUsedAt,
		&creds.CreatedAt, &creds.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credentials not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(credsJSON, &creds.Credentials)
	if lastUsedAt.Valid {
		creds.LastUsedAt = &lastUsedAt.Time
	}

	return &creds, nil
}

// ListCredentials lists provider credentials
func (r *PostgresRepository) ListCredentials(ctx context.Context, tenantID string, provider *Platform) ([]*ProviderCredentials, error) {
	query := `
		SELECT id, tenant_id, provider, name, credentials, environment,
			   is_default, status, last_used_at, created_at, updated_at
		FROM push_provider_credentials
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if provider != nil {
		query += " AND provider = $2"
		args = append(args, *provider)
	}

	query += " ORDER BY is_default DESC, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []*ProviderCredentials
	for rows.Next() {
		var creds ProviderCredentials
		var credsJSON []byte
		var lastUsedAt sql.NullTime

		err := rows.Scan(
			&creds.ID, &creds.TenantID, &creds.Provider, &creds.Name, &credsJSON,
			&creds.Environment, &creds.IsDefault, &creds.Status, &lastUsedAt,
			&creds.CreatedAt, &creds.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(credsJSON, &creds.Credentials)
		if lastUsedAt.Valid {
			creds.LastUsedAt = &lastUsedAt.Time
		}

		credentials = append(credentials, &creds)
	}

	return credentials, nil
}

// UpdateCredentials updates provider credentials
func (r *PostgresRepository) UpdateCredentials(ctx context.Context, creds *ProviderCredentials) error {
	credsJSON, _ := json.Marshal(creds.Credentials)

	query := `
		UPDATE push_provider_credentials SET
			name = $1, credentials = $2, environment = $3,
			is_default = $4, status = $5, last_used_at = $6, updated_at = $7
		WHERE tenant_id = $8 AND id = $9`

	_, err := r.db.ExecContext(ctx, query,
		creds.Name, credsJSON, creds.Environment,
		creds.IsDefault, creds.Status, creds.LastUsedAt, creds.UpdatedAt,
		creds.TenantID, creds.ID)
	return err
}

// DeleteCredentials deletes provider credentials
func (r *PostgresRepository) DeleteCredentials(ctx context.Context, tenantID, credID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM push_provider_credentials WHERE tenant_id = $1 AND id = $2",
		tenantID, credID)
	return err
}

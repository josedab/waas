package versioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines versioning data access
type Repository interface {
	// Versions
	SaveVersion(ctx context.Context, version *Version) error
	GetVersion(ctx context.Context, tenantID, versionID string) (*Version, error)
	GetVersionByLabel(ctx context.Context, tenantID, webhookID, label string) (*Version, error)
	ListVersions(ctx context.Context, tenantID, webhookID string) ([]Version, error)
	GetLatestVersion(ctx context.Context, tenantID, webhookID string) (*Version, error)
	DeleteVersion(ctx context.Context, tenantID, versionID string) error

	// Schemas
	SaveSchema(ctx context.Context, schema *VersionSchema) error
	GetSchema(ctx context.Context, tenantID, schemaID string) (*VersionSchema, error)
	ListSchemas(ctx context.Context, tenantID string) ([]VersionSchema, error)

	// Subscriptions
	SaveSubscription(ctx context.Context, sub *VersionSubscription) error
	GetSubscription(ctx context.Context, tenantID, subID string) (*VersionSubscription, error)
	ListSubscriptions(ctx context.Context, tenantID, versionID string) ([]VersionSubscription, error)
	GetEndpointSubscription(ctx context.Context, tenantID, endpointID, webhookID string) (*VersionSubscription, error)

	// Migrations
	SaveMigration(ctx context.Context, mig *Migration) error
	GetMigration(ctx context.Context, migID string) (*Migration, error)
	ListMigrations(ctx context.Context, tenantID, webhookID string) ([]Migration, error)
	UpdateMigrationProgress(ctx context.Context, migID string, progress MigrationProgress, status MigrationStatus) error

	// Notices
	SaveNotice(ctx context.Context, notice *DeprecationNotice) error
	ListNotices(ctx context.Context, tenantID, versionID string) ([]DeprecationNotice, error)
	AckNotice(ctx context.Context, noticeID, response string) error

	// Policy
	SavePolicy(ctx context.Context, policy *VersionPolicy) error
	GetPolicy(ctx context.Context, tenantID string) (*VersionPolicy, error)

	// Metrics
	RecordVersionUsage(ctx context.Context, tenantID, versionID, endpointID string) error
	GetVersionMetrics(ctx context.Context, tenantID, versionID string) (*VersionMetrics, error)
}

// PostgresRepository implements Repository
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveVersion saves a version
func (r *PostgresRepository) SaveVersion(ctx context.Context, version *Version) error {
	transformsJSON, _ := json.Marshal(version.Transforms)
	compatibleJSON, _ := json.Marshal(version.CompatibleWith)
	policyJSON, _ := json.Marshal(version.SunsetPolicy)

	query := `
		INSERT INTO webhook_versions (
			id, tenant_id, webhook_id, major, minor, patch, label,
			schema_id, status, changelog, breaking, deprecated_at, sunset_at,
			sunset_policy, replacement, compatible_with, transforms,
			created_at, updated_at, published_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			changelog = EXCLUDED.changelog,
			deprecated_at = EXCLUDED.deprecated_at,
			sunset_at = EXCLUDED.sunset_at,
			sunset_policy = EXCLUDED.sunset_policy,
			replacement = EXCLUDED.replacement,
			compatible_with = EXCLUDED.compatible_with,
			transforms = EXCLUDED.transforms,
			updated_at = EXCLUDED.updated_at,
			published_at = EXCLUDED.published_at`

	_, err := r.db.ExecContext(ctx, query,
		version.ID, version.TenantID, version.WebhookID,
		version.Major, version.Minor, version.Patch, version.Label,
		version.SchemaID, version.Status, version.Changelog, version.Breaking,
		version.DeprecatedAt, version.SunsetAt, policyJSON, version.Replacement,
		compatibleJSON, transformsJSON, version.CreatedAt, version.UpdatedAt,
		version.PublishedAt)

	return err
}

// GetVersion retrieves a version
func (r *PostgresRepository) GetVersion(ctx context.Context, tenantID, versionID string) (*Version, error) {
	query := `
		SELECT id, tenant_id, webhook_id, major, minor, patch, label,
			   schema_id, status, changelog, breaking, deprecated_at, sunset_at,
			   sunset_policy, replacement, compatible_with, transforms,
			   created_at, updated_at, published_at
		FROM webhook_versions
		WHERE tenant_id = $1 AND id = $2`

	var v Version
	var transformsJSON, compatibleJSON, policyJSON []byte
	var deprecatedAt, sunsetAt, publishedAt sql.NullTime
	var replacement sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, versionID).Scan(
		&v.ID, &v.TenantID, &v.WebhookID, &v.Major, &v.Minor, &v.Patch, &v.Label,
		&v.SchemaID, &v.Status, &v.Changelog, &v.Breaking, &deprecatedAt, &sunsetAt,
		&policyJSON, &replacement, &compatibleJSON, &transformsJSON,
		&v.CreatedAt, &v.UpdatedAt, &publishedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("version not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(transformsJSON, &v.Transforms)
	json.Unmarshal(compatibleJSON, &v.CompatibleWith)
	if len(policyJSON) > 0 {
		var policy SunsetPolicy
		json.Unmarshal(policyJSON, &policy)
		v.SunsetPolicy = &policy
	}
	if deprecatedAt.Valid {
		v.DeprecatedAt = &deprecatedAt.Time
	}
	if sunsetAt.Valid {
		v.SunsetAt = &sunsetAt.Time
	}
	if publishedAt.Valid {
		v.PublishedAt = &publishedAt.Time
	}
	if replacement.Valid {
		v.Replacement = replacement.String
	}

	return &v, nil
}

// GetVersionByLabel retrieves version by label
func (r *PostgresRepository) GetVersionByLabel(ctx context.Context, tenantID, webhookID, label string) (*Version, error) {
	query := `
		SELECT id FROM webhook_versions
		WHERE tenant_id = $1 AND webhook_id = $2 AND label = $3`

	var versionID string
	err := r.db.QueryRowContext(ctx, query, tenantID, webhookID, label).Scan(&versionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("version not found")
	}
	if err != nil {
		return nil, err
	}

	return r.GetVersion(ctx, tenantID, versionID)
}

// ListVersions lists versions for a webhook
func (r *PostgresRepository) ListVersions(ctx context.Context, tenantID, webhookID string) ([]Version, error) {
	query := `
		SELECT id, tenant_id, webhook_id, major, minor, patch, label,
			   schema_id, status, changelog, breaking, deprecated_at, sunset_at,
			   sunset_policy, replacement, compatible_with, transforms,
			   created_at, updated_at, published_at
		FROM webhook_versions
		WHERE tenant_id = $1 AND webhook_id = $2
		ORDER BY major DESC, minor DESC, patch DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, webhookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		var v Version
		var transformsJSON, compatibleJSON, policyJSON []byte
		var deprecatedAt, sunsetAt, publishedAt sql.NullTime
		var replacement sql.NullString

		err := rows.Scan(
			&v.ID, &v.TenantID, &v.WebhookID, &v.Major, &v.Minor, &v.Patch, &v.Label,
			&v.SchemaID, &v.Status, &v.Changelog, &v.Breaking, &deprecatedAt, &sunsetAt,
			&policyJSON, &replacement, &compatibleJSON, &transformsJSON,
			&v.CreatedAt, &v.UpdatedAt, &publishedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(transformsJSON, &v.Transforms)
		json.Unmarshal(compatibleJSON, &v.CompatibleWith)
		if len(policyJSON) > 0 {
			var policy SunsetPolicy
			json.Unmarshal(policyJSON, &policy)
			v.SunsetPolicy = &policy
		}
		if deprecatedAt.Valid {
			v.DeprecatedAt = &deprecatedAt.Time
		}
		if sunsetAt.Valid {
			v.SunsetAt = &sunsetAt.Time
		}
		if publishedAt.Valid {
			v.PublishedAt = &publishedAt.Time
		}
		if replacement.Valid {
			v.Replacement = replacement.String
		}

		versions = append(versions, v)
	}

	return versions, nil
}

// GetLatestVersion gets latest published version
func (r *PostgresRepository) GetLatestVersion(ctx context.Context, tenantID, webhookID string) (*Version, error) {
	query := `
		SELECT id FROM webhook_versions
		WHERE tenant_id = $1 AND webhook_id = $2 AND status = 'published'
		ORDER BY major DESC, minor DESC, patch DESC
		LIMIT 1`

	var versionID string
	err := r.db.QueryRowContext(ctx, query, tenantID, webhookID).Scan(&versionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("no published version found")
	}
	if err != nil {
		return nil, err
	}

	return r.GetVersion(ctx, tenantID, versionID)
}

// DeleteVersion deletes a version
func (r *PostgresRepository) DeleteVersion(ctx context.Context, tenantID, versionID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM webhook_versions WHERE tenant_id = $1 AND id = $2",
		tenantID, versionID)
	return err
}

// SaveSchema saves a schema
func (r *PostgresRepository) SaveSchema(ctx context.Context, schema *VersionSchema) error {
	definitionJSON, _ := json.Marshal(schema.Definition)
	examplesJSON, _ := json.Marshal(schema.Examples)

	query := `
		INSERT INTO version_schemas (
			id, tenant_id, name, description, format, definition, examples,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			definition = EXCLUDED.definition,
			examples = EXCLUDED.examples,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		schema.ID, schema.TenantID, schema.Name, schema.Description,
		schema.Format, definitionJSON, examplesJSON,
		schema.CreatedAt, schema.UpdatedAt)

	return err
}

// GetSchema retrieves a schema
func (r *PostgresRepository) GetSchema(ctx context.Context, tenantID, schemaID string) (*VersionSchema, error) {
	query := `
		SELECT id, tenant_id, name, description, format, definition, examples,
			   created_at, updated_at
		FROM version_schemas
		WHERE tenant_id = $1 AND id = $2`

	var s VersionSchema
	var definitionJSON, examplesJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, schemaID).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Format,
		&definitionJSON, &examplesJSON, &s.CreatedAt, &s.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("schema not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(definitionJSON, &s.Definition)
	json.Unmarshal(examplesJSON, &s.Examples)

	return &s, nil
}

// ListSchemas lists schemas
func (r *PostgresRepository) ListSchemas(ctx context.Context, tenantID string) ([]VersionSchema, error) {
	query := `
		SELECT id, tenant_id, name, description, format, definition, examples,
			   created_at, updated_at
		FROM version_schemas
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []VersionSchema
	for rows.Next() {
		var s VersionSchema
		var definitionJSON, examplesJSON []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Format,
			&definitionJSON, &examplesJSON, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(definitionJSON, &s.Definition)
		json.Unmarshal(examplesJSON, &s.Examples)

		schemas = append(schemas, s)
	}

	return schemas, nil
}

// SaveSubscription saves a subscription
func (r *PostgresRepository) SaveSubscription(ctx context.Context, sub *VersionSubscription) error {
	query := `
		INSERT INTO version_subscriptions (
			id, tenant_id, endpoint_id, version_id, webhook_id, status, pinned,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			version_id = EXCLUDED.version_id,
			status = EXCLUDED.status,
			pinned = EXCLUDED.pinned,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.EndpointID, sub.VersionID, sub.WebhookID,
		sub.Status, sub.Pinned, sub.CreatedAt, sub.UpdatedAt)

	return err
}

// GetSubscription retrieves a subscription
func (r *PostgresRepository) GetSubscription(ctx context.Context, tenantID, subID string) (*VersionSubscription, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, version_id, webhook_id, status, pinned,
			   created_at, updated_at
		FROM version_subscriptions
		WHERE tenant_id = $1 AND id = $2`

	var s VersionSubscription
	err := r.db.QueryRowContext(ctx, query, tenantID, subID).Scan(
		&s.ID, &s.TenantID, &s.EndpointID, &s.VersionID, &s.WebhookID,
		&s.Status, &s.Pinned, &s.CreatedAt, &s.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// ListSubscriptions lists subscriptions for a version
func (r *PostgresRepository) ListSubscriptions(ctx context.Context, tenantID, versionID string) ([]VersionSubscription, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, version_id, webhook_id, status, pinned,
			   created_at, updated_at
		FROM version_subscriptions
		WHERE tenant_id = $1 AND version_id = $2`

	rows, err := r.db.QueryContext(ctx, query, tenantID, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []VersionSubscription
	for rows.Next() {
		var s VersionSubscription
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.EndpointID, &s.VersionID, &s.WebhookID,
			&s.Status, &s.Pinned, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			continue
		}
		subs = append(subs, s)
	}

	return subs, nil
}

// GetEndpointSubscription gets subscription for an endpoint
func (r *PostgresRepository) GetEndpointSubscription(ctx context.Context, tenantID, endpointID, webhookID string) (*VersionSubscription, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, version_id, webhook_id, status, pinned,
			   created_at, updated_at
		FROM version_subscriptions
		WHERE tenant_id = $1 AND endpoint_id = $2 AND webhook_id = $3`

	var s VersionSubscription
	err := r.db.QueryRowContext(ctx, query, tenantID, endpointID, webhookID).Scan(
		&s.ID, &s.TenantID, &s.EndpointID, &s.VersionID, &s.WebhookID,
		&s.Status, &s.Pinned, &s.CreatedAt, &s.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// SaveMigration saves a migration
func (r *PostgresRepository) SaveMigration(ctx context.Context, mig *Migration) error {
	progressJSON, _ := json.Marshal(mig.Progress)

	query := `
		INSERT INTO version_migrations (
			id, tenant_id, webhook_id, from_version, to_version, status,
			strategy, progress, started_at, completed_at, error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			progress = EXCLUDED.progress,
			completed_at = EXCLUDED.completed_at,
			error = EXCLUDED.error`

	_, err := r.db.ExecContext(ctx, query,
		mig.ID, mig.TenantID, mig.WebhookID, mig.FromVersion, mig.ToVersion,
		mig.Status, mig.Strategy, progressJSON, mig.StartedAt,
		mig.CompletedAt, mig.Error)

	return err
}

// GetMigration retrieves a migration
func (r *PostgresRepository) GetMigration(ctx context.Context, migID string) (*Migration, error) {
	query := `
		SELECT id, tenant_id, webhook_id, from_version, to_version, status,
			   strategy, progress, started_at, completed_at, error
		FROM version_migrations
		WHERE id = $1`

	var m Migration
	var progressJSON []byte
	var completedAt sql.NullTime
	var errMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, migID).Scan(
		&m.ID, &m.TenantID, &m.WebhookID, &m.FromVersion, &m.ToVersion,
		&m.Status, &m.Strategy, &progressJSON, &m.StartedAt, &completedAt, &errMsg)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("migration not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(progressJSON, &m.Progress)
	if completedAt.Valid {
		m.CompletedAt = &completedAt.Time
	}
	if errMsg.Valid {
		m.Error = errMsg.String
	}

	return &m, nil
}

// ListMigrations lists migrations
func (r *PostgresRepository) ListMigrations(ctx context.Context, tenantID, webhookID string) ([]Migration, error) {
	query := `
		SELECT id, tenant_id, webhook_id, from_version, to_version, status,
			   strategy, progress, started_at, completed_at, error
		FROM version_migrations
		WHERE tenant_id = $1 AND webhook_id = $2
		ORDER BY started_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, webhookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migs []Migration
	for rows.Next() {
		var m Migration
		var progressJSON []byte
		var completedAt sql.NullTime
		var errMsg sql.NullString

		err := rows.Scan(
			&m.ID, &m.TenantID, &m.WebhookID, &m.FromVersion, &m.ToVersion,
			&m.Status, &m.Strategy, &progressJSON, &m.StartedAt, &completedAt, &errMsg)
		if err != nil {
			continue
		}

		json.Unmarshal(progressJSON, &m.Progress)
		if completedAt.Valid {
			m.CompletedAt = &completedAt.Time
		}
		if errMsg.Valid {
			m.Error = errMsg.String
		}

		migs = append(migs, m)
	}

	return migs, nil
}

// UpdateMigrationProgress updates migration progress
func (r *PostgresRepository) UpdateMigrationProgress(ctx context.Context, migID string, progress MigrationProgress, status MigrationStatus) error {
	progressJSON, _ := json.Marshal(progress)

	var completedAt *time.Time
	if status == MigStatusCompleted || status == MigStatusFailed {
		now := time.Now()
		completedAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		"UPDATE version_migrations SET progress = $1, status = $2, completed_at = $3 WHERE id = $4",
		progressJSON, status, completedAt, migID)
	return err
}

// SaveNotice saves a notice
func (r *PostgresRepository) SaveNotice(ctx context.Context, notice *DeprecationNotice) error {
	query := `
		INSERT INTO deprecation_notices (
			id, tenant_id, version_id, endpoint_id, type, message,
			sent_at, acked_at, response
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			acked_at = EXCLUDED.acked_at,
			response = EXCLUDED.response`

	_, err := r.db.ExecContext(ctx, query,
		notice.ID, notice.TenantID, notice.VersionID, notice.EndpointID,
		notice.Type, notice.Message, notice.SentAt, notice.AckedAt, notice.Response)

	return err
}

// ListNotices lists notices
func (r *PostgresRepository) ListNotices(ctx context.Context, tenantID, versionID string) ([]DeprecationNotice, error) {
	query := `
		SELECT id, tenant_id, version_id, endpoint_id, type, message,
			   sent_at, acked_at, response
		FROM deprecation_notices
		WHERE tenant_id = $1 AND version_id = $2
		ORDER BY sent_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notices []DeprecationNotice
	for rows.Next() {
		var n DeprecationNotice
		var ackedAt sql.NullTime
		var response sql.NullString

		err := rows.Scan(
			&n.ID, &n.TenantID, &n.VersionID, &n.EndpointID, &n.Type,
			&n.Message, &n.SentAt, &ackedAt, &response)
		if err != nil {
			continue
		}

		if ackedAt.Valid {
			n.AckedAt = &ackedAt.Time
		}
		if response.Valid {
			n.Response = response.String
		}

		notices = append(notices, n)
	}

	return notices, nil
}

// AckNotice acknowledges a notice
func (r *PostgresRepository) AckNotice(ctx context.Context, noticeID, response string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE deprecation_notices SET acked_at = $1, response = $2 WHERE id = $3",
		now, response, noticeID)
	return err
}

// SavePolicy saves a policy
func (r *PostgresRepository) SavePolicy(ctx context.Context, policy *VersionPolicy) error {
	channelsJSON, _ := json.Marshal(policy.NotificationChannels)

	query := `
		INSERT INTO version_policies (
			id, tenant_id, enabled, default_version, require_version_header,
			allow_deprecated, auto_upgrade, deprecation_days, sunset_days,
			notification_channels, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			default_version = EXCLUDED.default_version,
			require_version_header = EXCLUDED.require_version_header,
			allow_deprecated = EXCLUDED.allow_deprecated,
			auto_upgrade = EXCLUDED.auto_upgrade,
			deprecation_days = EXCLUDED.deprecation_days,
			sunset_days = EXCLUDED.sunset_days,
			notification_channels = EXCLUDED.notification_channels,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		policy.ID, policy.TenantID, policy.Enabled, policy.DefaultVersion,
		policy.RequireVersionHeader, policy.AllowDeprecated, policy.AutoUpgrade,
		policy.DeprecationDays, policy.SunsetDays, channelsJSON,
		policy.CreatedAt, policy.UpdatedAt)

	return err
}

// GetPolicy retrieves a policy
func (r *PostgresRepository) GetPolicy(ctx context.Context, tenantID string) (*VersionPolicy, error) {
	query := `
		SELECT id, tenant_id, enabled, default_version, require_version_header,
			   allow_deprecated, auto_upgrade, deprecation_days, sunset_days,
			   notification_channels, created_at, updated_at
		FROM version_policies
		WHERE tenant_id = $1`

	var p VersionPolicy
	var channelsJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Enabled, &p.DefaultVersion, &p.RequireVersionHeader,
		&p.AllowDeprecated, &p.AutoUpgrade, &p.DeprecationDays, &p.SunsetDays,
		&channelsJSON, &p.CreatedAt, &p.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("policy not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(channelsJSON, &p.NotificationChannels)

	return &p, nil
}

// RecordVersionUsage records version usage
func (r *PostgresRepository) RecordVersionUsage(ctx context.Context, tenantID, versionID, endpointID string) error {
	query := `
		INSERT INTO version_usage (
			id, tenant_id, version_id, endpoint_id, recorded_at
		) VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, query,
		uuid.New().String(), tenantID, versionID, endpointID, time.Now())

	return err
}

// GetVersionMetrics gets version metrics
func (r *PostgresRepository) GetVersionMetrics(ctx context.Context, tenantID, versionID string) (*VersionMetrics, error) {
	metrics := &VersionMetrics{
		VersionID: versionID,
		TenantID:  tenantID,
	}

	// Get totals
	query := `
		SELECT COUNT(*), COUNT(DISTINCT endpoint_id), MAX(recorded_at)
		FROM version_usage
		WHERE tenant_id = $1 AND version_id = $2`

	err := r.db.QueryRowContext(ctx, query, tenantID, versionID).Scan(
		&metrics.TotalRequests, &metrics.UniqueEndpoints, &metrics.LastUsedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Get daily breakdown
	dailyQuery := `
		SELECT DATE(recorded_at)::text, COUNT(*)
		FROM version_usage
		WHERE tenant_id = $1 AND version_id = $2
			AND recorded_at > NOW() - INTERVAL '30 days'
		GROUP BY DATE(recorded_at)
		ORDER BY DATE(recorded_at)`

	rows, err := r.db.QueryContext(ctx, dailyQuery, tenantID, versionID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dc DayCount
			if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
				continue
			}
			metrics.RequestsByDay = append(metrics.RequestsByDay, dc)
		}
	}

	return metrics, nil
}

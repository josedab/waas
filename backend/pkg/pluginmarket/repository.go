package pluginmarket

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines data access for the plugin marketplace
type Repository interface {
	CreatePlugin(ctx context.Context, plugin *Plugin) error
	GetPlugin(ctx context.Context, id string) (*Plugin, error)
	GetPluginBySlug(ctx context.Context, slug string) (*Plugin, error)
	UpdatePlugin(ctx context.Context, plugin *Plugin) error
	SearchPlugins(ctx context.Context, params *PluginSearchParams) (*PluginSearchResult, error)
	DeletePlugin(ctx context.Context, id string) error

	CreateVersion(ctx context.Context, version *PluginVersion) error
	GetVersions(ctx context.Context, pluginID string) ([]PluginVersion, error)
	GetLatestVersion(ctx context.Context, pluginID string) (*PluginVersion, error)

	InstallPlugin(ctx context.Context, install *PluginInstallation) error
	UninstallPlugin(ctx context.Context, tenantID, pluginID string) error
	GetInstallation(ctx context.Context, tenantID, pluginID string) (*PluginInstallation, error)
	ListInstallations(ctx context.Context, tenantID string) ([]PluginInstallation, error)

	CreateReview(ctx context.Context, review *PluginReview) error
	GetReviews(ctx context.Context, pluginID string, page, pageSize int) ([]PluginReview, int, error)

	GetPluginHooks(ctx context.Context, pluginID string) ([]PluginHook, error)
	CreatePluginHook(ctx context.Context, hook *PluginHook) error

	GetMarketplaceStats(ctx context.Context) (*PluginStats, error)
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreatePlugin(ctx context.Context, plugin *Plugin) error {
	if plugin.ID == "" {
		plugin.ID = uuid.New().String()
	}
	plugin.Slug = generateSlug(plugin.Name)
	now := time.Now()
	plugin.CreatedAt = now
	plugin.UpdatedAt = now

	query := `INSERT INTO marketplace_plugins (id, name, slug, description, long_description, author_id, author_name,
		type, status, pricing, price_monthly, version, icon_url, source_url, documentation_url,
		installs, avg_rating, review_count, verified, featured, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`

	_, err := r.db.ExecContext(ctx, query,
		plugin.ID, plugin.Name, plugin.Slug, plugin.Description, plugin.LongDesc,
		plugin.AuthorID, plugin.AuthorName, plugin.Type, plugin.Status, plugin.Pricing,
		plugin.PriceMonthly, plugin.Version, plugin.IconURL, plugin.SourceURL, plugin.DocURL,
		plugin.Installs, plugin.AvgRating, plugin.ReviewCount, plugin.Verified, plugin.Featured,
		plugin.CreatedAt, plugin.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetPlugin(ctx context.Context, id string) (*Plugin, error) {
	var plugin Plugin
	err := r.db.GetContext(ctx, &plugin,
		`SELECT * FROM marketplace_plugins WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found: %s", id)
	}
	return &plugin, err
}

func (r *PostgresRepository) GetPluginBySlug(ctx context.Context, slug string) (*Plugin, error) {
	var plugin Plugin
	err := r.db.GetContext(ctx, &plugin,
		`SELECT * FROM marketplace_plugins WHERE slug = $1`, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found: %s", slug)
	}
	return &plugin, err
}

func (r *PostgresRepository) UpdatePlugin(ctx context.Context, plugin *Plugin) error {
	plugin.UpdatedAt = time.Now()
	query := `UPDATE marketplace_plugins SET name=$1, description=$2, long_description=$3,
		type=$4, status=$5, pricing=$6, price_monthly=$7, version=$8, icon_url=$9,
		source_url=$10, documentation_url=$11, verified=$12, featured=$13, updated_at=$14
		WHERE id=$15`
	_, err := r.db.ExecContext(ctx, query,
		plugin.Name, plugin.Description, plugin.LongDesc, plugin.Type, plugin.Status,
		plugin.Pricing, plugin.PriceMonthly, plugin.Version, plugin.IconURL,
		plugin.SourceURL, plugin.DocURL, plugin.Verified, plugin.Featured,
		plugin.UpdatedAt, plugin.ID)
	return err
}

func (r *PostgresRepository) SearchPlugins(ctx context.Context, params *PluginSearchParams) (*PluginSearchResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	where := []string{"status = 'published'"}
	args := []interface{}{}
	argIdx := 1

	if params.Query != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+params.Query+"%")
		argIdx++
	}
	if params.Type != "" {
		where = append(where, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, params.Type)
		argIdx++
	}
	if params.Pricing != "" {
		where = append(where, fmt.Sprintf("pricing = $%d", argIdx))
		args = append(args, params.Pricing)
		argIdx++
	}
	if params.Verified != nil && *params.Verified {
		where = append(where, "verified = true")
	}
	if params.Featured != nil && *params.Featured {
		where = append(where, "featured = true")
	}

	whereClause := strings.Join(where, " AND ")

	orderBy := "installs DESC"
	switch params.SortBy {
	case "rating":
		orderBy = "avg_rating DESC"
	case "newest":
		orderBy = "created_at DESC"
	case "name":
		orderBy = "name ASC"
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM marketplace_plugins WHERE %s", whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	dataQuery := fmt.Sprintf("SELECT * FROM marketplace_plugins WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d",
		whereClause, orderBy, argIdx, argIdx+1)
	args = append(args, params.PageSize, offset)

	var plugins []Plugin
	err = r.db.SelectContext(ctx, &plugins, dataQuery, args...)
	if err != nil {
		return nil, err
	}

	totalPages := (total + params.PageSize - 1) / params.PageSize
	return &PluginSearchResult{
		Plugins:    plugins,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PostgresRepository) DeletePlugin(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketplace_plugins SET status = 'archived', updated_at = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

func (r *PostgresRepository) CreateVersion(ctx context.Context, version *PluginVersion) error {
	if version.ID == "" {
		version.ID = uuid.New().String()
	}
	version.ReleasedAt = time.Now()

	// Mark previous latest as not latest — propagate error to avoid stale is_latest
	if _, err := r.db.ExecContext(ctx, `UPDATE marketplace_plugin_versions SET is_latest = false WHERE plugin_id = $1`, version.PluginID); err != nil {
		return fmt.Errorf("failed to unset previous latest version for plugin %s: %w", version.PluginID, err)
	}

	query := `INSERT INTO marketplace_plugin_versions (id, plugin_id, version, changelog, min_platform_version,
		checksum, size_bytes, downloads, is_latest, released_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err := r.db.ExecContext(ctx, query,
		version.ID, version.PluginID, version.Version, version.Changelog,
		version.MinPlatform, version.Checksum, version.Size, version.Downloads,
		true, version.ReleasedAt)
	return err
}

func (r *PostgresRepository) GetVersions(ctx context.Context, pluginID string) ([]PluginVersion, error) {
	var versions []PluginVersion
	err := r.db.SelectContext(ctx, &versions,
		`SELECT * FROM marketplace_plugin_versions WHERE plugin_id = $1 ORDER BY released_at DESC`, pluginID)
	return versions, err
}

func (r *PostgresRepository) GetLatestVersion(ctx context.Context, pluginID string) (*PluginVersion, error) {
	var version PluginVersion
	err := r.db.GetContext(ctx, &version,
		`SELECT * FROM marketplace_plugin_versions WHERE plugin_id = $1 AND is_latest = true`, pluginID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no versions found for plugin: %s", pluginID)
	}
	return &version, err
}

func (r *PostgresRepository) InstallPlugin(ctx context.Context, install *PluginInstallation) error {
	if install.ID == "" {
		install.ID = uuid.New().String()
	}
	now := time.Now()
	install.InstalledAt = now
	install.UpdatedAt = now
	install.Status = "active"

	query := `INSERT INTO marketplace_installations (id, tenant_id, plugin_id, version_id, status, installed_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := r.db.ExecContext(ctx, query,
		install.ID, install.TenantID, install.PluginID, install.VersionID,
		install.Status, install.InstalledAt, install.UpdatedAt)
	if err != nil {
		return err
	}

	// Increment install count
	_, err = r.db.ExecContext(ctx, `UPDATE marketplace_plugins SET installs = installs + 1 WHERE id = $1`, install.PluginID)
	return err
}

func (r *PostgresRepository) UninstallPlugin(ctx context.Context, tenantID, pluginID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE marketplace_installations SET status = 'uninstalled', updated_at = $1 WHERE tenant_id = $2 AND plugin_id = $3`,
		time.Now(), tenantID, pluginID)
	return err
}

func (r *PostgresRepository) GetInstallation(ctx context.Context, tenantID, pluginID string) (*PluginInstallation, error) {
	var install PluginInstallation
	err := r.db.GetContext(ctx, &install,
		`SELECT * FROM marketplace_installations WHERE tenant_id = $1 AND plugin_id = $2 AND status = 'active'`,
		tenantID, pluginID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &install, err
}

func (r *PostgresRepository) ListInstallations(ctx context.Context, tenantID string) ([]PluginInstallation, error) {
	var installs []PluginInstallation
	err := r.db.SelectContext(ctx, &installs,
		`SELECT * FROM marketplace_installations WHERE tenant_id = $1 AND status = 'active' ORDER BY installed_at DESC`, tenantID)
	return installs, err
}

func (r *PostgresRepository) CreateReview(ctx context.Context, review *PluginReview) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	review.CreatedAt = time.Now()

	query := `INSERT INTO marketplace_reviews (id, plugin_id, tenant_id, rating, title, body, helpful_count, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.db.ExecContext(ctx, query,
		review.ID, review.PluginID, review.TenantID, review.Rating,
		review.Title, review.Body, 0, review.CreatedAt)
	if err != nil {
		return err
	}

	// Update plugin average rating
	_, err = r.db.ExecContext(ctx, `UPDATE marketplace_plugins SET
		avg_rating = (SELECT COALESCE(AVG(rating), 0) FROM marketplace_reviews WHERE plugin_id = $1),
		review_count = (SELECT COUNT(*) FROM marketplace_reviews WHERE plugin_id = $1)
		WHERE id = $1`, review.PluginID)
	return err
}

func (r *PostgresRepository) GetReviews(ctx context.Context, pluginID string, page, pageSize int) ([]PluginReview, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM marketplace_reviews WHERE plugin_id = $1`, pluginID)
	if err != nil {
		return nil, 0, err
	}

	var reviews []PluginReview
	offset := (page - 1) * pageSize
	err = r.db.SelectContext(ctx, &reviews,
		`SELECT * FROM marketplace_reviews WHERE plugin_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		pluginID, pageSize, offset)
	return reviews, total, err
}

func (r *PostgresRepository) GetPluginHooks(ctx context.Context, pluginID string) ([]PluginHook, error) {
	var hooks []PluginHook
	err := r.db.SelectContext(ctx, &hooks,
		`SELECT * FROM marketplace_plugin_hooks WHERE plugin_id = $1 ORDER BY priority ASC`, pluginID)
	return hooks, err
}

func (r *PostgresRepository) CreatePluginHook(ctx context.Context, hook *PluginHook) error {
	if hook.ID == "" {
		hook.ID = uuid.New().String()
	}
	query := `INSERT INTO marketplace_plugin_hooks (id, plugin_id, hook_point, priority) VALUES ($1,$2,$3,$4)`
	_, err := r.db.ExecContext(ctx, query, hook.ID, hook.PluginID, hook.HookPoint, hook.Priority)
	return err
}

func (r *PostgresRepository) GetMarketplaceStats(ctx context.Context) (*PluginStats, error) {
	stats := &PluginStats{
		TopCategories: make(map[string]int),
	}

	err := r.db.GetContext(ctx, &stats.TotalPlugins,
		`SELECT COUNT(*) FROM marketplace_plugins WHERE status = 'published'`)
	if err != nil {
		return stats, err
	}

	_ = r.db.GetContext(ctx, &stats.TotalInstalls,
		`SELECT COALESCE(SUM(installs), 0) FROM marketplace_plugins WHERE status = 'published'`)
	_ = r.db.GetContext(ctx, &stats.AvgRating,
		`SELECT COALESCE(AVG(avg_rating), 0) FROM marketplace_plugins WHERE status = 'published' AND review_count > 0`)
	_ = r.db.GetContext(ctx, &stats.PublishedLast30d,
		`SELECT COUNT(*) FROM marketplace_plugins WHERE status = 'published' AND published_at >= NOW() - INTERVAL '30 days'`)

	stats.ActivePlugins = stats.TotalPlugins
	return stats, nil
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	return slug
}

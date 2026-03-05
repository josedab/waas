package whitelabel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines whitelabel data access
type Repository interface {
	// Whitelabel configs
	CreateConfig(ctx context.Context, config *WhitelabelConfig) error
	GetConfig(ctx context.Context, tenantID string) (*WhitelabelConfig, error)
	UpdateConfig(ctx context.Context, config *WhitelabelConfig) error
	DeleteConfig(ctx context.Context, tenantID string) error

	// DNS verifications
	CreateDNSVerification(ctx context.Context, record *DNSVerification) error
	GetDNSVerifications(ctx context.Context, configID string) ([]DNSVerification, error)
	UpdateDNSVerification(ctx context.Context, record *DNSVerification) error

	// Sub-tenants
	CreateSubTenant(ctx context.Context, subTenant *SubTenant) error
	GetSubTenant(ctx context.Context, parentTenantID, subTenantID string) (*SubTenant, error)
	ListSubTenants(ctx context.Context, parentTenantID string) ([]SubTenant, error)
	UpdateSubTenantStatus(ctx context.Context, parentTenantID, subTenantID string, status SubTenantStatus) error

	// Partners
	CreatePartner(ctx context.Context, partner *Partner) error
	GetPartner(ctx context.Context, tenantID, partnerID string) (*Partner, error)
	ListPartners(ctx context.Context, tenantID string) ([]Partner, error)

	// Revenue
	CreatePartnerRevenue(ctx context.Context, revenue *PartnerRevenue) error
	GetPartnerRevenue(ctx context.Context, partnerID string) ([]PartnerRevenue, error)

	// Analytics
	GetAnalytics(ctx context.Context, configID, tenantID string) (*WhitelabelAnalytics, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateConfig creates a new whitelabel configuration
func (r *PostgresRepository) CreateConfig(ctx context.Context, config *WhitelabelConfig) error {
	query := `
		INSERT INTO whitelabel_configs (
			id, tenant_id, custom_domain, brand_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, custom_css,
			domain_verified, ssl_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.TenantID, config.CustomDomain, config.BrandName,
		config.LogoURL, config.FaviconURL, config.PrimaryColor, config.SecondaryColor,
		config.AccentColor, config.CustomCSS, config.DomainVerified, config.SSLStatus,
		config.CreatedAt, config.UpdatedAt)

	return err
}

// GetConfig retrieves a whitelabel configuration by tenant ID
func (r *PostgresRepository) GetConfig(ctx context.Context, tenantID string) (*WhitelabelConfig, error) {
	query := `
		SELECT id, tenant_id, custom_domain, brand_name, logo_url, favicon_url,
			   primary_color, secondary_color, accent_color, custom_css,
			   domain_verified, ssl_status, created_at, updated_at
		FROM whitelabel_configs
		WHERE tenant_id = $1`

	var config WhitelabelConfig
	var logoURL, faviconURL, primaryColor, secondaryColor, accentColor, customCSS sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.ID, &config.TenantID, &config.CustomDomain, &config.BrandName,
		&logoURL, &faviconURL, &primaryColor, &secondaryColor,
		&accentColor, &customCSS, &config.DomainVerified, &config.SSLStatus,
		&config.CreatedAt, &config.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("whitelabel config not found")
	}
	if err != nil {
		return nil, err
	}

	if logoURL.Valid {
		config.LogoURL = logoURL.String
	}
	if faviconURL.Valid {
		config.FaviconURL = faviconURL.String
	}
	if primaryColor.Valid {
		config.PrimaryColor = primaryColor.String
	}
	if secondaryColor.Valid {
		config.SecondaryColor = secondaryColor.String
	}
	if accentColor.Valid {
		config.AccentColor = accentColor.String
	}
	if customCSS.Valid {
		config.CustomCSS = customCSS.String
	}

	return &config, nil
}

// UpdateConfig updates a whitelabel configuration
func (r *PostgresRepository) UpdateConfig(ctx context.Context, config *WhitelabelConfig) error {
	query := `
		UPDATE whitelabel_configs SET
			custom_domain = $1, brand_name = $2, logo_url = $3, favicon_url = $4,
			primary_color = $5, secondary_color = $6, accent_color = $7, custom_css = $8,
			domain_verified = $9, ssl_status = $10, updated_at = $11
		WHERE tenant_id = $12`

	_, err := r.db.ExecContext(ctx, query,
		config.CustomDomain, config.BrandName, config.LogoURL, config.FaviconURL,
		config.PrimaryColor, config.SecondaryColor, config.AccentColor, config.CustomCSS,
		config.DomainVerified, config.SSLStatus, config.UpdatedAt, config.TenantID)

	return err
}

// DeleteConfig deletes a whitelabel configuration
func (r *PostgresRepository) DeleteConfig(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM whitelabel_configs WHERE tenant_id = $1", tenantID)
	return err
}

// CreateDNSVerification creates a DNS verification record
func (r *PostgresRepository) CreateDNSVerification(ctx context.Context, record *DNSVerification) error {
	query := `
		INSERT INTO whitelabel_dns_verifications (
			id, config_id, record_type, record_name, record_value, verified, verified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		record.ID, record.ConfigID, record.RecordType, record.RecordName,
		record.RecordValue, record.Verified, record.VerifiedAt)

	return err
}

// GetDNSVerifications retrieves DNS verification records for a config
func (r *PostgresRepository) GetDNSVerifications(ctx context.Context, configID string) ([]DNSVerification, error) {
	query := `
		SELECT id, config_id, record_type, record_name, record_value, verified, verified_at
		FROM whitelabel_dns_verifications
		WHERE config_id = $1
		ORDER BY record_type`

	rows, err := r.db.QueryContext(ctx, query, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DNSVerification
	for rows.Next() {
		var record DNSVerification
		var verifiedAt sql.NullTime

		if err := rows.Scan(&record.ID, &record.ConfigID, &record.RecordType,
			&record.RecordName, &record.RecordValue, &record.Verified, &verifiedAt); err != nil {
			continue
		}
		if verifiedAt.Valid {
			record.VerifiedAt = &verifiedAt.Time
		}
		records = append(records, record)
	}

	return records, nil
}

// UpdateDNSVerification updates a DNS verification record
func (r *PostgresRepository) UpdateDNSVerification(ctx context.Context, record *DNSVerification) error {
	query := `
		UPDATE whitelabel_dns_verifications SET
			verified = $1, verified_at = $2
		WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, record.Verified, record.VerifiedAt, record.ID)
	return err
}

// CreateSubTenant creates a new sub-tenant
func (r *PostgresRepository) CreateSubTenant(ctx context.Context, subTenant *SubTenant) error {
	query := `
		INSERT INTO whitelabel_sub_tenants (
			id, parent_tenant_id, name, email, custom_domain, plan,
			status, webhooks_used, webhooks_limit, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		subTenant.ID, subTenant.ParentTenantID, subTenant.Name, subTenant.Email,
		subTenant.CustomDomain, subTenant.Plan, subTenant.Status,
		subTenant.WebhooksUsed, subTenant.WebhooksLimit, subTenant.CreatedAt)

	return err
}

// GetSubTenant retrieves a sub-tenant by ID
func (r *PostgresRepository) GetSubTenant(ctx context.Context, parentTenantID, subTenantID string) (*SubTenant, error) {
	query := `
		SELECT id, parent_tenant_id, name, email, custom_domain, plan,
			   status, webhooks_used, webhooks_limit, created_at
		FROM whitelabel_sub_tenants
		WHERE parent_tenant_id = $1 AND id = $2`

	var subTenant SubTenant
	var customDomain sql.NullString

	err := r.db.QueryRowContext(ctx, query, parentTenantID, subTenantID).Scan(
		&subTenant.ID, &subTenant.ParentTenantID, &subTenant.Name, &subTenant.Email,
		&customDomain, &subTenant.Plan, &subTenant.Status,
		&subTenant.WebhooksUsed, &subTenant.WebhooksLimit, &subTenant.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sub-tenant not found")
	}
	if err != nil {
		return nil, err
	}

	if customDomain.Valid {
		subTenant.CustomDomain = customDomain.String
	}

	return &subTenant, nil
}

// ListSubTenants lists sub-tenants for a parent tenant
func (r *PostgresRepository) ListSubTenants(ctx context.Context, parentTenantID string) ([]SubTenant, error) {
	query := `
		SELECT id, parent_tenant_id, name, email, custom_domain, plan,
			   status, webhooks_used, webhooks_limit, created_at
		FROM whitelabel_sub_tenants
		WHERE parent_tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, parentTenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subTenants []SubTenant
	for rows.Next() {
		var subTenant SubTenant
		var customDomain sql.NullString

		err := rows.Scan(
			&subTenant.ID, &subTenant.ParentTenantID, &subTenant.Name, &subTenant.Email,
			&customDomain, &subTenant.Plan, &subTenant.Status,
			&subTenant.WebhooksUsed, &subTenant.WebhooksLimit, &subTenant.CreatedAt)
		if err != nil {
			continue
		}

		if customDomain.Valid {
			subTenant.CustomDomain = customDomain.String
		}

		subTenants = append(subTenants, subTenant)
	}

	return subTenants, nil
}

// UpdateSubTenantStatus updates a sub-tenant's status
func (r *PostgresRepository) UpdateSubTenantStatus(ctx context.Context, parentTenantID, subTenantID string, status SubTenantStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE whitelabel_sub_tenants SET status = $1 WHERE parent_tenant_id = $2 AND id = $3",
		status, parentTenantID, subTenantID)
	return err
}

// CreatePartner creates a new partner
func (r *PostgresRepository) CreatePartner(ctx context.Context, partner *Partner) error {
	query := `
		INSERT INTO whitelabel_partners (
			id, tenant_id, company_name, contact_email, revenue_share_pct,
			total_sub_tenants, total_revenue, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		partner.ID, partner.TenantID, partner.CompanyName, partner.ContactEmail,
		partner.RevenueSharePct, partner.TotalSubTenants, partner.TotalRevenue,
		partner.Status, partner.CreatedAt)

	return err
}

// GetPartner retrieves a partner by ID
func (r *PostgresRepository) GetPartner(ctx context.Context, tenantID, partnerID string) (*Partner, error) {
	query := `
		SELECT id, tenant_id, company_name, contact_email, revenue_share_pct,
			   total_sub_tenants, total_revenue, status, created_at
		FROM whitelabel_partners
		WHERE tenant_id = $1 AND id = $2`

	var partner Partner
	err := r.db.QueryRowContext(ctx, query, tenantID, partnerID).Scan(
		&partner.ID, &partner.TenantID, &partner.CompanyName, &partner.ContactEmail,
		&partner.RevenueSharePct, &partner.TotalSubTenants, &partner.TotalRevenue,
		&partner.Status, &partner.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("partner not found")
	}
	if err != nil {
		return nil, err
	}

	return &partner, nil
}

// ListPartners lists partners for a tenant
func (r *PostgresRepository) ListPartners(ctx context.Context, tenantID string) ([]Partner, error) {
	query := `
		SELECT id, tenant_id, company_name, contact_email, revenue_share_pct,
			   total_sub_tenants, total_revenue, status, created_at
		FROM whitelabel_partners
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partners []Partner
	for rows.Next() {
		var partner Partner
		err := rows.Scan(
			&partner.ID, &partner.TenantID, &partner.CompanyName, &partner.ContactEmail,
			&partner.RevenueSharePct, &partner.TotalSubTenants, &partner.TotalRevenue,
			&partner.Status, &partner.CreatedAt)
		if err != nil {
			continue
		}
		partners = append(partners, partner)
	}

	return partners, nil
}

// CreatePartnerRevenue creates a partner revenue record
func (r *PostgresRepository) CreatePartnerRevenue(ctx context.Context, revenue *PartnerRevenue) error {
	query := `
		INSERT INTO whitelabel_partner_revenue (
			partner_id, period, sub_tenant_revenue, share_amount, payment_status, paid_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (partner_id, period) DO UPDATE SET
			sub_tenant_revenue = EXCLUDED.sub_tenant_revenue,
			share_amount = EXCLUDED.share_amount,
			payment_status = EXCLUDED.payment_status,
			paid_at = EXCLUDED.paid_at`

	_, err := r.db.ExecContext(ctx, query,
		revenue.PartnerID, revenue.Period, revenue.SubTenantRevenue,
		revenue.ShareAmount, revenue.PaymentStatus, revenue.PaidAt)

	return err
}

// GetPartnerRevenue retrieves revenue records for a partner
func (r *PostgresRepository) GetPartnerRevenue(ctx context.Context, partnerID string) ([]PartnerRevenue, error) {
	query := `
		SELECT partner_id, period, sub_tenant_revenue, share_amount, payment_status, paid_at
		FROM whitelabel_partner_revenue
		WHERE partner_id = $1
		ORDER BY period DESC`

	rows, err := r.db.QueryContext(ctx, query, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revenues []PartnerRevenue
	for rows.Next() {
		var revenue PartnerRevenue
		var paidAt sql.NullTime

		err := rows.Scan(&revenue.PartnerID, &revenue.Period, &revenue.SubTenantRevenue,
			&revenue.ShareAmount, &revenue.PaymentStatus, &paidAt)
		if err != nil {
			continue
		}
		if paidAt.Valid {
			revenue.PaidAt = &paidAt.Time
		}
		revenues = append(revenues, revenue)
	}

	return revenues, nil
}

// GetAnalytics retrieves analytics for a whitelabel config
func (r *PostgresRepository) GetAnalytics(ctx context.Context, configID, tenantID string) (*WhitelabelAnalytics, error) {
	analytics := &WhitelabelAnalytics{
		ConfigID: configID,
	}

	// Get sub-tenant counts
	countQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active
		FROM whitelabel_sub_tenants
		WHERE parent_tenant_id = $1`

	err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(
		&analytics.TotalSubTenants, &analytics.ActiveSubTenants)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Get total webhooks
	webhookQuery := `
		SELECT COALESCE(SUM(webhooks_used), 0)
		FROM whitelabel_sub_tenants
		WHERE parent_tenant_id = $1`

	err = r.db.QueryRowContext(ctx, webhookQuery, tenantID).Scan(&analytics.TotalWebhooks)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Get monthly revenue
	period := time.Now().Format("2006-01")
	revenueQuery := `
		SELECT COALESCE(SUM(sub_tenant_revenue), 0)
		FROM whitelabel_partner_revenue pr
		JOIN whitelabel_partners p ON pr.partner_id = p.id
		WHERE p.tenant_id = $1 AND pr.period = $2`

	err = r.db.QueryRowContext(ctx, revenueQuery, tenantID, period).Scan(&analytics.MonthlyRevenue)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return analytics, nil
}

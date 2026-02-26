package whitelabel

import (
	"time"
)

// WhitelabelConfig represents a tenant's whitelabel configuration
type WhitelabelConfig struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	CustomDomain   string    `json:"custom_domain"`
	BrandName      string    `json:"brand_name"`
	LogoURL        string    `json:"logo_url,omitempty"`
	FaviconURL     string    `json:"favicon_url,omitempty"`
	PrimaryColor   string    `json:"primary_color,omitempty"`
	SecondaryColor string    `json:"secondary_color,omitempty"`
	AccentColor    string    `json:"accent_color,omitempty"`
	CustomCSS      string    `json:"custom_css,omitempty"`
	DomainVerified bool      `json:"domain_verified"`
	SSLStatus      SSLStatus `json:"ssl_status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SSLStatus defines SSL certificate status
type SSLStatus string

const (
	SSLPending      SSLStatus = "pending"
	SSLProvisioning SSLStatus = "provisioning"
	SSLActive       SSLStatus = "active"
	SSLExpired      SSLStatus = "expired"
	SSLFailed       SSLStatus = "failed"
)

// DNSVerification represents a DNS record for domain verification
type DNSVerification struct {
	ID         string     `json:"id"`
	ConfigID   string     `json:"config_id"`
	RecordType string     `json:"record_type"`
	RecordName string     `json:"record_name"`
	RecordValue string    `json:"record_value"`
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
}

// SubTenant represents a sub-tenant within a partner's account
type SubTenant struct {
	ID             string          `json:"id"`
	ParentTenantID string          `json:"parent_tenant_id"`
	Name           string          `json:"name"`
	Email          string          `json:"email"`
	CustomDomain   string          `json:"custom_domain,omitempty"`
	Plan           string          `json:"plan"`
	Status         SubTenantStatus `json:"status"`
	WebhooksUsed   int64           `json:"webhooks_used"`
	WebhooksLimit  int64           `json:"webhooks_limit"`
	CreatedAt      time.Time       `json:"created_at"`
}

// SubTenantStatus defines sub-tenant status
type SubTenantStatus string

const (
	SubTenantActive    SubTenantStatus = "active"
	SubTenantSuspended SubTenantStatus = "suspended"
	SubTenantTrial     SubTenantStatus = "trial"
	SubTenantCancelled SubTenantStatus = "cancelled"
)

// Partner represents a partner account
type Partner struct {
	ID              string        `json:"id"`
	TenantID        string        `json:"tenant_id"`
	CompanyName     string        `json:"company_name"`
	ContactEmail    string        `json:"contact_email"`
	RevenueSharePct float64       `json:"revenue_share_pct"`
	TotalSubTenants int           `json:"total_sub_tenants"`
	TotalRevenue    float64       `json:"total_revenue"`
	Status          PartnerStatus `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
}

// PartnerStatus defines partner status
type PartnerStatus string

const (
	PartnerActive    PartnerStatus = "active"
	PartnerPending   PartnerStatus = "pending"
	PartnerSuspended PartnerStatus = "suspended"
)

// PartnerRevenue represents revenue data for a partner
type PartnerRevenue struct {
	PartnerID        string     `json:"partner_id"`
	Period           string     `json:"period"`
	SubTenantRevenue float64    `json:"sub_tenant_revenue"`
	ShareAmount      float64    `json:"share_amount"`
	PaymentStatus    string     `json:"payment_status"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
}

// BrandingPreview represents a branding preview
type BrandingPreview struct {
	ConfigID    string    `json:"config_id"`
	PreviewURL  string    `json:"preview_url"`
	GeneratedAt time.Time `json:"generated_at"`
}

// WhitelabelAnalytics represents analytics for a whitelabel config
type WhitelabelAnalytics struct {
	ConfigID         string  `json:"config_id"`
	TotalSubTenants  int     `json:"total_sub_tenants"`
	ActiveSubTenants int     `json:"active_sub_tenants"`
	TotalWebhooks    int64   `json:"total_webhooks"`
	MonthlyRevenue   float64 `json:"monthly_revenue"`
	GrowthRate       float64 `json:"growth_rate"`
}

// CreateWhitelabelRequest represents a request to create a whitelabel config
type CreateWhitelabelRequest struct {
	CustomDomain   string `json:"custom_domain" binding:"required"`
	BrandName      string `json:"brand_name" binding:"required"`
	LogoURL        string `json:"logo_url,omitempty"`
	FaviconURL     string `json:"favicon_url,omitempty"`
	PrimaryColor   string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	AccentColor    string `json:"accent_color,omitempty"`
	CustomCSS      string `json:"custom_css,omitempty"`
}

// VerifyDomainRequest represents a request to verify a domain
type VerifyDomainRequest struct {
	ConfigID string `json:"config_id" binding:"required"`
}

// CreateSubTenantRequest represents a request to create a sub-tenant
type CreateSubTenantRequest struct {
	Name          string `json:"name" binding:"required"`
	Email         string `json:"email" binding:"required"`
	CustomDomain  string `json:"custom_domain,omitempty"`
	Plan          string `json:"plan" binding:"required"`
	WebhooksLimit int64  `json:"webhooks_limit,omitempty"`
}

// CreatePartnerRequest represents a request to create a partner
type CreatePartnerRequest struct {
	CompanyName     string  `json:"company_name" binding:"required"`
	ContactEmail    string  `json:"contact_email" binding:"required"`
	RevenueSharePct float64 `json:"revenue_share_pct" binding:"required"`
}

// UpdateBrandingRequest represents a request to update branding
type UpdateBrandingRequest struct {
	BrandName      string `json:"brand_name,omitempty"`
	LogoURL        string `json:"logo_url,omitempty"`
	FaviconURL     string `json:"favicon_url,omitempty"`
	PrimaryColor   string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	AccentColor    string `json:"accent_color,omitempty"`
	CustomCSS      string `json:"custom_css,omitempty"`
}

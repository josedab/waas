package whitelabel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides whitelabel operations
type Service struct {
	repo Repository
}

// NewService creates a new whitelabel service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateWhitelabelConfig creates a new whitelabel configuration
func (s *Service) CreateWhitelabelConfig(ctx context.Context, tenantID string, req *CreateWhitelabelRequest) (*WhitelabelConfig, error) {
	now := time.Now()
	config := &WhitelabelConfig{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		CustomDomain:   req.CustomDomain,
		BrandName:      req.BrandName,
		LogoURL:        req.LogoURL,
		FaviconURL:     req.FaviconURL,
		PrimaryColor:   req.PrimaryColor,
		SecondaryColor: req.SecondaryColor,
		AccentColor:    req.AccentColor,
		CustomCSS:      req.CustomCSS,
		DomainVerified: false,
		SSLStatus:      SSLPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("create whitelabel config: %w", err)
	}

	return config, nil
}

// GetConfig retrieves the whitelabel configuration for a tenant
func (s *Service) GetConfig(ctx context.Context, tenantID string) (*WhitelabelConfig, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get whitelabel config: %w", err)
	}
	return config, nil
}

// UpdateConfig updates a whitelabel configuration
func (s *Service) UpdateConfig(ctx context.Context, tenantID string, req *CreateWhitelabelRequest) (*WhitelabelConfig, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for update: %w", err)
	}

	config.CustomDomain = req.CustomDomain
	config.BrandName = req.BrandName
	config.LogoURL = req.LogoURL
	config.FaviconURL = req.FaviconURL
	config.PrimaryColor = req.PrimaryColor
	config.SecondaryColor = req.SecondaryColor
	config.AccentColor = req.AccentColor
	config.CustomCSS = req.CustomCSS
	config.UpdatedAt = time.Now()

	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("update whitelabel config: %w", err)
	}

	return config, nil
}

// DeleteConfig deletes a whitelabel configuration
func (s *Service) DeleteConfig(ctx context.Context, tenantID string) error {
	if err := s.repo.DeleteConfig(ctx, tenantID); err != nil {
		return fmt.Errorf("delete whitelabel config: %w", err)
	}
	return nil
}

// SetupCustomDomain creates DNS verification records for a custom domain
func (s *Service) SetupCustomDomain(ctx context.Context, tenantID string) (*WhitelabelConfig, []DNSVerification, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("get config for domain setup: %w", err)
	}

	// Create CNAME verification record
	cnameRecord := &DNSVerification{
		ID:          uuid.New().String(),
		ConfigID:    config.ID,
		RecordType:  "CNAME",
		RecordName:  config.CustomDomain,
		RecordValue: fmt.Sprintf("proxy.%s.webhookplatform.io", tenantID),
		Verified:    false,
	}

	// Create TXT verification record
	txtRecord := &DNSVerification{
		ID:          uuid.New().String(),
		ConfigID:    config.ID,
		RecordType:  "TXT",
		RecordName:  fmt.Sprintf("_verify.%s", config.CustomDomain),
		RecordValue: fmt.Sprintf("whitelabel-verify=%s", config.ID),
		Verified:    false,
	}

	if err := s.repo.CreateDNSVerification(ctx, cnameRecord); err != nil {
		return nil, nil, fmt.Errorf("create CNAME record: %w", err)
	}
	if err := s.repo.CreateDNSVerification(ctx, txtRecord); err != nil {
		return nil, nil, fmt.Errorf("create TXT record: %w", err)
	}

	records := []DNSVerification{*cnameRecord, *txtRecord}
	return config, records, nil
}

// VerifyDomain checks DNS verification records
func (s *Service) VerifyDomain(ctx context.Context, tenantID string) (*WhitelabelConfig, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for verification: %w", err)
	}

	records, err := s.repo.GetDNSVerifications(ctx, config.ID)
	if err != nil {
		return nil, fmt.Errorf("get DNS records: %w", err)
	}

	allVerified := true
	now := time.Now()
	for i := range records {
		// In production, this would perform actual DNS lookups
		records[i].Verified = true
		records[i].VerifiedAt = &now
		if err := s.repo.UpdateDNSVerification(ctx, &records[i]); err != nil {
			return nil, fmt.Errorf("update DNS record: %w", err)
		}
	}

	if allVerified {
		config.DomainVerified = true
		config.UpdatedAt = now
		if err := s.repo.UpdateConfig(ctx, config); err != nil {
			return nil, fmt.Errorf("update domain verified: %w", err)
		}
	}

	return config, nil
}

// ProvisionSSL provisions SSL for a verified domain
func (s *Service) ProvisionSSL(ctx context.Context, tenantID string) (*WhitelabelConfig, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for SSL: %w", err)
	}

	if !config.DomainVerified {
		return nil, fmt.Errorf("domain must be verified before SSL provisioning")
	}

	config.SSLStatus = SSLProvisioning
	config.UpdatedAt = time.Now()

	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("update SSL status: %w", err)
	}

	// In production, this would trigger async SSL provisioning via Let's Encrypt or similar
	go func() {
		config.SSLStatus = SSLActive
		config.UpdatedAt = time.Now()
		_ = s.repo.UpdateConfig(context.Background(), config)
	}()

	return config, nil
}

// CreateSubTenant creates a new sub-tenant
func (s *Service) CreateSubTenant(ctx context.Context, parentTenantID string, req *CreateSubTenantRequest) (*SubTenant, error) {
	webhooksLimit := req.WebhooksLimit
	if webhooksLimit <= 0 {
		webhooksLimit = 10000
	}

	subTenant := &SubTenant{
		ID:             uuid.New().String(),
		ParentTenantID: parentTenantID,
		Name:           req.Name,
		Email:          req.Email,
		CustomDomain:   req.CustomDomain,
		Plan:           req.Plan,
		Status:         SubTenantActive,
		WebhooksUsed:   0,
		WebhooksLimit:  webhooksLimit,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreateSubTenant(ctx, subTenant); err != nil {
		return nil, fmt.Errorf("create sub-tenant: %w", err)
	}

	return subTenant, nil
}

// ListSubTenants lists sub-tenants for a parent tenant
func (s *Service) ListSubTenants(ctx context.Context, parentTenantID string) ([]SubTenant, error) {
	subTenants, err := s.repo.ListSubTenants(ctx, parentTenantID)
	if err != nil {
		return nil, fmt.Errorf("list sub-tenants: %w", err)
	}
	return subTenants, nil
}

// GetSubTenant retrieves a sub-tenant by ID
func (s *Service) GetSubTenant(ctx context.Context, parentTenantID, subTenantID string) (*SubTenant, error) {
	subTenant, err := s.repo.GetSubTenant(ctx, parentTenantID, subTenantID)
	if err != nil {
		return nil, fmt.Errorf("get sub-tenant: %w", err)
	}
	return subTenant, nil
}

// SuspendSubTenant suspends a sub-tenant
func (s *Service) SuspendSubTenant(ctx context.Context, parentTenantID, subTenantID string) error {
	if err := s.repo.UpdateSubTenantStatus(ctx, parentTenantID, subTenantID, SubTenantSuspended); err != nil {
		return fmt.Errorf("suspend sub-tenant: %w", err)
	}
	return nil
}

// ReactivateSubTenant reactivates a suspended sub-tenant
func (s *Service) ReactivateSubTenant(ctx context.Context, parentTenantID, subTenantID string) error {
	if err := s.repo.UpdateSubTenantStatus(ctx, parentTenantID, subTenantID, SubTenantActive); err != nil {
		return fmt.Errorf("reactivate sub-tenant: %w", err)
	}
	return nil
}

// RegisterPartner registers a new partner
func (s *Service) RegisterPartner(ctx context.Context, tenantID string, req *CreatePartnerRequest) (*Partner, error) {
	partner := &Partner{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		CompanyName:     req.CompanyName,
		ContactEmail:    req.ContactEmail,
		RevenueSharePct: req.RevenueSharePct,
		TotalSubTenants: 0,
		TotalRevenue:    0,
		Status:          PartnerPending,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.CreatePartner(ctx, partner); err != nil {
		return nil, fmt.Errorf("register partner: %w", err)
	}

	return partner, nil
}

// GetPartner retrieves a partner by ID
func (s *Service) GetPartner(ctx context.Context, tenantID, partnerID string) (*Partner, error) {
	partner, err := s.repo.GetPartner(ctx, tenantID, partnerID)
	if err != nil {
		return nil, fmt.Errorf("get partner: %w", err)
	}
	return partner, nil
}

// ListPartners lists partners for a tenant
func (s *Service) ListPartners(ctx context.Context, tenantID string) ([]Partner, error) {
	partners, err := s.repo.ListPartners(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list partners: %w", err)
	}
	return partners, nil
}

// CalculatePartnerRevenue calculates and records revenue for a partner
func (s *Service) CalculatePartnerRevenue(ctx context.Context, tenantID, partnerID string) (*PartnerRevenue, error) {
	partner, err := s.repo.GetPartner(ctx, tenantID, partnerID)
	if err != nil {
		return nil, fmt.Errorf("get partner for revenue: %w", err)
	}

	period := time.Now().Format("2006-01")

	// Calculate sub-tenant revenue for the period
	subTenants, err := s.repo.ListSubTenants(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list sub-tenants for revenue: %w", err)
	}

	var totalRevenue float64
	for _, st := range subTenants {
		// Simple revenue calculation based on webhooks used
		totalRevenue += float64(st.WebhooksUsed) * 0.001
	}

	shareAmount := totalRevenue * (partner.RevenueSharePct / 100.0)

	revenue := &PartnerRevenue{
		PartnerID:        partnerID,
		Period:           period,
		SubTenantRevenue: totalRevenue,
		ShareAmount:      shareAmount,
		PaymentStatus:    "pending",
	}

	if err := s.repo.CreatePartnerRevenue(ctx, revenue); err != nil {
		return nil, fmt.Errorf("create partner revenue: %w", err)
	}

	return revenue, nil
}

// GetPartnerRevenue retrieves revenue records for a partner
func (s *Service) GetPartnerRevenue(ctx context.Context, tenantID, partnerID string) ([]PartnerRevenue, error) {
	// Verify partner belongs to tenant
	if _, err := s.repo.GetPartner(ctx, tenantID, partnerID); err != nil {
		return nil, fmt.Errorf("get partner: %w", err)
	}

	revenues, err := s.repo.GetPartnerRevenue(ctx, partnerID)
	if err != nil {
		return nil, fmt.Errorf("get partner revenue: %w", err)
	}

	return revenues, nil
}

// UpdateBranding updates branding for a whitelabel config
func (s *Service) UpdateBranding(ctx context.Context, tenantID string, req *UpdateBrandingRequest) (*WhitelabelConfig, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for branding: %w", err)
	}

	if req.BrandName != "" {
		config.BrandName = req.BrandName
	}
	if req.LogoURL != "" {
		config.LogoURL = req.LogoURL
	}
	if req.FaviconURL != "" {
		config.FaviconURL = req.FaviconURL
	}
	if req.PrimaryColor != "" {
		config.PrimaryColor = req.PrimaryColor
	}
	if req.SecondaryColor != "" {
		config.SecondaryColor = req.SecondaryColor
	}
	if req.AccentColor != "" {
		config.AccentColor = req.AccentColor
	}
	if req.CustomCSS != "" {
		config.CustomCSS = req.CustomCSS
	}
	config.UpdatedAt = time.Now()

	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("update branding: %w", err)
	}

	return config, nil
}

// GeneratePreview generates a branding preview
func (s *Service) GeneratePreview(ctx context.Context, tenantID string) (*BrandingPreview, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for preview: %w", err)
	}

	preview := &BrandingPreview{
		ConfigID:    config.ID,
		PreviewURL:  fmt.Sprintf("https://preview.webhookplatform.io/%s", config.ID),
		GeneratedAt: time.Now(),
	}

	return preview, nil
}

// GetAnalytics retrieves analytics for a whitelabel config
func (s *Service) GetAnalytics(ctx context.Context, tenantID string) (*WhitelabelAnalytics, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for analytics: %w", err)
	}

	analytics, err := s.repo.GetAnalytics(ctx, config.ID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get analytics: %w", err)
	}

	return analytics, nil
}

// GetDNSRecords retrieves DNS verification records for a tenant's config
func (s *Service) GetDNSRecords(ctx context.Context, tenantID string) ([]DNSVerification, error) {
	config, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get config for DNS records: %w", err)
	}

	records, err := s.repo.GetDNSVerifications(ctx, config.ID)
	if err != nil {
		return nil, fmt.Errorf("get DNS records: %w", err)
	}

	return records, nil
}

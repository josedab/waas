package whitelabel

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type mockWhitelabelRepo struct {
	configs      map[string]*WhitelabelConfig
	dnsRecords   map[string][]DNSVerification
	subTenants   map[string][]SubTenant
	partners     map[string][]Partner
	revenues     map[string][]PartnerRevenue
	analytics    *WhitelabelAnalytics
	createErr    error
	getConfigErr error
}

func newMockRepo() *mockWhitelabelRepo {
	return &mockWhitelabelRepo{
		configs:    make(map[string]*WhitelabelConfig),
		dnsRecords: make(map[string][]DNSVerification),
		subTenants: make(map[string][]SubTenant),
		partners:   make(map[string][]Partner),
		revenues:   make(map[string][]PartnerRevenue),
	}
}

func (m *mockWhitelabelRepo) CreateConfig(_ context.Context, config *WhitelabelConfig) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.configs[config.TenantID] = config
	return nil
}

func (m *mockWhitelabelRepo) GetConfig(_ context.Context, tenantID string) (*WhitelabelConfig, error) {
	if m.getConfigErr != nil {
		return nil, m.getConfigErr
	}
	c, ok := m.configs[tenantID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (m *mockWhitelabelRepo) UpdateConfig(_ context.Context, config *WhitelabelConfig) error {
	m.configs[config.TenantID] = config
	return nil
}

func (m *mockWhitelabelRepo) DeleteConfig(_ context.Context, tenantID string) error {
	delete(m.configs, tenantID)
	return nil
}

func (m *mockWhitelabelRepo) CreateDNSVerification(_ context.Context, record *DNSVerification) error {
	m.dnsRecords[record.ConfigID] = append(m.dnsRecords[record.ConfigID], *record)
	return nil
}

func (m *mockWhitelabelRepo) GetDNSVerifications(_ context.Context, configID string) ([]DNSVerification, error) {
	return m.dnsRecords[configID], nil
}

func (m *mockWhitelabelRepo) UpdateDNSVerification(_ context.Context, record *DNSVerification) error {
	records := m.dnsRecords[record.ConfigID]
	for i, r := range records {
		if r.ID == record.ID {
			records[i] = *record
		}
	}
	m.dnsRecords[record.ConfigID] = records
	return nil
}

func (m *mockWhitelabelRepo) CreateSubTenant(_ context.Context, st *SubTenant) error {
	m.subTenants[st.ParentTenantID] = append(m.subTenants[st.ParentTenantID], *st)
	return nil
}

func (m *mockWhitelabelRepo) GetSubTenant(_ context.Context, parentID, subID string) (*SubTenant, error) {
	for _, st := range m.subTenants[parentID] {
		if st.ID == subID {
			return &st, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockWhitelabelRepo) ListSubTenants(_ context.Context, parentID string) ([]SubTenant, error) {
	return m.subTenants[parentID], nil
}

func (m *mockWhitelabelRepo) UpdateSubTenantStatus(_ context.Context, parentID, subID string, status SubTenantStatus) error {
	subs := m.subTenants[parentID]
	for i, st := range subs {
		if st.ID == subID {
			subs[i].Status = status
		}
	}
	m.subTenants[parentID] = subs
	return nil
}

func (m *mockWhitelabelRepo) CreatePartner(_ context.Context, partner *Partner) error {
	m.partners[partner.TenantID] = append(m.partners[partner.TenantID], *partner)
	return nil
}

func (m *mockWhitelabelRepo) GetPartner(_ context.Context, tenantID, partnerID string) (*Partner, error) {
	for _, p := range m.partners[tenantID] {
		if p.ID == partnerID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockWhitelabelRepo) ListPartners(_ context.Context, tenantID string) ([]Partner, error) {
	return m.partners[tenantID], nil
}

func (m *mockWhitelabelRepo) CreatePartnerRevenue(_ context.Context, rev *PartnerRevenue) error {
	m.revenues[rev.PartnerID] = append(m.revenues[rev.PartnerID], *rev)
	return nil
}

func (m *mockWhitelabelRepo) GetPartnerRevenue(_ context.Context, partnerID string) ([]PartnerRevenue, error) {
	return m.revenues[partnerID], nil
}

func (m *mockWhitelabelRepo) GetAnalytics(_ context.Context, configID, tenantID string) (*WhitelabelAnalytics, error) {
	if m.analytics != nil {
		return m.analytics, nil
	}
	return &WhitelabelAnalytics{ConfigID: configID}, nil
}

// --- Constructor ---

func TestNewService(t *testing.T) {
	svc := NewService(nil)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}

// --- CreateWhitelabelConfig ---

func TestCreateWhitelabelConfig_Valid(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	config, err := svc.CreateWhitelabelConfig(context.Background(), "tenant-1", &CreateWhitelabelRequest{
		CustomDomain: "api.example.com",
		BrandName:    "MyBrand",
		PrimaryColor: "#FF0000",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, config.ID)
	assert.Equal(t, "tenant-1", config.TenantID)
	assert.Equal(t, SSLPending, config.SSLStatus)
	assert.False(t, config.DomainVerified)
	assert.Equal(t, "MyBrand", config.BrandName)
}

func TestCreateWhitelabelConfig_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = fmt.Errorf("db error")
	svc := NewService(repo)

	_, err := svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create whitelabel config")
}

// --- UpdateConfig ---

func TestUpdateConfig_Valid(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "old.com", BrandName: "Old",
	})

	updated, err := svc.UpdateConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "new.com", BrandName: "New", PrimaryColor: "#00FF00",
	})
	require.NoError(t, err)
	assert.Equal(t, "new.com", updated.CustomDomain)
	assert.Equal(t, "New", updated.BrandName)
	assert.Equal(t, "#00FF00", updated.PrimaryColor)
}

func TestUpdateConfig_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	_, err := svc.UpdateConfig(context.Background(), "nonexistent", &CreateWhitelabelRequest{})
	assert.Error(t, err)
}

// --- SetupCustomDomain ---

func TestSetupCustomDomain_CreatesRecords(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "webhooks.example.com", BrandName: "Test",
	})

	config, records, err := svc.SetupCustomDomain(context.Background(), "t1")
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, records, 2)

	var hasCNAME, hasTXT bool
	for _, r := range records {
		if r.RecordType == "CNAME" {
			hasCNAME = true
			assert.Equal(t, "webhooks.example.com", r.RecordName)
			assert.Contains(t, r.RecordValue, "t1")
		}
		if r.RecordType == "TXT" {
			hasTXT = true
			assert.Contains(t, r.RecordName, "_verify")
		}
	}
	assert.True(t, hasCNAME)
	assert.True(t, hasTXT)
}

// --- VerifyDomain ---

func TestVerifyDomain_SetsDomainVerified(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "Test",
	})
	svc.SetupCustomDomain(context.Background(), "t1")

	config, err := svc.VerifyDomain(context.Background(), "t1")
	require.NoError(t, err)
	assert.True(t, config.DomainVerified)
}

// --- ProvisionSSL ---

func TestProvisionSSL_DomainNotVerified(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "Test",
	})

	_, err := svc.ProvisionSSL(context.Background(), "t1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain must be verified")
}

func TestProvisionSSL_DomainVerified(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "Test",
	})
	svc.SetupCustomDomain(context.Background(), "t1")
	svc.VerifyDomain(context.Background(), "t1")

	config, err := svc.ProvisionSSL(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, SSLProvisioning, config.SSLStatus)
}

// --- CreateSubTenant ---

func TestCreateSubTenant_DefaultWebhooksLimit(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	st, err := svc.CreateSubTenant(context.Background(), "parent-1", &CreateSubTenantRequest{
		Name:  "Sub1",
		Email: "sub1@example.com",
		Plan:  "pro",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, st.ID)
	assert.Equal(t, int64(10000), st.WebhooksLimit)
	assert.Equal(t, SubTenantActive, st.Status)
}

func TestCreateSubTenant_CustomWebhooksLimit(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	st, err := svc.CreateSubTenant(context.Background(), "parent-1", &CreateSubTenantRequest{
		Name:          "Sub2",
		Email:         "sub2@example.com",
		Plan:          "enterprise",
		WebhooksLimit: 50000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(50000), st.WebhooksLimit)
}

// --- SuspendSubTenant / ReactivateSubTenant ---

func TestSuspendAndReactivateSubTenant(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	st, _ := svc.CreateSubTenant(context.Background(), "p1", &CreateSubTenantRequest{
		Name: "Sub", Email: "s@e.com", Plan: "basic",
	})

	err := svc.SuspendSubTenant(context.Background(), "p1", st.ID)
	assert.NoError(t, err)

	err = svc.ReactivateSubTenant(context.Background(), "p1", st.ID)
	assert.NoError(t, err)
}

// --- RegisterPartner ---

func TestRegisterPartner_Defaults(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	partner, err := svc.RegisterPartner(context.Background(), "t1", &CreatePartnerRequest{
		CompanyName:     "Acme",
		ContactEmail:    "acme@example.com",
		RevenueSharePct: 20.0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, partner.ID)
	assert.Equal(t, PartnerPending, partner.Status)
	assert.Equal(t, 0, partner.TotalSubTenants)
	assert.Equal(t, 0.0, partner.TotalRevenue)
}

// --- CalculatePartnerRevenue ---

func TestCalculatePartnerRevenue_WithSubTenants(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	// Create partner
	partner, _ := svc.RegisterPartner(context.Background(), "t1", &CreatePartnerRequest{
		CompanyName:     "Acme",
		ContactEmail:    "acme@example.com",
		RevenueSharePct: 20.0,
	})

	// Add sub-tenants with usage
	repo.subTenants["t1"] = []SubTenant{
		{WebhooksUsed: 10000},
		{WebhooksUsed: 5000},
	}

	revenue, err := svc.CalculatePartnerRevenue(context.Background(), "t1", partner.ID)
	require.NoError(t, err)
	assert.Equal(t, 15.0, revenue.SubTenantRevenue) // 15000 * 0.001
	assert.Equal(t, 3.0, revenue.ShareAmount)        // 15.0 * 0.20
	assert.Equal(t, "pending", revenue.PaymentStatus)
}

func TestCalculatePartnerRevenue_ZeroRevenue(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	partner, _ := svc.RegisterPartner(context.Background(), "t1", &CreatePartnerRequest{
		CompanyName:     "Empty",
		ContactEmail:    "e@e.com",
		RevenueSharePct: 50.0,
	})

	revenue, err := svc.CalculatePartnerRevenue(context.Background(), "t1", partner.ID)
	require.NoError(t, err)
	assert.Equal(t, 0.0, revenue.SubTenantRevenue)
	assert.Equal(t, 0.0, revenue.ShareAmount)
}

func TestCalculatePartnerRevenue_100PercentShare(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	partner, _ := svc.RegisterPartner(context.Background(), "t1", &CreatePartnerRequest{
		CompanyName: "All", ContactEmail: "a@e.com", RevenueSharePct: 100.0,
	})
	repo.subTenants["t1"] = []SubTenant{{WebhooksUsed: 1000}}

	revenue, err := svc.CalculatePartnerRevenue(context.Background(), "t1", partner.ID)
	require.NoError(t, err)
	assert.Equal(t, revenue.SubTenantRevenue, revenue.ShareAmount)
}

// --- UpdateBranding ---

func TestUpdateBranding_OnlyNonEmptyFields(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "Original", PrimaryColor: "#000",
	})

	config, err := svc.UpdateBranding(context.Background(), "t1", &UpdateBrandingRequest{
		BrandName: "Updated",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", config.BrandName)
	assert.Equal(t, "#000", config.PrimaryColor) // preserved
}

func TestUpdateBranding_AllFields(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "Old",
	})

	config, err := svc.UpdateBranding(context.Background(), "t1", &UpdateBrandingRequest{
		BrandName:      "New",
		LogoURL:        "https://logo.png",
		FaviconURL:     "https://fav.ico",
		PrimaryColor:   "#111",
		SecondaryColor: "#222",
		AccentColor:    "#333",
		CustomCSS:      "body{}",
	})
	require.NoError(t, err)
	assert.Equal(t, "New", config.BrandName)
	assert.Equal(t, "https://logo.png", config.LogoURL)
	assert.Equal(t, "#333", config.AccentColor)
}

// --- GeneratePreview ---

func TestGeneratePreview(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	svc.CreateWhitelabelConfig(context.Background(), "t1", &CreateWhitelabelRequest{
		CustomDomain: "test.com", BrandName: "Test",
	})

	preview, err := svc.GeneratePreview(context.Background(), "t1")
	require.NoError(t, err)
	assert.NotEmpty(t, preview.PreviewURL)
	assert.False(t, preview.GeneratedAt.IsZero())
}

// Ensure no unused imports
var _ = time.Now

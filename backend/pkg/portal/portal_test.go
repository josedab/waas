package portal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPortalRepository implements Repository for testing
type mockPortalRepository struct {
	portals    map[string]*PortalConfig
	tokens     map[string]*EmbedToken
	sessions   map[string]*PortalSession
	endpoints  []PortalEndpointView
	deliveries []PortalDeliveryView
	retried    map[string]bool
}

func newMockRepo() *mockPortalRepository {
	return &mockPortalRepository{
		portals:  make(map[string]*PortalConfig),
		tokens:   make(map[string]*EmbedToken),
		sessions: make(map[string]*PortalSession),
		retried:  make(map[string]bool),
	}
}

func (m *mockPortalRepository) CreatePortal(ctx context.Context, config *PortalConfig) error {
	m.portals[config.ID] = config
	return nil
}

func (m *mockPortalRepository) GetPortal(ctx context.Context, tenantID, portalID string) (*PortalConfig, error) {
	p, ok := m.portals[portalID]
	if !ok || p.TenantID != tenantID {
		return nil, fmt.Errorf("portal not found")
	}
	return p, nil
}

func (m *mockPortalRepository) GetPortalByTenantID(ctx context.Context, tenantID string) (*PortalConfig, error) {
	for _, p := range m.portals {
		if p.TenantID == tenantID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("portal not found for tenant")
}

func (m *mockPortalRepository) ListPortals(ctx context.Context, tenantID string) ([]PortalConfig, error) {
	var result []PortalConfig
	for _, p := range m.portals {
		if p.TenantID == tenantID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockPortalRepository) UpdatePortal(ctx context.Context, config *PortalConfig) error {
	m.portals[config.ID] = config
	return nil
}

func (m *mockPortalRepository) UpdatePortalByTenantID(ctx context.Context, tenantID string, config *PortalConfig) error {
	for id, p := range m.portals {
		if p.TenantID == tenantID {
			m.portals[id] = config
			return nil
		}
	}
	return fmt.Errorf("portal not found for tenant")
}

func (m *mockPortalRepository) DeletePortal(ctx context.Context, tenantID, portalID string) error {
	delete(m.portals, portalID)
	return nil
}

func (m *mockPortalRepository) CreateToken(ctx context.Context, token *EmbedToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockPortalRepository) GetToken(ctx context.Context, token string) (*EmbedToken, error) {
	t, ok := m.tokens[token]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockPortalRepository) ListTokens(ctx context.Context, tenantID, portalID string) ([]EmbedToken, error) {
	var result []EmbedToken
	for _, t := range m.tokens {
		if t.TenantID == tenantID && t.PortalID == portalID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockPortalRepository) RevokeToken(ctx context.Context, tenantID, tokenID string) error {
	for k, t := range m.tokens {
		if t.ID == tokenID && t.TenantID == tenantID {
			delete(m.tokens, k)
			return nil
		}
	}
	return fmt.Errorf("token not found")
}

func (m *mockPortalRepository) CreateSession(ctx context.Context, session *PortalSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockPortalRepository) ListSessions(ctx context.Context, tenantID string) ([]PortalSession, error) {
	var result []PortalSession
	for _, s := range m.sessions {
		if s.TenantID == tenantID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockPortalRepository) GetPortalEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]PortalEndpointView, int, error) {
	if offset >= len(m.endpoints) {
		return []PortalEndpointView{}, len(m.endpoints), nil
	}
	end := offset + limit
	if end > len(m.endpoints) {
		end = len(m.endpoints)
	}
	return m.endpoints[offset:end], len(m.endpoints), nil
}

func (m *mockPortalRepository) GetPortalDeliveries(ctx context.Context, tenantID string, filter DeliveryFilter, limit, offset int) ([]PortalDeliveryView, int, error) {
	var filtered []PortalDeliveryView
	for _, d := range m.deliveries {
		if filter.EndpointID != "" && d.EndpointID != filter.EndpointID {
			continue
		}
		if filter.Status == "success" && !d.Success {
			continue
		}
		if filter.Status == "failed" && d.Success {
			continue
		}
		filtered = append(filtered, d)
	}
	total := len(filtered)
	if offset >= total {
		return []PortalDeliveryView{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (m *mockPortalRepository) GetDelivery(ctx context.Context, tenantID, deliveryID string) (*PortalDeliveryView, error) {
	for _, d := range m.deliveries {
		if d.ID == deliveryID {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *mockPortalRepository) RetryDelivery(ctx context.Context, tenantID, deliveryID string) error {
	m.retried[deliveryID] = true
	return nil
}

func (m *mockPortalRepository) GetPortalStats(ctx context.Context, tenantID string) (*PortalStats, error) {
	return &PortalStats{
		TotalEndpoints:       len(m.endpoints),
		ActiveEndpoints:      len(m.endpoints),
		TotalDeliveries:      int64(len(m.deliveries)),
		SuccessfulDeliveries: 8,
		FailedDeliveries:     2,
		SuccessRate:          0.80,
		AvgLatencyMs:         120.5,
	}, nil
}

// --- Tests ---

func TestTokenGenerationAndValidation(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo)
	ctx := context.Background()

	// Create a portal first
	portal, err := service.CreatePortal(ctx, "tenant-1", &CreatePortalRequest{
		Name:           "Test Portal",
		AllowedOrigins: []string{"https://example.com"},
	})
	require.NoError(t, err)
	require.NotNil(t, portal)

	// Generate a token
	token, err := service.CreateEmbedToken(ctx, "tenant-1", &CreateTokenRequest{
		PortalID: portal.ID,
		Scopes:   []string{ScopeEndpointsRead, ScopeDeliveriesRead},
		TTLHours: 24,
	})
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.True(t, len(token.Token) > 0)
	assert.Equal(t, "wpt_", token.Token[:4])
	assert.Equal(t, "tenant-1", token.TenantID)
	assert.Equal(t, []string{ScopeEndpointsRead, ScopeDeliveriesRead}, token.Scopes)

	// Validate the token
	validated, err := service.ValidatePortalToken(ctx, token.Token)
	require.NoError(t, err)
	require.NotNil(t, validated)
	assert.Equal(t, token.ID, validated.ID)
	assert.Equal(t, token.TenantID, validated.TenantID)

	// Validate invalid token
	validated, err = service.ValidatePortalToken(ctx, "wpt_invalid")
	assert.Error(t, err)
	assert.Nil(t, validated)
	assert.Contains(t, err.Error(), "invalid token")

	// Validate expired token
	expiredToken := &EmbedToken{
		ID:        "expired-id",
		TenantID:  "tenant-1",
		PortalID:  portal.ID,
		Token:     "wpt_expired123",
		Scopes:    []string{ScopeEndpointsRead},
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	repo.tokens[expiredToken.Token] = expiredToken

	validated, err = service.ValidatePortalToken(ctx, expiredToken.Token)
	assert.Error(t, err)
	assert.Nil(t, validated)
	assert.Contains(t, err.Error(), "token expired")
}

func TestPortalConfigCRUD(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo)
	ctx := context.Background()

	// Create portal
	portal, err := service.CreatePortal(ctx, "tenant-1", &CreatePortalRequest{
		Name:           "My Portal",
		AllowedOrigins: []string{"https://app.example.com"},
		Branding: &Branding{
			PrimaryColor: "#FF0000",
			FontFamily:   "Inter",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "My Portal", portal.Name)
	assert.Equal(t, "#FF0000", portal.Branding.PrimaryColor)
	assert.True(t, portal.IsActive)

	// Get portal config by tenant ID
	config, err := service.GetPortalConfig(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, portal.ID, config.ID)

	// Update portal config
	isActive := false
	updated, err := service.UpdatePortalConfig(ctx, "tenant-1", &UpdatePortalConfigRequest{
		Name:     "Updated Portal",
		IsActive: &isActive,
		Branding: &Branding{
			PrimaryColor: "#00FF00",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Portal", updated.Name)
	assert.False(t, updated.IsActive)
	assert.Equal(t, "#00FF00", updated.Branding.PrimaryColor)

	// Update with nil repo
	nilService := NewService(nil)
	_, err = nilService.GetPortalConfig(ctx, "tenant-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestEmbedSnippetGeneration(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo)
	ctx := context.Background()

	// Create portal
	portal, err := service.CreatePortal(ctx, "tenant-1", &CreatePortalRequest{
		Name:           "Snippet Portal",
		AllowedOrigins: []string{"https://example.com"},
		Branding: &Branding{
			PrimaryColor: "#3B82F6",
		},
	})
	require.NoError(t, err)

	// Get full snippet
	snippet, err := service.GetEmbedSnippet(ctx, "tenant-1", portal.ID, "https://api.example.com")
	require.NoError(t, err)
	assert.Contains(t, snippet.HTML, "waas-portal")
	assert.Contains(t, snippet.React, "WaaSPortal")
	assert.Contains(t, snippet.IFrame, "iframe")

	// Generate by format - HTML
	code, err := service.GenerateEmbedSnippetForFormat(ctx, "tenant-1", portal.ID, "html", "https://api.example.com")
	require.NoError(t, err)
	assert.Contains(t, code, "waas-portal")

	// Generate by format - React
	code, err = service.GenerateEmbedSnippetForFormat(ctx, "tenant-1", portal.ID, "react", "https://api.example.com")
	require.NoError(t, err)
	assert.Contains(t, code, "WaaSPortal")

	// Generate by format - iframe
	code, err = service.GenerateEmbedSnippetForFormat(ctx, "tenant-1", portal.ID, "iframe", "https://api.example.com")
	require.NoError(t, err)
	assert.Contains(t, code, "iframe")
}

func TestPortalEndpointListing(t *testing.T) {
	repo := newMockRepo()
	repo.endpoints = []PortalEndpointView{
		{ID: "ep-1", URL: "https://hooks.example.com/a", Description: "Endpoint A", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ep-2", URL: "https://hooks.example.com/b", Description: "Endpoint B", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ep-3", URL: "https://hooks.example.com/c", Description: "Endpoint C", IsActive: false, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	service := NewService(repo)
	ctx := context.Background()

	// List all endpoints
	endpoints, total, err := service.GetPortalEndpoints(ctx, "tenant-1", 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, endpoints, 3)

	// List with pagination
	endpoints, total, err = service.GetPortalEndpoints(ctx, "tenant-1", 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, endpoints, 2)

	// List with offset
	endpoints, total, err = service.GetPortalEndpoints(ctx, "tenant-1", 50, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, endpoints, 1)

	// Default limit when 0
	endpoints, _, err = service.GetPortalEndpoints(ctx, "tenant-1", 0, 0)
	require.NoError(t, err)
	assert.Len(t, endpoints, 3)
}

func TestDeliveryRetryFromPortal(t *testing.T) {
	repo := newMockRepo()
	repo.deliveries = []PortalDeliveryView{
		{ID: "del-1", EndpointID: "ep-1", EventType: "user.created", StatusCode: 500, Success: false, Attempts: 3, LastAttemptAt: time.Now(), CreatedAt: time.Now()},
		{ID: "del-2", EndpointID: "ep-1", EventType: "user.updated", StatusCode: 200, Success: true, Attempts: 1, LastAttemptAt: time.Now(), CreatedAt: time.Now()},
	}
	service := NewService(repo)
	ctx := context.Background()

	// Retry failed delivery
	err := service.RetryPortalDelivery(ctx, "tenant-1", "del-1")
	require.NoError(t, err)
	assert.True(t, repo.retried["del-1"])

	// Retry successful delivery should fail
	err = service.RetryPortalDelivery(ctx, "tenant-1", "del-2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot retry successful delivery")

	// Retry non-existent delivery
	err = service.RetryPortalDelivery(ctx, "tenant-1", "del-999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delivery not found")

	// Retry with nil repo
	nilService := NewService(nil)
	err = nilService.RetryPortalDelivery(ctx, "tenant-1", "del-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestPortalDeliveriesWithFilter(t *testing.T) {
	repo := newMockRepo()
	repo.deliveries = []PortalDeliveryView{
		{ID: "del-1", EndpointID: "ep-1", Success: false, CreatedAt: time.Now()},
		{ID: "del-2", EndpointID: "ep-1", Success: true, CreatedAt: time.Now()},
		{ID: "del-3", EndpointID: "ep-2", Success: false, CreatedAt: time.Now()},
	}
	service := NewService(repo)
	ctx := context.Background()

	// No filter
	deliveries, total, err := service.GetPortalDeliveries(ctx, "tenant-1", DeliveryFilter{}, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, deliveries, 3)

	// Filter by endpoint
	deliveries, total, err = service.GetPortalDeliveries(ctx, "tenant-1", DeliveryFilter{EndpointID: "ep-1"}, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, deliveries, 2)

	// Filter by status
	deliveries, total, err = service.GetPortalDeliveries(ctx, "tenant-1", DeliveryFilter{Status: "failed"}, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, deliveries, 2)
}

func TestPortalStats(t *testing.T) {
	repo := newMockRepo()
	repo.endpoints = []PortalEndpointView{
		{ID: "ep-1"},
		{ID: "ep-2"},
	}
	repo.deliveries = []PortalDeliveryView{
		{ID: "del-1", Success: true},
		{ID: "del-2", Success: false},
	}
	service := NewService(repo)
	ctx := context.Background()

	stats, err := service.GetPortalStats(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalEndpoints)
	assert.Equal(t, int64(2), stats.TotalDeliveries)
	assert.Equal(t, 0.80, stats.SuccessRate)
}

func TestHasScope(t *testing.T) {
	token := &EmbedToken{
		Scopes: []string{ScopeEndpointsRead, ScopeDeliveriesRead, ScopeDeliveriesRetry},
	}

	assert.True(t, HasScope(token, ScopeEndpointsRead))
	assert.True(t, HasScope(token, ScopeDeliveriesRead))
	assert.True(t, HasScope(token, ScopeDeliveriesRetry))
	assert.False(t, HasScope(token, ScopeEndpointsWrite))
	assert.False(t, HasScope(token, ScopeAnalyticsRead))
	assert.False(t, HasScope(token, "unknown:scope"))
}

func TestValidScopes(t *testing.T) {
	assert.True(t, IsValidScope(ScopeEndpointsRead))
	assert.True(t, IsValidScope(ScopeEndpointsWrite))
	assert.True(t, IsValidScope(ScopeDeliveriesRead))
	assert.True(t, IsValidScope(ScopeDeliveriesRetry))
	assert.True(t, IsValidScope(ScopeAnalyticsRead))
	assert.True(t, IsValidScope(ScopeTestSend))
	assert.True(t, IsValidScope(ScopeSLARead))
	assert.False(t, IsValidScope("invalid:scope"))
	assert.False(t, IsValidScope(""))
}

func TestDefaultFeatureFlags(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo)
	ctx := context.Background()

	portal, err := service.CreatePortal(ctx, "tenant-1", &CreatePortalRequest{
		Name:           "Default Features Portal",
		AllowedOrigins: []string{"https://example.com"},
	})
	require.NoError(t, err)
	require.NotNil(t, portal.Features)
	assert.True(t, portal.Features.ShowEndpoints)
	assert.True(t, portal.Features.ShowDeliveries)
	assert.True(t, portal.Features.ShowAnalytics)
	assert.False(t, portal.Features.AllowCreate)
	assert.False(t, portal.Features.AllowDelete)
}

func TestTokenDefaultTTL(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo)
	ctx := context.Background()

	portal, err := service.CreatePortal(ctx, "tenant-1", &CreatePortalRequest{
		Name:           "TTL Portal",
		AllowedOrigins: []string{"https://example.com"},
	})
	require.NoError(t, err)

	// TTL of 0 should default to 24 hours
	token, err := service.CreateEmbedToken(ctx, "tenant-1", &CreateTokenRequest{
		PortalID: portal.ID,
		Scopes:   []string{ScopeEndpointsRead},
		TTLHours: 0,
	})
	require.NoError(t, err)
	expectedExpiry := time.Now().Add(24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, token.ExpiresAt, 5*time.Second)
}

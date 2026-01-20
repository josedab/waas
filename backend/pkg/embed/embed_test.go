package embed

import (
	"context"
	"testing"
	"time"
)

func TestPermissions(t *testing.T) {
	permissions := AllPermissions()
	
	if len(permissions) == 0 {
		t.Error("expected non-empty permissions list")
	}
	
	expectedPerms := []Permission{
		PermissionReadDeliveries,
		PermissionReadEndpoints,
		PermissionReadMetrics,
		PermissionReadEvents,
		PermissionReadActivity,
		PermissionReadErrors,
	}
	
	if len(permissions) != len(expectedPerms) {
		t.Errorf("expected %d permissions, got %d", len(expectedPerms), len(permissions))
	}
}

func TestComponents(t *testing.T) {
	components := AllComponents()
	
	if len(components) == 0 {
		t.Error("expected non-empty components list")
	}
	
	expectedComps := []EmbedComponent{
		ComponentDeliveryStats,
		ComponentActivityFeed,
		ComponentSuccessRateChart,
		ComponentLatencyChart,
		ComponentEndpointList,
		ComponentEventLog,
		ComponentErrorSummary,
		ComponentVolumeChart,
	}
	
	if len(components) != len(expectedComps) {
		t.Errorf("expected %d components, got %d", len(expectedComps), len(components))
	}
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	
	if theme == nil {
		t.Fatal("expected non-nil theme")
	}
	
	if theme.Mode == "" {
		t.Error("expected mode to be set")
	}
	
	if theme.PrimaryColor == "" {
		t.Error("expected primary color to be set")
	}
	
	if theme.FontFamily == "" {
		t.Error("expected font family to be set")
	}
}

func TestEmbedToken(t *testing.T) {
	token := &EmbedToken{
		ID:       "token-1",
		TenantID: "tenant-1",
		Name:     "Production Dashboard",
		Token:    "embed_live_abc123",
		Permissions: []Permission{
			PermissionReadDeliveries,
			PermissionReadMetrics,
		},
		Scopes: EmbedScopes{
			EndpointIDs: []string{"ep-1", "ep-2"},
			TimeRange:   "7d",
		},
		Theme:          DefaultTheme(),
		AllowedOrigins: []string{"https://app.example.com"},
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	if token.Name != "Production Dashboard" {
		t.Errorf("expected name 'Production Dashboard', got %s", token.Name)
	}
	
	if len(token.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(token.Permissions))
	}
	
	if !token.IsActive {
		t.Error("expected token to be active")
	}
}

func TestEmbedScopes(t *testing.T) {
	scopes := EmbedScopes{
		EndpointIDs: []string{"ep-1", "ep-2", "ep-3"},
		EventTypes:  []string{"delivery.succeeded", "delivery.failed"},
		CustomerID:  "customer-123",
		TimeRange:   "30d",
	}
	
	if len(scopes.EndpointIDs) != 3 {
		t.Errorf("expected 3 endpoint IDs, got %d", len(scopes.EndpointIDs))
	}
	
	if scopes.TimeRange != "30d" {
		t.Errorf("expected time range '30d', got %s", scopes.TimeRange)
	}
}

func TestThemeConfig(t *testing.T) {
	theme := &ThemeConfig{
		Mode:            "dark",
		PrimaryColor:    "#8b5cf6",
		BackgroundColor: "#1f2937",
		TextColor:       "#f9fafb",
		BorderRadius:    "12px",
		FontFamily:      "Roboto, sans-serif",
		CustomCSS:       ".widget { padding: 20px; }",
		Variables: map[string]string{
			"--accent-color": "#10b981",
		},
	}
	
	if theme.Mode != "dark" {
		t.Errorf("expected mode 'dark', got %s", theme.Mode)
	}
	
	if theme.CustomCSS == "" {
		t.Error("expected custom CSS to be set")
	}
}

func TestComponentConfig(t *testing.T) {
	config := ComponentConfig{
		Component:   ComponentDeliveryStats,
		Width:       "100%",
		Height:      "300px",
		Title:       "Delivery Statistics",
		ShowHeader:  true,
		RefreshRate: 30,
		Options: map[string]interface{}{
			"show_trend": true,
		},
	}
	
	if config.Component != ComponentDeliveryStats {
		t.Errorf("expected component delivery_stats, got %s", config.Component)
	}
	
	if config.RefreshRate != 30 {
		t.Errorf("expected refresh rate 30, got %d", config.RefreshRate)
	}
}

func TestEmbedSession(t *testing.T) {
	session := &EmbedSession{
		ID:        "session-1",
		TokenID:   "token-1",
		TenantID:  "tenant-1",
		Origin:    "https://app.example.com",
		UserAgent: "Mozilla/5.0",
		IP:        "192.168.1.1",
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	
	if session.Origin == "" {
		t.Error("expected origin to be set")
	}
	
	if session.IP == "" {
		t.Error("expected IP to be set")
	}
}

func TestDeliveryStats(t *testing.T) {
	stats := &DeliveryStats{
		TotalDeliveries: 10000,
		Successful:      9500,
		Failed:          400,
		Pending:         100,
		SuccessRate:     0.95,
		AvgLatencyMs:    125,
		Period:          "24h",
		UpdatedAt:       time.Now(),
	}
	
	if stats.SuccessRate != 0.95 {
		t.Errorf("expected success rate 0.95, got %f", stats.SuccessRate)
	}
	
	total := stats.Successful + stats.Failed + stats.Pending
	if total != stats.TotalDeliveries {
		t.Error("sum of successful, failed, pending should equal total")
	}
}

func TestActivityItem(t *testing.T) {
	item := &ActivityItem{
		ID:         "activity-1",
		Type:       "delivery_failed",
		Message:    "Webhook delivery failed after 3 retries",
		Severity:   "error",
		EndpointID: "ep-1",
		Details: map[string]interface{}{
			"status_code": 503,
			"attempts":    3,
		},
		Timestamp: time.Now(),
	}
	
	if item.Severity != "error" {
		t.Errorf("expected severity 'error', got %s", item.Severity)
	}
	
	if item.Message == "" {
		t.Error("expected message to be set")
	}
}

func TestChartData(t *testing.T) {
	chart := &ChartData{
		Title: "Success Rate (24h)",
		Type:  "line",
		Series: []ChartSeries{
			{
				Name:  "Success Rate",
				Color: "#10b981",
				Data: []ChartDataPoint{
					{Timestamp: time.Now().Add(-2 * time.Hour), Value: 0.94},
					{Timestamp: time.Now().Add(-1 * time.Hour), Value: 0.96},
					{Timestamp: time.Now(), Value: 0.95},
				},
			},
		},
		XAxis:  "Time",
		YAxis:  "Rate (%)",
		Period: "24h",
	}
	
	if chart.Type != "line" {
		t.Errorf("expected chart type 'line', got %s", chart.Type)
	}
	
	if len(chart.Series) != 1 {
		t.Errorf("expected 1 series, got %d", len(chart.Series))
	}
	
	if len(chart.Series[0].Data) != 3 {
		t.Errorf("expected 3 data points, got %d", len(chart.Series[0].Data))
	}
}

func TestErrorSummary(t *testing.T) {
	summary := &ErrorSummary{
		TotalErrors: 150,
		ByCategory: map[string]int64{
			"timeout":      50,
			"http_error":   80,
			"dns_failure":  20,
		},
		ByEndpoint: map[string]int64{
			"ep-1": 100,
			"ep-2": 50,
		},
		TopErrors: []ErrorDetail{
			{Message: "Connection timeout", Count: 50, LastSeen: time.Now()},
			{Message: "503 Service Unavailable", Count: 40, LastSeen: time.Now()},
		},
		Period: "7d",
	}
	
	if summary.TotalErrors != 150 {
		t.Errorf("expected 150 total errors, got %d", summary.TotalErrors)
	}
	
	if len(summary.TopErrors) != 2 {
		t.Errorf("expected 2 top errors, got %d", len(summary.TopErrors))
	}
}

func TestCreateTokenRequest(t *testing.T) {
	req := &CreateTokenRequest{
		Name: "Dashboard Token",
		Permissions: []Permission{
			PermissionReadDeliveries,
			PermissionReadMetrics,
		},
		Scopes: EmbedScopes{
			TimeRange: "7d",
		},
		Theme:          DefaultTheme(),
		ExpiresIn:      "90d",
		AllowedOrigins: []string{"https://app.example.com"},
	}
	
	if req.Name == "" {
		t.Error("expected name to be set")
	}
	
	if len(req.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(req.Permissions))
	}
}

func TestUpdateTokenRequest(t *testing.T) {
	name := "Updated Token"
	isActive := false
	
	req := &UpdateTokenRequest{
		Name:           &name,
		Permissions:    []Permission{PermissionReadDeliveries},
		IsActive:       &isActive,
		AllowedOrigins: []string{"https://new.example.com"},
	}
	
	if *req.Name != "Updated Token" {
		t.Errorf("expected name 'Updated Token', got %s", *req.Name)
	}
	
	if *req.IsActive {
		t.Error("expected is_active to be false")
	}
}

func TestServiceWithMockRepo(t *testing.T) {
	mockRepo := &mockEmbedRepository{}
	service := NewService(mockRepo)
	
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	
	ctx := context.Background()
	
	// Test creating a token
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Permissions: []Permission{PermissionReadDeliveries},
		Scopes:      EmbedScopes{TimeRange: "24h"},
	}
	
	token, err := service.CreateToken(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if token == nil {
		t.Fatal("expected non-nil token")
	}
	
	if token.Name != "Test Token" {
		t.Errorf("expected name 'Test Token', got %s", token.Name)
	}
}

// Mock repository for testing
type mockEmbedRepository struct {
	tokens   map[string]*EmbedToken
	sessions map[string]*EmbedSession
}

func (m *mockEmbedRepository) CreateToken(ctx context.Context, token *EmbedToken) error {
	if m.tokens == nil {
		m.tokens = make(map[string]*EmbedToken)
	}
	m.tokens[token.ID] = token
	return nil
}

func (m *mockEmbedRepository) GetToken(ctx context.Context, tenantID, tokenID string) (*EmbedToken, error) {
	if m.tokens == nil {
		return nil, nil
	}
	t, ok := m.tokens[tokenID]
	if !ok || t.TenantID != tenantID {
		return nil, nil
	}
	return t, nil
}

func (m *mockEmbedRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*EmbedToken, error) {
	for _, t := range m.tokens {
		if t.Token == tokenValue {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockEmbedRepository) ListTokens(ctx context.Context, tenantID string, limit, offset int) ([]EmbedToken, int, error) {
	var tokens []EmbedToken
	for _, t := range m.tokens {
		if t.TenantID == tenantID {
			tokens = append(tokens, *t)
		}
	}
	return tokens, len(tokens), nil
}

func (m *mockEmbedRepository) UpdateToken(ctx context.Context, token *EmbedToken) error {
	if m.tokens == nil {
		m.tokens = make(map[string]*EmbedToken)
	}
	m.tokens[token.ID] = token
	return nil
}

func (m *mockEmbedRepository) DeleteToken(ctx context.Context, tenantID, tokenID string) error {
	delete(m.tokens, tokenID)
	return nil
}

func (m *mockEmbedRepository) CreateSession(ctx context.Context, session *EmbedSession) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*EmbedSession)
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockEmbedRepository) UpdateSession(ctx context.Context, sessionID string) error {
	if m.sessions != nil {
		if s, ok := m.sessions[sessionID]; ok {
			s.LastSeen = time.Now()
		}
	}
	return nil
}

func (m *mockEmbedRepository) GetSessionsByToken(ctx context.Context, tokenID string, limit int) ([]EmbedSession, error) {
	var sessions []EmbedSession
	for _, s := range m.sessions {
		if s.TokenID == tokenID {
			sessions = append(sessions, *s)
		}
	}
	return sessions, nil
}

func (m *mockEmbedRepository) GetDeliveryStats(ctx context.Context, tenantID string, scopes EmbedScopes) (*DeliveryStats, error) {
	return &DeliveryStats{
		TotalDeliveries: 1000,
		Successful:      950,
		Failed:          50,
		SuccessRate:     0.95,
		AvgLatencyMs:    100,
		Period:          scopes.TimeRange,
		UpdatedAt:       time.Now(),
	}, nil
}

func (m *mockEmbedRepository) GetActivityFeed(ctx context.Context, tenantID string, scopes EmbedScopes, limit int) ([]ActivityItem, error) {
	return []ActivityItem{}, nil
}

func (m *mockEmbedRepository) GetChartData(ctx context.Context, tenantID, chartType string, scopes EmbedScopes) (*ChartData, error) {
	return &ChartData{
		Title:  chartType,
		Type:   "line",
		Series: []ChartSeries{},
		Period: scopes.TimeRange,
	}, nil
}

func (m *mockEmbedRepository) GetErrorSummary(ctx context.Context, tenantID string, scopes EmbedScopes) (*ErrorSummary, error) {
	return &ErrorSummary{
		TotalErrors: 0,
		ByCategory:  make(map[string]int64),
		ByEndpoint:  make(map[string]int64),
		TopErrors:   []ErrorDetail{},
		Period:      scopes.TimeRange,
	}, nil
}

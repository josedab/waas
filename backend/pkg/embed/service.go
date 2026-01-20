package embed

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service manages embeddable analytics tokens and data
type Service struct {
	repo Repository
}

// NewService creates a new embed service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateToken creates a new embed token
func (s *Service) CreateToken(ctx context.Context, tenantID string, req *CreateTokenRequest) (*EmbedToken, error) {
	// Validate permissions
	for _, perm := range req.Permissions {
		if !isValidPermission(perm) {
			return nil, fmt.Errorf("invalid permission: %s", perm)
		}
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		duration, err := parseDuration(req.ExpiresIn)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_in: %w", err)
		}
		exp := time.Now().Add(duration)
		expiresAt = &exp
	}

	theme := req.Theme
	if theme == nil {
		theme = DefaultTheme()
	}

	embedToken := &EmbedToken{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		Token:          "emb_" + token,
		Permissions:    req.Permissions,
		Scopes:         req.Scopes,
		Theme:          theme,
		ExpiresAt:      expiresAt,
		AllowedOrigins: req.AllowedOrigins,
		Metadata:       req.Metadata,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateToken(ctx, embedToken); err != nil {
		return nil, err
	}

	return embedToken, nil
}

// GetToken retrieves an embed token by ID
func (s *Service) GetToken(ctx context.Context, tenantID, tokenID string) (*EmbedToken, error) {
	return s.repo.GetToken(ctx, tenantID, tokenID)
}

// GetTokenByValue retrieves an embed token by its value
func (s *Service) GetTokenByValue(ctx context.Context, tokenValue string) (*EmbedToken, error) {
	return s.repo.GetTokenByValue(ctx, tokenValue)
}

// ListTokens lists embed tokens for a tenant
func (s *Service) ListTokens(ctx context.Context, tenantID string, limit, offset int) ([]EmbedToken, int, error) {
	return s.repo.ListTokens(ctx, tenantID, limit, offset)
}

// UpdateToken updates an embed token
func (s *Service) UpdateToken(ctx context.Context, tenantID, tokenID string, req *UpdateTokenRequest) (*EmbedToken, error) {
	token, err := s.repo.GetToken(ctx, tenantID, tokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, fmt.Errorf("token not found")
	}

	if req.Name != nil {
		token.Name = *req.Name
	}
	if len(req.Permissions) > 0 {
		for _, perm := range req.Permissions {
			if !isValidPermission(perm) {
				return nil, fmt.Errorf("invalid permission: %s", perm)
			}
		}
		token.Permissions = req.Permissions
	}
	if req.Scopes != nil {
		token.Scopes = *req.Scopes
	}
	if req.Theme != nil {
		token.Theme = req.Theme
	}
	if len(req.AllowedOrigins) > 0 {
		token.AllowedOrigins = req.AllowedOrigins
	}
	if req.IsActive != nil {
		token.IsActive = *req.IsActive
	}
	if req.Metadata != nil {
		token.Metadata = req.Metadata
	}

	token.UpdatedAt = time.Now()

	if err := s.repo.UpdateToken(ctx, token); err != nil {
		return nil, err
	}

	return token, nil
}

// DeleteToken deletes an embed token
func (s *Service) DeleteToken(ctx context.Context, tenantID, tokenID string) error {
	return s.repo.DeleteToken(ctx, tenantID, tokenID)
}

// RotateToken generates a new token value
func (s *Service) RotateToken(ctx context.Context, tenantID, tokenID string) (*EmbedToken, string, error) {
	token, err := s.repo.GetToken(ctx, tenantID, tokenID)
	if err != nil {
		return nil, "", err
	}
	if token == nil {
		return nil, "", fmt.Errorf("token not found")
	}

	newTokenValue, err := generateSecureToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	token.Token = "emb_" + newTokenValue
	token.UpdatedAt = time.Now()

	if err := s.repo.UpdateToken(ctx, token); err != nil {
		return nil, "", err
	}

	return token, token.Token, nil
}

// ValidateToken validates a token and returns its configuration
func (s *Service) ValidateToken(ctx context.Context, tokenValue, origin string) (*EmbedToken, error) {
	token, err := s.repo.GetTokenByValue(ctx, tokenValue)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, fmt.Errorf("invalid token")
	}

	if !token.IsActive {
		return nil, fmt.Errorf("token is inactive")
	}

	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	// Validate origin if origins are specified
	if len(token.AllowedOrigins) > 0 && origin != "" {
		if !isOriginAllowed(origin, token.AllowedOrigins) {
			return nil, fmt.Errorf("origin not allowed")
		}
	}

	return token, nil
}

// HasPermission checks if a token has a specific permission
func (s *Service) HasPermission(token *EmbedToken, permission Permission) bool {
	for _, p := range token.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// RecordSession records an embed session
func (s *Service) RecordSession(ctx context.Context, tokenID, tenantID, origin, userAgent, ip string) error {
	session := &EmbedSession{
		ID:        uuid.New().String(),
		TokenID:   tokenID,
		TenantID:  tenantID,
		Origin:    origin,
		UserAgent: userAgent,
		IP:        ip,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	return s.repo.CreateSession(ctx, session)
}

// GetDeliveryStats retrieves delivery statistics
func (s *Service) GetDeliveryStats(ctx context.Context, tenantID string, scopes EmbedScopes) (*DeliveryStats, error) {
	return s.repo.GetDeliveryStats(ctx, tenantID, scopes)
}

// GetActivityFeed retrieves activity feed items
func (s *Service) GetActivityFeed(ctx context.Context, tenantID string, scopes EmbedScopes, limit int) ([]ActivityItem, error) {
	return s.repo.GetActivityFeed(ctx, tenantID, scopes, limit)
}

// GetChartData retrieves chart data for a specific chart type
func (s *Service) GetChartData(ctx context.Context, tenantID string, chartType string, scopes EmbedScopes) (*ChartData, error) {
	return s.repo.GetChartData(ctx, tenantID, chartType, scopes)
}

// GetErrorSummary retrieves error summary data
func (s *Service) GetErrorSummary(ctx context.Context, tenantID string, scopes EmbedScopes) (*ErrorSummary, error) {
	return s.repo.GetErrorSummary(ctx, tenantID, scopes)
}

// GetAvailableComponents returns available embed components
func (s *Service) GetAvailableComponents() []ComponentConfig {
	return []ComponentConfig{
		{Component: ComponentDeliveryStats, Title: "Delivery Statistics", ShowHeader: true, RefreshRate: 30},
		{Component: ComponentActivityFeed, Title: "Activity Feed", ShowHeader: true, RefreshRate: 10},
		{Component: ComponentSuccessRateChart, Title: "Success Rate", ShowHeader: true, RefreshRate: 60},
		{Component: ComponentLatencyChart, Title: "Latency", ShowHeader: true, RefreshRate: 60},
		{Component: ComponentEndpointList, Title: "Endpoints", ShowHeader: true, RefreshRate: 60},
		{Component: ComponentEventLog, Title: "Event Log", ShowHeader: true, RefreshRate: 15},
		{Component: ComponentErrorSummary, Title: "Error Summary", ShowHeader: true, RefreshRate: 30},
		{Component: ComponentVolumeChart, Title: "Volume", ShowHeader: true, RefreshRate: 60},
	}
}

func isValidPermission(p Permission) bool {
	for _, valid := range AllPermissions() {
		if valid == p {
			return true
		}
	}
	return false
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(a, "*.") {
			suffix := a[1:] // Remove *
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}
	return false
}

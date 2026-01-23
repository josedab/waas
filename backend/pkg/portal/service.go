package portal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides embeddable portal management functionality
type Service struct {
	repo Repository
}

// NewService creates a new portal service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreatePortal creates a new embeddable portal configuration
func (s *Service) CreatePortal(ctx context.Context, tenantID string, req *CreatePortalRequest) (*PortalConfig, error) {
	if req.Features == nil {
		req.Features = &FeatureFlags{
			ShowEndpoints:  true,
			ShowDeliveries: true,
			ShowAnalytics:  true,
			ShowTestSender: true,
			AllowCreate:    false,
			AllowDelete:    false,
			ShowSLA:        false,
		}
	}

	config := &PortalConfig{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		Branding:       req.Branding,
		AllowedOrigins: req.AllowedOrigins,
		Features:       req.Features,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreatePortal(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create portal: %w", err)
	}

	return config, nil
}

// GetPortal retrieves a portal configuration
func (s *Service) GetPortal(ctx context.Context, tenantID, portalID string) (*PortalConfig, error) {
	return s.repo.GetPortal(ctx, tenantID, portalID)
}

// ListPortals lists all portal configurations for a tenant
func (s *Service) ListPortals(ctx context.Context, tenantID string) ([]PortalConfig, error) {
	return s.repo.ListPortals(ctx, tenantID)
}

// UpdatePortal updates a portal configuration
func (s *Service) UpdatePortal(ctx context.Context, tenantID, portalID string, req *CreatePortalRequest) (*PortalConfig, error) {
	config, err := s.repo.GetPortal(ctx, tenantID, portalID)
	if err != nil {
		return nil, err
	}

	config.Name = req.Name
	config.Branding = req.Branding
	config.AllowedOrigins = req.AllowedOrigins
	config.Features = req.Features
	config.UpdatedAt = time.Now()

	if err := s.repo.UpdatePortal(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update portal: %w", err)
	}

	return config, nil
}

// DeletePortal deletes a portal configuration
func (s *Service) DeletePortal(ctx context.Context, tenantID, portalID string) error {
	return s.repo.DeletePortal(ctx, tenantID, portalID)
}

// CreateEmbedToken creates a scoped access token for the portal
func (s *Service) CreateEmbedToken(ctx context.Context, tenantID string, req *CreateTokenRequest) (*EmbedToken, error) {
	// Verify portal exists
	if _, err := s.repo.GetPortal(ctx, tenantID, req.PortalID); err != nil {
		return nil, fmt.Errorf("portal not found: %w", err)
	}

	if req.TTLHours <= 0 {
		req.TTLHours = 24
	}

	token := &EmbedToken{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		PortalID:  req.PortalID,
		Token:     generateEmbedToken(),
		Scopes:    req.Scopes,
		ExpiresAt: time.Now().Add(time.Duration(req.TTLHours) * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateToken(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	return token, nil
}

// ListTokens lists all tokens for a portal
func (s *Service) ListTokens(ctx context.Context, tenantID, portalID string) ([]EmbedToken, error) {
	return s.repo.ListTokens(ctx, tenantID, portalID)
}

// RevokeToken revokes an embed token
func (s *Service) RevokeToken(ctx context.Context, tenantID, tokenID string) error {
	return s.repo.RevokeToken(ctx, tenantID, tokenID)
}

// GetEmbedSnippet generates embed code snippets for a portal
func (s *Service) GetEmbedSnippet(ctx context.Context, tenantID, portalID string, apiURL string) (*EmbedSnippet, error) {
	config, err := s.repo.GetPortal(ctx, tenantID, portalID)
	if err != nil {
		return nil, err
	}

	snippet := &EmbedSnippet{
		HTML: fmt.Sprintf(`<!-- WaaS Embeddable Portal -->
<div id="waas-portal"></div>
<script src="%s/embed/waas-portal.js"></script>
<script>
  WaaSPortal.init({
    containerId: 'waas-portal',
    portalId: '%s',
    apiUrl: '%s',
    theme: {
      primaryColor: '%s',
    }
  });
</script>`, apiURL, config.ID, apiURL, safeColor(config.Branding)),

		React: fmt.Sprintf(`import { WaaSPortal } from '@waas/react-portal';

function WebhookManager() {
  return (
    <WaaSPortal
      portalId="%s"
      apiUrl="%s"
      token={embedToken}
    />
  );
}`, config.ID, apiURL),

		IFrame: fmt.Sprintf(`<iframe
  src="%s/portal/%s?theme=light"
  width="100%%"
  height="600"
  frameborder="0"
  allow="clipboard-write"
></iframe>`, apiURL, config.ID),
	}

	return snippet, nil
}

// ListSessions lists active portal sessions
func (s *Service) ListSessions(ctx context.Context, tenantID string) ([]PortalSession, error) {
	return s.repo.ListSessions(ctx, tenantID)
}

func generateEmbedToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "wpt_" + hex.EncodeToString(b)
}

func safeColor(branding *Branding) string {
	if branding != nil && branding.PrimaryColor != "" {
		return branding.PrimaryColor
	}
	return "#3B82F6"
}

// ValidatePortalToken validates a portal token and returns the token details
func (s *Service) ValidatePortalToken(ctx context.Context, tokenValue string) (*EmbedToken, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	token, err := s.repo.GetToken(ctx, tokenValue)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("invalid token")
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}
	return token, nil
}

// GetPortalEndpoints lists endpoints visible in portal
func (s *Service) GetPortalEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]PortalEndpointView, int, error) {
	if s.repo == nil {
		return nil, 0, fmt.Errorf("repository not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetPortalEndpoints(ctx, tenantID, limit, offset)
}

// GetPortalDeliveries returns delivery history for portal
func (s *Service) GetPortalDeliveries(ctx context.Context, tenantID string, filter DeliveryFilter, limit, offset int) ([]PortalDeliveryView, int, error) {
	if s.repo == nil {
		return nil, 0, fmt.Errorf("repository not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetPortalDeliveries(ctx, tenantID, filter, limit, offset)
}

// RetryPortalDelivery retries a failed delivery from portal
func (s *Service) RetryPortalDelivery(ctx context.Context, tenantID, deliveryID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}

	delivery, err := s.repo.GetDelivery(ctx, tenantID, deliveryID)
	if err != nil {
		return fmt.Errorf("failed to get delivery: %w", err)
	}
	if delivery == nil {
		return fmt.Errorf("delivery not found")
	}
	if delivery.Success {
		return fmt.Errorf("cannot retry successful delivery")
	}

	return s.repo.RetryDelivery(ctx, tenantID, deliveryID)
}

// GetPortalConfig gets the portal configuration for a tenant
func (s *Service) GetPortalConfig(ctx context.Context, tenantID string) (*PortalConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetPortalByTenantID(ctx, tenantID)
}

// UpdatePortalConfig updates the portal configuration for a tenant
func (s *Service) UpdatePortalConfig(ctx context.Context, tenantID string, req *UpdatePortalConfigRequest) (*PortalConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	config, err := s.repo.GetPortalByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("portal not found: %w", err)
	}

	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Branding != nil {
		config.Branding = req.Branding
	}
	if req.AllowedOrigins != nil {
		config.AllowedOrigins = req.AllowedOrigins
	}
	if req.Features != nil {
		config.Features = req.Features
	}
	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}
	config.UpdatedAt = time.Now()

	if err := s.repo.UpdatePortalByTenantID(ctx, tenantID, config); err != nil {
		return nil, fmt.Errorf("failed to update portal config: %w", err)
	}

	return config, nil
}

// GetPortalStats gets usage statistics for the portal
func (s *Service) GetPortalStats(ctx context.Context, tenantID string) (*PortalStats, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetPortalStats(ctx, tenantID)
}

// GenerateEmbedSnippetForFormat generates embed code in a specific format
func (s *Service) GenerateEmbedSnippetForFormat(ctx context.Context, tenantID, portalID, format, apiURL string) (string, error) {
	snippet, err := s.GetEmbedSnippet(ctx, tenantID, portalID, apiURL)
	if err != nil {
		return "", err
	}

	switch format {
	case "react":
		return snippet.React, nil
	case "iframe":
		return snippet.IFrame, nil
	default:
		return snippet.HTML, nil
	}
}

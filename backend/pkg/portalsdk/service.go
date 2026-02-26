package portalsdk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Service provides portal SDK business logic.
type Service struct {
	repo   Repository
	logger *utils.Logger
}

// NewService creates a new portal SDK service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, logger: utils.NewLogger("portalsdk-service")}
}

// CreateConfig creates a new portal configuration.
func (s *Service) CreateConfig(ctx context.Context, tenantID string, req *CreatePortalConfigRequest) (*PortalConfig, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("portal config name is required")
	}
	if len(req.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("at least one allowed origin is required")
	}

	config := &PortalConfig{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		AllowedOrigins: req.AllowedOrigins,
		Components:     req.Components,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if len(config.Components) == 0 {
		config.Components = defaultComponents()
	} else {
		if err := validateComponents(config.Components); err != nil {
			return nil, err
		}
	}

	if req.Theme != nil {
		config.Theme = *req.Theme
	} else {
		config.Theme = defaultTheme()
	}

	if req.Features != nil {
		config.Features = *req.Features
	} else {
		config.Features = defaultFeatures()
	}

	if req.Branding != nil {
		config.Branding = *req.Branding
	}

	config.CustomCSS = req.CustomCSS

	if s.repo != nil {
		if err := s.repo.CreateConfig(ctx, config); err != nil {
			return nil, fmt.Errorf("failed to create portal config: %w", err)
		}
	}

	return config, nil
}

// GetConfig retrieves a portal configuration.
func (s *Service) GetConfig(ctx context.Context, tenantID, configID string) (*PortalConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetConfig(ctx, tenantID, configID)
}

// ListConfigs returns all portal configurations for a tenant.
func (s *Service) ListConfigs(ctx context.Context, tenantID string) ([]PortalConfig, error) {
	if s.repo == nil {
		return []PortalConfig{}, nil
	}
	return s.repo.ListConfigs(ctx, tenantID)
}

// UpdateConfig updates an existing portal configuration.
func (s *Service) UpdateConfig(ctx context.Context, tenantID, configID string, req *UpdatePortalConfigRequest) (*PortalConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	config, err := s.repo.GetConfig(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		config.Name = req.Name
	}
	if len(req.AllowedOrigins) > 0 {
		config.AllowedOrigins = req.AllowedOrigins
	}
	if len(req.Components) > 0 {
		config.Components = req.Components
	}
	if req.Theme != nil {
		config.Theme = *req.Theme
	}
	if req.Features != nil {
		config.Features = *req.Features
	}
	if req.Branding != nil {
		config.Branding = *req.Branding
	}
	if req.CustomCSS != "" {
		config.CustomCSS = req.CustomCSS
	}
	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}
	config.UpdatedAt = time.Now()

	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return config, nil
}

// DeleteConfig removes a portal configuration.
func (s *Service) DeleteConfig(ctx context.Context, tenantID, configID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.DeleteConfig(ctx, tenantID, configID)
}

// CreateSession creates an authenticated portal session for a customer.
func (s *Service) CreateSession(ctx context.Context, tenantID string, req *CreateSessionRequest) (*PortalSession, error) {
	if req.ConfigID == "" || req.CustomerID == "" {
		return nil, fmt.Errorf("config_id and customer_id are required")
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	expiresIn := 24 * time.Hour
	if req.ExpiresIn != "" {
		d, err := time.ParseDuration(req.ExpiresIn)
		if err == nil && d > 0 {
			expiresIn = d
		}
	}

	if len(req.Permissions) == 0 {
		req.Permissions = defaultPermissions()
	}

	session := &PortalSession{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		ConfigID:     req.ConfigID,
		CustomerID:   req.CustomerID,
		Token:        token,
		Permissions:  req.Permissions,
		Status:       SessionStatusActive,
		ExpiresAt:    time.Now().Add(expiresIn),
		CreatedAt:    time.Now(),
		LastAccessAt: time.Now(),
	}

	if s.repo != nil {
		if err := s.repo.CreateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	return session, nil
}

// ValidateSession validates a portal session token and checks expiration.
func (s *Service) ValidateSession(ctx context.Context, token string) (*PortalSession, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	session, err := s.repo.GetSession(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid session token")
	}

	if session.Status != SessionStatusActive {
		return nil, fmt.Errorf("session is %s", session.Status)
	}

	if time.Now().After(session.ExpiresAt) {
		session.Status = SessionStatusExpired
		if err := s.repo.UpdateSession(ctx, session); err != nil {
			s.logger.Error("failed to update session", map[string]interface{}{"error": err.Error(), "session_id": session.ID})
		}
		return nil, fmt.Errorf("session has expired")
	}

	session.LastAccessAt = time.Now()
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		s.logger.Error("failed to update session", map[string]interface{}{"error": err.Error(), "session_id": session.ID})
	}

	return session, nil
}

// RevokeSession revokes an active session.
func (s *Service) RevokeSession(ctx context.Context, sessionID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.RevokeSession(ctx, sessionID)
}

// GenerateSDKSnippet generates an embeddable code snippet for integration.
func (s *Service) GenerateSDKSnippet(ctx context.Context, tenantID string, req *GenerateSDKRequest) (string, error) {
	if req.ConfigID == "" || req.Framework == "" {
		return "", fmt.Errorf("config_id and framework are required")
	}

	switch req.Framework {
	case "react":
		return s.generateReactSnippet(req.ConfigID, tenantID), nil
	case "vue":
		return s.generateVueSnippet(req.ConfigID, tenantID), nil
	case "vanilla":
		return s.generateVanillaSnippet(req.ConfigID, tenantID), nil
	default:
		return "", fmt.Errorf("unsupported framework %q: supported frameworks are react, vue, vanilla", req.Framework)
	}
}

// GetUsageStats returns usage statistics for a portal config.
func (s *Service) GetUsageStats(ctx context.Context, configID string) (*PortalUsageStats, error) {
	if s.repo == nil {
		return &PortalUsageStats{ConfigID: configID}, nil
	}
	return s.repo.GetUsageStats(ctx, configID)
}

func (s *Service) generateReactSnippet(configID, tenantID string) string {
	return fmt.Sprintf(`import { WaaSPortal } from '@waas/portal-sdk-react';

function WebhookPortal({ sessionToken }) {
  return (
    <WaaSPortal
      configId="%s"
      tenantId="%s"
      token={sessionToken}
      components={['endpoint_manager', 'delivery_log', 'metrics_dashboard']}
      onEndpointCreated={(endpoint) => console.log('Created:', endpoint)}
      onError={(error) => console.error('Portal error:', error)}
    />
  );
}`, configID, tenantID)
}

func (s *Service) generateVueSnippet(configID, tenantID string) string {
	return fmt.Sprintf(`<template>
  <WaaSPortal
    config-id="%s"
    tenant-id="%s"
    :token="sessionToken"
    :components="['endpoint_manager', 'delivery_log', 'metrics_dashboard']"
    @endpoint-created="onEndpointCreated"
    @error="onError"
  />
</template>

<script setup>
import { WaaSPortal } from '@waas/portal-sdk-vue';
const sessionToken = inject('portalToken');
</script>`, configID, tenantID)
}

func (s *Service) generateVanillaSnippet(configID, tenantID string) string {
	return fmt.Sprintf(`<div id="waas-portal"></div>
<script src="https://cdn.waas.dev/portal-sdk.min.js"></script>
<script>
  WaaS.Portal.init({
    container: '#waas-portal',
    configId: '%s',
    tenantId: '%s',
    token: 'SESSION_TOKEN',
    components: ['endpoint_manager', 'delivery_log', 'metrics_dashboard'],
    onReady: () => console.log('Portal ready'),
    onError: (err) => console.error('Portal error:', err),
  });
</script>`, configID, tenantID)
}

func defaultComponents() []string {
	return []string{
		ComponentEndpointManager,
		ComponentEventBrowser,
		ComponentDeliveryLog,
		ComponentMetricsDash,
	}
}

func defaultTheme() ThemeConfig {
	return ThemeConfig{
		PrimaryColor:    "#6366f1",
		SecondaryColor:  "#8b5cf6",
		BackgroundColor: "#ffffff",
		TextColor:       "#1f2937",
		FontFamily:      "Inter, system-ui, sans-serif",
		BorderRadius:    "8px",
		DarkMode:        false,
	}
}

func defaultFeatures() FeatureConfig {
	return FeatureConfig{
		EndpointManagement: true,
		EventBrowsing:      true,
		DeliveryLogs:       true,
		MetricsDashboard:   true,
		AlertConfiguration: false,
		APIExplorer:        false,
		LogViewer:          true,
		Subscriptions:      true,
	}
}

func defaultPermissions() []string {
	return []string{"endpoints:read", "endpoints:write", "deliveries:read", "events:read"}
}

func validateComponents(components []string) error {
	valid := map[string]bool{
		ComponentEndpointManager: true, ComponentEventBrowser: true,
		ComponentDeliveryLog: true, ComponentMetricsDash: true,
		ComponentAlertConfig: true, ComponentAPIExplorer: true,
		ComponentLogViewer: true, ComponentSubscriptions: true,
	}
	for _, c := range components {
		if !valid[c] {
			return fmt.Errorf("invalid component %q", c)
		}
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "psk_" + hex.EncodeToString(b), nil
}

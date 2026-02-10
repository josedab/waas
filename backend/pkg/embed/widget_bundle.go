package embed

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WidgetType defines the type of embeddable widget
type WidgetType string

const (
	WidgetTypeWebhookManager WidgetType = "webhook_manager"
	WidgetTypeDeliveryLogs   WidgetType = "delivery_logs"
	WidgetTypeEndpointConfig WidgetType = "endpoint_config"
	WidgetTypeAnalytics      WidgetType = "analytics"
	WidgetTypeErrorSummary   WidgetType = "error_summary"
	WidgetTypeActivityFeed   WidgetType = "activity_feed"
)

// WidgetFramework defines the target framework for the widget
type WidgetFramework string

const (
	FrameworkReact        WidgetFramework = "react"
	FrameworkWebComponent WidgetFramework = "web_component"
	FrameworkVanillaJS    WidgetFramework = "vanilla_js"
	FrameworkIframe       WidgetFramework = "iframe"
)

// WidgetBundle represents a packaged widget bundle ready for embedding
type WidgetBundle struct {
	ID          string              `json:"id"`
	TenantID    string              `json:"tenant_id"`
	Name        string              `json:"name"`
	WidgetType  WidgetType          `json:"widget_type"`
	Framework   WidgetFramework     `json:"framework"`
	Version     string              `json:"version"`
	Config      *WidgetBundleConfig `json:"config"`
	Theme       *WhiteLabelTheme    `json:"theme,omitempty"`
	EmbedCode   string              `json:"embed_code"`
	CDNURL      string              `json:"cdn_url"`
	Integrity   string              `json:"integrity_hash"`
	Permissions []string            `json:"permissions"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// WidgetBundleConfig defines configuration for a widget bundle
type WidgetBundleConfig struct {
	Width        string            `json:"width,omitempty"`
	Height       string            `json:"height,omitempty"`
	RefreshRate  int               `json:"refresh_rate_seconds,omitempty"`
	Locale       string            `json:"locale,omitempty"`
	DateFormat   string            `json:"date_format,omitempty"`
	PageSize     int               `json:"page_size,omitempty"`
	Features     map[string]bool   `json:"features,omitempty"`
	CustomCSS    string            `json:"custom_css,omitempty"`
	CustomLabels map[string]string `json:"custom_labels,omitempty"`
}

// WhiteLabelTheme defines white-label customization for widgets
type WhiteLabelTheme struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	Name            string            `json:"name"`
	PrimaryColor    string            `json:"primary_color"`
	SecondaryColor  string            `json:"secondary_color"`
	BackgroundColor string            `json:"background_color"`
	TextColor       string            `json:"text_color"`
	FontFamily      string            `json:"font_family"`
	BorderRadius    string            `json:"border_radius"`
	LogoURL         string            `json:"logo_url,omitempty"`
	FaviconURL      string            `json:"favicon_url,omitempty"`
	CustomCSS       string            `json:"custom_css,omitempty"`
	HideBranding    bool              `json:"hide_branding"`
	CustomLabels    map[string]string `json:"custom_labels,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// WidgetSession represents an active widget session
type WidgetSession struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	BundleID   string    `json:"bundle_id"`
	TokenHash  string    `json:"token_hash"`
	Origin     string    `json:"origin"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	EventCount int64     `json:"event_count"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CreateWidgetBundleRequest represents a request to create a widget bundle
type CreateWidgetBundleRequest struct {
	Name        string              `json:"name" binding:"required"`
	WidgetType  WidgetType          `json:"widget_type" binding:"required"`
	Framework   WidgetFramework     `json:"framework" binding:"required"`
	Config      *WidgetBundleConfig `json:"config,omitempty"`
	ThemeID     string              `json:"theme_id,omitempty"`
	Permissions []string            `json:"permissions,omitempty"`
	Origins     []string            `json:"allowed_origins,omitempty"`
}

// CreateThemeRequest represents a request to create a white-label theme
type CreateThemeRequest struct {
	Name            string            `json:"name" binding:"required"`
	PrimaryColor    string            `json:"primary_color" binding:"required"`
	SecondaryColor  string            `json:"secondary_color,omitempty"`
	BackgroundColor string            `json:"background_color,omitempty"`
	TextColor       string            `json:"text_color,omitempty"`
	FontFamily      string            `json:"font_family,omitempty"`
	BorderRadius    string            `json:"border_radius,omitempty"`
	LogoURL         string            `json:"logo_url,omitempty"`
	CustomCSS       string            `json:"custom_css,omitempty"`
	HideBranding    bool              `json:"hide_branding"`
	CustomLabels    map[string]string `json:"custom_labels,omitempty"`
}

// WidgetBundleRepository defines storage for widget bundles
type WidgetBundleRepository interface {
	CreateBundle(ctx context.Context, bundle *WidgetBundle) error
	GetBundle(ctx context.Context, tenantID, bundleID string) (*WidgetBundle, error)
	ListBundles(ctx context.Context, tenantID string) ([]WidgetBundle, error)
	UpdateBundle(ctx context.Context, bundle *WidgetBundle) error
	DeleteBundle(ctx context.Context, tenantID, bundleID string) error
	CreateTheme(ctx context.Context, theme *WhiteLabelTheme) error
	GetTheme(ctx context.Context, tenantID, themeID string) (*WhiteLabelTheme, error)
	ListThemes(ctx context.Context, tenantID string) ([]WhiteLabelTheme, error)
	UpdateTheme(ctx context.Context, theme *WhiteLabelTheme) error
	DeleteTheme(ctx context.Context, tenantID, themeID string) error
	CreateSession(ctx context.Context, session *WidgetSession) error
	GetActiveSessionCount(ctx context.Context, tenantID string) (int, error)
}

// WidgetBundleService manages widget bundles and white-label themes
type WidgetBundleService struct {
	repo       WidgetBundleRepository
	signingKey string
}

// NewWidgetBundleService creates a new widget bundle service
func NewWidgetBundleService(repo WidgetBundleRepository, signingKey string) *WidgetBundleService {
	if signingKey == "" {
		signingKey = uuid.New().String()
	}
	return &WidgetBundleService{
		repo:       repo,
		signingKey: signingKey,
	}
}

// CreateBundle creates a new widget bundle
func (s *WidgetBundleService) CreateBundle(ctx context.Context, tenantID string, req *CreateWidgetBundleRequest) (*WidgetBundle, error) {
	if req.Config == nil {
		req.Config = &WidgetBundleConfig{
			Width:       "100%",
			Height:      "600px",
			RefreshRate: 30,
			Locale:      "en",
			PageSize:    20,
		}
	}

	now := time.Now()
	bundle := &WidgetBundle{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		WidgetType:  req.WidgetType,
		Framework:   req.Framework,
		Version:     "1.0.0",
		Config:      req.Config,
		Permissions: req.Permissions,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Generate embed code based on framework
	bundle.EmbedCode = s.generateEmbedCode(bundle)
	bundle.Integrity = s.generateIntegrity(bundle.ID, tenantID)

	// Load theme if specified
	if req.ThemeID != "" {
		theme, err := s.repo.GetTheme(ctx, tenantID, req.ThemeID)
		if err == nil {
			bundle.Theme = theme
		}
	}

	if err := s.repo.CreateBundle(ctx, bundle); err != nil {
		return nil, fmt.Errorf("failed to create widget bundle: %w", err)
	}

	return bundle, nil
}

func (s *WidgetBundleService) generateEmbedCode(bundle *WidgetBundle) string {
	switch bundle.Framework {
	case FrameworkReact:
		typeName := capitalizeWidgetType(bundle.WidgetType)
		return fmt.Sprintf("import { %sWidget } from '@waas/widgets';\n\n<%sWidget\n  bundleId=\"%s\"\n  token=\"{YOUR_EMBED_TOKEN}\"\n/>", typeName, typeName, bundle.ID)

	case FrameworkWebComponent:
		return fmt.Sprintf(`<script src="https://cdn.waas.io/widgets/v1/waas-widgets.js"></script>

<waas-%s
  bundle-id="%s"
  token="YOUR_EMBED_TOKEN">
</waas-%s>`, bundle.WidgetType, bundle.ID, bundle.WidgetType)

	case FrameworkIframe:
		return fmt.Sprintf(`<iframe
  src="https://embed.waas.io/v1/%s?bundle=%s&token=YOUR_EMBED_TOKEN"
  width="%s" height="%s"
  frameborder="0">
</iframe>`, bundle.WidgetType, bundle.ID, bundle.Config.Width, bundle.Config.Height)

	default:
		return fmt.Sprintf(`<div id="waas-widget-%s"></div>
<script src="https://cdn.waas.io/widgets/v1/waas-widgets.js"></script>
<script>
  WaasWidgets.init('%s', { token: 'YOUR_EMBED_TOKEN' });
</script>`, bundle.ID, bundle.ID)
	}
}

func capitalizeWidgetType(wt WidgetType) string {
	parts := make([]byte, 0, len(wt))
	capitalize := true
	for _, c := range string(wt) {
		if c == '_' {
			capitalize = true
			continue
		}
		if capitalize && c >= 'a' && c <= 'z' {
			parts = append(parts, byte(c-32))
			capitalize = false
		} else {
			parts = append(parts, byte(c))
			capitalize = false
		}
	}
	return string(parts)
}

func (s *WidgetBundleService) generateIntegrity(bundleID, tenantID string) string {
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	mac.Write([]byte(bundleID + ":" + tenantID))
	return "sha256-" + hex.EncodeToString(mac.Sum(nil))
}

// GetBundle retrieves a widget bundle
func (s *WidgetBundleService) GetBundle(ctx context.Context, tenantID, bundleID string) (*WidgetBundle, error) {
	return s.repo.GetBundle(ctx, tenantID, bundleID)
}

// ListBundles lists all widget bundles
func (s *WidgetBundleService) ListBundles(ctx context.Context, tenantID string) ([]WidgetBundle, error) {
	return s.repo.ListBundles(ctx, tenantID)
}

// DeleteBundle deletes a widget bundle
func (s *WidgetBundleService) DeleteBundle(ctx context.Context, tenantID, bundleID string) error {
	return s.repo.DeleteBundle(ctx, tenantID, bundleID)
}

// CreateTheme creates a white-label theme
func (s *WidgetBundleService) CreateTheme(ctx context.Context, tenantID string, req *CreateThemeRequest) (*WhiteLabelTheme, error) {
	now := time.Now()
	theme := &WhiteLabelTheme{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		PrimaryColor:    req.PrimaryColor,
		SecondaryColor:  defaultIfEmpty(req.SecondaryColor, "#6B7280"),
		BackgroundColor: defaultIfEmpty(req.BackgroundColor, "#FFFFFF"),
		TextColor:       defaultIfEmpty(req.TextColor, "#1F2937"),
		FontFamily:      defaultIfEmpty(req.FontFamily, "Inter, sans-serif"),
		BorderRadius:    defaultIfEmpty(req.BorderRadius, "8px"),
		LogoURL:         req.LogoURL,
		CustomCSS:       req.CustomCSS,
		HideBranding:    req.HideBranding,
		CustomLabels:    req.CustomLabels,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateTheme(ctx, theme); err != nil {
		return nil, fmt.Errorf("failed to create theme: %w", err)
	}

	return theme, nil
}

// ListThemes lists all themes
func (s *WidgetBundleService) ListThemes(ctx context.Context, tenantID string) ([]WhiteLabelTheme, error) {
	return s.repo.ListThemes(ctx, tenantID)
}

// DeleteTheme deletes a theme
func (s *WidgetBundleService) DeleteTheme(ctx context.Context, tenantID, themeID string) error {
	return s.repo.DeleteTheme(ctx, tenantID, themeID)
}

func defaultIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

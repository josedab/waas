package embed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WidgetType constants
const (
	WidgetEndpointConfig = "endpoint-config"
	WidgetDeliveryLog    = "delivery-log"
	WidgetRetryControls  = "retry-controls"
	WidgetAnalytics      = "analytics"
	WidgetEventCatalog   = "event-catalog"
	WidgetTestConsole    = "test-console"
)

// WidgetSDKConfig represents the embeddable widget SDK configuration
type WidgetSDKConfig struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	PartnerID      string          `json:"partner_id,omitempty"`
	Name           string          `json:"name"`
	Widgets        []WidgetConfig  `json:"widgets"`
	Theme          *WidgetTheme    `json:"theme,omitempty"`
	Whitelabel     *WhitelabelOpts `json:"whitelabel,omitempty"`
	AllowedOrigins []string        `json:"allowed_origins"`
	TokenTTLHours  int             `json:"token_ttl_hours"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// WidgetConfig defines a single embeddable widget
type WidgetConfig struct {
	Type     string            `json:"type"`
	Enabled  bool              `json:"enabled"`
	Position string            `json:"position,omitempty"` // top, bottom, sidebar
	Options  map[string]string `json:"options,omitempty"`
}

// WidgetTheme defines visual customization
type WidgetTheme struct {
	PrimaryColor    string `json:"primary_color,omitempty"`
	SecondaryColor  string `json:"secondary_color,omitempty"`
	BackgroundColor string `json:"background_color,omitempty"`
	TextColor       string `json:"text_color,omitempty"`
	FontFamily      string `json:"font_family,omitempty"`
	BorderRadius    string `json:"border_radius,omitempty"`
	DarkMode        bool   `json:"dark_mode,omitempty"`
	CustomCSS       string `json:"custom_css,omitempty"`
}

// WhitelabelOpts defines white-label branding options
type WhitelabelOpts struct {
	RemoveBranding bool   `json:"remove_branding"`
	CustomLogo     string `json:"custom_logo,omitempty"`
	CustomTitle    string `json:"custom_title,omitempty"`
	CustomFooter   string `json:"custom_footer,omitempty"`
	PoweredByText  string `json:"powered_by_text,omitempty"`
}

// WidgetSDKSnippet contains embed code for multiple frameworks
type WidgetSDKSnippet struct {
	React     string `json:"react"`
	Vue       string `json:"vue"`
	VanillaJS string `json:"vanilla_js"`
	NextJS    string `json:"nextjs"`
	Angular   string `json:"angular"`
	CDNScript string `json:"cdn_script"`
	IFrame    string `json:"iframe"`
}

// CreateWidgetSDKRequest is the request for creating a widget SDK config
type CreateWidgetSDKRequest struct {
	Name           string          `json:"name" binding:"required"`
	Widgets        []WidgetConfig  `json:"widgets" binding:"required,min=1"`
	Theme          *WidgetTheme    `json:"theme,omitempty"`
	Whitelabel     *WhitelabelOpts `json:"whitelabel,omitempty"`
	AllowedOrigins []string        `json:"allowed_origins"`
	TokenTTLHours  int             `json:"token_ttl_hours"`
}

// WidgetSDKService manages embeddable widget configurations
type WidgetSDKService struct{}

// NewWidgetSDKService creates a new widget SDK service
func NewWidgetSDKService() *WidgetSDKService {
	return &WidgetSDKService{}
}

// CreateWidgetSDK creates a new widget SDK configuration
func (s *WidgetSDKService) CreateWidgetSDK(ctx context.Context, tenantID string, req *CreateWidgetSDKRequest) (*WidgetSDKConfig, error) {
	ttl := req.TokenTTLHours
	if ttl <= 0 {
		ttl = 24
	}

	// Validate widget types
	for _, w := range req.Widgets {
		switch w.Type {
		case WidgetEndpointConfig, WidgetDeliveryLog, WidgetRetryControls,
			WidgetAnalytics, WidgetEventCatalog, WidgetTestConsole:
			// valid
		default:
			return nil, fmt.Errorf("unsupported widget type: %s", w.Type)
		}
	}

	config := &WidgetSDKConfig{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		Widgets:        req.Widgets,
		Theme:          req.Theme,
		Whitelabel:     req.Whitelabel,
		AllowedOrigins: req.AllowedOrigins,
		TokenTTLHours:  ttl,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if config.Theme == nil {
		config.Theme = &WidgetTheme{
			PrimaryColor:    "#3B82F6",
			BackgroundColor: "#FFFFFF",
			TextColor:       "#1F2937",
			FontFamily:      "Inter, system-ui, sans-serif",
			BorderRadius:    "8px",
		}
	}

	return config, nil
}

// GenerateEmbedSnippets generates embed code for all frameworks
func (s *WidgetSDKService) GenerateEmbedSnippets(config *WidgetSDKConfig, apiURL string) *WidgetSDKSnippet {
	configJSON, _ := json.Marshal(map[string]interface{}{
		"configId":   config.ID,
		"apiUrl":     apiURL,
		"theme":      config.Theme,
		"whitelabel": config.Whitelabel,
	})

	return &WidgetSDKSnippet{
		React: fmt.Sprintf(`import { WaaSWebhookWidget } from '@waas/react-widget';

function WebhookManager() {
  return (
    <WaaSWebhookWidget
      configId="%s"
      apiUrl="%s"
      token={embedToken}
      theme={{
        primaryColor: '%s',
        darkMode: %t,
      }}
    />
  );
}`, config.ID, apiURL, safeThemeColor(config.Theme), safeDarkMode(config.Theme)),

		Vue: fmt.Sprintf(`<template>
  <WaaSWebhookWidget
    config-id="%s"
    :api-url="'%s'"
    :token="embedToken"
  />
</template>

<script setup>
import { WaaSWebhookWidget } from '@waas/vue-widget';
const embedToken = ref('your-embed-token');
</script>`, config.ID, apiURL),

		VanillaJS: fmt.Sprintf(`<!-- WaaS Webhook Widget -->
<div id="waas-webhook-widget"></div>
<script src="%s/widget/waas-widget.js"></script>
<script>
  WaaSWidget.init({
    container: '#waas-webhook-widget',
    config: %s,
    token: 'your-embed-token'
  });
</script>`, apiURL, string(configJSON)),

		NextJS: fmt.Sprintf(`'use client';
import { WaaSWebhookWidget } from '@waas/react-widget';

export default function WebhookPage() {
  return (
    <WaaSWebhookWidget
      configId="%s"
      apiUrl="%s"
      token={process.env.NEXT_PUBLIC_WAAS_TOKEN}
    />
  );
}`, config.ID, apiURL),

		Angular: fmt.Sprintf(`<!-- In your component template -->
<waas-webhook-widget
  [configId]="'%s'"
  [apiUrl]="'%s'"
  [token]="embedToken">
</waas-webhook-widget>

<!-- In your module -->
import { WaaSWidgetModule } from '@waas/angular-widget';

@NgModule({
  imports: [WaaSWidgetModule]
})
export class AppModule {}`, config.ID, apiURL),

		CDNScript: fmt.Sprintf(`<script src="https://cdn.waas.dev/widget/v1/waas-widget.min.js"></script>
<link rel="stylesheet" href="https://cdn.waas.dev/widget/v1/waas-widget.min.css">
<div id="waas-widget"></div>
<script>
  window.WaaSWidget.mount('#waas-widget', {
    configId: '%s',
    apiUrl: '%s',
    token: 'your-embed-token'
  });
</script>`, config.ID, apiURL),

		IFrame: fmt.Sprintf(`<iframe
  src="%s/widget/embed/%s?theme=light"
  width="100%%"
  height="700"
  frameborder="0"
  allow="clipboard-write"
  style="border-radius: 8px; border: 1px solid #E5E7EB;"
></iframe>`, apiURL, config.ID),
	}
}

// GetAvailableWidgets returns all available widget types
func (s *WidgetSDKService) GetAvailableWidgets() []map[string]string {
	return []map[string]string{
		{"type": WidgetEndpointConfig, "name": "Endpoint Configuration", "description": "Allow users to manage webhook endpoints"},
		{"type": WidgetDeliveryLog, "name": "Delivery Log", "description": "Show webhook delivery history with status"},
		{"type": WidgetRetryControls, "name": "Retry Controls", "description": "Enable manual retry of failed deliveries"},
		{"type": WidgetAnalytics, "name": "Analytics Dashboard", "description": "Delivery metrics and success rates"},
		{"type": WidgetEventCatalog, "name": "Event Catalog", "description": "Browse available webhook event types"},
		{"type": WidgetTestConsole, "name": "Test Console", "description": "Send test webhooks from the UI"},
	}
}

func safeThemeColor(theme *WidgetTheme) string {
	if theme != nil && theme.PrimaryColor != "" {
		return theme.PrimaryColor
	}
	return "#3B82F6"
}

func safeDarkMode(theme *WidgetTheme) bool {
	if theme != nil {
		return theme.DarkMode
	}
	return false
}

// WidgetSDKHandler provides HTTP handlers for the widget SDK
type WidgetSDKHandler struct {
	service *WidgetSDKService
}

// NewWidgetSDKHandler creates a new handler
func NewWidgetSDKHandler(service *WidgetSDKService) *WidgetSDKHandler {
	return &WidgetSDKHandler{service: service}
}

// RegisterWidgetSDKRoutes registers widget SDK routes
func (h *WidgetSDKHandler) RegisterWidgetSDKRoutes(router *gin.RouterGroup) {
	w := router.Group("/widget-sdk")
	{
		w.POST("/configs", h.CreateWidgetSDK)
		w.GET("/configs/:id/snippets", h.GetEmbedSnippets)
		w.GET("/widgets", h.ListAvailableWidgets)
	}
}

func (h *WidgetSDKHandler) CreateWidgetSDK(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateWidgetSDKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	config, err := h.service.CreateWidgetSDK(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, config)
}

func (h *WidgetSDKHandler) GetEmbedSnippets(c *gin.Context) {
	configID := c.Param("id")
	apiURL := c.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.waas.dev"
	}
	// Minimal config for snippet generation
	config := &WidgetSDKConfig{
		ID:    configID,
		Theme: &WidgetTheme{PrimaryColor: "#3B82F6"},
	}
	snippets := h.service.GenerateEmbedSnippets(config, apiURL)
	c.JSON(http.StatusOK, snippets)
}

func (h *WidgetSDKHandler) ListAvailableWidgets(c *gin.Context) {
	widgets := h.service.GetAvailableWidgets()
	c.JSON(http.StatusOK, gin.H{"widgets": widgets, "total": len(widgets)})
}

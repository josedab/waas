package portalsdk

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebComponentConfig generates the <waas-portal> Web Component embed code.
type WebComponentConfig struct {
	TagName    string            `json:"tag_name"`
	ScriptURL  string            `json:"script_url"`
	EmbedCode  string            `json:"embed_code"`
	CSSVars    map[string]string `json:"css_custom_properties"`
	Attributes map[string]string `json:"attributes"`
}

// ReactHooksConfig generates React hook configurations.
type ReactHooksConfig struct {
	PackageName string           `json:"package_name"`
	Hooks       []HookDefinition `json:"hooks"`
	Types       string           `json:"type_definitions"`
}

// HookDefinition describes a React hook.
type HookDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReturnType  string `json:"return_type"`
	Example     string `json:"example"`
}

// GetEmbedPortalConfig retrieves the portal configuration for embedding.
func (s *Service) GetEmbedPortalConfig(ctx context.Context, tenantID string) (*PortalConfig, error) {
	return &PortalConfig{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Name:     "Default Portal",
		Theme: ThemeConfig{
			PrimaryColor:    "#6366f1",
			SecondaryColor:  "#8b5cf6",
			BackgroundColor: "#ffffff",
			TextColor:       "#1f2937",
			FontFamily:      "Inter, system-ui, sans-serif",
			BorderRadius:    "8px",
			DarkMode:        false,
		},
		Features: FeatureConfig{
			EndpointManagement: true,
			DeliveryLogs:       true,
			AlertConfiguration: true,
			APIExplorer:        true,
		},
		Branding: BrandingConfig{
			CompanyName: "WaaS",
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// GetWebComponentConfig generates the Web Component embed code.
func (s *Service) GetWebComponentConfig(ctx context.Context, tenantID string) (*WebComponentConfig, error) {
	config, err := s.GetEmbedPortalConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	embedCode := fmt.Sprintf(`<script src="https://cdn.waas.cloud/portal-sdk/v1/waas-portal.js"></script>
<waas-portal
  tenant-id="%s"
  theme="default"
  features="endpoints,logs,retry"
></waas-portal>`, tenantID)

	cssVars := map[string]string{
		"--waas-primary":    config.Theme.PrimaryColor,
		"--waas-secondary":  config.Theme.SecondaryColor,
		"--waas-bg":         config.Theme.BackgroundColor,
		"--waas-text":       config.Theme.TextColor,
		"--waas-font":       config.Theme.FontFamily,
		"--waas-radius":     config.Theme.BorderRadius,
	}

	return &WebComponentConfig{
		TagName:   "waas-portal",
		ScriptURL: "https://cdn.waas.cloud/portal-sdk/v1/waas-portal.js",
		EmbedCode: embedCode,
		CSSVars:   cssVars,
		Attributes: map[string]string{
			"tenant-id": tenantID,
			"theme":     "default",
		},
	}, nil
}

// GetReactHooksConfig generates React hook configurations and type definitions.
func (s *Service) GetReactHooksConfig(ctx context.Context, tenantID string) (*ReactHooksConfig, error) {
	hooks := []HookDefinition{
		{
			Name:        "useWebhookEndpoints",
			Description: "Hook to list, create, update, and delete webhook endpoints",
			ReturnType:  "{ endpoints: Endpoint[], create: (req) => Promise<Endpoint>, update: (id, req) => Promise<Endpoint>, remove: (id) => Promise<void>, loading: boolean, error: Error | null }",
			Example:     `const { endpoints, create, loading } = useWebhookEndpoints();`,
		},
		{
			Name:        "useDeliveryLogs",
			Description: "Hook to query delivery history with filtering and pagination",
			ReturnType:  "{ deliveries: Delivery[], total: number, loading: boolean, error: Error | null, refetch: () => void }",
			Example:     `const { deliveries, total, loading } = useDeliveryLogs({ endpointId, limit: 20 });`,
		},
		{
			Name:        "useEndpointHealth",
			Description: "Hook to monitor endpoint health and reliability score",
			ReturnType:  "{ health: HealthStatus, score: number, loading: boolean }",
			Example:     `const { health, score } = useEndpointHealth(endpointId);`,
		},
		{
			Name:        "useRetryConfig",
			Description: "Hook to get and update retry configuration for an endpoint",
			ReturnType:  "{ config: RetryConfig, update: (config) => Promise<RetryConfig>, loading: boolean }",
			Example:     `const { config, update } = useRetryConfig(endpointId);`,
		},
		{
			Name:        "useWebhookPortal",
			Description: "Root hook that provides the WaaS portal context and authentication",
			ReturnType:  "{ isReady: boolean, tenant: Tenant, config: PortalConfig }",
			Example:     `const { isReady, tenant } = useWebhookPortal({ tenantId, apiKey });`,
		},
	}

	types := `// WaaS Portal SDK TypeScript Types
export interface Endpoint {
  id: string;
  url: string;
  name: string;
  status: 'active' | 'inactive' | 'disabled';
  created_at: string;
  updated_at: string;
}

export interface Delivery {
  id: string;
  endpoint_id: string;
  status: 'success' | 'failed' | 'pending';
  status_code: number;
  latency_ms: number;
  created_at: string;
}

export interface RetryConfig {
  max_retries: number;
  initial_interval_ms: number;
  max_interval_ms: number;
  multiplier: number;
}

export interface HealthStatus {
  status: 'healthy' | 'degraded' | 'critical';
  score: number;
  success_rate: number;
  latency_p95_ms: number;
}
`

	return &ReactHooksConfig{
		PackageName: "@waas/react-portal",
		Hooks:       hooks,
		Types:       types,
	}, nil
}

// RegisterEmbedRoutes registers embeddable portal SDK routes.
func (h *Handler) RegisterEmbedRoutes(r *gin.RouterGroup) {
	portal := r.Group("/portal-sdk/embed")
	{
		portal.GET("/config", h.GetEmbedConfig)
		portal.GET("/web-component", h.GetWebComponentEmbed)
		portal.GET("/react", h.GetReactHooksEmbed)
	}
}

func (h *Handler) GetEmbedConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	config, err := h.service.GetEmbedPortalConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *Handler) GetWebComponentEmbed(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	config, err := h.service.GetWebComponentConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *Handler) GetReactHooksEmbed(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	config, err := h.service.GetReactHooksConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}


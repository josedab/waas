package portalsdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

// WebComponentGenerator generates framework-agnostic web component bundles
type WebComponentGenerator struct{}

// NewWebComponentGenerator creates a new web component generator
func NewWebComponentGenerator() *WebComponentGenerator {
	return &WebComponentGenerator{}
}

// WebComponentBundle represents a generated web component bundle
type WebComponentBundle struct {
	TagName    string `json:"tag_name"`
	JavaScript string `json:"javascript"`
	CSS        string `json:"css"`
	TypeDefs   string `json:"type_defs,omitempty"`
}

// ReactComponentBundle represents a generated React component library
type ReactComponentBundle struct {
	PackageName string           `json:"package_name"`
	Components  []ReactComponent `json:"components"`
	Hooks       []ReactHook      `json:"hooks"`
	TypeDefs    string           `json:"type_defs"`
}

// ReactComponent represents a React component definition
type ReactComponent struct {
	Name        string          `json:"name"`
	Props       []ComponentProp `json:"props"`
	Description string          `json:"description"`
	Source      string          `json:"source"`
}

// ReactHook represents a React hook definition
type ReactHook struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// ComponentProp represents a component property
type ComponentProp struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
	Description  string `json:"description"`
}

// ThemeVariables represents CSS custom properties for white-label theming
type ThemeVariables struct {
	PrimaryColor    string `json:"--waas-primary" yaml:"primary_color"`
	PrimaryHover    string `json:"--waas-primary-hover" yaml:"primary_hover"`
	SecondaryColor  string `json:"--waas-secondary" yaml:"secondary_color"`
	BackgroundColor string `json:"--waas-bg" yaml:"background_color"`
	SurfaceColor    string `json:"--waas-surface" yaml:"surface_color"`
	TextColor       string `json:"--waas-text" yaml:"text_color"`
	TextSecondary   string `json:"--waas-text-secondary" yaml:"text_secondary"`
	BorderColor     string `json:"--waas-border" yaml:"border_color"`
	SuccessColor    string `json:"--waas-success" yaml:"success_color"`
	ErrorColor      string `json:"--waas-error" yaml:"error_color"`
	WarningColor    string `json:"--waas-warning" yaml:"warning_color"`
	FontFamily      string `json:"--waas-font" yaml:"font_family"`
	FontSize        string `json:"--waas-font-size" yaml:"font_size"`
	BorderRadius    string `json:"--waas-radius" yaml:"border_radius"`
	ShadowSm        string `json:"--waas-shadow-sm" yaml:"shadow_sm"`
	ShadowMd        string `json:"--waas-shadow-md" yaml:"shadow_md"`
}

// DefaultLightTheme returns the default light theme
var DefaultLightTheme = ThemeVariables{
	PrimaryColor:    "#2563eb",
	PrimaryHover:    "#1d4ed8",
	SecondaryColor:  "#64748b",
	BackgroundColor: "#ffffff",
	SurfaceColor:    "#f8fafc",
	TextColor:       "#0f172a",
	TextSecondary:   "#64748b",
	BorderColor:     "#e2e8f0",
	SuccessColor:    "#16a34a",
	ErrorColor:      "#dc2626",
	WarningColor:    "#d97706",
	FontFamily:      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
	FontSize:        "14px",
	BorderRadius:    "8px",
	ShadowSm:        "0 1px 2px rgba(0,0,0,0.05)",
	ShadowMd:        "0 4px 6px rgba(0,0,0,0.1)",
}

// DefaultDarkTheme returns the default dark theme
var DefaultDarkTheme = ThemeVariables{
	PrimaryColor:    "#3b82f6",
	PrimaryHover:    "#60a5fa",
	SecondaryColor:  "#94a3b8",
	BackgroundColor: "#0f172a",
	SurfaceColor:    "#1e293b",
	TextColor:       "#f8fafc",
	TextSecondary:   "#94a3b8",
	BorderColor:     "#334155",
	SuccessColor:    "#22c55e",
	ErrorColor:      "#ef4444",
	WarningColor:    "#f59e0b",
	FontFamily:      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
	FontSize:        "14px",
	BorderRadius:    "8px",
	ShadowSm:        "0 1px 2px rgba(0,0,0,0.3)",
	ShadowMd:        "0 4px 6px rgba(0,0,0,0.4)",
}

// GenerateWebComponent generates a <waas-portal> web component
func (g *WebComponentGenerator) GenerateWebComponent(config *PortalConfig) *WebComponentBundle {
	theme := resolveTheme(config)

	return &WebComponentBundle{
		TagName:    "waas-portal",
		JavaScript: generateWebComponentJS(config, theme),
		CSS:        generateThemeCSS(theme),
		TypeDefs:   generateWebComponentTypeDefs(),
	}
}

// GenerateReactLibrary generates a React component library with hooks
func (g *WebComponentGenerator) GenerateReactLibrary(config *PortalConfig) *ReactComponentBundle {
	return &ReactComponentBundle{
		PackageName: "@waas/react-portal",
		Components:  generateReactComponents(config),
		Hooks:       generateReactHooks(),
		TypeDefs:    generateReactTypeDefs(config),
	}
}

func resolveTheme(config *PortalConfig) ThemeVariables {
	theme := DefaultLightTheme
	if config.Theme.DarkMode {
		theme = DefaultDarkTheme
	}
	if config.Theme.PrimaryColor != "" {
		theme.PrimaryColor = config.Theme.PrimaryColor
	}
	if config.Theme.BackgroundColor != "" {
		theme.BackgroundColor = config.Theme.BackgroundColor
	}
	if config.Theme.TextColor != "" {
		theme.TextColor = config.Theme.TextColor
	}
	if config.Theme.FontFamily != "" {
		theme.FontFamily = config.Theme.FontFamily
	}
	if config.Theme.BorderRadius != "" {
		theme.BorderRadius = config.Theme.BorderRadius
	}
	return theme
}

func generateThemeCSS(theme ThemeVariables) string {
	return fmt.Sprintf(`:host, .waas-portal {
  --waas-primary: %s;
  --waas-primary-hover: %s;
  --waas-secondary: %s;
  --waas-bg: %s;
  --waas-surface: %s;
  --waas-text: %s;
  --waas-text-secondary: %s;
  --waas-border: %s;
  --waas-success: %s;
  --waas-error: %s;
  --waas-warning: %s;
  --waas-font: %s;
  --waas-font-size: %s;
  --waas-radius: %s;
  --waas-shadow-sm: %s;
  --waas-shadow-md: %s;
  font-family: var(--waas-font);
  font-size: var(--waas-font-size);
  color: var(--waas-text);
  background: var(--waas-bg);
}`,
		theme.PrimaryColor, theme.PrimaryHover, theme.SecondaryColor,
		theme.BackgroundColor, theme.SurfaceColor,
		theme.TextColor, theme.TextSecondary, theme.BorderColor,
		theme.SuccessColor, theme.ErrorColor, theme.WarningColor,
		theme.FontFamily, theme.FontSize, theme.BorderRadius,
		theme.ShadowSm, theme.ShadowMd)
}

func generateWebComponentJS(config *PortalConfig, theme ThemeVariables) string {
	componentsList, _ := json.Marshal(config.Components)
	originsList, _ := json.Marshal(config.AllowedOrigins)

	return fmt.Sprintf(`// WaaS Portal Web Component - Auto-generated
class WaaSPortal extends HTMLElement {
  static get observedAttributes() {
    return ['token', 'theme', 'components', 'locale'];
  }

  constructor() {
    super();
    this.attachShadow({ mode: 'open' });
    this._config = {
      components: %s,
      allowedOrigins: %s,
    };
  }

  connectedCallback() {
    this.render();
    this._initializeAPI();
  }

  attributeChangedCallback(name, oldVal, newVal) {
    if (oldVal !== newVal) this.render();
  }

  async _initializeAPI() {
    const token = this.getAttribute('token');
    if (!token) {
      this._renderError('Missing token attribute');
      return;
    }
    this._apiClient = new WaaSAPIClient(token);
    await this._loadData();
  }

  async _loadData() {
    try {
      const endpoints = await this._apiClient.getEndpoints();
      this._renderPortal(endpoints);
    } catch (err) {
      this._renderError(err.message);
    }
  }

  render() {
    const style = document.createElement('style');
    style.textContent = this._getStyles();
    this.shadowRoot.innerHTML = '';
    this.shadowRoot.appendChild(style);

    const container = document.createElement('div');
    container.className = 'waas-portal';
    container.innerHTML = '<div class="waas-loading">Loading webhook portal...</div>';
    this.shadowRoot.appendChild(container);
  }

  _renderPortal(endpoints) {
    const container = this.shadowRoot.querySelector('.waas-portal');
    if (!container) return;
    container.innerHTML = this._buildPortalHTML(endpoints);
  }

  _renderError(message) {
    const container = this.shadowRoot.querySelector('.waas-portal');
    if (!container) return;
    container.innerHTML = '<div class="waas-error">' + message + '</div>';
  }

  _buildPortalHTML(endpoints) {
    return '<div class="waas-header"><h2>Webhook Portal</h2></div>' +
      '<div class="waas-content">' +
      (endpoints || []).map(ep =>
        '<div class="waas-endpoint">' +
          '<span class="waas-endpoint-name">' + ep.name + '</span>' +
          '<span class="waas-endpoint-url">' + ep.url + '</span>' +
        '</div>'
      ).join('') +
      '</div>';
  }

  _getStyles() {
    return %s;
  }
}

class WaaSAPIClient {
  constructor(token) { this.token = token; }
  async getEndpoints() { return []; }
  async getDeliveries(endpointId) { return []; }
  async getMetrics() { return {}; }
}

if (!customElements.get('waas-portal')) {
  customElements.define('waas-portal', WaaSPortal);
}`, componentsList, originsList, "`"+generateThemeCSS(theme)+"`")
}

func generateWebComponentTypeDefs() string {
	return `declare namespace JSX {
  interface IntrinsicElements {
    'waas-portal': {
      token: string;
      theme?: 'light' | 'dark' | 'auto';
      components?: string;
      locale?: string;
    };
  }
}`
}

func generateReactComponents(config *PortalConfig) []ReactComponent {
	components := []ReactComponent{
		{
			Name:        "WaaSPortal",
			Description: "Main portal component that renders the full webhook management UI",
			Props: []ComponentProp{
				{Name: "token", Type: "string", Required: true, Description: "API token for authentication"},
				{Name: "theme", Type: "'light' | 'dark' | 'auto'", Required: false, DefaultValue: "light", Description: "Color theme"},
				{Name: "components", Type: "string[]", Required: false, Description: "Components to display"},
				{Name: "onEndpointClick", Type: "(id: string) => void", Required: false, Description: "Callback when endpoint clicked"},
				{Name: "className", Type: "string", Required: false, Description: "Additional CSS class"},
			},
			Source: generateReactPortalComponent(),
		},
		{
			Name:        "EndpointManager",
			Description: "Endpoint management component for listing and configuring webhook endpoints",
			Props: []ComponentProp{
				{Name: "token", Type: "string", Required: true, Description: "API token"},
				{Name: "onEndpointCreate", Type: "(endpoint: Endpoint) => void", Required: false, Description: "Callback on create"},
			},
			Source: generateReactEndpointManager(),
		},
		{
			Name:        "DeliveryLog",
			Description: "Delivery log component showing webhook delivery history",
			Props: []ComponentProp{
				{Name: "token", Type: "string", Required: true, Description: "API token"},
				{Name: "endpointId", Type: "string", Required: false, Description: "Filter by endpoint"},
				{Name: "limit", Type: "number", Required: false, DefaultValue: "50", Description: "Max entries"},
			},
			Source: generateReactDeliveryLog(),
		},
	}
	return components
}

func generateReactHooks() []ReactHook {
	return []ReactHook{
		{
			Name:        "useWaaSEndpoints",
			Description: "Hook to fetch and manage webhook endpoints",
			Source: `export function useWaaSEndpoints(token: string) {
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const client = new WaaSClient(token);
    client.getEndpoints()
      .then(setEndpoints)
      .catch(setError)
      .finally(() => setLoading(false));
  }, [token]);

  return { endpoints, loading, error, refetch: () => {} };
}`,
		},
		{
			Name:        "useWaaSDeliveries",
			Description: "Hook to fetch delivery logs with real-time updates",
			Source: `export function useWaaSDeliveries(token: string, endpointId?: string) {
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const client = new WaaSClient(token);
    client.getDeliveries(endpointId)
      .then(setDeliveries)
      .finally(() => setLoading(false));
  }, [token, endpointId]);

  return { deliveries, loading };
}`,
		},
		{
			Name:        "useWaaSMetrics",
			Description: "Hook to fetch webhook delivery metrics",
			Source: `export function useWaaSMetrics(token: string) {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const client = new WaaSClient(token);
    client.getMetrics()
      .then(setMetrics)
      .finally(() => setLoading(false));
  }, [token]);

  return { metrics, loading };
}`,
		},
	}
}

func generateReactTypeDefs(config *PortalConfig) string {
	var sb strings.Builder
	sb.WriteString("// Auto-generated React type definitions for @waas/react-portal\n\n")
	sb.WriteString("export interface WaaSPortalProps {\n")
	sb.WriteString("  token: string;\n")
	sb.WriteString("  theme?: 'light' | 'dark' | 'auto';\n")
	sb.WriteString("  components?: string[];\n")
	sb.WriteString("  className?: string;\n")
	sb.WriteString("  onEndpointClick?: (id: string) => void;\n")
	sb.WriteString("}\n\n")
	sb.WriteString("export interface Endpoint {\n")
	sb.WriteString("  id: string;\n  name: string;\n  url: string;\n  status: string;\n}\n\n")
	sb.WriteString("export interface Delivery {\n")
	sb.WriteString("  id: string;\n  endpoint_id: string;\n  status: string;\n  http_status: number;\n  timestamp: string;\n}\n\n")
	sb.WriteString("export interface Metrics {\n")
	sb.WriteString("  total_deliveries: number;\n  success_rate: number;\n  avg_latency_ms: number;\n}\n")
	return sb.String()
}

func generateReactPortalComponent() string {
	return `import React, { useEffect, useState } from 'react';

export const WaaSPortal: React.FC<WaaSPortalProps> = ({ token, theme = 'light', components, className, onEndpointClick }) => {
  const { endpoints, loading } = useWaaSEndpoints(token);

  if (loading) return <div className="waas-loading">Loading...</div>;

  return (
    <div className={"waas-portal waas-theme-" + theme + " " + (className || "")}>
      <EndpointManager token={token} />
      <DeliveryLog token={token} />
    </div>
  );
};`
}

func generateReactEndpointManager() string {
	return `import React from 'react';

export const EndpointManager: React.FC<{ token: string }> = ({ token }) => {
  const { endpoints, loading } = useWaaSEndpoints(token);

  return (
    <div className="waas-endpoints">
      <h3>Webhook Endpoints</h3>
      {endpoints.map(ep => (
        <div key={ep.id} className="waas-endpoint-card">
          <span>{ep.name}</span>
          <span>{ep.url}</span>
          <span className={"waas-status-" + ep.status}>{ep.status}</span>
        </div>
      ))}
    </div>
  );
};`
}

func generateReactDeliveryLog() string {
	return `import React from 'react';

export const DeliveryLog: React.FC<{ token: string; endpointId?: string; limit?: number }> = ({ token, endpointId, limit = 50 }) => {
  const { deliveries, loading } = useWaaSDeliveries(token, endpointId);

  return (
    <div className="waas-delivery-log">
      <h3>Delivery Log</h3>
      <table>
        <thead><tr><th>Time</th><th>Status</th><th>HTTP</th></tr></thead>
        <tbody>
          {deliveries.slice(0, limit).map(d => (
            <tr key={d.id}>
              <td>{d.timestamp}</td>
              <td>{d.status}</td>
              <td>{d.http_status}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};`
}

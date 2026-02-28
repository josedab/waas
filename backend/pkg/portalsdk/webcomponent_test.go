package portalsdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebComponentGenerator_GenerateWebComponent(t *testing.T) {
	gen := NewWebComponentGenerator()
	config := &PortalConfig{
		Name:           "test-portal",
		AllowedOrigins: []string{"https://example.com"},
		Components:     []string{ComponentEndpointManager, ComponentDeliveryLog},
		Theme: ThemeConfig{
			PrimaryColor: "#ff0000",
			DarkMode:     false,
		},
	}

	bundle := gen.GenerateWebComponent(config)
	assert.Equal(t, "waas-portal", bundle.TagName)
	assert.Contains(t, bundle.JavaScript, "WaaSPortal")
	assert.Contains(t, bundle.JavaScript, "customElements.define")
	assert.Contains(t, bundle.CSS, "--waas-primary: #ff0000")
	assert.NotEmpty(t, bundle.TypeDefs)
}

func TestWebComponentGenerator_DarkTheme(t *testing.T) {
	gen := NewWebComponentGenerator()
	config := &PortalConfig{
		Name:           "dark-portal",
		AllowedOrigins: []string{"*"},
		Components:     []string{ComponentEndpointManager},
		Theme: ThemeConfig{
			DarkMode: true,
		},
	}

	bundle := gen.GenerateWebComponent(config)
	assert.Contains(t, bundle.CSS, DefaultDarkTheme.BackgroundColor)
}

func TestWebComponentGenerator_GenerateReactLibrary(t *testing.T) {
	gen := NewWebComponentGenerator()
	config := &PortalConfig{
		Name:           "react-portal",
		AllowedOrigins: []string{"https://app.example.com"},
		Components:     []string{ComponentEndpointManager, ComponentDeliveryLog, ComponentMetricsDash},
	}

	bundle := gen.GenerateReactLibrary(config)
	assert.Equal(t, "@waas/react-portal", bundle.PackageName)
	assert.GreaterOrEqual(t, len(bundle.Components), 3)
	assert.GreaterOrEqual(t, len(bundle.Hooks), 3)
	assert.Contains(t, bundle.TypeDefs, "WaaSPortalProps")

	// Verify component names
	componentNames := make([]string, len(bundle.Components))
	for i, c := range bundle.Components {
		componentNames[i] = c.Name
	}
	assert.Contains(t, componentNames, "WaaSPortal")
	assert.Contains(t, componentNames, "EndpointManager")
	assert.Contains(t, componentNames, "DeliveryLog")

	// Verify hooks
	hookNames := make([]string, len(bundle.Hooks))
	for i, h := range bundle.Hooks {
		hookNames[i] = h.Name
	}
	assert.Contains(t, hookNames, "useWaaSEndpoints")
	assert.Contains(t, hookNames, "useWaaSDeliveries")
	assert.Contains(t, hookNames, "useWaaSMetrics")
}

func TestGenerateThemeCSS(t *testing.T) {
	css := generateThemeCSS(DefaultLightTheme)
	assert.Contains(t, css, "--waas-primary:")
	assert.Contains(t, css, "--waas-bg:")
	assert.Contains(t, css, "--waas-text:")
	assert.Contains(t, css, "font-family: var(--waas-font)")
}

func TestResolveTheme_Overrides(t *testing.T) {
	config := &PortalConfig{
		Theme: ThemeConfig{
			PrimaryColor: "#custom",
			FontFamily:   "Comic Sans",
		},
	}
	theme := resolveTheme(config)
	assert.Equal(t, "#custom", theme.PrimaryColor)
	assert.Equal(t, "Comic Sans", theme.FontFamily)
	// Other defaults preserved
	assert.Equal(t, DefaultLightTheme.SuccessColor, theme.SuccessColor)
}

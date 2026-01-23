package analyticsembed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides embeddable analytics functionality
type Service struct {
	repo Repository
}

// NewService creates a new embeddable analytics service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateWidget creates a new analytics widget
func (s *Service) CreateWidget(ctx context.Context, tenantID string, req *CreateWidgetRequest) (*WidgetConfig, error) {
	widget := &WidgetConfig{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		Name:               req.Name,
		WidgetType:         req.WidgetType,
		DataSource:         req.DataSource,
		TimeRange:          req.TimeRange,
		RefreshIntervalSec: req.RefreshIntervalSec,
		CustomCSS:          req.CustomCSS,
		IsPublic:           req.IsPublic,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.CreateWidget(ctx, widget); err != nil {
		return nil, fmt.Errorf("failed to create widget: %w", err)
	}

	return widget, nil
}

// GetWidget retrieves a widget by ID
func (s *Service) GetWidget(ctx context.Context, tenantID, widgetID string) (*WidgetConfig, error) {
	return s.repo.GetWidget(ctx, tenantID, widgetID)
}

// ListWidgets retrieves all widgets for a tenant
func (s *Service) ListWidgets(ctx context.Context, tenantID string) ([]WidgetConfig, error) {
	return s.repo.ListWidgets(ctx, tenantID)
}

// UpdateWidget updates an existing widget
func (s *Service) UpdateWidget(ctx context.Context, tenantID, widgetID string, req *CreateWidgetRequest) (*WidgetConfig, error) {
	widget, err := s.repo.GetWidget(ctx, tenantID, widgetID)
	if err != nil {
		return nil, err
	}

	widget.Name = req.Name
	widget.WidgetType = req.WidgetType
	widget.DataSource = req.DataSource
	widget.TimeRange = req.TimeRange
	widget.RefreshIntervalSec = req.RefreshIntervalSec
	widget.CustomCSS = req.CustomCSS
	widget.IsPublic = req.IsPublic
	widget.UpdatedAt = time.Now()

	if err := s.repo.UpdateWidget(ctx, widget); err != nil {
		return nil, fmt.Errorf("failed to update widget: %w", err)
	}

	return widget, nil
}

// DeleteWidget deletes a widget
func (s *Service) DeleteWidget(ctx context.Context, tenantID, widgetID string) error {
	return s.repo.DeleteWidget(ctx, tenantID, widgetID)
}

// GenerateEmbedToken creates a secure embed token for a widget
func (s *Service) GenerateEmbedToken(ctx context.Context, tenantID string, req *CreateEmbedTokenRequest) (*EmbedToken, error) {
	// Verify the widget exists
	_, err := s.repo.GetWidget(ctx, tenantID, req.WidgetID)
	if err != nil {
		return nil, fmt.Errorf("widget not found: %w", err)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure token: %w", err)
	}

	token := &EmbedToken{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		WidgetID:       req.WidgetID,
		Token:          hex.EncodeToString(tokenBytes),
		Scopes:         req.Scopes,
		AllowedOrigins: req.AllowedOrigins,
		ExpiresAt:      time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreateEmbedToken(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to create embed token: %w", err)
	}

	return token, nil
}

// ValidateEmbedToken checks if a token is valid and has the required scope
func (s *Service) ValidateEmbedToken(ctx context.Context, tokenStr, requiredScope string) (*EmbedToken, error) {
	token, err := s.repo.GetEmbedTokenByToken(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	if !token.IsActive {
		return nil, fmt.Errorf("token is inactive")
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	if requiredScope != "" {
		hasScope := false
		for _, scope := range token.Scopes {
			if scope == requiredScope {
				hasScope = true
				break
			}
		}
		if !hasScope {
			return nil, fmt.Errorf("token does not have required scope: %s", requiredScope)
		}
	}

	return token, nil
}

// GetEmbedSnippet generates embeddable code snippets for a widget
func (s *Service) GetEmbedSnippet(ctx context.Context, tenantID, widgetID string) (*EmbedSnippet, error) {
	widget, err := s.repo.GetWidget(ctx, tenantID, widgetID)
	if err != nil {
		return nil, fmt.Errorf("widget not found: %w", err)
	}

	iframeURL := fmt.Sprintf("/embed/widgets/%s?tenant=%s", widget.ID, widget.TenantID)

	return &EmbedSnippet{
		WidgetID:   widget.ID,
		WidgetType: widget.WidgetType,
		HTML:       s.generateSnippetHTML(widget, iframeURL),
		React:      s.generateSnippetReact(widget, iframeURL),
		JavaScript: s.generateSnippetJS(widget, iframeURL),
		IframeURL:  iframeURL,
	}, nil
}

// GetWidgetData returns analytical data for a widget
func (s *Service) GetWidgetData(ctx context.Context, tenantID, widgetID string) (*WidgetData, error) {
	widget, err := s.repo.GetWidget(ctx, tenantID, widgetID)
	if err != nil {
		return nil, fmt.Errorf("widget not found: %w", err)
	}

	data := s.generateMockData(widget)

	return &WidgetData{
		WidgetID:    widget.ID,
		WidgetType:  widget.WidgetType,
		Data:        data,
		GeneratedAt: time.Now(),
	}, nil
}

// GetTheme retrieves the theme configuration for a tenant
func (s *Service) GetTheme(ctx context.Context, tenantID string) (*ThemeConfig, error) {
	theme, err := s.repo.GetThemeConfig(ctx, tenantID)
	if err != nil {
		// Return default theme if none exists
		return &ThemeConfig{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			PrimaryColor:    "#3B82F6",
			SecondaryColor:  "#10B981",
			BackgroundColor: "#FFFFFF",
			TextColor:       "#1F2937",
			FontFamily:      "Inter, sans-serif",
			BorderRadius:    "8px",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}, nil
	}
	return theme, nil
}

// UpdateTheme updates the theme configuration for a tenant
func (s *Service) UpdateTheme(ctx context.Context, tenantID string, req *UpdateThemeRequest) (*ThemeConfig, error) {
	theme := &ThemeConfig{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		PrimaryColor:    req.PrimaryColor,
		SecondaryColor:  req.SecondaryColor,
		BackgroundColor: req.BackgroundColor,
		TextColor:       req.TextColor,
		FontFamily:      req.FontFamily,
		BorderRadius:    req.BorderRadius,
		CustomCSS:       req.CustomCSS,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.UpsertThemeConfig(ctx, theme); err != nil {
		return nil, fmt.Errorf("failed to update theme: %w", err)
	}

	return theme, nil
}

func (s *Service) generateSnippetHTML(widget *WidgetConfig, iframeURL string) string {
	return fmt.Sprintf(`<!-- %s Widget -->
<div id="waas-widget-%s" style="width:100%%;height:400px;">
  <iframe
    src="%s"
    style="width:100%%;height:100%%;border:none;"
    title="%s"
    loading="lazy"
    sandbox="allow-scripts allow-same-origin">
  </iframe>
</div>`, widget.Name, widget.ID, iframeURL, widget.Name)
}

func (s *Service) generateSnippetReact(widget *WidgetConfig, iframeURL string) string {
	return fmt.Sprintf(`import React from 'react';

const %sWidget = () => (
  <div style={{ width: '100%%', height: '400px' }}>
    <iframe
      src="%s"
      style={{ width: '100%%', height: '100%%', border: 'none' }}
      title="%s"
      loading="lazy"
      sandbox="allow-scripts allow-same-origin"
    />
  </div>
);

export default %sWidget;`, widget.WidgetType, iframeURL, widget.Name, widget.WidgetType)
}

func (s *Service) generateSnippetJS(widget *WidgetConfig, iframeURL string) string {
	return fmt.Sprintf(`(function() {
  var container = document.getElementById('waas-widget-%s');
  if (!container) {
    container = document.createElement('div');
    container.id = 'waas-widget-%s';
    container.style.cssText = 'width:100%%;height:400px;';
    document.currentScript.parentElement.appendChild(container);
  }
  var iframe = document.createElement('iframe');
  iframe.src = '%s';
  iframe.style.cssText = 'width:100%%;height:100%%;border:none;';
  iframe.title = '%s';
  iframe.loading = 'lazy';
  iframe.sandbox = 'allow-scripts allow-same-origin';
  container.appendChild(iframe);
})();`, widget.ID, widget.ID, iframeURL, widget.Name)
}

func (s *Service) generateMockData(widget *WidgetConfig) interface{} {
	switch widget.WidgetType {
	case WidgetTypeDeliveryChart:
		return map[string]interface{}{
			"labels": []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
			"datasets": []map[string]interface{}{
				{
					"label": "Delivered",
					"data":  []int{1200, 1350, 1100, 1400, 1250, 900, 800},
				},
				{
					"label": "Failed",
					"data":  []int{15, 8, 22, 5, 12, 3, 7},
				},
			},
		}
	case WidgetTypeErrorBreakdown:
		return map[string]interface{}{
			"categories": []map[string]interface{}{
				{"name": "Timeout", "count": 45, "percentage": 35.2},
				{"name": "Connection Refused", "count": 28, "percentage": 21.9},
				{"name": "DNS Resolution", "count": 22, "percentage": 17.2},
				{"name": "TLS Handshake", "count": 18, "percentage": 14.1},
				{"name": "Other", "count": 15, "percentage": 11.6},
			},
			"total_errors": 128,
		}
	case WidgetTypeLatencyHeatmap:
		return map[string]interface{}{
			"buckets": []map[string]interface{}{
				{"range": "0-50ms", "count": 4500, "hour": 0},
				{"range": "50-100ms", "count": 3200, "hour": 6},
				{"range": "100-250ms", "count": 1800, "hour": 12},
				{"range": "250-500ms", "count": 450, "hour": 18},
				{"range": "500ms+", "count": 120, "hour": 23},
			},
			"p50_ms": 42,
			"p99_ms": 380,
		}
	case WidgetTypeEndpointHealth:
		return map[string]interface{}{
			"endpoints": []map[string]interface{}{
				{"url": "/api/orders", "status": "healthy", "uptime_pct": 99.95, "avg_latency_ms": 45},
				{"url": "/api/payments", "status": "healthy", "uptime_pct": 99.88, "avg_latency_ms": 120},
				{"url": "/api/notifications", "status": "degraded", "uptime_pct": 98.50, "avg_latency_ms": 350},
				{"url": "/api/users", "status": "healthy", "uptime_pct": 99.99, "avg_latency_ms": 30},
			},
			"overall_health": "healthy",
		}
	case WidgetTypeEventTimeline:
		return map[string]interface{}{
			"events": []map[string]interface{}{
				{"timestamp": time.Now().Add(-1 * time.Hour).Format(time.RFC3339), "type": "delivery", "message": "Batch delivery completed", "count": 500},
				{"timestamp": time.Now().Add(-2 * time.Hour).Format(time.RFC3339), "type": "error", "message": "Spike in timeout errors", "count": 25},
				{"timestamp": time.Now().Add(-4 * time.Hour).Format(time.RFC3339), "type": "recovery", "message": "Error rate normalized", "count": 0},
				{"timestamp": time.Now().Add(-6 * time.Hour).Format(time.RFC3339), "type": "config_change", "message": "Retry policy updated", "count": 0},
			},
		}
	case WidgetTypeRealtimeCounter:
		return map[string]interface{}{
			"counters": map[string]interface{}{
				"total_delivered":   52430,
				"total_failed":     128,
				"active_endpoints": 12,
				"avg_latency_ms":   67,
				"success_rate_pct": 99.76,
			},
			"updated_at": time.Now().Format(time.RFC3339),
		}
	default:
		return map[string]interface{}{
			"message": "Unknown widget type",
		}
	}
}

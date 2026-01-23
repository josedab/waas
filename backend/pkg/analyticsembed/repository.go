package analyticsembed

import "context"

// Repository defines the data access interface for embeddable analytics
type Repository interface {
	// Widgets
	CreateWidget(ctx context.Context, widget *WidgetConfig) error
	GetWidget(ctx context.Context, tenantID, widgetID string) (*WidgetConfig, error)
	ListWidgets(ctx context.Context, tenantID string) ([]WidgetConfig, error)
	ListWidgetsByType(ctx context.Context, tenantID, widgetType string) ([]WidgetConfig, error)
	UpdateWidget(ctx context.Context, widget *WidgetConfig) error
	DeleteWidget(ctx context.Context, tenantID, widgetID string) error

	// Embed tokens
	CreateEmbedToken(ctx context.Context, token *EmbedToken) error
	GetEmbedToken(ctx context.Context, tenantID, tokenID string) (*EmbedToken, error)
	GetEmbedTokenByToken(ctx context.Context, token string) (*EmbedToken, error)
	ListEmbedTokens(ctx context.Context, tenantID string) ([]EmbedToken, error)
	DeleteEmbedToken(ctx context.Context, tenantID, tokenID string) error

	// Theme configs
	GetThemeConfig(ctx context.Context, tenantID string) (*ThemeConfig, error)
	UpsertThemeConfig(ctx context.Context, theme *ThemeConfig) error
}

package portal

import "context"

// Repository defines the data access interface for the embeddable portal
type Repository interface {
	// Portal configs
	CreatePortal(ctx context.Context, config *PortalConfig) error
	GetPortal(ctx context.Context, tenantID, portalID string) (*PortalConfig, error)
	GetPortalByTenantID(ctx context.Context, tenantID string) (*PortalConfig, error)
	ListPortals(ctx context.Context, tenantID string) ([]PortalConfig, error)
	UpdatePortal(ctx context.Context, config *PortalConfig) error
	UpdatePortalByTenantID(ctx context.Context, tenantID string, config *PortalConfig) error
	DeletePortal(ctx context.Context, tenantID, portalID string) error

	// Tokens
	CreateToken(ctx context.Context, token *EmbedToken) error
	GetToken(ctx context.Context, token string) (*EmbedToken, error)
	ListTokens(ctx context.Context, tenantID, portalID string) ([]EmbedToken, error)
	RevokeToken(ctx context.Context, tenantID, tokenID string) error

	// Sessions
	CreateSession(ctx context.Context, session *PortalSession) error
	ListSessions(ctx context.Context, tenantID string) ([]PortalSession, error)

	// Portal views
	GetPortalEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]PortalEndpointView, int, error)
	GetPortalDeliveries(ctx context.Context, tenantID string, filter DeliveryFilter, limit, offset int) ([]PortalDeliveryView, int, error)
	GetDelivery(ctx context.Context, tenantID, deliveryID string) (*PortalDeliveryView, error)
	RetryDelivery(ctx context.Context, tenantID, deliveryID string) error
	GetPortalStats(ctx context.Context, tenantID string) (*PortalStats, error)
}

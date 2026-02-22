package portalsdk

import "context"

// Repository defines the data access interface for the portal SDK.
type Repository interface {
	// Portal configs
	CreateConfig(ctx context.Context, config *PortalConfig) error
	GetConfig(ctx context.Context, tenantID, configID string) (*PortalConfig, error)
	ListConfigs(ctx context.Context, tenantID string) ([]PortalConfig, error)
	UpdateConfig(ctx context.Context, config *PortalConfig) error
	DeleteConfig(ctx context.Context, tenantID, configID string) error

	// Sessions
	CreateSession(ctx context.Context, session *PortalSession) error
	GetSession(ctx context.Context, token string) (*PortalSession, error)
	UpdateSession(ctx context.Context, session *PortalSession) error
	RevokeSession(ctx context.Context, sessionID string) error
	ListActiveSessions(ctx context.Context, configID string) ([]PortalSession, error)

	// SDK bundles
	StoreBundle(ctx context.Context, bundle *SDKBundle) error
	GetLatestBundle(ctx context.Context, configID, framework string) (*SDKBundle, error)

	// Usage stats
	GetUsageStats(ctx context.Context, configID string) (*PortalUsageStats, error)
}

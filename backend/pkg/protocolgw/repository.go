package protocolgw

import "context"

// Repository defines the data access interface for protocol gateway management
type Repository interface {
	// Routes
	CreateRoute(ctx context.Context, route *ProtocolRoute) error
	GetRoute(ctx context.Context, tenantID, routeID string) (*ProtocolRoute, error)
	ListRoutes(ctx context.Context, tenantID string) ([]ProtocolRoute, error)
	UpdateRoute(ctx context.Context, route *ProtocolRoute) error
	DeleteRoute(ctx context.Context, tenantID, routeID string) error

	// Messages
	RecordMessage(ctx context.Context, message *ProtocolMessage) error
	ListMessagesByRoute(ctx context.Context, tenantID, routeID string, limit, offset int) ([]ProtocolMessage, error)

	// Stats
	GetRouteStats(ctx context.Context, tenantID, routeID string) (*ProtocolStats, error)
	GetAggregateStats(ctx context.Context, tenantID string) ([]ProtocolStats, error)
}

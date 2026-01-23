package eventmesh

import "context"

// Repository defines the data access interface for event mesh routing
type Repository interface {
	// Routes
	CreateRoute(ctx context.Context, route *Route) error
	GetRoute(ctx context.Context, tenantID, routeID string) (*Route, error)
	ListRoutes(ctx context.Context, tenantID string, limit, offset int) ([]Route, int, error)
	UpdateRoute(ctx context.Context, route *Route) error
	DeleteRoute(ctx context.Context, tenantID, routeID string) error
	GetMatchingRoutes(ctx context.Context, tenantID, eventType string) ([]Route, error)

	// Executions
	SaveExecution(ctx context.Context, exec *RouteExecution) error
	ListExecutions(ctx context.Context, tenantID, routeID string, limit, offset int) ([]RouteExecution, error)

	// Dead letter
	ConfigureDeadLetter(ctx context.Context, config *DeadLetterConfig) error
	GetDeadLetterConfig(ctx context.Context, tenantID, routeID string) (*DeadLetterConfig, error)
	AddDeadLetterEntry(ctx context.Context, entry *DeadLetterEntry) error
	ListDeadLetterEntries(ctx context.Context, tenantID, routeID string, limit, offset int) ([]DeadLetterEntry, int, error)
	DeleteDeadLetterEntry(ctx context.Context, tenantID, entryID string) error

	// Stats
	GetRouteStats(ctx context.Context, tenantID string) (*RouteStats, error)
}

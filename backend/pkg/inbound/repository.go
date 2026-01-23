package inbound

import "context"

// Repository defines the data access interface for inbound webhooks
type Repository interface {
	// InboundSource CRUD
	CreateSource(ctx context.Context, source *InboundSource) error
	GetSource(ctx context.Context, sourceID string) (*InboundSource, error)
	GetSourceByTenant(ctx context.Context, tenantID, sourceID string) (*InboundSource, error)
	ListSources(ctx context.Context, tenantID string, limit, offset int) ([]InboundSource, int, error)
	UpdateSource(ctx context.Context, source *InboundSource) error
	DeleteSource(ctx context.Context, tenantID, sourceID string) error

	// RoutingRule CRUD
	CreateRoutingRule(ctx context.Context, rule *RoutingRule) error
	GetRoutingRules(ctx context.Context, sourceID string) ([]RoutingRule, error)
	UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error
	DeleteRoutingRule(ctx context.Context, ruleID string) error

	// InboundEvent operations
	CreateEvent(ctx context.Context, event *InboundEvent) error
	GetEvent(ctx context.Context, eventID string) (*InboundEvent, error)
	GetEventByTenant(ctx context.Context, tenantID, eventID string) (*InboundEvent, error)
	ListEventsBySource(ctx context.Context, sourceID string, status string, limit, offset int) ([]InboundEvent, int, error)
	UpdateEventStatus(ctx context.Context, eventID, status, errorMsg string) error
}

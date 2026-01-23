package debugger

import "context"

// Repository defines the data access interface for the webhook debugger
type Repository interface {
	// Traces
	SaveTrace(ctx context.Context, trace *DeliveryTrace) error
	GetTrace(ctx context.Context, tenantID, deliveryID string) (*DeliveryTrace, error)
	ListTraces(ctx context.Context, tenantID string, endpointID string, limit, offset int) ([]DeliveryTrace, int, error)

	// Debug sessions
	CreateDebugSession(ctx context.Context, session *DebugSession) error
	GetDebugSession(ctx context.Context, tenantID, sessionID string) (*DebugSession, error)
	UpdateDebugSession(ctx context.Context, session *DebugSession) error
	DeleteDebugSession(ctx context.Context, tenantID, sessionID string) error
	ListDebugSessions(ctx context.Context, tenantID string) ([]DebugSession, error)

	// Delivery data (for replay)
	GetDeliveryPayload(ctx context.Context, tenantID, deliveryID string) (payload string, headers map[string]string, endpointID string, err error)
}

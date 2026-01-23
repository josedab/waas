package tracing

import "context"

// Repository defines the data access interface for distributed tracing
type Repository interface {
	// Traces
	CreateTrace(ctx context.Context, trace *Trace) error
	GetTrace(ctx context.Context, tenantID, traceID string) (*Trace, error)
	ListTraces(ctx context.Context, tenantID string, filter TraceSearchRequest) ([]Trace, int, error)
	UpdateTrace(ctx context.Context, trace *Trace) error
	DeleteTrace(ctx context.Context, tenantID, traceID string) error

	// Spans
	CreateSpan(ctx context.Context, span *Span) error
	GetSpan(ctx context.Context, tenantID, spanID string) (*Span, error)
	ListSpansByTrace(ctx context.Context, tenantID, traceID string) ([]Span, error)
	DeleteSpansByTrace(ctx context.Context, tenantID, traceID string) error

	// Propagation config
	GetPropagationConfig(ctx context.Context, tenantID string) (*PropagationConfig, error)
	UpsertPropagationConfig(ctx context.Context, config *PropagationConfig) error

	// Statistics
	GetTraceStats(ctx context.Context, tenantID string, startTime, endTime string) (*TraceStats, error)
}

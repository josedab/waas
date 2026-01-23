package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Service provides distributed tracing functionality
type Service struct {
	repo Repository
}

// NewService creates a new tracing service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RecordSpan records a new span and updates or creates its parent trace
func (s *Service) RecordSpan(ctx context.Context, tenantID string, req *CreateSpanRequest) (*Span, error) {
	startedAt, err := time.Parse(time.RFC3339, req.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid started_at format: %w", err)
	}

	endedAt := startedAt.Add(time.Duration(req.DurationMs) * time.Millisecond)

	span := &Span{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		TraceID:       req.TraceID,
		SpanID:        req.SpanID,
		ParentSpanID:  req.ParentSpanID,
		OperationName: req.OperationName,
		ServiceName:   req.ServiceName,
		SpanKind:      req.SpanKind,
		StatusCode:    req.StatusCode,
		StatusMessage: req.StatusMessage,
		Attributes:    req.Attributes,
		DurationMs:    req.DurationMs,
		StartedAt:     startedAt,
		EndedAt:       endedAt,
	}

	if span.StatusCode == "" {
		span.StatusCode = "OK"
	}

	if err := s.repo.CreateSpan(ctx, span); err != nil {
		return nil, fmt.Errorf("failed to create span: %w", err)
	}

	// Update or create the parent trace
	trace, err := s.repo.GetTrace(ctx, tenantID, req.TraceID)
	if err != nil {
		// Create new trace
		trace = &Trace{
			ID:            uuid.New().String(),
			TenantID:      tenantID,
			TraceID:       req.TraceID,
			RootSpanID:    req.SpanID,
			ServiceName:   req.ServiceName,
			OperationName: req.OperationName,
			Status:        TraceStatusActive,
			SpanCount:     1,
			DurationMs:    req.DurationMs,
			HasErrors:     span.StatusCode == "ERROR",
			StartedAt:     startedAt,
			EndedAt:       endedAt,
			CreatedAt:     time.Now(),
		}
		if err := s.repo.CreateTrace(ctx, trace); err != nil {
			return nil, fmt.Errorf("failed to create trace: %w", err)
		}
	} else {
		trace.SpanCount++
		if startedAt.Before(trace.StartedAt) {
			trace.StartedAt = startedAt
		}
		if endedAt.After(trace.EndedAt) {
			trace.EndedAt = endedAt
		}
		trace.DurationMs = trace.EndedAt.Sub(trace.StartedAt).Milliseconds()
		if span.StatusCode == "ERROR" {
			trace.HasErrors = true
		}
		if err := s.repo.UpdateTrace(ctx, trace); err != nil {
			return nil, fmt.Errorf("failed to update trace: %w", err)
		}
	}

	return span, nil
}

// GetTrace retrieves a trace by its trace ID
func (s *Service) GetTrace(ctx context.Context, tenantID, traceID string) (*Trace, error) {
	return s.repo.GetTrace(ctx, tenantID, traceID)
}

// SearchTraces searches traces with filters
func (s *Service) SearchTraces(ctx context.Context, tenantID string, filter TraceSearchRequest) ([]Trace, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	return s.repo.ListTraces(ctx, tenantID, filter)
}

// GetSpanWaterfall builds a hierarchical span waterfall for visualization
func (s *Service) GetSpanWaterfall(ctx context.Context, tenantID, traceID string) (*SpanWaterfall, error) {
	spans, err := s.repo.ListSpansByTrace(ctx, tenantID, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list spans: %w", err)
	}

	if len(spans) == 0 {
		return nil, fmt.Errorf("no spans found for trace %s", traceID)
	}

	// Build service map
	serviceMap := make(map[string]int)
	for _, sp := range spans {
		serviceMap[sp.ServiceName]++
	}

	// Build tree structure
	spanMap := make(map[string]*SpanNode)
	for i := range spans {
		spanMap[spans[i].SpanID] = &SpanNode{Span: spans[i]}
	}

	var root *SpanNode
	for _, node := range spanMap {
		if node.Span.ParentSpanID == "" {
			root = node
		} else if parent, ok := spanMap[node.Span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, *node)
		}
	}

	// If no explicit root found, use the earliest span
	if root == nil {
		sort.Slice(spans, func(i, j int) bool {
			return spans[i].StartedAt.Before(spans[j].StartedAt)
		})
		root = spanMap[spans[0].SpanID]
	}

	// Sort children by start time
	sortSpanNodeChildren(root)

	var totalDuration int64
	for _, sp := range spans {
		if sp.DurationMs > totalDuration {
			totalDuration = sp.DurationMs
		}
	}

	return &SpanWaterfall{
		TraceID:    traceID,
		RootSpan:   root,
		TotalSpans: len(spans),
		DurationMs: totalDuration,
		ServiceMap: serviceMap,
	}, nil
}

func sortSpanNodeChildren(node *SpanNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Span.StartedAt.Before(node.Children[j].Span.StartedAt)
	})
	for i := range node.Children {
		sortSpanNodeChildren(&node.Children[i])
	}
}

// GenerateTraceContext creates a new W3C TraceContext for outgoing webhook deliveries
func (s *Service) GenerateTraceContext(ctx context.Context, tenantID string) (*TraceContext, error) {
	config, _ := s.repo.GetPropagationConfig(ctx, tenantID)
	if config != nil && !config.IsActive {
		return nil, nil
	}

	traceIDBytes := make([]byte, 16)
	spanIDBytes := make([]byte, 8)
	if _, err := rand.Read(traceIDBytes); err != nil {
		return nil, fmt.Errorf("failed to generate trace ID: %w", err)
	}
	if _, err := rand.Read(spanIDBytes); err != nil {
		return nil, fmt.Errorf("failed to generate span ID: %w", err)
	}

	return &TraceContext{
		TraceID:    hex.EncodeToString(traceIDBytes),
		SpanID:     hex.EncodeToString(spanIDBytes),
		TraceFlags: "01", // sampled
	}, nil
}

// GetPropagationConfig retrieves the trace propagation configuration
func (s *Service) GetPropagationConfig(ctx context.Context, tenantID string) (*PropagationConfig, error) {
	config, err := s.repo.GetPropagationConfig(ctx, tenantID)
	if err != nil {
		// Return defaults
		return &PropagationConfig{
			TenantID:      tenantID,
			InjectHeaders: true,
			InjectPayload: false,
			HeaderPrefix:  "traceparent",
			PayloadField:  "_trace",
			SamplingRate:  1.0,
			IsActive:      true,
		}, nil
	}
	return config, nil
}

// UpdatePropagationConfig updates the trace propagation configuration
func (s *Service) UpdatePropagationConfig(ctx context.Context, tenantID string, req *UpdatePropagationConfigRequest) (*PropagationConfig, error) {
	config := &PropagationConfig{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		InjectHeaders: req.InjectHeaders,
		InjectPayload: req.InjectPayload,
		HeaderPrefix:  req.HeaderPrefix,
		PayloadField:  req.PayloadField,
		SamplingRate:  req.SamplingRate,
		IsActive:      req.IsActive,
		UpdatedAt:     time.Now(),
	}

	if config.HeaderPrefix == "" {
		config.HeaderPrefix = "traceparent"
	}
	if config.PayloadField == "" {
		config.PayloadField = "_trace"
	}
	if config.SamplingRate <= 0 {
		config.SamplingRate = 1.0
	}

	if err := s.repo.UpsertPropagationConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update propagation config: %w", err)
	}

	return config, nil
}

// GetTraceStats returns aggregate trace statistics
func (s *Service) GetTraceStats(ctx context.Context, tenantID, startTime, endTime string) (*TraceStats, error) {
	return s.repo.GetTraceStats(ctx, tenantID, startTime, endTime)
}

// CompleteTrace marks a trace as completed
func (s *Service) CompleteTrace(ctx context.Context, tenantID, traceID string) (*Trace, error) {
	trace, err := s.repo.GetTrace(ctx, tenantID, traceID)
	if err != nil {
		return nil, err
	}

	if trace.HasErrors {
		trace.Status = TraceStatusError
	} else {
		trace.Status = TraceStatusCompleted
	}
	trace.EndedAt = time.Now()
	trace.DurationMs = trace.EndedAt.Sub(trace.StartedAt).Milliseconds()

	if err := s.repo.UpdateTrace(ctx, trace); err != nil {
		return nil, fmt.Errorf("failed to complete trace: %w", err)
	}

	return trace, nil
}

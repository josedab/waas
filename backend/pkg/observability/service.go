package observability

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
)

// OTelExporter handles exporting traces to OpenTelemetry collectors
type OTelExporter struct {
	endpoint string
	enabled  bool
}

// Service provides observability functionality
type Service struct {
	repo       Repository
	propagator *TraceContextPropagator
	exporter   *OTelExporter
}

// NewService creates a new observability service
func NewService(repo Repository) *Service {
	return &Service{
		repo:       repo,
		propagator: NewTraceContextPropagator(),
	}
}

// StartSpan starts a new span
func (s *Service) StartSpan(ctx context.Context, tenantID string, req *CreateSpanRequest) (*WebhookSpan, error) {
	traceCtx := TraceContextFromContext(ctx)
	
	var traceID, parentSpanID string
	if traceCtx != nil {
		traceID = traceCtx.TraceID
		parentSpanID = traceCtx.SpanID
	} else if req.TraceID != "" {
		traceID = req.TraceID
		parentSpanID = req.ParentSpanID
	} else {
		newCtx, err := s.propagator.NewTraceContext()
		if err != nil {
			return nil, err
		}
		traceID = newCtx.TraceID
	}

	spanID, _ := generateSpanID()

	span := &WebhookSpan{
		ID:            uuid.New().String(),
		TraceID:       traceID,
		SpanID:        spanID,
		ParentSpanID:  parentSpanID,
		TenantID:      tenantID,
		OperationName: req.OperationName,
		ServiceName:   req.ServiceName,
		Kind:          req.Kind,
		Status:        SpanStatusUnset,
		StartTime:     time.Now(),
		Attributes:    req.Attributes,
		CreatedAt:     time.Now(),
	}

	if span.Kind == "" {
		span.Kind = SpanKindInternal
	}

	if err := s.repo.SaveSpan(ctx, span); err != nil {
		return nil, err
	}

	return span, nil
}

// EndSpan ends a span
func (s *Service) EndSpan(ctx context.Context, tenantID string, span *WebhookSpan, status SpanStatus, statusMsg string) error {
	span.EndTime = time.Now()
	span.DurationMs = span.EndTime.Sub(span.StartTime).Milliseconds()
	span.Status = status
	span.StatusMessage = statusMsg

	return s.repo.SaveSpan(ctx, span)
}

// AddSpanEvent adds an event to a span
func (s *Service) AddSpanEvent(ctx context.Context, span *WebhookSpan, name string, attrs map[string]interface{}) {
	span.Events = append(span.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// GetTrace retrieves a complete trace
func (s *Service) GetTrace(ctx context.Context, tenantID, traceID string) (*Trace, error) {
	return s.repo.GetTrace(ctx, tenantID, traceID)
}

// SearchTraces searches for traces
func (s *Service) SearchTraces(ctx context.Context, query *TraceSearchQuery) (*TraceSearchResult, error) {
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	return s.repo.SearchTraces(ctx, query)
}

// GetTraceTimeline generates a timeline view
func (s *Service) GetTraceTimeline(ctx context.Context, tenantID, traceID string) (*TraceTimeline, error) {
	trace, err := s.repo.GetTrace(ctx, tenantID, traceID)
	if err != nil {
		return nil, err
	}

	timeline := &TraceTimeline{
		TraceID:   traceID,
		StartTime: trace.StartTime,
		EndTime:   trace.EndTime,
		TotalMs:   trace.DurationMs,
	}

	// Build span tree and calculate depths
	spanMap := make(map[string]*WebhookSpan)
	children := make(map[string][]string)
	
	for i := range trace.Spans {
		span := &trace.Spans[i]
		spanMap[span.SpanID] = span
		if span.ParentSpanID != "" {
			children[span.ParentSpanID] = append(children[span.ParentSpanID], span.SpanID)
		}
	}

	// Calculate depths using BFS
	depths := make(map[string]int)
	var rootSpanID string
	for _, span := range trace.Spans {
		if span.ParentSpanID == "" {
			rootSpanID = span.SpanID
			break
		}
	}

	if rootSpanID != "" {
		queue := []string{rootSpanID}
		depths[rootSpanID] = 0
		for len(queue) > 0 {
			spanID := queue[0]
			queue = queue[1:]
			for _, childID := range children[spanID] {
				depths[childID] = depths[spanID] + 1
				queue = append(queue, childID)
			}
		}
	}

	// Build timeline spans
	for _, span := range trace.Spans {
		startOffset := span.StartTime.Sub(trace.StartTime).Milliseconds()
		percentOfRoot := float64(span.DurationMs) / float64(trace.DurationMs) * 100

		timelineSpan := TimelineSpan{
			SpanID:        span.SpanID,
			OperationName: span.OperationName,
			ServiceName:   span.ServiceName,
			StartOffset:   startOffset,
			DurationMs:    span.DurationMs,
			DepthLevel:    depths[span.SpanID],
			Status:        string(span.Status),
			PercentOfRoot: percentOfRoot,
		}
		timeline.Spans = append(timeline.Spans, timelineSpan)
	}

	// Sort by start time
	sort.Slice(timeline.Spans, func(i, j int) bool {
		return timeline.Spans[i].StartOffset < timeline.Spans[j].StartOffset
	})

	// Build waterfall
	for _, span := range timeline.Spans {
		startPct := float64(span.StartOffset) / float64(trace.DurationMs) * 100
		widthPct := float64(span.DurationMs) / float64(trace.DurationMs) * 100

		row := WaterfallRow{
			SpanID:   span.SpanID,
			Label:    span.ServiceName + " - " + span.OperationName,
			Depth:    span.DepthLevel,
			HasError: span.Status == string(SpanStatusError),
			Bars: []WaterfallBar{
				{
					Type:       "active",
					StartPct:   startPct,
					WidthPct:   widthPct,
					DurationMs: span.DurationMs,
				},
			},
		}
		timeline.Waterfall = append(timeline.Waterfall, row)
	}

	// Find critical path (longest path through the trace)
	timeline.CritPath = s.findCriticalPath(trace.Spans, spanMap, children, rootSpanID)

	return timeline, nil
}

func (s *Service) findCriticalPath(spans []WebhookSpan, spanMap map[string]*WebhookSpan, children map[string][]string, rootID string) []string {
	if rootID == "" {
		return nil
	}

	var path []string
	currentID := rootID

	for currentID != "" {
		path = append(path, currentID)
		
		childIDs := children[currentID]
		if len(childIDs) == 0 {
			break
		}

		// Find the child with the longest duration
		var longestChildID string
		var longestDuration int64
		for _, childID := range childIDs {
			if child, ok := spanMap[childID]; ok {
				if child.DurationMs > longestDuration {
					longestDuration = child.DurationMs
					longestChildID = childID
				}
			}
		}
		currentID = longestChildID
	}

	return path
}

// GetLatencyBreakdown analyzes latency for a trace
func (s *Service) GetLatencyBreakdown(ctx context.Context, tenantID, traceID string) (*LatencyBreakdown, error) {
	trace, err := s.repo.GetTrace(ctx, tenantID, traceID)
	if err != nil {
		return nil, err
	}

	breakdown := &LatencyBreakdown{
		TraceID:     traceID,
		TotalMs:     trace.DurationMs,
		ByService:   make(map[string]int64),
		ByOperation: make(map[string]int64),
	}

	var maxServiceTime int64
	var bottleneckService string

	for _, span := range trace.Spans {
		breakdown.ByService[span.ServiceName] += span.DurationMs
		breakdown.ByOperation[span.OperationName] += span.DurationMs

		// Estimate network time from spans with "http" or "grpc" in operation
		if span.Kind == SpanKindClient {
			breakdown.NetworkTime += span.DurationMs / 10 // Approximate
		}

		if breakdown.ByService[span.ServiceName] > maxServiceTime {
			maxServiceTime = breakdown.ByService[span.ServiceName]
			bottleneckService = span.ServiceName
		}
	}

	breakdown.ProcessingTime = trace.DurationMs - breakdown.NetworkTime - breakdown.QueueTime
	breakdown.Bottleneck = bottleneckService
	if trace.DurationMs > 0 {
		breakdown.BottleneckPct = float64(maxServiceTime) / float64(trace.DurationMs) * 100
	}

	// Generate recommendations
	if breakdown.NetworkTime > trace.DurationMs/3 {
		breakdown.Recommendations = append(breakdown.Recommendations,
			"High network latency detected. Consider using connection pooling or regional deployments.")
	}
	if breakdown.BottleneckPct > 70 {
		breakdown.Recommendations = append(breakdown.Recommendations,
			"Single service bottleneck detected in "+bottleneckService+". Consider optimizing or scaling this service.")
	}

	return breakdown, nil
}

// GetTraceMetrics retrieves aggregated metrics
func (s *Service) GetTraceMetrics(ctx context.Context, tenantID string, start, end time.Time) (*TraceMetrics, error) {
	return s.repo.GetTraceMetrics(ctx, tenantID, start, end)
}

// GetServiceMap retrieves the service dependency map
func (s *Service) GetServiceMap(ctx context.Context, tenantID string, start, end time.Time) (*ServiceMap, error) {
	return s.repo.GetServiceMap(ctx, tenantID, start, end)
}

// CreateExportConfig creates an OTel export configuration
func (s *Service) CreateExportConfig(ctx context.Context, tenantID string, req *CreateExportConfigRequest) (*OTelExportConfig, error) {
	config := &OTelExportConfig{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Name:      req.Name,
		Enabled:   true,
		Protocol:  req.Protocol,
		Endpoint:  req.Endpoint,
		Headers:   req.Headers,
		Sampling:  req.Sampling,
		BatchSize: req.BatchSize,
		Timeout:   req.Timeout,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.Sampling.Strategy == "" {
		config.Sampling.Strategy = "always"
	}

	if err := s.repo.SaveExportConfig(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// ListExportConfigs lists export configurations
func (s *Service) ListExportConfigs(ctx context.Context, tenantID string) ([]OTelExportConfig, error) {
	return s.repo.ListExportConfigs(ctx, tenantID)
}

// DeleteExportConfig deletes an export configuration
func (s *Service) DeleteExportConfig(ctx context.Context, tenantID, configID string) error {
	return s.repo.DeleteExportConfig(ctx, tenantID, configID)
}

// InjectTraceContext injects trace context into webhook headers
func (s *Service) InjectTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}

	traceCtx := TraceContextFromContext(ctx)
	if traceCtx == nil {
		newCtx, err := s.propagator.NewTraceContext()
		if err != nil {
			return headers
		}
		traceCtx = newCtx
	}

	s.propagator.Inject(traceCtx, headers)
	return headers
}

// ExtractTraceContext extracts trace context from headers
func (s *Service) ExtractTraceContext(headers map[string]string) (*TraceContext, error) {
	return s.propagator.Extract(headers)
}

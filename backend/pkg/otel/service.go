package otel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service manages OpenTelemetry configuration and tracing
type Service struct {
	repo    *Repository
	tracers map[string]*Tracer
	mu      sync.RWMutex
}

// NewService creates a new OTEL service
func NewService(repo *Repository) *Service {
	return &Service{
		repo:    repo,
		tracers: make(map[string]*Tracer),
	}
}

// CreateConfig creates a new OTEL configuration
func (s *Service) CreateConfig(ctx context.Context, tenantID string, req *CreateConfigRequest) (*Config, error) {
	config := &Config{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		ServiceName: req.ServiceName,
		Enabled:     req.Enabled,
		Traces:      req.Traces,
		Metrics:     req.Metrics,
		Logs:        req.Logs,
		Attributes:  req.Attributes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if config.ServiceName == "" {
		config.ServiceName = "waas-webhook-service"
	}

	if err := s.repo.Create(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Initialize tracer if enabled
	if config.Enabled {
		s.initTracer(ctx, config)
	}

	return config, nil
}

// GetConfig retrieves an OTEL configuration
func (s *Service) GetConfig(ctx context.Context, tenantID, configID string) (*Config, error) {
	return s.repo.GetByID(ctx, tenantID, configID)
}

// ListConfigs lists OTEL configurations for a tenant
func (s *Service) ListConfigs(ctx context.Context, tenantID string, limit, offset int) ([]*Config, int, error) {
	return s.repo.List(ctx, tenantID, limit, offset)
}

// UpdateConfig updates an OTEL configuration
func (s *Service) UpdateConfig(ctx context.Context, tenantID, configID string, req *UpdateConfigRequest) (*Config, error) {
	config, err := s.repo.GetByID(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("config not found")
	}

	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.ServiceName != nil {
		config.ServiceName = *req.ServiceName
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Traces != nil {
		config.Traces = *req.Traces
	}
	if req.Metrics != nil {
		config.Metrics = *req.Metrics
	}
	if req.Logs != nil {
		config.Logs = *req.Logs
	}
	if req.Attributes != nil {
		config.Attributes = req.Attributes
	}

	config.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// Reinitialize tracer
	s.mu.Lock()
	if tracer, exists := s.tracers[tenantID]; exists {
		tracer.Stop(ctx)
		delete(s.tracers, tenantID)
	}
	s.mu.Unlock()

	if config.Enabled {
		s.initTracer(ctx, config)
	}

	return config, nil
}

// DeleteConfig deletes an OTEL configuration
func (s *Service) DeleteConfig(ctx context.Context, tenantID, configID string) error {
	// Stop tracer if running
	s.mu.Lock()
	if tracer, exists := s.tracers[tenantID]; exists {
		tracer.Stop(ctx)
		delete(s.tracers, tenantID)
	}
	s.mu.Unlock()

	return s.repo.Delete(ctx, tenantID, configID)
}

// GetTracer retrieves or creates a tracer for a tenant
func (s *Service) GetTracer(ctx context.Context, tenantID string) (*Tracer, error) {
	s.mu.RLock()
	if tracer, exists := s.tracers[tenantID]; exists {
		s.mu.RUnlock()
		return tracer, nil
	}
	s.mu.RUnlock()

	// Load config and initialize tracer
	config, err := s.repo.GetDefault(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if config == nil || !config.Enabled {
		// Return a no-op tracer
		return NewTracer(DefaultConfig(tenantID)), nil
	}

	return s.initTracer(ctx, config), nil
}

// initTracer initializes and starts a tracer for a config
func (s *Service) initTracer(ctx context.Context, config *Config) *Tracer {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracer := NewTracer(config)

	// Add exporters based on config
	if config.Traces.Enabled {
		switch config.Traces.Exporter {
		case ExporterOTLP, ExporterOTLPHTTP:
			if config.Traces.Endpoint != "" {
				tracer.AddExporter(NewOTLPExporter(config.Traces.Endpoint, config.Traces.Headers))
			}
		case ExporterStdout:
			tracer.AddExporter(NewStdoutExporter())
		}
	}

	if config.Metrics.Enabled {
		switch config.Metrics.Exporter {
		case ExporterOTLP, ExporterOTLPHTTP:
			if config.Metrics.Endpoint != "" {
				tracer.AddExporter(NewOTLPExporter(config.Metrics.Endpoint, config.Metrics.Headers))
			}
		case ExporterPrometheus:
			tracer.AddExporter(NewPrometheusExporter())
		case ExporterStdout:
			tracer.AddExporter(NewStdoutExporter())
		}
	}

	tracer.Start(ctx)
	s.tracers[config.TenantID] = tracer

	return tracer
}

// TraceDelivery creates a traced delivery operation
func (s *Service) TraceDelivery(ctx context.Context, tenantID string, attrs DeliverySpanAttributes) (context.Context, func(error)) {
	tracer, err := s.GetTracer(ctx, tenantID)
	if err != nil || tracer == nil {
		return ctx, func(error) {}
	}

	ctx, span := tracer.StartSpan(ctx, StandardSpanNames.DeliveryProcess, WithAttributes(map[string]any{
		"webhook.id":               attrs.WebhookID,
		"webhook.endpoint_id":      attrs.EndpointID,
		"webhook.endpoint_url":     attrs.EndpointURL,
		"webhook.delivery_id":      attrs.DeliveryID,
		"webhook.delivery_attempt": attrs.DeliveryAttempt,
		"webhook.payload_size":     attrs.PayloadSizeBytes,
	}))

	// Record delivery metric
	tracer.IncrementCounter(StandardMetricNames.DeliveriesTotal, map[string]string{
		"endpoint_id": attrs.EndpointID,
	})

	return ctx, func(err error) {
		if err != nil {
			span.SetError(err)
			tracer.IncrementCounter(StandardMetricNames.DeliveriesFailed, map[string]string{
				"endpoint_id": attrs.EndpointID,
				"error_type":  attrs.ErrorType,
			})
		} else {
			span.SetStatus(0, "OK")
			tracer.IncrementCounter(StandardMetricNames.DeliveriesSuccessful, map[string]string{
				"endpoint_id": attrs.EndpointID,
			})
		}

		if attrs.ResponseStatus > 0 {
			span.SetAttribute("http.response_status_code", attrs.ResponseStatus)
		}

		span.End()
	}
}

// RecordDeliveryDuration records delivery duration histogram
func (s *Service) RecordDeliveryDuration(ctx context.Context, tenantID, endpointID string, duration time.Duration) {
	tracer, err := s.GetTracer(ctx, tenantID)
	if err != nil || tracer == nil {
		return
	}

	tracer.RecordHistogram(StandardMetricNames.DeliveryDuration, float64(duration.Milliseconds()), map[string]string{
		"endpoint_id": endpointID,
	})
}

// RecordQueueDepth records current queue depth gauge
func (s *Service) RecordQueueDepth(ctx context.Context, tenantID string, depth int) {
	tracer, err := s.GetTracer(ctx, tenantID)
	if err != nil || tracer == nil {
		return
	}

	tracer.RecordGauge(StandardMetricNames.QueueDepth, float64(depth), map[string]string{
		"tenant_id": tenantID,
	})
}

// Shutdown gracefully shuts down all tracers
func (s *Service) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID, tracer := range s.tracers {
		if err := tracer.Stop(ctx); err != nil {
			fmt.Printf("error stopping tracer for tenant %s: %v\n", tenantID, err)
		}
	}

	s.tracers = make(map[string]*Tracer)
	return nil
}

// TestConnection tests the OTEL endpoint connection
func (s *Service) TestConnection(ctx context.Context, config *Config) error {
	tracer := NewTracer(config)

	if config.Traces.Enabled && config.Traces.Endpoint != "" {
		exporter := NewOTLPExporter(config.Traces.Endpoint, config.Traces.Headers)

		// Send a test span
		testSpan := &SpanData{
			TraceID:       "test-trace-id",
			SpanID:        "test-span-id",
			OperationName: "connection-test",
			ServiceName:   config.ServiceName,
			StartTime:     time.Now(),
			EndTime:       time.Now(),
			Duration:      time.Millisecond,
			Status:        SpanStatus{Code: 0, Description: "test"},
			Attributes:    map[string]any{"test": true},
		}

		if err := exporter.ExportSpans(ctx, []*SpanData{testSpan}); err != nil {
			return fmt.Errorf("traces endpoint test failed: %w", err)
		}

		tracer.AddExporter(exporter)
	}

	return nil
}

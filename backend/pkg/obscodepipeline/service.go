package obscodepipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides observability-as-code pipeline business logic.
type Service struct {
	repo Repository
}

// NewService creates a new observability pipeline service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreatePipeline validates and stores a new observability pipeline definition.
func (s *Service) CreatePipeline(ctx context.Context, tenantID string, req *CreatePipelineRequest) (*ObservabilityPipeline, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}

	spec, err := s.parseAndValidateSpec(req.Spec)
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline spec: %w", err)
	}

	checksum := computeChecksum(req.Spec)

	pipeline := &ObservabilityPipeline{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		Status:      PipelineStatusDraft,
		Spec:        req.Spec,
		Checksum:    checksum,
		Signals:     spec.Signals,
		Exporters:   spec.Exporters,
		Alerts:      spec.Alerts,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if s.repo != nil {
		if err := s.repo.CreatePipeline(ctx, pipeline); err != nil {
			return nil, fmt.Errorf("failed to create pipeline: %w", err)
		}
	}

	return pipeline, nil
}

// GetPipeline retrieves a pipeline by ID.
func (s *Service) GetPipeline(ctx context.Context, tenantID, pipelineID string) (*ObservabilityPipeline, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetPipeline(ctx, tenantID, pipelineID)
}

// ListPipelines retrieves all pipelines for a tenant.
func (s *Service) ListPipelines(ctx context.Context, tenantID string) ([]ObservabilityPipeline, error) {
	if s.repo == nil {
		return []ObservabilityPipeline{}, nil
	}
	return s.repo.ListPipelines(ctx, tenantID)
}

// UpdatePipeline validates and applies changes to an existing pipeline.
func (s *Service) UpdatePipeline(ctx context.Context, tenantID, pipelineID string, req *UpdatePipelineRequest) (*ObservabilityPipeline, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	pipeline, err := s.repo.GetPipeline(ctx, tenantID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if req.Name != "" {
		pipeline.Name = req.Name
	}
	if req.Description != "" {
		pipeline.Description = req.Description
	}
	if req.Status != "" {
		if err := s.validateStatusTransition(pipeline.Status, req.Status); err != nil {
			return nil, err
		}
		pipeline.Status = req.Status
	}
	if req.Spec != nil {
		spec, err := s.parseAndValidateSpec(req.Spec)
		if err != nil {
			return nil, fmt.Errorf("invalid pipeline spec: %w", err)
		}
		pipeline.Spec = req.Spec
		pipeline.Checksum = computeChecksum(req.Spec)
		pipeline.Signals = spec.Signals
		pipeline.Exporters = spec.Exporters
		pipeline.Alerts = spec.Alerts
		pipeline.Version++
	}

	pipeline.UpdatedAt = time.Now()

	if err := s.repo.UpdatePipeline(ctx, pipeline); err != nil {
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}

	return pipeline, nil
}

// DeletePipeline removes a pipeline definition.
func (s *Service) DeletePipeline(ctx context.Context, tenantID, pipelineID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.DeletePipeline(ctx, tenantID, pipelineID)
}

// ActivatePipeline transitions a pipeline to active status.
func (s *Service) ActivatePipeline(ctx context.Context, tenantID, pipelineID string) (*ObservabilityPipeline, error) {
	return s.UpdatePipeline(ctx, tenantID, pipelineID, &UpdatePipelineRequest{Status: PipelineStatusActive})
}

// PausePipeline transitions a pipeline to paused status.
func (s *Service) PausePipeline(ctx context.Context, tenantID, pipelineID string) (*ObservabilityPipeline, error) {
	return s.UpdatePipeline(ctx, tenantID, pipelineID, &UpdatePipelineRequest{Status: PipelineStatusPaused})
}

// ValidateSpec validates a pipeline spec without persisting it.
func (s *Service) ValidateSpec(ctx context.Context, spec json.RawMessage) ([]string, error) {
	_, err := s.parseAndValidateSpec(spec)
	if err != nil {
		return []string{err.Error()}, err
	}
	return nil, nil
}

// GetPipelineStats returns aggregated stats for a pipeline.
func (s *Service) GetPipelineStats(ctx context.Context, pipelineID string) (*PipelineStats, error) {
	if s.repo == nil {
		return &PipelineStats{PipelineID: pipelineID}, nil
	}
	return s.repo.GetPipelineStats(ctx, pipelineID)
}

// ListExecutions returns recent executions of a pipeline.
func (s *Service) ListExecutions(ctx context.Context, pipelineID string, limit, offset int) ([]PipelineExecution, error) {
	if s.repo == nil {
		return []PipelineExecution{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListExecutions(ctx, pipelineID, limit, offset)
}

// ListAlertEvents returns recent alert events for a tenant.
func (s *Service) ListAlertEvents(ctx context.Context, tenantID string, limit, offset int) ([]AlertEvent, error) {
	if s.repo == nil {
		return []AlertEvent{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListAlertEvents(ctx, tenantID, limit, offset)
}

// GetActiveAlerts returns currently active (unresolved) alerts.
func (s *Service) GetActiveAlerts(ctx context.Context, tenantID string) ([]AlertEvent, error) {
	if s.repo == nil {
		return []AlertEvent{}, nil
	}
	return s.repo.GetActiveAlerts(ctx, tenantID)
}

// pipelineSpec is the internal representation of a parsed spec.
type pipelineSpec struct {
	Signals   []SignalConfig   `json:"signals" yaml:"signals"`
	Exporters []ExporterConfig `json:"exporters" yaml:"exporters"`
	Alerts    []AlertRule      `json:"alerts,omitempty" yaml:"alerts"`
}

func (s *Service) parseAndValidateSpec(raw json.RawMessage) (*pipelineSpec, error) {
	var spec pipelineSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	if len(spec.Signals) == 0 {
		return nil, fmt.Errorf("at least one signal is required")
	}

	validSignals := map[string]bool{SignalMetrics: true, SignalTraces: true, SignalLogs: true}
	for _, sig := range spec.Signals {
		if !validSignals[sig.Type] {
			return nil, fmt.Errorf("invalid signal type %q: must be one of metrics, traces, logs", sig.Type)
		}
		if sig.SampleRate < 0 || sig.SampleRate > 1 {
			return nil, fmt.Errorf("sample_rate for signal %q must be between 0 and 1", sig.Type)
		}
	}

	if len(spec.Exporters) == 0 {
		return nil, fmt.Errorf("at least one exporter is required")
	}

	validExporters := map[string]bool{
		ExporterPrometheus: true, ExporterDatadog: true, ExporterOTLP: true,
		ExporterCloudWatch: true, ExporterElastic: true, ExporterWebhook: true,
	}
	for _, exp := range spec.Exporters {
		if exp.Name == "" {
			return nil, fmt.Errorf("exporter name is required")
		}
		if !validExporters[exp.Type] {
			return nil, fmt.Errorf("invalid exporter type %q", exp.Type)
		}
		if exp.Endpoint == "" {
			return nil, fmt.Errorf("exporter %q requires an endpoint", exp.Name)
		}
		if len(exp.Signals) == 0 {
			return nil, fmt.Errorf("exporter %q must specify at least one signal", exp.Name)
		}
	}

	validSeverities := map[string]bool{AlertSeverityCritical: true, AlertSeverityWarning: true, AlertSeverityInfo: true}
	for _, alert := range spec.Alerts {
		if alert.Name == "" {
			return nil, fmt.Errorf("alert rule name is required")
		}
		if alert.Severity != "" && !validSeverities[alert.Severity] {
			return nil, fmt.Errorf("invalid alert severity %q", alert.Severity)
		}
	}

	return &spec, nil
}

func (s *Service) validateStatusTransition(current, next string) error {
	allowed := map[string][]string{
		PipelineStatusDraft:    {PipelineStatusActive, PipelineStatusArchived},
		PipelineStatusActive:   {PipelineStatusPaused, PipelineStatusFailed, PipelineStatusArchived},
		PipelineStatusPaused:   {PipelineStatusActive, PipelineStatusArchived},
		PipelineStatusFailed:   {PipelineStatusActive, PipelineStatusArchived},
		PipelineStatusArchived: {},
	}

	transitions, ok := allowed[current]
	if !ok {
		return fmt.Errorf("unknown pipeline status %q", current)
	}

	for _, t := range transitions {
		if t == next {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %q to %q; allowed: %s", current, next, strings.Join(transitions, ", "))
}

func computeChecksum(data json.RawMessage) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

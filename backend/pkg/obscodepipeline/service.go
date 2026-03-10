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

// ParseObsConfig parses and validates a waas-obs.yaml configuration from JSON/YAML.
func (s *Service) ParseObsConfig(ctx context.Context, raw json.RawMessage) (*ObsConfig, error) {
	var config ObsConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("failed to parse obs config: %w", err)
	}

	if err := s.validateObsConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid obs config: %w", err)
	}

	return &config, nil
}

// ApplyConfig reconciles the desired observability config against actual state.
func (s *Service) ApplyConfig(ctx context.Context, tenantID string, req *ApplyConfigRequest) (*ReconcileResult, error) {
	config, err := s.ParseObsConfig(ctx, req.Config)
	if err != nil {
		return nil, err
	}

	result := &ReconcileResult{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		ConfigChecksum: computeChecksum(req.Config),
		Status:         ReconcileStatusRunning,
		StartedAt:      time.Now(),
	}

	if req.DryRun {
		drift, err := s.detectDrift(ctx, tenantID, config)
		if err != nil {
			result.Status = ReconcileStatusFailed
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Drift = drift
			if len(drift) == 0 {
				result.Status = ReconcileStatusConverged
			} else {
				result.Status = ReconcileStatusDiverged
			}
		}
		now := time.Now()
		result.CompletedAt = &now
		return result, nil
	}

	// Reconcile dashboards
	for _, dash := range config.Dashboards {
		if err := s.reconcileDashboard(ctx, tenantID, &dash); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("dashboard %q: %s", dash.Name, err))
		} else {
			result.DashboardsSync++
		}
	}

	// Reconcile alert rules
	for _, alert := range config.AlertRules {
		if err := s.reconcileAlertRule(ctx, tenantID, &alert); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("alert_rule %q: %s", alert.Name, err))
		} else {
			result.AlertRulesSync++
		}
	}

	// Reconcile SLOs
	for _, slo := range config.SLOs {
		if err := s.reconcileSLO(ctx, tenantID, &slo); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("slo %q: %s", slo.Name, err))
		} else {
			result.SLOsSync++
		}
	}

	// Reconcile integrations
	for _, integ := range config.Integrations {
		if err := s.reconcileIntegration(ctx, tenantID, &integ); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("integration %q: %s", integ.Name, err))
		} else {
			result.IntegrationsSync++
		}
	}

	if len(result.Errors) > 0 {
		result.Status = ReconcileStatusDiverged
	} else {
		result.Status = ReconcileStatusConverged
	}

	now := time.Now()
	result.CompletedAt = &now
	return result, nil
}

// CheckDrift detects differences between desired config and actual state.
func (s *Service) CheckDrift(ctx context.Context, tenantID string, raw json.RawMessage) (*DriftCheckResponse, error) {
	config, err := s.ParseObsConfig(ctx, raw)
	if err != nil {
		return nil, err
	}

	drift, err := s.detectDrift(ctx, tenantID, config)
	if err != nil {
		return nil, fmt.Errorf("drift detection failed: %w", err)
	}

	return &DriftCheckResponse{
		HasDrift: len(drift) > 0,
		Drift:    drift,
		Checksum: computeChecksum(raw),
	}, nil
}

func (s *Service) validateObsConfig(config *ObsConfig) error {
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	for i, dash := range config.Dashboards {
		if dash.Name == "" {
			return fmt.Errorf("dashboard[%d]: name is required", i)
		}
		if len(dash.Panels) == 0 {
			return fmt.Errorf("dashboard %q: at least one panel is required", dash.Name)
		}
		for j, panel := range dash.Panels {
			if panel.Title == "" {
				return fmt.Errorf("dashboard %q panel[%d]: title is required", dash.Name, j)
			}
			if panel.Query == "" {
				return fmt.Errorf("dashboard %q panel %q: query is required", dash.Name, panel.Title)
			}
		}
	}

	validSeverities := map[string]bool{"critical": true, "warning": true, "info": true}
	for i, alert := range config.AlertRules {
		if alert.Name == "" {
			return fmt.Errorf("alert_rule[%d]: name is required", i)
		}
		if alert.Expr == "" {
			return fmt.Errorf("alert_rule %q: expr is required", alert.Name)
		}
		if alert.Severity != "" && !validSeverities[alert.Severity] {
			return fmt.Errorf("alert_rule %q: invalid severity %q", alert.Name, alert.Severity)
		}
	}

	for i, slo := range config.SLOs {
		if slo.Name == "" {
			return fmt.Errorf("slo[%d]: name is required", i)
		}
		if slo.TargetPercent <= 0 || slo.TargetPercent > 100 {
			return fmt.Errorf("slo %q: target_percent must be between 0 and 100", slo.Name)
		}
		if slo.Window == "" {
			return fmt.Errorf("slo %q: window is required", slo.Name)
		}
		if slo.Query == "" {
			return fmt.Errorf("slo %q: query is required", slo.Name)
		}
	}

	validIntegrations := map[string]bool{IntegrationGrafana: true, IntegrationPagerDuty: true, IntegrationSlack: true}
	for i, integ := range config.Integrations {
		if integ.Name == "" {
			return fmt.Errorf("integration[%d]: name is required", i)
		}
		if !validIntegrations[integ.Type] {
			return fmt.Errorf("integration %q: invalid type %q", integ.Name, integ.Type)
		}
	}

	return nil
}

func (s *Service) detectDrift(ctx context.Context, tenantID string, config *ObsConfig) ([]DriftItem, error) {
	var drift []DriftItem

	for _, dash := range config.Dashboards {
		drift = append(drift, DriftItem{
			Resource: "dashboard",
			Name:     dash.Name,
			Field:    "state",
			Expected: "provisioned",
			Actual:   "unknown",
		})
	}

	for _, slo := range config.SLOs {
		drift = append(drift, DriftItem{
			Resource: "slo",
			Name:     slo.Name,
			Field:    "state",
			Expected: "active",
			Actual:   "unknown",
		})
	}

	return drift, nil
}

func (s *Service) reconcileDashboard(ctx context.Context, tenantID string, dash *DashboardConfig) error {
	if dash.Name == "" {
		return fmt.Errorf("dashboard name is required")
	}
	return nil
}

func (s *Service) reconcileAlertRule(ctx context.Context, tenantID string, alert *PrometheusAlert) error {
	if alert.Name == "" || alert.Expr == "" {
		return fmt.Errorf("alert rule name and expression are required")
	}
	return nil
}

func (s *Service) reconcileSLO(ctx context.Context, tenantID string, slo *SLODefinition) error {
	if slo.Name == "" || slo.Query == "" {
		return fmt.Errorf("SLO name and query are required")
	}
	return nil
}

func (s *Service) reconcileIntegration(ctx context.Context, tenantID string, integ *IntegrationConfig) error {
	if integ.Name == "" || integ.Type == "" {
		return fmt.Errorf("integration name and type are required")
	}
	return nil
}

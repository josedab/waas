package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Repository defines the storage interface for pipelines
type Repository interface {
	Create(ctx context.Context, pipeline *Pipeline) error
	GetByID(ctx context.Context, tenantID, pipelineID string) (*Pipeline, error)
	List(ctx context.Context, tenantID string, limit, offset int) ([]Pipeline, int, error)
	Update(ctx context.Context, pipeline *Pipeline) error
	Delete(ctx context.Context, tenantID, pipelineID string) error

	SaveExecution(ctx context.Context, execution *PipelineExecution) error
	GetExecution(ctx context.Context, tenantID, executionID string) (*PipelineExecution, error)
	ListExecutions(ctx context.Context, tenantID, pipelineID string, limit, offset int) ([]PipelineExecution, int, error)
}

// StageExecutor is the interface each stage type must implement
type StageExecutor interface {
	Execute(ctx context.Context, input json.RawMessage, config json.RawMessage) (json.RawMessage, error)
}

// DeliveryFunc is the callback used by the deliver stage to actually send webhooks
type DeliveryFunc func(ctx context.Context, endpointID string, payload json.RawMessage, headers map[string]string) error

// Service provides pipeline management and execution functionality
type Service struct {
	repo        Repository
	executors   map[StageType]StageExecutor
	deliverFunc DeliveryFunc
	logger      *utils.Logger
}

// NewService creates a new pipeline service
func NewService(repo Repository) *Service {
	s := &Service{
		repo:      repo,
		executors: make(map[StageType]StageExecutor),
		logger:    utils.NewLogger("pipeline"),
	}

	// Register built-in stage executors
	s.executors[StageTransform] = &transformExecutor{}
	s.executors[StageValidate] = &validateExecutor{}
	s.executors[StageFilter] = &filterExecutor{}
	s.executors[StageEnrich] = &enrichExecutor{}
	s.executors[StageRoute] = &routeExecutor{}
	s.executors[StageFanOut] = &fanOutExecutor{service: s}
	s.executors[StageDeliver] = &deliverExecutor{service: s}
	s.executors[StageDelay] = &delayExecutor{}
	s.executors[StageLog] = &logExecutor{}

	return s
}

// SetDeliveryFunc sets the callback for actual webhook delivery
func (s *Service) SetDeliveryFunc(fn DeliveryFunc) {
	s.deliverFunc = fn
}

// CreatePipeline creates a new pipeline
func (s *Service) CreatePipeline(ctx context.Context, tenantID string, req *CreatePipelineRequest) (*Pipeline, error) {
	now := time.Now()
	pipeline := &Pipeline{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Stages:      req.Stages,
		Enabled:     true,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := pipeline.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pipeline: %w", err)
	}

	if err := s.repo.Create(ctx, pipeline); err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return pipeline, nil
}

// GetPipeline retrieves a pipeline by ID
func (s *Service) GetPipeline(ctx context.Context, tenantID, pipelineID string) (*Pipeline, error) {
	return s.repo.GetByID(ctx, tenantID, pipelineID)
}

// ListPipelines lists all pipelines for a tenant
func (s *Service) ListPipelines(ctx context.Context, tenantID string, limit, offset int) ([]Pipeline, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.List(ctx, tenantID, limit, offset)
}

// UpdatePipeline updates an existing pipeline
func (s *Service) UpdatePipeline(ctx context.Context, tenantID, pipelineID string, req *UpdatePipelineRequest) (*Pipeline, error) {
	pipeline, err := s.repo.GetByID(ctx, tenantID, pipelineID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		pipeline.Name = *req.Name
	}
	if req.Description != nil {
		pipeline.Description = *req.Description
	}
	if req.Stages != nil {
		pipeline.Stages = *req.Stages
	}
	if req.Enabled != nil {
		pipeline.Enabled = *req.Enabled
	}

	pipeline.Version++
	pipeline.UpdatedAt = time.Now()

	if err := pipeline.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pipeline: %w", err)
	}

	if err := s.repo.Update(ctx, pipeline); err != nil {
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}

	return pipeline, nil
}

// DeletePipeline deletes a pipeline
func (s *Service) DeletePipeline(ctx context.Context, tenantID, pipelineID string) error {
	return s.repo.Delete(ctx, tenantID, pipelineID)
}

// ExecutePipeline runs a pipeline for a given delivery payload
func (s *Service) ExecutePipeline(ctx context.Context, tenantID, pipelineID, deliveryID string, payload json.RawMessage) (*PipelineExecution, error) {
	pipeline, err := s.repo.GetByID(ctx, tenantID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	if !pipeline.Enabled {
		return nil, fmt.Errorf("pipeline is disabled")
	}

	startTime := time.Now()
	execution := &PipelineExecution{
		ID:         uuid.New().String(),
		PipelineID: pipelineID,
		TenantID:   tenantID,
		DeliveryID: deliveryID,
		Status:     StatusRunning,
		StartedAt:  startTime,
		Stages:     make([]StageExecution, 0, len(pipeline.Stages)),
	}

	currentPayload := payload

	for _, stageDef := range pipeline.Stages {
		stageStart := time.Now()
		stageExec := StageExecution{
			StageID:   stageDef.ID,
			StageName: stageDef.Name,
			StageType: stageDef.Type,
			Status:    StatusRunning,
			Input:     currentPayload,
			StartedAt: stageStart,
		}

		// Check conditional execution
		if stageDef.Condition != "" {
			shouldRun, err := evaluateCondition(stageDef.Condition, currentPayload)
			if err != nil || !shouldRun {
				now := time.Now()
				stageExec.Status = StatusSkipped
				stageExec.CompletedAt = &now
				stageExec.DurationMs = time.Since(stageStart).Milliseconds()
				if err != nil {
					stageExec.Error = fmt.Sprintf("condition evaluation failed: %v", err)
				}
				execution.Stages = append(execution.Stages, stageExec)
				continue
			}
		}

		executor, ok := s.executors[stageDef.Type]
		if !ok {
			stageExec.Status = StatusFailed
			stageExec.Error = fmt.Sprintf("unknown stage type: %s", stageDef.Type)
			now := time.Now()
			stageExec.CompletedAt = &now
			stageExec.DurationMs = time.Since(stageStart).Milliseconds()
			execution.Stages = append(execution.Stages, stageExec)

			if !stageDef.ContinueOnError {
				execution.Status = StatusFailed
				execution.Error = stageExec.Error
				break
			}
			continue
		}

		// Execute with optional timeout
		var stageCtx context.Context
		var cancel context.CancelFunc
		if stageDef.Timeout > 0 {
			stageCtx, cancel = context.WithTimeout(ctx, time.Duration(stageDef.Timeout)*time.Second)
		} else {
			stageCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
		}

		output, err := executor.Execute(stageCtx, currentPayload, stageDef.Config)
		cancel()

		now := time.Now()
		stageExec.CompletedAt = &now
		stageExec.DurationMs = time.Since(stageStart).Milliseconds()

		if err != nil {
			stageExec.Status = StatusFailed
			stageExec.Error = err.Error()
			execution.Stages = append(execution.Stages, stageExec)

			if !stageDef.ContinueOnError {
				execution.Status = StatusFailed
				execution.Error = fmt.Sprintf("stage '%s' failed: %v", stageDef.Name, err)
				break
			}
			continue
		}

		stageExec.Status = StatusCompleted
		stageExec.Output = output
		execution.Stages = append(execution.Stages, stageExec)

		// Use output as input for next stage
		if output != nil {
			currentPayload = output
		}
	}

	now := time.Now()
	execution.CompletedAt = &now
	execution.DurationMs = time.Since(startTime).Milliseconds()
	if execution.Status == StatusRunning {
		execution.Status = StatusCompleted
	}

	if err := s.repo.SaveExecution(ctx, execution); err != nil {
		// Log but don't fail the pipeline
		s.logger.Error("failed to save pipeline execution", map[string]interface{}{"error": err.Error()})
	}

	return execution, nil
}

// GetExecution retrieves a pipeline execution
func (s *Service) GetExecution(ctx context.Context, tenantID, executionID string) (*PipelineExecution, error) {
	return s.repo.GetExecution(ctx, tenantID, executionID)
}

// ListExecutions lists pipeline executions
func (s *Service) ListExecutions(ctx context.Context, tenantID, pipelineID string, limit, offset int) ([]PipelineExecution, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListExecutions(ctx, tenantID, pipelineID, limit, offset)
}

// GetTemplates returns pre-built pipeline templates
func (s *Service) GetTemplates() []Pipeline {
	return []Pipeline{
		{
			ID:          "tpl-transform-deliver",
			Name:        "Transform & Deliver",
			Description: "Apply a JavaScript transformation then deliver",
			Stages: []StageDefinition{
				{ID: "transform", Name: "Transform Payload", Type: StageTransform, Config: json.RawMessage(`{"script":"return payload;"}`)},
				{ID: "deliver", Name: "Deliver Webhook", Type: StageDeliver, Config: json.RawMessage(`{}`)},
			},
		},
		{
			ID:          "tpl-validate-transform-deliver",
			Name:        "Validate → Transform → Deliver",
			Description: "Validate schema, transform payload, then deliver",
			Stages: []StageDefinition{
				{ID: "validate", Name: "Validate Schema", Type: StageValidate, Config: json.RawMessage(`{"strictness":"standard","reject_on":"error"}`)},
				{ID: "transform", Name: "Transform Payload", Type: StageTransform, Config: json.RawMessage(`{"script":"return payload;"}`)},
				{ID: "deliver", Name: "Deliver Webhook", Type: StageDeliver, Config: json.RawMessage(`{}`)},
			},
		},
		{
			ID:          "tpl-route-fanout",
			Name:        "Route & Fan-Out",
			Description: "Route events by type and fan-out to multiple endpoints",
			Stages: []StageDefinition{
				{ID: "filter", Name: "Filter Events", Type: StageFilter, Config: json.RawMessage(`{"condition":"return true;","on_reject":"drop"}`)},
				{ID: "route", Name: "Route by Type", Type: StageRoute, Config: json.RawMessage(`{"rules":[]}`)},
				{ID: "fanout", Name: "Fan-Out to Endpoints", Type: StageFanOut, Config: json.RawMessage(`{"parallel":true,"max_parallel":5}`)},
			},
		},
		{
			ID:          "tpl-full-pipeline",
			Name:        "Full Pipeline",
			Description: "Complete transform → validate → route → fan-out → deliver workflow",
			Stages: []StageDefinition{
				{ID: "enrich", Name: "Enrich Payload", Type: StageEnrich, Config: json.RawMessage(`{"script":"payload.processed_at = new Date().toISOString(); return payload;"}`)},
				{ID: "validate", Name: "Validate Schema", Type: StageValidate, Config: json.RawMessage(`{"strictness":"standard","reject_on":"error"}`)},
				{ID: "transform", Name: "Transform", Type: StageTransform, Config: json.RawMessage(`{"script":"return payload;"}`)},
				{ID: "route", Name: "Route Events", Type: StageRoute, Config: json.RawMessage(`{"rules":[]}`)},
				{ID: "deliver", Name: "Deliver", Type: StageDeliver, Config: json.RawMessage(`{}`)},
			},
		},
	}
}

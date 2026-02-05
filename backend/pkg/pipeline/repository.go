package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryRepository is an in-memory implementation of the pipeline repository
type MemoryRepository struct {
	mu         sync.RWMutex
	pipelines  map[string]*Pipeline
	executions map[string]*PipelineExecution
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		pipelines:  make(map[string]*Pipeline),
		executions: make(map[string]*PipelineExecution),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, pipeline *Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stagesJSON, _ := json.Marshal(pipeline.Stages)
	var stages []StageDefinition
	json.Unmarshal(stagesJSON, &stages)

	p := *pipeline
	p.Stages = stages
	r.pipelines[pipeline.ID] = &p
	return nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, tenantID, pipelineID string) (*Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.pipelines[pipelineID]
	if !ok || p.TenantID != tenantID {
		return nil, fmt.Errorf("pipeline not found")
	}
	copy := *p
	return &copy, nil
}

func (r *MemoryRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]Pipeline, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []Pipeline
	for _, p := range r.pipelines {
		if p.TenantID == tenantID {
			results = append(results, *p)
		}
	}

	total := len(results)
	if offset >= total {
		return []Pipeline{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return results[offset:end], total, nil
}

func (r *MemoryRepository) Update(ctx context.Context, pipeline *Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p := *pipeline
	r.pipelines[pipeline.ID] = &p
	return nil
}

func (r *MemoryRepository) Delete(ctx context.Context, tenantID, pipelineID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.pipelines[pipelineID]; ok && p.TenantID == tenantID {
		delete(r.pipelines, pipelineID)
		return nil
	}
	return fmt.Errorf("pipeline not found")
}

func (r *MemoryRepository) SaveExecution(ctx context.Context, execution *PipelineExecution) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := *execution
	r.executions[execution.ID] = &e
	return nil
}

func (r *MemoryRepository) GetExecution(ctx context.Context, tenantID, executionID string) (*PipelineExecution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.executions[executionID]
	if !ok || e.TenantID != tenantID {
		return nil, fmt.Errorf("execution not found")
	}
	copy := *e
	return &copy, nil
}

func (r *MemoryRepository) ListExecutions(ctx context.Context, tenantID, pipelineID string, limit, offset int) ([]PipelineExecution, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []PipelineExecution
	for _, e := range r.executions {
		if e.TenantID == tenantID && e.PipelineID == pipelineID {
			results = append(results, *e)
		}
	}

	total := len(results)
	if offset >= total {
		return []PipelineExecution{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return results[offset:end], total, nil
}

// --- Pipeline execution test ---

func init() {
	_ = time.Now // ensure time is used
}

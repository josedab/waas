package faas

import "context"

// Repository defines persistence operations for FaaS functions.
type Repository interface {
	Create(ctx context.Context, fn *Function) error
	Get(ctx context.Context, tenantID, id string) (*Function, error)
	List(ctx context.Context, tenantID string, limit, offset int) ([]Function, error)
	Update(ctx context.Context, fn *Function) error
	Delete(ctx context.Context, tenantID, id string) error

	RecordExecution(ctx context.Context, exec *FunctionExecution) error
	ListExecutions(ctx context.Context, functionID string, limit int) ([]FunctionExecution, error)
}

type memoryRepository struct {
	functions  map[string]*Function
	executions map[string][]FunctionExecution
}

// NewMemoryRepository creates an in-memory FaaS repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{
		functions:  make(map[string]*Function),
		executions: make(map[string][]FunctionExecution),
	}
}

func (r *memoryRepository) Create(_ context.Context, fn *Function) error {
	r.functions[fn.ID] = fn
	return nil
}

func (r *memoryRepository) Get(_ context.Context, tenantID, id string) (*Function, error) {
	fn, ok := r.functions[id]
	if !ok || fn.TenantID != tenantID {
		return nil, ErrFunctionNotFound
	}
	return fn, nil
}

func (r *memoryRepository) List(_ context.Context, tenantID string, limit, offset int) ([]Function, error) {
	var results []Function
	for _, fn := range r.functions {
		if fn.TenantID == tenantID {
			results = append(results, *fn)
		}
	}
	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], nil
}

func (r *memoryRepository) Update(_ context.Context, fn *Function) error {
	if _, ok := r.functions[fn.ID]; !ok {
		return ErrFunctionNotFound
	}
	r.functions[fn.ID] = fn
	return nil
}

func (r *memoryRepository) Delete(_ context.Context, tenantID, id string) error {
	fn, ok := r.functions[id]
	if !ok || fn.TenantID != tenantID {
		return ErrFunctionNotFound
	}
	delete(r.functions, id)
	return nil
}

func (r *memoryRepository) RecordExecution(_ context.Context, exec *FunctionExecution) error {
	r.executions[exec.FunctionID] = append(r.executions[exec.FunctionID], *exec)
	return nil
}

func (r *memoryRepository) ListExecutions(_ context.Context, functionID string, limit int) ([]FunctionExecution, error) {
	execs := r.executions[functionID]
	if limit > 0 && limit < len(execs) {
		execs = execs[len(execs)-limit:]
	}
	return execs, nil
}

package faas

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingRepo wraps memoryRepository but fails on RecordExecution and/or Update.
type failingRepo struct {
	Repository
	recordExecErr error
	updateErr     error
}

func (f *failingRepo) RecordExecution(ctx context.Context, exec *FunctionExecution) error {
	if f.recordExecErr != nil {
		return f.recordExecErr
	}
	return f.Repository.RecordExecution(ctx, exec)
}

func (f *failingRepo) Update(ctx context.Context, fn *Function) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	return f.Repository.Update(ctx, fn)
}

func createTestFunctionInRepo(t *testing.T, svc *Service) *Function {
	t.Helper()
	fn, err := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "test-fn",
		Runtime: RuntimeJavaScript,
		Code:    "function transform(payload) { return payload; }",
	})
	require.NoError(t, err)
	return fn
}

func TestInvokeFunction_RecordExecutionFails_ExecutionStillCompletes(t *testing.T) {
	baseRepo := NewMemoryRepository()
	repo := &failingRepo{
		Repository:    baseRepo,
		recordExecErr: errors.New("disk full"),
	}
	svc := NewService(repo, nil)

	fn := createTestFunctionInRepo(t, svc)

	resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
		Payload: `{"value": 42}`,
	})

	// Execution should still complete even if recording fails
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Output)
}

func TestInvokeFunction_UpdateFails_ExecutionStillCompletes(t *testing.T) {
	baseRepo := NewMemoryRepository()
	repo := &failingRepo{
		Repository: baseRepo,
		updateErr:  errors.New("connection reset"),
	}
	svc := NewService(repo, nil)

	fn := createTestFunctionInRepo(t, svc)

	resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
		Payload: `{"value": 42}`,
	})

	// Execution should still return a result even if stats update fails
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInvokeFunction_BothRecordAndUpdateFail(t *testing.T) {
	baseRepo := NewMemoryRepository()
	repo := &failingRepo{
		Repository:    baseRepo,
		recordExecErr: errors.New("record failed"),
		updateErr:     errors.New("update failed"),
	}
	svc := NewService(repo, nil)

	fn := createTestFunctionInRepo(t, svc)

	resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
		Payload: `{"key": "value"}`,
	})

	// Execution itself should still succeed
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInvokeFunction_FailedExecution_StillRecordsAttempt(t *testing.T) {
	svc := NewService(nil, nil)

	fn, err := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "failing-fn",
		Runtime: RuntimeJavaScript,
		Code:    "function transform(p) { throw new Error('boom'); }",
	})
	require.NoError(t, err)

	resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
		Payload: `{}`,
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "boom")

	// Verify invocation count was incremented
	updated, _ := svc.GetFunction(context.Background(), "tenant-1", fn.ID)
	assert.Equal(t, int64(1), updated.Invocations)
}

func TestInvokeFunction_ConcurrentExecutionsWithRepoFailure(t *testing.T) {
	baseRepo := NewMemoryRepository()
	repo := &failingRepo{
		Repository:    baseRepo,
		recordExecErr: errors.New("concurrent write conflict"),
	}
	svc := NewService(repo, nil)

	fn := createTestFunctionInRepo(t, svc)

	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
				Payload: `{"concurrent": true}`,
			})
			assert.NoError(t, err)
			assert.True(t, resp.Success)
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

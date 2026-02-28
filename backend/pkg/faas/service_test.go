package faas

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCreateFunction(t *testing.T) {
	svc := NewService(nil, nil)
	fn, err := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "Transform Payload",
		Runtime: RuntimeJavaScript,
		Code:    "function transform(payload) { return payload; }",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn.Status != FunctionReady {
		t.Errorf("expected ready, got %s", fn.Status)
	}
	if fn.Version != 1 {
		t.Errorf("expected version 1, got %d", fn.Version)
	}
}

func TestCreateFunction_EmptyCode(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "Empty",
		Runtime: RuntimeJavaScript,
	})
	if err == nil {
		t.Error("expected error for empty code")
	}
}

func TestCreateFunction_InvalidRuntime(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "Bad",
		Runtime: "python",
		Code:    "print('hello')",
	})
	if err == nil {
		t.Error("expected error for invalid runtime")
	}
}

func TestInvokeFunction_JavaScript(t *testing.T) {
	svc := NewService(nil, nil)
	fn, err := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "Double Value",
		Runtime: RuntimeJavaScript,
		Code:    "function transform(payload) { payload.value = payload.value * 2; return payload; }",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
		Payload: `{"value": 21}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, got error: %s", resp.Error)
	}
	if resp.Output == "" {
		t.Error("expected output")
	}
}

func TestInvokeFunction_Error(t *testing.T) {
	svc := NewService(nil, nil)
	fn, _ := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "Bad Function",
		Runtime: RuntimeJavaScript,
		Code:    "function transform(p) { throw new Error('test error'); }",
	})

	resp, err := svc.InvokeFunction(context.Background(), "tenant-1", fn.ID, &InvokeFunctionRequest{
		Payload: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Success {
		t.Error("expected failure")
	}
}

func TestListTemplates(t *testing.T) {
	svc := NewService(nil, nil)
	templates := svc.ListTemplates()
	if len(templates) == 0 {
		t.Error("expected templates")
	}
}

func TestUpdateFunction(t *testing.T) {
	svc := NewService(nil, nil)
	fn, _ := svc.CreateFunction(context.Background(), "tenant-1", &CreateFunctionRequest{
		Name:    "v1",
		Runtime: RuntimeJavaScript,
		Code:    "function transform(p) { return p; }",
	})

	updated, err := svc.UpdateFunction(context.Background(), "tenant-1", fn.ID, &CreateFunctionRequest{
		Code: "function transform(p) { p.version = 2; return p; }",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Errorf("expected version 2, got %d", updated.Version)
	}
}

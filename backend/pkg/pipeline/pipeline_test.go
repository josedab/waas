package pipeline

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestPipelineCreation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	pipeline, err := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name:        "Test Pipeline",
		Description: "Transform then deliver",
		Stages: []StageDefinition{
			{ID: "transform", Name: "Transform", Type: StageTransform, Config: json.RawMessage(`{"script":"return payload;"}`)},
			{ID: "deliver", Name: "Deliver", Type: StageDeliver, Config: json.RawMessage(`{}`)},
		},
	})
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	if pipeline.ID == "" {
		t.Fatal("pipeline ID should not be empty")
	}
	if len(pipeline.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(pipeline.Stages))
	}
}

func TestPipelineValidation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	// Empty stages should fail
	_, err := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name:   "Bad Pipeline",
		Stages: []StageDefinition{},
	})
	if err == nil {
		t.Fatal("expected validation error for empty stages")
	}

	// Duplicate stage IDs should fail
	_, err = svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "Dup IDs",
		Stages: []StageDefinition{
			{ID: "s1", Name: "A", Type: StageTransform, Config: json.RawMessage(`{}`)},
			{ID: "s1", Name: "B", Type: StageDeliver, Config: json.RawMessage(`{}`)},
		},
	})
	if err == nil {
		t.Fatal("expected validation error for duplicate IDs")
	}
}

func TestPipelineExecution(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "Exec Test",
		Stages: []StageDefinition{
			{ID: "log", Name: "Log", Type: StageLog, Config: json.RawMessage(`{"message":"test","level":"info"}`)},
			{ID: "transform", Name: "Transform", Type: StageTransform, Config: json.RawMessage(`{"script":"return payload;"}`)},
		},
	})

	payload := json.RawMessage(`{"event":"order.created","amount":99.99}`)
	execution, err := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "delivery-1", payload)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if execution.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", execution.Status)
	}
	if len(execution.Stages) != 2 {
		t.Fatalf("expected 2 stage executions, got %d", len(execution.Stages))
	}
	if execution.DurationMs < 0 {
		t.Fatal("duration should be non-negative")
	}
}

func TestPipelineFilterStage(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "Filter Test",
		Stages: []StageDefinition{
			{ID: "filter", Name: "Filter", Type: StageFilter, Config: json.RawMessage(`{"condition":"return false;","on_reject":"drop"}`), ContinueOnError: false},
			{ID: "deliver", Name: "Deliver", Type: StageDeliver, Config: json.RawMessage(`{}`)},
		},
	})

	payload := json.RawMessage(`{"event":"test"}`)
	execution, err := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "delivery-2", payload)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if execution.Status != StatusFailed {
		t.Fatalf("expected failed (filtered), got %s", execution.Status)
	}
}

func TestPipelineValidateStage(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	schema := `{"properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name"]}`

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "Validate Test",
		Stages: []StageDefinition{
			{ID: "validate", Name: "Validate", Type: StageValidate, Config: json.RawMessage(`{"schema":` + schema + `,"strictness":"standard","reject_on":"error"}`)},
		},
	})

	// Valid payload
	execution, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", json.RawMessage(`{"name":"John","age":30}`))
	if execution.Status != StatusCompleted {
		t.Fatalf("valid payload should pass, got %s: %s", execution.Status, execution.Error)
	}

	// Invalid payload (missing required field)
	execution, _ = svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d2", json.RawMessage(`{"age":30}`))
	if execution.Status != StatusFailed {
		t.Fatalf("invalid payload should fail, got %s", execution.Status)
	}
}

func TestPipelineConditionalSkip(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "Conditional Test",
		Stages: []StageDefinition{
			{ID: "transform", Name: "Transform", Type: StageTransform, Config: json.RawMessage(`{"script":"return payload;"}`), Condition: "return false;"},
			{ID: "deliver", Name: "Deliver", Type: StageDeliver, Config: json.RawMessage(`{}`)},
		},
	})

	execution, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", json.RawMessage(`{}`))
	if execution.Stages[0].Status != StatusSkipped {
		t.Fatalf("expected skipped, got %s", execution.Stages[0].Status)
	}
}

func TestPipelineTemplates(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	templates := svc.GetTemplates()
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}

	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Fatal("template name should not be empty")
		}
		if len(tmpl.Stages) == 0 {
			t.Fatalf("template %s should have stages", tmpl.Name)
		}
	}
}

func TestPipelineDelayStage(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "Delay Test",
		Stages: []StageDefinition{
			{ID: "delay", Name: "Delay", Type: StageDelay, Config: json.RawMessage(`{"duration_ms":50}`)},
		},
	})

	start := time.Now()
	execution, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", json.RawMessage(`{}`))
	elapsed := time.Since(start)

	if execution.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", execution.Status)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("delay stage should have waited at least 40ms, took %v", elapsed)
	}
}

func TestPipelineRealJSTransform(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	// Script that adds a field and returns the modified payload
	script := `payload.processed = true; payload.doubled = payload.amount * 2; return payload;`

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "JS Transform Test",
		Stages: []StageDefinition{
			{ID: "transform", Name: "Transform", Type: StageTransform, Config: json.RawMessage(`{"script":"` + script + `"}`)},
		},
	})

	payload := json.RawMessage(`{"event":"order","amount":50}`)
	execution, err := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", payload)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if execution.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s: %s", execution.Status, execution.Error)
	}

	// Check output was actually transformed
	var output map[string]interface{}
	if err := json.Unmarshal(execution.Stages[0].Output, &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if output["processed"] != true {
		t.Fatal("expected processed=true in output")
	}
	if output["doubled"] != float64(100) {
		t.Fatalf("expected doubled=100, got %v", output["doubled"])
	}
}

func TestPipelineRealJSFilter(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	// Filter that only allows amounts > 100
	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "JS Filter Test",
		Stages: []StageDefinition{
			{ID: "filter", Name: "Filter", Type: StageFilter, Config: json.RawMessage(`{"condition":"return payload.amount > 100;","on_reject":"drop"}`)},
			{ID: "deliver", Name: "Deliver", Type: StageDeliver, Config: json.RawMessage(`{}`)},
		},
	})

	// Should pass: amount=200
	exec1, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", json.RawMessage(`{"amount":200}`))
	if exec1.Status != StatusCompleted {
		t.Fatalf("expected completed for amount=200, got %s: %s", exec1.Status, exec1.Error)
	}

	// Should fail: amount=50
	exec2, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d2", json.RawMessage(`{"amount":50}`))
	if exec2.Status != StatusFailed {
		t.Fatalf("expected failed for amount=50, got %s", exec2.Status)
	}
}

func TestPipelineRealJSCondition(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	// Conditional stage: only run transform if type is "premium"
	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "JS Condition Test",
		Stages: []StageDefinition{
			{ID: "transform", Name: "Premium Transform", Type: StageTransform,
				Config:    json.RawMessage(`{"script":"payload.premium = true; return payload;"}`),
				Condition: `return payload.type === "premium";`},
			{ID: "deliver", Name: "Deliver", Type: StageDeliver, Config: json.RawMessage(`{}`)},
		},
	})

	// Premium type: should run transform
	exec1, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", json.RawMessage(`{"type":"premium"}`))
	if exec1.Stages[0].Status != StatusCompleted {
		t.Fatalf("expected completed for premium, got %s", exec1.Stages[0].Status)
	}
	var out1 map[string]interface{}
	json.Unmarshal(exec1.Stages[0].Output, &out1)
	if out1["premium"] != true {
		t.Fatal("expected premium=true in output")
	}

	// Standard type: should skip transform
	exec2, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d2", json.RawMessage(`{"type":"standard"}`))
	if exec2.Stages[0].Status != StatusSkipped {
		t.Fatalf("expected skipped for standard, got %s", exec2.Stages[0].Status)
	}
}

func TestPipelineRealJSRoute(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	pipeline, _ := svc.CreatePipeline(context.Background(), "tenant-1", &CreatePipelineRequest{
		Name: "JS Route Test",
		Stages: []StageDefinition{
			{ID: "route", Name: "Route", Type: StageRoute, Config: json.RawMessage(`{
				"rules": [
					{"condition": "return payload.region === \"us\";", "endpoint_ids": ["ep-us-1", "ep-us-2"], "label": "US"},
					{"condition": "return payload.region === \"eu\";", "endpoint_ids": ["ep-eu-1"], "label": "EU"}
				]
			}`)},
		},
	})

	exec, _ := svc.ExecutePipeline(context.Background(), "tenant-1", pipeline.ID, "d1", json.RawMessage(`{"region":"us"}`))
	if exec.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s: %s", exec.Status, exec.Error)
	}

	var out map[string]interface{}
	json.Unmarshal(exec.Stages[0].Output, &out)
	routed, ok := out["_pipeline_routed_endpoints"].([]interface{})
	if !ok {
		t.Fatal("expected routed endpoints in output")
	}
	if len(routed) != 2 {
		t.Fatalf("expected 2 US endpoints, got %d", len(routed))
	}
}

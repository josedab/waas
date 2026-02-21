package obscodepipeline

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validSpec() json.RawMessage {
	return json.RawMessage(`{
		"signals": [{"type": "metrics", "enabled": true, "sample_rate": 1.0}],
		"exporters": [{"name": "prom", "type": "prometheus", "endpoint": "http://prom:9090", "signals": ["metrics"]}]
	}`)
}

func TestCreatePipeline(t *testing.T) {
	svc := NewService(nil)

	req := &CreatePipelineRequest{
		Name: "test-pipeline",
		Spec: validSpec(),
	}

	pipeline, err := svc.CreatePipeline(context.Background(), "tenant-1", req)
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", pipeline.Name)
	assert.Equal(t, PipelineStatusDraft, pipeline.Status)
	assert.Equal(t, 1, pipeline.Version)
	assert.NotEmpty(t, pipeline.ID)
	assert.NotEmpty(t, pipeline.Checksum)
	assert.Len(t, pipeline.Signals, 1)
	assert.Len(t, pipeline.Exporters, 1)
}

func TestCreatePipelineValidationErrors(t *testing.T) {
	svc := NewService(nil)

	tests := []struct {
		name string
		req  *CreatePipelineRequest
	}{
		{"empty name", &CreatePipelineRequest{Name: "", Spec: validSpec()}},
		{"no signals", &CreatePipelineRequest{Name: "test", Spec: json.RawMessage(`{"signals":[],"exporters":[{"name":"x","type":"otlp","endpoint":"http://x","signals":["metrics"]}]}`)}},
		{"no exporters", &CreatePipelineRequest{Name: "test", Spec: json.RawMessage(`{"signals":[{"type":"metrics","enabled":true}],"exporters":[]}`)}},
		{"invalid signal type", &CreatePipelineRequest{Name: "test", Spec: json.RawMessage(`{"signals":[{"type":"invalid","enabled":true}],"exporters":[{"name":"x","type":"otlp","endpoint":"http://x","signals":["metrics"]}]}`)}},
		{"invalid exporter type", &CreatePipelineRequest{Name: "test", Spec: json.RawMessage(`{"signals":[{"type":"metrics","enabled":true}],"exporters":[{"name":"x","type":"invalid","endpoint":"http://x","signals":["metrics"]}]}`)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreatePipeline(context.Background(), "tenant-1", tt.req)
			assert.Error(t, err)
		})
	}
}

func TestValidateStatusTransition(t *testing.T) {
	svc := NewService(nil)

	assert.NoError(t, svc.validateStatusTransition(PipelineStatusDraft, PipelineStatusActive))
	assert.NoError(t, svc.validateStatusTransition(PipelineStatusActive, PipelineStatusPaused))
	assert.NoError(t, svc.validateStatusTransition(PipelineStatusPaused, PipelineStatusActive))
	assert.Error(t, svc.validateStatusTransition(PipelineStatusArchived, PipelineStatusActive))
	assert.Error(t, svc.validateStatusTransition(PipelineStatusDraft, PipelineStatusFailed))
}

func TestValidateSpec(t *testing.T) {
	svc := NewService(nil)

	errors, err := svc.ValidateSpec(context.Background(), validSpec())
	assert.NoError(t, err)
	assert.Nil(t, errors)

	errors, err = svc.ValidateSpec(context.Background(), json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.NotEmpty(t, errors)
}

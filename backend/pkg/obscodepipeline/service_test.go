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

func validObsConfig() json.RawMessage {
	return json.RawMessage(`{
		"version": "1.0",
		"tenant_id": "tenant-1",
		"dashboards": [{
			"name": "webhook-overview",
			"panels": [{"title": "Delivery Rate", "type": "graph", "query": "rate(deliveries_total[5m])"}]
		}],
		"alert_rules": [{
			"name": "high-error-rate",
			"group": "webhook-alerts",
			"expr": "rate(delivery_errors_total[5m]) > 0.05",
			"for": "5m",
			"severity": "critical"
		}],
		"slos": [{
			"name": "delivery-success",
			"target_percent": 99.9,
			"window": "30d",
			"indicator": "availability",
			"query": "sum(rate(deliveries_success[30d])) / sum(rate(deliveries_total[30d]))"
		}],
		"integrations": [{
			"type": "pagerduty",
			"name": "oncall-team",
			"service_key": "pd-key-xxx",
			"severity": ["critical"]
		}]
	}`)
}

func TestParseObsConfig(t *testing.T) {
	svc := NewService(nil)

	config, err := svc.ParseObsConfig(context.Background(), validObsConfig())
	require.NoError(t, err)
	assert.Equal(t, "1.0", config.Version)
	assert.Len(t, config.Dashboards, 1)
	assert.Len(t, config.AlertRules, 1)
	assert.Len(t, config.SLOs, 1)
	assert.Len(t, config.Integrations, 1)
}

func TestParseObsConfigValidationErrors(t *testing.T) {
	svc := NewService(nil)

	tests := []struct {
		name   string
		config json.RawMessage
	}{
		{"missing version", json.RawMessage(`{"dashboards":[]}`)},
		{"empty dashboard name", json.RawMessage(`{"version":"1.0","dashboards":[{"name":"","panels":[{"title":"x","query":"y"}]}]}`)},
		{"dashboard no panels", json.RawMessage(`{"version":"1.0","dashboards":[{"name":"d","panels":[]}]}`)},
		{"invalid alert severity", json.RawMessage(`{"version":"1.0","alert_rules":[{"name":"a","expr":"x","severity":"invalid"}]}`)},
		{"slo invalid target", json.RawMessage(`{"version":"1.0","slos":[{"name":"s","target_percent":0,"window":"30d","query":"q"}]}`)},
		{"invalid integration type", json.RawMessage(`{"version":"1.0","integrations":[{"name":"i","type":"invalid"}]}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ParseObsConfig(context.Background(), tt.config)
			assert.Error(t, err)
		})
	}
}

func TestApplyConfigDryRun(t *testing.T) {
	svc := NewService(nil)

	result, err := svc.ApplyConfig(context.Background(), "tenant-1", &ApplyConfigRequest{
		Config: validObsConfig(),
		DryRun: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.NotEmpty(t, result.ConfigChecksum)
	assert.NotNil(t, result.CompletedAt)
}

func TestApplyConfigFull(t *testing.T) {
	svc := NewService(nil)

	result, err := svc.ApplyConfig(context.Background(), "tenant-1", &ApplyConfigRequest{
		Config: validObsConfig(),
		DryRun: false,
	})
	require.NoError(t, err)
	assert.Equal(t, ReconcileStatusConverged, result.Status)
	assert.Equal(t, 1, result.DashboardsSync)
	assert.Equal(t, 1, result.AlertRulesSync)
	assert.Equal(t, 1, result.SLOsSync)
	assert.Equal(t, 1, result.IntegrationsSync)
}

func TestCheckDrift(t *testing.T) {
	svc := NewService(nil)

	resp, err := svc.CheckDrift(context.Background(), "tenant-1", validObsConfig())
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Checksum)
}

package obscodepipeline

import (
	"encoding/json"
	"time"
)

// Pipeline status constants
const (
	PipelineStatusDraft    = "draft"
	PipelineStatusActive   = "active"
	PipelineStatusPaused   = "paused"
	PipelineStatusFailed   = "failed"
	PipelineStatusArchived = "archived"
)

// Signal type constants
const (
	SignalMetrics = "metrics"
	SignalTraces  = "traces"
	SignalLogs    = "logs"
)

// Exporter type constants
const (
	ExporterPrometheus = "prometheus"
	ExporterDatadog    = "datadog"
	ExporterOTLP       = "otlp"
	ExporterCloudWatch = "cloudwatch"
	ExporterElastic    = "elasticsearch"
	ExporterWebhook    = "webhook"
)

// Alert severity constants
const (
	AlertSeverityCritical = "critical"
	AlertSeverityWarning  = "warning"
	AlertSeverityInfo     = "info"
)

// ObservabilityPipeline represents a tenant-defined observability pipeline configuration.
type ObservabilityPipeline struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description,omitempty" db:"description"`
	Version     int             `json:"version" db:"version"`
	Status      string          `json:"status" db:"status"`
	Spec        json.RawMessage `json:"spec" db:"spec"`
	Checksum    string          `json:"checksum" db:"checksum"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`

	// Parsed fields (not persisted directly)
	Signals   []SignalConfig   `json:"signals,omitempty"`
	Exporters []ExporterConfig `json:"exporters,omitempty"`
	Alerts    []AlertRule      `json:"alerts,omitempty"`
}

// SignalConfig defines what telemetry signals to collect.
type SignalConfig struct {
	Type       string            `json:"type" yaml:"type"`
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	SampleRate float64           `json:"sample_rate,omitempty" yaml:"sample_rate"`
	Filters    []SignalFilter    `json:"filters,omitempty" yaml:"filters"`
	Labels     map[string]string `json:"labels,omitempty" yaml:"labels"`
}

// SignalFilter specifies which events/endpoints to instrument.
type SignalFilter struct {
	Field    string `json:"field" yaml:"field"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value" yaml:"value"`
}

// ExporterConfig defines where telemetry data is sent.
type ExporterConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Type        string            `json:"type" yaml:"type"`
	Endpoint    string            `json:"endpoint" yaml:"endpoint"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers"`
	BatchSize   int               `json:"batch_size,omitempty" yaml:"batch_size"`
	FlushPeriod int               `json:"flush_period_seconds,omitempty" yaml:"flush_period_seconds"`
	TLSEnabled  bool              `json:"tls_enabled,omitempty" yaml:"tls_enabled"`
	Signals     []string          `json:"signals" yaml:"signals"`
}

// AlertRule defines a declarative alerting rule on telemetry data.
type AlertRule struct {
	Name      string            `json:"name" yaml:"name"`
	Signal    string            `json:"signal" yaml:"signal"`
	Metric    string            `json:"metric" yaml:"metric"`
	Condition string            `json:"condition" yaml:"condition"`
	Threshold float64           `json:"threshold" yaml:"threshold"`
	Window    string            `json:"window" yaml:"window"`
	Severity  string            `json:"severity" yaml:"severity"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels"`
	NotifyVia []string          `json:"notify_via,omitempty" yaml:"notify_via"`
}

// PipelineExecution records a run of the pipeline processing loop.
type PipelineExecution struct {
	ID             string     `json:"id" db:"id"`
	PipelineID     string     `json:"pipeline_id" db:"pipeline_id"`
	Status         string     `json:"status" db:"status"`
	MetricsEmitted int64      `json:"metrics_emitted" db:"metrics_emitted"`
	TracesEmitted  int64      `json:"traces_emitted" db:"traces_emitted"`
	LogsEmitted    int64      `json:"logs_emitted" db:"logs_emitted"`
	Errors         []string   `json:"errors,omitempty"`
	StartedAt      time.Time  `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// AlertEvent records a triggered alert.
type AlertEvent struct {
	ID         string            `json:"id" db:"id"`
	PipelineID string            `json:"pipeline_id" db:"pipeline_id"`
	TenantID   string            `json:"tenant_id" db:"tenant_id"`
	RuleName   string            `json:"rule_name" db:"rule_name"`
	Severity   string            `json:"severity" db:"severity"`
	Message    string            `json:"message" db:"message"`
	Value      float64           `json:"value" db:"value"`
	Threshold  float64           `json:"threshold" db:"threshold"`
	Labels     map[string]string `json:"labels,omitempty"`
	FiredAt    time.Time         `json:"fired_at" db:"fired_at"`
	ResolvedAt *time.Time        `json:"resolved_at,omitempty" db:"resolved_at"`
}

// PipelineStats aggregates metrics about a pipeline's health.
type PipelineStats struct {
	PipelineID     string     `json:"pipeline_id"`
	TotalSignals   int64      `json:"total_signals"`
	FailedExports  int64      `json:"failed_exports"`
	ActiveAlerts   int        `json:"active_alerts"`
	UptimePercent  float64    `json:"uptime_percent"`
	LastExecutedAt *time.Time `json:"last_executed_at,omitempty"`
}

// CreatePipelineRequest is the request DTO for creating a pipeline.
type CreatePipelineRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description,omitempty"`
	Spec        json.RawMessage `json:"spec" binding:"required"`
}

// UpdatePipelineRequest is the request DTO for updating a pipeline.
type UpdatePipelineRequest struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Spec        json.RawMessage `json:"spec,omitempty"`
	Status      string          `json:"status,omitempty"`
}

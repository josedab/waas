package observability

import (
	"context"
	"fmt"
	"os"
	"time"
)

// OTelConfig holds configuration for OpenTelemetry integration
type OTelConfig struct {
	// Enabled determines if OTel export is active
	Enabled bool `json:"enabled"`

	// Endpoint is the OTel collector OTLP endpoint (e.g., "localhost:4317")
	Endpoint string `json:"endpoint"`

	// ServiceName identifies this service in traces
	ServiceName string `json:"service_name"`

	// ServiceVersion is the version of this service
	ServiceVersion string `json:"service_version"`

	// Environment (e.g., "production", "staging", "development")
	Environment string `json:"environment"`

	// BatchTimeout is the maximum time to wait before exporting spans
	BatchTimeout time.Duration `json:"batch_timeout"`

	// MaxExportBatchSize is the maximum number of spans to export in a batch
	MaxExportBatchSize int `json:"max_export_batch_size"`

	// MaxQueueSize is the maximum number of spans to queue before dropping
	MaxQueueSize int `json:"max_queue_size"`

	// SampleRate is the fraction of traces to sample (0.0 to 1.0)
	SampleRate float64 `json:"sample_rate"`

	// Headers to include in OTLP requests (e.g., for authentication)
	Headers map[string]string `json:"headers"`

	// Insecure allows non-TLS connections to the collector
	Insecure bool `json:"insecure"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *OTelConfig {
	return &OTelConfig{
		Enabled:            false,
		Endpoint:           "localhost:4317",
		ServiceName:        "waas",
		ServiceVersion:     "1.0.0",
		Environment:        "development",
		BatchTimeout:       5 * time.Second,
		MaxExportBatchSize: 512,
		MaxQueueSize:       2048,
		SampleRate:         1.0,
		Headers:            make(map[string]string),
		Insecure:           true,
	}
}

// ConfigFromEnv creates configuration from environment variables
func ConfigFromEnv() *OTelConfig {
	config := DefaultConfig()

	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
		config.Enabled = true
	}

	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	if serviceVersion := os.Getenv("OTEL_SERVICE_VERSION"); serviceVersion != "" {
		config.ServiceVersion = serviceVersion
	}

	if env := os.Getenv("OTEL_ENVIRONMENT"); env != "" {
		config.Environment = env
	}

	if os.Getenv("OTEL_INSECURE") == "false" {
		config.Insecure = false
	}

	return config
}

// Validate checks if the configuration is valid
func (c *OTelConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Endpoint == "" {
		return fmt.Errorf("otel endpoint is required when enabled")
	}

	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if c.SampleRate < 0 || c.SampleRate > 1 {
		return fmt.Errorf("sample rate must be between 0 and 1")
	}

	if c.BatchTimeout <= 0 {
		return fmt.Errorf("batch timeout must be positive")
	}

	if c.MaxExportBatchSize <= 0 {
		return fmt.Errorf("max export batch size must be positive")
	}

	return nil
}

// ConfigureExporter configures the OTel exporter on the service
func (s *Service) ConfigureExporter(config *OTelConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	s.exporter = &OTelExporter{
		endpoint: config.Endpoint,
		enabled:  config.Enabled,
	}

	return nil
}

// ExportSpans exports spans to the configured OTel collector
func (s *Service) ExportSpans(ctx context.Context, spans []*WebhookSpan) error {
	if s.exporter == nil || !s.exporter.enabled {
		return nil
	}

	// In a full implementation, this would use the OTLP exporter
	// to send spans to the collector. For now, we just validate
	// the spans are properly formatted.
	for _, span := range spans {
		if span.TraceID == "" || span.SpanID == "" {
			return fmt.Errorf("span missing required trace/span ID")
		}
	}

	return nil
}

// ResourceAttributes returns standard resource attributes for traces
func (c *OTelConfig) ResourceAttributes() map[string]string {
	return map[string]string{
		"service.name":        c.ServiceName,
		"service.version":     c.ServiceVersion,
		"deployment.environment": c.Environment,
		"telemetry.sdk.name":  "waas-observability",
		"telemetry.sdk.language": "go",
	}
}

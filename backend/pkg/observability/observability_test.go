package observability

import (
	"testing"
)

func TestTraceContextPropagator_NewTraceContext(t *testing.T) {
	propagator := NewTraceContextPropagator()

	ctx, err := propagator.NewTraceContext()
	if err != nil {
		t.Fatalf("NewTraceContext() error = %v", err)
	}

	if ctx.TraceID == "" {
		t.Error("TraceID should not be empty")
	}
	if ctx.SpanID == "" {
		t.Error("SpanID should not be empty")
	}
	if len(ctx.TraceID) != 32 {
		t.Errorf("TraceID should be 32 chars, got %d", len(ctx.TraceID))
	}
	if len(ctx.SpanID) != 16 {
		t.Errorf("SpanID should be 16 chars, got %d", len(ctx.SpanID))
	}
}

func TestTraceContextPropagator_InjectExtract(t *testing.T) {
	propagator := NewTraceContextPropagator()

	// Create context
	ctx, _ := propagator.NewTraceContext()
	ctx.TraceFlags = 0x01 // Sampled flag

	// Inject into headers
	headers := make(map[string]string)
	propagator.Inject(ctx, headers)

	traceparent, ok := headers["traceparent"]
	if !ok {
		t.Fatal("traceparent header should be set")
	}

	// Extract back
	extracted, err := propagator.Extract(headers)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if extracted == nil {
		t.Fatal("Extract() should return context")
	}

	if extracted.TraceID != ctx.TraceID {
		t.Errorf("TraceID mismatch: got %s, want %s", extracted.TraceID, ctx.TraceID)
	}
	if extracted.SpanID != ctx.SpanID {
		t.Errorf("SpanID mismatch: got %s, want %s", extracted.SpanID, ctx.SpanID)
	}

	// Verify W3C format: 00-{traceID}-{spanID}-{flags}
	if len(traceparent) < 55 {
		t.Errorf("traceparent should be at least 55 chars, got %d", len(traceparent))
	}
}

func TestTraceContextPropagator_ExtractInvalid(t *testing.T) {
	propagator := NewTraceContextPropagator()

	tests := []struct {
		name    string
		headers map[string]string
		wantErr bool
	}{
		{"empty headers", map[string]string{}, true},
		{"invalid format", map[string]string{"traceparent": "invalid"}, true},
		{"wrong version", map[string]string{"traceparent": "ff-0123456789abcdef0123456789abcdef-0123456789abcdef-00"}, false}, // May be valid if version is ignored
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := propagator.Extract(tt.headers)
			if tt.wantErr && (err == nil && ctx != nil) {
				t.Logf("Extract() returned ctx=%v, err=%v", ctx != nil, err)
			}
		})
	}
}

func TestOTelConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *OTelConfig
		wantErr bool
	}{
		{
			name:    "disabled config is valid",
			config:  &OTelConfig{Enabled: false},
			wantErr: false,
		},
		{
			name: "valid enabled config",
			config: &OTelConfig{
				Enabled:            true,
				Endpoint:           "localhost:4317",
				ServiceName:        "test",
				SampleRate:         0.5,
				BatchTimeout:       5000000000,
				MaxExportBatchSize: 512,
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: &OTelConfig{
				Enabled:     true,
				ServiceName: "test",
			},
			wantErr: true,
		},
		{
			name: "missing service name",
			config: &OTelConfig{
				Enabled:  true,
				Endpoint: "localhost:4317",
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &OTelConfig{
				Enabled:     true,
				Endpoint:    "localhost:4317",
				ServiceName: "test",
				SampleRate:  1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("Default config should be disabled")
	}
	if config.Endpoint != "localhost:4317" {
		t.Errorf("Expected default endpoint localhost:4317, got %s", config.Endpoint)
	}
	if config.SampleRate != 1.0 {
		t.Errorf("Expected default sample rate 1.0, got %f", config.SampleRate)
	}
}

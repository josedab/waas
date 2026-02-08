package services

import (
	"testing"
)

func TestNewAIComposerService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewAIComposerService(nil, nil, nil, nil)
	if svc == nil {
		t.Fatal("NewAIComposerService returned nil with nil deps")
	}
}

func TestNewBidirectionalSyncService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewBidirectionalSyncService(nil, nil)
	if svc == nil {
		t.Fatal("NewBidirectionalSyncService returned nil with nil deps")
	}
}

func TestNewComplianceService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewComplianceService(nil, nil)
	if svc == nil {
		t.Fatal("NewComplianceService returned nil with nil deps")
	}
}

func TestNewEdgeFunctionsService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewEdgeFunctionsService(nil, nil)
	if svc == nil {
		t.Fatal("NewEdgeFunctionsService returned nil with nil deps")
	}
}

func TestNewFederatedMeshService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewFederatedMeshService(nil, nil)
	if svc == nil {
		t.Fatal("NewFederatedMeshService returned nil with nil deps")
	}
}

func TestNewGraphQLService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewGraphQLService(nil, nil)
	if svc == nil {
		t.Fatal("NewGraphQLService returned nil with nil deps")
	}
}

func TestNewReplayService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewReplayService(nil, nil, nil)
	if svc == nil {
		t.Fatal("NewReplayService returned nil with nil deps")
	}
}

func TestNewSDKGeneratorService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, nil)
	if svc == nil {
		t.Fatal("NewSDKGeneratorService returned nil with nil deps")
	}
}

func TestNewSelfHealingService_NilDeps(t *testing.T) {
	t.Parallel()
	svc := NewSelfHealingService(nil, nil)
	if svc == nil {
		t.Fatal("NewSelfHealingService returned nil with nil deps")
	}
}

func TestJavaScriptValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		valid   bool
	}{
		{
			name:  "valid transformation",
			code:  `return { event: payload.type, data: payload };`,
			valid: true,
		},
		{
			name:  "unbalanced braces",
			code:  `return { event: payload.type`,
			valid: false,
		},
		{
			name:  "dangerous eval pattern",
			code:  `eval("alert(1)")`,
			valid: false,
		},
		{
			name:  "dangerous require pattern",
			code:  `const fs = require("fs")`,
			valid: false,
		},
	}

	v := NewJavaScriptValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			valid, errs := v.Validate(tt.code)
			if valid != tt.valid {
				t.Errorf("Validate(%q) = %v (errors: %v), want valid=%v", tt.code, valid, errs, tt.valid)
			}
		})
	}
}

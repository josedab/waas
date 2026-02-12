package edge

import "testing"

func TestNewService(t *testing.T) {
	// Verify constructor doesn't panic with nil dependencies
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestDefaultServiceConfig(t *testing.T) {
	cfg := DefaultServiceConfig()
	if cfg == nil {
		t.Fatal("DefaultServiceConfig returned nil")
	}
}

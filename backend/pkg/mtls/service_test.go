package mtls

import "testing"

func TestNewService(t *testing.T) {
	// Verify constructor doesn't panic with nil dependencies
	svc := NewService(nil)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}

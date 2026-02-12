package eventsourcing

import "testing"

func TestNewRepository(t *testing.T) {
	// Verify constructor doesn't panic with nil dependencies
	repo := NewRepository(nil)
	if repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

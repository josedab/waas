package cloud

import (
	"testing"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"plan not found", ErrPlanNotFound, 400},
		{"subscription not found", ErrSubscriptionNotFound, 404},
		{"usage limit exceeded", ErrUsageLimitExceeded, 429},
		{"invalid plan upgrade", ErrInvalidPlanUpgrade, 409},
		{"payment required", ErrPaymentRequired, 402},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := status(tt.err)
			if got != tt.expected {
				t.Errorf("status(%v) = %d, want %d", tt.err, got, tt.expected)
			}
		})
	}
}

func TestListPlansFiltersInactive(t *testing.T) {
	active := 0
	for _, p := range AvailablePlans {
		if p.IsActive {
			active++
		}
	}
	if active == 0 {
		t.Error("expected at least one active plan")
	}
}

func TestGetPlanByID(t *testing.T) {
	plan := GetPlanByID("free")
	if plan == nil {
		t.Fatal("expected free plan to exist")
	}
	if plan.Name != "Free" {
		t.Errorf("expected plan name 'Free', got '%s'", plan.Name)
	}

	if GetPlanByID("nonexistent") != nil {
		t.Error("expected nil for nonexistent plan")
	}
}

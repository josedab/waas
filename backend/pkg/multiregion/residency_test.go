package multiregion

import (
	"context"
	"testing"
)

func TestDefaultRegions(t *testing.T) {
	regions := DefaultRegions()
	if len(regions) < 5 {
		t.Errorf("expected at least 5 regions, got %d", len(regions))
	}
	// Verify US East is primary
	usEast := regions["us-east-1"]
	if usEast == nil {
		t.Fatal("us-east-1 region not found")
	}
	if !usEast.IsPrimary {
		t.Error("us-east-1 should be primary")
	}
}

func TestDataResidencyPolicy(t *testing.T) {
	svc := NewDataResidencyService()
	ctx := context.Background()

	// Set EU policy
	policy := &DataResidencyPolicy{
		TenantID:       "tenant-eu",
		PrimaryRegion:  "eu-west-1",
		AllowedRegions: []string{"eu-west-1"},
		BlockedRegions: []string{},
		Regulation:     RegulationGDPR,
		EnforceStrict:  true,
	}

	if err := svc.SetPolicy(ctx, policy); err != nil {
		t.Fatalf("failed to set policy: %v", err)
	}

	// Retrieve it
	got, err := svc.GetPolicy(ctx, "tenant-eu")
	if err != nil {
		t.Fatalf("failed to get policy: %v", err)
	}
	if got.PrimaryRegion != "eu-west-1" {
		t.Errorf("expected eu-west-1, got %s", got.PrimaryRegion)
	}
}

func TestCheckDataFlow(t *testing.T) {
	svc := NewDataResidencyService()
	ctx := context.Background()

	// Set GDPR policy: only EU allowed
	_ = svc.SetPolicy(ctx, &DataResidencyPolicy{
		TenantID:       "tenant-gdpr",
		PrimaryRegion:  "eu-west-1",
		AllowedRegions: []string{"eu-west-1"},
		Regulation:     RegulationGDPR,
		EnforceStrict:  true,
	})

	tests := []struct {
		name     string
		tenant   string
		source   string
		target   string
		allowed  bool
	}{
		{"same region", "tenant-gdpr", "eu-west-1", "eu-west-1", true},
		{"EU to EU allowed", "tenant-gdpr", "eu-west-1", "eu-west-1", true},
		{"EU to US blocked", "tenant-gdpr", "eu-west-1", "us-east-1", false},
		{"no policy = allowed", "tenant-noPolicy", "us-east-1", "eu-west-1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := svc.CheckDataFlow(ctx, tt.tenant, tt.source, tt.target)
			if tt.allowed && !allowed {
				t.Errorf("expected data flow to be allowed, got err=%v", err)
			}
			if !tt.allowed && allowed {
				t.Error("expected data flow to be blocked")
			}
		})
	}
}

func TestBlockedRegions(t *testing.T) {
	svc := NewDataResidencyService()
	ctx := context.Background()

	_ = svc.SetPolicy(ctx, &DataResidencyPolicy{
		TenantID:       "tenant-block",
		PrimaryRegion:  "us-east-1",
		BlockedRegions: []string{"ap-northeast-1"},
		Regulation:     RegulationCustom,
	})

	allowed, _ := svc.CheckDataFlow(ctx, "tenant-block", "us-east-1", "eu-west-1")
	if !allowed {
		t.Error("EU should be allowed (not in blocked list)")
	}

	allowed, _ = svc.CheckDataFlow(ctx, "tenant-block", "us-east-1", "ap-northeast-1")
	if allowed {
		t.Error("Tokyo should be blocked")
	}
}

func TestInvalidRegion(t *testing.T) {
	svc := NewDataResidencyService()
	ctx := context.Background()

	err := svc.SetPolicy(ctx, &DataResidencyPolicy{
		TenantID:      "tenant-invalid",
		PrimaryRegion: "invalid-region",
	})
	if err != ErrInvalidResidencyRegion {
		t.Errorf("expected ErrInvalidResidencyRegion, got %v", err)
	}
}

func TestGetRegionCompliance(t *testing.T) {
	svc := NewDataResidencyService()

	status, err := svc.GetRegionCompliance("eu-west-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(status.Certifications) == 0 {
		t.Error("expected certifications for EU region")
	}

	// Check GDPR is supported in EU
	found := false
	for _, reg := range status.SupportedRegulations {
		if reg == RegulationGDPR {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GDPR support in EU region")
	}

	_, err = svc.GetRegionCompliance("nonexistent")
	if err != ErrRegionNotFound {
		t.Error("expected region not found error")
	}
}

func TestListRegions(t *testing.T) {
	svc := NewDataResidencyService()
	regions := svc.ListRegions()
	if len(regions) < 5 {
		t.Errorf("expected at least 5 regions, got %d", len(regions))
	}
}

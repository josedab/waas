package migrationwizard

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestStartMigration_Svix(t *testing.T) {
	svc := NewService(nil, nil)
	m, err := svc.StartMigration(context.Background(), "tenant-1", &StartMigrationRequest{
		SourcePlatform: PlatformSvix,
		SourceAPIKey:   "svix_test_key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Status != MigrationReady {
		t.Errorf("expected ready, got %s", m.Status)
	}
	if m.Analysis == nil {
		t.Error("expected analysis")
	}
	if m.Analysis.EndpointsFound == 0 {
		t.Error("expected endpoints found > 0")
	}
}

func TestStartMigration_UnsupportedPlatform(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.StartMigration(context.Background(), "tenant-1", &StartMigrationRequest{
		SourcePlatform: "unsupported",
		SourceAPIKey:   "key",
	})
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}

func TestStartMigration_EmptyAPIKey(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.StartMigration(context.Background(), "tenant-1", &StartMigrationRequest{
		SourcePlatform: PlatformSvix,
	})
	if err == nil {
		t.Error("expected error for empty API key")
	}
}

func TestExecuteMigration(t *testing.T) {
	svc := NewService(nil, nil)
	m, _ := svc.StartMigration(context.Background(), "tenant-1", &StartMigrationRequest{
		SourcePlatform: PlatformConvoy,
		SourceAPIKey:   "convoy_key",
	})

	result, err := svc.ExecuteMigration(context.Background(), "tenant-1", m.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != MigrationCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}
	if result.Progress.PercentComplete != 100 {
		t.Errorf("expected 100%%, got %.1f%%", result.Progress.PercentComplete)
	}
}

func TestRollbackMigration(t *testing.T) {
	svc := NewService(nil, nil)
	m, _ := svc.StartMigration(context.Background(), "tenant-1", &StartMigrationRequest{
		SourcePlatform: PlatformHookdeck,
		SourceAPIKey:   "hd_key",
	})
	svc.ExecuteMigration(context.Background(), "tenant-1", m.ID)

	rolled, err := svc.RollbackMigration(context.Background(), "tenant-1", m.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rolled.Status != MigrationRolledBack {
		t.Errorf("expected rolled_back, got %s", rolled.Status)
	}
}

func TestGetCompatibility(t *testing.T) {
	svc := NewService(nil, nil)
	platforms := []SourcePlatform{PlatformSvix, PlatformHookdeck, PlatformConvoy, PlatformEventBridge}
	for _, p := range platforms {
		info := svc.GetCompatibility(p)
		if info["supported"] != true {
			t.Errorf("expected %s to be supported", p)
		}
		score, ok := info["compatibility_score"].(int)
		if !ok || score <= 0 {
			t.Errorf("expected positive compatibility score for %s", p)
		}
	}
}

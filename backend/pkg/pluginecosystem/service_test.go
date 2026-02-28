package pluginecosystem

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

func TestPublishPlugin(t *testing.T) {
	svc := NewService(nil, &ServiceConfig{RequireReview: false, MaxPluginsPerDeveloper: 50, MaxInstallsPerTenant: 100})
	plugin, err := svc.PublishPlugin(context.Background(), "dev-1", &PublishPluginRequest{
		Name:        "Slack Notifier",
		Description: "Send webhook events to Slack channels",
		Type:        PluginNotifier,
		Version:     "1.0.0",
		Manifest:    &PluginManifest{Runtime: "javascript", EntryPoint: "index.js"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plugin.Status != PluginPublished {
		t.Errorf("expected published, got %s", plugin.Status)
	}
	if plugin.Slug != "slack-notifier" {
		t.Errorf("expected slug slack-notifier, got %s", plugin.Slug)
	}
}

func TestInstallPlugin(t *testing.T) {
	svc := NewService(nil, &ServiceConfig{RequireReview: false, MaxPluginsPerDeveloper: 50, MaxInstallsPerTenant: 100})
	plugin, _ := svc.PublishPlugin(context.Background(), "dev-1", &PublishPluginRequest{
		Name: "Test Plugin", Description: "Test", Type: PluginTransform, Version: "1.0.0",
		Manifest: &PluginManifest{Runtime: "javascript", EntryPoint: "index.js"},
	})

	inst, err := svc.InstallPlugin(context.Background(), "tenant-1", &InstallPluginRequest{PluginID: plugin.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inst.Enabled {
		t.Error("expected installation to be enabled")
	}
}

func TestInstallPlugin_AlreadyInstalled(t *testing.T) {
	svc := NewService(nil, &ServiceConfig{RequireReview: false, MaxPluginsPerDeveloper: 50, MaxInstallsPerTenant: 100})
	plugin, _ := svc.PublishPlugin(context.Background(), "dev-1", &PublishPluginRequest{
		Name: "Test", Description: "Test", Type: PluginFilter, Version: "1.0.0",
		Manifest: &PluginManifest{Runtime: "javascript", EntryPoint: "index.js"},
	})
	svc.InstallPlugin(context.Background(), "tenant-1", &InstallPluginRequest{PluginID: plugin.ID})
	_, err := svc.InstallPlugin(context.Background(), "tenant-1", &InstallPluginRequest{PluginID: plugin.ID})
	if err == nil {
		t.Error("expected error for duplicate install")
	}
}

func TestAddReview(t *testing.T) {
	svc := NewService(nil, &ServiceConfig{RequireReview: false, MaxPluginsPerDeveloper: 50, MaxInstallsPerTenant: 100})
	plugin, _ := svc.PublishPlugin(context.Background(), "dev-1", &PublishPluginRequest{
		Name: "Review Test", Description: "Test", Type: PluginAnalytics, Version: "1.0.0",
		Manifest: &PluginManifest{Runtime: "javascript", EntryPoint: "index.js"},
	})

	review, err := svc.AddReview(context.Background(), "tenant-1", plugin.ID, 5, "Great plugin!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Rating != 5 {
		t.Errorf("expected rating 5, got %d", review.Rating)
	}
}

func TestAddReview_InvalidRating(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.AddReview(context.Background(), "t1", "p1", 6, "")
	if err == nil {
		t.Error("expected error for invalid rating")
	}
}

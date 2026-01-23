package mobilesdk

import (
	"context"
	"testing"
)

func TestRegisterApp(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	app, err := svc.RegisterApp(ctx, "tenant-1", &MobileApp{
		Name:     "MyApp iOS",
		BundleID: "com.myapp.ios",
		Platform: PlatformIOS,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.ID == "" {
		t.Error("expected non-empty app ID")
	}
	if app.Environment != "sandbox" {
		t.Errorf("expected sandbox environment, got %s", app.Environment)
	}
	if !app.PushEnabled {
		t.Error("expected push to be enabled by default")
	}
}

func TestRegisterAppInvalidPlatform(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.RegisterApp(ctx, "tenant-1", &MobileApp{
		Name:     "Test",
		BundleID: "com.test",
		Platform: "windows",
	})
	if err != ErrInvalidPlatform {
		t.Errorf("expected ErrInvalidPlatform, got %v", err)
	}
}

func TestRegisterAppDuplicate(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, _ = svc.RegisterApp(ctx, "tenant-1", &MobileApp{
		Name:     "App",
		BundleID: "com.dup",
		Platform: PlatformAndroid,
	})
	_, err := svc.RegisterApp(ctx, "tenant-1", &MobileApp{
		Name:     "App2",
		BundleID: "com.dup",
		Platform: PlatformAndroid,
	})
	if err != ErrAppAlreadyExists {
		t.Errorf("expected ErrAppAlreadyExists, got %v", err)
	}
}

func TestRegisterDevice(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	dev, err := svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{
		Platform:  PlatformIOS,
		PushToken: "apns-token-12345",
		Topics:    []string{"order.created", "payment.completed"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dev.ID == "" {
		t.Error("expected non-empty device ID")
	}
	if !dev.IsActive {
		t.Error("expected device to be active")
	}
	if len(dev.Topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(dev.Topics))
	}
}

func TestRegisterDeviceInvalidPlatform(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{
		Platform:  "invalid",
		PushToken: "token",
	})
	if err != ErrInvalidPlatform {
		t.Errorf("expected ErrInvalidPlatform, got %v", err)
	}
}

func TestRegisterDeviceInvalidToken(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{
		Platform:  PlatformAndroid,
		PushToken: "",
	})
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRegisterDeviceDedup(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	dev1, _ := svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{
		Platform:  PlatformIOS,
		PushToken: "same-token",
	})
	dev2, _ := svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{
		Platform:   PlatformIOS,
		PushToken:  "same-token",
		OSVersion:  "17.0",
		SDKVersion: "2.0",
	})

	if dev1.ID != dev2.ID {
		t.Error("expected same device ID for duplicate token")
	}
	if dev2.OSVersion != "17.0" {
		t.Error("expected device to be updated with new OS version")
	}
}

func TestUnregisterDevice(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	dev, _ := svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{
		Platform:  PlatformAndroid,
		PushToken: "fcm-token",
	})

	err := svc.UnregisterDevice(ctx, dev.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not appear in active devices
	active := svc.ListActiveDevices(ctx, "tenant-1")
	for _, d := range active {
		if d.ID == dev.ID {
			t.Error("expected device to be inactive")
		}
	}
}

func TestUnregisterDeviceNotFound(t *testing.T) {
	svc := NewService()
	err := svc.UnregisterDevice(context.Background(), "nonexistent")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestGetSDKConfig(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	app, _ := svc.RegisterApp(ctx, "tenant-1", &MobileApp{
		Name:          "ConfigApp",
		BundleID:      "com.config.test",
		Platform:      PlatformIOS,
		WebhookTopics: []string{"order.*"},
	})

	config, err := svc.GetSDKConfig(ctx, app.ID, "https://api.waas.io")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.APIEndpoint != "https://api.waas.io" {
		t.Errorf("expected API endpoint to be set, got %s", config.APIEndpoint)
	}
	if config.WebSocketURL != "wss://api.waas.io/ws" {
		t.Errorf("expected wss URL, got %s", config.WebSocketURL)
	}
	if !config.OfflineQueue.Enabled {
		t.Error("expected offline queue to be enabled")
	}
	if config.RetryPolicy.MaxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", config.RetryPolicy.MaxRetries)
	}
}

func TestGetSDKConfigNotFound(t *testing.T) {
	svc := NewService()
	_, err := svc.GetSDKConfig(context.Background(), "nonexistent", "http://localhost")
	if err != ErrSDKConfigNotFound {
		t.Errorf("expected ErrSDKConfigNotFound, got %v", err)
	}
}

func TestListApps(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	svc.RegisterApp(ctx, "tenant-1", &MobileApp{Name: "iOS", BundleID: "com.t1.ios", Platform: PlatformIOS})
	svc.RegisterApp(ctx, "tenant-1", &MobileApp{Name: "Android", BundleID: "com.t1.android", Platform: PlatformAndroid})
	svc.RegisterApp(ctx, "tenant-2", &MobileApp{Name: "Other", BundleID: "com.t2.ios", Platform: PlatformIOS})

	apps := svc.ListApps(ctx, "tenant-1")
	if len(apps) != 2 {
		t.Errorf("expected 2 apps for tenant-1, got %d", len(apps))
	}
}

func TestListActiveDevices(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{Platform: PlatformIOS, PushToken: "tok1"})
	svc.RegisterDevice(ctx, "tenant-1", &DeviceRegistration{Platform: PlatformAndroid, PushToken: "tok2"})
	svc.RegisterDevice(ctx, "tenant-2", &DeviceRegistration{Platform: PlatformIOS, PushToken: "tok3"})

	devices := svc.ListActiveDevices(ctx, "tenant-1")
	if len(devices) != 2 {
		t.Errorf("expected 2 active devices, got %d", len(devices))
	}
}

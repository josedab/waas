package mobileapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.config)
	assert.Equal(t, 10, svc.config.MaxDevicesPerTenant)
}

func TestRegisterDevice(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	device, err := svc.RegisterDevice("t1", &RegisterDeviceRequest{
		DeviceToken: "apns-token-xxx",
		Platform:    PlatformIOS,
		Name:        "iPhone 15",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, device.ID)
	assert.Equal(t, "t1", device.TenantID)
	assert.Equal(t, PlatformIOS, device.Platform)
	assert.Equal(t, "iPhone 15", device.Name)
	assert.True(t, device.Enabled)
}

func TestRegisterDevice_InvalidToken(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	_, err := svc.RegisterDevice("t1", &RegisterDeviceRequest{
		DeviceToken: "",
		Platform:    PlatformIOS,
	})
	assert.ErrorIs(t, err, ErrInvalidDeviceToken)
}

func TestGetDashboard(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	dashboard, err := svc.GetDashboard("t1")
	require.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.Greater(t, dashboard.ActiveEndpoints, 0)
	assert.NotEmpty(t, dashboard.TopEndpoints)
}

func TestListNotifications(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Register a device and send a test notification
	device, err := svc.RegisterDevice("t1", &RegisterDeviceRequest{
		DeviceToken: "token-123",
		Platform:    PlatformAndroid,
		Name:        "Pixel 8",
	})
	require.NoError(t, err)

	_, err = svc.SendTestNotification("t1", device.ID)
	require.NoError(t, err)

	notifications, err := svc.ListNotifications("t1", 10)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, "Test Notification", notifications[0].Title)
}

func TestUpdatePreferences(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	device, err := svc.RegisterDevice("t1", &RegisterDeviceRequest{
		DeviceToken: "token-456",
		Platform:    PlatformIOS,
		Name:        "iPad",
	})
	require.NoError(t, err)

	enabled := false
	updated, err := svc.UpdatePreferences(device.ID, &NotificationPreferencesRequest{
		Enabled: &enabled,
		Filters: &NotificationFilter{
			EndpointIDs: []string{"ep-1"},
			MinSeverity: "warning",
		},
	})
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	assert.Equal(t, []string{"ep-1"}, updated.Filters.EndpointIDs)
	assert.Equal(t, "warning", updated.Filters.MinSeverity)
}

func TestSendTestNotification(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	device, err := svc.RegisterDevice("t1", &RegisterDeviceRequest{
		DeviceToken: "token-789",
		Platform:    PlatformAndroid,
		Name:        "Galaxy S24",
	})
	require.NoError(t, err)

	notification, err := svc.SendTestNotification("t1", device.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, notification.ID)
	assert.Equal(t, device.ID, notification.DeviceID)
	assert.Equal(t, "Test Notification", notification.Title)
	assert.Equal(t, NotificationDeliveryFailure, notification.Type)
	assert.Nil(t, notification.ReadAt)

	// Test with non-existent device
	_, err = svc.SendTestNotification("t1", "non-existent")
	assert.ErrorIs(t, err, ErrDeviceNotFound)
}

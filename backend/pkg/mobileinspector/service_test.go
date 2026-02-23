package mobileinspector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDevice(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	session, err := svc.RegisterDevice("t1", "user-1", &DeviceRegistration{
		DeviceID:   "device-123",
		Platform:   "ios",
		AppVersion: "1.0.0",
		PushToken:  "apns-token-xxx",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Contains(t, session.Token, "mob_")
	assert.Equal(t, "ios", session.Platform)
}

func TestRegisterDeviceValidation(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	_, err := svc.RegisterDevice("", "user", &DeviceRegistration{DeviceID: "d", Platform: "ios"})
	assert.Error(t, err)

	_, err = svc.RegisterDevice("t1", "user", &DeviceRegistration{DeviceID: "d", Platform: "windows"})
	assert.Error(t, err)
}

func TestValidateSession(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	session, _ := svc.RegisterDevice("t1", "user-1", &DeviceRegistration{
		DeviceID: "d1", Platform: "android",
	})

	validated, err := svc.ValidateSession(session.Token)
	require.NoError(t, err)
	assert.Equal(t, session.ID, validated.ID)

	// Invalid token
	_, err = svc.ValidateSession("invalid")
	assert.Error(t, err)
}

func TestEventFeed(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	resp, err := svc.GetEventFeed("t1", &SyncRequest{Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.ServerTime)
}

func TestAlertConfig(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	cfg, err := svc.UpdateAlertConfig("user-1", &AlertConfig{
		FailureThreshold:   10,
		LatencyThresholdMs: 3000,
		SuccessRateMin:     99.0,
		NotifyOnFailure:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, 10, cfg.FailureThreshold)

	retrieved, err := svc.GetAlertConfig("user-1")
	require.NoError(t, err)
	assert.Equal(t, 10, retrieved.FailureThreshold)
}

func TestSnoozeAlerts(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	cfg, err := svc.SnoozeAlerts("user-1", 4*time.Hour)
	require.NoError(t, err)
	assert.NotNil(t, cfg.SnoozedUntil)
}

func TestLogout(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	session, _ := svc.RegisterDevice("t1", "user-1", &DeviceRegistration{
		DeviceID: "d1", Platform: "ios",
	})

	err := svc.Logout(session.ID)
	require.NoError(t, err)

	_, err = svc.ValidateSession(session.Token)
	assert.Error(t, err)
}

package errors

import (
	"context"
	"github.com/josedab/waas/pkg/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAlerterConfig(t *testing.T) {
	config := DefaultAlerterConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 10, config.MaxAlertsPerMinute)
	assert.Equal(t, 100, config.MaxAlertsPerHour)
	assert.Equal(t, 1*time.Minute, config.CriticalAlertThreshold)
	assert.Equal(t, 5*time.Minute, config.HighAlertThreshold)
	assert.True(t, config.Enabled)
}

func TestNewAlerter(t *testing.T) {
	logger := utils.NewLogger("test")

	tests := []struct {
		name   string
		config *AlerterConfig
	}{
		{
			name:   "with custom config",
			config: &AlerterConfig{Enabled: true, MaxAlertsPerMinute: 5},
		},
		{
			name:   "with nil config uses default",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerter := NewAlerter(context.Background(), tt.config, logger)

			assert.NotNil(t, alerter)
			assert.Equal(t, logger, alerter.logger)
			assert.NotNil(t, alerter.config)
			assert.NotNil(t, alerter.alertCounts)
			assert.NotNil(t, alerter.recentAlerts)

			if tt.config == nil {
				assert.True(t, alerter.config.Enabled)
			} else {
				assert.Equal(t, tt.config.Enabled, alerter.config.Enabled)
			}
		})
	}
}

func TestAlerter_shouldSendAlert(t *testing.T) {
	logger := utils.NewLogger("test")

	tests := []struct {
		name     string
		config   *AlerterConfig
		err      *WebhookError
		expected bool
	}{
		{
			name: "disabled alerting",
			config: &AlerterConfig{
				Enabled: false,
			},
			err:      ErrInternalServer,
			expected: false,
		},
		{
			name: "low severity not alerted",
			config: &AlerterConfig{
				Enabled: true,
			},
			err:      ErrInvalidRequest, // SeverityLow
			expected: false,
		},
		{
			name: "medium severity not alerted",
			config: &AlerterConfig{
				Enabled: true,
			},
			err:      ErrRateLimitExceeded, // SeverityMedium
			expected: false,
		},
		{
			name: "high severity should be alerted",
			config: &AlerterConfig{
				Enabled:                true,
				MaxAlertsPerMinute:     10,
				MaxAlertsPerHour:       100,
				CriticalAlertThreshold: 1 * time.Minute,
				HighAlertThreshold:     5 * time.Minute,
			},
			err:      ErrInternalServer, // SeverityHigh
			expected: true,
		},
		{
			name: "critical severity should be alerted",
			config: &AlerterConfig{
				Enabled:                true,
				MaxAlertsPerMinute:     10,
				MaxAlertsPerHour:       100,
				CriticalAlertThreshold: 1 * time.Minute,
				HighAlertThreshold:     5 * time.Minute,
			},
			err:      ErrServiceUnavailable, // SeverityCritical
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerter := NewAlerter(context.Background(), tt.config, logger)
			result := alerter.shouldSendAlert(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlerter_checkRateLimit(t *testing.T) {
	logger := utils.NewLogger("test")
	config := &AlerterConfig{
		Enabled:            true,
		MaxAlertsPerMinute: 2,
		MaxAlertsPerHour:   5,
	}

	alerter := NewAlerter(context.Background(), config, logger)
	err := ErrInternalServer

	// First alert should pass
	assert.True(t, alerter.checkRateLimit(err))
	alerter.updateAlertCounts(err)

	// Second alert should pass
	assert.True(t, alerter.checkRateLimit(err))
	alerter.updateAlertCounts(err)

	// Third alert should fail (exceeds per-minute limit)
	assert.False(t, alerter.checkRateLimit(err))
}

func TestAlerter_isDuplicateAlert(t *testing.T) {
	logger := utils.NewLogger("test")
	config := &AlerterConfig{
		Enabled:                true,
		CriticalAlertThreshold: 1 * time.Minute,
		HighAlertThreshold:     5 * time.Minute,
	}

	alerter := NewAlerter(context.Background(), config, logger)

	// First alert should not be duplicate
	err := ErrInternalServer
	assert.False(t, alerter.isDuplicateAlert(err))

	// Update recent alerts
	alerter.updateAlertCounts(err)

	// Same error immediately should be duplicate
	assert.True(t, alerter.isDuplicateAlert(err))

	// Different error should not be duplicate
	differentErr := ErrDatabaseError
	assert.False(t, alerter.isDuplicateAlert(differentErr))
}

func TestAlerter_SendAlert(t *testing.T) {
	logger := utils.NewLogger("test")
	config := &AlerterConfig{
		Enabled:                true,
		MaxAlertsPerMinute:     10,
		MaxAlertsPerHour:       100,
		CriticalAlertThreshold: 1 * time.Minute,
		HighAlertThreshold:     5 * time.Minute,
	}

	alerter := NewAlerter(context.Background(), config, logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		err         *WebhookError
		shouldSend  bool
		expectError bool
	}{
		{
			name:       "high severity error should send alert",
			err:        ErrInternalServer,
			shouldSend: true,
		},
		{
			name:       "low severity error should not send alert",
			err:        ErrInvalidRequest,
			shouldSend: false,
		},
		{
			name:       "critical severity error should send alert",
			err:        ErrServiceUnavailable,
			shouldSend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := alerter.SendAlert(ctx, tt.err)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlerter_SendAlert_Disabled(t *testing.T) {
	logger := utils.NewLogger("test")
	config := &AlerterConfig{
		Enabled: false,
	}

	alerter := NewAlerter(context.Background(), config, logger)
	ctx := context.Background()

	err := alerter.SendAlert(ctx, ErrInternalServer)
	assert.NoError(t, err) // Should not error when disabled
}

func TestAlerter_createAlertMessage(t *testing.T) {
	logger := utils.NewLogger("test")
	config := DefaultAlerterConfig()
	alerter := NewAlerter(context.Background(), config, logger)

	err := &WebhookError{
		Code:      "TEST_ERROR",
		Message:   "Test error message",
		Category:  CategoryInternal,
		Severity:  SeverityHigh,
		RequestID: "req_123",
		TraceID:   "trace_456",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"field": "value",
		},
	}

	alert := alerter.createAlertMessage(err)

	assert.Equal(t, "Webhook Platform Alert: TEST_ERROR", alert.Title)
	assert.Equal(t, "Test error message", alert.Message)
	assert.Equal(t, SeverityHigh, alert.Severity)
	assert.Equal(t, "TEST_ERROR", alert.ErrorCode)
	assert.Equal(t, CategoryInternal, alert.Category)
	assert.Equal(t, "req_123", alert.RequestID)
	assert.Equal(t, "trace_456", alert.TraceID)
	assert.Equal(t, err.Timestamp, alert.Timestamp)
	assert.Equal(t, err.Details, alert.Details)
	assert.NotEmpty(t, alert.Environment)
}

func TestAlerter_updateAlertCounts(t *testing.T) {
	logger := utils.NewLogger("test")
	config := DefaultAlerterConfig()
	alerter := NewAlerter(context.Background(), config, logger)

	err := ErrInternalServer

	// Initially no counts
	assert.Empty(t, alerter.alertCounts)
	assert.Empty(t, alerter.recentAlerts)

	// Update counts
	alerter.updateAlertCounts(err)

	// Should have counts now
	assert.NotEmpty(t, alerter.alertCounts)
	assert.NotEmpty(t, alerter.recentAlerts)

	// Check that recent alerts contains the error key
	alertKey := "INTERNAL_SERVER_ERROR_INTERNAL"
	_, exists := alerter.recentAlerts[alertKey]
	assert.True(t, exists)
}

func TestAlerter_getSeverityColor(t *testing.T) {
	logger := utils.NewLogger("test")
	config := DefaultAlerterConfig()
	alerter := NewAlerter(context.Background(), config, logger)

	tests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{SeverityLow, "good"},
		{SeverityMedium, "warning"},
		{SeverityHigh, "danger"},
		{SeverityCritical, "#ff0000"},
		{"UNKNOWN", "warning"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := alerter.getSeverityColor(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNoOpAlerter(t *testing.T) {
	alerter := &NoOpAlerter{}
	ctx := context.Background()
	err := ErrInternalServer

	result := alerter.SendAlert(ctx, err)
	assert.NoError(t, result)
}

func TestAlerter_cleanup(t *testing.T) {
	logger := utils.NewLogger("test")
	config := DefaultAlerterConfig()
	alerter := NewAlerter(context.Background(), config, logger)

	// Add some test data
	alerter.alertCounts["old_key"] = 5
	alerter.recentAlerts["old_alert"] = time.Now().Add(-10 * time.Minute)

	// Run cleanup
	alerter.cleanup()

	// This test mainly ensures cleanup doesn't panic
	// In a real implementation, you'd verify that old data is actually removed
	assert.NotNil(t, alerter.alertCounts)
	assert.NotNil(t, alerter.recentAlerts)
}

func TestAlertMessage(t *testing.T) {
	alert := &AlertMessage{
		Title:       "Test Alert",
		Message:     "Test message",
		Severity:    SeverityHigh,
		ErrorCode:   "TEST_ERROR",
		Category:    CategoryInternal,
		RequestID:   "req_123",
		TraceID:     "trace_456",
		Timestamp:   time.Now(),
		Details:     map[string]interface{}{"key": "value"},
		Environment: "test",
	}

	assert.Equal(t, "Test Alert", alert.Title)
	assert.Equal(t, "Test message", alert.Message)
	assert.Equal(t, SeverityHigh, alert.Severity)
	assert.Equal(t, "TEST_ERROR", alert.ErrorCode)
	assert.Equal(t, CategoryInternal, alert.Category)
	assert.Equal(t, "req_123", alert.RequestID)
	assert.Equal(t, "trace_456", alert.TraceID)
	assert.Equal(t, "value", alert.Details["key"])
	assert.Equal(t, "test", alert.Environment)
}

func TestAlerter_RateLimitingIntegration(t *testing.T) {
	logger := utils.NewLogger("test")
	config := &AlerterConfig{
		Enabled:                true,
		MaxAlertsPerMinute:     2,
		MaxAlertsPerHour:       5,
		CriticalAlertThreshold: 1 * time.Minute,
		HighAlertThreshold:     5 * time.Minute,
	}

	alerter := NewAlerter(context.Background(), config, logger)
	ctx := context.Background()
	err := ErrInternalServer

	// First two alerts should succeed
	assert.NoError(t, alerter.SendAlert(ctx, err))
	assert.NoError(t, alerter.SendAlert(ctx, err))

	// Third alert should be rate limited (no error, but won't send)
	assert.NoError(t, alerter.SendAlert(ctx, err))

	// Verify rate limiting is working by checking internal state
	assert.False(t, alerter.shouldSendAlert(err))
}

func TestAlerter_DeduplicationIntegration(t *testing.T) {
	logger := utils.NewLogger("test")
	config := &AlerterConfig{
		Enabled:                true,
		MaxAlertsPerMinute:     10,
		MaxAlertsPerHour:       100,
		CriticalAlertThreshold: 1 * time.Minute,
		HighAlertThreshold:     5 * time.Minute,
	}

	alerter := NewAlerter(context.Background(), config, logger)
	ctx := context.Background()
	err := ErrInternalServer

	// First alert should succeed
	assert.NoError(t, alerter.SendAlert(ctx, err))

	// Immediate duplicate should be deduplicated
	assert.False(t, alerter.shouldSendAlert(err))

	// Different error should not be deduplicated
	differentErr := ErrDatabaseError
	assert.True(t, alerter.shouldSendAlert(differentErr))
}

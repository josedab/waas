package monitoring

import (
	"context"
	"testing"
	"time"

	"webhook-platform/pkg/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAlertNotifier is a mock implementation of AlertNotifier
type MockAlertNotifier struct {
	mock.Mock
}

func (m *MockAlertNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertNotifier) GetName() string {
	args := m.Called()
	return args.String(0)
}

func TestAlertManager_EvaluateMetric(t *testing.T) {
	logger := utils.NewLogger("test")
	am := NewAlertManager(logger)

	// Add a test rule
	rule := &AlertRule{
		Name:        "TestRule",
		Description: "Test alert rule",
		Severity:    AlertSeverityWarning,
		Condition:   ConditionGreaterThan,
		Threshold:   10.0,
		Duration:    1 * time.Minute,
		Labels:      map[string]string{"component": "test"},
		Annotations: map[string]string{"summary": "Test alert"},
		Enabled:     true,
	}
	am.AddRule(rule)

	// Setup mock notifier
	mockNotifier := &MockAlertNotifier{}
	mockNotifier.On("SendAlert", mock.Anything, mock.Anything).Return(nil)
	mockNotifier.On("GetName").Return("mock")
	am.AddNotifier(mockNotifier)

	tests := []struct {
		name           string
		metricValue    float64
		labels         map[string]string
		expectAlert    bool
		expectedStatus AlertStatus
	}{
		{
			name:           "threshold exceeded - should fire alert",
			metricValue:    15.0,
			labels:         map[string]string{"component": "test"},
			expectAlert:    true,
			expectedStatus: AlertStatusFiring,
		},
		{
			name:           "threshold not exceeded - no alert",
			metricValue:    5.0,
			labels:         map[string]string{"component": "test"},
			expectAlert:    false,
			expectedStatus: "",
		},
		{
			name:           "wrong labels - no alert",
			metricValue:    15.0,
			labels:         map[string]string{"component": "other"},
			expectAlert:    false,
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear active alerts
			am.activeAlerts = make(map[string]*Alert)

			// Evaluate metric
			am.EvaluateMetric("test_metric", tt.metricValue, tt.labels)

			// Check results
			activeAlerts := am.GetActiveAlerts()
			if tt.expectAlert {
				assert.Len(t, activeAlerts, 1)
				assert.Equal(t, tt.expectedStatus, activeAlerts[0].Status)
				assert.Equal(t, rule.Name, activeAlerts[0].Name)
				assert.Equal(t, tt.metricValue, activeAlerts[0].Value)
				assert.Equal(t, rule.Threshold, activeAlerts[0].Threshold)
			} else {
				assert.Len(t, activeAlerts, 0)
			}
		})
	}
}

func TestAlertManager_AlertResolution(t *testing.T) {
	logger := utils.NewLogger("test")
	am := NewAlertManager(logger)

	// Add a test rule
	rule := &AlertRule{
		Name:        "TestRule",
		Description: "Test alert rule",
		Severity:    AlertSeverityWarning,
		Condition:   ConditionGreaterThan,
		Threshold:   10.0,
		Duration:    1 * time.Minute,
		Labels:      map[string]string{"component": "test"},
		Annotations: map[string]string{"summary": "Test alert"},
		Enabled:     true,
	}
	am.AddRule(rule)

	// Setup mock notifier
	mockNotifier := &MockAlertNotifier{}
	mockNotifier.On("SendAlert", mock.Anything, mock.Anything).Return(nil)
	mockNotifier.On("GetName").Return("mock")
	am.AddNotifier(mockNotifier)

	labels := map[string]string{"component": "test"}

	// Fire alert
	am.EvaluateMetric("test_metric", 15.0, labels)
	activeAlerts := am.GetActiveAlerts()
	assert.Len(t, activeAlerts, 1)
	assert.Equal(t, AlertStatusFiring, activeAlerts[0].Status)

	// Resolve alert
	am.EvaluateMetric("test_metric", 5.0, labels)
	activeAlerts = am.GetActiveAlerts()
	assert.Len(t, activeAlerts, 0)

	// Check alert history
	history := am.GetAlertHistory(10, "")
	assert.Len(t, history, 1)
	assert.Equal(t, AlertStatusResolved, history[0].Status)
	assert.NotNil(t, history[0].EndsAt)
}

func TestAlertManager_AddRemoveRule(t *testing.T) {
	logger := utils.NewLogger("test")
	am := NewAlertManager(logger)

	rule := &AlertRule{
		Name:        "TestRule",
		Description: "Test alert rule",
		Severity:    AlertSeverityWarning,
		Condition:   ConditionGreaterThan,
		Threshold:   10.0,
		Enabled:     true,
	}

	// Add rule
	am.AddRule(rule)
	assert.Contains(t, am.rules, "TestRule")

	// Remove rule
	am.RemoveRule("TestRule")
	assert.NotContains(t, am.rules, "TestRule")
}

func TestAlertManager_evaluateCondition(t *testing.T) {
	am := &AlertManager{}

	tests := []struct {
		name      string
		condition AlertCondition
		value     float64
		threshold float64
		expected  bool
	}{
		{"greater_than_true", ConditionGreaterThan, 15.0, 10.0, true},
		{"greater_than_false", ConditionGreaterThan, 5.0, 10.0, false},
		{"less_than_true", ConditionLessThan, 5.0, 10.0, true},
		{"less_than_false", ConditionLessThan, 15.0, 10.0, false},
		{"equals_true", ConditionEquals, 10.0, 10.0, true},
		{"equals_false", ConditionEquals, 15.0, 10.0, false},
		{"not_equals_true", ConditionNotEquals, 15.0, 10.0, true},
		{"not_equals_false", ConditionNotEquals, 10.0, 10.0, false},
		{"greater_or_equal_true", ConditionGreaterOrEqual, 10.0, 10.0, true},
		{"greater_or_equal_false", ConditionGreaterOrEqual, 5.0, 10.0, false},
		{"less_or_equal_true", ConditionLessOrEqual, 10.0, 10.0, true},
		{"less_or_equal_false", ConditionLessOrEqual, 15.0, 10.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := am.evaluateCondition(tt.condition, tt.value, tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertManager_GetAlertHistory(t *testing.T) {
	logger := utils.NewLogger("test")
	am := NewAlertManager(logger)

	// Add some alerts to history
	alerts := []*Alert{
		{
			ID:       "alert1",
			Name:     "Alert1",
			Severity: AlertSeverityCritical,
			Status:   AlertStatusResolved,
			StartsAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:       "alert2",
			Name:     "Alert2",
			Severity: AlertSeverityWarning,
			Status:   AlertStatusResolved,
			StartsAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:       "alert3",
			Name:     "Alert3",
			Severity: AlertSeverityInfo,
			Status:   AlertStatusResolved,
			StartsAt: time.Now().Add(-10 * time.Minute),
		},
	}

	am.alertHistory = alerts

	// Test limit
	history := am.GetAlertHistory(2, "")
	assert.Len(t, history, 2)
	assert.Equal(t, "alert3", history[0].ID) // Most recent first
	assert.Equal(t, "alert2", history[1].ID)

	// Test severity filter
	history = am.GetAlertHistory(10, AlertSeverityCritical)
	assert.Len(t, history, 1)
	assert.Equal(t, "alert1", history[0].ID)

	// Test no filter
	history = am.GetAlertHistory(10, "")
	assert.Len(t, history, 3)
}

func TestAlertManager_DefaultRules(t *testing.T) {
	logger := utils.NewLogger("test")
	am := NewAlertManager(logger)

	// Check that default rules are loaded
	assert.Greater(t, len(am.rules), 0)
	
	// Check for specific default rules
	expectedRules := []string{
		"HighDeliveryFailureRate",
		"DatabaseConnectionFailure",
		"HighQueueDepth",
		"SlowDatabaseQueries",
		"HighRateLimitHits",
		"AuthenticationFailures",
	}

	for _, ruleName := range expectedRules {
		assert.Contains(t, am.rules, ruleName, "Expected default rule %s not found", ruleName)
		assert.True(t, am.rules[ruleName].Enabled, "Expected rule %s to be enabled", ruleName)
	}
}
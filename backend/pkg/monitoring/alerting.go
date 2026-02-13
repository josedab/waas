package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/josedab/waas/pkg/utils"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

// Alert represents a monitoring alert
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Severity    AlertSeverity     `json:"severity"`
	Status      AlertStatus       `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at,omitempty"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Severity    AlertSeverity     `json:"severity"`
	Condition   AlertCondition    `json:"condition"`
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Enabled     bool              `json:"enabled"`
}

// AlertCondition defines the condition type for alert rules
type AlertCondition string

const (
	ConditionGreaterThan    AlertCondition = "greater_than"
	ConditionLessThan       AlertCondition = "less_than"
	ConditionEquals         AlertCondition = "equals"
	ConditionNotEquals      AlertCondition = "not_equals"
	ConditionGreaterOrEqual AlertCondition = "greater_or_equal"
	ConditionLessOrEqual    AlertCondition = "less_or_equal"
)

// AlertManager manages alert rules and active alerts
type AlertManager struct {
	rules        map[string]*AlertRule
	activeAlerts map[string]*Alert
	alertHistory []*Alert
	mutex        sync.RWMutex
	logger       *utils.Logger
	notifiers    []AlertNotifier
}

// AlertNotifier interface for sending alert notifications
type AlertNotifier interface {
	SendAlert(ctx context.Context, alert *Alert) error
	GetName() string
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger *utils.Logger) *AlertManager {
	am := &AlertManager{
		rules:        make(map[string]*AlertRule),
		activeAlerts: make(map[string]*Alert),
		alertHistory: make([]*Alert, 0),
		logger:       logger,
		notifiers:    make([]AlertNotifier, 0),
	}

	// Initialize default alert rules
	am.initializeDefaultRules()

	return am
}

// AddNotifier adds an alert notifier
func (am *AlertManager) AddNotifier(notifier AlertNotifier) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.notifiers = append(am.notifiers, notifier)
}

// AddRule adds or updates an alert rule
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.rules[rule.Name] = rule
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(name string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	delete(am.rules, name)
}

// EvaluateMetric evaluates a metric value against all applicable rules
func (am *AlertManager) EvaluateMetric(metricName string, value float64, labels map[string]string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// Check if rule applies to this metric (simple name matching for now)
		if !am.ruleApplies(rule, metricName, labels) {
			continue
		}

		alertID := am.generateAlertID(rule.Name, labels)

		if am.evaluateCondition(rule.Condition, value, rule.Threshold) {
			// Condition met - fire alert if not already active
			if _, exists := am.activeAlerts[alertID]; !exists {
				alert := &Alert{
					ID:          alertID,
					Name:        rule.Name,
					Description: rule.Description,
					Severity:    rule.Severity,
					Status:      AlertStatusFiring,
					Labels:      am.mergeLabels(rule.Labels, labels),
					Annotations: rule.Annotations,
					StartsAt:    time.Now(),
					Value:       value,
					Threshold:   rule.Threshold,
				}

				am.activeAlerts[alertID] = alert
				am.alertHistory = append(am.alertHistory, alert)

				am.logger.Warn("Alert fired", map[string]interface{}{
					"alert_id":   alertID,
					"alert_name": rule.Name,
					"severity":   rule.Severity,
					"value":      value,
					"threshold":  rule.Threshold,
					"labels":     labels,
				})

				// Send notifications asynchronously
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					am.sendNotifications(ctx, alert)
				}()
			}
		} else {
			// Condition not met - resolve alert if active
			if alert, exists := am.activeAlerts[alertID]; exists {
				now := time.Now()
				alert.Status = AlertStatusResolved
				alert.EndsAt = &now

				delete(am.activeAlerts, alertID)

				am.logger.Info("Alert resolved", map[string]interface{}{
					"alert_id":   alertID,
					"alert_name": rule.Name,
					"duration":   now.Sub(alert.StartsAt).String(),
				})

				// Send resolution notifications asynchronously
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					am.sendNotifications(ctx, alert)
				}()
			}
		}
	}
}

// GetActiveAlerts returns all currently active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	alerts := make([]*Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alertCopy := *alert
		alerts = append(alerts, &alertCopy)
	}
	return alerts
}

// GetAlertHistory returns alert history with optional filtering
func (am *AlertManager) GetAlertHistory(limit int, severity AlertSeverity) []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	filtered := make([]*Alert, 0)
	for i := len(am.alertHistory) - 1; i >= 0 && len(filtered) < limit; i-- {
		alert := am.alertHistory[i]
		if severity == "" || alert.Severity == severity {
			alertCopy := *alert
			filtered = append(filtered, &alertCopy)
		}
	}
	return filtered
}

// initializeDefaultRules sets up default alert rules for the webhook platform
func (am *AlertManager) initializeDefaultRules() {
	defaultRules := []*AlertRule{
		{
			Name:        "HighDeliveryFailureRate",
			Description: "Webhook delivery failure rate is above threshold",
			Severity:    AlertSeverityCritical,
			Condition:   ConditionGreaterThan,
			Threshold:   0.05, // 5% failure rate
			Duration:    5 * time.Minute,
			Labels:      map[string]string{"component": "delivery"},
			Annotations: map[string]string{
				"summary":     "High webhook delivery failure rate detected",
				"description": "Webhook delivery failure rate is {{ .Value }}%, which is above the threshold of {{ .Threshold }}%",
			},
			Enabled: true,
		},
		{
			Name:        "DatabaseConnectionFailure",
			Description: "Database connection health check failed",
			Severity:    AlertSeverityCritical,
			Condition:   ConditionLessThan,
			Threshold:   1.0, // Health status should be 1.0 for healthy
			Duration:    1 * time.Minute,
			Labels:      map[string]string{"component": "database"},
			Annotations: map[string]string{
				"summary":     "Database connection failure",
				"description": "Database health check is failing",
			},
			Enabled: true,
		},
		{
			Name:        "HighQueueDepth",
			Description: "Webhook delivery queue depth is too high",
			Severity:    AlertSeverityWarning,
			Condition:   ConditionGreaterThan,
			Threshold:   1000, // More than 1000 messages in queue
			Duration:    10 * time.Minute,
			Labels:      map[string]string{"component": "queue"},
			Annotations: map[string]string{
				"summary":     "High queue depth detected",
				"description": "Webhook delivery queue has {{ .Value }} messages, which is above the threshold of {{ .Threshold }}",
			},
			Enabled: true,
		},
		{
			Name:        "SlowDatabaseQueries",
			Description: "Database queries are taking too long",
			Severity:    AlertSeverityWarning,
			Condition:   ConditionGreaterThan,
			Threshold:   1.0, // Queries taking more than 1 second
			Duration:    5 * time.Minute,
			Labels:      map[string]string{"component": "database"},
			Annotations: map[string]string{
				"summary":     "Slow database queries detected",
				"description": "Database queries are taking {{ .Value }}s on average, which is above the threshold of {{ .Threshold }}s",
			},
			Enabled: true,
		},
		{
			Name:        "HighRateLimitHits",
			Description: "High number of rate limit hits detected",
			Severity:    AlertSeverityWarning,
			Condition:   ConditionGreaterThan,
			Threshold:   100, // More than 100 rate limit hits per minute
			Duration:    5 * time.Minute,
			Labels:      map[string]string{"component": "rate_limiting"},
			Annotations: map[string]string{
				"summary":     "High rate limit hits",
				"description": "{{ .Value }} rate limit hits detected in the last minute",
			},
			Enabled: true,
		},
		{
			Name:        "AuthenticationFailures",
			Description: "High number of authentication failures",
			Severity:    AlertSeverityWarning,
			Condition:   ConditionGreaterThan,
			Threshold:   50, // More than 50 auth failures per minute
			Duration:    5 * time.Minute,
			Labels:      map[string]string{"component": "authentication"},
			Annotations: map[string]string{
				"summary":     "High authentication failure rate",
				"description": "{{ .Value }} authentication failures detected in the last minute",
			},
			Enabled: true,
		},
	}

	for _, rule := range defaultRules {
		am.rules[rule.Name] = rule
	}
}

// ruleApplies checks if a rule applies to the given metric and labels
func (am *AlertManager) ruleApplies(rule *AlertRule, metricName string, labels map[string]string) bool {
	// Simple implementation - in a real system, you'd have more sophisticated matching
	for key, value := range rule.Labels {
		if labels[key] != value {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates if the condition is met
func (am *AlertManager) evaluateCondition(condition AlertCondition, value, threshold float64) bool {
	switch condition {
	case ConditionGreaterThan:
		return value > threshold
	case ConditionLessThan:
		return value < threshold
	case ConditionEquals:
		return value == threshold
	case ConditionNotEquals:
		return value != threshold
	case ConditionGreaterOrEqual:
		return value >= threshold
	case ConditionLessOrEqual:
		return value <= threshold
	default:
		return false
	}
}

// generateAlertID generates a unique ID for an alert
func (am *AlertManager) generateAlertID(ruleName string, labels map[string]string) string {
	id := ruleName
	for key, value := range labels {
		id += fmt.Sprintf("_%s_%s", key, value)
	}
	return id
}

// mergeLabels merges rule labels with metric labels
func (am *AlertManager) mergeLabels(ruleLabels, metricLabels map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range ruleLabels {
		merged[k] = v
	}
	for k, v := range metricLabels {
		merged[k] = v
	}
	return merged
}

// sendNotifications sends alert notifications to all configured notifiers
func (am *AlertManager) sendNotifications(ctx context.Context, alert *Alert) {
	for _, notifier := range am.notifiers {
		if err := notifier.SendAlert(ctx, alert); err != nil {
			am.logger.Error("Failed to send alert notification", map[string]interface{}{
				"notifier":   notifier.GetName(),
				"alert_id":   alert.ID,
				"alert_name": alert.Name,
				"error":      err.Error(),
			})
		}
	}
}

package errors

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/josedab/waas/pkg/utils"
)

// AlerterConfig holds configuration for the alerter
type AlerterConfig struct {
	// Rate limiting for alerts
	MaxAlertsPerMinute int
	MaxAlertsPerHour   int

	// Alert channels
	SlackWebhookURL string
	EmailRecipients []string

	// Alert thresholds
	CriticalAlertThreshold time.Duration
	HighAlertThreshold     time.Duration

	// Enable/disable alerting
	Enabled bool
}

// DefaultAlerterConfig returns default alerter configuration
func DefaultAlerterConfig() *AlerterConfig {
	return &AlerterConfig{
		MaxAlertsPerMinute:     10,
		MaxAlertsPerHour:       100,
		CriticalAlertThreshold: 1 * time.Minute,
		HighAlertThreshold:     5 * time.Minute,
		Enabled:                true,
	}
}

// Alerter handles sending alerts for errors
type Alerter struct {
	config *AlerterConfig
	logger *utils.Logger

	// Rate limiting
	alertCounts     map[string]int
	alertCountsLock sync.RWMutex
	lastReset       time.Time

	// Alert deduplication
	recentAlerts     map[string]time.Time
	recentAlertsLock sync.RWMutex
}

// NewAlerter creates a new alerter instance
func NewAlerter(config *AlerterConfig, logger *utils.Logger) *Alerter {
	if config == nil {
		config = DefaultAlerterConfig()
	}

	alerter := &Alerter{
		config:       config,
		logger:       logger,
		alertCounts:  make(map[string]int),
		recentAlerts: make(map[string]time.Time),
		lastReset:    time.Now(),
	}

	// Start cleanup goroutine
	go alerter.cleanupRoutine()

	return alerter
}

// SendAlert sends an alert for the given error
func (a *Alerter) SendAlert(ctx context.Context, err *WebhookError) error {
	if !a.config.Enabled {
		return nil
	}

	// Check if we should send this alert
	if !a.shouldSendAlert(err) {
		return nil
	}

	// Create alert message
	alert := a.createAlertMessage(err)

	// Send to configured channels
	var sendErrors []error

	// Send to Slack if configured
	if a.config.SlackWebhookURL != "" {
		if err := a.sendSlackAlert(ctx, alert); err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("slack alert failed: %w", err))
		}
	}

	// Send email alerts if configured
	if len(a.config.EmailRecipients) > 0 {
		if err := a.sendEmailAlert(ctx, alert); err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("email alert failed: %w", err))
		}
	}

	// Log the alert
	a.logAlert(alert)

	// Update rate limiting counters
	a.updateAlertCounts(err)

	// Return combined errors if any
	if len(sendErrors) > 0 {
		return fmt.Errorf("alert sending failed: %v", sendErrors)
	}

	return nil
}

// shouldSendAlert determines if an alert should be sent based on various criteria
func (a *Alerter) shouldSendAlert(err *WebhookError) bool {
	// Check if alerting is enabled
	if !a.config.Enabled {
		return false
	}

	// Check severity threshold
	if err.Severity != SeverityHigh && err.Severity != SeverityCritical {
		return false
	}

	// Check rate limits
	if !a.checkRateLimit(err) {
		return false
	}

	// Check for duplicate alerts (deduplication)
	if a.isDuplicateAlert(err) {
		return false
	}

	return true
}

// checkRateLimit checks if we're within rate limits for alerts
func (a *Alerter) checkRateLimit(err *WebhookError) bool {
	a.alertCountsLock.RLock()
	defer a.alertCountsLock.RUnlock()

	// Check per-minute limit
	minuteKey := fmt.Sprintf("minute_%s", time.Now().Format("2006-01-02_15:04"))
	if count, exists := a.alertCounts[minuteKey]; exists && count >= a.config.MaxAlertsPerMinute {
		return false
	}

	// Check per-hour limit
	hourKey := fmt.Sprintf("hour_%s", time.Now().Format("2006-01-02_15"))
	if count, exists := a.alertCounts[hourKey]; exists && count >= a.config.MaxAlertsPerHour {
		return false
	}

	return true
}

// isDuplicateAlert checks if this is a duplicate alert within the threshold period
func (a *Alerter) isDuplicateAlert(err *WebhookError) bool {
	a.recentAlertsLock.RLock()
	defer a.recentAlertsLock.RUnlock()

	// Create a key for deduplication based on error code and category
	alertKey := fmt.Sprintf("%s_%s", err.Code, err.Category)

	if lastSent, exists := a.recentAlerts[alertKey]; exists {
		threshold := a.config.HighAlertThreshold
		if err.Severity == SeverityCritical {
			threshold = a.config.CriticalAlertThreshold
		}

		if time.Since(lastSent) < threshold {
			return true // Duplicate within threshold
		}
	}

	return false
}

// updateAlertCounts updates the rate limiting counters
func (a *Alerter) updateAlertCounts(err *WebhookError) {
	a.alertCountsLock.Lock()
	defer a.alertCountsLock.Unlock()

	now := time.Now()

	// Update per-minute counter
	minuteKey := fmt.Sprintf("minute_%s", now.Format("2006-01-02_15:04"))
	a.alertCounts[minuteKey]++

	// Update per-hour counter
	hourKey := fmt.Sprintf("hour_%s", now.Format("2006-01-02_15"))
	a.alertCounts[hourKey]++

	// Update recent alerts for deduplication
	a.recentAlertsLock.Lock()
	alertKey := fmt.Sprintf("%s_%s", err.Code, err.Category)
	a.recentAlerts[alertKey] = now
	a.recentAlertsLock.Unlock()
}

// AlertMessage represents an alert message
type AlertMessage struct {
	Title       string
	Message     string
	Severity    ErrorSeverity
	ErrorCode   string
	Category    ErrorCategory
	RequestID   string
	TraceID     string
	Timestamp   time.Time
	Details     map[string]interface{}
	Environment string
}

// createAlertMessage creates an alert message from an error
func (a *Alerter) createAlertMessage(err *WebhookError) *AlertMessage {
	return &AlertMessage{
		Title:       fmt.Sprintf("Webhook Platform Alert: %s", err.Code),
		Message:     err.Message,
		Severity:    err.Severity,
		ErrorCode:   err.Code,
		Category:    err.Category,
		RequestID:   err.RequestID,
		TraceID:     err.TraceID,
		Timestamp:   err.Timestamp,
		Details:     err.Details,
		Environment: a.getEnvironment(),
	}
}

// sendSlackAlert sends an alert to Slack
func (a *Alerter) sendSlackAlert(ctx context.Context, alert *AlertMessage) error {
	// This is a placeholder implementation
	// In a real implementation, you would use the Slack API or webhook
	a.logger.Info("Slack alert would be sent", map[string]interface{}{
		"title":      alert.Title,
		"message":    alert.Message,
		"severity":   alert.Severity,
		"error_code": alert.ErrorCode,
		"request_id": alert.RequestID,
	})

	// TODO(#8): Implement actual Slack webhook integration — https://github.com/josedab/waas/issues/8
	// Example:
	// payload := map[string]interface{}{
	//     "text": alert.Title,
	//     "attachments": []map[string]interface{}{
	//         {
	//             "color": a.getSeverityColor(alert.Severity),
	//             "fields": []map[string]interface{}{
	//                 {"title": "Message", "value": alert.Message, "short": false},
	//                 {"title": "Error Code", "value": alert.ErrorCode, "short": true},
	//                 {"title": "Category", "value": string(alert.Category), "short": true},
	//                 {"title": "Request ID", "value": alert.RequestID, "short": true},
	//                 {"title": "Environment", "value": alert.Environment, "short": true},
	//             },
	//             "ts": alert.Timestamp.Unix(),
	//         },
	//     },
	// }
	// return a.sendWebhook(ctx, a.config.SlackWebhookURL, payload)

	return nil
}

// sendEmailAlert sends an alert via email
func (a *Alerter) sendEmailAlert(ctx context.Context, alert *AlertMessage) error {
	// This is a placeholder implementation
	// In a real implementation, you would use an email service like SendGrid, SES, etc.
	a.logger.Info("Email alert would be sent", map[string]interface{}{
		"title":      alert.Title,
		"message":    alert.Message,
		"severity":   alert.Severity,
		"recipients": a.config.EmailRecipients,
		"request_id": alert.RequestID,
	})

	// TODO(#9): Implement actual email sending — https://github.com/josedab/waas/issues/9
	// Example:
	// subject := fmt.Sprintf("[%s] %s", alert.Severity, alert.Title)
	// body := a.formatEmailBody(alert)
	// return a.emailService.SendEmail(ctx, a.config.EmailRecipients, subject, body)

	return nil
}

// logAlert logs the alert for audit purposes
func (a *Alerter) logAlert(alert *AlertMessage) {
	a.logger.Warn("Alert sent", map[string]interface{}{
		"alert_title":   alert.Title,
		"alert_message": alert.Message,
		"severity":      alert.Severity,
		"error_code":    alert.ErrorCode,
		"category":      alert.Category,
		"request_id":    alert.RequestID,
		"trace_id":      alert.TraceID,
		"environment":   alert.Environment,
		"timestamp":     alert.Timestamp,
	})
}

// cleanupRoutine periodically cleans up old rate limiting data
func (a *Alerter) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.cleanup()
	}
}

// cleanup removes old rate limiting and deduplication data
func (a *Alerter) cleanup() {
	now := time.Now()

	// Clean up alert counts (keep last 2 hours)
	a.alertCountsLock.Lock()
	for key := range a.alertCounts {
		// Parse timestamp from key and check if it's old
		if a.isOldCountKey(key, now) {
			delete(a.alertCounts, key)
		}
	}
	a.alertCountsLock.Unlock()

	// Clean up recent alerts (keep based on max threshold)
	a.recentAlertsLock.Lock()
	maxThreshold := a.config.HighAlertThreshold
	if a.config.CriticalAlertThreshold > maxThreshold {
		maxThreshold = a.config.CriticalAlertThreshold
	}

	for key, timestamp := range a.recentAlerts {
		if now.Sub(timestamp) > maxThreshold*2 { // Keep for 2x the threshold
			delete(a.recentAlerts, key)
		}
	}
	a.recentAlertsLock.Unlock()
}

// isOldCountKey checks if a count key is old and should be cleaned up
func (a *Alerter) isOldCountKey(key string, now time.Time) bool {
	// This is a simplified check - in a real implementation you'd parse the timestamp
	// For now, just clean up keys older than 2 hours
	return true // Placeholder - implement proper timestamp parsing
}

// getEnvironment returns the current environment (dev, staging, prod)
func (a *Alerter) getEnvironment() string {
	// This would typically come from environment variables or configuration
	return "production" // Placeholder
}

// getSeverityColor returns a color code for Slack based on severity
func (a *Alerter) getSeverityColor(severity ErrorSeverity) string {
	switch severity {
	case SeverityLow:
		return "good"
	case SeverityMedium:
		return "warning"
	case SeverityHigh:
		return "danger"
	case SeverityCritical:
		return "#ff0000"
	default:
		return "warning"
	}
}

// NoOpAlerter is a no-op implementation of AlerterInterface for testing
type NoOpAlerter struct{}

// SendAlert does nothing (no-op implementation)
func (n *NoOpAlerter) SendAlert(ctx context.Context, err *WebhookError) error {
	return nil
}

package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"webhook-platform/pkg/utils"
)

// LogNotifier sends alerts to the application logs
type LogNotifier struct {
	logger *utils.Logger
}

// NewLogNotifier creates a new log notifier
func NewLogNotifier(logger *utils.Logger) *LogNotifier {
	return &LogNotifier{
		logger: logger,
	}
}

// SendAlert sends an alert to the logs
func (ln *LogNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	logLevel := "warn"
	if alert.Severity == AlertSeverityCritical {
		logLevel = "error"
	} else if alert.Severity == AlertSeverityInfo {
		logLevel = "info"
	}

	message := fmt.Sprintf("Alert %s: %s", alert.Status, alert.Name)
	fields := map[string]interface{}{
		"alert_id":     alert.ID,
		"alert_name":   alert.Name,
		"severity":     alert.Severity,
		"status":       alert.Status,
		"value":        alert.Value,
		"threshold":    alert.Threshold,
		"labels":       alert.Labels,
		"annotations":  alert.Annotations,
		"starts_at":    alert.StartsAt,
	}

	if alert.EndsAt != nil {
		fields["ends_at"] = *alert.EndsAt
		fields["duration"] = alert.EndsAt.Sub(alert.StartsAt).String()
	}

	switch logLevel {
	case "error":
		ln.logger.Error(message, fields)
	case "warn":
		ln.logger.Warn(message, fields)
	case "info":
		ln.logger.Info(message, fields)
	}

	return nil
}

// GetName returns the notifier name
func (ln *LogNotifier) GetName() string {
	return "log"
}

// WebhookNotifier sends alerts via HTTP webhooks
type WebhookNotifier struct {
	webhookURL string
	timeout    time.Duration
	client     *http.Client
	logger     *utils.Logger
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(webhookURL string, timeout time.Duration, logger *utils.Logger) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SendAlert sends an alert via webhook
func (wn *WebhookNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	payload := map[string]interface{}{
		"alert_id":     alert.ID,
		"name":         alert.Name,
		"description":  alert.Description,
		"severity":     alert.Severity,
		"status":       alert.Status,
		"labels":       alert.Labels,
		"annotations":  alert.Annotations,
		"starts_at":    alert.StartsAt.Format(time.RFC3339),
		"value":        alert.Value,
		"threshold":    alert.Threshold,
	}

	if alert.EndsAt != nil {
		payload["ends_at"] = alert.EndsAt.Format(time.RFC3339)
		payload["duration"] = alert.EndsAt.Sub(alert.StartsAt).String()
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal alert payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", wn.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "webhook-platform-alerting/1.0")

	resp, err := wn.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	wn.logger.Info("Alert sent via webhook", map[string]interface{}{
		"alert_id":    alert.ID,
		"webhook_url": wn.webhookURL,
		"status_code": resp.StatusCode,
	})

	return nil
}

// GetName returns the notifier name
func (wn *WebhookNotifier) GetName() string {
	return "webhook"
}

// SlackNotifier sends alerts to Slack
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	timeout    time.Duration
	client     *http.Client
	logger     *utils.Logger
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL, channel, username string, timeout time.Duration, logger *utils.Logger) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SendAlert sends an alert to Slack
func (sn *SlackNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	color := "warning"
	if alert.Severity == AlertSeverityCritical {
		color = "danger"
	} else if alert.Severity == AlertSeverityInfo {
		color = "good"
	}

	if alert.Status == AlertStatusResolved {
		color = "good"
	}

	var title string
	if alert.Status == AlertStatusFiring {
		title = fmt.Sprintf("🚨 Alert: %s", alert.Name)
	} else {
		title = fmt.Sprintf("✅ Resolved: %s", alert.Name)
	}

	fields := []map[string]interface{}{
		{
			"title": "Severity",
			"value": string(alert.Severity),
			"short": true,
		},
		{
			"title": "Status",
			"value": string(alert.Status),
			"short": true,
		},
		{
			"title": "Value",
			"value": fmt.Sprintf("%.2f", alert.Value),
			"short": true,
		},
		{
			"title": "Threshold",
			"value": fmt.Sprintf("%.2f", alert.Threshold),
			"short": true,
		},
	}

	if alert.EndsAt != nil {
		fields = append(fields, map[string]interface{}{
			"title": "Duration",
			"value": alert.EndsAt.Sub(alert.StartsAt).String(),
			"short": true,
		})
	}

	payload := map[string]interface{}{
		"channel":  sn.channel,
		"username": sn.username,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     title,
				"text":      alert.Description,
				"fields":    fields,
				"timestamp": alert.StartsAt.Unix(),
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sn.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sn.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Slack webhook returned non-success status: %d", resp.StatusCode)
	}

	sn.logger.Info("Alert sent to Slack", map[string]interface{}{
		"alert_id": alert.ID,
		"channel":  sn.channel,
		"status_code": resp.StatusCode,
	})

	return nil
}

// GetName returns the notifier name
func (sn *SlackNotifier) GetName() string {
	return "slack"
}

// EmailNotifier sends alerts via email (placeholder implementation)
type EmailNotifier struct {
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	fromAddress  string
	toAddresses  []string
	logger       *utils.Logger
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(smtpHost string, smtpPort int, username, password, fromAddress string, toAddresses []string, logger *utils.Logger) *EmailNotifier {
	return &EmailNotifier{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		username:     username,
		password:     password,
		fromAddress:  fromAddress,
		toAddresses:  toAddresses,
		logger:       logger,
	}
}

// SendAlert sends an alert via email
func (en *EmailNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	// This is a placeholder implementation
	// In a real system, you would implement SMTP email sending
	
	subject := fmt.Sprintf("[%s] %s: %s", alert.Severity, alert.Status, alert.Name)
	
	en.logger.Info("Email alert notification (placeholder)", map[string]interface{}{
		"alert_id":     alert.ID,
		"subject":      subject,
		"to_addresses": en.toAddresses,
		"from_address": en.fromAddress,
	})

	// TODO: Implement actual email sending using net/smtp or a third-party service
	return nil
}

// GetName returns the notifier name
func (en *EmailNotifier) GetName() string {
	return "email"
}
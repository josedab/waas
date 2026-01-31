package sla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Notifier sends SLA alerts to external channels
type Notifier struct {
	client *http.Client
	config *NotifierConfig
}

// NotifierConfig holds notification integration settings
type NotifierConfig struct {
	SlackWebhookURL    string `json:"slack_webhook_url"`
	PagerDutyRoutingKey string `json:"pagerduty_routing_key"`
	EmailEndpoint      string `json:"email_endpoint"`
	WebhookURL         string `json:"webhook_url"`
}

// NewNotifier creates a notifier with the given config
func NewNotifier(config *NotifierConfig) *Notifier {
	return &Notifier{
		client: &http.Client{Timeout: 10 * time.Second},
		config: config,
	}
}

// NotifyBreach sends notifications for an SLA breach to all configured channels
func (n *Notifier) NotifyBreach(ctx context.Context, breach *Breach, target *Target, channels []string) error {
	var lastErr error
	for _, ch := range channels {
		var err error
		switch ch {
		case ChannelSlack:
			err = n.notifySlack(ctx, breach, target)
		case ChannelPagerDuty:
			err = n.notifyPagerDuty(ctx, breach, target)
		case ChannelEmail:
			err = n.notifyEmail(ctx, breach, target)
		case ChannelWebhook:
			err = n.notifyWebhook(ctx, breach, target)
		}
		if err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// NotifyRecovery sends recovery notifications
func (n *Notifier) NotifyRecovery(ctx context.Context, breach *Breach, target *Target, channels []string) error {
	var lastErr error
	for _, ch := range channels {
		var err error
		switch ch {
		case ChannelSlack:
			err = n.notifySlackRecovery(ctx, breach, target)
		case ChannelPagerDuty:
			err = n.resolvePagerDuty(ctx, breach)
		}
		if err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (n *Notifier) notifySlack(ctx context.Context, breach *Breach, target *Target) error {
	if n.config.SlackWebhookURL == "" {
		return nil
	}

	color := "#ff9900"
	if breach.Severity == SeverityCritical {
		color = "#ff0000"
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"pretext": fmt.Sprintf("🚨 SLA Breach: %s", target.Name),
				"fields": []map[string]interface{}{
					{"title": "Breach Type", "value": breach.BreachType, "short": true},
					{"title": "Severity", "value": breach.Severity, "short": true},
					{"title": "Expected", "value": fmt.Sprintf("%.2f", breach.ExpectedVal), "short": true},
					{"title": "Actual", "value": fmt.Sprintf("%.2f", breach.ActualVal), "short": true},
					{"title": "Endpoint", "value": breach.EndpointID, "short": true},
					{"title": "Time", "value": breach.CreatedAt.Format(time.RFC3339), "short": true},
				},
				"footer": "WaaS SLA Monitor",
				"ts":     breach.CreatedAt.Unix(),
			},
		},
	}

	return n.postJSON(ctx, n.config.SlackWebhookURL, payload)
}

func (n *Notifier) notifySlackRecovery(ctx context.Context, breach *Breach, target *Target) error {
	if n.config.SlackWebhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  "#36a64f",
				"pretext": fmt.Sprintf("✅ SLA Recovered: %s", target.Name),
				"fields": []map[string]interface{}{
					{"title": "Breach Type", "value": breach.BreachType, "short": true},
					{"title": "Duration", "value": formatDuration(time.Since(breach.CreatedAt)), "short": true},
				},
				"footer": "WaaS SLA Monitor",
			},
		},
	}

	return n.postJSON(ctx, n.config.SlackWebhookURL, payload)
}

func (n *Notifier) notifyPagerDuty(ctx context.Context, breach *Breach, target *Target) error {
	if n.config.PagerDutyRoutingKey == "" {
		return nil
	}

	severity := "warning"
	if breach.Severity == SeverityCritical {
		severity = "critical"
	}

	payload := map[string]interface{}{
		"routing_key":  n.config.PagerDutyRoutingKey,
		"event_action": "trigger",
		"dedup_key":    fmt.Sprintf("waas-sla-%s-%s", breach.TargetID, breach.BreachType),
		"payload": map[string]interface{}{
			"summary":   fmt.Sprintf("SLA Breach: %s - %s (expected %.2f, got %.2f)", target.Name, breach.BreachType, breach.ExpectedVal, breach.ActualVal),
			"severity":  severity,
			"source":    "waas-sla-monitor",
			"component": "webhook-delivery",
			"group":     breach.TenantID,
			"class":     breach.BreachType,
			"custom_details": map[string]interface{}{
				"target_id":     breach.TargetID,
				"endpoint_id":   breach.EndpointID,
				"expected_value": breach.ExpectedVal,
				"actual_value":  breach.ActualVal,
				"breach_time":   breach.CreatedAt.Format(time.RFC3339),
			},
		},
	}

	return n.postJSON(ctx, "https://events.pagerduty.com/v2/enqueue", payload)
}

func (n *Notifier) resolvePagerDuty(ctx context.Context, breach *Breach) error {
	if n.config.PagerDutyRoutingKey == "" {
		return nil
	}

	payload := map[string]interface{}{
		"routing_key":  n.config.PagerDutyRoutingKey,
		"event_action": "resolve",
		"dedup_key":    fmt.Sprintf("waas-sla-%s-%s", breach.TargetID, breach.BreachType),
	}

	return n.postJSON(ctx, "https://events.pagerduty.com/v2/enqueue", payload)
}

func (n *Notifier) notifyEmail(ctx context.Context, breach *Breach, target *Target) error {
	if n.config.EmailEndpoint == "" {
		return nil
	}

	payload := map[string]interface{}{
		"subject": fmt.Sprintf("[%s] SLA Breach: %s", breach.Severity, target.Name),
		"body": fmt.Sprintf(
			"SLA breach detected for %s.\n\nType: %s\nExpected: %.2f\nActual: %.2f\nEndpoint: %s\nTime: %s",
			target.Name, breach.BreachType, breach.ExpectedVal, breach.ActualVal,
			breach.EndpointID, breach.CreatedAt.Format(time.RFC3339),
		),
		"tenant_id": breach.TenantID,
	}

	return n.postJSON(ctx, n.config.EmailEndpoint, payload)
}

func (n *Notifier) notifyWebhook(ctx context.Context, breach *Breach, target *Target) error {
	if n.config.WebhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"event":    "sla.breach",
		"breach":   breach,
		"target":   target,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	return n.postJSON(ctx, n.config.WebhookURL, payload)
}

func (n *Notifier) postJSON(ctx context.Context, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification failed with status %d", resp.StatusCode)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// ForecastCompliance projects future SLA compliance based on current trends
type ComplianceForecast struct {
	TargetID             string    `json:"target_id"`
	ForecastPeriod       string    `json:"forecast_period"`
	ProjectedRate        float64   `json:"projected_delivery_rate_pct"`
	ProjectedP50Ms       int       `json:"projected_latency_p50_ms"`
	ProjectedP99Ms       int       `json:"projected_latency_p99_ms"`
	WillBreach           bool      `json:"will_breach"`
	ProjectedBreachTime  *time.Time `json:"projected_breach_time,omitempty"`
	Confidence           float64   `json:"confidence"`
	Recommendations      []string  `json:"recommendations,omitempty"`
	GeneratedAt          time.Time `json:"generated_at"`
}

// ForecastCompliance projects SLA compliance for the next period
func (s *Service) ForecastCompliance(ctx context.Context, tenantID, targetID string) (*ComplianceForecast, error) {
	target, err := s.repo.GetTarget(ctx, tenantID, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w", err)
	}

	status, err := s.GetComplianceStatus(ctx, tenantID, target)
	if err != nil {
		return nil, fmt.Errorf("failed to check compliance: %w", err)
	}

	breaches, err := s.repo.ListActiveBreaches(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get breaches: %w", err)
	}

	forecast := &ComplianceForecast{
		TargetID:        targetID,
		ForecastPeriod:  "24h",
		ProjectedRate:   status.CurrentRate,
		ProjectedP50Ms:  status.CurrentP50Ms,
		ProjectedP99Ms:  status.CurrentP99Ms,
		GeneratedAt:     time.Now(),
		Confidence:      0.7,
	}

	// Calculate trend from recent breaches
	recentBreachCount := 0
	for _, b := range breaches {
		if time.Since(b.CreatedAt) < 24*time.Hour {
			recentBreachCount++
		}
	}

	// Project based on trend
	if recentBreachCount > 2 {
		forecast.WillBreach = true
		forecast.Confidence = 0.85
		breachTime := time.Now().Add(time.Duration(24/recentBreachCount) * time.Hour)
		forecast.ProjectedBreachTime = &breachTime
		forecast.Recommendations = append(forecast.Recommendations,
			"Review endpoint health — multiple breaches detected in last 24h",
			"Consider adjusting SLA targets or investigating delivery failures",
		)
	} else if status.CurrentRate < target.DeliveryRatePct {
		forecast.WillBreach = true
		forecast.Confidence = 0.9
		forecast.Recommendations = append(forecast.Recommendations,
			"Currently below SLA target — immediate attention needed",
		)
	} else {
		margin := status.CurrentRate - target.DeliveryRatePct
		if margin < 5 {
			forecast.Recommendations = append(forecast.Recommendations,
				fmt.Sprintf("SLA margin is thin (%.1f%%) — monitor closely", margin),
			)
		}
	}

	return forecast, nil
}

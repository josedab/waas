package intelligence

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IncidentReport represents a comprehensive incident report for delivery failures
type IncidentReport struct {
	ID                string                  `json:"id"`
	TenantID          string                  `json:"tenant_id"`
	Title             string                  `json:"title"`
	Severity          IncidentSeverity        `json:"severity"`
	Status            IncidentStatus          `json:"status"`
	AffectedEndpoints []string                `json:"affected_endpoints"`
	FailureCategory   FailureCategory         `json:"failure_category"`
	RootCause         string                  `json:"root_cause"`
	Timeline          []IncidentTimelineEntry `json:"timeline"`
	Impact            *IncidentImpact         `json:"impact"`
	Remediation       *RemediationPlan        `json:"remediation"`
	Notifications     []NotificationRecord    `json:"notifications"`
	CreatedAt         time.Time               `json:"created_at"`
	ResolvedAt        *time.Time              `json:"resolved_at,omitempty"`
	TTRMinutes        int                     `json:"ttr_minutes,omitempty"`
}

// IncidentSeverity categorizes incident urgency
type IncidentSeverity string

const (
	SeverityCritical IncidentSeverity = "critical" // >50% failure rate or >10 endpoints
	SeverityHigh     IncidentSeverity = "high"     // >25% failure rate or >5 endpoints
	SeverityMedium   IncidentSeverity = "medium"   // >10% failure rate or >2 endpoints
	SeverityLow      IncidentSeverity = "low"      // <10% failure rate, 1 endpoint
)

// IncidentStatus tracks incident lifecycle
type IncidentStatus string

const (
	IncidentOpen          IncidentStatus = "open"
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentMitigating    IncidentStatus = "mitigating"
	IncidentResolved      IncidentStatus = "resolved"
)

// IncidentTimelineEntry represents a single event in the incident timeline
type IncidentTimelineEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Event       string    `json:"event"`
	Description string    `json:"description"`
	Actor       string    `json:"actor"` // system, user, auto
}

// IncidentImpact quantifies the incident's effect
type IncidentImpact struct {
	TotalFailures     int64   `json:"total_failures"`
	AffectedEndpoints int     `json:"affected_endpoints"`
	FailureRate       float64 `json:"failure_rate"`
	EstimatedRecovery string  `json:"estimated_recovery"`
	DataLossRisk      bool    `json:"data_loss_risk"`
}

// RemediationPlan provides steps to resolve the incident
type RemediationPlan struct {
	AutoRemediation   bool              `json:"auto_remediation"`
	Steps             []RemediationStep `json:"steps"`
	EstimatedTimeMins int               `json:"estimated_time_mins"`
}

// RemediationStep represents a single remediation action
type RemediationStep struct {
	Order       int    `json:"order"`
	Action      string `json:"action"`
	Description string `json:"description"`
	AutoApply   bool   `json:"auto_apply"`
	Status      string `json:"status"` // pending, applied, skipped
}

// NotificationRecord tracks a notification sent about the incident
type NotificationRecord struct {
	Channel   string    `json:"channel"` // slack, pagerduty, email, webhook
	Target    string    `json:"target"`  // channel name, email, etc.
	Status    string    `json:"status"`  // sent, failed, pending
	SentAt    time.Time `json:"sent_at"`
	MessageID string    `json:"message_id,omitempty"`
}

// --- Notification Integration Models ---

// SlackNotification represents a Slack message for incident alerts
type SlackNotification struct {
	Channel     string            `json:"channel"`
	Text        string            `json:"text"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type string      `json:"type"`
	Text interface{} `json:"text,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color  string `json:"color"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Footer string `json:"footer,omitempty"`
}

// PagerDutyEvent represents a PagerDuty incident event
type PagerDutyEvent struct {
	RoutingKey  string           `json:"routing_key"`
	EventAction string           `json:"event_action"` // trigger, acknowledge, resolve
	Payload     PagerDutyPayload `json:"payload"`
}

// PagerDutyPayload represents the PagerDuty event payload
type PagerDutyPayload struct {
	Summary   string `json:"summary"`
	Source    string `json:"source"`
	Severity  string `json:"severity"` // critical, error, warning, info
	Component string `json:"component,omitempty"`
	Group     string `json:"group,omitempty"`
	Class     string `json:"class,omitempty"`
}

// --- Incident Report Generation ---

// GenerateIncidentReport creates an incident report from failure analysis
func GenerateIncidentReport(tenantID string, analysis *RootCauseAnalysis, logs []DeliveryLog) *IncidentReport {
	now := time.Now()

	severity := classifyIncidentSeverity(analysis, logs)
	impact := computeIncidentImpact(analysis, logs)

	report := &IncidentReport{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Title:           generateIncidentTitle(analysis),
		Severity:        severity,
		Status:          IncidentOpen,
		FailureCategory: analysis.PrimaryCategory,
		RootCause:       formatRootCause(analysis),
		Impact:          impact,
		Remediation:     generateRemediationPlan(analysis),
		CreatedAt:       now,
	}

	// Collect affected endpoints
	endpointSet := make(map[string]bool)
	for _, log := range logs {
		if log.Status == "failed" {
			endpointSet[log.EndpointID] = true
		}
	}
	for ep := range endpointSet {
		report.AffectedEndpoints = append(report.AffectedEndpoints, ep)
	}

	// Build timeline
	report.Timeline = []IncidentTimelineEntry{
		{Timestamp: now, Event: "incident_created", Description: "Incident automatically created from failure analysis", Actor: "system"},
		{Timestamp: now, Event: "root_cause_identified", Description: fmt.Sprintf("Primary cause: %s (%.0f%% confidence)", analysis.PrimaryCategory, analysis.Confidence*100), Actor: "system"},
	}

	return report
}

func classifyIncidentSeverity(analysis *RootCauseAnalysis, logs []DeliveryLog) IncidentSeverity {
	endpointCount := countUniqueEndpoints(logs)
	failureRate := float64(analysis.FailureCount) / float64(max(len(logs), 1))

	switch {
	case failureRate > 0.5 || endpointCount > 10:
		return SeverityCritical
	case failureRate > 0.25 || endpointCount > 5:
		return SeverityHigh
	case failureRate > 0.10 || endpointCount > 2:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func countUniqueEndpoints(logs []DeliveryLog) int {
	eps := make(map[string]bool)
	for _, l := range logs {
		eps[l.EndpointID] = true
	}
	return len(eps)
}

func computeIncidentImpact(analysis *RootCauseAnalysis, logs []DeliveryLog) *IncidentImpact {
	endpointCount := countUniqueEndpoints(logs)
	failureRate := float64(analysis.FailureCount) / float64(max(len(logs), 1))

	recovery := "minutes"
	switch analysis.PrimaryCategory {
	case FailureCategoryDNS:
		recovery = "5-30 minutes (DNS propagation)"
	case FailureCategoryTLS:
		recovery = "manual intervention required"
	case FailureCategoryRateLimit:
		recovery = "auto-recovery with backoff"
	case FailureCategoryTimeout:
		recovery = "depends on endpoint recovery"
	case FailureCategory5xx:
		recovery = "depends on endpoint recovery"
	}

	return &IncidentImpact{
		TotalFailures:     int64(analysis.FailureCount),
		AffectedEndpoints: endpointCount,
		FailureRate:       failureRate,
		EstimatedRecovery: recovery,
		DataLossRisk:      failureRate > 0.5,
	}
}

func generateIncidentTitle(analysis *RootCauseAnalysis) string {
	return fmt.Sprintf("[%s] %d webhook delivery failures - %s",
		strings.ToUpper(string(analysis.PrimaryCategory)),
		analysis.FailureCount,
		analysis.RootCauses[0].Description,
	)
}

func formatRootCause(analysis *RootCauseAnalysis) string {
	if len(analysis.RootCauses) == 0 {
		return "Unknown root cause"
	}
	return analysis.RootCauses[0].Description
}

func generateRemediationPlan(analysis *RootCauseAnalysis) *RemediationPlan {
	plan := &RemediationPlan{}

	switch analysis.PrimaryCategory {
	case FailureCategoryTimeout:
		plan.Steps = []RemediationStep{
			{Order: 1, Action: "increase_timeout", Description: "Increase delivery timeout to 60s", AutoApply: true, Status: "pending"},
			{Order: 2, Action: "check_endpoint", Description: "Verify endpoint health and responsiveness", AutoApply: false, Status: "pending"},
			{Order: 3, Action: "adjust_retry", Description: "Increase retry backoff to avoid overwhelming endpoint", AutoApply: true, Status: "pending"},
		}
		plan.EstimatedTimeMins = 5
		plan.AutoRemediation = true

	case FailureCategoryRateLimit:
		plan.Steps = []RemediationStep{
			{Order: 1, Action: "reduce_rate", Description: "Reduce delivery rate by 50%", AutoApply: true, Status: "pending"},
			{Order: 2, Action: "enable_backoff", Description: "Enable exponential backoff for rate-limited endpoints", AutoApply: true, Status: "pending"},
			{Order: 3, Action: "batch_deliveries", Description: "Consider batching deliveries to reduce request count", AutoApply: false, Status: "pending"},
		}
		plan.EstimatedTimeMins = 10
		plan.AutoRemediation = true

	case FailureCategoryDNS:
		plan.Steps = []RemediationStep{
			{Order: 1, Action: "verify_dns", Description: "Verify endpoint DNS resolution", AutoApply: false, Status: "pending"},
			{Order: 2, Action: "pause_endpoint", Description: "Temporarily pause deliveries to affected endpoints", AutoApply: true, Status: "pending"},
			{Order: 3, Action: "notify_owner", Description: "Notify endpoint owner about DNS issue", AutoApply: false, Status: "pending"},
		}
		plan.EstimatedTimeMins = 30
		plan.AutoRemediation = false

	case FailureCategoryTLS:
		plan.Steps = []RemediationStep{
			{Order: 1, Action: "check_certificate", Description: "Verify endpoint SSL/TLS certificate validity", AutoApply: false, Status: "pending"},
			{Order: 2, Action: "pause_endpoint", Description: "Pause deliveries until certificate is fixed", AutoApply: true, Status: "pending"},
			{Order: 3, Action: "notify_owner", Description: "Notify endpoint owner about TLS certificate issue", AutoApply: false, Status: "pending"},
		}
		plan.EstimatedTimeMins = 60
		plan.AutoRemediation = false

	default:
		plan.Steps = []RemediationStep{
			{Order: 1, Action: "investigate", Description: "Investigate failure pattern and endpoint health", AutoApply: false, Status: "pending"},
			{Order: 2, Action: "retry_failed", Description: "Retry failed deliveries after investigation", AutoApply: false, Status: "pending"},
		}
		plan.EstimatedTimeMins = 30
		plan.AutoRemediation = false
	}

	return plan
}

// BuildSlackNotification creates a Slack message for an incident
func BuildSlackNotification(report *IncidentReport, channel string) *SlackNotification {
	color := "#36a64f" // green
	switch report.Severity {
	case SeverityCritical:
		color = "#ff0000"
	case SeverityHigh:
		color = "#ff6600"
	case SeverityMedium:
		color = "#ffcc00"
	}

	return &SlackNotification{
		Channel: channel,
		Text:    fmt.Sprintf("🚨 Incident: %s", report.Title),
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: report.Title,
				Text: fmt.Sprintf("Severity: %s\nAffected endpoints: %d\nFailure rate: %.1f%%\nRoot cause: %s\nEstimated recovery: %s",
					report.Severity,
					report.Impact.AffectedEndpoints,
					report.Impact.FailureRate*100,
					report.RootCause,
					report.Impact.EstimatedRecovery,
				),
				Footer: fmt.Sprintf("WaaS Incident %s • %s", report.ID[:8], report.CreatedAt.Format(time.RFC3339)),
			},
		},
	}
}

// BuildPagerDutyEvent creates a PagerDuty event for an incident
func BuildPagerDutyEvent(report *IncidentReport, routingKey string) *PagerDutyEvent {
	pdSeverity := "warning"
	switch report.Severity {
	case SeverityCritical:
		pdSeverity = "critical"
	case SeverityHigh:
		pdSeverity = "error"
	case SeverityMedium:
		pdSeverity = "warning"
	case SeverityLow:
		pdSeverity = "info"
	}

	return &PagerDutyEvent{
		RoutingKey:  routingKey,
		EventAction: "trigger",
		Payload: PagerDutyPayload{
			Summary:   report.Title,
			Source:    "waas-webhook-platform",
			Severity:  pdSeverity,
			Component: "delivery-engine",
			Group:     report.TenantID,
			Class:     string(report.FailureCategory),
		},
	}
}

package auth

import (
	"context"
	"fmt"
	"time"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"

	"github.com/google/uuid"
)

// NotificationService sends quota-related notifications to tenants
// (warnings, limit-reached, overage alerts).
type NotificationService struct {
	quotaRepo  repository.QuotaRepository
	tenantRepo repository.TenantRepository
}

// NotificationTemplate describes the subject, body, and priority of a
// notification type.
type NotificationTemplate struct {
	Type     string `json:"type"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Priority string `json:"priority"` // low, medium, high, critical
}

// NotificationContext carries the template variables used when rendering
// a notification message.
type NotificationContext struct {
	TenantName      string  `json:"tenant_name"`
	CurrentUsage    int     `json:"current_usage"`
	QuotaLimit      int     `json:"quota_limit"`
	UsagePercent    float64 `json:"usage_percent"`
	Remaining       int     `json:"remaining"`
	ResetDate       string  `json:"reset_date"`
	SubscriptionTier string `json:"subscription_tier"`
	OverageCount    int     `json:"overage_count,omitempty"`
	BurstAllowance  int     `json:"burst_allowance,omitempty"`
}

// NewNotificationService creates a NotificationService with the given repositories.
func NewNotificationService(quotaRepo repository.QuotaRepository, tenantRepo repository.TenantRepository) *NotificationService {
	return &NotificationService{
		quotaRepo:  quotaRepo,
		tenantRepo: tenantRepo,
	}
}

// ProcessPendingNotifications processes all pending notifications for all tenants
func (ns *NotificationService) ProcessPendingNotifications(ctx context.Context) error {
	// In a real implementation, you'd get all tenants with pending notifications
	// For now, we'll process a batch of tenants
	tenants, err := ns.tenantRepo.List(ctx, 100, 0)
	if err != nil {
		return fmt.Errorf("failed to get tenants: %w", err)
	}

	for _, tenant := range tenants {
		err := ns.ProcessTenantNotifications(ctx, tenant.ID)
		if err != nil {
			// Log error but continue processing other tenants
			continue
		}
	}

	return nil
}

// ProcessTenantNotifications processes pending notifications for a specific tenant
func (ns *NotificationService) ProcessTenantNotifications(ctx context.Context, tenantID uuid.UUID) error {
	notifications, err := ns.quotaRepo.GetPendingNotifications(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get pending notifications: %w", err)
	}

	if len(notifications) == 0 {
		return nil
	}

	tenant, err := ns.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	for _, notification := range notifications {
		err := ns.SendNotification(ctx, tenant, notification)
		if err != nil {
			// Log error but continue with other notifications
			continue
		}

		// Mark notification as sent
		err = ns.quotaRepo.MarkNotificationSent(ctx, notification.ID)
		if err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// SendNotification sends a notification to a tenant
func (ns *NotificationService) SendNotification(ctx context.Context, tenant *models.Tenant, notification *models.QuotaNotification) error {
	template := ns.getNotificationTemplate(notification.Type, notification.Threshold)
	context := ns.buildNotificationContext(tenant, notification)

	// In a real implementation, this would integrate with email service, SMS, webhook, etc.
	// For now, we'll just log the notification
	message := ns.renderNotification(template, context)
	
	// Simulate sending notification
	fmt.Printf("Sending notification to tenant %s (%s): %s\n", 
		tenant.Name, tenant.ID, message)

	return nil
}

// CreateQuotaWarningNotification creates a quota warning notification
func (ns *NotificationService) CreateQuotaWarningNotification(ctx context.Context, tenantID uuid.UUID, threshold int, currentUsage, quotaLimit int) error {
	// Check if notification already exists for this threshold this month
	now := time.Now().UTC()
	notifications, err := ns.quotaRepo.GetPendingNotifications(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check existing notifications: %w", err)
	}

	// Check if we already have a notification for this threshold
	for _, notification := range notifications {
		if notification.Threshold == threshold && 
		   notification.CreatedAt.Month() == now.Month() &&
		   notification.CreatedAt.Year() == now.Year() {
			return nil // Already exists
		}
	}

	notificationType := "warning"
	if threshold >= 100 {
		notificationType = "limit_reached"
	} else if threshold >= 90 {
		notificationType = "approaching_limit"
	}

	notification := &models.QuotaNotification{
		TenantID:    tenantID,
		Type:        notificationType,
		Threshold:   threshold,
		UsageCount:  currentUsage,
		QuotaLimit:  quotaLimit,
		Sent:        false,
	}

	return ns.quotaRepo.CreateQuotaNotification(ctx, notification)
}

// CreateOverageNotification creates an overage notification
func (ns *NotificationService) CreateOverageNotification(ctx context.Context, tenantID uuid.UUID, overageCount int, quotaLimit int) error {
	notification := &models.QuotaNotification{
		TenantID:    tenantID,
		Type:        "overage",
		Threshold:   100, // Over 100%
		UsageCount:  quotaLimit + overageCount,
		QuotaLimit:  quotaLimit,
		Sent:        false,
	}

	return ns.quotaRepo.CreateQuotaNotification(ctx, notification)
}

// getNotificationTemplate returns the appropriate template for a notification type
func (ns *NotificationService) getNotificationTemplate(notificationType string, threshold int) NotificationTemplate {
	templates := map[string]NotificationTemplate{
		"warning": {
			Type:     "warning",
			Subject:  "Webhook Usage Warning - {{.UsagePercent}}% of quota used",
			Body:     "Your webhook usage has reached {{.UsagePercent}}% of your monthly quota. Current usage: {{.CurrentUsage}}/{{.QuotaLimit}} requests. Quota resets on {{.ResetDate}}.",
			Priority: "medium",
		},
		"approaching_limit": {
			Type:     "approaching_limit",
			Subject:  "Webhook Quota Nearly Exhausted - {{.UsagePercent}}% used",
			Body:     "You're approaching your webhook quota limit. Current usage: {{.CurrentUsage}}/{{.QuotaLimit}} requests ({{.UsagePercent}}%). Consider upgrading your plan to avoid service interruption.",
			Priority: "high",
		},
		"limit_reached": {
			Type:     "limit_reached",
			Subject:  "Webhook Quota Limit Reached",
			Body:     "You have reached your monthly webhook quota of {{.QuotaLimit}} requests. Additional requests may be subject to overage charges or service limitations.",
			Priority: "critical",
		},
		"overage": {
			Type:     "overage",
			Subject:  "Webhook Overage Usage - Additional Charges Apply",
			Body:     "Your webhook usage has exceeded your monthly quota. Current usage: {{.CurrentUsage}} requests ({{.OverageCount}} over quota). Overage charges will apply to your next bill.",
			Priority: "critical",
		},
		"subscription_change": {
			Type:     "subscription_change",
			Subject:  "Webhook Subscription Updated",
			Body:     "Your webhook subscription has been updated. New quota limits are now in effect.",
			Priority: "low",
		},
	}

	template, exists := templates[notificationType]
	if !exists {
		return NotificationTemplate{
			Type:     "generic",
			Subject:  "Webhook Service Notification",
			Body:     "You have a notification regarding your webhook service usage.",
			Priority: "low",
		}
	}

	return template
}

// buildNotificationContext builds the context data for notification templates
func (ns *NotificationService) buildNotificationContext(tenant *models.Tenant, notification *models.QuotaNotification) NotificationContext {
	usagePercent := float64(0)
	remaining := notification.QuotaLimit
	
	if notification.QuotaLimit > 0 {
		usagePercent = float64(notification.UsageCount) / float64(notification.QuotaLimit) * 100
		remaining = max(0, notification.QuotaLimit-notification.UsageCount)
	}

	// Calculate next reset date (first day of next month)
	now := time.Now().UTC()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	tierConfig, _ := models.GetTierConfig(tenant.SubscriptionTier)

	return NotificationContext{
		TenantName:       tenant.Name,
		CurrentUsage:     notification.UsageCount,
		QuotaLimit:       notification.QuotaLimit,
		UsagePercent:     usagePercent,
		Remaining:        remaining,
		ResetDate:        nextMonth.Format("January 2, 2006"),
		SubscriptionTier: tenant.SubscriptionTier,
		BurstAllowance:   tierConfig.BurstAllowance,
	}
}

// renderNotification renders a notification template with context data
func (ns *NotificationService) renderNotification(template NotificationTemplate, context NotificationContext) string {
	// In a real implementation, you'd use a proper template engine
	// This is a simplified version for demonstration
	message := fmt.Sprintf("Subject: %s\n\nDear %s,\n\n%s\n\nCurrent Usage: %d/%d requests (%.1f%%)\nRemaining: %d requests\nQuota Reset: %s\nSubscription: %s",
		template.Subject,
		context.TenantName,
		template.Body,
		context.CurrentUsage,
		context.QuotaLimit,
		context.UsagePercent,
		context.Remaining,
		context.ResetDate,
		context.SubscriptionTier,
	)

	return message
}

// GetNotificationHistory returns notification history for a tenant
func (ns *NotificationService) GetNotificationHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.QuotaNotification, error) {
	return ns.quotaRepo.GetNotificationHistory(ctx, tenantID, limit, offset)
}

// ScheduleQuotaCheck schedules a quota check for all tenants (typically run hourly)
func (ns *NotificationService) ScheduleQuotaCheck(ctx context.Context) error {
	tenants, err := ns.tenantRepo.List(ctx, 1000, 0) // In production, paginate this
	if err != nil {
		return fmt.Errorf("failed to get tenants: %w", err)
	}

	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for _, tenant := range tenants {
		usage, err := ns.quotaRepo.GetQuotaUsageByTenant(ctx, tenant.ID, currentMonth)
		if err != nil {
			continue // Skip if no usage data
		}

		quota := tenant.MonthlyQuota
		thresholds := []int{80, 90, 100}

		for _, threshold := range thresholds {
			if usage.ShouldNotify(quota, threshold) {
				err := ns.CreateQuotaWarningNotification(ctx, tenant.ID, threshold, usage.RequestCount, quota)
				if err != nil {
					// Log error but continue
					continue
				}
			}
		}

		// Check for overage
		if usage.OverageCount > 0 {
			err := ns.CreateOverageNotification(ctx, tenant.ID, usage.OverageCount, quota)
			if err != nil {
				// Log error but continue
				continue
			}
		}
	}

	return nil
}
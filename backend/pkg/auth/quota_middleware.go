package auth

import (
	"context"
	stdlog "log"
	"net/http"
	"strconv"
	"time"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// QuotaMiddleware enforces monthly webhook delivery quotas per tenant,
// backed by Redis for fast counter lookups.
type QuotaMiddleware struct {
	quotaRepo   repository.QuotaRepository
	redisClient *redis.Client
}

// QuotaInfo describes a tenant's current quota status, including remaining
// allowance and overage details.
type QuotaInfo struct {
	Allowed       bool
	CurrentUsage  int
	MonthlyQuota  int
	Remaining     int
	UsagePercent  float64
	ResetDate     time.Time
	IsOverage     bool
	OverageCount  int
}

// NewQuotaMiddleware creates a QuotaMiddleware with the given quota repository and Redis client.
func NewQuotaMiddleware(quotaRepo repository.QuotaRepository, redisClient *redis.Client) *QuotaMiddleware {
	return &QuotaMiddleware{
		quotaRepo:   quotaRepo,
		redisClient: redisClient,
	}
}

// EnforceQuota middleware checks and enforces monthly quota limits
func (qm *QuotaMiddleware) EnforceQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant, exists := GetTenantFromContext(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "MISSING_TENANT_CONTEXT",
					"message": "Tenant context not found",
				},
			})
			c.Abort()
			return
		}

		quotaInfo, err := qm.CheckQuota(c.Request.Context(), tenant)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "QUOTA_CHECK_ERROR",
					"message": "Failed to check quota limits",
				},
			})
			c.Abort()
			return
		}

		// Set quota headers
		c.Header("X-Quota-Limit", strconv.Itoa(quotaInfo.MonthlyQuota))
		c.Header("X-Quota-Remaining", strconv.Itoa(quotaInfo.Remaining))
		c.Header("X-Quota-Used", strconv.Itoa(quotaInfo.CurrentUsage))
		c.Header("X-Quota-Reset", strconv.FormatInt(quotaInfo.ResetDate.Unix(), 10))

		if !quotaInfo.Allowed {
			// Check if burst allowance is available
			tierConfig, exists := models.GetTierConfig(tenant.SubscriptionTier)
			if exists && quotaInfo.OverageCount < tierConfig.BurstAllowance {
				// Allow request but mark as overage
				c.Set("quota_overage", true)
				c.Header("X-Quota-Overage", "true")
				c.Header("X-Quota-Overage-Remaining", strconv.Itoa(tierConfig.BurstAllowance-quotaInfo.OverageCount))
			} else {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": gin.H{
						"code":    "QUOTA_EXCEEDED",
						"message": "Monthly quota limit exceeded",
						"details": gin.H{
							"current_usage":   quotaInfo.CurrentUsage,
							"monthly_quota":   quotaInfo.MonthlyQuota,
							"overage_count":   quotaInfo.OverageCount,
							"reset_date":      quotaInfo.ResetDate.Unix(),
							"usage_percent":   quotaInfo.UsagePercent,
						},
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// CheckQuota checks current quota status for a tenant
func (qm *QuotaMiddleware) CheckQuota(ctx context.Context, tenant *models.Tenant) (*QuotaInfo, error) {
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := currentMonth.AddDate(0, 1, 0)

	// Get or create quota usage for current month
	usage, err := qm.quotaRepo.GetOrCreateQuotaUsage(ctx, tenant.ID, currentMonth)
	if err != nil {
		return nil, err
	}

	quota := tenant.MonthlyQuota
	remaining := maxInt(0, quota-usage.RequestCount)
	usagePercent := usage.GetUsagePercentage(quota)
	allowed := usage.RequestCount < quota
	isOverage := usage.RequestCount >= quota

	return &QuotaInfo{
		Allowed:      allowed,
		CurrentUsage: usage.RequestCount,
		MonthlyQuota: quota,
		Remaining:    remaining,
		UsagePercent: usagePercent,
		ResetDate:    nextMonth,
		IsOverage:    isOverage,
		OverageCount: usage.OverageCount,
	}, nil
}

// TrackUsage increments usage counters after a successful request
func (qm *QuotaMiddleware) TrackUsage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first

		// Only track if request was successful (not aborted)
		if c.IsAborted() {
			return
		}

		tenant, exists := GetTenantFromContext(c)
		if !exists {
			return
		}

		// Determine if this was a successful request based on status code
		success := c.Writer.Status() >= 200 && c.Writer.Status() < 400

		// Track usage asynchronously to avoid blocking the response
		reqCtx := c.Request.Context()
		go func() {
			ctx, cancel := context.WithTimeout(reqCtx, 5*time.Second)
			defer cancel()

			err := qm.quotaRepo.IncrementUsage(ctx, tenant.ID, success)
			if err != nil {
				stdlog.Printf("quota: failed to increment usage for tenant %s: %v", tenant.ID, err)
				return
			}

			// Check if we need to send quota notifications
			qm.checkAndSendNotifications(ctx, tenant)

			// Update overage count if this was an overage request
			if overage, exists := c.Get("quota_overage"); exists && overage.(bool) {
				qm.incrementOverageCount(ctx, tenant.ID)
			}
		}()
	}
}

// checkAndSendNotifications checks if quota notifications should be sent
func (qm *QuotaMiddleware) checkAndSendNotifications(ctx context.Context, tenant *models.Tenant) {
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	usage, err := qm.quotaRepo.GetQuotaUsageByTenant(ctx, tenant.ID, currentMonth)
	if err != nil {
		stdlog.Printf("quota: failed to get usage for tenant %s: %v", tenant.ID, err)
		return
	}

	quota := tenant.MonthlyQuota
	thresholds := []int{80, 90, 100} // 80%, 90%, 100% thresholds

	for _, threshold := range thresholds {
		if usage.ShouldNotify(quota, threshold) {
			// Check if notification already sent for this threshold this month
			notifications, err := qm.quotaRepo.GetPendingNotifications(ctx, tenant.ID)
			if err != nil {
				stdlog.Printf("quota: failed to get pending notifications for tenant %s: %v", tenant.ID, err)
				continue
			}

			// Check if we already have a notification for this threshold
			alreadyNotified := false
			for _, notification := range notifications {
				if notification.Threshold == threshold {
					alreadyNotified = true
					break
				}
			}

			if !alreadyNotified {
				notificationType := "warning"
				if threshold >= 100 {
					notificationType = "limit_reached"
				}

				notification := &models.QuotaNotification{
					TenantID:    tenant.ID,
					Type:        notificationType,
					Threshold:   threshold,
					UsageCount:  usage.RequestCount,
					QuotaLimit:  quota,
					Sent:        false,
				}

				if err := qm.quotaRepo.CreateQuotaNotification(ctx, notification); err != nil {
					stdlog.Printf("quota: failed to create notification for tenant %s at %d%%: %v", tenant.ID, threshold, err)
				}
			}
		}
	}
}

// incrementOverageCount increments the overage counter for the current month
func (qm *QuotaMiddleware) incrementOverageCount(ctx context.Context, tenantID uuid.UUID) {
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	usage, err := qm.quotaRepo.GetQuotaUsageByTenant(ctx, tenantID, currentMonth)
	if err != nil {
		stdlog.Printf("quota: failed to get usage for overage count, tenant %s: %v", tenantID, err)
		return
	}

	usage.OverageCount++
	if err := qm.quotaRepo.UpdateQuotaUsage(ctx, usage); err != nil {
		stdlog.Printf("quota: failed to update overage count for tenant %s: %v", tenantID, err)
	}
}

// GetQuotaStatus returns current quota status for a tenant (for API endpoints)
func (qm *QuotaMiddleware) GetQuotaStatus(ctx context.Context, tenant *models.Tenant) (*QuotaInfo, error) {
	return qm.CheckQuota(ctx, tenant)
}
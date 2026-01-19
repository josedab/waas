package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionTier represents different subscription levels
type SubscriptionTier struct {
	Name                string `json:"name" db:"name"`
	MonthlyQuota        int    `json:"monthly_quota" db:"monthly_quota"`
	RateLimitPerMinute  int    `json:"rate_limit_per_minute" db:"rate_limit_per_minute"`
	MaxEndpoints        int    `json:"max_endpoints" db:"max_endpoints"`
	PricePerRequest     int    `json:"price_per_request" db:"price_per_request"` // in cents
	OverageRate         int    `json:"overage_rate" db:"overage_rate"`           // in cents per request
	BurstAllowance      int    `json:"burst_allowance" db:"burst_allowance"`     // temporary overage allowed
}

// QuotaUsage tracks current usage for a tenant
type QuotaUsage struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Month           time.Time `json:"month" db:"month"`                         // First day of the month
	RequestCount    int       `json:"request_count" db:"request_count"`         // Total requests this month
	SuccessCount    int       `json:"success_count" db:"success_count"`         // Successful deliveries
	FailureCount    int       `json:"failure_count" db:"failure_count"`         // Failed deliveries
	OverageCount    int       `json:"overage_count" db:"overage_count"`         // Requests over quota
	LastUpdated     time.Time `json:"last_updated" db:"last_updated"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// BillingRecord represents a billing calculation for a tenant
type BillingRecord struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	BillingPeriod   time.Time `json:"billing_period" db:"billing_period"`       // First day of billing month
	BaseRequests    int       `json:"base_requests" db:"base_requests"`         // Requests within quota
	OverageRequests int       `json:"overage_requests" db:"overage_requests"`   // Requests over quota
	BaseAmount      int       `json:"base_amount" db:"base_amount"`             // Base subscription cost in cents
	OverageAmount   int       `json:"overage_amount" db:"overage_amount"`       // Overage charges in cents
	TotalAmount     int       `json:"total_amount" db:"total_amount"`           // Total bill in cents
	Status          string    `json:"status" db:"status"`                       // pending, processed, paid
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// QuotaNotification represents notifications sent to users about quota usage
type QuotaNotification struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Type         string    `json:"type" db:"type"`                   // warning, limit_reached, overage
	Threshold    int       `json:"threshold" db:"threshold"`         // Percentage threshold (e.g., 80, 90, 100)
	UsageCount   int       `json:"usage_count" db:"usage_count"`     // Current usage when notification sent
	QuotaLimit   int       `json:"quota_limit" db:"quota_limit"`     // Quota limit at time of notification
	Sent         bool      `json:"sent" db:"sent"`                   // Whether notification was sent
	SentAt       *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// GetSubscriptionTiers returns predefined subscription tiers
func GetSubscriptionTiers() map[string]SubscriptionTier {
	return map[string]SubscriptionTier{
		"free": {
			Name:               "free",
			MonthlyQuota:       1000,
			RateLimitPerMinute: 10,
			MaxEndpoints:       5,
			PricePerRequest:    0,
			OverageRate:        1, // 1 cent per request
			BurstAllowance:     100,
		},
		"starter": {
			Name:               "starter",
			MonthlyQuota:       10000,
			RateLimitPerMinute: 100,
			MaxEndpoints:       25,
			PricePerRequest:    0,
			OverageRate:        1,
			BurstAllowance:     1000,
		},
		"professional": {
			Name:               "professional",
			MonthlyQuota:       100000,
			RateLimitPerMinute: 500,
			MaxEndpoints:       100,
			PricePerRequest:    0,
			OverageRate:        1,
			BurstAllowance:     10000,
		},
		"enterprise": {
			Name:               "enterprise",
			MonthlyQuota:       1000000,
			RateLimitPerMinute: 2000,
			MaxEndpoints:       -1, // unlimited
			PricePerRequest:    0,
			OverageRate:        1,
			BurstAllowance:     100000,
		},
	}
}

// GetTierConfig returns the subscription tier configuration for a given tier name
func GetTierConfig(tierName string) (SubscriptionTier, bool) {
	tiers := GetSubscriptionTiers()
	tier, exists := tiers[tierName]
	return tier, exists
}

// IsOverQuota checks if current usage exceeds the monthly quota
func (qu *QuotaUsage) IsOverQuota(quota int) bool {
	return qu.RequestCount > quota
}

// GetUsagePercentage returns the percentage of quota used
func (qu *QuotaUsage) GetUsagePercentage(quota int) float64 {
	if quota <= 0 {
		return 0
	}
	return float64(qu.RequestCount) / float64(quota) * 100
}

// ShouldNotify checks if a notification should be sent for the given threshold
func (qu *QuotaUsage) ShouldNotify(quota int, threshold int) bool {
	percentage := qu.GetUsagePercentage(quota)
	return percentage >= float64(threshold)
}
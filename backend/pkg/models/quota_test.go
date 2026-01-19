package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestQuotaUsage_GetUsagePercentage(t *testing.T) {
	usage := &QuotaUsage{
		RequestCount: 750,
	}

	percentage := usage.GetUsagePercentage(1000)
	assert.Equal(t, 75.0, percentage)

	// Test edge case with zero quota
	percentage = usage.GetUsagePercentage(0)
	assert.Equal(t, 0.0, percentage)
}

func TestQuotaUsage_IsOverQuota(t *testing.T) {
	usage := &QuotaUsage{
		RequestCount: 1200,
	}

	assert.True(t, usage.IsOverQuota(1000))
	assert.False(t, usage.IsOverQuota(1500))
}

func TestQuotaUsage_ShouldNotify(t *testing.T) {
	usage := &QuotaUsage{
		RequestCount: 850,
	}

	quota := 1000

	// Should notify at 80% threshold
	assert.True(t, usage.ShouldNotify(quota, 80))
	assert.True(t, usage.ShouldNotify(quota, 85))

	// Should not notify at 90% threshold (usage is 85%)
	assert.False(t, usage.ShouldNotify(quota, 90))
}

func TestGetSubscriptionTiers(t *testing.T) {
	tiers := GetSubscriptionTiers()

	assert.NotEmpty(t, tiers)
	assert.Contains(t, tiers, "free")
	assert.Contains(t, tiers, "starter")
	assert.Contains(t, tiers, "professional")
	assert.Contains(t, tiers, "enterprise")

	// Test free tier configuration
	freeTier := tiers["free"]
	assert.Equal(t, "free", freeTier.Name)
	assert.Equal(t, 1000, freeTier.MonthlyQuota)
	assert.Equal(t, 10, freeTier.RateLimitPerMinute)
	assert.Equal(t, 5, freeTier.MaxEndpoints)
	assert.Equal(t, 100, freeTier.BurstAllowance)

	// Test enterprise tier configuration
	enterpriseTier := tiers["enterprise"]
	assert.Equal(t, "enterprise", enterpriseTier.Name)
	assert.Equal(t, 1000000, enterpriseTier.MonthlyQuota)
	assert.Equal(t, 2000, enterpriseTier.RateLimitPerMinute)
	assert.Equal(t, -1, enterpriseTier.MaxEndpoints) // Unlimited
	assert.Equal(t, 100000, enterpriseTier.BurstAllowance)
}

func TestGetTierConfig(t *testing.T) {
	// Test valid tier
	tier, exists := GetTierConfig("starter")
	assert.True(t, exists)
	assert.Equal(t, "starter", tier.Name)
	assert.Equal(t, 10000, tier.MonthlyQuota)

	// Test invalid tier
	_, exists = GetTierConfig("invalid")
	assert.False(t, exists)
}

func TestBillingRecord_Validation(t *testing.T) {
	record := &BillingRecord{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		BillingPeriod:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseRequests:    5000,
		OverageRequests: 1000,
		BaseAmount:      0,
		OverageAmount:   1000,
		TotalAmount:     1000,
		Status:          "pending",
	}

	assert.Equal(t, 5000, record.BaseRequests)
	assert.Equal(t, 1000, record.OverageRequests)
	assert.Equal(t, 1000, record.TotalAmount)
	assert.Equal(t, "pending", record.Status)
}

func TestQuotaNotification_Types(t *testing.T) {
	notification := &QuotaNotification{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Type:       "warning",
		Threshold:  80,
		UsageCount: 800,
		QuotaLimit: 1000,
		Sent:       false,
	}

	assert.Equal(t, "warning", notification.Type)
	assert.Equal(t, 80, notification.Threshold)
	assert.False(t, notification.Sent)
	assert.Nil(t, notification.SentAt)
}
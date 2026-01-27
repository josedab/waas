package models

import (
	"time"
	"github.com/google/uuid"
)

// Tenant represents an API consumer account with its subscription tier,
// rate-limit settings, and monthly quota.
type Tenant struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	APIKeyHash         string    `json:"-" db:"api_key_hash"`
	SubscriptionTier   string    `json:"subscription_tier" db:"subscription_tier"`
	RateLimitPerMinute int       `json:"rate_limit_per_minute" db:"rate_limit_per_minute"`
	MonthlyQuota       int       `json:"monthly_quota" db:"monthly_quota"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
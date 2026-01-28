package cloud

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPlanNotFound         = errors.New("plan not found")
	ErrUsageLimitExceeded   = errors.New("usage limit exceeded")
	ErrInvalidPlanUpgrade   = errors.New("invalid plan upgrade")
	ErrPaymentRequired      = errors.New("payment method required")
)

// BillingService manages billing operations
type BillingService struct {
	repo         Repository
	stripeClient StripeClient
	usageTracker *UsageTracker
}

// StripeClient interface for Stripe operations (implement with actual Stripe SDK)
type StripeClient interface {
	CreateCustomer(ctx context.Context, email, name string) (string, error)
	CreateSubscription(ctx context.Context, customerID, priceID string) (string, error)
	CancelSubscription(ctx context.Context, subscriptionID string, atPeriodEnd bool) error
	UpdateSubscription(ctx context.Context, subscriptionID, newPriceID string) error
	CreatePaymentIntent(ctx context.Context, customerID string, amount int64, currency string) (string, error)
	AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error
}

// NewBillingService creates a new billing service
func NewBillingService(repo Repository, stripeClient StripeClient) *BillingService {
	return &BillingService{
		repo:         repo,
		stripeClient: stripeClient,
		usageTracker: NewUsageTracker(repo),
	}
}

// CreateSubscription creates a new subscription
func (s *BillingService) CreateSubscription(ctx context.Context, tenantID, planID string, cycle BillingCycle) (*Subscription, error) {
	plan := GetPlanByID(planID)
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	// Check if already has active subscription
	existing, err := s.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err == nil && existing.Status == SubscriptionStatusActive {
		return nil, fmt.Errorf("tenant already has active subscription")
	}

	now := time.Now()
	var periodEnd time.Time
	if cycle == BillingCycleMonthly {
		periodEnd = now.AddDate(0, 1, 0)
	} else {
		periodEnd = now.AddDate(1, 0, 0)
	}

	sub := &Subscription{
		ID:                 generateID(),
		TenantID:           tenantID,
		PlanID:             planID,
		Status:             SubscriptionStatusActive,
		BillingCycle:       cycle,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Handle trial for paid plans
	if plan.TrialDays > 0 && !plan.IsFree {
		sub.Status = SubscriptionStatusTrialing
		trialEnd := now.AddDate(0, 0, plan.TrialDays)
		sub.TrialEnd = &trialEnd
	}

	// Create in Stripe if paid plan
	if !plan.IsFree && s.stripeClient != nil {
		customer, err := s.repo.GetCustomer(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("customer not found: %w", err)
		}

		stripeSub, err := s.stripeClient.CreateSubscription(ctx, customer.StripeCustomerID, plan.StripePriceID)
		if err != nil {
			return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
		}
		sub.StripeSubscriptionID = stripeSub
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// ChangePlan changes the subscription plan
func (s *BillingService) ChangePlan(ctx context.Context, tenantID, newPlanID string) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	newPlan := GetPlanByID(newPlanID)
	if newPlan == nil {
		return nil, ErrPlanNotFound
	}

	// Validate plan change
	currentPlan := GetPlanByID(sub.PlanID)
	if currentPlan != nil && newPlan.PriceMonthly < currentPlan.PriceMonthly {
		// Downgrade - check usage fits new limits
		usage, _ := s.repo.GetUsage(ctx, tenantID, time.Now().Format("2006-01"))
		if usage != nil && newPlan.Limits.MonthlyWebhooks > 0 && usage.WebhooksSent > newPlan.Limits.MonthlyWebhooks {
			return nil, ErrInvalidPlanUpgrade
		}
	}

	sub.PlanID = newPlanID
	sub.UpdatedAt = time.Now()

	// Update in Stripe
	if sub.StripeSubscriptionID != "" && s.stripeClient != nil {
		if err := s.stripeClient.UpdateSubscription(ctx, sub.StripeSubscriptionID, newPlan.StripePriceID); err != nil {
			return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
		}
	}

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// CancelSubscription cancels a subscription
func (s *BillingService) CancelSubscription(ctx context.Context, tenantID string, immediate bool) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	if immediate {
		now := time.Now()
		sub.Status = SubscriptionStatusCanceled
		sub.CanceledAt = &now
	} else {
		sub.CancelAtPeriodEnd = true
	}
	sub.UpdatedAt = time.Now()

	// Cancel in Stripe
	if sub.StripeSubscriptionID != "" && s.stripeClient != nil {
		if err := s.stripeClient.CancelSubscription(ctx, sub.StripeSubscriptionID, !immediate); err != nil {
			return nil, fmt.Errorf("failed to cancel Stripe subscription: %w", err)
		}
	}

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// GetSubscription retrieves the current subscription
func (s *BillingService) GetSubscription(ctx context.Context, tenantID string) (*Subscription, error) {
	return s.repo.GetSubscriptionByTenant(ctx, tenantID)
}

// GetPlanLimits returns the current plan limits for a tenant
func (s *BillingService) GetPlanLimits(ctx context.Context, tenantID string) (*PlanLimits, error) {
	sub, err := s.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err != nil {
		// Default to free plan limits
		return &FreePlan.Limits, nil
	}

	plan := GetPlanByID(sub.PlanID)
	if plan == nil {
		return &FreePlan.Limits, nil
	}

	return &plan.Limits, nil
}

// UsageTracker tracks and enforces usage limits
type UsageTracker struct {
	repo Repository
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker(repo Repository) *UsageTracker {
	return &UsageTracker{repo: repo}
}

// TrackWebhookSent tracks a sent webhook
func (t *UsageTracker) TrackWebhookSent(ctx context.Context, tenantID string, bytes int64) error {
	if err := t.repo.IncrementUsage(ctx, tenantID, "webhooks_sent", 1); err != nil {
		return err
	}
	return t.repo.IncrementUsage(ctx, tenantID, "total_bytes", bytes)
}

// TrackWebhookReceived tracks a received webhook
func (t *UsageTracker) TrackWebhookReceived(ctx context.Context, tenantID string, bytes int64) error {
	if err := t.repo.IncrementUsage(ctx, tenantID, "webhooks_received", 1); err != nil {
		return err
	}
	return t.repo.IncrementUsage(ctx, tenantID, "total_bytes", bytes)
}

// TrackAPIRequest tracks an API request
func (t *UsageTracker) TrackAPIRequest(ctx context.Context, tenantID string) error {
	return t.repo.IncrementUsage(ctx, tenantID, "api_requests", 1)
}

// TrackTransform tracks a transformation execution
func (t *UsageTracker) TrackTransform(ctx context.Context, tenantID string) error {
	return t.repo.IncrementUsage(ctx, tenantID, "transform_executions", 1)
}

// CheckLimit checks if usage is within limits
func (t *UsageTracker) CheckLimit(ctx context.Context, tenantID string, limits *PlanLimits) error {
	period := time.Now().Format("2006-01")
	usage, err := t.repo.GetUsage(ctx, tenantID, period)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No usage yet
		}
		return err
	}

	if limits.MonthlyWebhooks > 0 && usage.WebhooksSent >= limits.MonthlyWebhooks {
		return ErrUsageLimitExceeded
	}

	return nil
}

// GetCurrentUsage returns current period usage
func (t *UsageTracker) GetCurrentUsage(ctx context.Context, tenantID string) (*UsageRecord, error) {
	period := time.Now().Format("2006-01")
	return t.repo.GetUsage(ctx, tenantID, period)
}

// TeamService manages team members
type TeamService struct {
	repo Repository
}

// NewTeamService creates a new team service
func NewTeamService(repo Repository) *TeamService {
	return &TeamService{repo: repo}
}

// InviteMember invites a new team member
func (ts *TeamService) InviteMember(ctx context.Context, tenantID, inviterID, email, name string, role TeamRole) (*TeamMember, error) {
	member := &TeamMember{
		ID:        generateID(),
		TenantID:  tenantID,
		Email:     email,
		Name:      name,
		Role:      role,
		Status:    MemberStatusPending,
		InvitedBy: inviterID,
		InvitedAt: time.Now(),
	}

	if err := ts.repo.CreateTeamMember(ctx, member); err != nil {
		return nil, err
	}

	// TODO(#6): Send invitation email — https://github.com/josedab/waas/issues/6

	return member, nil
}

// AcceptInvitation accepts a team invitation
func (ts *TeamService) AcceptInvitation(ctx context.Context, memberID, userID string) error {
	member, err := ts.repo.GetTeamMember(ctx, memberID)
	if err != nil {
		return err
	}

	now := time.Now()
	member.UserID = userID
	member.Status = MemberStatusActive
	member.JoinedAt = &now

	return ts.repo.UpdateTeamMember(ctx, member)
}

// UpdateMemberRole updates a team member's role
func (ts *TeamService) UpdateMemberRole(ctx context.Context, memberID string, role TeamRole) error {
	member, err := ts.repo.GetTeamMember(ctx, memberID)
	if err != nil {
		return err
	}

	member.Role = role
	return ts.repo.UpdateTeamMember(ctx, member)
}

// RemoveMember removes a team member
func (ts *TeamService) RemoveMember(ctx context.Context, memberID string) error {
	return ts.repo.DeleteTeamMember(ctx, memberID)
}

// ListMembers lists team members
func (ts *TeamService) ListMembers(ctx context.Context, tenantID string) ([]*TeamMember, error) {
	return ts.repo.ListTeamMembers(ctx, tenantID)
}

// AuditService manages audit logging
type AuditService struct {
	repo Repository
}

// NewAuditService creates a new audit service
func NewAuditService(repo Repository) *AuditService {
	return &AuditService{repo: repo}
}

// Log creates an audit log entry
func (as *AuditService) Log(ctx context.Context, tenantID, userID, action, resource, resourceID, ipAddress, userAgent string, details map[string]interface{}) error {
	log := &AuditLog{
		ID:         generateID(),
		TenantID:   tenantID,
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Details:    details,
		CreatedAt:  time.Now(),
	}

	return as.repo.CreateAuditLog(ctx, log)
}

// ListLogs lists audit logs with pagination
func (as *AuditService) ListLogs(ctx context.Context, tenantID string, page, pageSize int) ([]*AuditLog, error) {
	offset := (page - 1) * pageSize
	return as.repo.ListAuditLogs(ctx, tenantID, pageSize, offset)
}

// Common audit actions
const (
	AuditActionCreate        = "create"
	AuditActionUpdate        = "update"
	AuditActionDelete        = "delete"
	AuditActionLogin         = "login"
	AuditActionLogout        = "logout"
	AuditActionAPIKeyCreated = "api_key_created"
	AuditActionAPIKeyRevoked = "api_key_revoked"
	AuditActionPlanChanged   = "plan_changed"
	AuditActionMemberInvited = "member_invited"
	AuditActionMemberRemoved = "member_removed"
)

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

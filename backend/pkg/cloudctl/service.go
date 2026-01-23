package cloudctl

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides cloud control plane functionality
type Service struct {
	repo Repository
}

// NewService creates a new cloud control plane service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ProvisionTenant provisions a new cloud tenant
func (s *Service) ProvisionTenant(ctx context.Context, req *ProvisionRequest) (*CloudTenant, error) {
	if !isValidRegion(req.Region) {
		return nil, fmt.Errorf("invalid region: %s", req.Region)
	}

	quota, ok := PlanQuotas[req.Plan]
	if !ok {
		return nil, fmt.Errorf("invalid plan: %s", req.Plan)
	}

	now := time.Now()
	tenant := &CloudTenant{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Email:         req.Email,
		Plan:          req.Plan,
		Region:        req.Region,
		Status:        StatusProvisioning,
		Namespace:     fmt.Sprintf("waas-%s", uuid.New().String()[:8]),
		ResourceQuota: &quota,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Mark as active (in production, this would trigger k8s namespace creation)
	tenant.Status = StatusActive
	provisioned := time.Now()
	tenant.ProvisionedAt = &provisioned
	tenant.UpdatedAt = time.Now()

	if err := s.repo.UpdateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to activate tenant: %w", err)
	}

	return tenant, nil
}

// GetTenant retrieves a cloud tenant
func (s *Service) GetTenant(ctx context.Context, tenantID string) (*CloudTenant, error) {
	return s.repo.GetTenant(ctx, tenantID)
}

// ListTenants lists all cloud tenants
func (s *Service) ListTenants(ctx context.Context, status string, limit, offset int) ([]CloudTenant, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListTenants(ctx, status, limit, offset)
}

// UpdatePlan changes a tenant's subscription plan
func (s *Service) UpdatePlan(ctx context.Context, tenantID string, req *UpdatePlanRequest) (*CloudTenant, error) {
	tenant, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	quota, ok := PlanQuotas[req.Plan]
	if !ok {
		return nil, fmt.Errorf("invalid plan: %s", req.Plan)
	}

	tenant.Plan = req.Plan
	tenant.ResourceQuota = &quota
	tenant.UpdatedAt = time.Now()

	if err := s.repo.UpdateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	return tenant, nil
}

// SuspendTenant suspends a cloud tenant
func (s *Service) SuspendTenant(ctx context.Context, tenantID string) error {
	tenant, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	tenant.Status = StatusSuspended
	tenant.UpdatedAt = time.Now()
	return s.repo.UpdateTenant(ctx, tenant)
}

// ReactivateTenant reactivates a suspended tenant
func (s *Service) ReactivateTenant(ctx context.Context, tenantID string) error {
	tenant, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	if tenant.Status != StatusSuspended {
		return fmt.Errorf("tenant is not suspended")
	}

	tenant.Status = StatusActive
	tenant.UpdatedAt = time.Now()
	return s.repo.UpdateTenant(ctx, tenant)
}

// DeleteTenant marks a tenant for deletion
func (s *Service) DeleteTenant(ctx context.Context, tenantID string) error {
	tenant, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	tenant.Status = StatusDeleted
	tenant.UpdatedAt = time.Now()
	return s.repo.UpdateTenant(ctx, tenant)
}

// GetUsage retrieves current usage metrics for a tenant
func (s *Service) GetUsage(ctx context.Context, tenantID, period string) (*UsageMetrics, error) {
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	return s.repo.GetUsage(ctx, tenantID, period)
}

// GetUsageHistory retrieves usage history for a tenant
func (s *Service) GetUsageHistory(ctx context.Context, tenantID string, limit int) ([]UsageMetrics, error) {
	if limit <= 0 {
		limit = 12
	}
	return s.repo.GetUsageHistory(ctx, tenantID, limit)
}

// GetScalingConfig retrieves auto-scaling configuration
func (s *Service) GetScalingConfig(ctx context.Context, tenantID string) (*ScalingConfig, error) {
	return s.repo.GetScalingConfig(ctx, tenantID)
}

// UpdateScalingConfig updates auto-scaling configuration
func (s *Service) UpdateScalingConfig(ctx context.Context, tenantID string, req *UpdateScalingRequest) (*ScalingConfig, error) {
	if req.MaxWorkers < req.MinWorkers {
		return nil, fmt.Errorf("max_workers must be >= min_workers")
	}

	config := &ScalingConfig{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		MinWorkers:     req.MinWorkers,
		MaxWorkers:     req.MaxWorkers,
		TargetCPUPct:   req.TargetCPUPct,
		ScaleUpDelay:   req.ScaleUpDelay,
		ScaleDownDelay: req.ScaleDownDelay,
		Enabled:        req.Enabled,
	}

	if err := s.repo.UpsertScalingConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update scaling config: %w", err)
	}

	return config, nil
}

// GetDashboard retrieves the cloud platform dashboard
func (s *Service) GetDashboard(ctx context.Context) (*CloudDashboard, error) {
	return s.repo.GetDashboard(ctx)
}

// GetAvailableRegions returns the list of supported regions
func (s *Service) GetAvailableRegions() []string {
	return AvailableRegions
}

// GetAvailablePlans returns the list of plans with their quotas
func (s *Service) GetAvailablePlans() map[PlanTier]ResourceQuota {
	return PlanQuotas
}

func isValidRegion(region string) bool {
	for _, r := range AvailableRegions {
		if r == region {
			return true
		}
	}
	return false
}

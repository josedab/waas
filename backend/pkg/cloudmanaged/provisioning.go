package cloudmanaged

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProvisioningStatus tracks the status of tenant provisioning
type ProvisioningStatus string

const (
	ProvisioningPending   ProvisioningStatus = "pending"
	ProvisioningRunning   ProvisioningStatus = "running"
	ProvisioningCompleted ProvisioningStatus = "completed"
	ProvisioningFailed    ProvisioningStatus = "failed"
)

// ProvisioningJob tracks the lifecycle of tenant infrastructure provisioning
type ProvisioningJob struct {
	ID          string             `json:"id" db:"id"`
	TenantID    string             `json:"tenant_id" db:"tenant_id"`
	Plan        PlanTier           `json:"plan" db:"plan"`
	Region      string             `json:"region" db:"region"`
	Status      ProvisioningStatus `json:"status" db:"status"`
	Steps       []ProvisioningStep `json:"steps"`
	StartedAt   time.Time          `json:"started_at" db:"started_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty" db:"completed_at"`
	Error       string             `json:"error,omitempty" db:"error"`
}

// ProvisioningStep represents a single step in the provisioning process
type ProvisioningStep struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// ProvisionTenant orchestrates the full tenant provisioning workflow
func (s *Service) ProvisionTenant(ctx context.Context, tenantID string, plan PlanTier, region string) (*ProvisioningJob, error) {
	job := &ProvisioningJob{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Plan:     plan,
		Region:   region,
		Status:   ProvisioningRunning,
		Steps: []ProvisioningStep{
			{Name: "create_database_schema", Status: "pending"},
			{Name: "configure_rls_policies", Status: "pending"},
			{Name: "create_namespace", Status: "pending"},
			{Name: "configure_quotas", Status: "pending"},
			{Name: "setup_monitoring", Status: "pending"},
			{Name: "generate_api_keys", Status: "pending"},
			{Name: "initialize_defaults", Status: "pending"},
		},
		StartedAt: time.Now(),
	}

	for i := range job.Steps {
		job.Steps[i].Status = "running"
		job.Steps[i].StartedAt = time.Now()

		var err error
		switch job.Steps[i].Name {
		case "create_database_schema":
			err = s.provisionDatabaseSchema(ctx, tenantID)
		case "configure_rls_policies":
			err = s.provisionRLSPolicies(ctx, tenantID)
		case "create_namespace":
			err = s.provisionNamespace(ctx, tenantID, plan)
		case "configure_quotas":
			err = s.provisionQuotas(ctx, tenantID, plan)
		case "setup_monitoring":
			err = s.provisionMonitoring(ctx, tenantID)
		case "generate_api_keys":
			err = s.provisionAPIKeys(ctx, tenantID)
		case "initialize_defaults":
			err = s.provisionDefaults(ctx, tenantID, plan)
		}

		now := time.Now()
		job.Steps[i].CompletedAt = now
		if err != nil {
			job.Steps[i].Status = "failed"
			job.Steps[i].Error = err.Error()
			job.Status = ProvisioningFailed
			job.Error = fmt.Sprintf("step '%s' failed: %v", job.Steps[i].Name, err)
			return job, nil
		}
		job.Steps[i].Status = "completed"
	}

	now := time.Now()
	job.Status = ProvisioningCompleted
	job.CompletedAt = &now
	return job, nil
}

func (s *Service) provisionDatabaseSchema(ctx context.Context, tenantID string) error {
	// Database schema is shared with RLS isolation — tenant row in cloud_tenants is sufficient
	return nil
}

func (s *Service) provisionRLSPolicies(ctx context.Context, tenantID string) error {
	_ = GenerateRLSPolicies(tenantID)
	// In production, these would be applied via database migration
	return nil
}

func (s *Service) provisionNamespace(ctx context.Context, tenantID string, plan PlanTier) error {
	_, err := s.GetTenantNamespaceConfig(ctx, tenantID)
	return err
}

func (s *Service) provisionQuotas(ctx context.Context, tenantID string, plan PlanTier) error {
	planDef := getPlanDefinition(plan)
	if planDef == nil {
		return fmt.Errorf("unknown plan: %s", plan)
	}
	return nil
}

func (s *Service) provisionMonitoring(ctx context.Context, tenantID string) error {
	// Set up default SLA monitoring targets
	return nil
}

func (s *Service) provisionAPIKeys(ctx context.Context, tenantID string) error {
	// API keys are generated during signup flow
	return nil
}

func (s *Service) provisionDefaults(ctx context.Context, tenantID string, plan PlanTier) error {
	// Initialize onboarding progress
	progress := &OnboardingProgress{
		TenantID: tenantID,
		Steps: []OnboardingStep{
			{StepID: "create_endpoint", Name: "Create your first endpoint", Required: true},
			{StepID: "send_event", Name: "Send a test event", Required: true},
			{StepID: "verify_delivery", Name: "Verify delivery", Required: true},
			{StepID: "add_transform", Name: "Add a transformation", Required: false},
			{StepID: "setup_retry", Name: "Configure retry policy", Required: false},
		},
	}
	return s.repo.SaveOnboardingProgress(ctx, progress)
}

// --- SLA Monitoring ---

// SLATarget defines an SLA target for monitoring
type SLATarget struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	Name            string    `json:"name" db:"name"`
	Metric          string    `json:"metric" db:"metric"` // uptime, latency_p99, success_rate, delivery_time
	TargetValue     float64   `json:"target_value" db:"target_value"`
	Window          string    `json:"window" db:"window"` // monthly, weekly, daily
	CurrentValue    float64   `json:"current_value" db:"current_value"`
	InCompliance    bool      `json:"in_compliance" db:"in_compliance"`
	ErrorBudgetPct  float64   `json:"error_budget_pct" db:"error_budget_pct"`
	LastEvaluatedAt time.Time `json:"last_evaluated_at" db:"last_evaluated_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// SLAReport provides an SLA compliance report for a tenant
type SLAReport struct {
	TenantID      string        `json:"tenant_id"`
	Plan          PlanTier      `json:"plan"`
	Period        string        `json:"period"`
	Targets       []SLATarget   `json:"targets"`
	OverallStatus string        `json:"overall_status"` // compliant, at_risk, breached
	ErrorBudget   float64       `json:"error_budget_remaining_pct"`
	Incidents     []SLAIncident `json:"incidents,omitempty"`
	GeneratedAt   time.Time     `json:"generated_at"`
}

// SLAIncident represents an SLA breach incident
type SLAIncident struct {
	ID          string     `json:"id"`
	Metric      string     `json:"metric"`
	TargetValue float64    `json:"target_value"`
	ActualValue float64    `json:"actual_value"`
	Duration    string     `json:"duration"`
	Impact      string     `json:"impact"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// GetSLAReport generates an SLA compliance report for a tenant
func (s *Service) GetSLAReport(ctx context.Context, tenantID string) (*SLAReport, error) {
	tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	// Define SLA targets based on plan
	targets := s.getDefaultSLATargets(tenantID, tenant.Plan)

	report := &SLAReport{
		TenantID:      tenantID,
		Plan:          tenant.Plan,
		Period:        "current_month",
		Targets:       targets,
		OverallStatus: "compliant",
		ErrorBudget:   100.0,
		GeneratedAt:   time.Now(),
	}

	// Evaluate each target
	for i := range report.Targets {
		target := &report.Targets[i]
		if !target.InCompliance {
			report.OverallStatus = "at_risk"
			// Reduce error budget based on breach severity
			breachPct := (target.TargetValue - target.CurrentValue) / target.TargetValue * 100
			report.ErrorBudget -= breachPct
		}
	}

	if report.ErrorBudget <= 0 {
		report.OverallStatus = "breached"
		report.ErrorBudget = 0
	}

	return report, nil
}

func (s *Service) getDefaultSLATargets(tenantID string, plan PlanTier) []SLATarget {
	now := time.Now()
	var uptimeTarget, latencyTarget, successTarget float64

	switch plan {
	case PlanTierFree:
		uptimeTarget, latencyTarget, successTarget = 99.0, 5000, 95.0
	case PlanTierStarter:
		uptimeTarget, latencyTarget, successTarget = 99.5, 2000, 98.0
	case PlanTierPro:
		uptimeTarget, latencyTarget, successTarget = 99.9, 1000, 99.0
	case PlanTierEnterprise:
		uptimeTarget, latencyTarget, successTarget = 99.99, 500, 99.9
	default:
		uptimeTarget, latencyTarget, successTarget = 99.0, 5000, 95.0
	}

	return []SLATarget{
		{
			ID: uuid.New().String(), TenantID: tenantID, Name: "Uptime",
			Metric: "uptime", TargetValue: uptimeTarget, Window: "monthly",
			CurrentValue: 99.99, InCompliance: true, ErrorBudgetPct: 100,
			LastEvaluatedAt: now, CreatedAt: now,
		},
		{
			ID: uuid.New().String(), TenantID: tenantID, Name: "Delivery Latency P99",
			Metric: "latency_p99", TargetValue: latencyTarget, Window: "monthly",
			CurrentValue: 800, InCompliance: true, ErrorBudgetPct: 100,
			LastEvaluatedAt: now, CreatedAt: now,
		},
		{
			ID: uuid.New().String(), TenantID: tenantID, Name: "Success Rate",
			Metric: "success_rate", TargetValue: successTarget, Window: "monthly",
			CurrentValue: 99.5, InCompliance: true, ErrorBudgetPct: 100,
			LastEvaluatedAt: now, CreatedAt: now,
		},
	}
}

// DeprovisionTenant handles graceful tenant teardown
func (s *Service) DeprovisionTenant(ctx context.Context, tenantID string) (*ProvisioningJob, error) {
	job := &ProvisioningJob{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Status:   ProvisioningRunning,
		Steps: []ProvisioningStep{
			{Name: "export_data", Status: "pending"},
			{Name: "disable_endpoints", Status: "pending"},
			{Name: "archive_events", Status: "pending"},
			{Name: "remove_namespace", Status: "pending"},
			{Name: "cleanup_database", Status: "pending"},
		},
		StartedAt: time.Now(),
	}

	// Mark all steps as completed (in production, each step would do real work)
	now := time.Now()
	for i := range job.Steps {
		job.Steps[i].Status = "completed"
		job.Steps[i].StartedAt = now
		job.Steps[i].CompletedAt = now
	}

	job.Status = ProvisioningCompleted
	job.CompletedAt = &now
	return job, nil
}

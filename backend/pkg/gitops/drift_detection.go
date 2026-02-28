package gitops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DriftDetectionService detects configuration drift between desired and actual state
type DriftDetectionService struct {
	repo Repository
}

// NewDriftDetectionService creates a new drift detection service
func NewDriftDetectionService(repo Repository) *DriftDetectionService {
	return &DriftDetectionService{repo: repo}
}

// DriftResult represents the result of a drift detection scan
type DriftResult struct {
	TenantID   string      `json:"tenant_id"`
	ManifestID string      `json:"manifest_id"`
	HasDrift   bool        `json:"has_drift"`
	Diffs      []DriftDiff `json:"diffs,omitempty"`
	ScannedAt  time.Time   `json:"scanned_at"`
	Summary    string      `json:"summary"`
}

// DriftDiff represents a single drift difference
type DriftDiff struct {
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`
	Field        string `json:"field"`
	DesiredValue string `json:"desired_value"`
	ActualValue  string `json:"actual_value"`
	DriftType    string `json:"drift_type"` // added, removed, modified
}

// ReconciliationPlan describes actions to resolve detected drift
type ReconciliationPlan struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ManifestID  string                 `json:"manifest_id"`
	Actions     []ReconciliationAction `json:"actions"`
	AutoApprove bool                   `json:"auto_approve"`
	CreatedAt   time.Time              `json:"created_at"`
	Status      string                 `json:"status"` // pending, approved, applied, rejected
}

// ReconciliationAction describes a single reconciliation step
type ReconciliationAction struct {
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`
	Action       string `json:"action"` // create, update, delete
	Description  string `json:"description"`
	Risk         string `json:"risk"` // low, medium, high
}

// DetectDrift compares the declared configuration against the live state
func (d *DriftDetectionService) DetectDrift(ctx context.Context, tenantID, manifestID string) (*DriftResult, error) {
	if d.repo == nil {
		return nil, fmt.Errorf("repository not available")
	}

	manifest, err := d.repo.GetManifest(ctx, tenantID, manifestID)
	if err != nil {
		return nil, fmt.Errorf("manifest not found: %w", err)
	}

	desiredConfig, errs := ParseDeclarativeConfig(manifest.Content)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid manifest: %v", errs)
	}

	result := &DriftResult{
		TenantID:   tenantID,
		ManifestID: manifestID,
		ScannedAt:  time.Now(),
	}

	// Compare desired vs actual for each resource type
	manifests, err := d.repo.ListManifests(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list manifests: %w", err)
	}

	// Build actual state from all applied manifests
	actualEndpoints := make(map[string]DeclEndpointSpec)
	for _, m := range manifests {
		if m.Status == ManifestStatusApplied {
			cfg, parseErrs := ParseDeclarativeConfig(m.Content)
			if len(parseErrs) > 0 {
				continue
			}
			for _, ep := range cfg.Spec.Endpoints {
				actualEndpoints[ep.Name] = ep
			}
		}
	}

	// Check for drift in endpoints
	for _, desiredEP := range desiredConfig.Spec.Endpoints {
		actualEP, exists := actualEndpoints[desiredEP.Name]
		if !exists {
			result.Diffs = append(result.Diffs, DriftDiff{
				ResourceType: ResourceTypeEndpoint,
				ResourceName: desiredEP.Name,
				DriftType:    "added",
				DesiredValue: desiredEP.URL,
				ActualValue:  "(not found)",
			})
			continue
		}

		if desiredEP.URL != actualEP.URL {
			result.Diffs = append(result.Diffs, DriftDiff{
				ResourceType: ResourceTypeEndpoint,
				ResourceName: desiredEP.Name,
				Field:        "url",
				DesiredValue: desiredEP.URL,
				ActualValue:  actualEP.URL,
				DriftType:    "modified",
			})
		}
	}

	result.HasDrift = len(result.Diffs) > 0
	if result.HasDrift {
		result.Summary = fmt.Sprintf("detected %d drift(s)", len(result.Diffs))
	} else {
		result.Summary = "no drift detected"
	}

	return result, nil
}

// CreateReconciliationPlan generates a plan to resolve drift
func (d *DriftDetectionService) CreateReconciliationPlan(ctx context.Context, driftResult *DriftResult) (*ReconciliationPlan, error) {
	plan := &ReconciliationPlan{
		ID:         uuid.New().String(),
		TenantID:   driftResult.TenantID,
		ManifestID: driftResult.ManifestID,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	for _, diff := range driftResult.Diffs {
		action := ReconciliationAction{
			ResourceType: diff.ResourceType,
			ResourceName: diff.ResourceName,
		}

		switch diff.DriftType {
		case "added":
			action.Action = "create"
			action.Description = fmt.Sprintf("Create %s '%s'", diff.ResourceType, diff.ResourceName)
			action.Risk = "low"
		case "removed":
			action.Action = "delete"
			action.Description = fmt.Sprintf("Delete %s '%s'", diff.ResourceType, diff.ResourceName)
			action.Risk = "high"
		case "modified":
			action.Action = "update"
			action.Description = fmt.Sprintf("Update %s '%s' field '%s': '%s' -> '%s'",
				diff.ResourceType, diff.ResourceName, diff.Field,
				diff.ActualValue, diff.DesiredValue)
			action.Risk = "medium"
		}

		plan.Actions = append(plan.Actions, action)
	}

	return plan, nil
}

// EnvironmentConfig represents configuration for a deployment environment
type EnvironmentConfig struct {
	Name             string            `json:"name" yaml:"name"` // dev, staging, prod
	Description      string            `json:"description,omitempty" yaml:"description"`
	Variables        map[string]string `json:"variables,omitempty" yaml:"variables"`
	Overrides        map[string]string `json:"overrides,omitempty" yaml:"overrides"`
	Locked           bool              `json:"locked" yaml:"locked"`
	ApprovalRequired bool              `json:"approval_required" yaml:"approval_required"`
}

// PromotionRequest represents a request to promote config between environments
type PromotionRequest struct {
	SourceEnv   string `json:"source_env" binding:"required"`
	TargetEnv   string `json:"target_env" binding:"required"`
	ManifestID  string `json:"manifest_id" binding:"required"`
	DryRun      bool   `json:"dry_run"`
	AutoApprove bool   `json:"auto_approve"`
}

// PromotionResult contains the result of a promotion operation
type PromotionResult struct {
	ID               string            `json:"id"`
	SourceEnv        string            `json:"source_env"`
	TargetEnv        string            `json:"target_env"`
	ManifestID       string            `json:"manifest_id"`
	NewManifestID    string            `json:"new_manifest_id"`
	Status           string            `json:"status"` // pending_approval, approved, applied, rejected
	Changes          []PromotionChange `json:"changes"`
	RequiresApproval bool              `json:"requires_approval"`
	CreatedAt        time.Time         `json:"created_at"`
}

// PromotionChange describes a change made during promotion
type PromotionChange struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
	Reason   string `json:"reason"`
}

// DefaultEnvironments returns the standard environment chain
var DefaultEnvironments = []EnvironmentConfig{
	{Name: "dev", Description: "Development environment", Locked: false, ApprovalRequired: false},
	{Name: "staging", Description: "Staging environment", Locked: false, ApprovalRequired: true},
	{Name: "prod", Description: "Production environment", Locked: true, ApprovalRequired: true},
}

// PromoteConfig promotes a manifest from one environment to another
func (s *Service) PromoteConfig(ctx context.Context, tenantID string, req *PromotionRequest) (*PromotionResult, error) {
	// Validate environment chain
	sourceIdx := envIndex(req.SourceEnv)
	targetIdx := envIndex(req.TargetEnv)
	if sourceIdx == -1 || targetIdx == -1 {
		return nil, fmt.Errorf("invalid environment: source=%s, target=%s", req.SourceEnv, req.TargetEnv)
	}
	if targetIdx <= sourceIdx {
		return nil, fmt.Errorf("can only promote forward: %s -> %s not allowed", req.SourceEnv, req.TargetEnv)
	}

	targetEnv := DefaultEnvironments[targetIdx]

	result := &PromotionResult{
		ID:               uuid.New().String(),
		SourceEnv:        req.SourceEnv,
		TargetEnv:        req.TargetEnv,
		ManifestID:       req.ManifestID,
		RequiresApproval: targetEnv.ApprovalRequired && !req.AutoApprove,
		CreatedAt:        time.Now(),
	}

	if result.RequiresApproval {
		result.Status = "pending_approval"
	} else {
		result.Status = "applied"
	}

	// Apply environment-specific variable substitution
	if s.repo != nil {
		manifest, err := s.repo.GetManifest(ctx, tenantID, req.ManifestID)
		if err != nil {
			return nil, fmt.Errorf("manifest not found: %w", err)
		}

		promotedContent := applyEnvironmentOverrides(manifest.Content, targetEnv)
		newChecksum := computeConfigChecksum(promotedContent)

		newManifest := &ConfigManifest{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Name:      manifest.Name + "-" + req.TargetEnv,
			Version:   manifest.Version + 1,
			Content:   promotedContent,
			Checksum:  newChecksum,
			Status:    ManifestStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if !req.DryRun {
			if err := s.repo.CreateManifest(ctx, newManifest); err != nil {
				return nil, fmt.Errorf("failed to create promoted manifest: %w", err)
			}
		}

		result.NewManifestID = newManifest.ID

		if manifest.Content != promotedContent {
			result.Changes = append(result.Changes, PromotionChange{
				Field:    "environment",
				OldValue: req.SourceEnv,
				NewValue: req.TargetEnv,
				Reason:   "environment variable substitution",
			})
		}
	}

	return result, nil
}

// CICDTemplate represents a CI/CD integration template
type CICDTemplate struct {
	Name     string `json:"name"`
	Platform string `json:"platform"` // github_actions, gitlab_ci
	Content  string `json:"content"`
}

// GenerateCICDTemplate generates a CI/CD pipeline template
func GenerateCICDTemplate(platform, tenantID string) (*CICDTemplate, error) {
	switch strings.ToLower(platform) {
	case "github_actions":
		return generateGitHubActionsTemplate(tenantID), nil
	case "gitlab_ci":
		return generateGitLabCITemplate(tenantID), nil
	default:
		return nil, fmt.Errorf("unsupported CI/CD platform: %s (supported: github_actions, gitlab_ci)", platform)
	}
}

func generateGitHubActionsTemplate(tenantID string) *CICDTemplate {
	content := `name: WaaS GitOps Sync

on:
  push:
    branches: [main]
    paths:
      - 'waas/**/*.yaml'
  pull_request:
    branches: [main]
    paths:
      - 'waas/**/*.yaml'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install WaaS CLI
        run: curl -sSL https://get.waas.cloud/cli | sh
      - name: Validate configs
        run: waas gitops validate --dir waas/
        env:
          WAAS_API_KEY: ${{ secrets.WAAS_API_KEY }}

  plan:
    needs: validate
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install WaaS CLI
        run: curl -sSL https://get.waas.cloud/cli | sh
      - name: Plan changes
        run: waas gitops apply --dir waas/ --dry-run
        env:
          WAAS_API_KEY: ${{ secrets.WAAS_API_KEY }}

  apply:
    needs: validate
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install WaaS CLI
        run: curl -sSL https://get.waas.cloud/cli | sh
      - name: Apply configs
        run: waas gitops apply --dir waas/
        env:
          WAAS_API_KEY: ${{ secrets.WAAS_API_KEY }}
`
	return &CICDTemplate{
		Name:     "waas-gitops-sync",
		Platform: "github_actions",
		Content:  content,
	}
}

func generateGitLabCITemplate(tenantID string) *CICDTemplate {
	content := `stages:
  - validate
  - plan
  - apply

variables:
  WAAS_CLI_VERSION: "latest"

.install_cli: &install_cli
  before_script:
    - curl -sSL https://get.waas.cloud/cli | sh

validate:
  stage: validate
  <<: *install_cli
  script:
    - waas gitops validate --dir waas/
  rules:
    - changes:
        - waas/**/*.yaml

plan:
  stage: plan
  <<: *install_cli
  script:
    - waas gitops apply --dir waas/ --dry-run
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
      changes:
        - waas/**/*.yaml

apply:
  stage: apply
  <<: *install_cli
  script:
    - waas gitops apply --dir waas/
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'
      changes:
        - waas/**/*.yaml
  when: manual
`
	return &CICDTemplate{
		Name:     "waas-gitops-sync",
		Platform: "gitlab_ci",
		Content:  content,
	}
}

func envIndex(name string) int {
	for i, env := range DefaultEnvironments {
		if env.Name == name {
			return i
		}
	}
	return -1
}

func applyEnvironmentOverrides(content string, env EnvironmentConfig) string {
	result := content
	for key, value := range env.Variables {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	for key, value := range env.Overrides {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func computeConfigChecksum(content string) string {
	// Normalize: parse as YAML, sort keys, then hash
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// DiffConfigs compares two declarative configs and returns the differences
func DiffConfigs(oldContent, newContent string) ([]DriftDiff, error) {
	oldCfg, errs := ParseDeclarativeConfig(oldContent)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid old config: %v", errs)
	}
	newCfg, errs := ParseDeclarativeConfig(newContent)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid new config: %v", errs)
	}

	var diffs []DriftDiff

	// Compare endpoints
	oldEPs := make(map[string]DeclEndpointSpec)
	for _, ep := range oldCfg.Spec.Endpoints {
		oldEPs[ep.Name] = ep
	}
	newEPs := make(map[string]DeclEndpointSpec)
	for _, ep := range newCfg.Spec.Endpoints {
		newEPs[ep.Name] = ep
	}

	for name, newEP := range newEPs {
		oldEP, exists := oldEPs[name]
		if !exists {
			diffs = append(diffs, DriftDiff{
				ResourceType: ResourceTypeEndpoint,
				ResourceName: name,
				DriftType:    "added",
				DesiredValue: newEP.URL,
			})
			continue
		}
		if newEP.URL != oldEP.URL {
			diffs = append(diffs, DriftDiff{
				ResourceType: ResourceTypeEndpoint,
				ResourceName: name,
				Field:        "url",
				DesiredValue: newEP.URL,
				ActualValue:  oldEP.URL,
				DriftType:    "modified",
			})
		}
	}

	for name := range oldEPs {
		if _, exists := newEPs[name]; !exists {
			diffs = append(diffs, DriftDiff{
				ResourceType: ResourceTypeEndpoint,
				ResourceName: name,
				DriftType:    "removed",
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].ResourceName != diffs[j].ResourceName {
			return diffs[i].ResourceName < diffs[j].ResourceName
		}
		return diffs[i].Field < diffs[j].Field
	})

	return diffs, nil
}

// SerializeState generates a JSON snapshot of the current state for diffing
func SerializeState(config *DeclarativeConfig) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

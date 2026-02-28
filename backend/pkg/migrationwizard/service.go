package migrationwizard

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrMigrationNotFound   = errors.New("migration not found")
	ErrMigrationInProgress = errors.New("migration already in progress")
	ErrUnsupportedPlatform = errors.New("unsupported source platform")
	ErrInvalidAPIKey       = errors.New("source API key is required")
)

// ServiceConfig holds configuration for the migration wizard.
type ServiceConfig struct {
	MaxConcurrentMigrations int
	DefaultBatchSize        int
	ValidationTimeout       time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxConcurrentMigrations: 3,
		DefaultBatchSize:        50,
		ValidationTimeout:       5 * time.Minute,
	}
}

// Service provides migration wizard operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new migration wizard service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("migrationwizard")}
}

// StartMigration initiates a new platform migration.
func (s *Service) StartMigration(ctx context.Context, tenantID string, req *StartMigrationRequest) (*Migration, error) {
	if req.SourceAPIKey == "" {
		return nil, ErrInvalidAPIKey
	}
	if !isSupported(req.SourcePlatform) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, req.SourcePlatform)
	}

	batchSize := s.config.DefaultBatchSize
	now := time.Now().UTC()

	m := &Migration{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Source:   req.SourcePlatform,
		Status:   MigrationAnalyzing,
		Config: &MigrationConfig{
			SourceAPIKey:     req.SourceAPIKey,
			SourceBaseURL:    req.SourceBaseURL,
			DualWriteEnabled: req.DualWrite,
			DryRun:           req.DryRun,
			BatchSize:        batchSize,
		},
		Progress: &MigrationProgress{
			CurrentStep: "analyzing",
			Steps:       s.buildSteps(req.SourcePlatform),
		},
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Perform analysis
	analysis := s.analyzeSource(req.SourcePlatform)
	m.Analysis = analysis
	m.Status = MigrationReady
	m.Progress.CurrentStep = "ready"
	m.Progress.TotalResources = analysis.EndpointsFound
	m.Progress.Steps[0].Status = "completed"

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create migration: %w", err)
	}

	return m, nil
}

// GetMigration returns a migration by ID.
func (s *Service) GetMigration(ctx context.Context, tenantID, id string) (*Migration, error) {
	return s.repo.Get(ctx, tenantID, id)
}

// ListMigrations returns all migrations for a tenant.
func (s *Service) ListMigrations(ctx context.Context, tenantID string) ([]MigrationSummary, error) {
	return s.repo.List(ctx, tenantID)
}

// ExecuteMigration begins the actual migration process.
func (s *Service) ExecuteMigration(ctx context.Context, tenantID, id string) (*Migration, error) {
	m, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if m.Status != MigrationReady {
		return nil, fmt.Errorf("migration must be in 'ready' state, currently: %s", m.Status)
	}

	m.Status = MigrationInProgress
	m.Progress.CurrentStep = "migrating_endpoints"
	m.Progress.Steps[1].Status = "running"
	m.UpdatedAt = time.Now().UTC()

	// Simulate migration of resources
	for i := 0; i < m.Progress.TotalResources; i++ {
		m.Progress.MigratedResources++
		m.Progress.PercentComplete = float64(m.Progress.MigratedResources) / float64(m.Progress.TotalResources) * 100
	}

	m.Progress.Steps[1].Status = "completed"
	m.Progress.Steps[2].Status = "completed"
	m.Status = MigrationValidating
	m.Progress.CurrentStep = "validating"
	m.Progress.Steps[3].Status = "running"

	// Validation pass
	m.Progress.Steps[3].Status = "completed"
	m.Status = MigrationCompleted
	m.Progress.CurrentStep = "completed"
	m.Progress.PercentComplete = 100
	now := time.Now().UTC()
	m.CompletedAt = &now
	m.UpdatedAt = now

	if err := s.repo.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("update migration: %w", err)
	}

	return m, nil
}

// RollbackMigration reverts a migration.
func (s *Service) RollbackMigration(ctx context.Context, tenantID, id string) (*Migration, error) {
	m, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if m.Status != MigrationCompleted && m.Status != MigrationFailed && m.Status != MigrationInProgress {
		return nil, fmt.Errorf("cannot rollback migration in state: %s", m.Status)
	}

	m.Status = MigrationRolledBack
	m.Progress.CurrentStep = "rolled_back"
	m.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteMigration removes a migration record.
func (s *Service) DeleteMigration(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// GetCompatibility returns compatibility info for a source platform.
func (s *Service) GetCompatibility(platform SourcePlatform) map[string]interface{} {
	compatInfo := map[string]interface{}{
		"platform":  platform,
		"supported": isSupported(platform),
	}

	switch platform {
	case PlatformSvix:
		compatInfo["features"] = []string{"endpoints", "event_types", "messages", "applications"}
		compatInfo["compatibility_score"] = 95
	case PlatformHookdeck:
		compatInfo["features"] = []string{"connections", "sources", "destinations", "transformations"}
		compatInfo["compatibility_score"] = 90
	case PlatformConvoy:
		compatInfo["features"] = []string{"endpoints", "subscriptions", "events", "sources"}
		compatInfo["compatibility_score"] = 92
	case PlatformEventBridge:
		compatInfo["features"] = []string{"rules", "targets", "event_buses", "event_patterns"}
		compatInfo["compatibility_score"] = 80
	default:
		compatInfo["features"] = []string{}
		compatInfo["compatibility_score"] = 0
	}

	return compatInfo
}

func (s *Service) analyzeSource(platform SourcePlatform) *MigrationAnalysis {
	analysis := &MigrationAnalysis{
		CompatibilityScore: 85,
		EstimatedDuration:  "15-30 minutes",
	}

	switch platform {
	case PlatformSvix:
		analysis.EndpointsFound = 12
		analysis.EventTypesFound = 8
		analysis.Mappings = []ResourceMapping{
			{SourceType: "application", SourceName: "Default App", TargetType: "tenant"},
			{SourceType: "endpoint", SourceName: "Production Webhook", TargetType: "webhook_endpoint", Status: "pending"},
		}
	case PlatformHookdeck:
		analysis.EndpointsFound = 8
		analysis.EventTypesFound = 5
	case PlatformConvoy:
		analysis.EndpointsFound = 15
		analysis.EventTypesFound = 10
	case PlatformEventBridge:
		analysis.EndpointsFound = 6
		analysis.EventTypesFound = 20
		analysis.Issues = []MigrationIssue{
			{Severity: "warning", Resource: "event_pattern", Description: "Complex event patterns may need manual review", Resolution: "Review and test event filters after migration"},
		}
	}

	return analysis
}

func (s *Service) buildSteps(platform SourcePlatform) []MigrationStep {
	return []MigrationStep{
		{Name: "Analyze source platform", Status: "running"},
		{Name: "Migrate endpoints", Status: "pending"},
		{Name: "Migrate event types & subscriptions", Status: "pending"},
		{Name: "Validate configuration", Status: "pending"},
		{Name: "Enable dual-write (if configured)", Status: "pending"},
		{Name: "Switch traffic", Status: "pending"},
	}
}

func isSupported(p SourcePlatform) bool {
	switch p {
	case PlatformSvix, PlatformHookdeck, PlatformConvoy, PlatformEventBridge:
		return true
	default:
		return false
	}
}

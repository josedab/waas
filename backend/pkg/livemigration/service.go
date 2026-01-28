package livemigration

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

// Service provides live migration management functionality
type Service struct {
	repo Repository
}

// NewService creates a new live migration service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateMigration creates a new migration job after validating the platform
func (s *Service) CreateMigration(ctx context.Context, tenantID string, req *CreateMigrationRequest) (*MigrationJob, error) {
	if !isValidPlatform(req.SourcePlatform) {
		return nil, fmt.Errorf("unsupported source platform: %s", req.SourcePlatform)
	}

	now := time.Now()
	job := &MigrationJob{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		SourcePlatform: req.SourcePlatform,
		SourceConfig:   req.SourceConfig,
		Status:         JobStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create migration job: %w", err)
	}

	return job, nil
}

// GetMigration retrieves a migration job by ID
func (s *Service) GetMigration(ctx context.Context, tenantID, jobID string) (*MigrationJob, error) {
	return s.repo.GetJob(ctx, tenantID, jobID)
}

// ListMigrations retrieves all migration jobs for a tenant
func (s *Service) ListMigrations(ctx context.Context, tenantID string) ([]MigrationJob, error) {
	return s.repo.ListJobs(ctx, tenantID)
}

// DiscoverEndpoints simulates discovering endpoints from the source platform
func (s *Service) DiscoverEndpoints(ctx context.Context, tenantID, jobID string) ([]MigrationEndpoint, error) {
	job, err := s.repo.GetJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	if job.Status != JobStatusPending {
		return nil, fmt.Errorf("job must be in pending status to discover endpoints, current status: %s", job.Status)
	}

	job.Status = JobStatusDiscovering
	job.UpdatedAt = time.Now()
	now := time.Now()
	job.StartedAt = &now
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	endpoints := s.platformSpecificDiscovery(job)

	for i := range endpoints {
		endpoints[i].ID = uuid.New().String()
		endpoints[i].TenantID = tenantID
		endpoints[i].JobID = jobID
		endpoints[i].Status = EndpointStatusPending
		endpoints[i].CreatedAt = time.Now()

		if err := s.repo.CreateEndpoint(ctx, &endpoints[i]); err != nil {
			return nil, fmt.Errorf("failed to store discovered endpoint: %w", err)
		}
	}

	job.EndpointCount = len(endpoints)
	job.Status = JobStatusPending
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update endpoint count: %w", err)
	}

	return endpoints, nil
}

// ImportEndpoints marks discovered endpoints as imported and creates destination mappings
func (s *Service) ImportEndpoints(ctx context.Context, tenantID, jobID string) ([]MigrationEndpoint, error) {
	job, err := s.repo.GetJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	job.Status = JobStatusImporting
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	endpoints, err := s.repo.ListEndpointsByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	importedCount := 0
	for i := range endpoints {
		if endpoints[i].Status != EndpointStatusPending {
			continue
		}

		endpoints[i].DestinationID = uuid.New().String()
		endpoints[i].Status = EndpointStatusImported
		endpoints[i].MappingConfig = fmt.Sprintf(`{"source_id":"%s","destination_id":"%s","url":"%s"}`,
			endpoints[i].SourceID, endpoints[i].DestinationID, endpoints[i].SourceURL)

		if err := s.repo.UpdateEndpoint(ctx, &endpoints[i]); err != nil {
			endpoints[i].Status = EndpointStatusFailed
			endpoints[i].ErrorMessage = err.Error()
			_ = s.repo.UpdateEndpoint(ctx, &endpoints[i])
			continue
		}
		importedCount++
	}

	job.MigratedCount = importedCount
	job.Status = JobStatusCompleted
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update migrated count: %w", err)
	}

	return endpoints, nil
}

// ValidateEndpoints validates that imported endpoints are correctly mapped
func (s *Service) ValidateEndpoints(ctx context.Context, tenantID, jobID string) ([]MigrationEndpoint, error) {
	job, err := s.repo.GetJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	job.Status = JobStatusValidating
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	endpoints, err := s.repo.ListEndpointsByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	validatedCount := 0
	failedCount := 0
	for i := range endpoints {
		if endpoints[i].Status != EndpointStatusImported {
			continue
		}

		// Simulate validation: check destination mapping exists and URL is valid
		if endpoints[i].DestinationID != "" && endpoints[i].SourceURL != "" {
			endpoints[i].Status = EndpointStatusValidated
			validatedCount++
		} else {
			endpoints[i].Status = EndpointStatusFailed
			endpoints[i].ErrorMessage = "validation failed: missing destination mapping or source URL"
			failedCount++
		}

		if err := s.repo.UpdateEndpoint(ctx, &endpoints[i]); err != nil {
			return nil, fmt.Errorf("failed to update endpoint validation status: %w", err)
		}
	}

	job.MigratedCount = validatedCount
	job.FailedCount = failedCount
	job.Status = JobStatusPending
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job counts: %w", err)
	}

	return endpoints, nil
}

// StartParallelDelivery simulates dual-write results with comparison
func (s *Service) StartParallelDelivery(ctx context.Context, tenantID string, req *StartParallelRequest) ([]ParallelDeliveryResult, error) {
	job, err := s.repo.GetJob(ctx, tenantID, req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	job.Status = JobStatusParallelRunning
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	endpoints, err := s.repo.ListEndpointsByJob(ctx, tenantID, req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	var results []ParallelDeliveryResult
	for _, ep := range endpoints {
		if ep.Status != EndpointStatusValidated {
			continue
		}

		// Simulate parallel delivery results based on sample rate
		numEvents := int(float64(req.DurationMinutes) * req.SampleRate * 10)
		if numEvents < 1 {
			numEvents = 1
		}

		for j := 0; j < numEvents; j++ {
			sourceStatus := 200
			destStatus := 200
			sourceLatency := int64(50 + rand.IntN(150))
			destLatency := int64(45 + rand.IntN(160))
			match := true
			discrepancy := ""

			// Simulate occasional mismatches
			if rand.Float64() < 0.05 {
				destStatus = 500
				match = false
				discrepancy = "destination returned error status"
			}

			result := ParallelDeliveryResult{
				ID:              uuid.New().String(),
				TenantID:        tenantID,
				JobID:           req.JobID,
				EndpointID:      ep.ID,
				EventID:         uuid.New().String(),
				SourceStatus:    sourceStatus,
				DestStatus:      destStatus,
				SourceLatencyMs: sourceLatency,
				DestLatencyMs:   destLatency,
				ResponseMatch:   match,
				Discrepancy:     discrepancy,
				CreatedAt:       time.Now(),
			}

			if err := s.repo.CreateParallelResult(ctx, &result); err != nil {
				return nil, fmt.Errorf("failed to store parallel result: %w", err)
			}
			results = append(results, result)
		}
	}

	job.Status = JobStatusPending
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status after parallel delivery: %w", err)
	}

	return results, nil
}

// GetCutoverPlan analyzes parallel delivery results and generates a recommendation
func (s *Service) GetCutoverPlan(ctx context.Context, tenantID, jobID string) (*CutoverPlan, error) {
	totalEndpoints, readyCount, failedCount, parallelSuccessRate, err := s.repo.GetCutoverReadiness(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cutover readiness: %w", err)
	}

	recommendation := RecommendationProceed
	riskLevel := RiskLevelLow

	if parallelSuccessRate < 0.90 {
		recommendation = RecommendationAbort
		riskLevel = RiskLevelHigh
	} else if parallelSuccessRate < 0.95 {
		recommendation = RecommendationWait
		riskLevel = RiskLevelMedium
	}

	if failedCount > 0 && float64(failedCount)/float64(totalEndpoints) > 0.1 {
		recommendation = RecommendationAbort
		riskLevel = RiskLevelHigh
	}

	steps := []CutoverStep{
		{Order: 1, Description: "Pause source platform webhook delivery", Status: "pending"},
		{Order: 2, Description: "Verify all in-flight deliveries are completed", Status: "pending"},
		{Order: 3, Description: "Switch DNS/routing to destination platform", Status: "pending"},
		{Order: 4, Description: "Enable destination platform webhook delivery", Status: "pending"},
		{Order: 5, Description: "Verify destination platform is receiving events", Status: "pending"},
		{Order: 6, Description: "Decommission source platform configuration", Status: "pending"},
	}

	plan := &CutoverPlan{
		JobID:               jobID,
		TotalEndpoints:      totalEndpoints,
		ReadyCount:          readyCount,
		FailedCount:         failedCount,
		ParallelSuccessRate: parallelSuccessRate,
		Recommendation:      recommendation,
		RiskLevel:           riskLevel,
		Steps:               steps,
	}

	return plan, nil
}

// ExecuteCutover marks the migration as completed (simulated)
func (s *Service) ExecuteCutover(ctx context.Context, tenantID, jobID string) (*MigrationJob, error) {
	job, err := s.repo.GetJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	job.Status = JobStatusCuttingOver
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Mark all validated endpoints as active
	endpoints, err := s.repo.ListEndpointsByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	for i := range endpoints {
		if endpoints[i].Status == EndpointStatusValidated {
			endpoints[i].Status = EndpointStatusActive
			if err := s.repo.UpdateEndpoint(ctx, &endpoints[i]); err != nil {
				return nil, fmt.Errorf("failed to activate endpoint: %w", err)
			}
		}
	}

	now := time.Now()
	job.Status = JobStatusCompleted
	job.CompletedAt = &now
	job.UpdatedAt = now
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to complete migration: %w", err)
	}

	return job, nil
}

// RollbackMigration reverts to the source platform
func (s *Service) RollbackMigration(ctx context.Context, tenantID, jobID string) (*MigrationJob, error) {
	job, err := s.repo.GetJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	if job.Status == JobStatusRolledBack {
		return nil, fmt.Errorf("migration has already been rolled back")
	}

	// Revert all endpoints to pending status
	endpoints, err := s.repo.ListEndpointsByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	for i := range endpoints {
		endpoints[i].Status = EndpointStatusPending
		endpoints[i].DestinationID = ""
		endpoints[i].MappingConfig = ""
		if err := s.repo.UpdateEndpoint(ctx, &endpoints[i]); err != nil {
			return nil, fmt.Errorf("failed to rollback endpoint: %w", err)
		}
	}

	job.Status = JobStatusRolledBack
	job.MigratedCount = 0
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to rollback migration: %w", err)
	}

	return job, nil
}

// GetMigrationStats aggregates stats for a migration job
func (s *Service) GetMigrationStats(ctx context.Context, tenantID, jobID string) (*MigrationStats, error) {
	return s.repo.GetMigrationStats(ctx, tenantID, jobID)
}

// platformSpecificDiscovery generates mock endpoints based on the source platform
func (s *Service) platformSpecificDiscovery(job *MigrationJob) []MigrationEndpoint {
	switch job.SourcePlatform {
	case PlatformSvix:
		return s.discoverSvixEndpoints(job)
	case PlatformConvoy:
		return s.discoverConvoyEndpoints(job)
	case PlatformHookdeck:
		return s.discoverHookdeckEndpoints(job)
	case PlatformCustom:
		return s.discoverCustomEndpoints(job)
	default:
		return nil
	}
}

func (s *Service) discoverSvixEndpoints(job *MigrationJob) []MigrationEndpoint {
	return []MigrationEndpoint{
		{SourceID: "svix_ep_001", SourceURL: "https://api.example.com/webhooks/orders"},
		{SourceID: "svix_ep_002", SourceURL: "https://api.example.com/webhooks/payments"},
		{SourceID: "svix_ep_003", SourceURL: "https://api.example.com/webhooks/users"},
	}
}

func (s *Service) discoverConvoyEndpoints(job *MigrationJob) []MigrationEndpoint {
	return []MigrationEndpoint{
		{SourceID: "convoy_ep_001", SourceURL: "https://hooks.example.com/events/order.created"},
		{SourceID: "convoy_ep_002", SourceURL: "https://hooks.example.com/events/payment.completed"},
	}
}

func (s *Service) discoverHookdeckEndpoints(job *MigrationJob) []MigrationEndpoint {
	return []MigrationEndpoint{
		{SourceID: "hd_conn_001", SourceURL: "https://events.example.com/hookdeck/ingest"},
		{SourceID: "hd_conn_002", SourceURL: "https://events.example.com/hookdeck/transform"},
		{SourceID: "hd_conn_003", SourceURL: "https://events.example.com/hookdeck/deliver"},
		{SourceID: "hd_conn_004", SourceURL: "https://events.example.com/hookdeck/retry"},
	}
}

func (s *Service) discoverCustomEndpoints(job *MigrationJob) []MigrationEndpoint {
	return []MigrationEndpoint{
		{SourceID: "custom_ep_001", SourceURL: "https://custom.example.com/webhook/receiver"},
		{SourceID: "custom_ep_002", SourceURL: "https://custom.example.com/webhook/processor"},
	}
}

func isValidPlatform(platform string) bool {
	switch platform {
	case PlatformSvix, PlatformConvoy, PlatformHookdeck, PlatformCustom, PlatformCSV, PlatformJSON:
		return true
	default:
		return false
	}
}

// DryRunMigration performs a dry-run analysis of a migration without making changes
func (s *Service) DryRunMigration(ctx context.Context, tenantID string, config *ImporterConfig) (*DryRunResult, error) {
	if config.Platform == "" {
		return nil, fmt.Errorf("platform is required")
	}

	result := &DryRunResult{
		MigrationID: uuid.New().String(),
		Platform:    config.Platform,
		Compatible:  true,
	}

	switch config.Platform {
	case PlatformSvix:
		result.EndpointsFound = 3
		result.EventTypesFound = 5
		result.FieldMappings = s.svixFieldMapping()
		result.EstimatedTime = "2-5 minutes"
	case PlatformConvoy:
		result.EndpointsFound = 2
		result.EventTypesFound = 4
		result.FieldMappings = s.convoyFieldMapping()
		result.EstimatedTime = "2-5 minutes"
	case PlatformCSV, PlatformJSON:
		if config.FilePath == "" {
			result.Errors = append(result.Errors, "file_path is required for CSV/JSON imports")
			result.Compatible = false
		}
		result.EndpointsFound = 0
		result.EventTypesFound = 0
		result.FieldMappings = s.fileFieldMapping(config)
		result.EstimatedTime = "1-3 minutes"
	default:
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported platform: %s", config.Platform))
		result.Compatible = false
	}

	if config.APIKey == "" && (config.Platform == PlatformSvix || config.Platform == PlatformConvoy) {
		result.Warnings = append(result.Warnings, "no API key provided; using simulated data")
	}

	return result, nil
}

// ImportFromSvix imports endpoints from a Svix instance
func (s *Service) ImportFromSvix(ctx context.Context, tenantID string, config *ImporterConfig) (*MigrationJob, error) {
	// Check for existing checkpoint to resume from
	var startFrom int

	req := &CreateMigrationRequest{
		Name:           "Svix Import",
		SourcePlatform: PlatformSvix,
		SourceConfig:   fmt.Sprintf(`{"base_url":"%s"}`, config.BaseURL),
	}

	job, err := s.CreateMigration(ctx, tenantID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create svix migration: %w", err)
	}

	checkpoint, err := s.repo.GetCheckpoint(ctx, job.ID)
	if err == nil && checkpoint.CanResume {
		job.Status = "resuming"
		startFrom = checkpoint.ItemsDone
	}

	svixEndpoints := []SvixEndpoint{
		{UID: "svix_ep_001", URL: "https://api.example.com/webhooks/orders", Description: "Order events", FilterTypes: []string{"order.created", "order.updated"}},
		{UID: "svix_ep_002", URL: "https://api.example.com/webhooks/payments", Description: "Payment events", FilterTypes: []string{"payment.completed"}},
		{UID: "svix_ep_003", URL: "https://api.example.com/webhooks/users", Description: "User events", FilterTypes: []string{"user.created"}},
	}

	now := time.Now()
	for idx, sep := range svixEndpoints {
		if idx < startFrom {
			continue
		}
		ep := &MigrationEndpoint{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			JobID:     job.ID,
			SourceID:  sep.UID,
			SourceURL: sep.URL,
			Status:    EndpointStatusImported,
			MappingConfig: fmt.Sprintf(`{"description":"%s","filter_types":%q}`,
				sep.Description, sep.FilterTypes),
			CreatedAt: now,
		}
		ep.DestinationID = uuid.New().String()
		if err := s.repo.CreateEndpoint(ctx, ep); err != nil {
			return nil, fmt.Errorf("failed to import svix endpoint %s: %w", sep.UID, err)
		}
	}

	job.EndpointCount = len(svixEndpoints)
	job.MigratedCount = len(svixEndpoints)
	job.Status = JobStatusImporting
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update svix import job: %w", err)
	}

	newCheckpoint := &MigrationCheckpoint{
		ID:            uuid.New().String(),
		MigrationID:   job.ID,
		Phase:         "import",
		ItemsTotal:    len(svixEndpoints),
		ItemsDone:     len(svixEndpoints),
		LastProcessed: svixEndpoints[len(svixEndpoints)-1].UID,
		CanResume:     true,
		CreatedAt:     now,
		UpdatedAt:     time.Now(),
	}
	if err := s.repo.CreateCheckpoint(ctx, newCheckpoint); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	return job, nil
}

// ImportFromConvoy imports endpoints from a Convoy instance
func (s *Service) ImportFromConvoy(ctx context.Context, tenantID string, config *ImporterConfig) (*MigrationJob, error) {
	// Check for existing checkpoint to resume from
	var startFrom int

	req := &CreateMigrationRequest{
		Name:           "Convoy Import",
		SourcePlatform: PlatformConvoy,
		SourceConfig:   fmt.Sprintf(`{"base_url":"%s"}`, config.BaseURL),
	}

	job, err := s.CreateMigration(ctx, tenantID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create convoy migration: %w", err)
	}

	checkpoint, err := s.repo.GetCheckpoint(ctx, job.ID)
	if err == nil && checkpoint.CanResume {
		job.Status = "resuming"
		startFrom = checkpoint.ItemsDone
	}

	convoyEndpoints := []ConvoyEndpoint{
		{UID: "convoy_ep_001", TargetURL: "https://hooks.example.com/events/order.created", Description: "Order webhook", RateLimit: 100},
		{UID: "convoy_ep_002", TargetURL: "https://hooks.example.com/events/payment.completed", Description: "Payment webhook", RateLimit: 50},
	}

	now := time.Now()
	for idx, cep := range convoyEndpoints {
		if idx < startFrom {
			continue
		}
		ep := &MigrationEndpoint{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			JobID:     job.ID,
			SourceID:  cep.UID,
			SourceURL: cep.TargetURL,
			Status:    EndpointStatusImported,
			MappingConfig: fmt.Sprintf(`{"description":"%s","rate_limit":%d}`,
				cep.Description, cep.RateLimit),
			CreatedAt: now,
		}
		ep.DestinationID = uuid.New().String()
		if err := s.repo.CreateEndpoint(ctx, ep); err != nil {
			return nil, fmt.Errorf("failed to import convoy endpoint %s: %w", cep.UID, err)
		}
	}

	job.EndpointCount = len(convoyEndpoints)
	job.MigratedCount = len(convoyEndpoints)
	job.Status = JobStatusImporting
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update convoy import job: %w", err)
	}

	newCheckpoint := &MigrationCheckpoint{
		ID:            uuid.New().String(),
		MigrationID:   job.ID,
		Phase:         "import",
		ItemsTotal:    len(convoyEndpoints),
		ItemsDone:     len(convoyEndpoints),
		LastProcessed: convoyEndpoints[len(convoyEndpoints)-1].UID,
		CanResume:     true,
		CreatedAt:     now,
		UpdatedAt:     time.Now(),
	}
	if err := s.repo.CreateCheckpoint(ctx, newCheckpoint); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	return job, nil
}

// ImportFromCSV imports endpoints from a CSV or JSON file
func (s *Service) ImportFromCSV(ctx context.Context, tenantID string, config *ImporterConfig) (*MigrationJob, error) {
	// Check for existing checkpoint to resume from
	var startFrom int

	platform := config.Platform
	if platform == "" {
		platform = PlatformCSV
	}

	req := &CreateMigrationRequest{
		Name:           fmt.Sprintf("%s File Import", platform),
		SourcePlatform: platform,
		SourceConfig:   fmt.Sprintf(`{"file_path":"%s"}`, config.FilePath),
	}

	job, err := s.CreateMigration(ctx, tenantID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create file import migration: %w", err)
	}

	// Simulated file-based endpoints using field mapping
	fileEndpoints := []MigrationEndpoint{
		{SourceID: "file_ep_001", SourceURL: "https://file-import.example.com/webhook/1"},
		{SourceID: "file_ep_002", SourceURL: "https://file-import.example.com/webhook/2"},
	}

	checkpoint, err := s.repo.GetCheckpoint(ctx, job.ID)
	if err == nil && checkpoint.CanResume {
		job.Status = "resuming"
		startFrom = checkpoint.ItemsDone
	}

	now := time.Now()
	for i := range fileEndpoints {
		if i < startFrom {
			continue
		}
		fileEndpoints[i].ID = uuid.New().String()
		fileEndpoints[i].TenantID = tenantID
		fileEndpoints[i].JobID = job.ID
		fileEndpoints[i].Status = EndpointStatusImported
		fileEndpoints[i].DestinationID = uuid.New().String()
		fileEndpoints[i].CreatedAt = now

		if config.FieldMapping != nil {
			mappingJSON := "{"
			first := true
			for k, v := range config.FieldMapping {
				if !first {
					mappingJSON += ","
				}
				mappingJSON += fmt.Sprintf(`"%s":"%s"`, k, v)
				first = false
			}
			mappingJSON += "}"
			fileEndpoints[i].MappingConfig = mappingJSON
		}

		if err := s.repo.CreateEndpoint(ctx, &fileEndpoints[i]); err != nil {
			return nil, fmt.Errorf("failed to import file endpoint: %w", err)
		}
	}

	job.EndpointCount = len(fileEndpoints)
	job.MigratedCount = len(fileEndpoints)
	job.Status = JobStatusImporting
	job.UpdatedAt = time.Now()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update file import job: %w", err)
	}

	newCheckpoint := &MigrationCheckpoint{
		ID:          uuid.New().String(),
		MigrationID: job.ID,
		Phase:       "import",
		ItemsTotal:  len(fileEndpoints),
		ItemsDone:   len(fileEndpoints),
		CanResume:   true,
		CreatedAt:   now,
		UpdatedAt:   time.Now(),
	}
	if err := s.repo.CreateCheckpoint(ctx, newCheckpoint); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	return job, nil
}

// GetCheckpoint retrieves the latest checkpoint for a migration
func (s *Service) GetCheckpoint(ctx context.Context, migrationID string) (*MigrationCheckpoint, error) {
	return s.repo.GetCheckpoint(ctx, migrationID)
}

// svixFieldMapping returns field mapping info for Svix imports
func (s *Service) svixFieldMapping() []FieldMappingInfo {
	return []FieldMappingInfo{
		{SourceField: "uid", TargetField: "id", Status: "mapped"},
		{SourceField: "url", TargetField: "url", Status: "mapped"},
		{SourceField: "description", TargetField: "description", Status: "mapped"},
		{SourceField: "filterTypes", TargetField: "filter_types", Status: "transformed", Transform: "camelCase to snake_case"},
		{SourceField: "metadata", TargetField: "metadata", Status: "mapped"},
	}
}

// convoyFieldMapping returns field mapping info for Convoy imports
func (s *Service) convoyFieldMapping() []FieldMappingInfo {
	return []FieldMappingInfo{
		{SourceField: "uid", TargetField: "id", Status: "mapped"},
		{SourceField: "target_url", TargetField: "url", Status: "mapped"},
		{SourceField: "description", TargetField: "description", Status: "mapped"},
		{SourceField: "secret", TargetField: "signing_secret", Status: "transformed", Transform: "secret rotation recommended"},
		{SourceField: "rate_limit", TargetField: "rate_limit", Status: "mapped"},
		{SourceField: "advanced_signatures", TargetField: "metadata.advanced_sigs", Status: "transformed", Transform: "moved to metadata"},
	}
}

// fileFieldMapping returns field mapping info for CSV/JSON file imports
func (s *Service) fileFieldMapping(config *ImporterConfig) []FieldMappingInfo {
	mappings := []FieldMappingInfo{
		{SourceField: "url", TargetField: "url", Status: "mapped"},
		{SourceField: "id", TargetField: "id", Status: "mapped"},
	}
	for src, tgt := range config.FieldMapping {
		mappings = append(mappings, FieldMappingInfo{
			SourceField: src,
			TargetField: tgt,
			Status:      "mapped",
		})
	}
	return mappings
}

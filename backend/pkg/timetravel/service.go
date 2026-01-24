package timetravel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides time-travel business logic
type Service struct {
	repo Repository
}

// NewService creates a new time-travel service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RecordEvent records a webhook event for time-travel capability
func (s *Service) RecordEvent(ctx context.Context, tenantID string, event *EventRecord) (*EventRecord, error) {
	event.ID = uuid.New().String()
	event.TenantID = tenantID
	event.Timestamp = time.Now()

	// Compute checksum from payload
	hash := sha256.Sum256(event.Payload)
	event.Checksum = hex.EncodeToString(hash[:])

	if err := s.repo.RecordEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to record event: %w", err)
	}

	return event, nil
}

// GetEventTimeline retrieves events with filters and pagination
func (s *Service) GetEventTimeline(ctx context.Context, tenantID string, filters *ReplayFilter, page, pageSize int) ([]EventRecord, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.repo.GetEvents(ctx, tenantID, filters, page, pageSize)
}

// CreateReplayJob creates a new replay job with validation
func (s *Service) CreateReplayJob(ctx context.Context, tenantID string, req *CreateReplayJobRequest) (*ReplayJob, error) {
	if req.TimeWindow.Start.IsZero() || req.TimeWindow.End.IsZero() {
		return nil, fmt.Errorf("time_window start and end are required")
	}
	if req.TimeWindow.End.Before(req.TimeWindow.Start) {
		return nil, fmt.Errorf("time_window end must be after start")
	}

	now := time.Now()
	job := &ReplayJob{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Status:         ReplayJobStatusPending,
		TimeWindow:     req.TimeWindow,
		Filters:        req.Filters,
		TargetEndpoint: req.TargetEndpoint,
		DryRun:         req.DryRun,
		Progress:       0,
		Results:        ReplayResult{},
		CreatedAt:      now,
	}

	if err := s.repo.CreateReplayJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create replay job: %w", err)
	}

	return job, nil
}

// ExecuteReplay executes a replay job by iterating events and simulating replay
func (s *Service) ExecuteReplay(ctx context.Context, tenantID, jobID string) (*ReplayJob, error) {
	job, err := s.repo.GetReplayJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get replay job: %w", err)
	}
	if job == nil {
		return nil, fmt.Errorf("replay job not found: %s", jobID)
	}

	if job.Status != ReplayJobStatusPending {
		return nil, fmt.Errorf("replay job is not in pending status: %s", job.Status)
	}

	// Mark as running
	job.Status = ReplayJobStatusRunning
	if err := s.repo.UpdateReplayJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update replay job: %w", err)
	}

	startTime := time.Now()

	// Build filters from job configuration
	filters := &job.Filters
	if !job.TimeWindow.Start.IsZero() || !job.TimeWindow.End.IsZero() {
		filters.TimeWindow = job.TimeWindow
	}

	// Fetch all matching events
	events, total, err := s.repo.GetEvents(ctx, tenantID, filters, 1, 10000)
	if err != nil {
		job.Status = ReplayJobStatusFailed
		s.repo.UpdateReplayJob(ctx, job)
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	job.Results.TotalEvents = total

	// Iterate and simulate replay
	for i, event := range events {
		select {
		case <-ctx.Done():
			job.Status = ReplayJobStatusCancelled
			s.repo.UpdateReplayJob(ctx, job)
			return job, ctx.Err()
		default:
		}

		if job.DryRun {
			job.Results.Skipped++
		} else {
			// Simulate replay: in production this would re-deliver the event
			_ = event
			job.Results.Replayed++
			job.Results.Succeeded++
		}

		job.Progress = ((i + 1) * 100) / len(events)
	}

	job.Results.DurationMs = time.Since(startTime).Milliseconds()
	job.Status = ReplayJobStatusCompleted
	job.Progress = 100

	if err := s.repo.UpdateReplayJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update replay job: %w", err)
	}

	return job, nil
}

// CancelReplay cancels a running replay job
func (s *Service) CancelReplay(ctx context.Context, tenantID, jobID string) (*ReplayJob, error) {
	job, err := s.repo.GetReplayJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get replay job: %w", err)
	}
	if job == nil {
		return nil, fmt.Errorf("replay job not found: %s", jobID)
	}

	if job.Status != ReplayJobStatusPending && job.Status != ReplayJobStatusRunning {
		return nil, fmt.Errorf("cannot cancel replay job in status: %s", job.Status)
	}

	job.Status = ReplayJobStatusCancelled
	if err := s.repo.UpdateReplayJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to cancel replay job: %w", err)
	}

	return job, nil
}

// CreateSnapshot creates a point-in-time snapshot
func (s *Service) CreateSnapshot(ctx context.Context, tenantID string, req *CreateSnapshotRequest) (*PointInTimeSnapshot, error) {
	// Count events up to the snapshot time point
	filters := &ReplayFilter{
		TimeWindow: TimeWindow{End: req.TimePoint},
	}
	_, eventCount, err := s.repo.GetEvents(ctx, tenantID, filters, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	snapshot := &PointInTimeSnapshot{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		TimePoint:   req.TimePoint,
		EventCount:  eventCount,
		SizeBytes:   int64(eventCount) * 1024, // Estimated size
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot by ID
func (s *Service) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*PointInTimeSnapshot, error) {
	snapshot, err := s.repo.GetSnapshot(ctx, tenantID, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found")
	}
	return snapshot, nil
}

// ListSnapshots lists snapshots for a tenant
func (s *Service) ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]PointInTimeSnapshot, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListSnapshots(ctx, tenantID, limit, offset)
}

// AnalyzeBlastRadius calculates the impact of an endpoint going down
func (s *Service) AnalyzeBlastRadius(ctx context.Context, tenantID, endpointID string) (*BlastRadiusAnalysis, error) {
	// Get events for this endpoint
	filters := &ReplayFilter{
		EndpointIDs: []string{endpointID},
	}
	events, total, err := s.repo.GetEvents(ctx, tenantID, filters, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze blast radius: %w", err)
	}

	// Determine affected endpoints by looking at event types shared with other endpoints
	affectedEndpoints := make(map[string]bool)
	eventTypes := make(map[string]bool)
	for _, event := range events {
		eventTypes[event.EventType] = true
	}

	// Query for other endpoints sharing the same event types
	for eventType := range eventTypes {
		typeFilters := &ReplayFilter{
			EventTypes: []string{eventType},
		}
		relatedEvents, _, err := s.repo.GetEvents(ctx, tenantID, typeFilters, 1, 1000)
		if err != nil {
			continue
		}
		for _, e := range relatedEvents {
			if e.EndpointID != endpointID {
				affectedEndpoints[e.EndpointID] = true
			}
		}
	}

	affected := make([]string, 0, len(affectedEndpoints))
	for ep := range affectedEndpoints {
		affected = append(affected, ep)
	}

	// Calculate impact score (0-100)
	impactScore := float64(total) * 0.1
	if impactScore > 100 {
		impactScore = 100
	}
	impactScore += float64(len(affected)) * 10
	if impactScore > 100 {
		impactScore = 100
	}

	recommendation := "low impact; no immediate action required"
	if impactScore > 70 {
		recommendation = "critical impact; immediate failover recommended"
	} else if impactScore > 40 {
		recommendation = "moderate impact; monitor closely and prepare failover"
	}

	return &BlastRadiusAnalysis{
		EndpointID:        endpointID,
		AffectedEvents:    total,
		AffectedEndpoints: affected,
		ImpactScore:       impactScore,
		Recommendation:    recommendation,
	}, nil
}

// RunWhatIfScenario compares original vs modified payload and saves the analysis
func (s *Service) RunWhatIfScenario(ctx context.Context, tenantID string, req *WhatIfRequest) (*WhatIfScenario, error) {
	// Parse payloads for comparison
	var original, modified map[string]interface{}
	if err := json.Unmarshal(req.OriginalPayload, &original); err != nil {
		return nil, fmt.Errorf("invalid original payload: %w", err)
	}
	if err := json.Unmarshal(req.ModifiedPayload, &modified); err != nil {
		return nil, fmt.Errorf("invalid modified payload: %w", err)
	}

	// Build diff summary
	diffSummary := buildDiffSummary(original, modified)

	scenario := &WhatIfScenario{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		OriginalPayload: req.OriginalPayload,
		ModifiedPayload: req.ModifiedPayload,
		DiffSummary:     diffSummary,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.SaveWhatIfScenario(ctx, scenario); err != nil {
		return nil, fmt.Errorf("failed to save what-if scenario: %w", err)
	}

	return scenario, nil
}

// buildDiffSummary generates a human-readable summary of differences between two payloads
func buildDiffSummary(original, modified map[string]interface{}) string {
	added, removed, changed := 0, 0, 0

	for key := range modified {
		origVal, exists := original[key]
		if !exists {
			added++
		} else {
			origJSON, _ := json.Marshal(origVal)
			modJSON, _ := json.Marshal(modified[key])
			if string(origJSON) != string(modJSON) {
				changed++
			}
		}
	}

	for key := range original {
		if _, exists := modified[key]; !exists {
			removed++
		}
	}

	return fmt.Sprintf("%d field(s) added, %d removed, %d changed", added, removed, changed)
}

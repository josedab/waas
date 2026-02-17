package services

import (
	"context"
	"crypto/subtle"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// ReplayService handles event replay operations
type ReplayService struct {
	repo        repository.ReplayRepository
	webhookRepo repository.WebhookEndpointRepository
	publisher   interface {
		Publish(ctx context.Context, msg interface{}) error
	}
	logger     *utils.Logger
	mu         sync.RWMutex
	activeJobs map[uuid.UUID]context.CancelFunc
}

// NewReplayService creates a new replay service
func NewReplayService(
	repo repository.ReplayRepository,
	webhookRepo repository.WebhookEndpointRepository,
	logger *utils.Logger,
) *ReplayService {
	return &ReplayService{
		repo:        repo,
		webhookRepo: webhookRepo,
		logger:      logger,
		activeJobs:  make(map[uuid.UUID]context.CancelFunc),
	}
}

// CreateReplayJob creates a new replay job
func (s *ReplayService) CreateReplayJob(ctx context.Context, tenantID uuid.UUID, req *models.CreateReplayJobRequest) (*models.ReplayJob, error) {
	// Validate time range
	if req.TimeRangeEnd.Before(req.TimeRangeStart) {
		return nil, fmt.Errorf("time_range_end must be after time_range_start")
	}

	// Count events in range
	eventCount, err := s.repo.CountEventsByTimeRange(ctx, tenantID, req.TimeRangeStart, req.TimeRangeEnd, &req.FilterCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	if eventCount == 0 {
		return nil, fmt.Errorf("no events found in specified time range")
	}

	var targetEndpointID *uuid.UUID
	if req.TargetEndpointID != "" {
		id, err := uuid.Parse(req.TargetEndpointID)
		if err != nil {
			return nil, fmt.Errorf("invalid target_endpoint_id")
		}
		targetEndpointID = &id
	}

	job := &models.ReplayJob{
		TenantID:         tenantID,
		Name:             req.Name,
		Description:      req.Description,
		TimeRangeStart:   req.TimeRangeStart,
		TimeRangeEnd:     req.TimeRangeEnd,
		FilterCriteria:   req.FilterCriteria,
		TargetEndpointID: targetEndpointID,
		Options:          req.Options,
		TotalEvents:      eventCount,
	}

	if err := s.repo.CreateReplayJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create replay job: %w", err)
	}

	return job, nil
}

// StartReplayJob starts executing a replay job
func (s *ReplayService) StartReplayJob(ctx context.Context, tenantID, jobID uuid.UUID) error {
	job, err := s.repo.GetReplayJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	if job.TenantID != tenantID {
		return fmt.Errorf("job not found")
	}

	if job.Status != models.ReplayStatusPending && job.Status != models.ReplayStatusPaused {
		return fmt.Errorf("job cannot be started in status: %s", job.Status)
	}

	// Load events to replay
	events, err := s.repo.GetEventsByTimeRange(ctx, tenantID, job.TimeRangeStart, job.TimeRangeEnd, job.TotalEvents, 0)
	if err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Create replay job events
	var replayEvents []*models.ReplayJobEvent
	for _, event := range events {
		// Apply filters
		if !s.matchesFilter(event, &job.FilterCriteria) {
			continue
		}

		replayEvents = append(replayEvents, &models.ReplayJobEvent{
			ReplayJobID:     jobID,
			EventArchiveID:  event.ID,
			OriginalPayload: event.Payload,
		})
	}

	if err := s.repo.CreateReplayJobEvents(ctx, replayEvents); err != nil {
		return fmt.Errorf("failed to create replay events: %w", err)
	}

	// Update job status
	if err := s.repo.UpdateReplayJobStatus(ctx, jobID, models.ReplayStatusRunning, ""); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Start processing in background
	jobCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.activeJobs[jobID] = cancel
	s.mu.Unlock()

	go s.processReplayJob(jobCtx, job)

	return nil
}

// processReplayJob processes events for a replay job
func (s *ReplayService) processReplayJob(ctx context.Context, job *models.ReplayJob) {
	defer func() {
		s.mu.Lock()
		delete(s.activeJobs, job.ID)
		s.mu.Unlock()
	}()

	batchSize := job.Options.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}

	rateLimit := job.Options.RateLimit
	if rateLimit == 0 {
		rateLimit = 1000 // default 1000 events/sec
	}

	ticker := time.NewTicker(time.Second / time.Duration(rateLimit))
	defer ticker.Stop()

	var processed, successful, failed int

	for {
		select {
		case <-ctx.Done():
			s.repo.UpdateReplayJobStatus(context.Background(), job.ID, models.ReplayStatusCancelled, "Job cancelled")
			return
		default:
		}

		// Get next batch of pending events
		events, err := s.repo.GetPendingReplayEvents(ctx, job.ID, batchSize)
		if err != nil {
			s.logger.Error("Failed to get pending events", map[string]interface{}{"error": err, "job_id": job.ID})
			continue
		}

		if len(events) == 0 {
			// All events processed
			s.repo.UpdateReplayJobStatus(context.Background(), job.ID, models.ReplayStatusCompleted, "")
			return
		}

		for _, event := range events {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			// Process the event
			if err := s.processReplayEvent(ctx, job, event); err != nil {
				event.Status = models.ReplayEventFailed
				event.ErrorMessage = err.Error()
				failed++

				if job.Options.StopOnError {
					s.repo.UpdateReplayJobEvent(ctx, event)
					s.repo.UpdateReplayJobStatus(ctx, job.ID, models.ReplayStatusFailed, err.Error())
					return
				}
			} else {
				event.Status = models.ReplayEventSuccess
				successful++
			}

			s.repo.UpdateReplayJobEvent(ctx, event)
			processed++

			// Update progress periodically
			if processed%100 == 0 {
				s.repo.UpdateReplayJobProgress(ctx, job.ID, processed, successful, failed)
			}
		}
	}
}

// processReplayEvent processes a single replay event
func (s *ReplayService) processReplayEvent(ctx context.Context, job *models.ReplayJob, event *models.ReplayJobEvent) error {
	payload := event.OriginalPayload

	// Apply transformation if specified
	if job.Options.TransformationCode != "" {
		transformed, err := s.applyTransformation(payload, job.Options.TransformationCode)
		if err != nil {
			return fmt.Errorf("transformation failed: %w", err)
		}
		event.TransformedPayload = transformed
		payload = transformed
	}

	// If dry run, don't actually send
	if job.Options.DryRun {
		return nil
	}

	// Send webhook (would integrate with delivery engine)
	// For now, simulate successful delivery

	return nil
}

// applyTransformation applies JavaScript transformation to payload
func (s *ReplayService) applyTransformation(payload map[string]interface{}, code string) (map[string]interface{}, error) {
	// Would use goja VM here - simplified for now
	return payload, nil
}

// matchesFilter checks if an event matches the filter criteria
func (s *ReplayService) matchesFilter(event *models.EventArchive, filter *models.ReplayFilterCriteria) bool {
	if filter == nil {
		return true
	}

	// Check event type
	if len(filter.EventTypes) > 0 {
		found := false
		for _, t := range filter.EventTypes {
			if t == event.EventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check endpoint
	if len(filter.EndpointIDs) > 0 && event.EndpointID != nil {
		found := false
		for _, id := range filter.EndpointIDs {
			if id == *event.EndpointID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check excluded hashes (constant-time comparison to prevent timing attacks)
	for _, hash := range filter.ExcludeHashes {
		if subtle.ConstantTimeCompare([]byte(hash), []byte(event.PayloadHash)) == 1 {
			return false
		}
	}

	return true
}

// PauseReplayJob pauses a running replay job
func (s *ReplayService) PauseReplayJob(ctx context.Context, tenantID, jobID uuid.UUID) error {
	job, err := s.repo.GetReplayJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.TenantID != tenantID {
		return fmt.Errorf("job not found")
	}

	if job.Status != models.ReplayStatusRunning {
		return fmt.Errorf("can only pause running jobs")
	}

	// Cancel the job context
	s.mu.Lock()
	if cancel, ok := s.activeJobs[jobID]; ok {
		cancel()
	}
	s.mu.Unlock()

	return s.repo.UpdateReplayJobStatus(ctx, jobID, models.ReplayStatusPaused, "")
}

// CancelReplayJob cancels a replay job
func (s *ReplayService) CancelReplayJob(ctx context.Context, tenantID, jobID uuid.UUID) error {
	job, err := s.repo.GetReplayJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.TenantID != tenantID {
		return fmt.Errorf("job not found")
	}

	// Cancel the job context
	s.mu.Lock()
	if cancel, ok := s.activeJobs[jobID]; ok {
		cancel()
	}
	s.mu.Unlock()

	return s.repo.UpdateReplayJobStatus(ctx, jobID, models.ReplayStatusCancelled, "Cancelled by user")
}

// GetReplayJobProgress returns current progress of a replay job
func (s *ReplayService) GetReplayJobProgress(ctx context.Context, tenantID, jobID uuid.UUID) (*models.ReplayJobProgress, error) {
	job, err := s.repo.GetReplayJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if job.TenantID != tenantID {
		return nil, fmt.Errorf("job not found")
	}

	progress := &models.ReplayJobProgress{
		JobID:            job.ID,
		Status:           job.Status,
		TotalEvents:      job.TotalEvents,
		ProcessedEvents:  job.ProcessedEvents,
		SuccessfulEvents: job.SuccessfulEvents,
		FailedEvents:     job.FailedEvents,
	}

	if job.TotalEvents > 0 {
		progress.ProgressPercent = float64(job.ProcessedEvents) / float64(job.TotalEvents) * 100
	}

	// Estimate time remaining
	if job.StartedAt != nil && job.ProcessedEvents > 0 {
		elapsed := time.Since(*job.StartedAt)
		eventsPerSecond := float64(job.ProcessedEvents) / elapsed.Seconds()
		remaining := float64(job.TotalEvents-job.ProcessedEvents) / eventsPerSecond
		progress.EstimatedTimeRemaining = time.Duration(remaining * float64(time.Second)).String()
	}

	return progress, nil
}

// SearchEvents searches archived events
func (s *ReplayService) SearchEvents(ctx context.Context, tenantID uuid.UUID, req *models.EventSearchRequest) ([]*models.EventArchive, int, error) {
	return s.repo.SearchEvents(ctx, tenantID, req)
}

// CreateSnapshot creates a point-in-time snapshot
func (s *ReplayService) CreateSnapshot(ctx context.Context, tenantID uuid.UUID, req *models.CreateSnapshotRequest) (*models.ReplaySnapshot, error) {
	// Count events for snapshot
	eventCount, err := s.repo.CountEventsByTimeRange(ctx, tenantID, time.Time{}, req.SnapshotTime, &req.FilterCriteria)
	if err != nil {
		return nil, err
	}

	snapshot := &models.ReplaySnapshot{
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		SnapshotTime:   req.SnapshotTime,
		FilterCriteria: req.FilterCriteria,
		EventCount:     eventCount,
	}

	if req.ExpiresInDays > 0 {
		expiry := time.Now().AddDate(0, 0, req.ExpiresInDays)
		snapshot.ExpiresAt = &expiry
	}

	if err := s.repo.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

// GetSnapshots returns all snapshots for a tenant
func (s *ReplayService) GetSnapshots(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplaySnapshot, error) {
	return s.repo.GetSnapshots(ctx, tenantID)
}

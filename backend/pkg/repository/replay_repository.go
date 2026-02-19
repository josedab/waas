package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/models"
)

// ReplayRepository defines data access operations for the event replay system,
// including event archival, replay job management, snapshots, and comparisons.
type ReplayRepository interface {
	// Event archive operations
	ArchiveEvent(ctx context.Context, event *models.EventArchive) error
	GetArchivedEvent(ctx context.Context, id uuid.UUID) (*models.EventArchive, error)
	SearchEvents(ctx context.Context, tenantID uuid.UUID, req *models.EventSearchRequest) ([]*models.EventArchive, int, error)
	GetEventsByTimeRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit, offset int) ([]*models.EventArchive, error)
	CountEventsByTimeRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter *models.ReplayFilterCriteria) (int, error)
	DeleteOldEvents(ctx context.Context, tenantID uuid.UUID, before time.Time) (int64, error)

	// Replay job operations
	CreateReplayJob(ctx context.Context, job *models.ReplayJob) error
	GetReplayJob(ctx context.Context, id uuid.UUID) (*models.ReplayJob, error)
	GetReplayJobs(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]*models.ReplayJob, error)
	UpdateReplayJob(ctx context.Context, job *models.ReplayJob) error
	UpdateReplayJobProgress(ctx context.Context, id uuid.UUID, processed, successful, failed int) error
	UpdateReplayJobStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error

	// Replay job event operations
	CreateReplayJobEvents(ctx context.Context, events []*models.ReplayJobEvent) error
	GetReplayJobEvents(ctx context.Context, jobID uuid.UUID, status string, limit, offset int) ([]*models.ReplayJobEvent, error)
	UpdateReplayJobEvent(ctx context.Context, event *models.ReplayJobEvent) error
	GetPendingReplayEvents(ctx context.Context, jobID uuid.UUID, batchSize int) ([]*models.ReplayJobEvent, error)

	// Snapshot operations
	CreateSnapshot(ctx context.Context, snapshot *models.ReplaySnapshot) error
	GetSnapshot(ctx context.Context, id uuid.UUID) (*models.ReplaySnapshot, error)
	GetSnapshots(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplaySnapshot, error)
	DeleteSnapshot(ctx context.Context, id uuid.UUID) error

	// Comparison operations
	CreateComparison(ctx context.Context, comparison *models.ReplayComparison) error
	GetComparison(ctx context.Context, id uuid.UUID) (*models.ReplayComparison, error)
	UpdateComparison(ctx context.Context, comparison *models.ReplayComparison) error
}

type replayRepository struct {
	db *database.DB
}

// NewReplayRepository creates a new ReplayRepository backed by PostgreSQL.
func NewReplayRepository(db *database.DB) ReplayRepository {
	return &replayRepository{db: db}
}

// HashPayload generates a SHA256 hash of the payload
func HashPayload(payload map[string]interface{}) string {
	data, _ := json.Marshal(payload)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (r *replayRepository) ArchiveEvent(ctx context.Context, event *models.EventArchive) error {
	query := `
		INSERT INTO event_archive (id, tenant_id, endpoint_id, event_type, payload, payload_hash, headers, source_ip, received_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now()
	}
	event.PayloadHash = HashPayload(event.Payload)

	payloadJSON, _ := json.Marshal(event.Payload)
	headersJSON, _ := json.Marshal(event.Headers)
	metadataJSON, _ := json.Marshal(event.Metadata)

	_, err := r.db.Pool.Exec(ctx, query,
		event.ID, event.TenantID, event.EndpointID, event.EventType,
		payloadJSON, event.PayloadHash, headersJSON, event.SourceIP, event.ReceivedAt, metadataJSON)
	return err
}

func (r *replayRepository) GetArchivedEvent(ctx context.Context, id uuid.UUID) (*models.EventArchive, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, event_type, payload, payload_hash, headers, source_ip, received_at, metadata
		FROM event_archive WHERE id = $1
	`

	var event models.EventArchive
	var payloadJSON, headersJSON, metadataJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&event.ID, &event.TenantID, &event.EndpointID, &event.EventType,
		&payloadJSON, &event.PayloadHash, &headersJSON, &event.SourceIP, &event.ReceivedAt, &metadataJSON)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(payloadJSON, &event.Payload)
	json.Unmarshal(headersJSON, &event.Headers)
	json.Unmarshal(metadataJSON, &event.Metadata)

	return &event, nil
}

func (r *replayRepository) SearchEvents(ctx context.Context, tenantID uuid.UUID, req *models.EventSearchRequest) ([]*models.EventArchive, int, error) {
	baseQuery := `FROM event_archive WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if !req.TimeRangeStart.IsZero() {
		baseQuery += fmt.Sprintf(" AND received_at >= $%d", argIdx)
		args = append(args, req.TimeRangeStart)
		argIdx++
	}
	if !req.TimeRangeEnd.IsZero() {
		baseQuery += fmt.Sprintf(" AND received_at <= $%d", argIdx)
		args = append(args, req.TimeRangeEnd)
		argIdx++
	}
	if len(req.EventTypes) > 0 {
		baseQuery += fmt.Sprintf(" AND event_type = ANY($%d)", argIdx)
		args = append(args, req.EventTypes)
		argIdx++
	}
	if req.PayloadQuery != "" {
		baseQuery += fmt.Sprintf(" AND payload::text ILIKE $%d", argIdx)
		args = append(args, "%"+req.PayloadQuery+"%")
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get results
	limit := req.Limit
	if limit == 0 {
		limit = 100
	}
	selectQuery := `SELECT id, tenant_id, endpoint_id, event_type, payload, payload_hash, headers, source_ip, received_at, metadata ` +
		baseQuery + fmt.Sprintf(" ORDER BY received_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, req.Offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*models.EventArchive
	for rows.Next() {
		var event models.EventArchive
		var payloadJSON, headersJSON, metadataJSON []byte

		if err := rows.Scan(&event.ID, &event.TenantID, &event.EndpointID, &event.EventType,
			&payloadJSON, &event.PayloadHash, &headersJSON, &event.SourceIP, &event.ReceivedAt, &metadataJSON); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(payloadJSON, &event.Payload)
		json.Unmarshal(headersJSON, &event.Headers)
		json.Unmarshal(metadataJSON, &event.Metadata)
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *replayRepository) GetEventsByTimeRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit, offset int) ([]*models.EventArchive, error) {
	query := `
		SELECT id, tenant_id, endpoint_id, event_type, payload, payload_hash, headers, source_ip, received_at, metadata
		FROM event_archive 
		WHERE tenant_id = $1 AND received_at BETWEEN $2 AND $3
		ORDER BY received_at ASC
		LIMIT $4 OFFSET $5
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, start, end, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.EventArchive
	for rows.Next() {
		var event models.EventArchive
		var payloadJSON, headersJSON, metadataJSON []byte

		if err := rows.Scan(&event.ID, &event.TenantID, &event.EndpointID, &event.EventType,
			&payloadJSON, &event.PayloadHash, &headersJSON, &event.SourceIP, &event.ReceivedAt, &metadataJSON); err != nil {
			return nil, err
		}

		json.Unmarshal(payloadJSON, &event.Payload)
		json.Unmarshal(headersJSON, &event.Headers)
		json.Unmarshal(metadataJSON, &event.Metadata)
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *replayRepository) CountEventsByTimeRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter *models.ReplayFilterCriteria) (int, error) {
	query := `SELECT COUNT(*) FROM event_archive WHERE tenant_id = $1 AND received_at BETWEEN $2 AND $3`
	args := []interface{}{tenantID, start, end}

	if filter != nil && len(filter.EventTypes) > 0 {
		query += " AND event_type = ANY($4)"
		args = append(args, filter.EventTypes)
	}

	var count int
	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *replayRepository) DeleteOldEvents(ctx context.Context, tenantID uuid.UUID, before time.Time) (int64, error) {
	query := `DELETE FROM event_archive WHERE tenant_id = $1 AND received_at < $2`
	result, err := r.db.Pool.Exec(ctx, query, tenantID, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Replay job operations

func (r *replayRepository) CreateReplayJob(ctx context.Context, job *models.ReplayJob) error {
	query := `
		INSERT INTO replay_jobs (id, tenant_id, name, description, status, filter_criteria, time_range_start, time_range_end, 
		                         target_endpoint_id, transformation_id, options, total_events, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now
	job.Status = models.ReplayStatusPending

	filterJSON, _ := json.Marshal(job.FilterCriteria)
	optionsJSON, _ := json.Marshal(job.Options)

	_, err := r.db.Pool.Exec(ctx, query,
		job.ID, job.TenantID, job.Name, job.Description, job.Status,
		filterJSON, job.TimeRangeStart, job.TimeRangeEnd,
		job.TargetEndpointID, job.TransformationID, optionsJSON,
		job.TotalEvents, job.CreatedBy, job.CreatedAt, job.UpdatedAt)
	return err
}

func (r *replayRepository) GetReplayJob(ctx context.Context, id uuid.UUID) (*models.ReplayJob, error) {
	query := `
		SELECT id, tenant_id, name, description, status, filter_criteria, time_range_start, time_range_end,
		       target_endpoint_id, transformation_id, options, total_events, processed_events, 
		       successful_events, failed_events, started_at, completed_at, error_message, created_by, created_at, updated_at
		FROM replay_jobs WHERE id = $1
	`

	var job models.ReplayJob
	var filterJSON, optionsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.TenantID, &job.Name, &job.Description, &job.Status,
		&filterJSON, &job.TimeRangeStart, &job.TimeRangeEnd,
		&job.TargetEndpointID, &job.TransformationID, &optionsJSON,
		&job.TotalEvents, &job.ProcessedEvents, &job.SuccessfulEvents, &job.FailedEvents,
		&job.StartedAt, &job.CompletedAt, &job.ErrorMessage, &job.CreatedBy, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(filterJSON, &job.FilterCriteria)
	json.Unmarshal(optionsJSON, &job.Options)

	return &job, nil
}

func (r *replayRepository) GetReplayJobs(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]*models.ReplayJob, error) {
	query := `
		SELECT id, tenant_id, name, description, status, filter_criteria, time_range_start, time_range_end,
		       target_endpoint_id, transformation_id, options, total_events, processed_events,
		       successful_events, failed_events, started_at, completed_at, error_message, created_by, created_at, updated_at
		FROM replay_jobs WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.ReplayJob
	for rows.Next() {
		var job models.ReplayJob
		var filterJSON, optionsJSON []byte

		if err := rows.Scan(&job.ID, &job.TenantID, &job.Name, &job.Description, &job.Status,
			&filterJSON, &job.TimeRangeStart, &job.TimeRangeEnd,
			&job.TargetEndpointID, &job.TransformationID, &optionsJSON,
			&job.TotalEvents, &job.ProcessedEvents, &job.SuccessfulEvents, &job.FailedEvents,
			&job.StartedAt, &job.CompletedAt, &job.ErrorMessage, &job.CreatedBy, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(filterJSON, &job.FilterCriteria)
		json.Unmarshal(optionsJSON, &job.Options)
		jobs = append(jobs, &job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *replayRepository) UpdateReplayJob(ctx context.Context, job *models.ReplayJob) error {
	query := `
		UPDATE replay_jobs 
		SET status = $2, processed_events = $3, successful_events = $4, failed_events = $5,
		    started_at = $6, completed_at = $7, error_message = $8, updated_at = $9
		WHERE id = $1
	`
	job.UpdatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, query,
		job.ID, job.Status, job.ProcessedEvents, job.SuccessfulEvents, job.FailedEvents,
		job.StartedAt, job.CompletedAt, job.ErrorMessage, job.UpdatedAt)
	return err
}

func (r *replayRepository) UpdateReplayJobProgress(ctx context.Context, id uuid.UUID, processed, successful, failed int) error {
	query := `UPDATE replay_jobs SET processed_events = $2, successful_events = $3, failed_events = $4, updated_at = $5 WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id, processed, successful, failed, time.Now())
	return err
}

func (r *replayRepository) UpdateReplayJobStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error {
	query := `UPDATE replay_jobs SET status = $2, error_message = $3, updated_at = $4`
	args := []interface{}{id, status, errorMsg, time.Now()}

	if status == models.ReplayStatusRunning {
		query += ", started_at = $5 WHERE id = $1"
		args = append(args, time.Now())
	} else if status == models.ReplayStatusCompleted || status == models.ReplayStatusFailed {
		query += ", completed_at = $5 WHERE id = $1"
		args = append(args, time.Now())
	} else {
		query += " WHERE id = $1"
	}

	_, err := r.db.Pool.Exec(ctx, query, args...)
	return err
}

// Replay job event operations

func (r *replayRepository) CreateReplayJobEvents(ctx context.Context, events []*models.ReplayJobEvent) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO replay_job_events (id, replay_job_id, event_archive_id, status, original_payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, event := range events {
		if event.ID == uuid.Nil {
			event.ID = uuid.New()
		}
		event.CreatedAt = time.Now()
		event.Status = models.ReplayEventPending

		payloadJSON, _ := json.Marshal(event.OriginalPayload)

		if _, err := r.db.Pool.Exec(ctx, query,
			event.ID, event.ReplayJobID, event.EventArchiveID, event.Status, payloadJSON, event.CreatedAt); err != nil {
			return err
		}
	}

	return nil
}

func (r *replayRepository) GetReplayJobEvents(ctx context.Context, jobID uuid.UUID, status string, limit, offset int) ([]*models.ReplayJobEvent, error) {
	query := `
		SELECT id, replay_job_id, event_archive_id, status, original_payload, transformed_payload, 
		       delivery_attempt_id, error_message, processed_at, created_at
		FROM replay_job_events WHERE replay_job_id = $1
	`
	args := []interface{}{jobID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += fmt.Sprintf(" ORDER BY created_at ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.ReplayJobEvent
	for rows.Next() {
		var event models.ReplayJobEvent
		var originalJSON, transformedJSON []byte

		if err := rows.Scan(&event.ID, &event.ReplayJobID, &event.EventArchiveID, &event.Status,
			&originalJSON, &transformedJSON, &event.DeliveryAttemptID, &event.ErrorMessage,
			&event.ProcessedAt, &event.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(originalJSON, &event.OriginalPayload)
		json.Unmarshal(transformedJSON, &event.TransformedPayload)
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *replayRepository) UpdateReplayJobEvent(ctx context.Context, event *models.ReplayJobEvent) error {
	query := `
		UPDATE replay_job_events 
		SET status = $2, transformed_payload = $3, delivery_attempt_id = $4, error_message = $5, processed_at = $6
		WHERE id = $1
	`

	now := time.Now()
	event.ProcessedAt = &now
	transformedJSON, _ := json.Marshal(event.TransformedPayload)

	_, err := r.db.Pool.Exec(ctx, query,
		event.ID, event.Status, transformedJSON, event.DeliveryAttemptID, event.ErrorMessage, event.ProcessedAt)
	return err
}

func (r *replayRepository) GetPendingReplayEvents(ctx context.Context, jobID uuid.UUID, batchSize int) ([]*models.ReplayJobEvent, error) {
	query := `
		SELECT id, replay_job_id, event_archive_id, status, original_payload, created_at
		FROM replay_job_events 
		WHERE replay_job_id = $1 AND status = $2
		ORDER BY created_at ASC
		LIMIT $3
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.db.Pool.Query(ctx, query, jobID, models.ReplayEventPending, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.ReplayJobEvent
	for rows.Next() {
		var event models.ReplayJobEvent
		var originalJSON []byte

		if err := rows.Scan(&event.ID, &event.ReplayJobID, &event.EventArchiveID, &event.Status,
			&originalJSON, &event.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(originalJSON, &event.OriginalPayload)
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// Snapshot operations

func (r *replayRepository) CreateSnapshot(ctx context.Context, snapshot *models.ReplaySnapshot) error {
	query := `
		INSERT INTO replay_snapshots (id, tenant_id, name, description, snapshot_time, filter_criteria, event_count, size_bytes, storage_location, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if snapshot.ID == uuid.Nil {
		snapshot.ID = uuid.New()
	}
	snapshot.CreatedAt = time.Now()

	filterJSON, _ := json.Marshal(snapshot.FilterCriteria)

	_, err := r.db.Pool.Exec(ctx, query,
		snapshot.ID, snapshot.TenantID, snapshot.Name, snapshot.Description, snapshot.SnapshotTime,
		filterJSON, snapshot.EventCount, snapshot.SizeBytes, snapshot.StorageLocation, snapshot.ExpiresAt, snapshot.CreatedAt)
	return err
}

func (r *replayRepository) GetSnapshot(ctx context.Context, id uuid.UUID) (*models.ReplaySnapshot, error) {
	query := `
		SELECT id, tenant_id, name, description, snapshot_time, filter_criteria, event_count, size_bytes, storage_location, expires_at, created_at
		FROM replay_snapshots WHERE id = $1
	`

	var snapshot models.ReplaySnapshot
	var filterJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&snapshot.ID, &snapshot.TenantID, &snapshot.Name, &snapshot.Description, &snapshot.SnapshotTime,
		&filterJSON, &snapshot.EventCount, &snapshot.SizeBytes, &snapshot.StorageLocation, &snapshot.ExpiresAt, &snapshot.CreatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(filterJSON, &snapshot.FilterCriteria)
	return &snapshot, nil
}

func (r *replayRepository) GetSnapshots(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplaySnapshot, error) {
	query := `
		SELECT id, tenant_id, name, description, snapshot_time, filter_criteria, event_count, size_bytes, storage_location, expires_at, created_at
		FROM replay_snapshots WHERE tenant_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []*models.ReplaySnapshot
	for rows.Next() {
		var snapshot models.ReplaySnapshot
		var filterJSON []byte

		if err := rows.Scan(&snapshot.ID, &snapshot.TenantID, &snapshot.Name, &snapshot.Description, &snapshot.SnapshotTime,
			&filterJSON, &snapshot.EventCount, &snapshot.SizeBytes, &snapshot.StorageLocation, &snapshot.ExpiresAt, &snapshot.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(filterJSON, &snapshot.FilterCriteria)
		snapshots = append(snapshots, &snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return snapshots, nil
}

func (r *replayRepository) DeleteSnapshot(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM replay_snapshots WHERE id = $1`, id)
	return err
}

// Comparison operations

func (r *replayRepository) CreateComparison(ctx context.Context, comparison *models.ReplayComparison) error {
	query := `
		INSERT INTO replay_comparisons (id, tenant_id, name, description, original_job_id, comparison_job_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if comparison.ID == uuid.Nil {
		comparison.ID = uuid.New()
	}
	comparison.CreatedAt = time.Now()
	comparison.Status = models.ReplayStatusPending

	_, err := r.db.Pool.Exec(ctx, query,
		comparison.ID, comparison.TenantID, comparison.Name, comparison.Description,
		comparison.OriginalJobID, comparison.ComparisonJobID, comparison.Status, comparison.CreatedAt)
	return err
}

func (r *replayRepository) GetComparison(ctx context.Context, id uuid.UUID) (*models.ReplayComparison, error) {
	query := `
		SELECT id, tenant_id, name, description, original_job_id, comparison_job_id, status,
		       total_events, matching_events, differing_events, diff_report, completed_at, created_at
		FROM replay_comparisons WHERE id = $1
	`

	var comparison models.ReplayComparison
	var diffReportJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&comparison.ID, &comparison.TenantID, &comparison.Name, &comparison.Description,
		&comparison.OriginalJobID, &comparison.ComparisonJobID, &comparison.Status,
		&comparison.TotalEvents, &comparison.MatchingEvents, &comparison.DifferingEvents,
		&diffReportJSON, &comparison.CompletedAt, &comparison.CreatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(diffReportJSON, &comparison.DiffReport)
	return &comparison, nil
}

func (r *replayRepository) UpdateComparison(ctx context.Context, comparison *models.ReplayComparison) error {
	query := `
		UPDATE replay_comparisons 
		SET status = $2, total_events = $3, matching_events = $4, differing_events = $5, diff_report = $6, completed_at = $7
		WHERE id = $1
	`

	diffReportJSON, _ := json.Marshal(comparison.DiffReport)

	_, err := r.db.Pool.Exec(ctx, query,
		comparison.ID, comparison.Status, comparison.TotalEvents, comparison.MatchingEvents,
		comparison.DifferingEvents, diffReportJSON, comparison.CompletedAt)
	return err
}

package timetravel

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data access interface for time-travel operations
type Repository interface {
	RecordEvent(ctx context.Context, event *EventRecord) error
	GetEvents(ctx context.Context, tenantID string, filters *ReplayFilter, page, pageSize int) ([]EventRecord, int, error)
	GetEvent(ctx context.Context, tenantID, eventID string) (*EventRecord, error)

	CreateReplayJob(ctx context.Context, job *ReplayJob) error
	GetReplayJob(ctx context.Context, tenantID, jobID string) (*ReplayJob, error)
	UpdateReplayJob(ctx context.Context, job *ReplayJob) error
	ListReplayJobs(ctx context.Context, tenantID string, limit, offset int) ([]ReplayJob, int, error)

	CreateSnapshot(ctx context.Context, snapshot *PointInTimeSnapshot) error
	GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*PointInTimeSnapshot, error)
	ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]PointInTimeSnapshot, int, error)
	DeleteSnapshot(ctx context.Context, tenantID, snapshotID string) error

	SaveWhatIfScenario(ctx context.Context, scenario *WhatIfScenario) error
	GetWhatIfScenarios(ctx context.Context, tenantID string, limit, offset int) ([]WhatIfScenario, int, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL time-travel repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// RecordEvent stores a webhook event for time-travel
func (r *PostgresRepository) RecordEvent(ctx context.Context, event *EventRecord) error {
	query := `
		INSERT INTO timetravel_events (id, tenant_id, webhook_id, endpoint_id, event_type, payload, headers, timestamp, checksum)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.TenantID, event.WebhookID, event.EndpointID,
		event.EventType, event.Payload, event.Headers, event.Timestamp, event.Checksum)
	return err
}

// GetEvents retrieves events with filters and pagination
func (r *PostgresRepository) GetEvents(ctx context.Context, tenantID string, filters *ReplayFilter, page, pageSize int) ([]EventRecord, int, error) {
	var events []EventRecord
	var total int

	baseQuery := ` FROM timetravel_events WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argNum := 2

	if filters != nil {
		if len(filters.EndpointIDs) > 0 {
			baseQuery += fmt.Sprintf(` AND endpoint_id = ANY($%d)`, argNum)
			args = append(args, filters.EndpointIDs)
			argNum++
		}
		if len(filters.EventTypes) > 0 {
			baseQuery += fmt.Sprintf(` AND event_type = ANY($%d)`, argNum)
			args = append(args, filters.EventTypes)
			argNum++
		}
		if !filters.TimeWindow.Start.IsZero() {
			baseQuery += fmt.Sprintf(` AND timestamp >= $%d`, argNum)
			args = append(args, filters.TimeWindow.Start)
			argNum++
		}
		if !filters.TimeWindow.End.IsZero() {
			baseQuery += fmt.Sprintf(` AND timestamp <= $%d`, argNum)
			args = append(args, filters.TimeWindow.End)
			argNum++
		}
	}

	countQuery := `SELECT COUNT(*)` + baseQuery
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQuery := `SELECT id, tenant_id, webhook_id, endpoint_id, event_type, payload, headers, timestamp, checksum` +
		baseQuery + fmt.Sprintf(` ORDER BY timestamp DESC LIMIT $%d OFFSET $%d`, argNum, argNum+1)
	args = append(args, pageSize, offset)

	if err := r.db.SelectContext(ctx, &events, selectQuery, args...); err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// GetEvent retrieves a single event by ID
func (r *PostgresRepository) GetEvent(ctx context.Context, tenantID, eventID string) (*EventRecord, error) {
	var event EventRecord
	query := `SELECT * FROM timetravel_events WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &event, query, tenantID, eventID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &event, err
}

// CreateReplayJob creates a new replay job
func (r *PostgresRepository) CreateReplayJob(ctx context.Context, job *ReplayJob) error {
	twJSON, _ := json.Marshal(job.TimeWindow)
	filtersJSON, _ := json.Marshal(job.Filters)
	resultsJSON, _ := json.Marshal(job.Results)

	query := `
		INSERT INTO timetravel_replay_jobs (id, tenant_id, status, time_window, filters, target_endpoint, dry_run, progress, results, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.TenantID, job.Status, twJSON, filtersJSON,
		job.TargetEndpoint, job.DryRun, job.Progress, resultsJSON, job.CreatedAt)
	return err
}

// GetReplayJob retrieves a replay job by ID
func (r *PostgresRepository) GetReplayJob(ctx context.Context, tenantID, jobID string) (*ReplayJob, error) {
	var job ReplayJob
	query := `SELECT * FROM timetravel_replay_jobs WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &job, query, tenantID, jobID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if job.TimeWindowJSON != nil {
		json.Unmarshal(job.TimeWindowJSON, &job.TimeWindow)
	}
	if job.FiltersJSON != nil {
		json.Unmarshal(job.FiltersJSON, &job.Filters)
	}
	if job.ResultsJSON != nil {
		json.Unmarshal(job.ResultsJSON, &job.Results)
	}

	return &job, nil
}

// UpdateReplayJob updates an existing replay job
func (r *PostgresRepository) UpdateReplayJob(ctx context.Context, job *ReplayJob) error {
	twJSON, _ := json.Marshal(job.TimeWindow)
	filtersJSON, _ := json.Marshal(job.Filters)
	resultsJSON, _ := json.Marshal(job.Results)

	query := `
		UPDATE timetravel_replay_jobs
		SET status = $1, progress = $2, results = $3, time_window = $4, filters = $5, target_endpoint = $6
		WHERE tenant_id = $7 AND id = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		job.Status, job.Progress, resultsJSON, twJSON, filtersJSON,
		job.TargetEndpoint, job.TenantID, job.ID)
	return err
}

// ListReplayJobs lists replay jobs for a tenant
func (r *PostgresRepository) ListReplayJobs(ctx context.Context, tenantID string, limit, offset int) ([]ReplayJob, int, error) {
	var jobs []ReplayJob
	var total int

	countQuery := `SELECT COUNT(*) FROM timetravel_replay_jobs WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT * FROM timetravel_replay_jobs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &jobs, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	for i := range jobs {
		if jobs[i].TimeWindowJSON != nil {
			json.Unmarshal(jobs[i].TimeWindowJSON, &jobs[i].TimeWindow)
		}
		if jobs[i].FiltersJSON != nil {
			json.Unmarshal(jobs[i].FiltersJSON, &jobs[i].Filters)
		}
		if jobs[i].ResultsJSON != nil {
			json.Unmarshal(jobs[i].ResultsJSON, &jobs[i].Results)
		}
	}

	return jobs, total, nil
}

// CreateSnapshot creates a point-in-time snapshot
func (r *PostgresRepository) CreateSnapshot(ctx context.Context, snapshot *PointInTimeSnapshot) error {
	query := `
		INSERT INTO timetravel_snapshots (id, tenant_id, name, description, time_point, event_count, size_bytes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		snapshot.ID, snapshot.TenantID, snapshot.Name, snapshot.Description,
		snapshot.TimePoint, snapshot.EventCount, snapshot.SizeBytes, snapshot.CreatedAt)
	return err
}

// GetSnapshot retrieves a snapshot by ID
func (r *PostgresRepository) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*PointInTimeSnapshot, error) {
	var snapshot PointInTimeSnapshot
	query := `SELECT * FROM timetravel_snapshots WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &snapshot, query, tenantID, snapshotID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &snapshot, err
}

// ListSnapshots lists snapshots for a tenant
func (r *PostgresRepository) ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]PointInTimeSnapshot, int, error) {
	var snapshots []PointInTimeSnapshot
	var total int

	countQuery := `SELECT COUNT(*) FROM timetravel_snapshots WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT * FROM timetravel_snapshots
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &snapshots, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	return snapshots, total, nil
}

// DeleteSnapshot deletes a snapshot
func (r *PostgresRepository) DeleteSnapshot(ctx context.Context, tenantID, snapshotID string) error {
	query := `DELETE FROM timetravel_snapshots WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, snapshotID)
	return err
}

// SaveWhatIfScenario saves a what-if scenario
func (r *PostgresRepository) SaveWhatIfScenario(ctx context.Context, scenario *WhatIfScenario) error {
	query := `
		INSERT INTO timetravel_whatif_scenarios (id, tenant_id, name, description, modified_payload, original_payload, diff_summary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		scenario.ID, scenario.TenantID, scenario.Name, scenario.Description,
		scenario.ModifiedPayload, scenario.OriginalPayload, scenario.DiffSummary, scenario.CreatedAt)
	return err
}

// GetWhatIfScenarios lists what-if scenarios for a tenant
func (r *PostgresRepository) GetWhatIfScenarios(ctx context.Context, tenantID string, limit, offset int) ([]WhatIfScenario, int, error) {
	var scenarios []WhatIfScenario
	var total int

	countQuery := `SELECT COUNT(*) FROM timetravel_whatif_scenarios WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT * FROM timetravel_whatif_scenarios
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &scenarios, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	return scenarios, total, nil
}

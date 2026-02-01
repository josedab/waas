package eventsourcing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/josedab/waas/pkg/database"
)

// Repository implements event sourcing persistence
type Repository struct {
	db *database.DB
}

// NewRepository creates a new event sourcing repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Append adds events to a stream with optimistic concurrency
func (r *Repository) Append(ctx context.Context, tenantID string, expectedVersion int, events ...EventInput) ([]Event, error) {
	if len(events) == 0 {
		return nil, nil
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current version
	var currentVersion int
	streamID := events[0].StreamID
	err = tx.QueryRow(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM event_store WHERE tenant_id = $1 AND stream_id = $2",
		tenantID, streamID,
	).Scan(&currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// Check expected version (-1 means any)
	if expectedVersion >= 0 && currentVersion != expectedVersion {
		return nil, fmt.Errorf("%w: expected %d, got %d", ErrConcurrencyConflict, expectedVersion, currentVersion)
	}

	// Insert events
	var result []Event
	for i, e := range events {
		version := currentVersion + i + 1

		payloadJSON, err := json.Marshal(e.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}

		metadataJSON, err := json.Marshal(e.Metadata)
		if err != nil {
			metadataJSON = []byte("{}")
		}

		var event Event
		err = tx.QueryRow(ctx, `
			INSERT INTO event_store (tenant_id, stream_id, stream_type, event_type, payload, metadata, version, correlation_id, causation_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING sequence_id, event_id, tenant_id, stream_id, stream_type, event_type, payload, metadata, version, correlation_id, causation_id, created_at
		`, tenantID, streamID, e.StreamType, e.EventType, payloadJSON, metadataJSON, version, e.CorrelationID, e.CausationID,
		).Scan(
			&event.SequenceID, &event.EventID, &event.TenantID, &event.StreamID,
			&event.StreamType, &event.EventType, &event.Payload, &event.Metadata,
			&event.Version, &event.CorrelationID, &event.CausationID, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert event: %w", err)
		}

		result = append(result, event)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return result, nil
}

// Read reads events from a stream starting from a version
func (r *Repository) Read(ctx context.Context, tenantID, streamID string, fromVersion int, limit int) ([]Event, error) {
	query := `
		SELECT sequence_id, event_id, tenant_id, stream_id, stream_type, event_type, 
			payload, metadata, version, correlation_id, causation_id, created_at
		FROM event_store
		WHERE tenant_id = $1 AND stream_id = $2 AND version > $3
		ORDER BY version ASC
		LIMIT $4`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, streamID, fromVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

// ReadAll reads all events from a global sequence position
func (r *Repository) ReadAll(ctx context.Context, tenantID string, fromSequence int64, limit int) ([]Event, error) {
	query := `
		SELECT sequence_id, event_id, tenant_id, stream_id, stream_type, event_type, 
			payload, metadata, version, correlation_id, causation_id, created_at
		FROM event_store
		WHERE tenant_id = $1 AND sequence_id > $2
		ORDER BY sequence_id ASC
		LIMIT $3`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, fromSequence, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

// ReadByType reads events of specific types
func (r *Repository) ReadByType(ctx context.Context, tenantID string, eventTypes []string, fromSequence int64, limit int) ([]Event, error) {
	query := `
		SELECT sequence_id, event_id, tenant_id, stream_id, stream_type, event_type, 
			payload, metadata, version, correlation_id, causation_id, created_at
		FROM event_store
		WHERE tenant_id = $1 AND event_type = ANY($2) AND sequence_id > $3
		ORDER BY sequence_id ASC
		LIMIT $4`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, eventTypes, fromSequence, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *Repository) scanEvents(rows pgx.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		var metadataJSON []byte
		err := rows.Scan(
			&e.SequenceID, &e.EventID, &e.TenantID, &e.StreamID,
			&e.StreamType, &e.EventType, &e.Payload, &metadataJSON,
			&e.Version, &e.CorrelationID, &e.CausationID, &e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}
		events = append(events, e)
	}
	return events, nil
}

// GetStreamVersion gets the current version of a stream
func (r *Repository) GetStreamVersion(ctx context.Context, tenantID, streamID string) (int, error) {
	var version int
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM event_store WHERE tenant_id = $1 AND stream_id = $2",
		tenantID, streamID,
	).Scan(&version)
	return version, err
}

// GetCheckpoint retrieves a consumer checkpoint
func (r *Repository) GetCheckpoint(ctx context.Context, tenantID, consumerGroup string, streamID *string) (*Checkpoint, error) {
	var cp Checkpoint
	var metadataJSON []byte

	query := `
		SELECT id, tenant_id, consumer_group, stream_id, last_sequence_id, 
			last_processed_at, metadata, created_at, updated_at
		FROM consumer_checkpoints
		WHERE tenant_id = $1 AND consumer_group = $2 AND 
			(($3::text IS NULL AND stream_id IS NULL) OR stream_id = $3)`

	err := r.db.Pool.QueryRow(ctx, query, tenantID, consumerGroup, streamID).Scan(
		&cp.ID, &cp.TenantID, &cp.ConsumerGroup, &cp.StreamID,
		&cp.LastSequenceID, &cp.LastProcessedAt, &metadataJSON,
		&cp.CreatedAt, &cp.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrCheckpointNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &cp.Metadata)
	}

	return &cp, nil
}

// SaveCheckpoint saves or updates a consumer checkpoint
func (r *Repository) SaveCheckpoint(ctx context.Context, cp *Checkpoint) error {
	metadataJSON, _ := json.Marshal(cp.Metadata)

	query := `
		INSERT INTO consumer_checkpoints (tenant_id, consumer_group, stream_id, last_sequence_id, last_processed_at, metadata)
		VALUES ($1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (tenant_id, consumer_group, stream_id)
		DO UPDATE SET last_sequence_id = $4, last_processed_at = NOW(), metadata = $5, updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, query,
		cp.TenantID, cp.ConsumerGroup, cp.StreamID, cp.LastSequenceID, metadataJSON,
	).Scan(&cp.ID, &cp.CreatedAt, &cp.UpdatedAt)
}

// GetSnapshot retrieves the latest snapshot for a stream
func (r *Repository) GetSnapshot(ctx context.Context, tenantID, streamID string) (*Snapshot, error) {
	var s Snapshot

	query := `
		SELECT id, tenant_id, stream_id, stream_type, version, state, created_at
		FROM stream_snapshots
		WHERE tenant_id = $1 AND stream_id = $2
		ORDER BY version DESC
		LIMIT 1`

	err := r.db.Pool.QueryRow(ctx, query, tenantID, streamID).Scan(
		&s.ID, &s.TenantID, &s.StreamID, &s.StreamType, &s.Version, &s.State, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// SaveSnapshot saves a state snapshot
func (r *Repository) SaveSnapshot(ctx context.Context, s *Snapshot) error {
	query := `
		INSERT INTO stream_snapshots (tenant_id, stream_id, stream_type, version, state)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	return r.db.Pool.QueryRow(ctx, query,
		s.TenantID, s.StreamID, s.StreamType, s.Version, s.State,
	).Scan(&s.ID, &s.CreatedAt)
}

// CreateReplayJob creates a new replay job
func (r *Repository) CreateReplayJob(ctx context.Context, job *ReplayJob) error {
	query := `
		INSERT INTO replay_jobs (tenant_id, stream_id, stream_type, event_types, from_sequence_id, to_sequence_id, target_type, target_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status, created_at`

	return r.db.Pool.QueryRow(ctx, query,
		job.TenantID, job.StreamID, job.StreamType, job.EventTypes,
		job.FromSequenceID, job.ToSequenceID, job.TargetType, job.TargetID,
	).Scan(&job.ID, &job.Status, &job.CreatedAt)
}

// GetReplayJob retrieves a replay job by ID
func (r *Repository) GetReplayJob(ctx context.Context, id string) (*ReplayJob, error) {
	var job ReplayJob

	query := `
		SELECT id, tenant_id, stream_id, stream_type, event_types, from_sequence_id, to_sequence_id,
			target_type, target_id, status, current_sequence_id, events_processed, events_total,
			error_message, started_at, completed_at, created_at
		FROM replay_jobs
		WHERE id = $1`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.TenantID, &job.StreamID, &job.StreamType, &job.EventTypes,
		&job.FromSequenceID, &job.ToSequenceID, &job.TargetType, &job.TargetID,
		&job.Status, &job.CurrentSequenceID, &job.EventsProcessed, &job.EventsTotal,
		&job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &job, nil
}

// UpdateReplayJobProgress updates replay job progress
func (r *Repository) UpdateReplayJobProgress(ctx context.Context, id string, sequenceID int64, processed int) error {
	query := `
		UPDATE replay_jobs
		SET current_sequence_id = $2, events_processed = $3
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, sequenceID, processed)
	return err
}

// CompleteReplayJob marks a replay job as completed
func (r *Repository) CompleteReplayJob(ctx context.Context, id string, status string, errorMsg string) error {
	query := `
		UPDATE replay_jobs
		SET status = $2, error_message = $3, completed_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, status, errorMsg)
	return err
}

// ListReplayJobs lists replay jobs for a tenant
func (r *Repository) ListReplayJobs(ctx context.Context, tenantID string, status *string) ([]ReplayJob, error) {
	query := `
		SELECT id, tenant_id, stream_id, stream_type, event_types, from_sequence_id, to_sequence_id,
			target_type, target_id, status, current_sequence_id, events_processed, events_total,
			error_message, started_at, completed_at, created_at
		FROM replay_jobs
		WHERE tenant_id = $1 AND ($2::text IS NULL OR status = $2)
		ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []ReplayJob
	for rows.Next() {
		var job ReplayJob
		err := rows.Scan(
			&job.ID, &job.TenantID, &job.StreamID, &job.StreamType, &job.EventTypes,
			&job.FromSequenceID, &job.ToSequenceID, &job.TargetType, &job.TargetID,
			&job.Status, &job.CurrentSequenceID, &job.EventsProcessed, &job.EventsTotal,
			&job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

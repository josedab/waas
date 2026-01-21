package chaos

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for chaos storage
type Repository interface {
	// Experiment operations
	SaveExperiment(ctx context.Context, exp *ChaosExperiment) error
	GetExperiment(ctx context.Context, tenantID, expID string) (*ChaosExperiment, error)
	ListExperiments(ctx context.Context, tenantID string, status *ExperimentStatus, limit, offset int) ([]ChaosExperiment, int, error)
	DeleteExperiment(ctx context.Context, tenantID, expID string) error

	// Event operations
	SaveEvent(ctx context.Context, event *ChaosEvent) error
	GetEvents(ctx context.Context, tenantID, expID string, limit int) ([]ChaosEvent, error)
	GetEventsByDelivery(ctx context.Context, tenantID, deliveryID string) ([]ChaosEvent, error)

	// Stats for resilience report
	GetExperimentStats(ctx context.Context, tenantID string, start, end time.Time) (map[string]interface{}, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveExperiment(ctx context.Context, exp *ChaosExperiment) error {
	targetJSON, _ := json.Marshal(exp.TargetConfig)
	faultJSON, _ := json.Marshal(exp.FaultConfig)
	scheduleJSON, _ := json.Marshal(exp.Schedule)
	blastJSON, _ := json.Marshal(exp.BlastRadius)
	resultsJSON, _ := json.Marshal(exp.Results)

	query := `
		INSERT INTO chaos_experiments (
			id, tenant_id, name, description, type, status, target_config,
			fault_config, schedule, blast_radius, duration_seconds,
			started_at, completed_at, results, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			target_config = EXCLUDED.target_config,
			fault_config = EXCLUDED.fault_config,
			schedule = EXCLUDED.schedule,
			blast_radius = EXCLUDED.blast_radius,
			duration_seconds = EXCLUDED.duration_seconds,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			results = EXCLUDED.results,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(ctx, query,
		exp.ID, exp.TenantID, exp.Name, exp.Description, exp.Type, exp.Status,
		targetJSON, faultJSON, scheduleJSON, blastJSON, exp.Duration,
		exp.StartedAt, exp.CompletedAt, resultsJSON, exp.CreatedBy,
		exp.CreatedAt, exp.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetExperiment(ctx context.Context, tenantID, expID string) (*ChaosExperiment, error) {
	query := `
		SELECT id, tenant_id, name, description, type, status, target_config,
			fault_config, schedule, blast_radius, duration_seconds,
			started_at, completed_at, results, created_by, created_at, updated_at
		FROM chaos_experiments
		WHERE tenant_id = $1 AND id = $2`

	var exp ChaosExperiment
	var targetJSON, faultJSON, scheduleJSON, blastJSON, resultsJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, expID).Scan(
		&exp.ID, &exp.TenantID, &exp.Name, &exp.Description, &exp.Type, &exp.Status,
		&targetJSON, &faultJSON, &scheduleJSON, &blastJSON, &exp.Duration,
		&exp.StartedAt, &exp.CompletedAt, &resultsJSON, &exp.CreatedBy,
		&exp.CreatedAt, &exp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(targetJSON, &exp.TargetConfig)
	json.Unmarshal(faultJSON, &exp.FaultConfig)
	json.Unmarshal(scheduleJSON, &exp.Schedule)
	json.Unmarshal(blastJSON, &exp.BlastRadius)
	json.Unmarshal(resultsJSON, &exp.Results)

	return &exp, nil
}

func (r *PostgresRepository) ListExperiments(ctx context.Context, tenantID string, status *ExperimentStatus, limit, offset int) ([]ChaosExperiment, int, error) {
	baseQuery := `
		SELECT id, tenant_id, name, description, type, status, target_config,
			fault_config, schedule, blast_radius, duration_seconds,
			started_at, completed_at, results, created_by, created_at, updated_at
		FROM chaos_experiments
		WHERE tenant_id = $1`

	countQuery := `SELECT COUNT(*) FROM chaos_experiments WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	countArgs := []interface{}{tenantID}

	if status != nil {
		baseQuery += " AND status = $2"
		countQuery += " AND status = $2"
		args = append(args, *status)
		countArgs = append(countArgs, *status)
	}

	baseQuery += " ORDER BY created_at DESC LIMIT $" + string(rune(len(args)+1+'0')) + " OFFSET $" + string(rune(len(args)+2+'0'))
	args = append(args, limit, offset)

	// Get total count
	var totalCount int
	r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var experiments []ChaosExperiment
	for rows.Next() {
		var exp ChaosExperiment
		var targetJSON, faultJSON, scheduleJSON, blastJSON, resultsJSON []byte

		err := rows.Scan(
			&exp.ID, &exp.TenantID, &exp.Name, &exp.Description, &exp.Type, &exp.Status,
			&targetJSON, &faultJSON, &scheduleJSON, &blastJSON, &exp.Duration,
			&exp.StartedAt, &exp.CompletedAt, &resultsJSON, &exp.CreatedBy,
			&exp.CreatedAt, &exp.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		json.Unmarshal(targetJSON, &exp.TargetConfig)
		json.Unmarshal(faultJSON, &exp.FaultConfig)
		json.Unmarshal(scheduleJSON, &exp.Schedule)
		json.Unmarshal(blastJSON, &exp.BlastRadius)
		json.Unmarshal(resultsJSON, &exp.Results)

		experiments = append(experiments, exp)
	}

	return experiments, totalCount, nil
}

func (r *PostgresRepository) DeleteExperiment(ctx context.Context, tenantID, expID string) error {
	// Delete events first
	_, err := r.db.ExecContext(ctx, 
		"DELETE FROM chaos_events WHERE tenant_id = $1 AND experiment_id = $2",
		tenantID, expID)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		"DELETE FROM chaos_experiments WHERE tenant_id = $1 AND id = $2",
		tenantID, expID)
	return err
}

func (r *PostgresRepository) SaveEvent(ctx context.Context, event *ChaosEvent) error {
	query := `
		INSERT INTO chaos_events (
			id, experiment_id, tenant_id, endpoint_id, delivery_id,
			event_type, injected_fault, original_state, injected_state,
			recovered, recovery_time_ms, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.ExperimentID, event.TenantID, event.EndpointID,
		event.DeliveryID, event.EventType, event.InjectedFault,
		event.OriginalState, event.InjectedState, event.Recovered,
		event.RecoveryTime, event.Timestamp,
	)
	return err
}

func (r *PostgresRepository) GetEvents(ctx context.Context, tenantID, expID string, limit int) ([]ChaosEvent, error) {
	query := `
		SELECT id, experiment_id, tenant_id, endpoint_id, delivery_id,
			event_type, injected_fault, original_state, injected_state,
			recovered, recovery_time_ms, timestamp
		FROM chaos_events
		WHERE tenant_id = $1 AND experiment_id = $2
		ORDER BY timestamp DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, expID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ChaosEvent
	for rows.Next() {
		var event ChaosEvent
		err := rows.Scan(
			&event.ID, &event.ExperimentID, &event.TenantID, &event.EndpointID,
			&event.DeliveryID, &event.EventType, &event.InjectedFault,
			&event.OriginalState, &event.InjectedState, &event.Recovered,
			&event.RecoveryTime, &event.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresRepository) GetEventsByDelivery(ctx context.Context, tenantID, deliveryID string) ([]ChaosEvent, error) {
	query := `
		SELECT id, experiment_id, tenant_id, endpoint_id, delivery_id,
			event_type, injected_fault, original_state, injected_state,
			recovered, recovery_time_ms, timestamp
		FROM chaos_events
		WHERE tenant_id = $1 AND delivery_id = $2
		ORDER BY timestamp DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, deliveryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ChaosEvent
	for rows.Next() {
		var event ChaosEvent
		err := rows.Scan(
			&event.ID, &event.ExperimentID, &event.TenantID, &event.EndpointID,
			&event.DeliveryID, &event.EventType, &event.InjectedFault,
			&event.OriginalState, &event.InjectedState, &event.Recovered,
			&event.RecoveryTime, &event.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresRepository) GetExperimentStats(ctx context.Context, tenantID string, start, end time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get experiment counts by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM chaos_experiments
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3
		GROUP BY status`

	rows, err := r.db.QueryContext(ctx, statusQuery, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byStatus := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		rows.Scan(&status, &count)
		byStatus[status] = count
	}
	stats["by_status"] = byStatus

	// Get event aggregates
	eventQuery := `
		SELECT 
			COUNT(*) as total_events,
			SUM(CASE WHEN recovered THEN 1 ELSE 0 END) as recovered,
			AVG(CASE WHEN recovered THEN recovery_time_ms ELSE NULL END) as avg_recovery
		FROM chaos_events
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3`

	var totalEvents, recovered int64
	var avgRecovery *float64
	r.db.QueryRowContext(ctx, eventQuery, tenantID, start, end).Scan(&totalEvents, &recovered, &avgRecovery)

	stats["total_events"] = totalEvents
	stats["recovered_events"] = recovered
	if avgRecovery != nil {
		stats["avg_recovery_ms"] = *avgRecovery
	}

	return stats, nil
}

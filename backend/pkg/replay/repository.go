package replay

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for replay storage
type Repository interface {
	// Delivery archive operations
	GetDeliveryArchive(ctx context.Context, tenantID, deliveryID string) (*DeliveryArchive, error)
	ListDeliveryArchives(ctx context.Context, tenantID string, filters *BulkReplayRequest) ([]DeliveryArchive, int, error)
	ArchiveDelivery(ctx context.Context, archive *DeliveryArchive) error

	// Snapshot operations
	CreateSnapshot(ctx context.Context, snapshot *Snapshot, deliveryIDs []string) error
	GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*Snapshot, error)
	ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]Snapshot, int, error)
	GetSnapshotDeliveryIDs(ctx context.Context, snapshotID string) ([]string, error)
	DeleteSnapshot(ctx context.Context, tenantID, snapshotID string) error
	CleanupExpiredSnapshots(ctx context.Context) (int64, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL replay repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetDeliveryArchive retrieves an archived delivery
func (r *PostgresRepository) GetDeliveryArchive(ctx context.Context, tenantID, deliveryID string) (*DeliveryArchive, error) {
	var archive DeliveryArchive
	query := `
		SELECT da.id, da.tenant_id, da.endpoint_id, da.endpoint_url, da.payload, 
		       da.headers, da.status, da.attempt_count, da.last_http_status, 
		       da.last_error, da.created_at, da.completed_at
		FROM delivery_archives da
		WHERE da.tenant_id = $1 AND da.id = $2
	`
	err := r.db.GetContext(ctx, &archive, query, tenantID, deliveryID)
	if err == sql.ErrNoRows {
		// Fallback to live deliveries table
		return r.getFromLiveDeliveries(ctx, tenantID, deliveryID)
	}
	if err != nil {
		return nil, err
	}

	// Parse headers
	if archive.HeadersJSON != nil {
		json.Unmarshal(archive.HeadersJSON, &archive.Headers)
	}

	return &archive, nil
}

func (r *PostgresRepository) getFromLiveDeliveries(ctx context.Context, tenantID, deliveryID string) (*DeliveryArchive, error) {
	var archive DeliveryArchive
	query := `
		SELECT dr.id, dr.tenant_id, dr.endpoint_id, we.url as endpoint_url, 
		       dr.payload, dr.headers, dr.status, dr.attempt_count, 
		       dr.last_http_status, dr.last_error, dr.created_at, dr.completed_at
		FROM delivery_requests dr
		JOIN webhook_endpoints we ON dr.endpoint_id = we.id
		WHERE dr.tenant_id = $1 AND dr.id = $2
	`
	err := r.db.GetContext(ctx, &archive, query, tenantID, deliveryID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse headers
	if archive.HeadersJSON != nil {
		json.Unmarshal(archive.HeadersJSON, &archive.Headers)
	}

	return &archive, nil
}

// ListDeliveryArchives lists archived deliveries with filters
func (r *PostgresRepository) ListDeliveryArchives(ctx context.Context, tenantID string, filters *BulkReplayRequest) ([]DeliveryArchive, int, error) {
	var archives []DeliveryArchive
	var total int

	// Build query with filters
	baseQuery := `
		FROM delivery_requests dr
		JOIN webhook_endpoints we ON dr.endpoint_id = we.id
		WHERE dr.tenant_id = $1
	`
	args := []interface{}{tenantID}
	argNum := 2

	if filters.EndpointID != "" {
		baseQuery += ` AND dr.endpoint_id = $` + string(rune('0'+argNum))
		args = append(args, filters.EndpointID)
		argNum++
	}

	if filters.Status != "" {
		baseQuery += ` AND dr.status = $` + string(rune('0'+argNum))
		args = append(args, filters.Status)
		argNum++
	}

	if !filters.StartTime.IsZero() {
		baseQuery += ` AND dr.created_at >= $` + string(rune('0'+argNum))
		args = append(args, filters.StartTime)
		argNum++
	}

	if !filters.EndTime.IsZero() {
		baseQuery += ` AND dr.created_at <= $` + string(rune('0'+argNum))
		args = append(args, filters.EndTime)
		argNum++
	}

	// Count query
	countQuery := `SELECT COUNT(*) ` + baseQuery
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Main query
	limit := filters.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	selectQuery := `
		SELECT dr.id, dr.tenant_id, dr.endpoint_id, we.url as endpoint_url, 
		       dr.payload, dr.headers, dr.status, dr.attempt_count, 
		       dr.last_http_status, dr.last_error, dr.created_at, dr.completed_at
	` + baseQuery + ` ORDER BY dr.created_at DESC LIMIT $` + string(rune('0'+argNum))
	args = append(args, limit)

	if err := r.db.SelectContext(ctx, &archives, selectQuery, args...); err != nil {
		return nil, 0, err
	}

	// Parse headers for each archive
	for i := range archives {
		if archives[i].HeadersJSON != nil {
			json.Unmarshal(archives[i].HeadersJSON, &archives[i].Headers)
		}
	}

	return archives, total, nil
}

// ArchiveDelivery archives a delivery
func (r *PostgresRepository) ArchiveDelivery(ctx context.Context, archive *DeliveryArchive) error {
	headersJSON, _ := json.Marshal(archive.Headers)

	query := `
		INSERT INTO delivery_archives (id, tenant_id, endpoint_id, endpoint_url, payload, 
		                               headers, status, attempt_count, last_http_status, 
		                               last_error, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		archive.ID, archive.TenantID, archive.EndpointID, archive.EndpointURL,
		archive.Payload, headersJSON, archive.Status, archive.AttemptCount,
		archive.LastHTTPStatus, archive.LastError, archive.CreatedAt, archive.CompletedAt)
	return err
}

// CreateSnapshot creates a new snapshot
func (r *PostgresRepository) CreateSnapshot(ctx context.Context, snapshot *Snapshot, deliveryIDs []string) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}
	snapshot.CreatedAt = time.Now()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert snapshot
	query := `
		INSERT INTO replay_snapshots (id, tenant_id, name, description, filters, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.ExecContext(ctx, query,
		snapshot.ID, snapshot.TenantID, snapshot.Name, snapshot.Description,
		snapshot.Filters, snapshot.CreatedAt, snapshot.ExpiresAt)
	if err != nil {
		return err
	}

	// Insert delivery IDs
	for _, deliveryID := range deliveryIDs {
		insertQuery := `INSERT INTO snapshot_deliveries (snapshot_id, delivery_id) VALUES ($1, $2)`
		_, err = tx.ExecContext(ctx, insertQuery, snapshot.ID, deliveryID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetSnapshot retrieves a snapshot
func (r *PostgresRepository) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*Snapshot, error) {
	var snapshot Snapshot
	query := `SELECT * FROM replay_snapshots WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &snapshot, query, tenantID, snapshotID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &snapshot, err
}

// ListSnapshots lists all snapshots for a tenant
func (r *PostgresRepository) ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]Snapshot, int, error) {
	var snapshots []Snapshot
	var total int

	countQuery := `SELECT COUNT(*) FROM replay_snapshots WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT * FROM replay_snapshots 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &snapshots, query, tenantID, limit, offset); err != nil {
		return nil, 0, err
	}

	return snapshots, total, nil
}

// GetSnapshotDeliveryIDs retrieves delivery IDs for a snapshot
func (r *PostgresRepository) GetSnapshotDeliveryIDs(ctx context.Context, snapshotID string) ([]string, error) {
	var ids []string
	query := `SELECT delivery_id FROM snapshot_deliveries WHERE snapshot_id = $1`
	err := r.db.SelectContext(ctx, &ids, query, snapshotID)
	return ids, err
}

// DeleteSnapshot deletes a snapshot
func (r *PostgresRepository) DeleteSnapshot(ctx context.Context, tenantID, snapshotID string) error {
	query := `DELETE FROM replay_snapshots WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, snapshotID)
	return err
}

// CleanupExpiredSnapshots removes expired snapshots
func (r *PostgresRepository) CleanupExpiredSnapshots(ctx context.Context) (int64, error) {
	query := `DELETE FROM replay_snapshots WHERE expires_at IS NOT NULL AND expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

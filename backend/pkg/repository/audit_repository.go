package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type auditRepository struct {
	db *sqlx.DB
}

// AuditEvent represents an auditable event in the system
type AuditEvent struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	UserID      *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	Action      string                 `json:"action" db:"action"`
	Resource    string                 `json:"resource" db:"resource"`
	ResourceID  *string                `json:"resource_id,omitempty" db:"resource_id"`
	Details     map[string]interface{} `json:"details" db:"details"`
	IPAddress   string                 `json:"ip_address" db:"ip_address"`
	UserAgent   string                 `json:"user_agent" db:"user_agent"`
	Success     bool                   `json:"success" db:"success"`
	ErrorMsg    *string                `json:"error_message,omitempty" db:"error_message"`
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"`
}

// AuditFilter defines filtering options for audit log queries
type AuditFilter struct {
	StartTime  *time.Time
	EndTime    *time.Time
	Actions    []string
	Resources  []string
	UserID     *uuid.UUID
	Success    *bool
	Limit      int
	Offset     int
}

// AuditRepository defines the interface for audit log storage
type AuditRepository interface {
	LogEvent(ctx context.Context, event *AuditEvent) error
	GetAuditLogs(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error)
	GetAuditLogsByTenant(ctx context.Context, tenantID uuid.UUID, filter AuditFilter) ([]*AuditEvent, error)
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sqlx.DB) AuditRepository {
	return &auditRepository{db: db}
}

// LogEvent stores an audit event
func (r *auditRepository) LogEvent(ctx context.Context, event *AuditEvent) error {
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal event details: %w", err)
	}

	query := `
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource, resource_id, details, ip_address, user_agent, success, error_message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		event.TenantID,
		event.UserID,
		event.Action,
		event.Resource,
		event.ResourceID,
		detailsJSON,
		event.IPAddress,
		event.UserAgent,
		event.Success,
		event.ErrorMsg,
		event.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to log audit event: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs with filtering
func (r *auditRepository) GetAuditLogs(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error) {
	query, args := r.buildAuditQuery(filter, false, uuid.Nil)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event, err := r.scanAuditEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return events, nil
}

// GetAuditLogsByTenant retrieves audit logs for a specific tenant
func (r *auditRepository) GetAuditLogsByTenant(ctx context.Context, tenantID uuid.UUID, filter AuditFilter) ([]*AuditEvent, error) {
	query, args := r.buildAuditQuery(filter, true, tenantID)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant audit logs: %w", err)
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event, err := r.scanAuditEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant audit logs: %w", err)
	}

	return events, nil
}

// buildAuditQuery constructs the SQL query with filters
func (r *auditRepository) buildAuditQuery(filter AuditFilter, filterByTenant bool, tenantID uuid.UUID) (string, []interface{}) {
	baseQuery := `
		SELECT id, tenant_id, user_id, action, resource, resource_id, details, ip_address, user_agent, success, error_message, timestamp
		FROM audit_logs`
	
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add tenant filter if specified
	if filterByTenant {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
		args = append(args, tenantID)
		argIndex++
	}

	// Add time range filters
	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	// Add action filters
	if len(filter.Actions) > 0 {
		placeholders := make([]string, len(filter.Actions))
		for i, action := range filter.Actions {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, action)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("action IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add resource filters
	if len(filter.Resources) > 0 {
		placeholders := make([]string, len(filter.Resources))
		for i, resource := range filter.Resources {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, resource)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("resource IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add user filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	// Add success filter
	if filter.Success != nil {
		conditions = append(conditions, fmt.Sprintf("success = $%d", argIndex))
		args = append(args, *filter.Success)
		argIndex++
	}

	// Build WHERE clause
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	baseQuery += " ORDER BY timestamp DESC"

	// Add pagination
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	return baseQuery, args
}

// scanAuditEvent scans a database row into an AuditEvent
func (r *auditRepository) scanAuditEvent(scanner interface {
	Scan(dest ...interface{}) error
}) (*AuditEvent, error) {
	var event AuditEvent
	var detailsJSON []byte

	err := scanner.Scan(
		&event.ID,
		&event.TenantID,
		&event.UserID,
		&event.Action,
		&event.Resource,
		&event.ResourceID,
		&detailsJSON,
		&event.IPAddress,
		&event.UserAgent,
		&event.Success,
		&event.ErrorMsg,
		&event.Timestamp,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal details JSON
	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &event.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event details: %w", err)
		}
	}

	return &event, nil
}
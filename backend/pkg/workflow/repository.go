package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines workflow data access
type Repository interface {
	// Workflow CRUD
	SaveWorkflow(ctx context.Context, wf *Workflow) error
	GetWorkflow(ctx context.Context, tenantID, workflowID string) (*Workflow, error)
	ListWorkflows(ctx context.Context, tenantID string, filter *WorkflowFilter) ([]Workflow, error)
	DeleteWorkflow(ctx context.Context, tenantID, workflowID string) error
	
	// Workflow versions
	GetWorkflowVersion(ctx context.Context, tenantID, workflowID string, version int) (*Workflow, error)
	ListWorkflowVersions(ctx context.Context, tenantID, workflowID string) ([]WorkflowVersionInfo, error)
	
	// Executions
	SaveExecution(ctx context.Context, exec *WorkflowExecution) error
	GetExecution(ctx context.Context, tenantID, execID string) (*WorkflowExecution, error)
	ListExecutions(ctx context.Context, tenantID, workflowID string, filter *ExecutionFilter) ([]WorkflowExecution, error)
	UpdateExecutionStatus(ctx context.Context, execID string, status ExecutionStatus, output json.RawMessage, err *ExecutionError) error
	
	// Stats
	GetWorkflowStats(ctx context.Context, tenantID, workflowID string) (*WorkflowStats, error)
	
	// Templates
	ListTemplates(ctx context.Context, category string) ([]WorkflowTemplate, error)
	GetTemplate(ctx context.Context, templateID string) (*WorkflowTemplate, error)
}

// WorkflowFilter defines workflow list filters
type WorkflowFilter struct {
	Status    *WorkflowStatus
	Search    string
	Tags      []string
	Limit     int
	Offset    int
	OrderBy   string
	OrderDesc bool
}

// ExecutionFilter defines execution list filters
type ExecutionFilter struct {
	Status    *ExecutionStatus
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// WorkflowVersionInfo holds version metadata
type WorkflowVersionInfo struct {
	Version     int       `json:"version"`
	Status      WorkflowStatus `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveWorkflow saves a workflow
func (r *PostgresRepository) SaveWorkflow(ctx context.Context, wf *Workflow) error {
	nodesJSON, err := json.Marshal(wf.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}
	edgesJSON, err := json.Marshal(wf.Edges)
	if err != nil {
		return fmt.Errorf("marshal edges: %w", err)
	}
	variablesJSON, err := json.Marshal(wf.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	settingsJSON, err := json.Marshal(wf.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	triggerJSON, err := json.Marshal(wf.Trigger)
	if err != nil {
		return fmt.Errorf("marshal trigger: %w", err)
	}
	canvasJSON, err := json.Marshal(wf.Canvas)
	if err != nil {
		return fmt.Errorf("marshal canvas: %w", err)
	}

	query := `
		INSERT INTO workflows (
			id, tenant_id, name, description, version, status,
			trigger_config, nodes, edges, variables, settings, canvas,
			created_by, created_at, updated_at, published_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			version = EXCLUDED.version,
			status = EXCLUDED.status,
			trigger_config = EXCLUDED.trigger_config,
			nodes = EXCLUDED.nodes,
			edges = EXCLUDED.edges,
			variables = EXCLUDED.variables,
			settings = EXCLUDED.settings,
			canvas = EXCLUDED.canvas,
			updated_at = EXCLUDED.updated_at,
			published_at = EXCLUDED.published_at`

	_, err = r.db.ExecContext(ctx, query,
		wf.ID, wf.TenantID, wf.Name, wf.Description, wf.Version, wf.Status,
		triggerJSON, nodesJSON, edgesJSON, variablesJSON, settingsJSON, canvasJSON,
		wf.CreatedBy, wf.CreatedAt, wf.UpdatedAt, wf.PublishedAt)

	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	// Save version snapshot
	versionQuery := `
		INSERT INTO workflow_versions (
			workflow_id, version, trigger_config, nodes, edges,
			variables, settings, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (workflow_id, version) DO UPDATE SET
			trigger_config = EXCLUDED.trigger_config,
			nodes = EXCLUDED.nodes,
			edges = EXCLUDED.edges,
			variables = EXCLUDED.variables,
			settings = EXCLUDED.settings`

	r.db.ExecContext(ctx, versionQuery,
		wf.ID, wf.Version, triggerJSON, nodesJSON, edgesJSON,
		variablesJSON, settingsJSON, wf.UpdatedAt)

	return nil
}

// GetWorkflow retrieves a workflow
func (r *PostgresRepository) GetWorkflow(ctx context.Context, tenantID, workflowID string) (*Workflow, error) {
	query := `
		SELECT id, tenant_id, name, description, version, status,
			   trigger_config, nodes, edges, variables, settings, canvas,
			   created_by, created_at, updated_at, published_at
		FROM workflows
		WHERE tenant_id = $1 AND id = $2`

	var wf Workflow
	var nodesJSON, edgesJSON, variablesJSON, settingsJSON, triggerJSON, canvasJSON []byte
	var publishedAt sql.NullTime
	var createdBy sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, workflowID).Scan(
		&wf.ID, &wf.TenantID, &wf.Name, &wf.Description, &wf.Version, &wf.Status,
		&triggerJSON, &nodesJSON, &edgesJSON, &variablesJSON, &settingsJSON, &canvasJSON,
		&createdBy, &wf.CreatedAt, &wf.UpdatedAt, &publishedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	json.Unmarshal(triggerJSON, &wf.Trigger)
	json.Unmarshal(nodesJSON, &wf.Nodes)
	json.Unmarshal(edgesJSON, &wf.Edges)
	json.Unmarshal(variablesJSON, &wf.Variables)
	json.Unmarshal(settingsJSON, &wf.Settings)
	json.Unmarshal(canvasJSON, &wf.Canvas)

	if publishedAt.Valid {
		wf.PublishedAt = &publishedAt.Time
	}
	if createdBy.Valid {
		wf.CreatedBy = createdBy.String
	}

	return &wf, nil
}

// ListWorkflows lists workflows for a tenant
func (r *PostgresRepository) ListWorkflows(ctx context.Context, tenantID string, filter *WorkflowFilter) ([]Workflow, error) {
	query := `
		SELECT id, tenant_id, name, description, version, status,
			   trigger_config, nodes, edges, variables, settings, canvas,
			   created_by, created_at, updated_at, published_at
		FROM workflows
		WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if filter != nil {
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.Search != "" {
			query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
			args = append(args, "%"+filter.Search+"%")
			argIdx++
		}
		if len(filter.Tags) > 0 {
			query += fmt.Sprintf(" AND settings->'tags' ?| $%d", argIdx)
			args = append(args, filter.Tags)
			argIdx++
		}

		orderBy := "updated_at"
		if filter.OrderBy != "" {
			orderBy = filter.OrderBy
		}
		order := "DESC"
		if !filter.OrderDesc {
			order = "ASC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	} else {
		query += " ORDER BY updated_at DESC LIMIT 50"
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []Workflow
	for rows.Next() {
		var wf Workflow
		var nodesJSON, edgesJSON, variablesJSON, settingsJSON, triggerJSON, canvasJSON []byte
		var publishedAt sql.NullTime
		var createdBy sql.NullString

		err := rows.Scan(
			&wf.ID, &wf.TenantID, &wf.Name, &wf.Description, &wf.Version, &wf.Status,
			&triggerJSON, &nodesJSON, &edgesJSON, &variablesJSON, &settingsJSON, &canvasJSON,
			&createdBy, &wf.CreatedAt, &wf.UpdatedAt, &publishedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(triggerJSON, &wf.Trigger)
		json.Unmarshal(nodesJSON, &wf.Nodes)
		json.Unmarshal(edgesJSON, &wf.Edges)
		json.Unmarshal(variablesJSON, &wf.Variables)
		json.Unmarshal(settingsJSON, &wf.Settings)
		json.Unmarshal(canvasJSON, &wf.Canvas)

		if publishedAt.Valid {
			wf.PublishedAt = &publishedAt.Time
		}
		if createdBy.Valid {
			wf.CreatedBy = createdBy.String
		}

		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// DeleteWorkflow deletes a workflow
func (r *PostgresRepository) DeleteWorkflow(ctx context.Context, tenantID, workflowID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM workflows WHERE tenant_id = $1 AND id = $2",
		tenantID, workflowID)
	return err
}

// GetWorkflowVersion retrieves a specific workflow version
func (r *PostgresRepository) GetWorkflowVersion(ctx context.Context, tenantID, workflowID string, version int) (*Workflow, error) {
	// First get the base workflow
	wf, err := r.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	// Then get the version data
	query := `
		SELECT trigger_config, nodes, edges, variables, settings, created_at
		FROM workflow_versions
		WHERE workflow_id = $1 AND version = $2`

	var nodesJSON, edgesJSON, variablesJSON, settingsJSON, triggerJSON []byte
	var createdAt time.Time

	err = r.db.QueryRowContext(ctx, query, workflowID, version).Scan(
		&triggerJSON, &nodesJSON, &edgesJSON, &variablesJSON, &settingsJSON, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("version not found")
	}
	if err != nil {
		return nil, err
	}

	wf.Version = version
	json.Unmarshal(triggerJSON, &wf.Trigger)
	json.Unmarshal(nodesJSON, &wf.Nodes)
	json.Unmarshal(edgesJSON, &wf.Edges)
	json.Unmarshal(variablesJSON, &wf.Variables)
	json.Unmarshal(settingsJSON, &wf.Settings)

	return wf, nil
}

// ListWorkflowVersions lists all versions of a workflow
func (r *PostgresRepository) ListWorkflowVersions(ctx context.Context, tenantID, workflowID string) ([]WorkflowVersionInfo, error) {
	query := `
		SELECT wv.version, w.status, wv.created_at, w.published_at
		FROM workflow_versions wv
		JOIN workflows w ON w.id = wv.workflow_id
		WHERE w.tenant_id = $1 AND wv.workflow_id = $2
		ORDER BY wv.version DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []WorkflowVersionInfo
	for rows.Next() {
		var v WorkflowVersionInfo
		var publishedAt sql.NullTime

		if err := rows.Scan(&v.Version, &v.Status, &v.CreatedAt, &publishedAt); err != nil {
			continue
		}
		if publishedAt.Valid {
			v.PublishedAt = &publishedAt.Time
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// SaveExecution saves a workflow execution
func (r *PostgresRepository) SaveExecution(ctx context.Context, exec *WorkflowExecution) error {
	variablesJSON, err := json.Marshal(exec.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	nodeStatesJSON, err := json.Marshal(exec.NodeStates)
	if err != nil {
		return fmt.Errorf("marshal node states: %w", err)
	}
	errorJSON, err := json.Marshal(exec.Error)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	query := `
		INSERT INTO workflow_executions (
			id, workflow_id, workflow_name, tenant_id, version, status,
			trigger_type, trigger_data, input, output,
			variables, node_states, error,
			started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			output = EXCLUDED.output,
			node_states = EXCLUDED.node_states,
			error = EXCLUDED.error,
			completed_at = EXCLUDED.completed_at`

	_, err = r.db.ExecContext(ctx, query,
		exec.ID, exec.WorkflowID, exec.WorkflowName, exec.TenantID, exec.Version, exec.Status,
		exec.TriggerType, exec.TriggerData, exec.Input, exec.Output,
		variablesJSON, nodeStatesJSON, errorJSON,
		exec.StartedAt, exec.CompletedAt)

	return err
}

// GetExecution retrieves an execution
func (r *PostgresRepository) GetExecution(ctx context.Context, tenantID, execID string) (*WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, workflow_name, tenant_id, version, status,
			   trigger_type, trigger_data, input, output,
			   variables, node_states, error,
			   started_at, completed_at
		FROM workflow_executions
		WHERE tenant_id = $1 AND id = $2`

	var exec WorkflowExecution
	var variablesJSON, nodeStatesJSON, errorJSON []byte
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, execID).Scan(
		&exec.ID, &exec.WorkflowID, &exec.WorkflowName, &exec.TenantID, &exec.Version, &exec.Status,
		&exec.TriggerType, &exec.TriggerData, &exec.Input, &exec.Output,
		&variablesJSON, &nodeStatesJSON, &errorJSON,
		&exec.StartedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(variablesJSON, &exec.Variables)
	json.Unmarshal(nodeStatesJSON, &exec.NodeStates)
	json.Unmarshal(errorJSON, &exec.Error)

	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
		exec.Duration = completedAt.Time.Sub(exec.StartedAt)
	}

	return &exec, nil
}

// ListExecutions lists executions for a workflow
func (r *PostgresRepository) ListExecutions(ctx context.Context, tenantID, workflowID string, filter *ExecutionFilter) ([]WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, workflow_name, tenant_id, version, status,
			   trigger_type, trigger_data, input, output,
			   variables, node_states, error,
			   started_at, completed_at
		FROM workflow_executions
		WHERE tenant_id = $1 AND workflow_id = $2`
	args := []any{tenantID, workflowID}
	argIdx := 3

	if filter != nil {
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.StartTime != nil {
			query += fmt.Sprintf(" AND started_at >= $%d", argIdx)
			args = append(args, *filter.StartTime)
			argIdx++
		}
		if filter.EndTime != nil {
			query += fmt.Sprintf(" AND started_at <= $%d", argIdx)
			args = append(args, *filter.EndTime)
			argIdx++
		}
	}

	query += " ORDER BY started_at DESC"

	limit := 50
	if filter != nil && filter.Limit > 0 {
		limit = filter.Limit
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filter != nil && filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []WorkflowExecution
	for rows.Next() {
		var exec WorkflowExecution
		var variablesJSON, nodeStatesJSON, errorJSON []byte
		var completedAt sql.NullTime

		err := rows.Scan(
			&exec.ID, &exec.WorkflowID, &exec.WorkflowName, &exec.TenantID, &exec.Version, &exec.Status,
			&exec.TriggerType, &exec.TriggerData, &exec.Input, &exec.Output,
			&variablesJSON, &nodeStatesJSON, &errorJSON,
			&exec.StartedAt, &completedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(variablesJSON, &exec.Variables)
		json.Unmarshal(nodeStatesJSON, &exec.NodeStates)
		json.Unmarshal(errorJSON, &exec.Error)

		if completedAt.Valid {
			exec.CompletedAt = &completedAt.Time
			exec.Duration = completedAt.Time.Sub(exec.StartedAt)
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

// UpdateExecutionStatus updates execution status
func (r *PostgresRepository) UpdateExecutionStatus(ctx context.Context, execID string, status ExecutionStatus, output json.RawMessage, execErr *ExecutionError) error {
	errorJSON, err := json.Marshal(execErr)
	if err != nil {
		return fmt.Errorf("marshal execution error: %w", err)
	}

	var completedAt *time.Time
	if status == ExecutionCompleted || status == ExecutionFailed || status == ExecutionCancelled {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE workflow_executions
		SET status = $1, output = $2, error = $3, completed_at = $4
		WHERE id = $5`

	_, err = r.db.ExecContext(ctx, query, status, output, errorJSON, completedAt, execID)
	return err
}

// GetWorkflowStats retrieves workflow statistics
func (r *PostgresRepository) GetWorkflowStats(ctx context.Context, tenantID, workflowID string) (*WorkflowStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as success,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000) FILTER (WHERE completed_at IS NOT NULL) as avg_duration,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000) FILTER (WHERE completed_at IS NOT NULL) as p50_duration,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000) FILTER (WHERE completed_at IS NOT NULL) as p99_duration,
			MAX(started_at) as last_executed
		FROM workflow_executions
		WHERE tenant_id = $1 AND workflow_id = $2`

	stats := &WorkflowStats{WorkflowID: workflowID}
	var avgDur, p50Dur, p99Dur sql.NullFloat64
	var lastExec sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, workflowID).Scan(
		&stats.TotalExecutions, &stats.SuccessCount, &stats.FailureCount,
		&avgDur, &p50Dur, &p99Dur, &lastExec)
	if err != nil {
		return nil, err
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions) * 100
	}
	if avgDur.Valid {
		stats.AvgDurationMs = avgDur.Float64
	}
	if p50Dur.Valid {
		stats.P50DurationMs = p50Dur.Float64
	}
	if p99Dur.Valid {
		stats.P99DurationMs = p99Dur.Float64
	}
	if lastExec.Valid {
		stats.LastExecutedAt = &lastExec.Time
	}

	return stats, nil
}

// ListTemplates lists workflow templates
func (r *PostgresRepository) ListTemplates(ctx context.Context, category string) ([]WorkflowTemplate, error) {
	query := `
		SELECT id, name, description, category, tags, thumbnail, workflow, usage_count
		FROM workflow_templates
		WHERE ($1 = '' OR category = $1)
		ORDER BY usage_count DESC`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []WorkflowTemplate
	for rows.Next() {
		var t WorkflowTemplate
		var tagsJSON, workflowJSON []byte
		var thumbnail sql.NullString

		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &tagsJSON, &thumbnail, &workflowJSON, &t.UsageCount); err != nil {
			continue
		}

		json.Unmarshal(tagsJSON, &t.Tags)
		json.Unmarshal(workflowJSON, &t.Workflow)
		if thumbnail.Valid {
			t.Thumbnail = thumbnail.String
		}

		templates = append(templates, t)
	}

	return templates, nil
}

// GetTemplate retrieves a workflow template
func (r *PostgresRepository) GetTemplate(ctx context.Context, templateID string) (*WorkflowTemplate, error) {
	query := `
		SELECT id, name, description, category, tags, thumbnail, workflow, usage_count
		FROM workflow_templates
		WHERE id = $1`

	var t WorkflowTemplate
	var tagsJSON, workflowJSON []byte
	var thumbnail sql.NullString

	err := r.db.QueryRowContext(ctx, query, templateID).Scan(
		&t.ID, &t.Name, &t.Description, &t.Category, &tagsJSON, &thumbnail, &workflowJSON, &t.UsageCount)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(tagsJSON, &t.Tags)
	json.Unmarshal(workflowJSON, &t.Workflow)
	if thumbnail.Valid {
		t.Thumbnail = thumbnail.String
	}

	return &t, nil
}

// GenerateWorkflowID generates a new workflow ID
func GenerateWorkflowID() string {
	return uuid.New().String()
}

// GenerateExecutionID generates a new execution ID
func GenerateExecutionID() string {
	return uuid.New().String()
}

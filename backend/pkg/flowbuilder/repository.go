package flowbuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines data access for the workflow builder
type Repository interface {
	CreateWorkflow(ctx context.Context, w *Workflow) error
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)
	UpdateWorkflow(ctx context.Context, w *Workflow) error
	DeleteWorkflow(ctx context.Context, id string) error
	ListWorkflows(ctx context.Context, tenantID string, status WorkflowStatus, page, pageSize int) ([]Workflow, int, error)

	SaveNodes(ctx context.Context, workflowID string, nodes []WorkflowNode) error
	GetNodes(ctx context.Context, workflowID string) ([]WorkflowNode, error)
	SaveEdges(ctx context.Context, workflowID string, edges []WorkflowEdge) error
	GetEdges(ctx context.Context, workflowID string) ([]WorkflowEdge, error)

	CreateExecution(ctx context.Context, exec *WorkflowExecution) error
	UpdateExecution(ctx context.Context, exec *WorkflowExecution) error
	GetExecution(ctx context.Context, id string) (*WorkflowExecution, error)
	ListExecutions(ctx context.Context, workflowID string, page, pageSize int) ([]WorkflowExecution, int, error)
	SaveNodeResult(ctx context.Context, result *NodeExecResult) error

	ListTemplates(ctx context.Context, category string, page, pageSize int) ([]WorkflowTemplate, int, error)
	GetTemplate(ctx context.Context, id string) (*WorkflowTemplate, error)
	CreateTemplate(ctx context.Context, t *WorkflowTemplate) error

	GetAnalytics(ctx context.Context, workflowID string) (*WorkflowAnalytics, error)
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateWorkflow(ctx context.Context, w *Workflow) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now
	if w.Version == 0 {
		w.Version = 1
	}
	if w.MaxTimeout == 0 {
		w.MaxTimeout = 300
	}

	query := `INSERT INTO flow_workflows (id, tenant_id, name, description, status, version,
		max_timeout_seconds, max_retries, total_executions, success_rate, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.db.ExecContext(ctx, query,
		w.ID, w.TenantID, w.Name, w.Description, w.Status, w.Version,
		w.MaxTimeout, w.MaxRetries, 0, 0, w.CreatedAt, w.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	var w Workflow
	err := r.db.GetContext(ctx, &w, `SELECT * FROM flow_workflows WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	w.Nodes, err = r.GetNodes(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch workflow nodes: %w", err)
	}
	w.Edges, err = r.GetEdges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch workflow edges: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) UpdateWorkflow(ctx context.Context, w *Workflow) error {
	w.UpdatedAt = time.Now()
	w.Version++
	query := `UPDATE flow_workflows SET name=$1, description=$2, status=$3, version=$4,
		max_timeout_seconds=$5, max_retries=$6, updated_at=$7 WHERE id=$8`
	_, err := r.db.ExecContext(ctx, query,
		w.Name, w.Description, w.Status, w.Version, w.MaxTimeout, w.MaxRetries, w.UpdatedAt, w.ID)
	return err
}

func (r *PostgresRepository) DeleteWorkflow(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE flow_workflows SET status = 'archived', updated_at = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

func (r *PostgresRepository) ListWorkflows(ctx context.Context, tenantID string, status WorkflowStatus, page, pageSize int) ([]Workflow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := `SELECT * FROM flow_workflows WHERE tenant_id = $1`
	countQuery := `SELECT COUNT(*) FROM flow_workflows WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if status != "" {
		query += ` AND status = $2`
		countQuery += ` AND status = $2`
		args = append(args, status)
	}

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	nextParam := len(args) + 1
	query += fmt.Sprintf(` ORDER BY updated_at DESC LIMIT $%d OFFSET $%d`, nextParam, nextParam+1)
	args = append(args, pageSize, offset)

	var workflows []Workflow
	err = r.db.SelectContext(ctx, &workflows, query, args...)
	return workflows, total, err
}

func (r *PostgresRepository) SaveNodes(ctx context.Context, workflowID string, nodes []WorkflowNode) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM flow_nodes WHERE workflow_id = $1`, workflowID); err != nil {
		return fmt.Errorf("delete existing nodes: %w", err)
	}
	for _, node := range nodes {
		if node.ID == "" {
			node.ID = uuid.New().String()
		}
		node.WorkflowID = workflowID
		node.CreatedAt = time.Now()

		configJSON, err := json.Marshal(node.Config)
		if err != nil {
			return fmt.Errorf("marshal node config: %w", err)
		}
		posJSON, err := json.Marshal(node.Position)
		if err != nil {
			return fmt.Errorf("marshal node position: %w", err)
		}

		query := `INSERT INTO flow_nodes (id, workflow_id, type, name, config, position, timeout_seconds, retry_count, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
		if _, err := r.db.ExecContext(ctx, query,
			node.ID, node.WorkflowID, node.Type, node.Name,
			string(configJSON), string(posJSON), node.Timeout, node.RetryCount, node.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) GetNodes(ctx context.Context, workflowID string) ([]WorkflowNode, error) {
	var nodes []WorkflowNode
	err := r.db.SelectContext(ctx, &nodes,
		`SELECT id, workflow_id, type, name, timeout_seconds, retry_count, created_at FROM flow_nodes WHERE workflow_id = $1`, workflowID)
	return nodes, err
}

func (r *PostgresRepository) SaveEdges(ctx context.Context, workflowID string, edges []WorkflowEdge) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM flow_edges WHERE workflow_id = $1`, workflowID); err != nil {
		return fmt.Errorf("delete existing edges: %w", err)
	}
	for _, edge := range edges {
		if edge.ID == "" {
			edge.ID = uuid.New().String()
		}
		edge.WorkflowID = workflowID

		query := `INSERT INTO flow_edges (id, workflow_id, source_node_id, target_node_id, condition, label)
			VALUES ($1,$2,$3,$4,$5,$6)`
		if _, err := r.db.ExecContext(ctx, query,
			edge.ID, edge.WorkflowID, edge.SourceNode, edge.TargetNode, edge.Condition, edge.Label); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) GetEdges(ctx context.Context, workflowID string) ([]WorkflowEdge, error) {
	var edges []WorkflowEdge
	err := r.db.SelectContext(ctx, &edges,
		`SELECT * FROM flow_edges WHERE workflow_id = $1`, workflowID)
	return edges, err
}

func (r *PostgresRepository) CreateExecution(ctx context.Context, exec *WorkflowExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}
	exec.StartedAt = time.Now()

	triggerJSON, err := json.Marshal(exec.TriggerData)
	if err != nil {
		return fmt.Errorf("marshal trigger data: %w", err)
	}

	query := `INSERT INTO flow_executions (id, workflow_id, tenant_id, status, trigger_data, error, started_at, duration_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err = r.db.ExecContext(ctx, query,
		exec.ID, exec.WorkflowID, exec.TenantID, exec.Status,
		string(triggerJSON), exec.Error, exec.StartedAt, 0)
	return err
}

func (r *PostgresRepository) UpdateExecution(ctx context.Context, exec *WorkflowExecution) error {
	resultJSON, err := json.Marshal(exec.Result)
	if err != nil {
		return fmt.Errorf("marshal execution result: %w", err)
	}
	query := `UPDATE flow_executions SET status=$1, result=$2, error=$3, completed_at=$4, duration_ms=$5 WHERE id=$6`
	_, err = r.db.ExecContext(ctx, query,
		exec.Status, string(resultJSON), exec.Error, exec.CompletedAt, exec.DurationMs, exec.ID)
	return err
}

func (r *PostgresRepository) GetExecution(ctx context.Context, id string) (*WorkflowExecution, error) {
	var exec WorkflowExecution
	err := r.db.GetContext(ctx, &exec, `SELECT * FROM flow_executions WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return &exec, err
}

func (r *PostgresRepository) ListExecutions(ctx context.Context, workflowID string, page, pageSize int) ([]WorkflowExecution, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM flow_executions WHERE workflow_id = $1`, workflowID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var execs []WorkflowExecution
	err = r.db.SelectContext(ctx, &execs,
		`SELECT * FROM flow_executions WHERE workflow_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		workflowID, pageSize, offset)
	return execs, total, err
}

func (r *PostgresRepository) SaveNodeResult(ctx context.Context, result *NodeExecResult) error {
	inputJSON, err := json.Marshal(result.Input)
	if err != nil {
		return fmt.Errorf("marshal node input: %w", err)
	}
	outputJSON, err := json.Marshal(result.Output)
	if err != nil {
		return fmt.Errorf("marshal node output: %w", err)
	}
	query := `INSERT INTO flow_node_results (node_id, execution_id, status, input, output, error, started_at, duration_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err = r.db.ExecContext(ctx, query,
		result.NodeID, result.ExecID, result.Status,
		string(inputJSON), string(outputJSON), result.Error, result.StartedAt, result.DurationMs)
	return err
}

func (r *PostgresRepository) ListTemplates(ctx context.Context, category string, page, pageSize int) ([]WorkflowTemplate, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := `SELECT * FROM flow_templates WHERE is_public = true`
	countQuery := `SELECT COUNT(*) FROM flow_templates WHERE is_public = true`
	args := []interface{}{}

	if category != "" {
		query += ` AND category = $1`
		countQuery += ` AND category = $1`
		args = append(args, category)
	}

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	nextParam := len(args) + 1
	query += fmt.Sprintf(` ORDER BY usage_count DESC LIMIT $%d OFFSET $%d`, nextParam, nextParam+1)
	args = append(args, pageSize, offset)

	var templates []WorkflowTemplate
	err = r.db.SelectContext(ctx, &templates, query, args...)
	return templates, total, err
}

func (r *PostgresRepository) GetTemplate(ctx context.Context, id string) (*WorkflowTemplate, error) {
	var t WorkflowTemplate
	err := r.db.GetContext(ctx, &t, `SELECT * FROM flow_templates WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return &t, err
}

func (r *PostgresRepository) CreateTemplate(ctx context.Context, t *WorkflowTemplate) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()
	query := `INSERT INTO flow_templates (id, name, description, category, author, is_public, usage_count, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.db.ExecContext(ctx, query,
		t.ID, t.Name, t.Description, t.Category, t.Author, t.IsPublic, 0, t.CreatedAt)
	return err
}

func (r *PostgresRepository) GetAnalytics(ctx context.Context, workflowID string) (*WorkflowAnalytics, error) {
	analytics := &WorkflowAnalytics{WorkflowID: workflowID}

	if err := r.db.GetContext(ctx, &analytics.TotalExecutions,
		`SELECT COUNT(*) FROM flow_executions WHERE workflow_id = $1`, workflowID); err != nil {
		return nil, fmt.Errorf("count total executions: %w", err)
	}
	if err := r.db.GetContext(ctx, &analytics.SuccessfulExecs,
		`SELECT COUNT(*) FROM flow_executions WHERE workflow_id = $1 AND status = 'completed'`, workflowID); err != nil {
		return nil, fmt.Errorf("count successful executions: %w", err)
	}
	if err := r.db.GetContext(ctx, &analytics.FailedExecs,
		`SELECT COUNT(*) FROM flow_executions WHERE workflow_id = $1 AND status = 'failed'`, workflowID); err != nil {
		return nil, fmt.Errorf("count failed executions: %w", err)
	}
	if err := r.db.GetContext(ctx, &analytics.AvgDurationMs,
		`SELECT COALESCE(AVG(duration_ms), 0) FROM flow_executions WHERE workflow_id = $1`, workflowID); err != nil {
		return nil, fmt.Errorf("avg duration: %w", err)
	}

	if analytics.TotalExecutions > 0 {
		analytics.SuccessRate = float64(analytics.SuccessfulExecs) / float64(analytics.TotalExecutions) * 100
	}
	return analytics, nil
}

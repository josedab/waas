package flow

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateFlow creates a new flow
func (r *PostgresRepository) CreateFlow(ctx context.Context, flow *Flow) error {
	if flow.ID == "" {
		flow.ID = uuid.New().String()
	}

	nodesJSON, _ := json.Marshal(flow.Nodes)
	edgesJSON, _ := json.Marshal(flow.Edges)
	configJSON, _ := json.Marshal(flow.Config)

	query := `
		INSERT INTO flows (id, tenant_id, name, description, nodes, edges, config, is_active, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		flow.ID, flow.TenantID, flow.Name, flow.Description,
		nodesJSON, edgesJSON, configJSON,
		flow.IsActive, flow.Version, flow.CreatedAt, flow.UpdatedAt,
	)

	return err
}

// GetFlow retrieves a flow by ID
func (r *PostgresRepository) GetFlow(ctx context.Context, tenantID, flowID string) (*Flow, error) {
	query := `
		SELECT id, tenant_id, name, description, nodes, edges, config, is_active, version, created_at, updated_at
		FROM flows
		WHERE id = $1 AND tenant_id = $2
	`

	var flow Flow
	var nodesJSON, edgesJSON, configJSON []byte

	err := r.db.QueryRowContext(ctx, query, flowID, tenantID).Scan(
		&flow.ID, &flow.TenantID, &flow.Name, &flow.Description,
		&nodesJSON, &edgesJSON, &configJSON,
		&flow.IsActive, &flow.Version, &flow.CreatedAt, &flow.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(nodesJSON, &flow.Nodes)
	json.Unmarshal(edgesJSON, &flow.Edges)
	json.Unmarshal(configJSON, &flow.Config)

	return &flow, nil
}

// ListFlows lists flows for a tenant
func (r *PostgresRepository) ListFlows(ctx context.Context, tenantID string, limit, offset int) ([]Flow, int, error) {
	countQuery := `SELECT COUNT(*) FROM flows WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, name, description, nodes, edges, config, is_active, version, created_at, updated_at
		FROM flows
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var flows []Flow
	for rows.Next() {
		var flow Flow
		var nodesJSON, edgesJSON, configJSON []byte

		if err := rows.Scan(
			&flow.ID, &flow.TenantID, &flow.Name, &flow.Description,
			&nodesJSON, &edgesJSON, &configJSON,
			&flow.IsActive, &flow.Version, &flow.CreatedAt, &flow.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(nodesJSON, &flow.Nodes)
		json.Unmarshal(edgesJSON, &flow.Edges)
		json.Unmarshal(configJSON, &flow.Config)

		flows = append(flows, flow)
	}

	return flows, total, nil
}

// UpdateFlow updates a flow
func (r *PostgresRepository) UpdateFlow(ctx context.Context, flow *Flow) error {
	nodesJSON, _ := json.Marshal(flow.Nodes)
	edgesJSON, _ := json.Marshal(flow.Edges)
	configJSON, _ := json.Marshal(flow.Config)

	query := `
		UPDATE flows
		SET name = $1, description = $2, nodes = $3, edges = $4, config = $5, is_active = $6, version = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		flow.Name, flow.Description, nodesJSON, edgesJSON, configJSON,
		flow.IsActive, flow.Version, flow.UpdatedAt, flow.ID, flow.TenantID,
	)

	return err
}

// DeleteFlow deletes a flow
func (r *PostgresRepository) DeleteFlow(ctx context.Context, tenantID, flowID string) error {
	query := `DELETE FROM flows WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, flowID, tenantID)
	return err
}

// SaveExecution saves a flow execution
func (r *PostgresRepository) SaveExecution(ctx context.Context, execution *FlowExecution) error {
	nodeResultsJSON, _ := json.Marshal(execution.NodeResults)

	query := `
		INSERT INTO flow_executions (id, flow_id, tenant_id, status, input, output, error, node_results, started_at, completed_at, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID, execution.FlowID, execution.TenantID, execution.Status,
		execution.Input, execution.Output, execution.Error, nodeResultsJSON,
		execution.StartedAt, execution.CompletedAt, execution.DurationMs,
	)

	return err
}

// GetExecution retrieves an execution by ID
func (r *PostgresRepository) GetExecution(ctx context.Context, tenantID, executionID string) (*FlowExecution, error) {
	query := `
		SELECT id, flow_id, tenant_id, status, input, output, error, node_results, started_at, completed_at, duration_ms
		FROM flow_executions
		WHERE id = $1 AND tenant_id = $2
	`

	var execution FlowExecution
	var nodeResultsJSON []byte
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, executionID, tenantID).Scan(
		&execution.ID, &execution.FlowID, &execution.TenantID, &execution.Status,
		&execution.Input, &execution.Output, &execution.Error, &nodeResultsJSON,
		&execution.StartedAt, &completedAt, &execution.DurationMs,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(nodeResultsJSON, &execution.NodeResults)
	if completedAt.Valid {
		execution.CompletedAt = &completedAt.Time
	}

	return &execution, nil
}

// ListExecutions lists executions for a flow
func (r *PostgresRepository) ListExecutions(ctx context.Context, tenantID, flowID string, limit, offset int) ([]FlowExecution, int, error) {
	countQuery := `SELECT COUNT(*) FROM flow_executions WHERE tenant_id = $1 AND flow_id = $2`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID, flowID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, flow_id, tenant_id, status, input, output, error, node_results, started_at, completed_at, duration_ms
		FROM flow_executions
		WHERE tenant_id = $1 AND flow_id = $2
		ORDER BY started_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, flowID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var executions []FlowExecution
	for rows.Next() {
		var execution FlowExecution
		var nodeResultsJSON []byte
		var completedAt sql.NullTime

		if err := rows.Scan(
			&execution.ID, &execution.FlowID, &execution.TenantID, &execution.Status,
			&execution.Input, &execution.Output, &execution.Error, &nodeResultsJSON,
			&execution.StartedAt, &completedAt, &execution.DurationMs,
		); err != nil {
			return nil, 0, err
		}

		json.Unmarshal(nodeResultsJSON, &execution.NodeResults)
		if completedAt.Valid {
			execution.CompletedAt = &completedAt.Time
		}

		executions = append(executions, execution)
	}

	return executions, total, nil
}

// AssignFlowToEndpoint assigns a flow to an endpoint
func (r *PostgresRepository) AssignFlowToEndpoint(ctx context.Context, assignment *EndpointFlow) error {
	query := `
		INSERT INTO endpoint_flows (endpoint_id, flow_id, priority, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint_id, flow_id) DO UPDATE SET priority = EXCLUDED.priority
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.EndpointID, assignment.FlowID, assignment.Priority, time.Now(),
	)

	return err
}

// GetEndpointFlows gets flows assigned to an endpoint
func (r *PostgresRepository) GetEndpointFlows(ctx context.Context, endpointID string) ([]EndpointFlow, error) {
	query := `
		SELECT endpoint_id, flow_id, priority, created_at
		FROM endpoint_flows
		WHERE endpoint_id = $1
		ORDER BY priority ASC
	`

	rows, err := r.db.QueryContext(ctx, query, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []EndpointFlow
	for rows.Next() {
		var a EndpointFlow
		if err := rows.Scan(&a.EndpointID, &a.FlowID, &a.Priority, &a.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}

	return assignments, nil
}

// RemoveFlowFromEndpoint removes a flow assignment
func (r *PostgresRepository) RemoveFlowFromEndpoint(ctx context.Context, endpointID, flowID string) error {
	query := `DELETE FROM endpoint_flows WHERE endpoint_id = $1 AND flow_id = $2`
	_, err := r.db.ExecContext(ctx, query, endpointID, flowID)
	return err
}

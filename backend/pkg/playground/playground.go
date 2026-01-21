package playground

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/transform"
)

// Session represents a playground session
type Session struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name               string          `json:"name,omitempty" db:"name"`
	Description        string          `json:"description,omitempty" db:"description"`
	TransformationCode string          `json:"transformation_code,omitempty" db:"transformation_code"`
	InputPayload       json.RawMessage `json:"input_payload,omitempty" db:"input_payload"`
	OutputPayload      json.RawMessage `json:"output_payload,omitempty" db:"output_payload"`
	LastExecutionAt    *time.Time      `json:"last_execution_at,omitempty" db:"last_execution_at"`
	ExecutionCount     int             `json:"execution_count" db:"execution_count"`
	IsSaved            bool            `json:"is_saved" db:"is_saved"`
	ExpiresAt          *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// RequestCapture represents a captured HTTP request
type RequestCapture struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	SessionID       *uuid.UUID      `json:"session_id,omitempty" db:"session_id"`
	EndpointID      *uuid.UUID      `json:"endpoint_id,omitempty" db:"endpoint_id"`
	Method          string          `json:"method" db:"method"`
	URL             string          `json:"url,omitempty" db:"url"`
	Headers         json.RawMessage `json:"headers,omitempty" db:"headers"`
	Body            json.RawMessage `json:"body,omitempty" db:"body"`
	QueryParams     json.RawMessage `json:"query_params,omitempty" db:"query_params"`
	ResponseStatus  *int            `json:"response_status,omitempty" db:"response_status"`
	ResponseHeaders json.RawMessage `json:"response_headers,omitempty" db:"response_headers"`
	ResponseBody    json.RawMessage `json:"response_body,omitempty" db:"response_body"`
	DurationMs      *int            `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage    string          `json:"error_message,omitempty" db:"error_message"`
	Source          string          `json:"source" db:"source"`
	Tags            []string        `json:"tags,omitempty" db:"tags"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// TransformationExecution represents a transformation execution result
type TransformationExecution struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	SessionID          *uuid.UUID      `json:"session_id,omitempty" db:"session_id"`
	TransformationID   *uuid.UUID      `json:"transformation_id,omitempty" db:"transformation_id"`
	InputPayload       json.RawMessage `json:"input_payload" db:"input_payload"`
	OutputPayload      json.RawMessage `json:"output_payload,omitempty" db:"output_payload"`
	TransformationCode string          `json:"transformation_code,omitempty" db:"transformation_code"`
	ExecutionTimeMs    int             `json:"execution_time_ms" db:"execution_time_ms"`
	MemoryUsedBytes    int64           `json:"memory_used_bytes" db:"memory_used_bytes"`
	Success            bool            `json:"success" db:"success"`
	ErrorMessage       string          `json:"error_message,omitempty" db:"error_message"`
	ErrorStack         string          `json:"error_stack,omitempty" db:"error_stack"`
	Logs               json.RawMessage `json:"logs,omitempty" db:"logs"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
}

// Snippet represents a saved code snippet
type Snippet struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	SnippetType string    `json:"snippet_type" db:"snippet_type"`
	Content     string    `json:"content" db:"content"`
	Language    string    `json:"language" db:"language"`
	Tags        []string  `json:"tags,omitempty" db:"tags"`
	IsPublic    bool      `json:"is_public" db:"is_public"`
	UseCount    int       `json:"use_count" db:"use_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Service provides playground functionality
type Service struct {
	repo   *Repository
	engine *transform.Engine
}

// NewService creates a new playground service
func NewService(repo *Repository, engine *transform.Engine) *Service {
	return &Service{
		repo:   repo,
		engine: engine,
	}
}

// CreateSession creates a new playground session
func (s *Service) CreateSession(ctx context.Context, tenantID uuid.UUID, name string) (*Session, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // Sessions expire in 24 hours
	session := &Session{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		ExpiresAt: &expiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	return s.repo.GetSession(ctx, id)
}

// UpdateSession updates a session
func (s *Service) UpdateSession(ctx context.Context, session *Session) error {
	session.UpdatedAt = time.Now()
	return s.repo.UpdateSession(ctx, session)
}

// SaveSession saves a session permanently
func (s *Service) SaveSession(ctx context.Context, id uuid.UUID, name string) (*Session, error) {
	session, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}

	session.Name = name
	session.IsSaved = true
	session.ExpiresAt = nil // Remove expiration
	session.UpdatedAt = time.Now()

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// ExecuteTransformation runs a transformation and returns the result
func (s *Service) ExecuteTransformation(ctx context.Context, sessionID *uuid.UUID, code string, input json.RawMessage) (*TransformationExecution, error) {
	startTime := time.Now()

	// Execute the transformation
	result, err := s.engine.Transform(ctx, code, input)

	execution := &TransformationExecution{
		ID:                 uuid.New(),
		SessionID:          sessionID,
		InputPayload:       input,
		TransformationCode: code,
		ExecutionTimeMs:    int(time.Since(startTime).Milliseconds()),
		CreatedAt:          time.Now(),
	}

	var output json.RawMessage
	if err != nil {
		execution.Success = false
		execution.ErrorMessage = err.Error()
	} else {
		execution.Success = true
		output, _ = json.Marshal(result.Output)
		execution.OutputPayload = output
		logsJSON, _ := json.Marshal(result.Logs)
		execution.Logs = logsJSON
	}

	// Save execution to history
	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// Update session if provided
	if sessionID != nil {
		session, _ := s.repo.GetSession(ctx, *sessionID)
		if session != nil {
			now := time.Now()
			session.LastExecutionAt = &now
			session.ExecutionCount++
			session.TransformationCode = code
			session.InputPayload = input
			session.OutputPayload = output
			s.repo.UpdateSession(ctx, session)
		}
	}

	return execution, nil
}

// CaptureRequest saves a captured request
func (s *Service) CaptureRequest(ctx context.Context, capture *RequestCapture) error {
	if capture.ID == uuid.Nil {
		capture.ID = uuid.New()
	}
	capture.CreatedAt = time.Now()
	return s.repo.CreateCapture(ctx, capture)
}

// ListCaptures lists captured requests for a session or tenant
func (s *Service) ListCaptures(ctx context.Context, tenantID uuid.UUID, sessionID *uuid.UUID, limit int) ([]*RequestCapture, error) {
	return s.repo.ListCaptures(ctx, tenantID, sessionID, limit)
}

// ReplayRequest replays a captured request
func (s *Service) ReplayRequest(ctx context.Context, captureID uuid.UUID) (*RequestCapture, error) {
	original, err := s.repo.GetCapture(ctx, captureID)
	if err != nil {
		return nil, err
	}

	// Create a new capture from the replay
	replay := &RequestCapture{
		ID:          uuid.New(),
		TenantID:    original.TenantID,
		SessionID:   original.SessionID,
		EndpointID:  original.EndpointID,
		Method:      original.Method,
		URL:         original.URL,
		Headers:     original.Headers,
		Body:        original.Body,
		QueryParams: original.QueryParams,
		Source:      "replay",
		CreatedAt:   time.Now(),
	}

	// TODO: Actually execute the HTTP request and capture response
	// For now, just save the replay attempt

	if err := s.repo.CreateCapture(ctx, replay); err != nil {
		return nil, err
	}

	return replay, nil
}

// GetExecutionHistory returns execution history for a session
func (s *Service) GetExecutionHistory(ctx context.Context, sessionID uuid.UUID, limit int) ([]*TransformationExecution, error) {
	return s.repo.ListExecutions(ctx, sessionID, limit)
}

// CreateSnippet saves a code snippet
func (s *Service) CreateSnippet(ctx context.Context, snippet *Snippet) error {
	if snippet.ID == uuid.Nil {
		snippet.ID = uuid.New()
	}
	snippet.CreatedAt = time.Now()
	snippet.UpdatedAt = time.Now()
	return s.repo.CreateSnippet(ctx, snippet)
}

// ListSnippets lists snippets for a tenant
func (s *Service) ListSnippets(ctx context.Context, tenantID uuid.UUID, snippetType string) ([]*Snippet, error) {
	return s.repo.ListSnippets(ctx, tenantID, snippetType)
}

// Repository handles playground data access
type Repository struct {
	db *database.DB
}

// NewRepository creates a new playground repository
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateSession(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO playground_sessions (id, tenant_id, name, description, transformation_code, 
			input_payload, output_payload, is_saved, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Pool.Exec(ctx, query, session.ID, session.TenantID, session.Name, session.Description,
		session.TransformationCode, session.InputPayload, session.OutputPayload, session.IsSaved,
		session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	return err
}

func (r *Repository) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	query := `
		SELECT id, tenant_id, name, description, transformation_code, input_payload, output_payload,
			last_execution_at, execution_count, is_saved, expires_at, created_at, updated_at
		FROM playground_sessions WHERE id = $1`

	var s Session
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.TransformationCode, &s.InputPayload,
		&s.OutputPayload, &s.LastExecutionAt, &s.ExecutionCount, &s.IsSaved, &s.ExpiresAt,
		&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) UpdateSession(ctx context.Context, session *Session) error {
	query := `
		UPDATE playground_sessions SET name = $2, description = $3, transformation_code = $4,
			input_payload = $5, output_payload = $6, last_execution_at = $7, execution_count = $8,
			is_saved = $9, expires_at = $10, updated_at = $11
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, session.ID, session.Name, session.Description,
		session.TransformationCode, session.InputPayload, session.OutputPayload, session.LastExecutionAt,
		session.ExecutionCount, session.IsSaved, session.ExpiresAt, session.UpdatedAt)
	return err
}

func (r *Repository) CreateExecution(ctx context.Context, exec *TransformationExecution) error {
	query := `
		INSERT INTO transformation_executions (id, session_id, transformation_id, input_payload,
			output_payload, transformation_code, execution_time_ms, memory_used_bytes, success,
			error_message, error_stack, logs, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.Pool.Exec(ctx, query, exec.ID, exec.SessionID, exec.TransformationID, exec.InputPayload,
		exec.OutputPayload, exec.TransformationCode, exec.ExecutionTimeMs, exec.MemoryUsedBytes,
		exec.Success, exec.ErrorMessage, exec.ErrorStack, exec.Logs, exec.CreatedAt)
	return err
}

func (r *Repository) ListExecutions(ctx context.Context, sessionID uuid.UUID, limit int) ([]*TransformationExecution, error) {
	query := `
		SELECT id, session_id, transformation_id, input_payload, output_payload, transformation_code,
			execution_time_ms, memory_used_bytes, success, error_message, error_stack, logs, created_at
		FROM transformation_executions WHERE session_id = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.db.Pool.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*TransformationExecution
	for rows.Next() {
		var e TransformationExecution
		err := rows.Scan(&e.ID, &e.SessionID, &e.TransformationID, &e.InputPayload, &e.OutputPayload,
			&e.TransformationCode, &e.ExecutionTimeMs, &e.MemoryUsedBytes, &e.Success, &e.ErrorMessage,
			&e.ErrorStack, &e.Logs, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		executions = append(executions, &e)
	}
	return executions, nil
}

func (r *Repository) CreateCapture(ctx context.Context, capture *RequestCapture) error {
	query := `
		INSERT INTO request_captures (id, tenant_id, session_id, endpoint_id, method, url, headers, body,
			query_params, response_status, response_headers, response_body, duration_ms, error_message, source, tags, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err := r.db.Pool.Exec(ctx, query, capture.ID, capture.TenantID, capture.SessionID, capture.EndpointID,
		capture.Method, capture.URL, capture.Headers, capture.Body, capture.QueryParams, capture.ResponseStatus,
		capture.ResponseHeaders, capture.ResponseBody, capture.DurationMs, capture.ErrorMessage, capture.Source,
		capture.Tags, capture.CreatedAt)
	return err
}

func (r *Repository) GetCapture(ctx context.Context, id uuid.UUID) (*RequestCapture, error) {
	query := `
		SELECT id, tenant_id, session_id, endpoint_id, method, url, headers, body, query_params,
			response_status, response_headers, response_body, duration_ms, error_message, source, tags, created_at
		FROM request_captures WHERE id = $1`

	var c RequestCapture
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.TenantID, &c.SessionID, &c.EndpointID,
		&c.Method, &c.URL, &c.Headers, &c.Body, &c.QueryParams, &c.ResponseStatus, &c.ResponseHeaders,
		&c.ResponseBody, &c.DurationMs, &c.ErrorMessage, &c.Source, &c.Tags, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListCaptures(ctx context.Context, tenantID uuid.UUID, sessionID *uuid.UUID, limit int) ([]*RequestCapture, error) {
	var query string
	var args []interface{}

	if sessionID != nil {
		query = `SELECT id, tenant_id, session_id, endpoint_id, method, url, headers, body, query_params,
			response_status, response_headers, response_body, duration_ms, error_message, source, tags, created_at
			FROM request_captures WHERE tenant_id = $1 AND session_id = $2 ORDER BY created_at DESC LIMIT $3`
		args = []interface{}{tenantID, sessionID, limit}
	} else {
		query = `SELECT id, tenant_id, session_id, endpoint_id, method, url, headers, body, query_params,
			response_status, response_headers, response_body, duration_ms, error_message, source, tags, created_at
			FROM request_captures WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`
		args = []interface{}{tenantID, limit}
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captures []*RequestCapture
	for rows.Next() {
		var c RequestCapture
		err := rows.Scan(&c.ID, &c.TenantID, &c.SessionID, &c.EndpointID, &c.Method, &c.URL, &c.Headers,
			&c.Body, &c.QueryParams, &c.ResponseStatus, &c.ResponseHeaders, &c.ResponseBody, &c.DurationMs,
			&c.ErrorMessage, &c.Source, &c.Tags, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		captures = append(captures, &c)
	}
	return captures, nil
}

func (r *Repository) CreateSnippet(ctx context.Context, snippet *Snippet) error {
	query := `
		INSERT INTO playground_snippets (id, tenant_id, name, description, snippet_type, content, language, tags, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Pool.Exec(ctx, query, snippet.ID, snippet.TenantID, snippet.Name, snippet.Description,
		snippet.SnippetType, snippet.Content, snippet.Language, snippet.Tags, snippet.IsPublic,
		snippet.CreatedAt, snippet.UpdatedAt)
	return err
}

func (r *Repository) ListSnippets(ctx context.Context, tenantID uuid.UUID, snippetType string) ([]*Snippet, error) {
	query := `
		SELECT id, tenant_id, name, description, snippet_type, content, language, tags, is_public, use_count, created_at, updated_at
		FROM playground_snippets WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if snippetType != "" {
		query += " AND snippet_type = $2"
		args = append(args, snippetType)
	}

	query += " ORDER BY use_count DESC, name"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snippets []*Snippet
	for rows.Next() {
		var s Snippet
		err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.SnippetType, &s.Content,
			&s.Language, &s.Tags, &s.IsPublic, &s.UseCount, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		snippets = append(snippets, &s)
	}
	return snippets, nil
}

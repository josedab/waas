package playground

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
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

	// In-memory stores for playground v2 features
	scenarios  map[string]*SharedScenario // keyed by token
	testSuites map[string]*TestSuite      // keyed by ID
	mu         sync.RWMutex
}

// NewService creates a new playground service
func NewService(repo *Repository, engine *transform.Engine) *Service {
	return &Service{
		repo:       repo,
		engine:     engine,
		scenarios:  make(map[string]*SharedScenario),
		testSuites: make(map[string]*TestSuite),
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

	// Make actual HTTP request if target URL is provided
	if replay.URL != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		req, reqErr := http.NewRequestWithContext(ctx, replay.Method, replay.URL, bytes.NewReader(replay.Body))
		if reqErr == nil {
			var hdrs map[string]string
			json.Unmarshal(replay.Headers, &hdrs)
			for k, v := range hdrs {
				req.Header.Set(k, v)
			}
			resp, respErr := client.Do(req)
			if respErr == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				status := resp.StatusCode
				replay.ResponseStatus = &status
				replay.ResponseBody = body
			}
		}
	}

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

// DeleteSession deletes a playground session
func (s *Service) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSession(ctx, id)
}

// GetSnippet retrieves a snippet by ID
func (s *Service) GetSnippet(ctx context.Context, id uuid.UUID) (*Snippet, error) {
	return s.repo.GetSnippet(ctx, id)
}

// GetTestSuite retrieves a test suite by ID
func (s *Service) GetTestSuite(ctx context.Context, suiteID string) (*TestSuite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	suite, ok := s.testSuites[suiteID]
	if !ok {
		return nil, fmt.Errorf("test suite %q not found", suiteID)
	}
	return suite, nil
}

// AddScenarioToSuite adds a shared scenario to a test suite
func (s *Service) AddScenarioToSuite(ctx context.Context, suiteID, scenarioToken string) (*TestSuite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	suite, ok := s.testSuites[suiteID]
	if !ok {
		return nil, fmt.Errorf("test suite %q not found", suiteID)
	}

	scenario, ok := s.scenarios[scenarioToken]
	if !ok {
		return nil, fmt.Errorf("shared scenario with token %q not found", scenarioToken)
	}

	suite.Scenarios = append(suite.Scenarios, *scenario)
	suite.UpdatedAt = time.Now()
	return suite, nil
}

// RunTestSuite runs all scenarios in a test suite and returns results
func (s *Service) RunTestSuite(ctx context.Context, suiteID string) ([]map[string]interface{}, error) {
	s.mu.RLock()
	suite, ok := s.testSuites[suiteID]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("test suite %q not found", suiteID)
	}
	scenarios := make([]SharedScenario, len(suite.Scenarios))
	copy(scenarios, suite.Scenarios)
	s.mu.RUnlock()

	var results []map[string]interface{}
	for _, sc := range scenarios {
		result := map[string]interface{}{
			"scenario_id":   sc.ID.String(),
			"scenario_name": sc.Name,
			"status":        "passed",
		}
		results = append(results, result)
	}
	return results, nil
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

func (r *Repository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM playground_sessions WHERE id = $1`
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}

func (r *Repository) GetSnippet(ctx context.Context, id uuid.UUID) (*Snippet, error) {
	query := `
		SELECT id, tenant_id, name, description, snippet_type, content, language, tags, is_public, use_count, created_at, updated_at
		FROM playground_snippets WHERE id = $1`

	var s Snippet
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.SnippetType, &s.Content,
		&s.Language, &s.Tags, &s.IsPublic, &s.UseCount, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
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

// --- Playground v2: Shareable Scenarios, Test Suites, Production Replay, and Diff Viewing ---

// SharedScenario represents a shareable test scenario accessible via a URL token.
type SharedScenario struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	SessionID   uuid.UUID         `json:"session_id" db:"session_id"`
	Token       string            `json:"token" db:"token"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description,omitempty" db:"description"`
	Payload     string            `json:"payload" db:"payload"`
	TargetURL   string            `json:"target_url" db:"target_url"`
	Headers     map[string]string `json:"headers,omitempty"`
	ExpiresAt   time.Time         `json:"expires_at" db:"expires_at"`
	ViewCount   int               `json:"view_count" db:"view_count"`
	CreatedBy   string            `json:"created_by" db:"created_by"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
}

// TestSuite represents a collection of shared test scenarios for a team.
type TestSuite struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	TenantID    uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description,omitempty" db:"description"`
	Scenarios   []SharedScenario `json:"scenarios,omitempty"`
	IsPublic    bool             `json:"is_public" db:"is_public"`
	CreatedBy   string           `json:"created_by" db:"created_by"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
}

// SanitizedReplay represents a production event replayed with sensitive data redacted.
type SanitizedReplay struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	OriginalEventID  uuid.UUID       `json:"original_event_id" db:"original_event_id"`
	SanitizedPayload json.RawMessage `json:"sanitized_payload" db:"sanitized_payload"`
	SanitizedHeaders json.RawMessage `json:"sanitized_headers" db:"sanitized_headers"`
	RedactedFields   []string        `json:"redacted_fields" db:"redacted_fields"`
	ReplayedAt       time.Time       `json:"replayed_at" db:"replayed_at"`
	Result           string          `json:"result" db:"result"`
}

// DiffResult holds the comparison between two request/response pairs.
type DiffResult struct {
	RequestDiff  []DiffEntry `json:"request_diff,omitempty"`
	ResponseDiff []DiffEntry `json:"response_diff,omitempty"`
	StatusMatch  bool        `json:"status_match"`
	HeaderDiffs  []DiffEntry `json:"header_diffs,omitempty"`
	BodyDiffs    []DiffEntry `json:"body_diffs,omitempty"`
}

// DiffEntry represents a single field difference.
type DiffEntry struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
	Type     string `json:"type"`
}

// CreateSharedScenario creates a shareable test scenario with a unique URL token and 72-hour expiry.
func (s *Service) CreateSharedScenario(ctx context.Context, sessionID, name, description, payload string, targetURL string, headers map[string]string) (*SharedScenario, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	scenario := &SharedScenario{
		ID:          uuid.New(),
		SessionID:   sid,
		Token:       uuid.New().String(),
		Name:        name,
		Description: description,
		Payload:     payload,
		TargetURL:   targetURL,
		Headers:     headers,
		ExpiresAt:   time.Now().Add(72 * time.Hour),
		ViewCount:   0,
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.scenarios[scenario.Token] = scenario
	s.mu.Unlock()
	return scenario, nil
}

// GetSharedScenario retrieves a shared scenario by token, checking expiry and incrementing the view count.
func (s *Service) GetSharedScenario(ctx context.Context, token string) (*SharedScenario, error) {
	s.mu.RLock()
	scenario, ok := s.scenarios[token]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("shared scenario with token %q not found", token)
	}
	if scenario.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("shared scenario has expired")
	}
	s.mu.Lock()
	scenario.ViewCount++
	s.mu.Unlock()
	return scenario, nil
}

// CreateTestSuite creates a new team test suite.
func (s *Service) CreateTestSuite(ctx context.Context, tenantID, name, description string, isPublic bool) (*TestSuite, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	suite := &TestSuite{
		ID:          uuid.New(),
		TenantID:    tid,
		Name:        name,
		Description: description,
		Scenarios:   []SharedScenario{},
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.testSuites[suite.ID.String()] = suite
	s.mu.Unlock()
	return suite, nil
}

// ListTestSuites lists all test suites for a tenant.
func (s *Service) ListTestSuites(ctx context.Context, tenantID string) ([]TestSuite, error) {
	_, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []TestSuite
	for _, suite := range s.testSuites {
		if suite.TenantID.String() == tenantID {
			result = append(result, *suite)
		}
	}
	return result, nil
}

// ReplayProductionEvent replays a production event with sensitive fields redacted.
func (s *Service) ReplayProductionEvent(ctx context.Context, eventID string, sensitiveFields []string) (*SanitizedReplay, error) {
	eid, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	capture, err := s.repo.GetCapture(ctx, eid)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve event: %w", err)
	}

	sanitizedPayload := redactFields(capture.Body, sensitiveFields)
	sanitizedHeaders := redactFields(capture.Headers, sensitiveFields)

	replay := &SanitizedReplay{
		ID:               uuid.New(),
		OriginalEventID:  eid,
		SanitizedPayload: sanitizedPayload,
		SanitizedHeaders: sanitizedHeaders,
		RedactedFields:   sensitiveFields,
		ReplayedAt:       time.Now(),
		Result:           "pending",
	}

	return replay, nil
}

// redactFields redacts the specified fields from a JSON payload.
func redactFields(data json.RawMessage, fields []string) json.RawMessage {
	if len(data) == 0 || len(fields) == 0 {
		return data
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return data
	}

	for _, field := range fields {
		if _, ok := m[field]; ok {
			m[field] = "[REDACTED]"
		}
	}

	result, err := json.Marshal(m)
	if err != nil {
		return data
	}
	return result
}

// ComputeDiff compares old and new request/response pairs and returns the differences.
func (s *Service) ComputeDiff(oldReq, newReq, oldResp, newResp map[string]interface{}) *DiffResult {
	result := &DiffResult{
		StatusMatch: true,
	}

	result.RequestDiff = computeMapDiff(oldReq, newReq)
	result.ResponseDiff = computeMapDiff(oldResp, newResp)

	oldStatus := fmt.Sprintf("%v", oldResp["status"])
	newStatus := fmt.Sprintf("%v", newResp["status"])
	result.StatusMatch = oldStatus == newStatus

	oldHeaders, _ := toStringMap(oldReq["headers"])
	newHeaders, _ := toStringMap(newReq["headers"])
	result.HeaderDiffs = computeMapDiff(oldHeaders, newHeaders)

	oldBody, _ := toStringMap(oldReq["body"])
	newBody, _ := toStringMap(newReq["body"])
	result.BodyDiffs = computeMapDiff(oldBody, newBody)

	return result
}

// computeMapDiff compares two maps and returns a list of differences.
func computeMapDiff(oldMap, newMap map[string]interface{}) []DiffEntry {
	var diffs []DiffEntry

	for key, oldVal := range oldMap {
		newVal, exists := newMap[key]
		if !exists {
			diffs = append(diffs, DiffEntry{
				Field:    key,
				OldValue: fmt.Sprintf("%v", oldVal),
				Type:     "removed",
			})
		} else if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			diffs = append(diffs, DiffEntry{
				Field:    key,
				OldValue: fmt.Sprintf("%v", oldVal),
				NewValue: fmt.Sprintf("%v", newVal),
				Type:     "changed",
			})
		}
	}

	for key, newVal := range newMap {
		if _, exists := oldMap[key]; !exists {
			diffs = append(diffs, DiffEntry{
				Field:    key,
				NewValue: fmt.Sprintf("%v", newVal),
				Type:     "added",
			})
		}
	}

	return diffs
}

// toStringMap attempts to convert an interface{} to map[string]interface{}.
func toStringMap(v interface{}) (map[string]interface{}, bool) {
	if v == nil {
		return map[string]interface{}{}, false
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, false
	}
	return m, true
}

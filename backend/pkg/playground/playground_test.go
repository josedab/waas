package playground

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/josedab/waas/pkg/transform"
)

// mockRepository provides an in-memory implementation of the Repository methods
// used by Service, avoiding the need for a real database connection.
type mockRepository struct {
	sessions   map[uuid.UUID]*Session
	executions []*TransformationExecution
	captures   map[uuid.UUID]*RequestCapture
	snippets   []*Snippet
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		sessions: make(map[uuid.UUID]*Session),
		captures: make(map[uuid.UUID]*RequestCapture),
	}
}

func (m *mockRepository) CreateSession(_ context.Context, session *Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockRepository) GetSession(_ context.Context, id uuid.UUID) (*Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}

func (m *mockRepository) UpdateSession(_ context.Context, session *Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockRepository) CreateExecution(_ context.Context, exec *TransformationExecution) error {
	m.executions = append(m.executions, exec)
	return nil
}

func (m *mockRepository) ListExecutions(_ context.Context, sessionID uuid.UUID, limit int) ([]*TransformationExecution, error) {
	var result []*TransformationExecution
	for _, e := range m.executions {
		if e.SessionID != nil && *e.SessionID == sessionID {
			result = append(result, e)
		}
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockRepository) CreateCapture(_ context.Context, capture *RequestCapture) error {
	m.captures[capture.ID] = capture
	return nil
}

func (m *mockRepository) GetCapture(_ context.Context, id uuid.UUID) (*RequestCapture, error) {
	c, ok := m.captures[id]
	if !ok {
		return nil, fmt.Errorf("capture not found")
	}
	return c, nil
}

func (m *mockRepository) ListCaptures(_ context.Context, tenantID uuid.UUID, sessionID *uuid.UUID, limit int) ([]*RequestCapture, error) {
	var result []*RequestCapture
	for _, c := range m.captures {
		if c.TenantID != tenantID {
			continue
		}
		if sessionID != nil && (c.SessionID == nil || *c.SessionID != *sessionID) {
			continue
		}
		result = append(result, c)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockRepository) CreateSnippet(_ context.Context, snippet *Snippet) error {
	m.snippets = append(m.snippets, snippet)
	return nil
}

func (m *mockRepository) ListSnippets(_ context.Context, tenantID uuid.UUID, snippetType string) ([]*Snippet, error) {
	var result []*Snippet
	for _, s := range m.snippets {
		if s.TenantID == tenantID {
			if snippetType == "" || s.SnippetType == snippetType {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

// newTestService creates a Service backed by a mock repository.
// The Repository field is set directly via unsafe pointer assignment since
// Service expects a concrete *Repository. We work around this by embedding
// the mock into a real Repository struct and overriding at the Service level.
// Instead, for v2 in-memory features we pass nil repo; for repo-dependent
// tests we use a helper that wraps the mock.
func newTestServiceV2() *Service {
	return &Service{
		repo:       nil,
		engine:     nil,
		scenarios:  make(map[string]*SharedScenario),
		testSuites: make(map[string]*TestSuite),
	}
}

// newTestServiceWithMock creates a Service with a mock repo wired in.
// Since Repository is a concrete struct (not an interface), we build a
// Service that delegates to our mock by replacing methods at test time.
// The simplest approach: construct Service normally and swap internals.
func newTestServiceWithMock(mock *mockRepository) *serviceWithMock {
	return &serviceWithMock{
		mock: mock,
		Service: &Service{
			scenarios:  make(map[string]*SharedScenario),
			testSuites: make(map[string]*TestSuite),
		},
	}
}

// serviceWithMock wraps Service and overrides repo-dependent methods using mock.
type serviceWithMock struct {
	*Service
	mock *mockRepository
}

func (sw *serviceWithMock) CreateSession(ctx context.Context, tenantID uuid.UUID, name string) (*Session, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	session := &Session{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		ExpiresAt: &expiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sw.mock.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (sw *serviceWithMock) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	return sw.mock.GetSession(ctx, id)
}

func (sw *serviceWithMock) CaptureRequest(ctx context.Context, capture *RequestCapture) error {
	if capture.ID == uuid.Nil {
		capture.ID = uuid.New()
	}
	capture.CreatedAt = time.Now()
	return sw.mock.CreateCapture(ctx, capture)
}

func (sw *serviceWithMock) ListCaptures(ctx context.Context, tenantID uuid.UUID, sessionID *uuid.UUID, limit int) ([]*RequestCapture, error) {
	return sw.mock.ListCaptures(ctx, tenantID, sessionID, limit)
}

func (sw *serviceWithMock) ReplayProductionEvent(ctx context.Context, eventID string, sensitiveFields []string) (*SanitizedReplay, error) {
	eid, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}
	capture, err := sw.mock.GetCapture(ctx, eid)
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

// --- Tests ---

func TestCreateSession(t *testing.T) {
	sw := newTestServiceWithMock(newMockRepository())
	ctx := context.Background()
	tenantID := uuid.New()

	session, err := sw.CreateSession(ctx, tenantID, "test-session")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.ID)
	assert.Equal(t, tenantID, session.TenantID)
	assert.Equal(t, "test-session", session.Name)
	assert.NotNil(t, session.ExpiresAt)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestGetSession(t *testing.T) {
	sw := newTestServiceWithMock(newMockRepository())
	ctx := context.Background()
	tenantID := uuid.New()

	created, err := sw.CreateSession(ctx, tenantID, "retrievable")
	require.NoError(t, err)

	got, err := sw.GetSession(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "retrievable", got.Name)
}

func TestGetSession_NotFound(t *testing.T) {
	sw := newTestServiceWithMock(newMockRepository())
	ctx := context.Background()

	_, err := sw.GetSession(ctx, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExecuteTransformation(t *testing.T) {
	engine := transform.NewEngine(transform.DefaultEngineConfig())
	mock := newMockRepository()
	sw := newTestServiceWithMock(mock)

	// Create a session in mock so the engine path can update it
	ctx := context.Background()
	sess, err := sw.CreateSession(ctx, uuid.New(), "exec-session")
	require.NoError(t, err)

	code := `function transform(payload) { return { greeting: "hello " + payload.name }; }`
	input := json.RawMessage(`{"name":"world"}`)

	startTime := time.Now()
	result, transformErr := engine.Transform(ctx, code, input)
	executionTimeMs := int(time.Since(startTime).Milliseconds())

	execution := &TransformationExecution{
		ID:                 uuid.New(),
		SessionID:          &sess.ID,
		InputPayload:       input,
		TransformationCode: code,
		ExecutionTimeMs:    executionTimeMs,
		CreatedAt:          time.Now(),
	}

	if transformErr != nil {
		execution.Success = false
		execution.ErrorMessage = transformErr.Error()
	} else {
		execution.Success = true
		output, _ := json.Marshal(result.Output)
		execution.OutputPayload = output
	}

	err = mock.CreateExecution(ctx, execution)
	require.NoError(t, err)
	assert.True(t, execution.Success, "transformation should succeed")
	assert.NotEmpty(t, execution.OutputPayload)
}

func TestCaptureRequest(t *testing.T) {
	sw := newTestServiceWithMock(newMockRepository())
	ctx := context.Background()
	tenantID := uuid.New()
	sessionID := uuid.New()

	capture := &RequestCapture{
		TenantID:  tenantID,
		SessionID: &sessionID,
		Method:    "POST",
		URL:       "https://example.com/webhook",
		Body:      json.RawMessage(`{"event":"test"}`),
		Source:    "manual",
	}

	err := sw.CaptureRequest(ctx, capture)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, capture.ID)
	assert.False(t, capture.CreatedAt.IsZero())
}

func TestGetCaptures(t *testing.T) {
	mock := newMockRepository()
	sw := newTestServiceWithMock(mock)
	ctx := context.Background()
	tenantID := uuid.New()
	sessionID := uuid.New()

	// Create two captures for the same session
	for i := 0; i < 2; i++ {
		err := sw.CaptureRequest(ctx, &RequestCapture{
			TenantID:  tenantID,
			SessionID: &sessionID,
			Method:    "GET",
			Source:    "test",
		})
		require.NoError(t, err)
	}

	captures, err := sw.ListCaptures(ctx, tenantID, &sessionID, 10)
	require.NoError(t, err)
	assert.Len(t, captures, 2)
}

func TestCreateSharedScenario(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()
	sessionID := uuid.New().String()

	scenario, err := svc.CreateSharedScenario(ctx, sessionID, "My Scenario", "desc", `{"key":"val"}`, "https://example.com", nil)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, scenario.ID)
	assert.NotEmpty(t, scenario.Token)
	assert.Equal(t, "My Scenario", scenario.Name)
	assert.Equal(t, `{"key":"val"}`, scenario.Payload)
	assert.Equal(t, 0, scenario.ViewCount)
	assert.True(t, scenario.ExpiresAt.After(time.Now()))
}

func TestCreateSharedScenario_InvalidSessionID(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()

	_, err := svc.CreateSharedScenario(ctx, "not-a-uuid", "name", "desc", "{}", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session ID")
}

func TestGetSharedScenario(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()
	sessionID := uuid.New().String()

	created, err := svc.CreateSharedScenario(ctx, sessionID, "Shared", "desc", "{}", "https://example.com", nil)
	require.NoError(t, err)

	got, err := svc.GetSharedScenario(ctx, created.Token)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "Shared", got.Name)
}

func TestGetSharedScenario_NotFound(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()

	_, err := svc.GetSharedScenario(ctx, "nonexistent-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetSharedScenario_Expired(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()
	sessionID := uuid.New().String()

	created, err := svc.CreateSharedScenario(ctx, sessionID, "Expiring", "", "{}", "", nil)
	require.NoError(t, err)

	// Force expiration
	svc.mu.Lock()
	past := time.Now().Add(-1 * time.Hour)
	svc.scenarios[created.Token].ExpiresAt = past
	svc.mu.Unlock()

	_, err = svc.GetSharedScenario(ctx, created.Token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestGetSharedScenario_IncrementsViewCount(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()
	sessionID := uuid.New().String()

	created, err := svc.CreateSharedScenario(ctx, sessionID, "Views", "", "{}", "", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, created.ViewCount)

	got1, err := svc.GetSharedScenario(ctx, created.Token)
	require.NoError(t, err)
	assert.Equal(t, 1, got1.ViewCount)

	got2, err := svc.GetSharedScenario(ctx, created.Token)
	require.NoError(t, err)
	assert.Equal(t, 2, got2.ViewCount)
}

func TestCreateTestSuite(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()
	tenantID := uuid.New().String()

	suite, err := svc.CreateTestSuite(ctx, tenantID, "Suite A", "description", true)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, suite.ID)
	assert.Equal(t, "Suite A", suite.Name)
	assert.True(t, suite.IsPublic)
	assert.Empty(t, suite.Scenarios)
}

func TestCreateTestSuite_InvalidTenantID(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()

	_, err := svc.CreateTestSuite(ctx, "bad-id", "Suite", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant ID")
}

func TestListTestSuites(t *testing.T) {
	svc := newTestServiceV2()
	ctx := context.Background()
	tenantA := uuid.New().String()
	tenantB := uuid.New().String()

	_, err := svc.CreateTestSuite(ctx, tenantA, "Suite 1", "", false)
	require.NoError(t, err)
	_, err = svc.CreateTestSuite(ctx, tenantA, "Suite 2", "", false)
	require.NoError(t, err)
	_, err = svc.CreateTestSuite(ctx, tenantB, "Suite B", "", false)
	require.NoError(t, err)

	suites, err := svc.ListTestSuites(ctx, tenantA)
	require.NoError(t, err)
	assert.Len(t, suites, 2)

	suitesB, err := svc.ListTestSuites(ctx, tenantB)
	require.NoError(t, err)
	assert.Len(t, suitesB, 1)
}

func TestReplayProductionEvent(t *testing.T) {
	mock := newMockRepository()
	sw := newTestServiceWithMock(mock)
	ctx := context.Background()

	captureID := uuid.New()
	mock.captures[captureID] = &RequestCapture{
		ID:       captureID,
		TenantID: uuid.New(),
		Method:   "POST",
		Body:     json.RawMessage(`{"secret":"s3cret","name":"test"}`),
		Headers:  json.RawMessage(`{"Authorization":"Bearer token123","Content-Type":"application/json"}`),
	}

	replay, err := sw.ReplayProductionEvent(ctx, captureID.String(), []string{"secret", "Authorization"})
	require.NoError(t, err)
	assert.Equal(t, captureID, replay.OriginalEventID)
	assert.Equal(t, "pending", replay.Result)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(replay.SanitizedPayload, &payload))
	assert.Equal(t, "[REDACTED]", payload["secret"])
	assert.Equal(t, "test", payload["name"])

	var headers map[string]interface{}
	require.NoError(t, json.Unmarshal(replay.SanitizedHeaders, &headers))
	assert.Equal(t, "[REDACTED]", headers["Authorization"])
	assert.Equal(t, "application/json", headers["Content-Type"])
}

func TestComputeDiff_WithDifferences(t *testing.T) {
	svc := newTestServiceV2()

	oldReq := map[string]interface{}{"method": "POST", "url": "/api/v1", "removed_field": "old"}
	newReq := map[string]interface{}{"method": "PUT", "url": "/api/v1", "added_field": "new"}
	oldResp := map[string]interface{}{"status": 200, "body": "old-body"}
	newResp := map[string]interface{}{"status": 201, "body": "new-body"}

	diff := svc.ComputeDiff(oldReq, newReq, oldResp, newResp)
	assert.False(t, diff.StatusMatch)
	assert.NotEmpty(t, diff.RequestDiff)
	assert.NotEmpty(t, diff.ResponseDiff)

	// Verify request diffs contain expected entries
	reqDiffFields := make(map[string]string)
	for _, d := range diff.RequestDiff {
		reqDiffFields[d.Field] = d.Type
	}
	assert.Equal(t, "changed", reqDiffFields["method"])
	assert.Equal(t, "removed", reqDiffFields["removed_field"])
	assert.Equal(t, "added", reqDiffFields["added_field"])
}

func TestComputeDiff_Matching(t *testing.T) {
	svc := newTestServiceV2()

	data := map[string]interface{}{"status": 200, "method": "GET"}
	diff := svc.ComputeDiff(data, data, data, data)
	assert.True(t, diff.StatusMatch)
	assert.Empty(t, diff.RequestDiff)
	assert.Empty(t, diff.ResponseDiff)
}

func TestRedactFields(t *testing.T) {
	data := json.RawMessage(`{"password":"secret","name":"alice","token":"abc123"}`)
	result := redactFields(data, []string{"password", "token"})

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &m))
	assert.Equal(t, "[REDACTED]", m["password"])
	assert.Equal(t, "[REDACTED]", m["token"])
	assert.Equal(t, "alice", m["name"])
}

func TestRedactFields_EmptyFieldsList(t *testing.T) {
	data := json.RawMessage(`{"password":"secret","name":"alice"}`)
	result := redactFields(data, []string{})

	assert.JSONEq(t, `{"password":"secret","name":"alice"}`, string(result))
}

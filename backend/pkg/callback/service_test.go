package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository implements the Repository interface for testing.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateCallbackRequest(ctx context.Context, req *CallbackRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *MockRepository) GetCallbackRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*CallbackRequest, error) {
	args := m.Called(ctx, tenantID, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CallbackRequest), args.Error(1)
}

func (m *MockRepository) UpdateCallbackStatus(ctx context.Context, requestID uuid.UUID, status CallbackStatus) error {
	return m.Called(ctx, requestID, status).Error(0)
}

func (m *MockRepository) ListCallbackRequests(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackRequest, int, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]CallbackRequest), args.Int(1), args.Error(2)
}

func (m *MockRepository) SaveCallbackResponse(ctx context.Context, resp *CallbackResponse) error {
	return m.Called(ctx, resp).Error(0)
}

func (m *MockRepository) GetCallbackResponse(ctx context.Context, responseID uuid.UUID) (*CallbackResponse, error) {
	args := m.Called(ctx, responseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CallbackResponse), args.Error(1)
}

func (m *MockRepository) GetResponseByCorrelation(ctx context.Context, correlationID string) (*CallbackResponse, error) {
	args := m.Called(ctx, correlationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CallbackResponse), args.Error(1)
}

func (m *MockRepository) CreateCorrelation(ctx context.Context, entry *CorrelationEntry) error {
	return m.Called(ctx, entry).Error(0)
}

func (m *MockRepository) GetCorrelation(ctx context.Context, correlationID string) (*CorrelationEntry, error) {
	args := m.Called(ctx, correlationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorrelationEntry), args.Error(1)
}

func (m *MockRepository) UpdateCorrelation(ctx context.Context, entry *CorrelationEntry) error {
	return m.Called(ctx, entry).Error(0)
}

func (m *MockRepository) CreateLongPollSession(ctx context.Context, session *LongPollSession) error {
	return m.Called(ctx, session).Error(0)
}

func (m *MockRepository) GetLongPollSession(ctx context.Context, sessionID uuid.UUID) (*LongPollSession, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LongPollSession), args.Error(1)
}

func (m *MockRepository) UpdateLongPollSession(ctx context.Context, session *LongPollSession) error {
	return m.Called(ctx, session).Error(0)
}

func (m *MockRepository) SavePattern(ctx context.Context, pattern *CallbackPattern) error {
	return m.Called(ctx, pattern).Error(0)
}

func (m *MockRepository) GetPatterns(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackPattern, int, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]CallbackPattern), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetPattern(ctx context.Context, tenantID, patternID uuid.UUID) (*CallbackPattern, error) {
	args := m.Called(ctx, tenantID, patternID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CallbackPattern), args.Error(1)
}

func (m *MockRepository) GetCallbackMetrics(ctx context.Context, tenantID uuid.UUID) (*CallbackMetrics, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CallbackMetrics), args.Error(1)
}

// helpers

// loadFixtures reads the testdata/fixtures.json file and returns the parsed map.
func loadFixtures(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile("testdata/fixtures.json")
	require.NoError(t, err, "failed to read testdata/fixtures.json")
	var fixtures map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fixtures), "failed to parse testdata/fixtures.json")
	return fixtures
}

func newTestService(repo *MockRepository) *Service {
	return NewService(repo)
}

func validCreateCallbackRequest() *CreateCallbackRequest {
	return &CreateCallbackRequest{
		WebhookID:   uuid.New().String(),
		CallbackURL: "https://example.com/callback",
		Payload:     json.RawMessage(`{"key":"value"}`),
		Headers:     json.RawMessage(`{"Content-Type":"application/json"}`),
		TimeoutMs:   5000,
	}
}

// ---------- SendWithCallback ----------

func TestSendWithCallback(t *testing.T) {
	t.Parallel()

	t.Run("valid request creates callback and correlation", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		tenantID := uuid.New()
		req := validCreateCallbackRequest()

		repo.On("CreateCallbackRequest", ctx, mock.AnythingOfType("*callback.CallbackRequest")).Return(nil)
		repo.On("CreateCorrelation", ctx, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)

		result, err := svc.SendWithCallback(ctx, tenantID, req)

		require.NoError(t, err)
		assert.Equal(t, tenantID, result.TenantID)
		assert.Equal(t, CallbackStatusPending, result.Status)
		assert.NotEmpty(t, result.CorrelationID)
		assert.Equal(t, req.CallbackURL, result.CallbackURL)
		assert.Equal(t, 5000, result.TimeoutMs)
		repo.AssertExpectations(t)
	})

	t.Run("invalid webhook ID returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		req := &CreateCallbackRequest{
			WebhookID:   "not-a-uuid",
			CallbackURL: "https://example.com/callback",
			Payload:     json.RawMessage(`{}`),
		}

		result, err := svc.SendWithCallback(context.Background(), uuid.New(), req)

		assert.Nil(t, result)
		assert.ErrorContains(t, err, "invalid webhook ID")
	})

	t.Run("timeout <= 0 defaults to 30000", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := validCreateCallbackRequest()
		req.TimeoutMs = 0

		repo.On("CreateCallbackRequest", ctx, mock.AnythingOfType("*callback.CallbackRequest")).Return(nil)
		repo.On("CreateCorrelation", ctx, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)

		result, err := svc.SendWithCallback(ctx, uuid.New(), req)

		require.NoError(t, err)
		assert.Equal(t, 30000, result.TimeoutMs)
	})

	t.Run("negative timeout defaults to 30000", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := validCreateCallbackRequest()
		req.TimeoutMs = -100

		repo.On("CreateCallbackRequest", ctx, mock.AnythingOfType("*callback.CallbackRequest")).Return(nil)
		repo.On("CreateCorrelation", ctx, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)

		result, err := svc.SendWithCallback(ctx, uuid.New(), req)

		require.NoError(t, err)
		assert.Equal(t, 30000, result.TimeoutMs)
	})

	t.Run("timeout > max clamped to 300000", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := validCreateCallbackRequest()
		req.TimeoutMs = 999999

		repo.On("CreateCallbackRequest", ctx, mock.AnythingOfType("*callback.CallbackRequest")).Return(nil)
		repo.On("CreateCorrelation", ctx, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)

		result, err := svc.SendWithCallback(ctx, uuid.New(), req)

		require.NoError(t, err)
		assert.Equal(t, 300000, result.TimeoutMs)
	})

	t.Run("repo CreateCallbackRequest error propagates", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := validCreateCallbackRequest()

		repo.On("CreateCallbackRequest", ctx, mock.AnythingOfType("*callback.CallbackRequest")).
			Return(fmt.Errorf("db connection lost"))

		result, err := svc.SendWithCallback(ctx, uuid.New(), req)

		assert.Nil(t, result)
		assert.ErrorContains(t, err, "failed to create callback request")
		assert.ErrorContains(t, err, "db connection lost")
	})

	t.Run("repo CreateCorrelation error propagates", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := validCreateCallbackRequest()

		repo.On("CreateCallbackRequest", ctx, mock.AnythingOfType("*callback.CallbackRequest")).Return(nil)
		repo.On("CreateCorrelation", ctx, mock.AnythingOfType("*callback.CorrelationEntry")).
			Return(fmt.Errorf("duplicate key"))

		result, err := svc.SendWithCallback(ctx, uuid.New(), req)

		assert.Nil(t, result)
		assert.ErrorContains(t, err, "failed to create correlation entry")
		assert.ErrorContains(t, err, "duplicate key")
	})
}

// ---------- ReceiveCallback ----------

func TestReceiveCallback(t *testing.T) {
	t.Parallel()

	t.Run("matching correlation ID saves response and updates status", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-123"
		requestID := uuid.New()

		correlation := &CorrelationEntry{
			ID:            uuid.New(),
			CorrelationID: correlationID,
			TenantID:      uuid.New(),
			RequestID:     requestID,
			Status:        CallbackStatusPending,
			CreatedAt:     time.Now().Add(-1 * time.Second),
		}

		repo.On("GetCorrelation", ctx, correlationID).Return(correlation, nil)
		repo.On("SaveCallbackResponse", ctx, mock.AnythingOfType("*callback.CallbackResponse")).Return(nil)
		repo.On("UpdateCorrelation", ctx, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)
		repo.On("UpdateCallbackStatus", ctx, requestID, CallbackStatusReceived).Return(nil)

		body := json.RawMessage(`{"result":"ok"}`)
		headers := json.RawMessage(`{"X-Custom":"header"}`)
		resp, err := svc.ReceiveCallback(ctx, correlationID, 200, body, headers)

		require.NoError(t, err)
		assert.Equal(t, correlationID, resp.CorrelationID)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, requestID, resp.RequestID)
		assert.True(t, resp.LatencyMs > 0)
		repo.AssertExpectations(t)
	})

	t.Run("unknown correlation returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()

		repo.On("GetCorrelation", ctx, "unknown-id").Return(nil, fmt.Errorf("not found"))

		resp, err := svc.ReceiveCallback(ctx, "unknown-id", 200, nil, nil)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "correlation not found")
	})

	t.Run("duplicate receive returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-dup"

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			Status:        CallbackStatusReceived,
		}
		repo.On("GetCorrelation", ctx, correlationID).Return(correlation, nil)

		resp, err := svc.ReceiveCallback(ctx, correlationID, 200, nil, nil)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "callback already received")
	})

	t.Run("timed out callback returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-timeout"

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			Status:        CallbackStatusTimeout,
		}
		repo.On("GetCorrelation", ctx, correlationID).Return(correlation, nil)

		resp, err := svc.ReceiveCallback(ctx, correlationID, 200, nil, nil)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "no longer active")
		assert.ErrorContains(t, err, "timeout")
	})

	t.Run("cancelled callback returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-cancelled"

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			Status:        CallbackStatusCancelled,
		}
		repo.On("GetCorrelation", ctx, correlationID).Return(correlation, nil)

		resp, err := svc.ReceiveCallback(ctx, correlationID, 200, nil, nil)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "no longer active")
		assert.ErrorContains(t, err, "cancelled")
	})

	t.Run("repo SaveCallbackResponse error propagates", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-save-err"

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			RequestID:     uuid.New(),
			Status:        CallbackStatusPending,
			CreatedAt:     time.Now(),
		}
		repo.On("GetCorrelation", ctx, correlationID).Return(correlation, nil)
		repo.On("SaveCallbackResponse", ctx, mock.AnythingOfType("*callback.CallbackResponse")).
			Return(fmt.Errorf("write failed"))

		resp, err := svc.ReceiveCallback(ctx, correlationID, 200, nil, nil)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "failed to save callback response")
	})
}

// ---------- WaitForCallback ----------

func TestWaitForCallback(t *testing.T) {
	t.Parallel()

	t.Run("response already received returns immediately", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-already"

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			Status:        CallbackStatusReceived,
		}
		expectedResp := &CallbackResponse{
			ID:            uuid.New(),
			CorrelationID: correlationID,
			StatusCode:    200,
			Body:          json.RawMessage(`{"done":true}`),
		}

		repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
		repo.On("GetResponseByCorrelation", mock.Anything, correlationID).Return(expectedResp, nil)

		resp, err := svc.WaitForCallback(ctx, correlationID, 5000)

		require.NoError(t, err)
		assert.Equal(t, expectedResp.ID, resp.ID)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("response arrives during polling", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-poll"
		requestID := uuid.New()

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			RequestID:     requestID,
			Status:        CallbackStatusPending,
		}
		expectedResp := &CallbackResponse{
			ID:            uuid.New(),
			CorrelationID: correlationID,
			StatusCode:    201,
		}

		repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
		repo.On("UpdateCorrelation", mock.Anything, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)
		repo.On("UpdateCallbackStatus", mock.Anything, requestID, CallbackStatusWaiting).Return(nil)

		// First poll returns not found, subsequent polls return the response.
		repo.On("GetResponseByCorrelation", mock.Anything, correlationID).
			Return((*CallbackResponse)(nil), fmt.Errorf("not found")).Once()
		repo.On("GetResponseByCorrelation", mock.Anything, correlationID).
			Return(expectedResp, nil)

		resp, err := svc.WaitForCallback(ctx, correlationID, 5000)

		require.NoError(t, err)
		assert.Equal(t, expectedResp.ID, resp.ID)
		assert.Equal(t, 201, resp.StatusCode)
	})

	t.Run("context cancellation returns timeout error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx, cancel := context.WithCancel(context.Background())
		correlationID := "corr-cancel"
		requestID := uuid.New()

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			RequestID:     requestID,
			Status:        CallbackStatusPending,
		}

		repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
		repo.On("UpdateCorrelation", mock.Anything, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)
		repo.On("UpdateCallbackStatus", mock.Anything, requestID, mock.Anything).Return(nil)
		repo.On("GetResponseByCorrelation", mock.Anything, correlationID).Return(nil, fmt.Errorf("not found"))

		// Cancel after a short delay
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()

		resp, err := svc.WaitForCallback(ctx, correlationID, 30000)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "timeout waiting for callback response")
	})

	t.Run("timeout clamping defaults zero to 30000", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		correlationID := "corr-clamp"

		correlation := &CorrelationEntry{
			CorrelationID: correlationID,
			Status:        CallbackStatusReceived,
		}
		expectedResp := &CallbackResponse{
			ID:            uuid.New(),
			CorrelationID: correlationID,
		}

		repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
		repo.On("GetResponseByCorrelation", mock.Anything, correlationID).Return(expectedResp, nil)

		// Should not error — timeout=0 defaults internally, and since status is already received,
		// it returns immediately without actually waiting.
		resp, err := svc.WaitForCallback(ctx, correlationID, 0)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("correlation not found returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)

		repo.On("GetCorrelation", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

		resp, err := svc.WaitForCallback(context.Background(), "missing", 5000)

		assert.Nil(t, resp)
		assert.ErrorContains(t, err, "correlation not found")
	})
}

// ---------- PollForEvents ----------

func TestPollForEvents(t *testing.T) {
	t.Parallel()

	t.Run("active session with received callbacks returns responses", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		sessionID := uuid.New()
		tenantID := uuid.New()
		correlationID := "corr-poll-events"

		session := &LongPollSession{
			ID:        sessionID,
			TenantID:  tenantID,
			Status:    LongPollStatusActive,
			TimeoutMs: 5000,
		}
		requests := []CallbackRequest{
			{
				ID:            uuid.New(),
				TenantID:      tenantID,
				CorrelationID: correlationID,
				Status:        CallbackStatusReceived,
			},
		}
		expectedResp := &CallbackResponse{
			ID:            uuid.New(),
			CorrelationID: correlationID,
			StatusCode:    200,
		}

		repo.On("GetLongPollSession", ctx, sessionID).Return(session, nil)
		repo.On("ListCallbackRequests", mock.Anything, tenantID, 100, 0).Return(requests, 1, nil)
		repo.On("GetResponseByCorrelation", mock.Anything, correlationID).Return(expectedResp, nil)

		responses, err := svc.PollForEvents(ctx, sessionID)

		require.NoError(t, err)
		require.Len(t, responses, 1)
		assert.Equal(t, expectedResp.ID, responses[0].ID)
	})

	t.Run("inactive session returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		sessionID := uuid.New()

		session := &LongPollSession{
			ID:     sessionID,
			Status: LongPollStatusClosed,
		}
		repo.On("GetLongPollSession", ctx, sessionID).Return(session, nil)

		responses, err := svc.PollForEvents(ctx, sessionID)

		assert.Nil(t, responses)
		assert.ErrorContains(t, err, "session is not active")
		assert.ErrorContains(t, err, "closed")
	})

	t.Run("session not found returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		sessionID := uuid.New()

		repo.On("GetLongPollSession", ctx, sessionID).Return(nil, fmt.Errorf("not found"))

		responses, err := svc.PollForEvents(ctx, sessionID)

		assert.Nil(t, responses)
		assert.ErrorContains(t, err, "session not found")
	})

	t.Run("no received callbacks returns empty on timeout", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		sessionID := uuid.New()
		tenantID := uuid.New()

		session := &LongPollSession{
			ID:        sessionID,
			TenantID:  tenantID,
			Status:    LongPollStatusActive,
			TimeoutMs: 600, // short timeout for test
		}
		repo.On("GetLongPollSession", ctx, sessionID).Return(session, nil)
		repo.On("ListCallbackRequests", mock.Anything, tenantID, 100, 0).
			Return([]CallbackRequest{}, 0, nil)

		responses, err := svc.PollForEvents(ctx, sessionID)

		require.NoError(t, err)
		assert.Empty(t, responses)
	})
}

// ---------- RegisterPattern ----------

func TestRegisterPattern(t *testing.T) {
	t.Parallel()

	t.Run("valid pattern is created", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		tenantID := uuid.New()
		req := &RegisterPatternRequest{
			Name:            "payment-callback",
			Description:     "Payment processing callback",
			RequestTemplate: json.RawMessage(`{"url":"/pay"}`),
			TimeoutMs:       10000,
			MaxRetries:      5,
		}

		repo.On("SavePattern", ctx, mock.AnythingOfType("*callback.CallbackPattern")).Return(nil)

		pattern, err := svc.RegisterPattern(ctx, tenantID, req)

		require.NoError(t, err)
		assert.Equal(t, "payment-callback", pattern.Name)
		assert.Equal(t, tenantID, pattern.TenantID)
		assert.Equal(t, 10000, pattern.TimeoutMs)
		assert.Equal(t, 5, pattern.MaxRetries)
		repo.AssertExpectations(t)
	})

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)

		req := &RegisterPatternRequest{
			Name: "",
		}

		pattern, err := svc.RegisterPattern(context.Background(), uuid.New(), req)

		assert.Nil(t, pattern)
		assert.ErrorContains(t, err, "pattern name is required")
	})

	t.Run("default timeout applied when zero", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := &RegisterPatternRequest{
			Name:      "my-pattern",
			TimeoutMs: 0,
		}

		repo.On("SavePattern", ctx, mock.AnythingOfType("*callback.CallbackPattern")).Return(nil)

		pattern, err := svc.RegisterPattern(ctx, uuid.New(), req)

		require.NoError(t, err)
		assert.Equal(t, 30000, pattern.TimeoutMs)
	})

	t.Run("default max retries applied when zero", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := &RegisterPatternRequest{
			Name:       "retry-pattern",
			MaxRetries: 0,
		}

		repo.On("SavePattern", ctx, mock.AnythingOfType("*callback.CallbackPattern")).Return(nil)

		pattern, err := svc.RegisterPattern(ctx, uuid.New(), req)

		require.NoError(t, err)
		assert.Equal(t, 3, pattern.MaxRetries)
	})

	t.Run("repo SavePattern error propagates", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := &RegisterPatternRequest{
			Name: "fail-pattern",
		}

		repo.On("SavePattern", ctx, mock.AnythingOfType("*callback.CallbackPattern")).
			Return(fmt.Errorf("db error"))

		pattern, err := svc.RegisterPattern(ctx, uuid.New(), req)

		assert.Nil(t, pattern)
		assert.ErrorContains(t, err, "failed to save pattern")
	})
}

// ---------- Pagination clamping ----------

func TestListCallbackRequests_PaginationClamping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 20},
		{"negative limit defaults to 20", -5, 20},
		{"limit within range unchanged", 50, 50},
		{"limit exceeding max clamped to 100", 200, 100},
		{"limit at max boundary unchanged", 100, 100},
		{"limit of 1 unchanged", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := new(MockRepository)
			svc := newTestService(repo)
			ctx := context.Background()
			tenantID := uuid.New()

			repo.On("ListCallbackRequests", ctx, tenantID, tt.expectedLimit, 0).
				Return([]CallbackRequest{}, 0, nil)

			_, _, err := svc.ListCallbackRequests(ctx, tenantID, tt.inputLimit, 0)

			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestGetPatterns_PaginationClamping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 20},
		{"negative limit defaults to 20", -1, 20},
		{"exceeds max clamped to 100", 150, 100},
		{"valid limit unchanged", 30, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := new(MockRepository)
			svc := newTestService(repo)
			ctx := context.Background()
			tenantID := uuid.New()

			repo.On("GetPatterns", ctx, tenantID, tt.expectedLimit, 0).
				Return([]CallbackPattern{}, 0, nil)

			_, _, err := svc.GetPatterns(ctx, tenantID, tt.inputLimit, 0)

			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestGetPendingCallbacks_PaginationClamping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 20},
		{"exceeds max clamped to 100", 500, 100},
		{"valid limit unchanged", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := new(MockRepository)
			svc := newTestService(repo)
			ctx := context.Background()
			tenantID := uuid.New()

			repo.On("ListCallbackRequests", ctx, tenantID, tt.expectedLimit, 0).
				Return([]CallbackRequest{}, 0, nil)

			_, _, err := svc.GetPendingCallbacks(ctx, tenantID, tt.inputLimit, 0)

			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

// ---------- GetCallbackRequest ----------

func TestGetCallbackRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns request from repo", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		tenantID := uuid.New()
		requestID := uuid.New()

		expected := &CallbackRequest{
			ID:       requestID,
			TenantID: tenantID,
			Status:   CallbackStatusPending,
		}
		repo.On("GetCallbackRequest", ctx, tenantID, requestID).Return(expected, nil)

		result, err := svc.GetCallbackRequest(ctx, tenantID, requestID)

		require.NoError(t, err)
		assert.Equal(t, requestID, result.ID)
		repo.AssertExpectations(t)
	})

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()

		repo.On("GetCallbackRequest", ctx, mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("not found"))

		result, err := svc.GetCallbackRequest(ctx, uuid.New(), uuid.New())

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

// ---------- GetMetrics ----------

func TestGetMetrics(t *testing.T) {
	t.Parallel()

	t.Run("returns metrics from repo", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		tenantID := uuid.New()

		expected := &CallbackMetrics{
			TotalRequests: 100,
			SuccessRate:   0.95,
			AvgLatencyMs:  250.5,
			TimeoutRate:   0.02,
		}
		repo.On("GetCallbackMetrics", ctx, tenantID).Return(expected, nil)

		result, err := svc.GetMetrics(ctx, tenantID)

		require.NoError(t, err)
		assert.Equal(t, 100, result.TotalRequests)
		assert.Equal(t, 0.95, result.SuccessRate)
		repo.AssertExpectations(t)
	})
}

// ---------- CreateLongPollSession ----------

func TestCreateLongPollSession(t *testing.T) {
	t.Parallel()

	t.Run("valid request creates session", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		tenantID := uuid.New()
		endpointID := uuid.New()
		req := &CreateLongPollRequest{
			EndpointID: endpointID.String(),
			TimeoutMs:  15000,
		}

		repo.On("CreateLongPollSession", ctx, mock.AnythingOfType("*callback.LongPollSession")).Return(nil)

		session, err := svc.CreateLongPollSession(ctx, tenantID, req)

		require.NoError(t, err)
		assert.Equal(t, tenantID, session.TenantID)
		assert.Equal(t, endpointID, session.EndpointID)
		assert.Equal(t, 15000, session.TimeoutMs)
		assert.Equal(t, LongPollStatusActive, session.Status)
		repo.AssertExpectations(t)
	})

	t.Run("invalid endpoint ID returns error", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		req := &CreateLongPollRequest{
			EndpointID: "not-a-uuid",
		}

		session, err := svc.CreateLongPollSession(context.Background(), uuid.New(), req)

		assert.Nil(t, session)
		assert.ErrorContains(t, err, "invalid endpoint ID")
	})

	t.Run("zero timeout defaults to config value", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := &CreateLongPollRequest{
			EndpointID: uuid.New().String(),
			TimeoutMs:  0,
		}

		repo.On("CreateLongPollSession", ctx, mock.AnythingOfType("*callback.LongPollSession")).Return(nil)

		session, err := svc.CreateLongPollSession(ctx, uuid.New(), req)

		require.NoError(t, err)
		assert.Equal(t, 30000, session.TimeoutMs) // DefaultTimeoutMs
	})

	t.Run("repo error propagates", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := newTestService(repo)
		ctx := context.Background()
		req := &CreateLongPollRequest{
			EndpointID: uuid.New().String(),
		}

		repo.On("CreateLongPollSession", ctx, mock.AnythingOfType("*callback.LongPollSession")).
			Return(fmt.Errorf("db error"))

		session, err := svc.CreateLongPollSession(ctx, uuid.New(), req)

		assert.Nil(t, session)
		assert.ErrorContains(t, err, "failed to create long-poll session")
	})
}

// ---------- WaitForCallback actual timeout (not context cancel) ----------

func TestWaitForCallback_ActualTimeoutExpiresNaturally(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	// Use a short timeout and short poll interval so timeout fires quickly
	svc := NewServiceWithConfig(repo, nil, ServiceConfig{
		DefaultTimeoutMs: 30000,
		MaxTimeoutMs:     300000,
		PollIntervalMs:   50,
	})
	correlationID := "corr-natural-timeout"
	requestID := uuid.New()

	correlation := &CorrelationEntry{
		CorrelationID: correlationID,
		RequestID:     requestID,
		Status:        CallbackStatusPending,
	}

	repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
	repo.On("UpdateCorrelation", mock.Anything, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)
	repo.On("UpdateCallbackStatus", mock.Anything, requestID, mock.Anything).Return(nil)
	// Response never arrives
	repo.On("GetResponseByCorrelation", mock.Anything, correlationID).
		Return((*CallbackResponse)(nil), fmt.Errorf("not found"))

	// Use a very short timeout (200ms) so the internal deadline fires
	resp, err := svc.WaitForCallback(context.Background(), correlationID, 200)

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.ErrorContains(t, err, "timeout waiting for callback response")
	assert.Contains(t, err.Error(), correlationID)
}

// ---------- WaitForCallback timeout > max clamped ----------

func TestWaitForCallback_TimeoutExceedingMaxClamped(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := newTestService(repo)
	correlationID := "corr-max-clamp"

	// Return already-received so we can verify the function completes
	correlation := &CorrelationEntry{
		CorrelationID: correlationID,
		Status:        CallbackStatusReceived,
	}
	expectedResp := &CallbackResponse{ID: uuid.New(), CorrelationID: correlationID}

	repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
	repo.On("GetResponseByCorrelation", mock.Anything, correlationID).Return(expectedResp, nil)

	// Pass timeout > max (300000), should be clamped but still work
	resp, err := svc.WaitForCallback(context.Background(), correlationID, 999999)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ---------- Concurrent ReceiveCallback + WaitForCallback ----------

func TestConcurrentReceiveAndWait(t *testing.T) {
	t.Parallel()

	repo := new(MockRepository)
	svc := newTestService(repo)
	correlationID := "corr-concurrent"
	requestID := uuid.New()

	correlation := &CorrelationEntry{
		CorrelationID: correlationID,
		RequestID:     requestID,
		Status:        CallbackStatusPending,
		CreatedAt:     time.Now(),
	}

	expectedResp := &CallbackResponse{
		ID:            uuid.New(),
		CorrelationID: correlationID,
		StatusCode:    200,
		Body:          json.RawMessage(`{"ok":true}`),
	}

	// Shared mocks for both paths
	repo.On("GetCorrelation", mock.Anything, correlationID).Return(correlation, nil)
	repo.On("UpdateCorrelation", mock.Anything, mock.AnythingOfType("*callback.CorrelationEntry")).Return(nil)
	repo.On("UpdateCallbackStatus", mock.Anything, requestID, mock.Anything).Return(nil)

	// ReceiveCallback path
	repo.On("SaveCallbackResponse", mock.Anything, mock.AnythingOfType("*callback.CallbackResponse")).Return(nil)

	// First two polls return not-found, then subsequent polls return the response.
	repo.On("GetResponseByCorrelation", mock.Anything, correlationID).
		Return((*CallbackResponse)(nil), fmt.Errorf("not found")).Times(2)
	repo.On("GetResponseByCorrelation", mock.Anything, correlationID).
		Return(expectedResp, nil)

	var wg sync.WaitGroup
	var waitResp *CallbackResponse
	var waitErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		waitResp, waitErr = svc.WaitForCallback(context.Background(), correlationID, 5000)
	}()

	// Simulate receive arriving concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(300 * time.Millisecond)
		_, _ = svc.ReceiveCallback(context.Background(), correlationID, 200,
			json.RawMessage(`{"ok":true}`), nil)
	}()

	wg.Wait()

	require.NoError(t, waitErr)
	assert.NotNil(t, waitResp)
	assert.Equal(t, 200, waitResp.StatusCode)
}

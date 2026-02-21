package playground

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- SessionManager Tests ---

func TestSessionManager_CreateSession(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	tenantID := uuid.New()

	session := sm.CreateSession(tenantID, "Test Playground")

	assert.NotEqual(t, uuid.Nil, session.ID)
	assert.Equal(t, tenantID, session.TenantID)
	assert.Equal(t, "Test Playground", session.Name)
	assert.Equal(t, "active", session.Status)
	assert.Contains(t, session.InboundURL, session.ID.String())
	assert.NotEmpty(t, session.ShareToken)
	assert.NotEmpty(t, session.ShareURL)
	assert.True(t, session.ExpiresAt.After(time.Now()))
	assert.Equal(t, 0, session.WebhooksSent)
	assert.Equal(t, 0, session.WebhooksReceived)
}

func TestSessionManager_GetSession(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	created := sm.CreateSession(uuid.New(), "get-test")

	got, err := sm.GetSession(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "get-test", got.Name)
}

func TestSessionManager_GetSession_NotFound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")

	_, err := sm.GetSession(uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_GetSessionByToken(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	created := sm.CreateSession(uuid.New(), "token-test")

	got, err := sm.GetSessionByToken(created.ShareToken)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestSessionManager_GetSessionByToken_NotFound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")

	_, err := sm.GetSessionByToken("nonexistent-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_DeleteSession(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	created := sm.CreateSession(uuid.New(), "delete-me")

	err := sm.DeleteSession(created.ID)
	require.NoError(t, err)

	_, err = sm.GetSession(created.ID)
	require.Error(t, err)
}

func TestSessionManager_DeleteSession_NotFound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")

	err := sm.DeleteSession(uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_AddMessage_Inbound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	session := sm.CreateSession(uuid.New(), "msg-test")

	msg := &SessionMessage{
		Type:      "webhook",
		Direction: "inbound",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Payload:   json.RawMessage(`{"event":"test"}`),
	}

	err := sm.AddMessage(session.ID, msg)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, msg.ID)
	assert.Equal(t, session.ID, msg.SessionID)
	assert.False(t, msg.Timestamp.IsZero())

	got, _ := sm.GetSession(session.ID)
	assert.Equal(t, 1, got.WebhooksReceived)
	assert.Equal(t, 0, got.WebhooksSent)
}

func TestSessionManager_AddMessage_Outbound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	session := sm.CreateSession(uuid.New(), "outbound-test")

	msg := &SessionMessage{
		Type:       "response",
		Direction:  "outbound",
		StatusCode: 200,
		Payload:    json.RawMessage(`{"status":"ok"}`),
	}

	err := sm.AddMessage(session.ID, msg)
	require.NoError(t, err)

	got, _ := sm.GetSession(session.ID)
	assert.Equal(t, 0, got.WebhooksReceived)
	assert.Equal(t, 1, got.WebhooksSent)
}

func TestSessionManager_AddMessage_SessionNotFound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")

	err := sm.AddMessage(uuid.New(), &SessionMessage{Direction: "inbound"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionManager_ListMessages(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	session := sm.CreateSession(uuid.New(), "list-msgs")

	for i := 0; i < 5; i++ {
		err := sm.AddMessage(session.ID, &SessionMessage{
			Type:      "webhook",
			Direction: "inbound",
			Payload:   json.RawMessage(`{"seq":` + string(rune('0'+i)) + `}`),
		})
		require.NoError(t, err)
	}

	msgs, err := sm.ListMessages(session.ID, 3)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestSessionManager_ListMessages_AllMessages(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	session := sm.CreateSession(uuid.New(), "list-all")

	for i := 0; i < 3; i++ {
		_ = sm.AddMessage(session.ID, &SessionMessage{Direction: "inbound"})
	}

	msgs, err := sm.ListMessages(session.ID, 0)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestSessionManager_ListMessages_SessionNotFound(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")

	_, err := sm.ListMessages(uuid.New(), 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- FailureSimulator Tests ---

func TestFailureSimulator_SetAndGet(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()

	sim := &FailureSimulation{
		Type:    "server_error",
		Enabled: true,
		StatusCode: 503,
	}
	fs.SetSimulation(sessionID, sim)

	got := fs.GetSimulation(sessionID)
	require.NotNil(t, got)
	assert.Equal(t, "server_error", got.Type)
	assert.Equal(t, 503, got.StatusCode)
}

func TestFailureSimulator_GetNil(t *testing.T) {
	fs := NewFailureSimulator()
	assert.Nil(t, fs.GetSimulation(uuid.New()))
}

func TestFailureSimulator_ClearSimulation(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{Type: "timeout", Enabled: true})

	fs.ClearSimulation(sessionID)
	assert.Nil(t, fs.GetSimulation(sessionID))
}

func TestFailureSimulator_SimulateResponse_NoSimulation(t *testing.T) {
	fs := NewFailureSimulator()
	code, body, delay, simulated := fs.SimulateResponse(uuid.New())
	assert.Equal(t, 200, code)
	assert.Empty(t, body)
	assert.Equal(t, int64(0), delay)
	assert.False(t, simulated)
}

func TestFailureSimulator_SimulateResponse_Disabled(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{Type: "server_error", Enabled: false})

	_, _, _, simulated := fs.SimulateResponse(sessionID)
	assert.False(t, simulated)
}

func TestFailureSimulator_SimulateResponse_Timeout(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:    "timeout",
		Enabled: true,
		DelayMs: 30000,
	})

	code, _, delay, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 0, code)
	assert.Equal(t, int64(30000), delay)
}

func TestFailureSimulator_SimulateResponse_ServerError(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:         "server_error",
		Enabled:      true,
		StatusCode:   502,
		ErrorMessage: "Bad Gateway",
	})

	code, body, _, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 502, code)
	assert.Equal(t, "Bad Gateway", body)
}

func TestFailureSimulator_SimulateResponse_ServerErrorDefaults(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:    "server_error",
		Enabled: true,
	})

	code, body, _, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 500, code)
	assert.Equal(t, "Internal Server Error", body)
}

func TestFailureSimulator_SimulateResponse_RateLimit(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:          "rate_limit",
		Enabled:       true,
		RetryAfterSec: 60,
	})

	code, body, _, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 429, code)
	assert.Contains(t, body, "rate limited")
	assert.Contains(t, body, "60")
}

func TestFailureSimulator_SimulateResponse_NetworkError(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:    "network_error",
		Enabled: true,
	})

	code, body, _, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 0, code)
	assert.Contains(t, body, "connection reset")
}

func TestFailureSimulator_SimulateResponse_SlowResponse(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:    "slow_response",
		Enabled: true,
		DelayMs: 5000,
	})

	code, _, delay, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 200, code)
	assert.Equal(t, int64(5000), delay)
}

// --- TransformPreviewer Tests ---

func TestTransformPreviewer_Preview_Success(t *testing.T) {
	tp := NewTransformPreviewer()
	sessionID := uuid.New()
	input := json.RawMessage(`{"name":"world"}`)

	transformFn := func(code string, in json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"greeting":"hello world"}`), nil
	}

	preview := tp.Preview(sessionID, "test-code", input, transformFn)
	assert.True(t, preview.Success)
	assert.Empty(t, preview.ErrorMessage)
	assert.JSONEq(t, `{"greeting":"hello world"}`, string(preview.OutputPayload))
	assert.JSONEq(t, `{"name":"world"}`, string(preview.InputPayload))
}

func TestTransformPreviewer_Preview_Error(t *testing.T) {
	tp := NewTransformPreviewer()
	sessionID := uuid.New()
	input := json.RawMessage(`{"bad":"input"}`)

	transformFn := func(code string, in json.RawMessage) (json.RawMessage, error) {
		return nil, assert.AnError
	}

	preview := tp.Preview(sessionID, "bad-code", input, transformFn)
	assert.False(t, preview.Success)
	assert.NotEmpty(t, preview.ErrorMessage)
	assert.Nil(t, preview.OutputPayload)
}

func TestTransformPreviewer_GetLastPreview(t *testing.T) {
	tp := NewTransformPreviewer()
	sessionID := uuid.New()

	assert.Nil(t, tp.GetLastPreview(sessionID))

	transformFn := func(code string, in json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"ok":true}`), nil
	}
	tp.Preview(sessionID, "code", json.RawMessage(`{}`), transformFn)

	last := tp.GetLastPreview(sessionID)
	require.NotNil(t, last)
	assert.True(t, last.Success)
}

// --- ScenarioTemplate Tests ---

func TestPredefinedScenarioTemplates(t *testing.T) {
	templates := PredefinedScenarioTemplates()
	assert.GreaterOrEqual(t, len(templates), 5, "should have at least 5 predefined templates")

	ids := make(map[string]bool)
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.ID)
		assert.NotEmpty(t, tmpl.Name)
		assert.NotEmpty(t, tmpl.Description)
		assert.NotEmpty(t, tmpl.Provider)
		assert.NotEmpty(t, tmpl.EventType)
		assert.NotNil(t, tmpl.Headers)
		assert.True(t, json.Valid(tmpl.Payload), "payload for %s should be valid JSON", tmpl.ID)
		ids[tmpl.ID] = true
	}
	assert.Len(t, ids, len(templates), "all template IDs should be unique")
}

func TestPredefinedScenarioTemplates_KnownProviders(t *testing.T) {
	templates := PredefinedScenarioTemplates()
	providers := make(map[string]bool)
	for _, tmpl := range templates {
		providers[tmpl.Provider] = true
	}
	assert.True(t, providers["Stripe"])
	assert.True(t, providers["GitHub"])
	assert.True(t, providers["Shopify"])
}

func TestGetScenarioTemplate_Found(t *testing.T) {
	tmpl, err := GetScenarioTemplate("stripe-payment-success")
	require.NoError(t, err)
	assert.Equal(t, "Stripe Payment Success", tmpl.Name)
	assert.Equal(t, "Stripe", tmpl.Provider)
}

func TestGetScenarioTemplate_NotFound(t *testing.T) {
	_, err := GetScenarioTemplate("nonexistent-template")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- PlaygroundSession struct Tests ---

func TestPlaygroundSession_Fields(t *testing.T) {
	now := time.Now()
	session := &PlaygroundSession{
		ID:               uuid.New(),
		TenantID:         uuid.New(),
		Name:             "Test",
		InboundURL:       "https://example.com/inbound",
		Status:           "active",
		WebhooksSent:     5,
		WebhooksReceived: 10,
		ShareURL:         "https://example.com/share/abc",
		ShareToken:       "abc",
		ExpiresAt:        now.Add(time.Hour),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, "Test", session.Name)
	assert.Equal(t, "active", session.Status)
	assert.Equal(t, 5, session.WebhooksSent)
	assert.Equal(t, 10, session.WebhooksReceived)
}

// --- SessionMessage struct Tests ---

func TestSessionMessage_JSON(t *testing.T) {
	msg := &SessionMessage{
		ID:         uuid.New(),
		SessionID:  uuid.New(),
		Type:       "webhook",
		Timestamp:  time.Now(),
		Direction:  "inbound",
		Headers:    map[string]string{"X-Test": "value"},
		Payload:    json.RawMessage(`{"event":"ping"}`),
		StatusCode: 200,
		LatencyMs:  42,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded SessionMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.Direction, decoded.Direction)
	assert.Equal(t, msg.StatusCode, decoded.StatusCode)
	assert.Equal(t, msg.LatencyMs, decoded.LatencyMs)
}

// --- Integration-style Tests ---

func TestSessionManager_FullWorkflow(t *testing.T) {
	sm := NewSessionManager("https://api.example.com")
	tenantID := uuid.New()

	// Create session
	session := sm.CreateSession(tenantID, "workflow-test")
	require.NotNil(t, session)

	// Add inbound messages
	for i := 0; i < 3; i++ {
		err := sm.AddMessage(session.ID, &SessionMessage{
			Type:      "webhook",
			Direction: "inbound",
			Payload:   json.RawMessage(`{"seq":` + string(rune('0'+i)) + `}`),
		})
		require.NoError(t, err)
	}

	// Add outbound message
	err := sm.AddMessage(session.ID, &SessionMessage{
		Type:       "response",
		Direction:  "outbound",
		StatusCode: 200,
	})
	require.NoError(t, err)

	// Verify counters
	got, err := sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, got.WebhooksReceived)
	assert.Equal(t, 1, got.WebhooksSent)

	// List messages
	msgs, err := sm.ListMessages(session.ID, 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 4)

	// Access by share token
	shared, err := sm.GetSessionByToken(session.ShareToken)
	require.NoError(t, err)
	assert.Equal(t, session.ID, shared.ID)

	// Delete session
	err = sm.DeleteSession(session.ID)
	require.NoError(t, err)

	_, err = sm.GetSession(session.ID)
	require.Error(t, err)
}

func TestFailureSimulator_FullWorkflow(t *testing.T) {
	fs := NewFailureSimulator()
	sessionID := uuid.New()

	// No simulation set
	_, _, _, simulated := fs.SimulateResponse(sessionID)
	assert.False(t, simulated)

	// Set a simulation
	fs.SetSimulation(sessionID, &FailureSimulation{
		Type:       "server_error",
		Enabled:    true,
		StatusCode: 503,
	})

	code, _, _, simulated := fs.SimulateResponse(sessionID)
	assert.True(t, simulated)
	assert.Equal(t, 503, code)

	// Clear simulation
	fs.ClearSimulation(sessionID)
	_, _, _, simulated = fs.SimulateResponse(sessionID)
	assert.False(t, simulated)
}

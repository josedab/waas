package playground

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PlaygroundSession represents an interactive real-time webhook playground environment.
type PlaygroundSession struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Name             string     `json:"name"`
	InboundURL       string     `json:"inbound_url"`
	Status           string     `json:"status"` // "active", "paused", "expired"
	WebhooksSent     int        `json:"webhooks_sent"`
	WebhooksReceived int        `json:"webhooks_received"`
	ShareURL         string     `json:"share_url,omitempty"`
	ShareToken       string     `json:"share_token,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// SessionMessage represents a single message exchanged in a playground session,
// suitable for WebSocket communication.
type SessionMessage struct {
	ID         uuid.UUID         `json:"id"`
	SessionID  uuid.UUID         `json:"session_id"`
	Type       string            `json:"type"` // "webhook", "response", "error", "info"
	Timestamp  time.Time         `json:"timestamp"`
	Direction  string            `json:"direction"` // "inbound", "outbound"
	Headers    map[string]string `json:"headers,omitempty"`
	Payload    json.RawMessage   `json:"payload,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	LatencyMs  int64             `json:"latency_ms,omitempty"`
}

// FailureSimulation configures how the playground should simulate failures
// for incoming webhooks.
type FailureSimulation struct {
	Type          string  `json:"type"`           // "timeout", "server_error", "rate_limit", "network_error", "slow_response"
	Enabled       bool    `json:"enabled"`
	StatusCode    int     `json:"status_code,omitempty"`
	DelayMs       int64   `json:"delay_ms,omitempty"`
	ErrorMessage  string  `json:"error_message,omitempty"`
	FailureRate   float64 `json:"failure_rate,omitempty"` // 0.0 - 1.0
	RetryAfterSec int     `json:"retry_after_sec,omitempty"`
}

// FailureSimulator manages failure simulations for a playground session.
type FailureSimulator struct {
	mu          sync.RWMutex
	simulations map[uuid.UUID]*FailureSimulation // keyed by session ID
}

// NewFailureSimulator creates a new FailureSimulator.
func NewFailureSimulator() *FailureSimulator {
	return &FailureSimulator{
		simulations: make(map[uuid.UUID]*FailureSimulation),
	}
}

// SetSimulation configures a failure simulation for a session.
func (fs *FailureSimulator) SetSimulation(sessionID uuid.UUID, sim *FailureSimulation) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.simulations[sessionID] = sim
}

// GetSimulation retrieves the active failure simulation for a session.
func (fs *FailureSimulator) GetSimulation(sessionID uuid.UUID) *FailureSimulation {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.simulations[sessionID]
}

// ClearSimulation removes the failure simulation for a session.
func (fs *FailureSimulator) ClearSimulation(sessionID uuid.UUID) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.simulations, sessionID)
}

// SimulateResponse returns a simulated response based on the configured failure.
// Returns statusCode, body, delay, and whether a simulation was applied.
func (fs *FailureSimulator) SimulateResponse(sessionID uuid.UUID) (statusCode int, body string, delayMs int64, simulated bool) {
	fs.mu.RLock()
	sim := fs.simulations[sessionID]
	fs.mu.RUnlock()

	if sim == nil || !sim.Enabled {
		return 200, "", 0, false
	}

	switch sim.Type {
	case "timeout":
		return 0, "", sim.DelayMs, true
	case "server_error":
		code := sim.StatusCode
		if code == 0 {
			code = 500
		}
		msg := sim.ErrorMessage
		if msg == "" {
			msg = "Internal Server Error"
		}
		return code, msg, 0, true
	case "rate_limit":
		return 429, fmt.Sprintf(`{"error":"rate limited","retry_after":%d}`, sim.RetryAfterSec), 0, true
	case "network_error":
		return 0, "connection reset by peer", 0, true
	case "slow_response":
		return 200, "", sim.DelayMs, true
	default:
		return 200, "", 0, false
	}
}

// TransformPreview holds the result of a live transformation preview.
type TransformPreview struct {
	InputPayload  json.RawMessage `json:"input_payload"`
	OutputPayload json.RawMessage `json:"output_payload,omitempty"`
	Success       bool            `json:"success"`
	ErrorMessage  string          `json:"error_message,omitempty"`
	ExecutionMs   int64           `json:"execution_ms"`
	Diff          []DiffEntry     `json:"diff,omitempty"`
}

// TransformPreviewer provides live transformation previewing capabilities.
type TransformPreviewer struct {
	mu       sync.RWMutex
	previews map[uuid.UUID]*TransformPreview // keyed by session ID
}

// NewTransformPreviewer creates a new TransformPreviewer.
func NewTransformPreviewer() *TransformPreviewer {
	return &TransformPreviewer{
		previews: make(map[uuid.UUID]*TransformPreview),
	}
}

// Preview executes a transformation preview and stores the result.
func (tp *TransformPreviewer) Preview(sessionID uuid.UUID, code string, input json.RawMessage, transformFn func(code string, input json.RawMessage) (json.RawMessage, error)) *TransformPreview {
	start := time.Now()
	preview := &TransformPreview{
		InputPayload: input,
	}

	output, err := transformFn(code, input)
	preview.ExecutionMs = time.Since(start).Milliseconds()

	if err != nil {
		preview.Success = false
		preview.ErrorMessage = err.Error()
	} else {
		preview.Success = true
		preview.OutputPayload = output
	}

	tp.mu.Lock()
	tp.previews[sessionID] = preview
	tp.mu.Unlock()

	return preview
}

// GetLastPreview retrieves the last preview result for a session.
func (tp *TransformPreviewer) GetLastPreview(sessionID uuid.UUID) *TransformPreview {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.previews[sessionID]
}

// ScenarioTemplate represents a predefined webhook scenario template.
type ScenarioTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Provider    string            `json:"provider"`
	EventType   string            `json:"event_type"`
	Headers     map[string]string `json:"headers"`
	Payload     json.RawMessage   `json:"payload"`
}

// PredefinedScenarioTemplates returns the built-in scenario templates.
func PredefinedScenarioTemplates() []ScenarioTemplate {
	return []ScenarioTemplate{
		{
			ID:          "stripe-payment-success",
			Name:        "Stripe Payment Success",
			Description: "Simulates a successful Stripe payment_intent.succeeded event",
			Provider:    "Stripe",
			EventType:   "payment_intent.succeeded",
			Headers: map[string]string{
				"Content-Type":    "application/json",
				"Stripe-Signature": "t=1234567890,v1=signature_placeholder",
			},
			Payload: json.RawMessage(`{"id":"evt_1ABC","object":"event","type":"payment_intent.succeeded","data":{"object":{"id":"pi_1DEF","amount":2000,"currency":"usd","status":"succeeded","payment_method":"pm_card_visa"}}}`),
		},
		{
			ID:          "github-push",
			Name:        "GitHub Push Event",
			Description: "Simulates a GitHub push event to a repository",
			Provider:    "GitHub",
			EventType:   "push",
			Headers: map[string]string{
				"Content-Type":      "application/json",
				"X-GitHub-Event":    "push",
				"X-GitHub-Delivery": "72d3162e-cc78-11e3-81ab-4c9367dc0958",
			},
			Payload: json.RawMessage(`{"ref":"refs/heads/main","before":"abc123","after":"def456","repository":{"id":12345,"full_name":"octocat/Hello-World"},"pusher":{"name":"octocat","email":"octocat@github.com"},"commits":[{"id":"def456","message":"Fix bug","timestamp":"2024-01-15T10:00:00Z"}]}`),
		},
		{
			ID:          "shopify-order-created",
			Name:        "Shopify Order Created",
			Description: "Simulates a Shopify order creation webhook",
			Provider:    "Shopify",
			EventType:   "orders/create",
			Headers: map[string]string{
				"Content-Type":              "application/json",
				"X-Shopify-Topic":           "orders/create",
				"X-Shopify-Shop-Domain":     "example.myshopify.com",
				"X-Shopify-Hmac-SHA256":     "hmac_placeholder",
			},
			Payload: json.RawMessage(`{"id":820982911946154508,"email":"jon@example.com","total_price":"199.00","currency":"USD","financial_status":"paid","line_items":[{"title":"Widget","quantity":1,"price":"199.00"}],"shipping_address":{"city":"Ottawa","country":"Canada"}}`),
		},
		{
			ID:          "twilio-sms-received",
			Name:        "Twilio SMS Received",
			Description: "Simulates an inbound SMS received via Twilio",
			Provider:    "Twilio",
			EventType:   "sms.received",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Payload: json.RawMessage(`{"MessageSid":"SM1234567890","AccountSid":"AC1234567890","From":"+15551234567","To":"+15559876543","Body":"Hello from Twilio!","NumMedia":"0"}`),
		},
		{
			ID:          "slack-event-callback",
			Name:        "Slack Event Callback",
			Description: "Simulates a Slack event callback for a message posted in a channel",
			Provider:    "Slack",
			EventType:   "event_callback",
			Headers: map[string]string{
				"Content-Type":       "application/json",
				"X-Slack-Request-Timestamp": "1234567890",
				"X-Slack-Signature":  "v0=signature_placeholder",
			},
			Payload: json.RawMessage(`{"token":"verification_token","team_id":"T0001","event":{"type":"message","channel":"C2147483705","user":"U2147483697","text":"Hello world","ts":"1355517523.000005"},"type":"event_callback","event_id":"Ev0001","event_time":1355517523}`),
		},
		{
			ID:          "sendgrid-email-event",
			Name:        "SendGrid Email Event",
			Description: "Simulates a SendGrid email delivery event webhook",
			Provider:    "SendGrid",
			EventType:   "delivered",
			Headers: map[string]string{
				"Content-Type":    "application/json",
				"User-Agent":     "SendGrid Event API",
			},
			Payload: json.RawMessage(`[{"email":"test@example.com","timestamp":1234567890,"event":"delivered","sg_event_id":"evt_123","sg_message_id":"msg_123.filter0001.12345.abc-1","response":"250 OK","attempt":"1"}]`),
		},
	}
}

// GetScenarioTemplate retrieves a predefined scenario template by ID.
func GetScenarioTemplate(id string) (*ScenarioTemplate, error) {
	for _, t := range PredefinedScenarioTemplates() {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("scenario template %q not found", id)
}

// generateShareToken creates a cryptographically random URL-safe token.
func generateShareToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SessionManager manages playground sessions with thread-safe in-memory storage.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*PlaygroundSession
	messages map[uuid.UUID][]*SessionMessage // keyed by session ID
	baseURL  string
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(baseURL string) *SessionManager {
	return &SessionManager{
		sessions: make(map[uuid.UUID]*PlaygroundSession),
		messages: make(map[uuid.UUID][]*SessionMessage),
		baseURL:  baseURL,
	}
}

// CreateSession creates a new playground session.
func (sm *SessionManager) CreateSession(tenantID uuid.UUID, name string) *PlaygroundSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := uuid.New()
	token := generateShareToken()
	now := time.Now()

	session := &PlaygroundSession{
		ID:         id,
		TenantID:   tenantID,
		Name:       name,
		InboundURL: fmt.Sprintf("%s/playground/sessions/%s/inbound", sm.baseURL, id.String()),
		Status:     "active",
		ShareToken: token,
		ShareURL:   fmt.Sprintf("%s/playground/shared/%s", sm.baseURL, token),
		ExpiresAt:  now.Add(24 * time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	sm.sessions[id] = session
	sm.messages[id] = []*SessionMessage{}
	return session
}

// GetSession retrieves a session by ID.
func (sm *SessionManager) GetSession(id uuid.UUID) (*PlaygroundSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("playground session %q not found", id)
	}
	return session, nil
}

// GetSessionByToken retrieves a session by its share token.
func (sm *SessionManager) GetSessionByToken(token string) (*PlaygroundSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, session := range sm.sessions {
		if session.ShareToken == token {
			return session, nil
		}
	}
	return nil, fmt.Errorf("playground session with token %q not found", token)
}

// DeleteSession removes a session and its messages.
func (sm *SessionManager) DeleteSession(id uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessions[id]; !ok {
		return fmt.Errorf("playground session %q not found", id)
	}
	delete(sm.sessions, id)
	delete(sm.messages, id)
	return nil
}

// AddMessage appends a message to a session's message log and updates counters.
func (sm *SessionManager) AddMessage(sessionID uuid.UUID, msg *SessionMessage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("playground session %q not found", sessionID)
	}

	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	msg.SessionID = sessionID
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	sm.messages[sessionID] = append(sm.messages[sessionID], msg)

	if msg.Direction == "inbound" {
		session.WebhooksReceived++
	} else if msg.Direction == "outbound" {
		session.WebhooksSent++
	}
	session.UpdatedAt = time.Now()

	return nil
}

// ListMessages returns messages for a session, up to limit.
func (sm *SessionManager) ListMessages(sessionID uuid.UUID, limit int) ([]*SessionMessage, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	msgs, ok := sm.messages[sessionID]
	if !ok {
		return nil, fmt.Errorf("playground session %q not found", sessionID)
	}

	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}

	// Return the most recent messages
	start := len(msgs) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*SessionMessage, limit)
	copy(result, msgs[start:start+limit])
	return result, nil
}

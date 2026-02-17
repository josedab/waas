// Package graphqlsub provides GraphQL subscriptions gateway for webhook events
package graphqlsub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrClientNotFound       = errors.New("client not found")
	ErrInvalidQuery         = errors.New("invalid GraphQL query")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrConnectionClosed     = errors.New("connection closed")
)

// SubscriptionType represents types of subscriptions
type SubscriptionType string

const (
	SubscriptionWebhookEvents  SubscriptionType = "webhook_events"
	SubscriptionDeliveryStatus SubscriptionType = "delivery_status"
	SubscriptionEndpointHealth SubscriptionType = "endpoint_health"
	SubscriptionWorkflowStatus SubscriptionType = "workflow_status"
)

// ClientState represents WebSocket client states
type ClientState string

const (
	ClientStateConnecting   ClientState = "connecting"
	ClientStateConnected    ClientState = "connected"
	ClientStateSubscribed   ClientState = "subscribed"
	ClientStateDisconnected ClientState = "disconnected"
)

// Subscription represents a GraphQL subscription
type Subscription struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ClientID    string                 `json:"client_id"`
	Type        SubscriptionType       `json:"type"`
	Query       string                 `json:"query"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Filters     *SubscriptionFilters   `json:"filters,omitempty"`
	Active      bool                   `json:"active"`
	CreatedAt   time.Time              `json:"created_at"`
	LastEventAt *time.Time             `json:"last_event_at,omitempty"`
}

// SubscriptionFilters for filtering subscription events
type SubscriptionFilters struct {
	EventTypes  []string `json:"event_types,omitempty"`
	EndpointIDs []string `json:"endpoint_ids,omitempty"`
	WorkflowIDs []string `json:"workflow_ids,omitempty"`
	Severity    []string `json:"severity,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID            string                   `json:"id"`
	TenantID      string                   `json:"tenant_id"`
	State         ClientState              `json:"state"`
	Protocol      string                   `json:"protocol"` // graphql-ws, subscriptions-transport-ws
	ConnectedAt   time.Time                `json:"connected_at"`
	LastPingAt    *time.Time               `json:"last_ping_at,omitempty"`
	Subscriptions map[string]*Subscription `json:"-"`
	SendCh        chan []byte              `json:"-"`
	CloseCh       chan struct{}            `json:"-"`
	mu            sync.RWMutex
}

// Event represents an event to be published to subscriptions
type Event struct {
	ID        string            `json:"id"`
	Type      SubscriptionType  `json:"type"`
	TenantID  string            `json:"tenant_id"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// GraphQLMessage represents a GraphQL WebSocket message
type GraphQLMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// GraphQL message types (graphql-ws protocol)
const (
	MessageTypeConnectionInit      = "connection_init"
	MessageTypeConnectionAck       = "connection_ack"
	MessageTypePing                = "ping"
	MessageTypePong                = "pong"
	MessageTypeSubscribe           = "subscribe"
	MessageTypeNext                = "next"
	MessageTypeError               = "error"
	MessageTypeComplete            = "complete"
	MessageTypeConnectionTerminate = "connection_terminate"
)

// SubscriptionPayload represents a subscribe message payload
type SubscriptionPayload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
}

// EventPayload represents an event data payload
type EventPayload struct {
	Data   interface{} `json:"data"`
	Errors []GQLError  `json:"errors,omitempty"`
}

// GQLError represents a GraphQL error
type GQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// SchemaField represents a field in the GraphQL schema
type SchemaField struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Args        []SchemaArg `json:"args,omitempty"`
	Nullable    bool        `json:"nullable"`
	IsList      bool        `json:"is_list"`
}

// SchemaArg represents an argument in a GraphQL field
type SchemaArg struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	Required     bool   `json:"required"`
}

// SchemaType represents a type in the GraphQL schema
type SchemaType struct {
	Name        string        `json:"name"`
	Kind        string        `json:"kind"` // OBJECT, SUBSCRIPTION, INPUT_OBJECT, ENUM
	Description string        `json:"description,omitempty"`
	Fields      []SchemaField `json:"fields,omitempty"`
	EnumValues  []string      `json:"enum_values,omitempty"`
}

// Schema represents the GraphQL subscription schema
type Schema struct {
	Types         []SchemaType  `json:"types"`
	Subscriptions []SchemaField `json:"subscriptions"`
}

// ConnectionConfig represents WebSocket connection configuration
type ConnectionConfig struct {
	MaxConnections    int           `json:"max_connections"`
	PingInterval      time.Duration `json:"ping_interval"`
	PongTimeout       time.Duration `json:"pong_timeout"`
	MaxMessageSize    int64         `json:"max_message_size"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	MaxSubscriptions  int           `json:"max_subscriptions"`
	EnableCompression bool          `json:"enable_compression"`
}

// DefaultConnectionConfig returns default configuration
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		MaxConnections:    10000,
		PingInterval:      30 * time.Second,
		PongTimeout:       10 * time.Second,
		MaxMessageSize:    1024 * 1024, // 1MB
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       60 * time.Second,
		MaxSubscriptions:  100,
		EnableCompression: true,
	}
}

// WebhookEventData represents webhook event subscription data
type WebhookEventData struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	EndpointID   string          `json:"endpoint_id"`
	EndpointURL  string          `json:"endpoint_url"`
	Payload      json.RawMessage `json:"payload"`
	Timestamp    time.Time       `json:"timestamp"`
	Attempt      int             `json:"attempt"`
	Status       string          `json:"status"`
	ResponseCode int             `json:"response_code,omitempty"`
	ResponseTime int64           `json:"response_time_ms,omitempty"`
	NextRetryAt  *time.Time      `json:"next_retry_at,omitempty"`
}

// DeliveryStatusData represents delivery status subscription data
type DeliveryStatusData struct {
	DeliveryID   string    `json:"delivery_id"`
	WebhookID    string    `json:"webhook_id"`
	EndpointID   string    `json:"endpoint_id"`
	Status       string    `json:"status"` // pending, delivered, failed, retrying
	Attempt      int       `json:"attempt"`
	ResponseCode int       `json:"response_code,omitempty"`
	ResponseTime int64     `json:"response_time_ms,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// EndpointHealthData represents endpoint health subscription data
type EndpointHealthData struct {
	EndpointID     string    `json:"endpoint_id"`
	EndpointURL    string    `json:"endpoint_url"`
	Status         string    `json:"status"` // healthy, degraded, unhealthy
	HealthScore    float64   `json:"health_score"`
	SuccessRate    float64   `json:"success_rate"`
	AverageLatency float64   `json:"average_latency_ms"`
	LastChecked    time.Time `json:"last_checked"`
}

// WorkflowStatusData represents workflow status subscription data
type WorkflowStatusData struct {
	WorkflowID   string     `json:"workflow_id"`
	ExecutionID  string     `json:"execution_id"`
	Status       string     `json:"status"` // running, completed, failed, cancelled
	CurrentNode  string     `json:"current_node,omitempty"`
	Progress     int        `json:"progress"` // 0-100
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// ClientStats represents client statistics
type ClientStats struct {
	TotalClients       int   `json:"total_clients"`
	ActiveClients      int   `json:"active_clients"`
	TotalSubscriptions int   `json:"total_subscriptions"`
	EventsPublished    int64 `json:"events_published"`
	EventsDelivered    int64 `json:"events_delivered"`
}

// Repository defines the interface for subscription data storage
type Repository interface {
	// Subscriptions
	SaveSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, subID string) (*Subscription, error)
	DeleteSubscription(ctx context.Context, subID string) error
	ListSubscriptions(ctx context.Context, tenantID string) ([]Subscription, error)
	GetSubscriptionsByType(ctx context.Context, tenantID string, subType SubscriptionType) ([]Subscription, error)

	// Client sessions (for distributed environments)
	SaveClientSession(ctx context.Context, client *Client) error
	GetClientSession(ctx context.Context, clientID string) (*Client, error)
	DeleteClientSession(ctx context.Context, clientID string) error
	ListClientSessions(ctx context.Context, tenantID string) ([]Client, error)

	// Events (for replay/history)
	SaveEvent(ctx context.Context, event *Event) error
	GetEvents(ctx context.Context, tenantID string, subType SubscriptionType, since time.Time, limit int) ([]Event, error)

	// Stats
	GetStats(ctx context.Context, tenantID string) (*ClientStats, error)
	IncrementEventCounter(ctx context.Context, tenantID string, count int64) error
}

// Authenticator defines the interface for authenticating WebSocket connections
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (tenantID string, err error)
	ValidateSubscription(ctx context.Context, tenantID string, sub *Subscription) error
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(ctx context.Context, tenantID string, types []SubscriptionType) (<-chan *Event, error)
	Unsubscribe(ctx context.Context, tenantID string) error
}

// SchemaGenerator defines the interface for generating GraphQL schemas
type SchemaGenerator interface {
	GenerateSchema(ctx context.Context, tenantID string) (*Schema, error)
	ValidateQuery(ctx context.Context, query string, variables map[string]interface{}) error
	ParseSubscription(ctx context.Context, payload *SubscriptionPayload) (*Subscription, error)
}

// Service provides GraphQL subscription operations
type Service struct {
	repo      Repository
	auth      Authenticator
	publisher EventPublisher
	schemaGen SchemaGenerator
	clients   map[string]*Client
	mu        sync.RWMutex
	config    *ConnectionConfig
	schema    *Schema
}

// NewService creates a new GraphQL subscription service
func NewService(repo Repository, config *ConnectionConfig) *Service {
	if config == nil {
		config = DefaultConnectionConfig()
	}

	return &Service{
		repo:    repo,
		clients: make(map[string]*Client),
		config:  config,
		schema:  GenerateDefaultSchema(),
	}
}

// SetAuthenticator sets the authenticator
func (s *Service) SetAuthenticator(auth Authenticator) {
	s.auth = auth
}

// SetPublisher sets the event publisher
func (s *Service) SetPublisher(publisher EventPublisher) {
	s.publisher = publisher
}

// SetSchemaGenerator sets the schema generator
func (s *Service) SetSchemaGenerator(gen SchemaGenerator) {
	s.schemaGen = gen
}

// RegisterClient registers a new WebSocket client
func (s *Service) RegisterClient(tenantID, protocol string) (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.clients) >= s.config.MaxConnections {
		return nil, errors.New("max connections reached")
	}

	client := &Client{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		State:         ClientStateConnected,
		Protocol:      protocol,
		ConnectedAt:   time.Now(),
		Subscriptions: make(map[string]*Subscription),
		SendCh:        make(chan []byte, 256),
		CloseCh:       make(chan struct{}),
	}

	s.clients[client.ID] = client
	return client, nil
}

// UnregisterClient removes a client
func (s *Service) UnregisterClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.clients[clientID]; ok {
		close(client.CloseCh)
		delete(s.clients, clientID)
	}
}

// Subscribe creates a new subscription for a client
func (s *Service) Subscribe(ctx context.Context, clientID, subscriptionID string, payload *SubscriptionPayload) error {
	s.mu.RLock()
	client, ok := s.clients[clientID]
	s.mu.RUnlock()

	if !ok {
		return ErrClientNotFound
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if len(client.Subscriptions) >= s.config.MaxSubscriptions {
		return errors.New("max subscriptions reached")
	}

	// Parse subscription type from query
	subType := s.parseSubscriptionType(payload.Query)

	sub := &Subscription{
		ID:        subscriptionID,
		TenantID:  client.TenantID,
		ClientID:  clientID,
		Type:      subType,
		Query:     payload.Query,
		Variables: payload.Variables,
		Active:    true,
		CreatedAt: time.Now(),
	}

	// Validate subscription
	if s.auth != nil {
		if err := s.auth.ValidateSubscription(ctx, client.TenantID, sub); err != nil {
			return err
		}
	}

	client.Subscriptions[subscriptionID] = sub
	client.State = ClientStateSubscribed

	// best-effort: persist subscription state after client operation succeeds
	_ = s.repo.SaveSubscription(ctx, sub)

	return nil
}

// Unsubscribe removes a subscription
func (s *Service) Unsubscribe(ctx context.Context, clientID, subscriptionID string) error {
	s.mu.RLock()
	client, ok := s.clients[clientID]
	s.mu.RUnlock()

	if !ok {
		return ErrClientNotFound
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	delete(client.Subscriptions, subscriptionID)
	// best-effort: clean up subscription record after in-memory removal succeeds
	_ = s.repo.DeleteSubscription(ctx, subscriptionID)

	return nil
}

// parseSubscriptionType parses the subscription type from a GraphQL query
func (s *Service) parseSubscriptionType(query string) SubscriptionType {
	// Simple parsing - in production, use a proper GraphQL parser
	switch {
	case contains(query, "webhookEvents"):
		return SubscriptionWebhookEvents
	case contains(query, "deliveryStatus"):
		return SubscriptionDeliveryStatus
	case contains(query, "endpointHealth"):
		return SubscriptionEndpointHealth
	case contains(query, "workflowStatus"):
		return SubscriptionWorkflowStatus
	default:
		return SubscriptionWebhookEvents
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// PublishEvent publishes an event to matching subscriptions
func (s *Service) PublishEvent(ctx context.Context, event *Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.TenantID != event.TenantID {
			continue
		}

		client.mu.RLock()
		for subID, sub := range client.Subscriptions {
			if sub.Type != event.Type || !sub.Active {
				continue
			}

			// Check filters
			if !s.matchesFilters(sub.Filters, event) {
				continue
			}

			// Send event
			msg := &GraphQLMessage{
				ID:   subID,
				Type: MessageTypeNext,
				Payload: mustMarshal(&EventPayload{
					Data: json.RawMessage(event.Payload),
				}),
			}

			select {
			case client.SendCh <- mustMarshal(msg):
				now := time.Now()
				sub.LastEventAt = &now
			default:
				// Channel full, skip
			}
		}
		client.mu.RUnlock()
	}

	// best-effort: persist event for replay; delivery to subscribers already succeeded
	_ = s.repo.SaveEvent(ctx, event)
	_ = s.repo.IncrementEventCounter(ctx, event.TenantID, 1)

	return nil
}

// matchesFilters checks if an event matches subscription filters
func (s *Service) matchesFilters(filters *SubscriptionFilters, event *Event) bool {
	if filters == nil {
		return true
	}

	// Get event type from metadata
	if len(filters.EventTypes) > 0 {
		eventType := event.Metadata["event_type"]
		matched := false
		for _, t := range filters.EventTypes {
			if t == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check endpoint ID
	if len(filters.EndpointIDs) > 0 {
		endpointID := event.Metadata["endpoint_id"]
		matched := false
		for _, id := range filters.EndpointIDs {
			if id == endpointID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// GetSchema returns the GraphQL schema
func (s *Service) GetSchema() *Schema {
	return s.schema
}

// GetStats returns service statistics
func (s *Service) GetStats(ctx context.Context, tenantID string) (*ClientStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &ClientStats{}

	for _, client := range s.clients {
		if tenantID != "" && client.TenantID != tenantID {
			continue
		}
		stats.TotalClients++
		if client.State == ClientStateSubscribed {
			stats.ActiveClients++
		}
		stats.TotalSubscriptions += len(client.Subscriptions)
	}

	// Get persisted stats
	if repoStats, err := s.repo.GetStats(ctx, tenantID); err == nil {
		stats.EventsPublished = repoStats.EventsPublished
		stats.EventsDelivered = repoStats.EventsDelivered
	}

	return stats, nil
}

// GenerateDefaultSchema generates the default GraphQL subscription schema
func GenerateDefaultSchema() *Schema {
	return &Schema{
		Types: []SchemaType{
			{
				Name: "WebhookEvent",
				Kind: "OBJECT",
				Fields: []SchemaField{
					{Name: "id", Type: "ID!", Nullable: false},
					{Name: "type", Type: "String!", Nullable: false},
					{Name: "endpointId", Type: "ID!", Nullable: false},
					{Name: "endpointUrl", Type: "String!", Nullable: false},
					{Name: "payload", Type: "JSON!", Nullable: false},
					{Name: "timestamp", Type: "DateTime!", Nullable: false},
					{Name: "attempt", Type: "Int!", Nullable: false},
					{Name: "status", Type: "String!", Nullable: false},
					{Name: "responseCode", Type: "Int", Nullable: true},
					{Name: "responseTimeMs", Type: "Int", Nullable: true},
				},
			},
			{
				Name: "DeliveryStatus",
				Kind: "OBJECT",
				Fields: []SchemaField{
					{Name: "deliveryId", Type: "ID!", Nullable: false},
					{Name: "webhookId", Type: "ID!", Nullable: false},
					{Name: "endpointId", Type: "ID!", Nullable: false},
					{Name: "status", Type: "String!", Nullable: false},
					{Name: "attempt", Type: "Int!", Nullable: false},
					{Name: "responseCode", Type: "Int", Nullable: true},
					{Name: "responseTimeMs", Type: "Int", Nullable: true},
					{Name: "errorMessage", Type: "String", Nullable: true},
					{Name: "timestamp", Type: "DateTime!", Nullable: false},
				},
			},
			{
				Name: "EndpointHealth",
				Kind: "OBJECT",
				Fields: []SchemaField{
					{Name: "endpointId", Type: "ID!", Nullable: false},
					{Name: "endpointUrl", Type: "String!", Nullable: false},
					{Name: "status", Type: "String!", Nullable: false},
					{Name: "healthScore", Type: "Float!", Nullable: false},
					{Name: "successRate", Type: "Float!", Nullable: false},
					{Name: "averageLatencyMs", Type: "Float!", Nullable: false},
					{Name: "lastChecked", Type: "DateTime!", Nullable: false},
				},
			},
			{
				Name: "WorkflowStatus",
				Kind: "OBJECT",
				Fields: []SchemaField{
					{Name: "workflowId", Type: "ID!", Nullable: false},
					{Name: "executionId", Type: "ID!", Nullable: false},
					{Name: "status", Type: "String!", Nullable: false},
					{Name: "currentNode", Type: "String", Nullable: true},
					{Name: "progress", Type: "Int!", Nullable: false},
					{Name: "startedAt", Type: "DateTime!", Nullable: false},
					{Name: "completedAt", Type: "DateTime", Nullable: true},
					{Name: "errorMessage", Type: "String", Nullable: true},
				},
			},
		},
		Subscriptions: []SchemaField{
			{
				Name:        "webhookEvents",
				Type:        "WebhookEvent!",
				Description: "Subscribe to webhook events",
				Args: []SchemaArg{
					{Name: "eventTypes", Type: "[String!]", Description: "Filter by event types"},
					{Name: "endpointIds", Type: "[ID!]", Description: "Filter by endpoint IDs"},
				},
			},
			{
				Name:        "deliveryStatus",
				Type:        "DeliveryStatus!",
				Description: "Subscribe to delivery status updates",
				Args: []SchemaArg{
					{Name: "webhookId", Type: "ID", Description: "Filter by webhook ID"},
					{Name: "endpointId", Type: "ID", Description: "Filter by endpoint ID"},
				},
			},
			{
				Name:        "endpointHealth",
				Type:        "EndpointHealth!",
				Description: "Subscribe to endpoint health updates",
				Args: []SchemaArg{
					{Name: "endpointIds", Type: "[ID!]", Description: "Filter by endpoint IDs"},
				},
			},
			{
				Name:        "workflowStatus",
				Type:        "WorkflowStatus!",
				Description: "Subscribe to workflow execution status",
				Args: []SchemaArg{
					{Name: "workflowId", Type: "ID", Description: "Filter by workflow ID"},
					{Name: "executionId", Type: "ID", Description: "Filter by execution ID"},
				},
			},
		},
	}
}

// mustMarshal marshals to JSON, returning nil on error (internal use only).
// Error is intentionally discarded: callers expect valid input for known types.
func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

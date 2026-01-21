package federation

import (
	"time"
)

// FederationMember represents an organization in the federation
type FederationMember struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Domain         string         `json:"domain"`
	Status         MemberStatus   `json:"status"`
	PublicKey      string         `json:"public_key"`
	Endpoints      []FedEndpoint  `json:"endpoints"`
	Capabilities   []Capability   `json:"capabilities"`
	TrustLevel     TrustLevel     `json:"trust_level"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	JoinedAt       time.Time      `json:"joined_at"`
	LastSeenAt     time.Time      `json:"last_seen_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// MemberStatus federation member status
type MemberStatus string

const (
	MemberPending   MemberStatus = "pending"
	MemberActive    MemberStatus = "active"
	MemberSuspended MemberStatus = "suspended"
	MemberRevoked   MemberStatus = "revoked"
)

// TrustLevel trust level between members
type TrustLevel string

const (
	TrustNone     TrustLevel = "none"
	TrustBasic    TrustLevel = "basic"
	TrustVerified TrustLevel = "verified"
	TrustTrusted  TrustLevel = "trusted"
)

// Capability federation capabilities
type Capability string

const (
	CapabilityReceive   Capability = "receive"
	CapabilitySend      Capability = "send"
	CapabilityRelay     Capability = "relay"
	CapabilityDiscover  Capability = "discover"
)

// FedEndpoint federation endpoint
type FedEndpoint struct {
	URL      string            `json:"url"`
	Type     FedEndpointType   `json:"type"`
	Priority int               `json:"priority"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// FedEndpointType federation endpoint type
type FedEndpointType string

const (
	EndpointWebhook  FedEndpointType = "webhook"
	EndpointGRPC     FedEndpointType = "grpc"
	EndpointGraphQL  FedEndpointType = "graphql"
)

// TrustRelationship represents trust between two members
type TrustRelationship struct {
	ID             string        `json:"id"`
	TenantID       string        `json:"tenant_id"`
	SourceMemberID string        `json:"source_member_id"`
	TargetMemberID string        `json:"target_member_id"`
	Status         TrustStatus   `json:"status"`
	TrustLevel     TrustLevel    `json:"trust_level"`
	Permissions    []Permission  `json:"permissions"`
	ExpiresAt      *time.Time    `json:"expires_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// TrustStatus trust relationship status
type TrustStatus string

const (
	TrustStatusPending   TrustStatus = "pending"
	TrustStatusActive    TrustStatus = "active"
	TrustStatusExpired   TrustStatus = "expired"
	TrustStatusRevoked   TrustStatus = "revoked"
)

// Permission federation permission
type Permission struct {
	Type        PermissionType `json:"type"`
	EventTypes  []string       `json:"event_types,omitempty"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// PermissionType permission types
type PermissionType string

const (
	PermissionPublish    PermissionType = "publish"
	PermissionSubscribe  PermissionType = "subscribe"
	PermissionRelay      PermissionType = "relay"
	PermissionInspect    PermissionType = "inspect"
)

// EventCatalog shared event catalog
type EventCatalog struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	MemberID    string       `json:"member_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	EventTypes  []EventType  `json:"event_types"`
	Version     string       `json:"version"`
	Public      bool         `json:"public"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// EventType event type definition
type EventType struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
	Version     string         `json:"version"`
	Deprecated  bool           `json:"deprecated"`
	Examples    []map[string]any `json:"examples,omitempty"`
}

// FederatedSubscription cross-org subscription
type FederatedSubscription struct {
	ID              string      `json:"id"`
	TenantID        string      `json:"tenant_id"`
	SourceMemberID  string      `json:"source_member_id"`
	TargetMemberID  string      `json:"target_member_id"`
	CatalogID       string      `json:"catalog_id"`
	EventTypes      []string    `json:"event_types"`
	Filter          *EventFilter `json:"filter,omitempty"`
	Status          SubStatus   `json:"status"`
	DeliveryConfig  DeliveryConfig `json:"delivery_config"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// SubStatus subscription status
type SubStatus string

const (
	SubStatusPending   SubStatus = "pending"
	SubStatusActive    SubStatus = "active"
	SubStatusPaused    SubStatus = "paused"
	SubStatusCanceled  SubStatus = "canceled"
)

// EventFilter filters events for subscription
type EventFilter struct {
	Conditions []FilterCondition `json:"conditions"`
	Expression string            `json:"expression,omitempty"` // CEL expression
}

// FilterCondition filter condition
type FilterCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, gt, lt, contains, etc.
	Value    any    `json:"value"`
}

// DeliveryConfig delivery configuration
type DeliveryConfig struct {
	Endpoint     string            `json:"endpoint"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers,omitempty"`
	SignatureKey string            `json:"signature_key,omitempty"`
	RetryPolicy  RetryPolicy       `json:"retry_policy"`
	Timeout      int               `json:"timeout"` // seconds
	BatchSize    int               `json:"batch_size"`
}

// RetryPolicy retry configuration
type RetryPolicy struct {
	MaxRetries     int   `json:"max_retries"`
	InitialDelay   int   `json:"initial_delay"`   // seconds
	MaxDelay       int   `json:"max_delay"`       // seconds
	BackoffFactor  float64 `json:"backoff_factor"`
}

// FederatedDelivery represents a cross-org delivery
type FederatedDelivery struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	SubscriptionID   string         `json:"subscription_id"`
	SourceMemberID   string         `json:"source_member_id"`
	TargetMemberID   string         `json:"target_member_id"`
	EventType        string         `json:"event_type"`
	EventID          string         `json:"event_id"`
	Payload          map[string]any `json:"payload"`
	Status           DeliveryStatus `json:"status"`
	Attempts         int            `json:"attempts"`
	LastAttemptAt    *time.Time     `json:"last_attempt_at,omitempty"`
	NextRetryAt      *time.Time     `json:"next_retry_at,omitempty"`
	Error            string         `json:"error,omitempty"`
	ResponseCode     int            `json:"response_code,omitempty"`
	ResponseBody     string         `json:"response_body,omitempty"`
	Latency          int64          `json:"latency"` // milliseconds
	DeliveredAt      *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// DeliveryStatus delivery status
type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryInFlight  DeliveryStatus = "in_flight"
	DeliverySucceeded DeliveryStatus = "succeeded"
	DeliveryFailed    DeliveryStatus = "failed"
	DeliveryRetrying  DeliveryStatus = "retrying"
)

// FederationPolicy federation policies
type FederationPolicy struct {
	ID                   string        `json:"id"`
	TenantID             string        `json:"tenant_id"`
	Enabled              bool          `json:"enabled"`
	AutoAcceptTrust      bool          `json:"auto_accept_trust"`
	MinTrustLevel        TrustLevel    `json:"min_trust_level"`
	AllowedDomains       []string      `json:"allowed_domains"`
	BlockedDomains       []string      `json:"blocked_domains"`
	RequireEncryption    bool          `json:"require_encryption"`
	AllowRelay           bool          `json:"allow_relay"`
	MaxSubscriptions     int           `json:"max_subscriptions"`
	RateLimitPerMember   int           `json:"rate_limit_per_member"` // requests per minute
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

// FederationEvent event received from federation
type FederationEvent struct {
	ID              string         `json:"id"`
	SourceMemberID  string         `json:"source_member_id"`
	EventType       string         `json:"event_type"`
	EventID         string         `json:"event_id"`
	Payload         map[string]any `json:"payload"`
	Timestamp       time.Time      `json:"timestamp"`
	Signature       string         `json:"signature"`
	Metadata        EventMetadata  `json:"metadata"`
}

// EventMetadata event metadata
type EventMetadata struct {
	TraceID      string            `json:"trace_id,omitempty"`
	CorrelationID string           `json:"correlation_id,omitempty"`
	Hops         []HopInfo         `json:"hops,omitempty"`
	Custom       map[string]string `json:"custom,omitempty"`
}

// HopInfo relay hop information
type HopInfo struct {
	MemberID    string    `json:"member_id"`
	ReceivedAt  time.Time `json:"received_at"`
	ForwardedAt time.Time `json:"forwarded_at"`
}

// DiscoveryRequest discovery protocol request
type DiscoveryRequest struct {
	RequestID    string   `json:"request_id"`
	RequesterID  string   `json:"requester_id"`
	EventTypes   []string `json:"event_types,omitempty"`
	Capabilities []Capability `json:"capabilities,omitempty"`
}

// DiscoveryResponse discovery protocol response
type DiscoveryResponse struct {
	RequestID   string             `json:"request_id"`
	Members     []DiscoveredMember `json:"members"`
	Catalogs    []CatalogSummary   `json:"catalogs"`
}

// DiscoveredMember discovered federation member
type DiscoveredMember struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Domain       string       `json:"domain"`
	Capabilities []Capability `json:"capabilities"`
	TrustLevel   TrustLevel   `json:"trust_level"`
	EventTypes   []string     `json:"event_types"`
}

// CatalogSummary catalog summary for discovery
type CatalogSummary struct {
	ID          string   `json:"id"`
	MemberID    string   `json:"member_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	EventCount  int      `json:"event_count"`
	Version     string   `json:"version"`
}

// HealthCheck federation health check
type HealthCheck struct {
	MemberID    string    `json:"member_id"`
	Status      string    `json:"status"` // healthy, degraded, unhealthy
	Latency     int64     `json:"latency"` // milliseconds
	CheckedAt   time.Time `json:"checked_at"`
	Details     map[string]any `json:"details,omitempty"`
}

// FederationMetrics federation metrics
type FederationMetrics struct {
	TenantID              string    `json:"tenant_id"`
	TotalMembers          int       `json:"total_members"`
	ActiveMembers         int       `json:"active_members"`
	TotalSubscriptions    int       `json:"total_subscriptions"`
	TotalDeliveries       int64     `json:"total_deliveries"`
	SuccessfulDeliveries  int64     `json:"successful_deliveries"`
	FailedDeliveries      int64     `json:"failed_deliveries"`
	AverageLatency        float64   `json:"average_latency"` // milliseconds
	EventsReceived        int64     `json:"events_received"`
	EventsSent            int64     `json:"events_sent"`
	Period                string    `json:"period"` // last hour, day, week
	UpdatedAt             time.Time `json:"updated_at"`
}

// TrustRequest trust establishment request
type TrustRequest struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	RequesterID     string          `json:"requester_id"`
	TargetMemberID  string          `json:"target_member_id"`
	RequestedLevel  TrustLevel      `json:"requested_level"`
	Permissions     []Permission    `json:"permissions"`
	Message         string          `json:"message,omitempty"`
	Status          TrustReqStatus  `json:"status"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	RespondedAt     *time.Time      `json:"responded_at,omitempty"`
	Response        string          `json:"response,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// TrustReqStatus trust request status
type TrustReqStatus string

const (
	TrustReqPending  TrustReqStatus = "pending"
	TrustReqApproved TrustReqStatus = "approved"
	TrustReqRejected TrustReqStatus = "rejected"
	TrustReqExpired  TrustReqStatus = "expired"
)

// CryptoKeys federation cryptographic keys
type CryptoKeys struct {
	MemberID    string    `json:"member_id"`
	PublicKey   string    `json:"public_key"`
	Algorithm   string    `json:"algorithm"` // ed25519, rsa-2048
	KeyID       string    `json:"key_id"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Revoked     bool      `json:"revoked"`
}

// SignedMessage signed federation message
type SignedMessage struct {
	Message   []byte    `json:"message"`
	Signature string    `json:"signature"`
	KeyID     string    `json:"key_id"`
	Algorithm string    `json:"algorithm"`
	Timestamp time.Time `json:"timestamp"`
}

// Type aliases and additional types for test compatibility

// SharedEvent represents an event shared across the federation
type SharedEvent struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	MemberID      string                 `json:"member_id"`
	EventType     string                 `json:"event_type"`
	Visibility    EventVisibility        `json:"visibility"`
	Schema        map[string]interface{} `json:"schema"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Tags          []string               `json:"tags"`
	SamplePayload map[string]interface{} `json:"sample_payload"`
	PublishedAt   time.Time              `json:"published_at"`
}

// EventVisibility defines who can see/subscribe to events
type EventVisibility string

const (
	VisibilityPrivate    EventVisibility = "private"
	VisibilityFederation EventVisibility = "federation"
	VisibilityPublic     EventVisibility = "public"
)

// CrossOrgSubscription represents a cross-organization subscription
type CrossOrgSubscription struct {
	ID             string        `json:"id"`
	SubscriberID   string        `json:"subscriber_id"`
	PublisherID    string        `json:"publisher_id"`
	EventType      string        `json:"event_type"`
	TargetEndpoint string        `json:"target_endpoint"`
	TransformRules []TransformRule `json:"transform_rules"`
	FilterRules    []FilterRule  `json:"filter_rules"`
	RequiredTrust  TrustLevel    `json:"required_trust"`
	Status         SubStatus     `json:"status"`
	CreatedAt      time.Time     `json:"created_at"`
}

// TransformRule defines payload transformation
type TransformRule struct {
	SourcePath string                 `json:"source_path"`
	TargetPath string                 `json:"target_path"`
	Transform  string                 `json:"transform"`
	Params     map[string]interface{} `json:"params"`
}

// FilterRule defines event filtering
type FilterRule struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

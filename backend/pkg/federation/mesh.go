package federation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// FederationProtocol defines the protocol version for federation
const FederationProtocol = "waas-federation/v1"

// PeerStatus represents the status of a federation peer
type PeerStatus string

const (
	PeerStatusPending   PeerStatus = "pending"
	PeerStatusActive    PeerStatus = "active"
	PeerStatusSuspended PeerStatus = "suspended"
	PeerStatusRevoked   PeerStatus = "revoked"
)

// EventAttestation provides cryptographic proof of event origin
type EventAttestation struct {
	ID         string     `json:"id"`
	EventID    string     `json:"event_id"`
	SourcePeer string     `json:"source_peer"`
	TargetPeer string     `json:"target_peer"`
	Signature  string     `json:"signature"`    // Base64-encoded Ed25519 signature
	PublicKey  string     `json:"public_key"`   // Base64-encoded public key
	Algorithm  string     `json:"algorithm"`    // ed25519
	Payload    string     `json:"payload_hash"` // SHA256 of payload
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// FederatedEvent represents a webhook event routed across federation peers
type FederatedEvent struct {
	ID            string            `json:"id"`
	SourcePeerID  string            `json:"source_peer_id"`
	TargetPeerID  string            `json:"target_peer_id"`
	EventType     string            `json:"event_type"`
	SchemaVersion string            `json:"schema_version"`
	Payload       json.RawMessage   `json:"payload"`
	Headers       map[string]string `json:"headers,omitempty"`
	Attestation   *EventAttestation `json:"attestation,omitempty"`
	Status        string            `json:"status"` // pending, delivered, failed, rejected
	Hops          []string          `json:"hops"`   // Peer IDs traversed
	MaxHops       int               `json:"max_hops"`
	TTL           time.Duration     `json:"ttl"`
	CreatedAt     time.Time         `json:"created_at"`
	DeliveredAt   *time.Time        `json:"delivered_at,omitempty"`
}

// SharedEventSchema represents a shared event schema in the federation
type SharedEventSchema struct {
	ID          string    `json:"id"`
	PeerID      string    `json:"peer_id"`
	EventType   string    `json:"event_type"`
	Version     string    `json:"version"`
	Schema      string    `json:"schema"` // JSON Schema
	Description string    `json:"description,omitempty"`
	IsPublic    bool      `json:"is_public"`
	Subscribers []string  `json:"subscribers,omitempty"` // Peer IDs subscribed
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FederationGovernancePolicy defines rules for federation mesh
type FederationGovernancePolicy struct {
	ID                 string    `json:"id"`
	PeerID             string    `json:"peer_id"`
	MaxEventsPerDay    int64     `json:"max_events_per_day"`
	MaxPayloadSize     int64     `json:"max_payload_size_bytes"`
	RequireAttestation bool      `json:"require_attestation"`
	AllowedEventTypes  []string  `json:"allowed_event_types,omitempty"`
	BlockedPeers       []string  `json:"blocked_peers,omitempty"`
	MaxHops            int       `json:"max_hops"`
	RetentionDays      int       `json:"retention_days"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// RegisterPeerRequest represents a request to register as a federation peer
type RegisterPeerRequest struct {
	Name         string   `json:"name" binding:"required"`
	Endpoint     string   `json:"endpoint" binding:"required"`
	PublicKey    string   `json:"public_key" binding:"required"`
	Capabilities []string `json:"capabilities,omitempty"`
	Description  string   `json:"description,omitempty"`
}

// RouteEventRequest represents a request to route an event across federation
type RouteEventRequest struct {
	TargetPeerID  string            `json:"target_peer_id" binding:"required"`
	EventType     string            `json:"event_type" binding:"required"`
	SchemaVersion string            `json:"schema_version,omitempty"`
	Payload       json.RawMessage   `json:"payload" binding:"required"`
	Headers       map[string]string `json:"headers,omitempty"`
	MaxHops       int               `json:"max_hops,omitempty"`
}

// PublishSchemaRequest represents a request to publish an event schema
type PublishSchemaRequest struct {
	EventType   string `json:"event_type" binding:"required"`
	Version     string `json:"version" binding:"required"`
	Schema      string `json:"schema" binding:"required"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public"`
}

// FederationMeshRepository defines storage for federation mesh
type FederationMeshRepository interface {
	SaveFederatedEvent(ctx context.Context, event *FederatedEvent) error
	GetFederatedEvent(ctx context.Context, eventID string) (*FederatedEvent, error)
	ListFederatedEvents(ctx context.Context, peerID string, limit int) ([]FederatedEvent, error)
	UpdateEventStatus(ctx context.Context, eventID, status string) error
	SaveAttestation(ctx context.Context, attestation *EventAttestation) error
	VerifyAttestation(ctx context.Context, attestationID string) error
	SaveSchema(ctx context.Context, schema *SharedEventSchema) error
	GetSchema(ctx context.Context, eventType, version string) (*SharedEventSchema, error)
	ListSchemas(ctx context.Context, publicOnly bool) ([]SharedEventSchema, error)
	SubscribeToSchema(ctx context.Context, schemaID, peerID string) error
	SaveGovernancePolicy(ctx context.Context, policy *FederationGovernancePolicy) error
	GetGovernancePolicy(ctx context.Context, peerID string) (*FederationGovernancePolicy, error)
}

// FederationMesh manages cross-organization webhook routing
type FederationMesh struct {
	repo       FederationMeshRepository
	memberRepo Repository
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	selfPeerID string
	logger     *utils.Logger
}

// NewFederationMesh creates a new federation mesh
func NewFederationMesh(repo FederationMeshRepository, memberRepo Repository, selfPeerID string) (*FederationMesh, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	return &FederationMesh{
		repo:       repo,
		memberRepo: memberRepo,
		privateKey: priv,
		publicKey:  pub,
		selfPeerID: selfPeerID,
		logger:     utils.NewLogger("federation"),
	}, nil
}

// RouteEvent routes a webhook event to a federation peer
func (m *FederationMesh) RouteEvent(ctx context.Context, req *RouteEventRequest) (*FederatedEvent, error) {
	maxHops := req.MaxHops
	if maxHops == 0 {
		maxHops = 3
	}

	// Create attestation
	attestation := m.createAttestation(req.TargetPeerID, req.Payload)

	now := time.Now()
	event := &FederatedEvent{
		ID:            uuid.New().String(),
		SourcePeerID:  m.selfPeerID,
		TargetPeerID:  req.TargetPeerID,
		EventType:     req.EventType,
		SchemaVersion: req.SchemaVersion,
		Payload:       req.Payload,
		Headers:       req.Headers,
		Attestation:   attestation,
		Status:        "pending",
		Hops:          []string{m.selfPeerID},
		MaxHops:       maxHops,
		TTL:           24 * time.Hour,
		CreatedAt:     now,
	}

	// Validate against governance policy
	if err := m.validateGovernance(ctx, event); err != nil {
		event.Status = "rejected"
		if saveErr := m.repo.SaveFederatedEvent(ctx, event); saveErr != nil {
			m.logger.Error("SaveFederatedEvent (rejected) error", map[string]interface{}{"error": saveErr.Error()})
		}
		return event, err
	}

	if err := m.repo.SaveFederatedEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to save federated event: %w", err)
	}

	return event, nil
}

func (m *FederationMesh) createAttestation(targetPeer string, payload json.RawMessage) *EventAttestation {
	attestationID := uuid.New().String()

	// Sign the payload
	signature := ed25519.Sign(m.privateKey, payload)

	return &EventAttestation{
		ID:         attestationID,
		SourcePeer: m.selfPeerID,
		TargetPeer: targetPeer,
		Signature:  base64.StdEncoding.EncodeToString(signature),
		PublicKey:  base64.StdEncoding.EncodeToString(m.publicKey),
		Algorithm:  "ed25519",
		CreatedAt:  time.Now(),
	}
}

func (m *FederationMesh) validateGovernance(ctx context.Context, event *FederatedEvent) error {
	policy, err := m.repo.GetGovernancePolicy(ctx, m.selfPeerID)
	if err != nil {
		return nil // No policy = allow all
	}

	// Check payload size
	if policy.MaxPayloadSize > 0 && int64(len(event.Payload)) > policy.MaxPayloadSize {
		return fmt.Errorf("payload exceeds maximum size: %d bytes", policy.MaxPayloadSize)
	}

	// Check blocked peers
	for _, blocked := range policy.BlockedPeers {
		if blocked == event.TargetPeerID {
			return fmt.Errorf("peer is blocked: %s", event.TargetPeerID)
		}
	}

	// Check allowed event types
	if len(policy.AllowedEventTypes) > 0 {
		allowed := false
		for _, et := range policy.AllowedEventTypes {
			if et == event.EventType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("event type not allowed: %s", event.EventType)
		}
	}

	// Check max hops
	if policy.MaxHops > 0 && event.MaxHops > policy.MaxHops {
		event.MaxHops = policy.MaxHops
	}

	return nil
}

// VerifyAttestation verifies the cryptographic attestation of a federated event
func (m *FederationMesh) VerifyAttestation(ctx context.Context, event *FederatedEvent) (bool, error) {
	if event.Attestation == nil {
		return false, fmt.Errorf("no attestation present")
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(event.Attestation.PublicKey)
	if err != nil {
		return false, fmt.Errorf("invalid public key: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(event.Attestation.Signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature: %w", err)
	}

	verified := ed25519.Verify(ed25519.PublicKey(pubKeyBytes), event.Payload, sigBytes)
	if verified {
		now := time.Now()
		event.Attestation.Verified = true
		event.Attestation.VerifiedAt = &now
		if err := m.repo.VerifyAttestation(ctx, event.Attestation.ID); err != nil {
			m.logger.Warn("VerifyAttestation error", map[string]interface{}{"attestation_id": event.Attestation.ID, "error": err.Error()})
		}
	}

	return verified, nil
}

// PublishSchema publishes an event schema to the federation
func (m *FederationMesh) PublishSchema(ctx context.Context, peerID string, req *PublishSchemaRequest) (*SharedEventSchema, error) {
	now := time.Now()
	schema := &SharedEventSchema{
		ID:          uuid.New().String(),
		PeerID:      peerID,
		EventType:   req.EventType,
		Version:     req.Version,
		Schema:      req.Schema,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.repo.SaveSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to publish schema: %w", err)
	}

	return schema, nil
}

// ListSchemas lists available event schemas
func (m *FederationMesh) ListSchemas(ctx context.Context, publicOnly bool) ([]SharedEventSchema, error) {
	return m.repo.ListSchemas(ctx, publicOnly)
}

// ListFederatedEvents lists federated events
func (m *FederationMesh) ListFederatedEvents(ctx context.Context, peerID string, limit int) ([]FederatedEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return m.repo.ListFederatedEvents(ctx, peerID, limit)
}

// SetGovernancePolicy sets governance rules
func (m *FederationMesh) SetGovernancePolicy(ctx context.Context, policy *FederationGovernancePolicy) error {
	policy.ID = uuid.New().String()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	return m.repo.SaveGovernancePolicy(ctx, policy)
}

// GetGovernancePolicy retrieves governance policy
func (m *FederationMesh) GetGovernancePolicy(ctx context.Context, peerID string) (*FederationGovernancePolicy, error) {
	return m.repo.GetGovernancePolicy(ctx, peerID)
}

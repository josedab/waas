package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConnectorInterface is the standardized interface for all event source connectors
type ConnectorInterface interface {
	// ID returns the unique connector identifier
	ID() string
	// Name returns the human-readable connector name
	Name() string
	// Provider returns the provider slug (e.g., "stripe", "github")
	Provider() string
	// Version returns the connector version
	Version() string
	// EventTypes returns the list of event types this connector handles
	EventTypes() []string
	// VerifySignature verifies the webhook signature from this provider
	VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error)
	// NormalizePayload converts provider-specific payload to the standard WaaS envelope
	NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error)
	// DetectProvider checks if inbound headers/payload match this connector
	DetectProvider(headers map[string]string, payload []byte) (float64, error) // returns confidence 0-1
	// ConfigSchema returns the JSON schema for connector configuration
	ConfigSchema() json.RawMessage
}

// NormalizedEvent is the standard event envelope after normalization
type NormalizedEvent struct {
	ID         string                 `json:"id"`
	Provider   string                 `json:"provider"`
	EventType  string                 `json:"event_type"`
	ExternalID string                 `json:"external_id,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Payload    map[string]interface{} `json:"payload"`
	Metadata   map[string]string      `json:"metadata,omitempty"`
	RawPayload json.RawMessage        `json:"raw_payload,omitempty"`
}

// ConnectorRegistry manages all registered connectors
type ConnectorRegistry struct {
	connectors map[string]ConnectorInterface
}

// NewConnectorRegistry creates a registry with all built-in connectors
func NewConnectorRegistry() *ConnectorRegistry {
	r := &ConnectorRegistry{
		connectors: make(map[string]ConnectorInterface),
	}
	// Register built-in connectors
	for _, c := range builtinConnectors() {
		r.connectors[c.Provider()] = c
	}
	return r
}

// Get returns a connector by provider name
func (r *ConnectorRegistry) Get(provider string) (ConnectorInterface, bool) {
	c, ok := r.connectors[strings.ToLower(provider)]
	return c, ok
}

// List returns all registered connectors
func (r *ConnectorRegistry) List() []ConnectorInterface {
	var list []ConnectorInterface
	for _, c := range r.connectors {
		list = append(list, c)
	}
	return list
}

// Register adds a custom connector to the registry
func (r *ConnectorRegistry) Register(c ConnectorInterface) {
	r.connectors[c.Provider()] = c
}

// AutoDetect identifies the provider from headers and payload
func (r *ConnectorRegistry) AutoDetect(headers map[string]string, payload []byte) (ConnectorInterface, float64, error) {
	var bestMatch ConnectorInterface
	var bestConfidence float64

	for _, c := range r.connectors {
		confidence, err := c.DetectProvider(headers, payload)
		if err != nil {
			continue
		}
		if confidence > bestConfidence {
			bestConfidence = confidence
			bestMatch = c
		}
	}

	if bestMatch == nil {
		return nil, 0, fmt.Errorf("no matching connector found")
	}

	return bestMatch, bestConfidence, nil
}

// builtinConnectors returns the pre-built connectors for top providers
func builtinConnectors() []ConnectorInterface {
	return []ConnectorInterface{
		NewStripeConnector(),
		NewGitHubConnector(),
		NewShopifyConnector(),
		NewTwilioConnector(),
		NewSlackConnector(),
		NewSendGridConnector(),
		NewLinearConnector(),
		NewIntercomConnector(),
		NewHubSpotConnector(),
		NewZoomConnector(),
		NewAsanaConnector(),
		NewJiraConnector(),
		NewPaddleConnector(),
		NewSquareConnector(),
		NewBrexConnector(),
		NewPostmarkConnector(),
		NewSvixConnector(),
		NewClerkConnector(),
		NewResendConnector(),
		NewVercelConnector(),
	}
}

// ConnectorMetadata returns metadata for display in marketplace
type ConnectorMetadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	IconURL     string   `json:"icon_url"`
	DocsURL     string   `json:"docs_url"`
	EventTypes  []string `json:"event_types"`
	Category    string   `json:"category"`
	IsOfficial  bool     `json:"is_official"`
}

// GetConnectorMetadata returns metadata for all connectors
func (r *ConnectorRegistry) GetConnectorMetadata() []ConnectorMetadata {
	var metadata []ConnectorMetadata
	for _, c := range r.connectors {
		metadata = append(metadata, ConnectorMetadata{
			ID:         c.ID(),
			Name:       c.Name(),
			Provider:   c.Provider(),
			Version:    c.Version(),
			EventTypes: c.EventTypes(),
			IsOfficial: true,
		})
	}
	return metadata
}

// ProcessInbound processes an inbound webhook through the appropriate connector
func (r *ConnectorRegistry) ProcessInbound(ctx context.Context, provider string, payload []byte, headers map[string]string, secret string) (*NormalizedEvent, error) {
	connector, ok := r.Get(provider)
	if !ok {
		// Try auto-detection
		var err error
		connector, _, err = r.AutoDetect(headers, payload)
		if err != nil {
			return nil, fmt.Errorf("unknown provider %q and auto-detection failed: %w", provider, err)
		}
	}

	// Verify signature
	if secret != "" {
		valid, err := connector.VerifySignature(payload, headers, secret)
		if err != nil {
			return nil, fmt.Errorf("signature verification error: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("invalid signature for provider %s", connector.Provider())
		}
	}

	// Normalize payload
	event, err := connector.NormalizePayload(payload, headers)
	if err != nil {
		return nil, fmt.Errorf("payload normalization error: %w", err)
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	return event, nil
}

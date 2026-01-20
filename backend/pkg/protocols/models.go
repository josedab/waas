package protocols

import (
	"context"
	"time"
)

// Protocol represents a webhook delivery protocol
type Protocol string

const (
	ProtocolHTTP      Protocol = "http"
	ProtocolHTTPS     Protocol = "https"
	ProtocolGRPC      Protocol = "grpc"
	ProtocolGRPCS     Protocol = "grpcs"
	ProtocolWebSocket Protocol = "websocket"
	ProtocolMQTT      Protocol = "mqtt"
)

// DeliveryConfig represents configuration for a protocol delivery
type DeliveryConfig struct {
	ID         string                 `json:"id" db:"id"`
	TenantID   string                 `json:"tenant_id" db:"tenant_id"`
	EndpointID string                 `json:"endpoint_id" db:"endpoint_id"`
	Protocol   Protocol               `json:"protocol" db:"protocol"`
	Target     string                 `json:"target" db:"target"`
	Options    map[string]interface{} `json:"options" db:"options"`
	Headers    map[string]string      `json:"headers,omitempty" db:"headers"`
	TLS        *TLSConfig             `json:"tls,omitempty" db:"tls"`
	Auth       *AuthConfig            `json:"auth,omitempty" db:"auth"`
	Timeout    int                    `json:"timeout" db:"timeout"` // seconds
	Retries    int                    `json:"retries" db:"retries"`
	Enabled    bool                   `json:"enabled" db:"enabled"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CACert             string `json:"ca_cert,omitempty"`
	ClientCert         string `json:"client_cert,omitempty"`
	ClientKey          string `json:"client_key,omitempty"`
	ServerName         string `json:"server_name,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type        AuthType          `json:"type"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// AuthType represents the type of authentication
type AuthType string

const (
	AuthNone        AuthType = "none"
	AuthBasic       AuthType = "basic"
	AuthBearer      AuthType = "bearer"
	AuthAPIKey      AuthType = "api_key"
	AuthOAuth2      AuthType = "oauth2"
	AuthMTLS        AuthType = "mtls"
	AuthGRPCPerCall AuthType = "grpc_per_call"
)

// DeliveryRequest represents a request to deliver a webhook
type DeliveryRequest struct {
	ID            string            `json:"id"`
	WebhookID     string            `json:"webhook_id"`
	EndpointID    string            `json:"endpoint_id"`
	Payload       []byte            `json:"payload"`
	ContentType   string            `json:"content_type"`
	Headers       map[string]string `json:"headers"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
	AttemptNumber int               `json:"attempt_number"`
}

// DeliveryResponse represents the response from a delivery attempt
type DeliveryResponse struct {
	Success      bool              `json:"success"`
	StatusCode   int               `json:"status_code,omitempty"`
	Body         []byte            `json:"body,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Error        string            `json:"error,omitempty"`
	ErrorType    DeliveryErrorType `json:"error_type,omitempty"`
	RetryAfter   *time.Duration    `json:"retry_after,omitempty"`
	ProtocolInfo map[string]any    `json:"protocol_info,omitempty"`
}

// DeliveryErrorType categorizes delivery errors
type DeliveryErrorType string

const (
	ErrorTypeNone        DeliveryErrorType = ""
	ErrorTypeConnection  DeliveryErrorType = "connection"
	ErrorTypeTimeout     DeliveryErrorType = "timeout"
	ErrorTypeTLS         DeliveryErrorType = "tls"
	ErrorTypeAuth        DeliveryErrorType = "auth"
	ErrorTypeProtocol    DeliveryErrorType = "protocol"
	ErrorTypeServer      DeliveryErrorType = "server"
	ErrorTypeClientError DeliveryErrorType = "client_error"
	ErrorTypeRateLimit   DeliveryErrorType = "rate_limit"
)

// GRPCOptions represents gRPC-specific delivery options
type GRPCOptions struct {
	Service           string            `json:"service"`
	Method            string            `json:"method"`
	ProtoFile         string            `json:"proto_file,omitempty"`
	ProtoDescriptor   []byte            `json:"proto_descriptor,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	MaxMessageSize    int               `json:"max_message_size,omitempty"`
	WaitForReady      bool              `json:"wait_for_ready"`
	Compression       string            `json:"compression,omitempty"`
	LoadBalancing     string            `json:"load_balancing,omitempty"`
	ConnectionTimeout int               `json:"connection_timeout,omitempty"`
}

// MQTTOptions represents MQTT-specific delivery options
type MQTTOptions struct {
	Topic      string `json:"topic"`
	QoS        int    `json:"qos"`
	Retain     bool   `json:"retain"`
	ClientID   string `json:"client_id,omitempty"`
	CleanStart bool   `json:"clean_start"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

// WebSocketOptions represents WebSocket-specific delivery options
type WebSocketOptions struct {
	Path               string            `json:"path,omitempty"`
	Subprotocols       []string          `json:"subprotocols,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	PingInterval       int               `json:"ping_interval,omitempty"`
	ReconnectOnFailure bool              `json:"reconnect_on_failure"`
	BinaryMode         bool              `json:"binary_mode"`
	WaitForResponse    bool              `json:"wait_for_response"`
}

// HTTPOptions represents HTTP-specific delivery options
type HTTPOptions struct {
	Method           string            `json:"method"`
	Path             string            `json:"path,omitempty"`
	QueryParams      map[string]string `json:"query_params,omitempty"`
	FollowRedirects  bool              `json:"follow_redirects"`
	MaxRedirects     int               `json:"max_redirects"`
	Compression      string            `json:"compression,omitempty"`
	ExpectedStatuses []int             `json:"expected_statuses,omitempty"`
}

// ProtocolInfo provides information about a protocol
type ProtocolInfo struct {
	Name          Protocol       `json:"name"`
	DisplayName   string         `json:"display_name"`
	Description   string         `json:"description"`
	Version       string         `json:"version"`
	Supported     bool           `json:"supported"`
	DefaultPort   int            `json:"default_port"`
	RequiresTLS   bool           `json:"requires_tls"`
	SupportsAuth  []AuthType     `json:"supports_auth"`
	OptionsSchema map[string]any `json:"options_schema"`
}

// Deliverer interface for protocol-specific delivery
type Deliverer interface {
	// Protocol returns the protocol this deliverer handles
	Protocol() Protocol

	// Deliver performs the delivery
	Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error)

	// Validate validates the delivery config
	Validate(config *DeliveryConfig) error

	// Close cleans up any resources
	Close() error
}

// CreateConfigRequest represents a request to create a protocol config
type CreateConfigRequest struct {
	EndpointID string                 `json:"endpoint_id" binding:"required"`
	Protocol   Protocol               `json:"protocol" binding:"required"`
	Target     string                 `json:"target" binding:"required"`
	Options    map[string]interface{} `json:"options"`
	Headers    map[string]string      `json:"headers,omitempty"`
	TLS        *TLSConfig             `json:"tls,omitempty"`
	Auth       *AuthConfig            `json:"auth,omitempty"`
	Timeout    int                    `json:"timeout"`
	Retries    int                    `json:"retries"`
}

// UpdateConfigRequest represents a request to update a protocol config
type UpdateConfigRequest struct {
	Protocol *Protocol              `json:"protocol,omitempty"`
	Target   *string                `json:"target,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	TLS      *TLSConfig             `json:"tls,omitempty"`
	Auth     *AuthConfig            `json:"auth,omitempty"`
	Timeout  *int                   `json:"timeout,omitempty"`
	Retries  *int                   `json:"retries,omitempty"`
	Enabled  *bool                  `json:"enabled,omitempty"`
}

// SupportedProtocols returns information about all supported protocols
func SupportedProtocols() []ProtocolInfo {
	return []ProtocolInfo{
		{
			Name:        ProtocolHTTP,
			DisplayName: "HTTP",
			Description: "Standard HTTP webhook delivery",
			Version:     "1.1/2",
			Supported:   true,
			DefaultPort: 80,
			RequiresTLS: false,
			SupportsAuth: []AuthType{
				AuthNone, AuthBasic, AuthBearer, AuthAPIKey, AuthOAuth2,
			},
			OptionsSchema: map[string]any{
				"method":           "string",
				"follow_redirects": "boolean",
				"max_redirects":    "integer",
			},
		},
		{
			Name:        ProtocolHTTPS,
			DisplayName: "HTTPS",
			Description: "HTTP over TLS webhook delivery",
			Version:     "1.1/2",
			Supported:   true,
			DefaultPort: 443,
			RequiresTLS: true,
			SupportsAuth: []AuthType{
				AuthNone, AuthBasic, AuthBearer, AuthAPIKey, AuthOAuth2, AuthMTLS,
			},
			OptionsSchema: map[string]any{
				"method":           "string",
				"follow_redirects": "boolean",
				"max_redirects":    "integer",
			},
		},
		{
			Name:        ProtocolGRPC,
			DisplayName: "gRPC",
			Description: "gRPC unary call delivery",
			Version:     "1.0",
			Supported:   true,
			DefaultPort: 50051,
			RequiresTLS: false,
			SupportsAuth: []AuthType{
				AuthNone, AuthGRPCPerCall, AuthMTLS,
			},
			OptionsSchema: map[string]any{
				"service":          "string",
				"method":           "string",
				"proto_file":       "string",
				"metadata":         "object",
				"max_message_size": "integer",
				"wait_for_ready":   "boolean",
			},
		},
		{
			Name:        ProtocolGRPCS,
			DisplayName: "gRPC (TLS)",
			Description: "gRPC over TLS",
			Version:     "1.0",
			Supported:   true,
			DefaultPort: 443,
			RequiresTLS: true,
			SupportsAuth: []AuthType{
				AuthNone, AuthGRPCPerCall, AuthMTLS,
			},
			OptionsSchema: map[string]any{
				"service":          "string",
				"method":           "string",
				"proto_file":       "string",
				"metadata":         "object",
				"max_message_size": "integer",
				"wait_for_ready":   "boolean",
			},
		},
		{
			Name:        ProtocolWebSocket,
			DisplayName: "WebSocket",
			Description: "WebSocket message delivery",
			Version:     "13",
			Supported:   true,
			DefaultPort: 80,
			RequiresTLS: false,
			SupportsAuth: []AuthType{
				AuthNone, AuthBearer, AuthAPIKey,
			},
			OptionsSchema: map[string]any{
				"path":                 "string",
				"subprotocols":         "array",
				"ping_interval":        "integer",
				"reconnect_on_failure": "boolean",
			},
		},
		{
			Name:        ProtocolMQTT,
			DisplayName: "MQTT",
			Description: "MQTT message publishing",
			Version:     "5.0",
			Supported:   true,
			DefaultPort: 1883,
			RequiresTLS: false,
			SupportsAuth: []AuthType{
				AuthNone, AuthBasic,
			},
			OptionsSchema: map[string]any{
				"topic":       "string",
				"qos":         "integer",
				"retain":      "boolean",
				"client_id":   "string",
				"clean_start": "boolean",
			},
		},
	}
}

// IsValidProtocol checks if a protocol is valid
func IsValidProtocol(p Protocol) bool {
	switch p {
	case ProtocolHTTP, ProtocolHTTPS, ProtocolGRPC, ProtocolGRPCS, ProtocolWebSocket, ProtocolMQTT:
		return true
	default:
		return false
	}
}

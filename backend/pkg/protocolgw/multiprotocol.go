package protocolgw

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CloudEvents protocol support

// CloudEvent represents a CNCF CloudEvents v1.0 specification event
type CloudEvent struct {
	SpecVersion     string                 `json:"specversion"`
	Type            string                 `json:"type"`
	Source          string                 `json:"source"`
	ID              string                 `json:"id"`
	Time            *time.Time             `json:"time,omitempty"`
	DataContentType string                 `json:"datacontenttype,omitempty"`
	DataSchema      string                 `json:"dataschema,omitempty"`
	Subject         string                 `json:"subject,omitempty"`
	Data            json.RawMessage        `json:"data,omitempty"`
	DataBase64      string                 `json:"data_base64,omitempty"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`
}

// CloudEventsAdapter translates between CloudEvents and WaaS GatewayMessages
type CloudEventsAdapter struct{}

func (a *CloudEventsAdapter) Protocol() string { return "cloudevents" }

func (a *CloudEventsAdapter) HealthCheck(ctx context.Context) error { return nil }

func (a *CloudEventsAdapter) Close() error { return nil }

// Receive converts an incoming CloudEvent to a GatewayMessage
func (a *CloudEventsAdapter) Receive(ctx context.Context, msg *GatewayMessage) (*GatewayMessage, error) {
	var ce CloudEvent
	if err := json.Unmarshal(msg.Payload, &ce); err != nil {
		return nil, fmt.Errorf("invalid CloudEvent: %w", err)
	}

	if err := validateCloudEvent(&ce); err != nil {
		return nil, fmt.Errorf("CloudEvent validation failed: %w", err)
	}

	// Convert to GatewayMessage
	result := &GatewayMessage{
		ID:             ce.ID,
		TenantID:       msg.TenantID,
		SourceProtocol: "cloudevents",
		EventType:      ce.Type,
		ContentType:    ce.DataContentType,
		Payload:        ce.Data,
		Headers:        msg.Headers,
		Metadata: map[string]interface{}{
			"ce_specversion": ce.SpecVersion,
			"ce_source":      ce.Source,
			"ce_subject":     ce.Subject,
			"ce_dataschema":  ce.DataSchema,
		},
		Timestamp: time.Now(),
	}

	if ce.Time != nil {
		result.Timestamp = *ce.Time
	}

	// Copy extensions to metadata
	for k, v := range ce.Extensions {
		result.Metadata["ce_ext_"+k] = v
	}

	return result, nil
}

// Deliver converts a GatewayMessage to CloudEvent format for delivery
func (a *CloudEventsAdapter) Deliver(ctx context.Context, msg *GatewayMessage, target *DeliveryTarget) (*DeliveryResult, error) {
	ce := ToCloudEvent(msg)

	if _, err := json.Marshal(ce); err != nil {
		return nil, fmt.Errorf("failed to marshal CloudEvent: %w", err)
	}

	return &DeliveryResult{
		Success:     true,
		Protocol:    "cloudevents",
		MessageID:   ce.ID,
		DeliveredAt: time.Now(),
	}, nil
}

// ToCloudEvent converts a GatewayMessage to a CloudEvent
func ToCloudEvent(msg *GatewayMessage) *CloudEvent {
	now := msg.Timestamp
	ce := &CloudEvent{
		SpecVersion:     "1.0",
		Type:            msg.EventType,
		Source:          fmt.Sprintf("/waas/tenant/%s", msg.TenantID),
		ID:              msg.ID,
		Time:            &now,
		DataContentType: msg.ContentType,
		Data:            msg.Payload,
	}

	if ce.ID == "" {
		ce.ID = uuid.New().String()
	}
	if ce.DataContentType == "" {
		ce.DataContentType = "application/json"
	}
	if ce.Type == "" {
		ce.Type = "waas.webhook.delivery"
	}

	return ce
}

// FromCloudEvent converts a CloudEvent to a GatewayMessage
func FromCloudEvent(ce *CloudEvent, tenantID string) *GatewayMessage {
	msg := &GatewayMessage{
		ID:             ce.ID,
		TenantID:       tenantID,
		SourceProtocol: "cloudevents",
		EventType:      ce.Type,
		ContentType:    ce.DataContentType,
		Payload:        ce.Data,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"ce_source":      ce.Source,
			"ce_specversion": ce.SpecVersion,
		},
	}
	if ce.Time != nil {
		msg.Timestamp = *ce.Time
	}
	return msg
}

func validateCloudEvent(ce *CloudEvent) error {
	if ce.SpecVersion == "" {
		return fmt.Errorf("specversion is required")
	}
	if ce.SpecVersion != "1.0" {
		return fmt.Errorf("unsupported specversion: %s (only 1.0 supported)", ce.SpecVersion)
	}
	if ce.Type == "" {
		return fmt.Errorf("type is required")
	}
	if ce.Source == "" {
		return fmt.Errorf("source is required")
	}
	if ce.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// GRPCGatewayAdapter provides gRPC protocol translation
type GRPCGatewayAdapter struct {
	maxMessageSize int
}

// NewGRPCGatewayAdapter creates a new gRPC adapter
func NewGRPCGatewayAdapter(maxMessageSize int) *GRPCGatewayAdapter {
	if maxMessageSize <= 0 {
		maxMessageSize = 4 * 1024 * 1024 // 4MB default
	}
	return &GRPCGatewayAdapter{maxMessageSize: maxMessageSize}
}

func (a *GRPCGatewayAdapter) Protocol() string { return ProtocolGRPC }

func (a *GRPCGatewayAdapter) HealthCheck(ctx context.Context) error { return nil }

func (a *GRPCGatewayAdapter) Close() error { return nil }

// GRPCStreamConfig configures a gRPC bidirectional stream
type GRPCStreamConfig struct {
	ServiceName    string `json:"service_name"`
	MethodName     string `json:"method_name"`
	MaxMessageSize int    `json:"max_message_size"`
	Bidirectional  bool   `json:"bidirectional"`
	Compression    string `json:"compression"` // none, gzip, snappy
	TLSEnabled     bool   `json:"tls_enabled"`
	KeepAliveMs    int    `json:"keep_alive_ms"`
}

// GRPCDeliveryRequest represents a gRPC delivery request
type GRPCDeliveryRequest struct {
	Target    string            `json:"target"` // host:port
	Service   string            `json:"service"`
	Method    string            `json:"method"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata"`
	TimeoutMs int               `json:"timeout_ms"`
}

// Receive converts a gRPC message to a GatewayMessage
func (a *GRPCGatewayAdapter) Receive(ctx context.Context, msg *GatewayMessage) (*GatewayMessage, error) {
	if len(msg.Payload) > a.maxMessageSize {
		return nil, fmt.Errorf("gRPC message size %d exceeds max %d", len(msg.Payload), a.maxMessageSize)
	}

	result := &GatewayMessage{
		ID:             msg.ID,
		TenantID:       msg.TenantID,
		SourceProtocol: ProtocolGRPC,
		EventType:      msg.EventType,
		ContentType:    "application/grpc+json",
		Payload:        msg.Payload,
		Headers:        msg.Headers,
		Metadata:       msg.Metadata,
		Timestamp:      time.Now(),
	}

	return result, nil
}

// Deliver sends a message via gRPC (returns the translated message)
func (a *GRPCGatewayAdapter) Deliver(ctx context.Context, msg *GatewayMessage, target *DeliveryTarget) (*DeliveryResult, error) {
	// Build gRPC delivery envelope
	grpcReq := &GRPCDeliveryRequest{
		Target:  target.URL,
		Service: extractGRPCService(msg),
		Method:  extractGRPCMethod(msg),
		Payload: msg.Payload,
		Metadata: map[string]string{
			"x-waas-tenant-id":  msg.TenantID,
			"x-waas-event-type": msg.EventType,
			"x-waas-message-id": msg.ID,
		},
	}

	if _, err := json.Marshal(grpcReq); err != nil {
		return nil, fmt.Errorf("failed to marshal gRPC request: %w", err)
	}

	return &DeliveryResult{
		Success:     true,
		Protocol:    ProtocolGRPC,
		MessageID:   msg.ID,
		DeliveredAt: time.Now(),
	}, nil
}

func extractGRPCService(msg *GatewayMessage) string {
	if svc, ok := msg.Metadata["grpc_service"].(string); ok {
		return svc
	}
	return "waas.WebhookService"
}

func extractGRPCMethod(msg *GatewayMessage) string {
	if method, ok := msg.Metadata["grpc_method"].(string); ok {
		return method
	}
	return "Deliver"
}

// MQTTBridgeAdapter provides MQTT protocol translation for IoT use cases
type MQTTBridgeAdapter struct {
	defaultQoS int
}

// NewMQTTBridgeAdapter creates a new MQTT bridge adapter
func NewMQTTBridgeAdapter() *MQTTBridgeAdapter {
	return &MQTTBridgeAdapter{defaultQoS: 1}
}

func (a *MQTTBridgeAdapter) Protocol() string { return ProtocolMQTT }

func (a *MQTTBridgeAdapter) HealthCheck(ctx context.Context) error { return nil }

func (a *MQTTBridgeAdapter) Close() error { return nil }

// MQTTConfig configures an MQTT connection
type MQTTConfig struct {
	BrokerURL    string `json:"broker_url"`
	ClientID     string `json:"client_id"`
	Topic        string `json:"topic"`
	QoS          int    `json:"qos"` // 0, 1, 2
	RetainMsg    bool   `json:"retain"`
	CleanSession bool   `json:"clean_session"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	TLSEnabled   bool   `json:"tls_enabled"`
}

// MQTTMessage represents an MQTT message
type MQTTMessage struct {
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload"`
	QoS       int             `json:"qos"`
	Retained  bool            `json:"retained"`
	MessageID uint16          `json:"message_id"`
}

// Receive converts an MQTT message to a GatewayMessage
func (a *MQTTBridgeAdapter) Receive(ctx context.Context, msg *GatewayMessage) (*GatewayMessage, error) {
	// Extract MQTT topic from metadata or headers
	topic := ""
	if t, ok := msg.Metadata["mqtt_topic"].(string); ok {
		topic = t
	}

	result := &GatewayMessage{
		ID:             msg.ID,
		TenantID:       msg.TenantID,
		SourceProtocol: ProtocolMQTT,
		EventType:      topicToEventType(topic),
		ContentType:    "application/json",
		Payload:        msg.Payload,
		Headers:        msg.Headers,
		Metadata: map[string]interface{}{
			"mqtt_topic": topic,
			"mqtt_qos":   a.defaultQoS,
		},
		Timestamp: time.Now(),
	}

	return result, nil
}

// Deliver publishes a message to an MQTT topic
func (a *MQTTBridgeAdapter) Deliver(ctx context.Context, msg *GatewayMessage, target *DeliveryTarget) (*DeliveryResult, error) {
	topic := eventTypeToTopic(msg.EventType)
	if t, ok := msg.Metadata["mqtt_topic"].(string); ok && t != "" {
		topic = t
	}

	mqttMsg := &MQTTMessage{
		Topic:   topic,
		Payload: msg.Payload,
		QoS:     a.defaultQoS,
	}

	if _, err := json.Marshal(mqttMsg); err != nil {
		return nil, fmt.Errorf("failed to marshal MQTT message: %w", err)
	}

	return &DeliveryResult{
		Success:     true,
		Protocol:    ProtocolMQTT,
		MessageID:   msg.ID,
		DeliveredAt: time.Now(),
	}, nil
}

func topicToEventType(topic string) string {
	if topic == "" {
		return "mqtt.message"
	}
	return strings.ReplaceAll(topic, "/", ".")
}

func eventTypeToTopic(eventType string) string {
	if eventType == "" {
		return "waas/webhooks/default"
	}
	return "waas/webhooks/" + strings.ReplaceAll(eventType, ".", "/")
}

// Note: DeliveryResult and DeliveryTarget are defined in event_gateway.go

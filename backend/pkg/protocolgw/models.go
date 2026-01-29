package protocolgw

import "time"

// Protocol constants
const (
	ProtocolHTTP    = "http"
	ProtocolGRPC    = "grpc"
	ProtocolMQTT    = "mqtt"
	ProtocolKafka   = "kafka"
	ProtocolKinesis     = "kinesis"
	ProtocolWebSocket   = "websocket"
	ProtocolSNS         = "sns"
	ProtocolEventBridge = "eventbridge"
	ProtocolPubSub      = "pubsub"
)

// OrderingGuarantee constants
const (
	OrderingNone     = "none"
	OrderingFIFO     = "fifo"
	OrderingKeyBased = "key_based"
)

// DeliveryGuarantee constants
const (
	DeliveryAtMostOnce  = "at_most_once"
	DeliveryAtLeastOnce = "at_least_once"
	DeliveryExactlyOnce = "exactly_once"
)

// MessageStatus constants
const (
	MessageStatusPending    = "pending"
	MessageStatusTranslated = "translated"
	MessageStatusDelivered  = "delivered"
	MessageStatusFailed     = "failed"
)

// ProtocolRoute defines a translation route between two protocols
type ProtocolRoute struct {
	ID                 string    `json:"id" db:"id"`
	TenantID           string    `json:"tenant_id" db:"tenant_id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	SourceProtocol     string    `json:"source_protocol" db:"source_protocol"`
	SourceConfig       string    `json:"source_config" db:"source_config"`
	DestProtocol       string    `json:"dest_protocol" db:"dest_protocol"`
	DestConfig         string    `json:"dest_config" db:"dest_config"`
	TransformRule      string    `json:"transform_rule" db:"transform_rule"`
	OrderingGuarantee  string    `json:"ordering_guarantee" db:"ordering_guarantee"`
	DeliveryGuarantee  string    `json:"delivery_guarantee" db:"delivery_guarantee"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// ProtocolMessage represents a message being translated between protocols
type ProtocolMessage struct {
	ID                string            `json:"id" db:"id"`
	TenantID          string            `json:"tenant_id" db:"tenant_id"`
	RouteID           string            `json:"route_id" db:"route_id"`
	SourceProtocol    string            `json:"source_protocol" db:"source_protocol"`
	DestProtocol      string            `json:"dest_protocol" db:"dest_protocol"`
	Payload           string            `json:"payload" db:"payload"`
	Headers           map[string]string `json:"headers"`
	PartitionKey      string            `json:"partition_key" db:"partition_key"`
	Status            string            `json:"status" db:"status"`
	TranslatedPayload string           `json:"translated_payload" db:"translated_payload"`
	ErrorMessage      string            `json:"error_message,omitempty" db:"error_message"`
	LatencyMs         int64             `json:"latency_ms" db:"latency_ms"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
}

// TranslationResult holds the outcome of a protocol translation
type TranslationResult struct {
	RouteID        string `json:"route_id"`
	SourceProtocol string `json:"source_protocol"`
	DestProtocol   string `json:"dest_protocol"`
	OriginalSize   int    `json:"original_size"`
	TranslatedSize int    `json:"translated_size"`
	LatencyMs      int64  `json:"latency_ms"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}

// ProtocolStats provides statistics for a protocol route
type ProtocolStats struct {
	RouteID       string  `json:"route_id"`
	RouteName     string  `json:"route_name"`
	TotalMessages int64   `json:"total_messages"`
	SuccessCount  int64   `json:"success_count"`
	FailureCount  int64   `json:"failure_count"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	LastMessageAt string  `json:"last_message_at,omitempty"`
}

// CreateRouteRequest is the request DTO for creating a protocol route
type CreateRouteRequest struct {
	Name              string `json:"name" binding:"required,min=1,max=255"`
	Description       string `json:"description" binding:"max=1024"`
	SourceProtocol    string `json:"source_protocol" binding:"required"`
	SourceConfig      string `json:"source_config"`
	DestProtocol      string `json:"dest_protocol" binding:"required"`
	DestConfig        string `json:"dest_config"`
	TransformRule     string `json:"transform_rule"`
	OrderingGuarantee string `json:"ordering_guarantee"`
	DeliveryGuarantee string `json:"delivery_guarantee"`
}

// TranslateMessageRequest is the request DTO for translating a message
type TranslateMessageRequest struct {
	RouteID      string            `json:"route_id" binding:"required"`
	Payload      string            `json:"payload" binding:"required"`
	Headers      map[string]string `json:"headers"`
	PartitionKey string            `json:"partition_key"`
}

// WebSocketConfig defines WebSocket push delivery configuration
type WebSocketConfig struct {
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers,omitempty"`
	PingIntervalSec int               `json:"ping_interval_sec,omitempty"`
	ReconnectSec    int               `json:"reconnect_sec,omitempty"`
}

// SNSConfig defines AWS SNS delivery configuration
type SNSConfig struct {
	TopicARN   string            `json:"topic_arn"`
	Region     string            `json:"region"`
	AccessKey  string            `json:"access_key,omitempty"`
	SecretKey  string            `json:"-"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// EventBridgeConfig defines AWS EventBridge delivery configuration
type EventBridgeConfig struct {
	EventBusName string `json:"event_bus_name"`
	Region       string `json:"region"`
	Source       string `json:"source"`
	DetailType   string `json:"detail_type"`
	AccessKey    string `json:"access_key,omitempty"`
	SecretKey    string `json:"-"`
}

// PubSubConfig defines GCP Pub/Sub delivery configuration
type PubSubConfig struct {
	ProjectID       string            `json:"project_id"`
	TopicID         string            `json:"topic_id"`
	CredentialsJSON string            `json:"-"`
	Attributes      map[string]string `json:"attributes,omitempty"`
	OrderingKey     string            `json:"ordering_key,omitempty"`
}

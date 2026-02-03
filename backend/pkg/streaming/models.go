// Package streaming provides bi-directional integration with event streaming platforms
package streaming

import (
	"encoding/json"
	"sync"
	"time"
)

// StreamType represents supported streaming platforms
type StreamType string

const (
	StreamTypeKafka       StreamType = "kafka"
	StreamTypeKinesis     StreamType = "kinesis"
	StreamTypePulsar      StreamType = "pulsar"
	StreamTypeEventBridge StreamType = "eventbridge"
	StreamTypeRedis       StreamType = "redis_streams"
	StreamTypeNATS        StreamType = "nats"
	StreamTypeRabbitMQ    StreamType = "rabbitmq"
	StreamTypeSQS         StreamType = "sqs"
	StreamTypeSNS         StreamType = "sns"
)

// Direction indicates whether the bridge is inbound or outbound
type Direction string

const (
	DirectionInbound  Direction = "inbound"  // Stream -> Webhook
	DirectionOutbound Direction = "outbound" // Webhook -> Stream
	DirectionBoth     Direction = "both"
)

// BridgeStatus represents the lifecycle status of a streaming bridge
type BridgeStatus string

const (
	BridgeStatusActive   BridgeStatus = "active"
	BridgeStatusPaused   BridgeStatus = "paused"
	BridgeStatusError    BridgeStatus = "error"
	BridgeStatusCreating BridgeStatus = "creating"
	BridgeStatusDeleting BridgeStatus = "deleting"
)

// SchemaFormat represents supported schema formats
type SchemaFormat string

const (
	SchemaFormatJSON     SchemaFormat = "json"
	SchemaFormatAvro     SchemaFormat = "avro"
	SchemaFormatProtobuf SchemaFormat = "protobuf"
	SchemaFormatNone     SchemaFormat = "none"
)

// StreamingBridge represents a connection between webhooks and a streaming platform
type StreamingBridge struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	StreamType      StreamType        `json:"stream_type"`
	Direction       Direction         `json:"direction"`
	Status          BridgeStatus      `json:"status"`
	Config          *BridgeConfig     `json:"config"`
	SchemaConfig    *SchemaConfig     `json:"schema_config,omitempty"`
	TransformScript string            `json:"transform_script,omitempty"`
	FilterRules     []FilterRule      `json:"filter_rules,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	LastEventAt     *time.Time        `json:"last_event_at,omitempty"`
	EventsProcessed int64             `json:"events_processed"`
	EventsFailed    int64             `json:"events_failed"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// BridgeConfig contains platform-specific configuration
type BridgeConfig struct {
	// Kafka-specific
	KafkaConfig *KafkaConfig `json:"kafka,omitempty"`
	// Kinesis-specific
	KinesisConfig *KinesisConfig `json:"kinesis,omitempty"`
	// Pulsar-specific
	PulsarConfig *PulsarConfig `json:"pulsar,omitempty"`
	// EventBridge-specific
	EventBridgeConfig *EventBridgeConfig `json:"eventbridge,omitempty"`

	// NATS-specific
	NATSConfig *NATSConfig `json:"nats,omitempty"`
	// RabbitMQ-specific
	RabbitMQConfig *RabbitMQConfig `json:"rabbitmq,omitempty"`
	// SQS-specific
	SQSConfig *SQSConfig `json:"sqs,omitempty"`
	// SNS-specific
	SNSConfig *SNSConfig `json:"sns,omitempty"`

	// Common settings
	BatchSize       int    `json:"batch_size,omitempty"`
	FlushIntervalMs int    `json:"flush_interval_ms,omitempty"`
	MaxRetries      int    `json:"max_retries,omitempty"`
	RetryBackoffMs  int    `json:"retry_backoff_ms,omitempty"`
	Compression     string `json:"compression,omitempty"`
	AckMode         string `json:"ack_mode,omitempty"`
}

// KafkaConfig contains Kafka-specific configuration
type KafkaConfig struct {
	Brokers           []string          `json:"brokers"`
	Topic             string            `json:"topic"`
	ConsumerGroup     string            `json:"consumer_group,omitempty"`
	SecurityProtocol  string            `json:"security_protocol,omitempty"`
	SASLMechanism     string            `json:"sasl_mechanism,omitempty"`
	SASLUsername      string            `json:"sasl_username,omitempty"`
	SASLPassword      string            `json:"-"` // Never serialize
	SSLEnabled        bool              `json:"ssl_enabled"`
	SSLCAPath         string            `json:"ssl_ca_path,omitempty"`
	SSLCertPath       string            `json:"ssl_cert_path,omitempty"`
	SSLKeyPath        string            `json:"ssl_key_path,omitempty"`
	Partitioner       string            `json:"partitioner,omitempty"`
	RequiredAcks      int               `json:"required_acks,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	AutoOffsetReset   string            `json:"auto_offset_reset,omitempty"`
	SessionTimeoutMs  int               `json:"session_timeout_ms,omitempty"`
	HeartbeatInterval int               `json:"heartbeat_interval_ms,omitempty"`
}

// KinesisConfig contains AWS Kinesis-specific configuration
type KinesisConfig struct {
	StreamName      string `json:"stream_name"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"-"` // Never serialize
	RoleARN         string `json:"role_arn,omitempty"`
	PartitionKey    string `json:"partition_key,omitempty"`
	ShardIterator   string `json:"shard_iterator_type,omitempty"`
	StartingPos     string `json:"starting_position,omitempty"`
}

// PulsarConfig contains Apache Pulsar-specific configuration
type PulsarConfig struct {
	ServiceURL       string `json:"service_url"`
	Topic            string `json:"topic"`
	Subscription     string `json:"subscription,omitempty"`
	SubscriptionType string `json:"subscription_type,omitempty"`
	AuthToken        string `json:"-"` // Never serialize
	TLSEnabled       bool   `json:"tls_enabled"`
	TLSTrustCertPath string `json:"tls_trust_cert_path,omitempty"`
}

// EventBridgeConfig contains AWS EventBridge-specific configuration
type EventBridgeConfig struct {
	EventBusName    string `json:"event_bus_name"`
	Region          string `json:"region"`
	Source          string `json:"source"`
	DetailType      string `json:"detail_type"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"-"` // Never serialize
	RoleARN         string `json:"role_arn,omitempty"`
}

// NATSConfig contains NATS-specific configuration
type NATSConfig struct {
	URL           string `json:"url"`
	Subject       string `json:"subject"`
	QueueGroup    string `json:"queue_group,omitempty"`
	ClusterID     string `json:"cluster_id,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
	Token         string `json:"-"`
	TLSEnabled    bool   `json:"tls_enabled"`
	TLSCertPath   string `json:"tls_cert_path,omitempty"`
	TLSKeyPath    string `json:"tls_key_path,omitempty"`
	JetStream     bool   `json:"jetstream"`
	StreamName    string `json:"stream_name,omitempty"`
	ConsumerName  string `json:"consumer_name,omitempty"`
	DeliverPolicy string `json:"deliver_policy,omitempty"`
}

// RabbitMQConfig contains RabbitMQ-specific configuration
type RabbitMQConfig struct {
	URL           string `json:"url"`
	Exchange      string `json:"exchange"`
	ExchangeType  string `json:"exchange_type,omitempty"` // direct, fanout, topic, headers
	Queue         string `json:"queue,omitempty"`
	RoutingKey    string `json:"routing_key,omitempty"`
	ConsumerTag   string `json:"consumer_tag,omitempty"`
	Durable       bool   `json:"durable"`
	AutoDelete    bool   `json:"auto_delete"`
	Exclusive     bool   `json:"exclusive"`
	NoWait        bool   `json:"no_wait"`
	PrefetchCount int    `json:"prefetch_count,omitempty"`
	PrefetchSize  int    `json:"prefetch_size,omitempty"`
	TLSEnabled    bool   `json:"tls_enabled"`
	VHost         string `json:"vhost,omitempty"`
}

// SQSConfig contains AWS SQS-specific configuration
type SQSConfig struct {
	QueueURL          string `json:"queue_url"`
	Region            string `json:"region"`
	AccessKeyID       string `json:"access_key_id,omitempty"`
	SecretAccessKey   string `json:"-"`
	RoleARN           string `json:"role_arn,omitempty"`
	MaxMessages       int    `json:"max_messages,omitempty"`
	WaitTimeSeconds   int    `json:"wait_time_seconds,omitempty"`
	VisibilityTimeout int    `json:"visibility_timeout,omitempty"`
	FIFOQueue         bool   `json:"fifo_queue"`
	MessageGroupID    string `json:"message_group_id,omitempty"`
	DeduplicationID   string `json:"deduplication_id,omitempty"`
}

// SNSConfig contains AWS SNS-specific configuration
type SNSConfig struct {
	TopicARN        string            `json:"topic_arn"`
	Region          string            `json:"region"`
	AccessKeyID     string            `json:"access_key_id,omitempty"`
	SecretAccessKey string            `json:"-"`
	RoleARN         string            `json:"role_arn,omitempty"`
	MessageAttrs    map[string]string `json:"message_attributes,omitempty"`
	FIFOTopic       bool              `json:"fifo_topic"`
	MessageGroupID  string            `json:"message_group_id,omitempty"`
}

// SchemaConfig contains schema registry configuration
type SchemaConfig struct {
	Format         SchemaFormat `json:"format"`
	RegistryURL    string       `json:"registry_url,omitempty"`
	RegistryType   string       `json:"registry_type,omitempty"` // confluent, aws_glue, apicurio
	SubjectName    string       `json:"subject_name,omitempty"`
	SchemaID       int          `json:"schema_id,omitempty"`
	SchemaVersion  int          `json:"schema_version,omitempty"`
	SchemaContent  string       `json:"schema_content,omitempty"`
	AutoRegister   bool         `json:"auto_register"`
	CompatMode     string       `json:"compat_mode,omitempty"` // backward, forward, full, none
	ValidationMode string       `json:"validation_mode,omitempty"`
}

// FilterRule defines event filtering logic
type FilterRule struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, neq, contains, regex, exists, gt, lt
	Value    interface{} `json:"value"`
	Negate   bool        `json:"negate,omitempty"`
}

// StreamEvent represents an event from/to a streaming platform
type StreamEvent struct {
	ID           string            `json:"id"`
	BridgeID     string            `json:"bridge_id"`
	TenantID     string            `json:"tenant_id"`
	Key          string            `json:"key,omitempty"`
	Value        json.RawMessage   `json:"value"`
	Headers      map[string]string `json:"headers,omitempty"`
	Partition    int               `json:"partition,omitempty"`
	Offset       int64             `json:"offset,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	SchemaID     int               `json:"schema_id,omitempty"`
	SourceTopic  string            `json:"source_topic,omitempty"`
	ProcessedAt  *time.Time        `json:"processed_at,omitempty"`
	DeliveryID   string            `json:"delivery_id,omitempty"`
	Status       string            `json:"status"`
	ErrorMessage string            `json:"error_message,omitempty"`
	RetryCount   int               `json:"retry_count"`
}

// BridgeMetrics contains operational metrics for a bridge
type BridgeMetrics struct {
	BridgeID           string    `json:"bridge_id"`
	TenantID           string    `json:"tenant_id"`
	EventsIn           int64     `json:"events_in"`
	EventsOut          int64     `json:"events_out"`
	EventsFailed       int64     `json:"events_failed"`
	BytesIn            int64     `json:"bytes_in"`
	BytesOut           int64     `json:"bytes_out"`
	AvgLatencyMs       float64   `json:"avg_latency_ms"`
	P99LatencyMs       float64   `json:"p99_latency_ms"`
	ErrorRate          float64   `json:"error_rate"`
	CurrentLag         int64     `json:"current_lag,omitempty"`
	LastEventTimestamp time.Time `json:"last_event_timestamp"`
	Period             string    `json:"period"`
	CollectedAt        time.Time `json:"collected_at"`
}

// CreateBridgeRequest represents a request to create a streaming bridge
type CreateBridgeRequest struct {
	Name            string            `json:"name" binding:"required"`
	Description     string            `json:"description,omitempty"`
	StreamType      StreamType        `json:"stream_type" binding:"required"`
	Direction       Direction         `json:"direction" binding:"required"`
	Config          *BridgeConfig     `json:"config" binding:"required"`
	SchemaConfig    *SchemaConfig     `json:"schema_config,omitempty"`
	TransformScript string            `json:"transform_script,omitempty"`
	FilterRules     []FilterRule      `json:"filter_rules,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	// For outbound bridges: which webhooks to forward
	WebhookEndpointIDs []string `json:"webhook_endpoint_ids,omitempty"`
	// For inbound bridges: target webhook endpoint
	TargetEndpointID string `json:"target_endpoint_id,omitempty"`
}

// UpdateBridgeRequest represents a request to update a streaming bridge
type UpdateBridgeRequest struct {
	Name            *string           `json:"name,omitempty"`
	Description     *string           `json:"description,omitempty"`
	Status          *BridgeStatus     `json:"status,omitempty"`
	Config          *BridgeConfig     `json:"config,omitempty"`
	SchemaConfig    *SchemaConfig     `json:"schema_config,omitempty"`
	TransformScript *string           `json:"transform_script,omitempty"`
	FilterRules     []FilterRule      `json:"filter_rules,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// BridgeResponse represents the API response for a streaming bridge
type BridgeResponse struct {
	*StreamingBridge
	Metrics *BridgeMetrics `json:"metrics,omitempty"`
}

// ListBridgesResponse represents paginated bridge list response
type ListBridgesResponse struct {
	Bridges    []BridgeResponse `json:"bridges"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// ProducerMetrics tracks metrics for message producers
type ProducerMetrics struct {
	MessagesSent   int64     `json:"messages_sent"`
	MessagesAcked  int64     `json:"messages_acked"`
	MessagesFailed int64     `json:"messages_failed"`
	BytesSent      int64     `json:"bytes_sent"`
	LastSendTime   time.Time `json:"last_send_time"`
	LastAckTime    time.Time `json:"last_ack_time"`
	LastErrorTime  time.Time `json:"last_error_time,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	mu             sync.RWMutex
}

// ConsumerMetrics tracks metrics for message consumers
type ConsumerMetrics struct {
	MessagesReceived  int64     `json:"messages_received"`
	MessagesProcessed int64     `json:"messages_processed"`
	MessagesFailed    int64     `json:"messages_failed"`
	LastReceiveTime   time.Time `json:"last_receive_time"`
	LastProcessTime   time.Time `json:"last_process_time"`
	LastErrorTime     time.Time `json:"last_error_time,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
	mu                sync.RWMutex
}

// Message represents a generic message for streaming platforms
type Message struct {
	ID        string            `json:"id"`
	Key       string            `json:"key,omitempty"`
	Value     []byte            `json:"value"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

package protocols

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

// KafkaDeliverer implements Kafka webhook delivery
type KafkaDeliverer struct {
	producers map[string]sarama.SyncProducer
}

// KafkaOptions represents Kafka-specific delivery options
type KafkaOptions struct {
	Brokers      []string `json:"brokers"`
	Topic        string   `json:"topic"`
	PartitionKey string   `json:"partition_key,omitempty"`
	Compression  string   `json:"compression,omitempty"` // none, gzip, snappy, lz4, zstd
	AckMode      string   `json:"ack_mode,omitempty"`    // none, leader, all
	Idempotent   bool     `json:"idempotent"`
	MaxRetries   int      `json:"max_retries,omitempty"`
}

// NewKafkaDeliverer creates a new Kafka deliverer
func NewKafkaDeliverer() *KafkaDeliverer {
	return &KafkaDeliverer{
		producers: make(map[string]sarama.SyncProducer),
	}
}

// Protocol returns the protocol
func (d *KafkaDeliverer) Protocol() Protocol {
	return ProtocolKafka
}

// Validate validates the delivery config
func (d *KafkaDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("target broker is required")
	}

	opts := parseKafkaOptions(config.Options)
	if opts.Topic == "" {
		return fmt.Errorf("kafka topic is required")
	}
	if len(opts.Brokers) == 0 && config.Target == "" {
		return fmt.Errorf("at least one broker is required")
	}

	return nil
}

// Deliver performs the Kafka delivery
func (d *KafkaDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{
		ProtocolInfo: make(map[string]any),
	}

	opts := parseKafkaOptions(config.Options)

	producer, err := d.getProducer(config, opts)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	// Build message
	msg := &sarama.ProducerMessage{
		Topic: opts.Topic,
		Value: sarama.ByteEncoder(request.Payload),
		Headers: []sarama.RecordHeader{
			{Key: []byte("X-Webhook-ID"), Value: []byte(request.WebhookID)},
			{Key: []byte("X-Delivery-ID"), Value: []byte(request.ID)},
			{Key: []byte("X-Delivery-Attempt"), Value: []byte(fmt.Sprintf("%d", request.AttemptNumber))},
		},
	}

	if request.ContentType != "" {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{
			Key: []byte("Content-Type"), Value: []byte(request.ContentType),
		})
	}

	// Set partition key
	if opts.PartitionKey != "" {
		msg.Key = sarama.StringEncoder(opts.PartitionKey)
	} else if request.WebhookID != "" {
		msg.Key = sarama.StringEncoder(request.WebhookID)
	}

	// Add custom headers from request and config
	for k, v := range request.Headers {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}
	for k, v := range config.Headers {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}

	// Send message
	partition, offset, err := producer.SendMessage(msg)
	response.Duration = time.Since(start)

	if err != nil {
		response.Error = err.Error()
		response.ErrorType = categorizeKafkaError(err)
		return response, nil
	}

	response.Success = true
	response.ProtocolInfo["topic"] = opts.Topic
	response.ProtocolInfo["partition"] = partition
	response.ProtocolInfo["offset"] = offset
	response.ProtocolInfo["brokers"] = config.Target

	return response, nil
}

// Close closes the deliverer and all producers
func (d *KafkaDeliverer) Close() error {
	var errs []string
	for key, producer := range d.producers {
		if err := producer.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
		}
	}
	d.producers = make(map[string]sarama.SyncProducer)
	if len(errs) > 0 {
		return fmt.Errorf("errors closing kafka producers: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (d *KafkaDeliverer) getProducer(config *DeliveryConfig, opts KafkaOptions) (sarama.SyncProducer, error) {
	key := config.Target

	if producer, ok := d.producers[key]; ok {
		return producer, nil
	}

	brokers := opts.Brokers
	if len(brokers) == 0 {
		brokers = strings.Split(config.Target, ",")
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	// Configure ack mode (at-least-once by default)
	switch opts.AckMode {
	case "none":
		saramaConfig.Producer.RequiredAcks = sarama.NoResponse
	case "leader":
		saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	default:
		saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	}

	// Configure compression
	switch opts.Compression {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}

	// Idempotent producer settings
	if opts.Idempotent {
		saramaConfig.Producer.Idempotent = true
		saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
		saramaConfig.Net.MaxOpenRequests = 1
	}

	if opts.MaxRetries > 0 {
		saramaConfig.Producer.Retry.Max = opts.MaxRetries
	} else {
		saramaConfig.Producer.Retry.Max = 3
	}

	// TLS config
	if config.TLS != nil && config.TLS.Enabled {
		saramaConfig.Net.TLS.Enable = true
		saramaConfig.Net.TLS.Config = buildTLSConfig(config.TLS)
	}

	// SASL auth
	if config.Auth != nil {
		switch config.Auth.Type {
		case AuthBasic:
			saramaConfig.Net.SASL.Enable = true
			saramaConfig.Net.SASL.User = config.Auth.Credentials["username"]
			saramaConfig.Net.SASL.Password = config.Auth.Credentials["password"]
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("kafka producer creation failed: %w", err)
	}

	d.producers[key] = producer
	return producer, nil
}

func categorizeKafkaError(err error) DeliveryErrorType {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorTypeTimeout
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "broker") {
		return ErrorTypeConnection
	}
	if strings.Contains(errStr, "authorization") || strings.Contains(errStr, "authentication") {
		return ErrorTypeAuth
	}

	return ErrorTypeServer
}

func parseKafkaOptions(opts map[string]interface{}) KafkaOptions {
	options := KafkaOptions{
		AckMode:    "all",
		MaxRetries: 3,
	}

	if opts == nil {
		return options
	}

	if topic, ok := opts["topic"].(string); ok {
		options.Topic = topic
	}
	if key, ok := opts["partition_key"].(string); ok {
		options.PartitionKey = key
	}
	if compression, ok := opts["compression"].(string); ok {
		options.Compression = compression
	}
	if ackMode, ok := opts["ack_mode"].(string); ok {
		options.AckMode = ackMode
	}
	if idempotent, ok := opts["idempotent"].(bool); ok {
		options.Idempotent = idempotent
	}
	if maxRetries, ok := opts["max_retries"].(float64); ok {
		options.MaxRetries = int(maxRetries)
	}
	if brokers, ok := opts["brokers"].([]interface{}); ok {
		for _, b := range brokers {
			if s, ok := b.(string); ok {
				options.Brokers = append(options.Brokers, s)
			}
		}
	}

	return options
}

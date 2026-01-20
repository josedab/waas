package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// SaramaKafkaProducer implements a production-ready Kafka producer using Sarama
type SaramaKafkaProducer struct {
	producer sarama.AsyncProducer
	config   *SaramaKafkaConfig
	mu       sync.RWMutex
	closed   bool
	metrics  *ProducerMetrics
}

// SaramaKafkaConfig holds Sarama Kafka-specific configuration
type SaramaKafkaConfig struct {
	Brokers           []string
	ClientID          string
	RequiredAcks      sarama.RequiredAcks
	Compression       sarama.CompressionCodec
	MaxMessageBytes   int
	FlushFrequency    time.Duration
	FlushMessages     int
	RetryMax          int
	RetryBackoff      time.Duration
	Idempotent        bool
	EnableTLS         bool
	TLSCertFile       string
	TLSKeyFile        string
	TLSCAFile         string
	SASLEnabled       bool
	SASLMechanism     string
	SASLUsername      string
	SASLPassword      string
}

// DefaultSaramaKafkaConfig returns default Sarama Kafka configuration
func DefaultSaramaKafkaConfig() *SaramaKafkaConfig {
	return &SaramaKafkaConfig{
		Brokers:         []string{"localhost:9092"},
		ClientID:        "waas-streaming",
		RequiredAcks:    sarama.WaitForAll,
		Compression:     sarama.CompressionSnappy,
		MaxMessageBytes: 1024 * 1024, // 1MB
		FlushFrequency:  100 * time.Millisecond,
		FlushMessages:   100,
		RetryMax:        3,
		RetryBackoff:    100 * time.Millisecond,
		Idempotent:      true,
	}
}

// NewSaramaKafkaProducer creates a new Kafka producer
func NewSaramaKafkaProducer(config *SaramaKafkaConfig) (*SaramaKafkaProducer, error) {
	if config == nil {
		config = DefaultSaramaKafkaConfig()
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID
	saramaConfig.Producer.RequiredAcks = config.RequiredAcks
	saramaConfig.Producer.Compression = config.Compression
	saramaConfig.Producer.MaxMessageBytes = config.MaxMessageBytes
	saramaConfig.Producer.Flush.Frequency = config.FlushFrequency
	saramaConfig.Producer.Flush.Messages = config.FlushMessages
	saramaConfig.Producer.Retry.Max = config.RetryMax
	saramaConfig.Producer.Retry.Backoff = config.RetryBackoff
	saramaConfig.Producer.Idempotent = config.Idempotent
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	// TLS configuration
	if config.EnableTLS {
		saramaConfig.Net.TLS.Enable = true
		// Add TLS config loading here if needed
	}

	// SASL configuration
	if config.SASLEnabled {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(config.SASLMechanism)
		saramaConfig.Net.SASL.User = config.SASLUsername
		saramaConfig.Net.SASL.Password = config.SASLPassword
	}

	producer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	p := &SaramaKafkaProducer{
		producer: producer,
		config:   config,
		metrics:  &ProducerMetrics{},
	}

	// Start goroutines to handle successes and errors
	go p.handleSuccesses()
	go p.handleErrors()

	return p, nil
}

// Send sends a message to Kafka
func (p *SaramaKafkaProducer) Send(ctx context.Context, topic string, message *Message) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.New("producer is closed")
	}
	p.mu.RUnlock()

	// Serialize message
	value, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build Kafka message
	kafkaMsg := &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(value),
		Timestamp: time.Now(),
	}

	// Set key if provided
	if message.Key != "" {
		kafkaMsg.Key = sarama.StringEncoder(message.Key)
	}

	// Add headers
	for key, val := range message.Headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: []byte(val),
		})
	}

	// Send message
	select {
	case p.producer.Input() <- kafkaMsg:
		p.metrics.mu.Lock()
		p.metrics.MessagesSent++
		p.metrics.BytesSent += int64(len(value))
		p.metrics.LastSendTime = time.Now()
		p.metrics.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendBatch sends multiple messages to Kafka
func (p *SaramaKafkaProducer) SendBatch(ctx context.Context, topic string, messages []*Message) error {
	for _, msg := range messages {
		if err := p.Send(ctx, topic, msg); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the producer
func (p *SaramaKafkaProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	return p.producer.Close()
}

// handleSuccesses processes successful message sends
func (p *SaramaKafkaProducer) handleSuccesses() {
	for range p.producer.Successes() {
		p.metrics.mu.Lock()
		p.metrics.MessagesAcked++
		p.metrics.LastAckTime = time.Now()
		p.metrics.mu.Unlock()
	}
}

// handleErrors processes message send errors
func (p *SaramaKafkaProducer) handleErrors() {
	for err := range p.producer.Errors() {
		p.metrics.mu.Lock()
		p.metrics.MessagesFailed++
		p.metrics.LastErrorTime = time.Now()
		if err.Err != nil {
			p.metrics.LastError = err.Err.Error()
		}
		p.metrics.mu.Unlock()
	}
}

// GetMetrics returns current producer metrics
func (p *SaramaKafkaProducer) GetMetrics() ProducerMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// SaramaKafkaConsumer implements a production-ready Kafka consumer using Sarama
type SaramaKafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	config        *SaramaKafkaConsumerConfig
	handler       func(ctx context.Context, topic string, event *StreamEvent) error
	mu            sync.RWMutex
	closed        bool
	metrics       *ConsumerMetrics
	cancelFunc    context.CancelFunc
}

// SaramaKafkaConsumerConfig holds Sarama Kafka consumer configuration
type SaramaKafkaConsumerConfig struct {
	Brokers         []string
	GroupID         string
	ClientID        string
	Topics          []string
	OffsetInitial   int64
	SessionTimeout  time.Duration
	HeartbeatInterval time.Duration
	MaxProcessingTime time.Duration
	AutoCommit      bool
	AutoCommitInterval time.Duration
	EnableTLS       bool
	SASLEnabled     bool
	SASLMechanism   string
	SASLUsername    string
	SASLPassword    string
}

// DefaultSaramaKafkaConsumerConfig returns default consumer configuration
func DefaultSaramaKafkaConsumerConfig() *SaramaKafkaConsumerConfig {
	return &SaramaKafkaConsumerConfig{
		Brokers:           []string{"localhost:9092"},
		GroupID:           "waas-consumer",
		ClientID:          "waas-consumer",
		OffsetInitial:     sarama.OffsetNewest,
		SessionTimeout:    30 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		MaxProcessingTime: 60 * time.Second,
		AutoCommit:        true,
		AutoCommitInterval: 1 * time.Second,
	}
}

// NewSaramaKafkaConsumer creates a new Kafka consumer
func NewSaramaKafkaConsumer(config *SaramaKafkaConsumerConfig, handler func(ctx context.Context, topic string, event *StreamEvent) error) (*SaramaKafkaConsumer, error) {
	if config == nil {
		config = DefaultSaramaKafkaConsumerConfig()
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID
	saramaConfig.Consumer.Group.Session.Timeout = config.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = config.HeartbeatInterval
	saramaConfig.Consumer.MaxProcessingTime = config.MaxProcessingTime
	saramaConfig.Consumer.Offsets.Initial = config.OffsetInitial
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.AutoCommit
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = config.AutoCommitInterval
	saramaConfig.Consumer.Return.Errors = true

	// TLS configuration
	if config.EnableTLS {
		saramaConfig.Net.TLS.Enable = true
	}

	// SASL configuration
	if config.SASLEnabled {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(config.SASLMechanism)
		saramaConfig.Net.SASL.User = config.SASLUsername
		saramaConfig.Net.SASL.Password = config.SASLPassword
	}

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &SaramaKafkaConsumer{
		consumerGroup: consumerGroup,
		config:        config,
		handler:       handler,
		metrics:       &ConsumerMetrics{},
	}, nil
}

// Start starts consuming messages
func (c *SaramaKafkaConsumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	handler := &consumerGroupHandler{
		consumer: c,
		handler:  c.handler,
	}

	// Consume in a loop (handles rebalancing)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := c.consumerGroup.Consume(ctx, c.config.Topics, handler); err != nil {
					c.metrics.mu.Lock()
					c.metrics.LastErrorTime = time.Now()
					c.metrics.LastError = err.Error()
					c.metrics.mu.Unlock()
				}
			}
		}
	}()

	return nil
}

// Stop stops consuming messages
func (c *SaramaKafkaConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	return c.consumerGroup.Close()
}

// GetMetrics returns current consumer metrics
func (c *SaramaKafkaConsumer) GetMetrics() ConsumerMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	return *c.metrics
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	consumer *SaramaKafkaConsumer
	handler  func(ctx context.Context, topic string, event *StreamEvent) error
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.consumer.metrics.mu.Lock()
		h.consumer.metrics.MessagesReceived++
		h.consumer.metrics.LastReceiveTime = time.Now()
		h.consumer.metrics.mu.Unlock()

		// Parse event
		var event StreamEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			// If not JSON, create a raw event
			event = StreamEvent{
				ID:        fmt.Sprintf("%d-%d", msg.Partition, msg.Offset),
				Key:       string(msg.Key),
				Value:     json.RawMessage(msg.Value),
				Timestamp: msg.Timestamp,
			}
			// Parse headers
			event.Headers = make(map[string]string)
			for _, header := range msg.Headers {
				event.Headers[string(header.Key)] = string(header.Value)
			}
		}

		// Call handler
		if err := h.handler(session.Context(), msg.Topic, &event); err != nil {
			h.consumer.metrics.mu.Lock()
			h.consumer.metrics.MessagesFailed++
			h.consumer.metrics.LastErrorTime = time.Now()
			h.consumer.metrics.LastError = err.Error()
			h.consumer.metrics.mu.Unlock()
			continue
		}

		// Mark message as processed
		session.MarkMessage(msg, "")

		h.consumer.metrics.mu.Lock()
		h.consumer.metrics.MessagesProcessed++
		h.consumer.metrics.LastProcessTime = time.Now()
		h.consumer.metrics.mu.Unlock()
	}

	return nil
}

// SaramaKafkaAdmin provides Kafka admin operations using Sarama
type SaramaKafkaAdmin struct {
	admin  sarama.ClusterAdmin
	config *SaramaKafkaConfig
}

// NewSaramaKafkaAdmin creates a new Kafka admin client
func NewSaramaKafkaAdmin(config *SaramaKafkaConfig) (*SaramaKafkaAdmin, error) {
	if config == nil {
		config = DefaultSaramaKafkaConfig()
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID

	if config.EnableTLS {
		saramaConfig.Net.TLS.Enable = true
	}

	if config.SASLEnabled {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(config.SASLMechanism)
		saramaConfig.Net.SASL.User = config.SASLUsername
		saramaConfig.Net.SASL.Password = config.SASLPassword
	}

	admin, err := sarama.NewClusterAdmin(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka admin: %w", err)
	}

	return &SaramaKafkaAdmin{
		admin:  admin,
		config: config,
	}, nil
}

// CreateTopic creates a Kafka topic
func (a *SaramaKafkaAdmin) CreateTopic(name string, partitions int32, replicationFactor int16, config map[string]*string) error {
	topicDetail := &sarama.TopicDetail{
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
		ConfigEntries:     config,
	}

	return a.admin.CreateTopic(name, topicDetail, false)
}

// DeleteTopic deletes a Kafka topic
func (a *SaramaKafkaAdmin) DeleteTopic(name string) error {
	return a.admin.DeleteTopic(name)
}

// ListTopics lists all Kafka topics
func (a *SaramaKafkaAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	return a.admin.ListTopics()
}

// DescribeTopic describes a Kafka topic
func (a *SaramaKafkaAdmin) DescribeTopic(topics []string) ([]*sarama.TopicMetadata, error) {
	return a.admin.DescribeTopics(topics)
}

// Close closes the admin client
func (a *SaramaKafkaAdmin) Close() error {
	return a.admin.Close()
}

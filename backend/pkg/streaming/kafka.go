package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// KafkaProducer implements Producer for Apache Kafka
type KafkaProducer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
	buffer      []*StreamEvent
	bufferSize  int
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer() *KafkaProducer {
	return &KafkaProducer{
		buffer:     make([]*StreamEvent, 0),
		bufferSize: 100, // Default batch size
		stopChan:   make(chan struct{}),
	}
}

// Init initializes the Kafka producer
func (p *KafkaProducer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bridge.Config == nil || bridge.Config.KafkaConfig == nil {
		return ErrInvalidConfig
	}

	kafkaCfg := bridge.Config.KafkaConfig
	if len(kafkaCfg.Brokers) == 0 || kafkaCfg.Topic == "" {
		return fmt.Errorf("%w: brokers and topic are required", ErrInvalidConfig)
	}

	p.bridge = bridge
	p.credentials = credentials

	if bridge.Config.BatchSize > 0 {
		p.bufferSize = bridge.Config.BatchSize
	}

	// Start background flusher
	flushInterval := 5 * time.Second
	if bridge.Config.FlushIntervalMs > 0 {
		flushInterval = time.Duration(bridge.Config.FlushIntervalMs) * time.Millisecond
	}
	p.flushTicker = time.NewTicker(flushInterval)

	go p.backgroundFlusher()

	return nil
}

// backgroundFlusher periodically flushes the buffer
func (p *KafkaProducer) backgroundFlusher() {
	for {
		select {
		case <-p.flushTicker.C:
			_ = p.Flush(context.Background())
		case <-p.stopChan:
			return
		}
	}
}

// Send sends a single event to Kafka
func (p *KafkaProducer) Send(ctx context.Context, event *StreamEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("producer is closed")
	}

	// Set event metadata
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.BridgeID = p.bridge.ID
	event.TenantID = p.bridge.TenantID
	event.Timestamp = time.Now()
	event.SourceTopic = p.bridge.Config.KafkaConfig.Topic

	p.buffer = append(p.buffer, event)

	// Flush if buffer is full
	if len(p.buffer) >= p.bufferSize {
		return p.flushLocked(ctx)
	}

	return nil
}

// SendBatch sends multiple events to Kafka
func (p *KafkaProducer) SendBatch(ctx context.Context, events []*StreamEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("producer is closed")
	}

	for _, event := range events {
		if event.ID == "" {
			event.ID = uuid.New().String()
		}
		event.BridgeID = p.bridge.ID
		event.TenantID = p.bridge.TenantID
		event.Timestamp = time.Now()
		event.SourceTopic = p.bridge.Config.KafkaConfig.Topic
	}

	p.buffer = append(p.buffer, events...)

	if len(p.buffer) >= p.bufferSize {
		return p.flushLocked(ctx)
	}

	return nil
}

// Flush flushes all pending messages
func (p *KafkaProducer) Flush(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flushLocked(ctx)
}

// flushLocked flushes buffer (must hold lock)
func (p *KafkaProducer) flushLocked(ctx context.Context) error {
	if len(p.buffer) == 0 {
		return nil
	}

	// In production, this would use the actual Kafka client
	// For now, we simulate successful send
	for _, event := range p.buffer {
		event.Status = "sent"
		now := time.Now()
		event.ProcessedAt = &now
	}

	p.buffer = p.buffer[:0]
	return nil
}

// Close closes the producer
func (p *KafkaProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.stopChan)
	if p.flushTicker != nil {
		p.flushTicker.Stop()
	}

	// Flush remaining messages
	return p.flushLocked(context.Background())
}

// Healthy checks if the producer is healthy
func (p *KafkaProducer) Healthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// KafkaConsumer implements Consumer for Apache Kafka
type KafkaConsumer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	running     bool
	paused      bool
	stopChan    chan struct{}
	handler     EventHandler
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer() *KafkaConsumer {
	return &KafkaConsumer{
		stopChan: make(chan struct{}),
	}
}

// Init initializes the Kafka consumer
func (c *KafkaConsumer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if bridge.Config == nil || bridge.Config.KafkaConfig == nil {
		return ErrInvalidConfig
	}

	kafkaCfg := bridge.Config.KafkaConfig
	if len(kafkaCfg.Brokers) == 0 || kafkaCfg.Topic == "" {
		return fmt.Errorf("%w: brokers and topic are required", ErrInvalidConfig)
	}

	c.bridge = bridge
	c.credentials = credentials

	return nil
}

// Start begins consuming messages
func (c *KafkaConsumer) Start(ctx context.Context, handler EventHandler) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return errors.New("consumer already running")
	}
	c.running = true
	c.handler = handler
	c.mu.Unlock()

	// Simulated consumption loop
	// In production, this would use the actual Kafka consumer
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		case <-ticker.C:
			c.mu.RLock()
			paused := c.paused
			c.mu.RUnlock()

			if paused {
				continue
			}
			// In production, poll for messages here
		}
	}
}

// Pause pauses message consumption
func (c *KafkaConsumer) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	return nil
}

// Resume resumes message consumption
func (c *KafkaConsumer) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	return nil
}

// Commit commits the offset for an event
func (c *KafkaConsumer) Commit(ctx context.Context, event *StreamEvent) error {
	// In production, commit offset to Kafka
	return nil
}

// GetLag returns the current consumer lag
func (c *KafkaConsumer) GetLag(ctx context.Context) (int64, error) {
	// In production, calculate lag from high watermark - committed offset
	return 0, nil
}

// Close closes the consumer
func (c *KafkaConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false
	close(c.stopChan)
	return nil
}

// Healthy checks if the consumer is healthy
func (c *KafkaConsumer) Healthy(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running && !c.paused
}

// KinesisProducer implements Producer for AWS Kinesis
type KinesisProducer struct {
	bridge      *StreamingBridge
	credentials map[string]string
	mu          sync.RWMutex
	closed      bool
	buffer      []*StreamEvent
}

// NewKinesisProducer creates a new Kinesis producer
func NewKinesisProducer() *KinesisProducer {
	return &KinesisProducer{
		buffer: make([]*StreamEvent, 0),
	}
}

// Init initializes the Kinesis producer
func (p *KinesisProducer) Init(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bridge.Config == nil || bridge.Config.KinesisConfig == nil {
		return ErrInvalidConfig
	}

	cfg := bridge.Config.KinesisConfig
	if cfg.StreamName == "" || cfg.Region == "" {
		return fmt.Errorf("%w: stream_name and region are required", ErrInvalidConfig)
	}

	p.bridge = bridge
	p.credentials = credentials

	return nil
}

// Send sends a single event to Kinesis
func (p *KinesisProducer) Send(ctx context.Context, event *StreamEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("producer is closed")
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.BridgeID = p.bridge.ID
	event.TenantID = p.bridge.TenantID
	event.Timestamp = time.Now()

	// Set partition key
	if event.Key == "" {
		event.Key = event.ID
	}

	// In production, use PutRecord API
	event.Status = "sent"
	now := time.Now()
	event.ProcessedAt = &now

	return nil
}

// SendBatch sends multiple events to Kinesis
func (p *KinesisProducer) SendBatch(ctx context.Context, events []*StreamEvent) error {
	// Kinesis supports up to 500 records per PutRecords call
	const maxBatchSize = 500

	for i := 0; i < len(events); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(events) {
			end = len(events)
		}
		batch := events[i:end]

		for _, event := range batch {
			if err := p.Send(ctx, event); err != nil {
				return err
			}
		}
	}

	return nil
}

// Flush flushes pending messages
func (p *KinesisProducer) Flush(ctx context.Context) error {
	return nil // Kinesis sends synchronously
}

// Close closes the producer
func (p *KinesisProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

// Healthy checks if the producer is healthy
func (p *KinesisProducer) Healthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// ConfluentSchemaRegistry implements SchemaRegistry for Confluent Schema Registry
type ConfluentSchemaRegistry struct {
	url      string
	username string
	password string
	cache    map[int]string
	mu       sync.RWMutex
}

// NewConfluentSchemaRegistry creates a new Confluent Schema Registry client
func NewConfluentSchemaRegistry(url, username, password string) *ConfluentSchemaRegistry {
	return &ConfluentSchemaRegistry{
		url:      url,
		username: username,
		password: password,
		cache:    make(map[int]string),
	}
}

// RegisterSchema registers a new schema
func (r *ConfluentSchemaRegistry) RegisterSchema(ctx context.Context, subject string, schema string, format SchemaFormat) (int, error) {
	// In production, POST to /subjects/{subject}/versions
	// For now, return a mock schema ID
	r.mu.Lock()
	defer r.mu.Unlock()

	schemaID := len(r.cache) + 1
	r.cache[schemaID] = schema

	return schemaID, nil
}

// GetSchema retrieves a schema by ID
func (r *ConfluentSchemaRegistry) GetSchema(ctx context.Context, schemaID int) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, ok := r.cache[schemaID]
	if !ok {
		return "", fmt.Errorf("schema not found: %d", schemaID)
	}

	return schema, nil
}

// GetLatestSchema retrieves the latest schema for a subject
func (r *ConfluentSchemaRegistry) GetLatestSchema(ctx context.Context, subject string) (string, int, error) {
	// In production, GET /subjects/{subject}/versions/latest
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.cache) == 0 {
		return "", 0, fmt.Errorf("no schemas registered for subject: %s", subject)
	}

	// Return the last registered schema
	lastID := len(r.cache)
	return r.cache[lastID], lastID, nil
}

// ValidateSchema validates data against a schema
func (r *ConfluentSchemaRegistry) ValidateSchema(ctx context.Context, schemaID int, data []byte) error {
	schema, err := r.GetSchema(ctx, schemaID)
	if err != nil {
		return err
	}

	// For JSON schema, attempt to validate structure
	if schema != "" {
		var v interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("%w: invalid JSON", ErrSchemaValidation)
		}
	}

	return nil
}

// Encode encodes data with schema
func (r *ConfluentSchemaRegistry) Encode(ctx context.Context, schemaID int, data interface{}) ([]byte, error) {
	// For JSON, just marshal
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// In production with Avro, prepend magic byte and schema ID
	return encoded, nil
}

// Decode decodes data with schema
func (r *ConfluentSchemaRegistry) Decode(ctx context.Context, data []byte) (interface{}, int, error) {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, 0, err
	}
	return result, 0, nil
}

// CheckCompatibility checks if a schema is compatible
func (r *ConfluentSchemaRegistry) CheckCompatibility(ctx context.Context, subject string, schema string) (bool, error) {
	// In production, POST to /compatibility/subjects/{subject}/versions/latest
	return true, nil
}

// Close closes the registry client
func (r *ConfluentSchemaRegistry) Close() error {
	return nil
}

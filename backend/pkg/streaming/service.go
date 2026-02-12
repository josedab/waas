package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/queue"
)

// Service provides streaming bridge operations
type Service struct {
	repo           Repository
	producers      map[string]Producer
	consumers      map[string]Consumer
	schemaRegistry SchemaRegistry
	webhookQueue   queue.PublisherInterface
	mu             sync.RWMutex
	config         *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxBridgesPerTenant    int
	DefaultBatchSize       int
	DefaultFlushIntervalMs int
	MaxEventsPerSecond     int
	EnableSchemaValidation bool
	SchemaRegistryURL      string
	SchemaRegistryUsername string
	SchemaRegistryPassword string
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxBridgesPerTenant:    50,
		DefaultBatchSize:       100,
		DefaultFlushIntervalMs: 5000,
		MaxEventsPerSecond:     10000,
		EnableSchemaValidation: true,
	}
}

// NewService creates a new streaming service
func NewService(repo Repository, webhookQueue queue.PublisherInterface, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	s := &Service{
		repo:         repo,
		producers:    make(map[string]Producer),
		consumers:    make(map[string]Consumer),
		webhookQueue: webhookQueue,
		config:       config,
	}

	// Initialize schema registry if configured
	if config.SchemaRegistryURL != "" {
		s.schemaRegistry = NewConfluentSchemaRegistry(
			config.SchemaRegistryURL,
			config.SchemaRegistryUsername,
			config.SchemaRegistryPassword,
		)
	}

	return s
}

// CreateBridge creates a new streaming bridge
func (s *Service) CreateBridge(ctx context.Context, tenantID string, req *CreateBridgeRequest) (*StreamingBridge, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Check bridge limit
	existing, _, err := s.repo.ListBridges(ctx, tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing bridges: %w", err)
	}
	if len(existing) >= s.config.MaxBridgesPerTenant {
		return nil, fmt.Errorf("maximum bridges reached: %d", s.config.MaxBridgesPerTenant)
	}

	// Check for duplicate name
	if _, err := s.repo.GetBridgeByName(ctx, tenantID, req.Name); err == nil {
		return nil, ErrBridgeAlreadyExists
	}

	now := time.Now()
	bridge := &StreamingBridge{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		StreamType:      req.StreamType,
		Direction:       req.Direction,
		Status:          BridgeStatusCreating,
		Config:          req.Config,
		SchemaConfig:    req.SchemaConfig,
		TransformScript: req.TransformScript,
		FilterRules:     req.FilterRules,
		Metadata:        req.Metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Apply defaults
	if bridge.Config.BatchSize == 0 {
		bridge.Config.BatchSize = s.config.DefaultBatchSize
	}
	if bridge.Config.FlushIntervalMs == 0 {
		bridge.Config.FlushIntervalMs = s.config.DefaultFlushIntervalMs
	}

	// Save bridge
	if err := s.repo.CreateBridge(ctx, bridge); err != nil {
		return nil, fmt.Errorf("failed to create bridge: %w", err)
	}

	// Extract and save credentials
	credentials := s.extractCredentials(bridge)
	if len(credentials) > 0 {
		if err := s.repo.SaveCredentials(ctx, bridge.ID, credentials); err != nil {
			// Rollback bridge creation
			if delErr := s.repo.DeleteBridge(ctx, tenantID, bridge.ID); delErr != nil {
				log.Printf("streaming: failed to rollback bridge %s after credential save failure: %v", bridge.ID, delErr)
			}
			return nil, fmt.Errorf("failed to save credentials: %w", err)
		}
	}

	// Initialize producer/consumer
	if err := s.initializeBridge(ctx, bridge, credentials); err != nil {
		bridge.Status = BridgeStatusError
		bridge.ErrorMessage = err.Error()
		if updateErr := s.repo.UpdateBridge(ctx, bridge); updateErr != nil {
			log.Printf("streaming: failed to update bridge %s error status: %v", bridge.ID, updateErr)
		}
		return bridge, nil // Return bridge with error status
	}

	bridge.Status = BridgeStatusActive
	if err := s.repo.UpdateBridge(ctx, bridge); err != nil {
		return nil, fmt.Errorf("failed to update bridge status: %w", err)
	}

	return bridge, nil
}

// validateCreateRequest validates the create bridge request
func (s *Service) validateCreateRequest(req *CreateBridgeRequest) error {
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}

	validTypes := map[StreamType]bool{
		StreamTypeKafka:       true,
		StreamTypeKinesis:     true,
		StreamTypePulsar:      true,
		StreamTypeEventBridge: true,
		StreamTypeNATS:        true,
		StreamTypeRabbitMQ:    true,
		StreamTypeSQS:         true,
		StreamTypeSNS:         true,
	}
	if !validTypes[req.StreamType] {
		return fmt.Errorf("%w: invalid stream type: %s", ErrInvalidConfig, req.StreamType)
	}

	validDirections := map[Direction]bool{
		DirectionInbound:  true,
		DirectionOutbound: true,
		DirectionBoth:     true,
	}
	if !validDirections[req.Direction] {
		return fmt.Errorf("%w: invalid direction: %s", ErrInvalidConfig, req.Direction)
	}

	if req.Config == nil {
		return fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}

	// Validate platform-specific config
	switch req.StreamType {
	case StreamTypeKafka:
		if req.Config.KafkaConfig == nil {
			return fmt.Errorf("%w: kafka config is required", ErrInvalidConfig)
		}
		if len(req.Config.KafkaConfig.Brokers) == 0 {
			return fmt.Errorf("%w: kafka brokers are required", ErrInvalidConfig)
		}
		if req.Config.KafkaConfig.Topic == "" {
			return fmt.Errorf("%w: kafka topic is required", ErrInvalidConfig)
		}
	case StreamTypeKinesis:
		if req.Config.KinesisConfig == nil {
			return fmt.Errorf("%w: kinesis config is required", ErrInvalidConfig)
		}
		if req.Config.KinesisConfig.StreamName == "" {
			return fmt.Errorf("%w: kinesis stream name is required", ErrInvalidConfig)
		}
		if req.Config.KinesisConfig.Region == "" {
			return fmt.Errorf("%w: kinesis region is required", ErrInvalidConfig)
		}
	case StreamTypePulsar:
		if req.Config.PulsarConfig == nil {
			return fmt.Errorf("%w: pulsar config is required", ErrInvalidConfig)
		}
	case StreamTypeEventBridge:
		if req.Config.EventBridgeConfig == nil {
			return fmt.Errorf("%w: eventbridge config is required", ErrInvalidConfig)
		}
	case StreamTypeNATS:
		if req.Config.NATSConfig == nil {
			return fmt.Errorf("%w: NATS config is required", ErrInvalidConfig)
		}
		if req.Config.NATSConfig.URL == "" || req.Config.NATSConfig.Subject == "" {
			return fmt.Errorf("%w: NATS URL and subject are required", ErrInvalidConfig)
		}
	case StreamTypeRabbitMQ:
		if req.Config.RabbitMQConfig == nil {
			return fmt.Errorf("%w: RabbitMQ config is required", ErrInvalidConfig)
		}
		if req.Config.RabbitMQConfig.URL == "" || req.Config.RabbitMQConfig.Exchange == "" {
			return fmt.Errorf("%w: RabbitMQ URL and exchange are required", ErrInvalidConfig)
		}
	case StreamTypeSQS:
		if req.Config.SQSConfig == nil {
			return fmt.Errorf("%w: SQS config is required", ErrInvalidConfig)
		}
		if req.Config.SQSConfig.QueueURL == "" || req.Config.SQSConfig.Region == "" {
			return fmt.Errorf("%w: SQS queue URL and region are required", ErrInvalidConfig)
		}
	case StreamTypeSNS:
		if req.Config.SNSConfig == nil {
			return fmt.Errorf("%w: SNS config is required", ErrInvalidConfig)
		}
		if req.Config.SNSConfig.TopicARN == "" || req.Config.SNSConfig.Region == "" {
			return fmt.Errorf("%w: SNS topic ARN and region are required", ErrInvalidConfig)
		}
	}

	return nil
}

// extractCredentials extracts sensitive credentials from bridge config
func (s *Service) extractCredentials(bridge *StreamingBridge) map[string]string {
	creds := make(map[string]string)

	if bridge.Config.KafkaConfig != nil {
		if bridge.Config.KafkaConfig.SASLPassword != "" {
			creds["kafka_sasl_password"] = bridge.Config.KafkaConfig.SASLPassword
			bridge.Config.KafkaConfig.SASLPassword = "" // Clear from config
		}
	}

	if bridge.Config.KinesisConfig != nil {
		if bridge.Config.KinesisConfig.SecretAccessKey != "" {
			creds["kinesis_secret_key"] = bridge.Config.KinesisConfig.SecretAccessKey
			bridge.Config.KinesisConfig.SecretAccessKey = ""
		}
	}

	if bridge.Config.PulsarConfig != nil {
		if bridge.Config.PulsarConfig.AuthToken != "" {
			creds["pulsar_auth_token"] = bridge.Config.PulsarConfig.AuthToken
			bridge.Config.PulsarConfig.AuthToken = ""
		}
	}

	if bridge.Config.EventBridgeConfig != nil {
		if bridge.Config.EventBridgeConfig.SecretAccessKey != "" {
			creds["eventbridge_secret_key"] = bridge.Config.EventBridgeConfig.SecretAccessKey
			bridge.Config.EventBridgeConfig.SecretAccessKey = ""
		}
	}

	if bridge.Config.NATSConfig != nil {
		if bridge.Config.NATSConfig.Token != "" {
			creds["nats_token"] = bridge.Config.NATSConfig.Token
			bridge.Config.NATSConfig.Token = ""
		}
	}

	if bridge.Config.SQSConfig != nil {
		if bridge.Config.SQSConfig.SecretAccessKey != "" {
			creds["sqs_secret_key"] = bridge.Config.SQSConfig.SecretAccessKey
			bridge.Config.SQSConfig.SecretAccessKey = ""
		}
	}

	if bridge.Config.SNSConfig != nil {
		if bridge.Config.SNSConfig.SecretAccessKey != "" {
			creds["sns_secret_key"] = bridge.Config.SNSConfig.SecretAccessKey
			bridge.Config.SNSConfig.SecretAccessKey = ""
		}
	}

	return creds
}

// initializeBridge initializes the producer/consumer for a bridge
func (s *Service) initializeBridge(ctx context.Context, bridge *StreamingBridge, credentials map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create producer for outbound/both directions
	if bridge.Direction == DirectionOutbound || bridge.Direction == DirectionBoth {
		producer, err := s.createProducer(bridge.StreamType)
		if err != nil {
			return fmt.Errorf("failed to create producer: %w", err)
		}

		if err := producer.Init(ctx, bridge, credentials); err != nil {
			return fmt.Errorf("failed to initialize producer: %w", err)
		}

		s.producers[bridge.ID] = producer
	}

	// Create consumer for inbound/both directions
	if bridge.Direction == DirectionInbound || bridge.Direction == DirectionBoth {
		consumer, err := s.createConsumer(bridge.StreamType)
		if err != nil {
			return fmt.Errorf("failed to create consumer: %w", err)
		}

		if err := consumer.Init(ctx, bridge, credentials); err != nil {
			return fmt.Errorf("failed to initialize consumer: %w", err)
		}

		s.consumers[bridge.ID] = consumer

		// Start consumer in background
		go s.runConsumer(bridge)
	}

	return nil
}

// createProducer creates a producer for the given stream type
func (s *Service) createProducer(streamType StreamType) (Producer, error) {
	switch streamType {
	case StreamTypeKafka:
		return NewKafkaProducer(), nil
	case StreamTypeKinesis:
		return NewKinesisProducer(), nil
	case StreamTypeNATS:
		return NewNATSProducer(), nil
	case StreamTypeRabbitMQ:
		return NewRabbitMQProducer(), nil
	case StreamTypeSQS:
		return NewSQSProducer(), nil
	case StreamTypeSNS:
		return NewSNSProducer(), nil
	default:
		return nil, fmt.Errorf("unsupported stream type for producer: %s", streamType)
	}
}

// createConsumer creates a consumer for the given stream type
func (s *Service) createConsumer(streamType StreamType) (Consumer, error) {
	switch streamType {
	case StreamTypeKafka:
		return NewKafkaConsumer(), nil
	case StreamTypeNATS:
		return NewNATSConsumer(), nil
	case StreamTypeRabbitMQ:
		return NewRabbitMQConsumer(), nil
	case StreamTypeSQS:
		return NewSQSConsumer(), nil
	default:
		return nil, fmt.Errorf("unsupported stream type for consumer: %s", streamType)
	}
}

// runConsumer runs the consumer and forwards events to webhooks
func (s *Service) runConsumer(bridge *StreamingBridge) {
	s.mu.RLock()
	consumer, ok := s.consumers[bridge.ID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	ctx := context.Background()
	handler := func(ctx context.Context, event *StreamEvent) error {
		return s.handleInboundEvent(ctx, bridge, event)
	}

	if err := consumer.Start(ctx, handler); err != nil {
		log.Printf("streaming: consumer failed for bridge %s: %v", bridge.ID, err)
	}
}

// handleInboundEvent processes an inbound streaming event
func (s *Service) handleInboundEvent(ctx context.Context, bridge *StreamingBridge, event *StreamEvent) error {
	// Apply filters
	if !s.passesFilters(event, bridge.FilterRules) {
		return nil // Filtered out
	}

	// Apply transformation if configured
	payload := event.Value
	if bridge.TransformScript != "" {
		// Execute simple JSON transformation via script evaluation
		// In production, use proper sandboxed execution
		transformed, err := applyTransform(bridge.TransformScript, event.Value, event.Headers)
		if err != nil {
			event.Status = "transform_failed"
			event.ErrorMessage = err.Error()
			if saveErr := s.repo.SaveEvent(ctx, event); saveErr != nil {
				log.Printf("streaming: failed to save transform_failed event for bridge %s: %v", bridge.ID, saveErr)
			}
			return err
		}
		payload = transformed
	}

	// Create webhook delivery message
	deliveryMsg := &queue.DeliveryMessage{
		DeliveryID:    uuid.New(),
		TenantID:      uuid.MustParse(bridge.TenantID),
		Payload:       payload,
		Headers:       event.Headers,
		AttemptNumber: 1,
		ScheduledAt:   time.Now(),
		MaxAttempts:   5,
	}

	// Publish to webhook queue
	if err := s.webhookQueue.PublishDelivery(ctx, deliveryMsg); err != nil {
		event.Status = "queue_failed"
		event.ErrorMessage = err.Error()
		if saveErr := s.repo.SaveEvent(ctx, event); saveErr != nil {
			log.Printf("streaming: failed to save queue_failed event for bridge %s: %v", bridge.ID, saveErr)
		}
		return err
	}

	event.Status = "forwarded"
	event.DeliveryID = deliveryMsg.DeliveryID.String()
	now := time.Now()
	event.ProcessedAt = &now
	if err := s.repo.SaveEvent(ctx, event); err != nil {
		log.Printf("streaming: failed to save forwarded event for bridge %s: %v", bridge.ID, err)
	}

	// Update metrics
	if err := s.repo.IncrementEventCounters(ctx, bridge.ID, 1, 1, 0); err != nil {
		log.Printf("streaming: failed to increment event counters for bridge %s: %v", bridge.ID, err)
	}

	return nil
}

// passesFilters checks if an event passes the filter rules
func (s *Service) passesFilters(event *StreamEvent, rules []FilterRule) bool {
	if len(rules) == 0 {
		return true
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Value, &data); err != nil {
		return true // Can't parse, let it through
	}

	for _, rule := range rules {
		fieldValue, exists := data[rule.Field]

		var passes bool
		switch rule.Operator {
		case "exists":
			passes = exists
		case "eq":
			passes = exists && fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", rule.Value)
		case "neq":
			passes = !exists || fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", rule.Value)
		case "contains":
			passes = exists && containsString(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", rule.Value))
		default:
			passes = true
		}

		if rule.Negate {
			passes = !passes
		}

		if !passes {
			return false
		}
	}

	return true
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// applyTransform applies a simple transformation to payload
func applyTransform(script string, payload json.RawMessage, headers map[string]string) (json.RawMessage, error) {
	// Simple transformation: wrap payload in envelope if script indicates
	// In production, use sandboxed JavaScript execution
	if script == "" {
		return payload, nil
	}

	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, nil
	}

	// Apply transformation based on script type
	// For now, just pass through
	return payload, nil
}

// GetBridge retrieves a streaming bridge
func (s *Service) GetBridge(ctx context.Context, tenantID, bridgeID string) (*StreamingBridge, error) {
	return s.repo.GetBridge(ctx, tenantID, bridgeID)
}

// UpdateBridge updates a streaming bridge
func (s *Service) UpdateBridge(ctx context.Context, tenantID, bridgeID string, req *UpdateBridgeRequest) (*StreamingBridge, error) {
	bridge, err := s.repo.GetBridge(ctx, tenantID, bridgeID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		bridge.Name = *req.Name
	}
	if req.Description != nil {
		bridge.Description = *req.Description
	}
	if req.Status != nil {
		// Handle status changes
		if *req.Status == BridgeStatusPaused && bridge.Status == BridgeStatusActive {
			s.pauseBridge(bridge.ID)
		} else if *req.Status == BridgeStatusActive && bridge.Status == BridgeStatusPaused {
			s.resumeBridge(bridge.ID)
		}
		bridge.Status = *req.Status
	}
	if req.Config != nil {
		bridge.Config = req.Config
	}
	if req.SchemaConfig != nil {
		bridge.SchemaConfig = req.SchemaConfig
	}
	if req.TransformScript != nil {
		bridge.TransformScript = *req.TransformScript
	}
	if req.FilterRules != nil {
		bridge.FilterRules = req.FilterRules
	}
	if req.Metadata != nil {
		bridge.Metadata = req.Metadata
	}

	bridge.UpdatedAt = time.Now()

	if err := s.repo.UpdateBridge(ctx, bridge); err != nil {
		return nil, err
	}

	return bridge, nil
}

// pauseBridge pauses a bridge's consumer
func (s *Service) pauseBridge(bridgeID string) {
	s.mu.RLock()
	consumer, ok := s.consumers[bridgeID]
	s.mu.RUnlock()

	if ok {
		if err := consumer.Pause(); err != nil {
			log.Printf("streaming: failed to pause consumer for bridge %s: %v", bridgeID, err)
		}
	}
}

// resumeBridge resumes a bridge's consumer
func (s *Service) resumeBridge(bridgeID string) {
	s.mu.RLock()
	consumer, ok := s.consumers[bridgeID]
	s.mu.RUnlock()

	if ok {
		if err := consumer.Resume(); err != nil {
			log.Printf("streaming: failed to resume consumer for bridge %s: %v", bridgeID, err)
		}
	}
}

// DeleteBridge deletes a streaming bridge
func (s *Service) DeleteBridge(ctx context.Context, tenantID, bridgeID string) error {
	bridge, err := s.repo.GetBridge(ctx, tenantID, bridgeID)
	if err != nil {
		return err
	}

	// Stop and remove producer/consumer
	s.mu.Lock()
	if producer, ok := s.producers[bridgeID]; ok {
		if err := producer.Close(); err != nil {
			log.Printf("streaming: failed to close producer for bridge %s: %v", bridgeID, err)
		}
		delete(s.producers, bridgeID)
	}
	if consumer, ok := s.consumers[bridgeID]; ok {
		if err := consumer.Close(); err != nil {
			log.Printf("streaming: failed to close consumer for bridge %s: %v", bridgeID, err)
		}
		delete(s.consumers, bridgeID)
	}
	s.mu.Unlock()

	// Delete credentials
	if err := s.repo.DeleteCredentials(ctx, bridgeID); err != nil {
		log.Printf("streaming: failed to delete credentials for bridge %s: %v", bridgeID, err)
	}

	// Update status to deleting
	bridge.Status = BridgeStatusDeleting
	if err := s.repo.UpdateBridge(ctx, bridge); err != nil {
		log.Printf("streaming: failed to update bridge %s to deleting status: %v", bridgeID, err)
	}

	// Delete bridge
	return s.repo.DeleteBridge(ctx, tenantID, bridgeID)
}

// ListBridges lists streaming bridges for a tenant
func (s *Service) ListBridges(ctx context.Context, tenantID string, filters *BridgeFilters) (*ListBridgesResponse, error) {
	bridges, total, err := s.repo.ListBridges(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	page := 1
	pageSize := 20
	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := &ListBridgesResponse{
		Bridges:    make([]BridgeResponse, len(bridges)),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	for i, bridge := range bridges {
		response.Bridges[i] = BridgeResponse{
			StreamingBridge: &bridge,
		}
		// Optionally fetch metrics
		if metrics, err := s.repo.GetLatestMetrics(ctx, tenantID, bridge.ID); err == nil {
			response.Bridges[i].Metrics = metrics
		}
	}

	return response, nil
}

// SendToStream sends a webhook event to a streaming platform
func (s *Service) SendToStream(ctx context.Context, tenantID, bridgeID string, payload json.RawMessage, headers map[string]string) error {
	s.mu.RLock()
	producer, ok := s.producers[bridgeID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("bridge not found or not configured for outbound: %s", bridgeID)
	}

	bridge, err := s.repo.GetBridge(ctx, tenantID, bridgeID)
	if err != nil {
		return err
	}

	// Apply transformation if configured
	var finalPayload json.RawMessage = payload
	if bridge.TransformScript != "" {
		transformed, err := applyTransform(bridge.TransformScript, payload, headers)
		if err != nil {
			return fmt.Errorf("transformation failed: %w", err)
		}
		finalPayload = transformed
	}

	// Validate schema if configured
	if bridge.SchemaConfig != nil && bridge.SchemaConfig.SchemaID > 0 && s.schemaRegistry != nil {
		if err := s.schemaRegistry.ValidateSchema(ctx, bridge.SchemaConfig.SchemaID, finalPayload); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
	}

	event := &StreamEvent{
		BridgeID:  bridgeID,
		TenantID:  tenantID,
		Value:     finalPayload,
		Headers:   headers,
		Timestamp: time.Now(),
		Status:    "pending",
	}

	if err := producer.Send(ctx, event); err != nil {
		if counterErr := s.repo.IncrementEventCounters(ctx, bridgeID, 0, 0, 1); counterErr != nil {
			log.Printf("streaming: failed to increment error counters for bridge %s: %v", bridgeID, counterErr)
		}
		return err
	}

	if err := s.repo.IncrementEventCounters(ctx, bridgeID, 0, 1, 0); err != nil {
		log.Printf("streaming: failed to increment success counters for bridge %s: %v", bridgeID, err)
	}
	return nil
}

// GetBridgeMetrics retrieves metrics for a bridge
func (s *Service) GetBridgeMetrics(ctx context.Context, tenantID, bridgeID string) (*BridgeMetrics, error) {
	return s.repo.GetLatestMetrics(ctx, tenantID, bridgeID)
}

// TestConnection tests connectivity to a streaming platform
func (s *Service) TestConnection(ctx context.Context, req *CreateBridgeRequest) error {
	// Create temporary producer/consumer to test connection
	switch req.StreamType {
	case StreamTypeKafka:
		producer := NewKafkaProducer()
		tempBridge := &StreamingBridge{
			ID:         "test",
			TenantID:   "test",
			StreamType: req.StreamType,
			Config:     req.Config,
		}
		creds := make(map[string]string)
		if req.Config.KafkaConfig.SASLPassword != "" {
			creds["kafka_sasl_password"] = req.Config.KafkaConfig.SASLPassword
		}
		if err := producer.Init(ctx, tempBridge, creds); err != nil {
			return err
		}
		defer producer.Close()
		return nil
	case StreamTypeKinesis:
		producer := NewKinesisProducer()
		tempBridge := &StreamingBridge{
			ID:         "test",
			TenantID:   "test",
			StreamType: req.StreamType,
			Config:     req.Config,
		}
		creds := make(map[string]string)
		if req.Config.KinesisConfig.SecretAccessKey != "" {
			creds["kinesis_secret_key"] = req.Config.KinesisConfig.SecretAccessKey
		}
		if err := producer.Init(ctx, tempBridge, creds); err != nil {
			return err
		}
		defer producer.Close()
		return nil
	default:
		return fmt.Errorf("connection test not supported for: %s", req.StreamType)
	}
}

// Close shuts down all producers and consumers
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, producer := range s.producers {
		if err := producer.Close(); err != nil {
			log.Printf("streaming: failed to close producer %s: %v", id, err)
		}
	}
	for id, consumer := range s.consumers {
		if err := consumer.Close(); err != nil {
			log.Printf("streaming: failed to close consumer %s: %v", id, err)
		}
	}

	if s.schemaRegistry != nil {
		if err := s.schemaRegistry.Close(); err != nil {
			log.Printf("streaming: failed to close schema registry: %v", err)
		}
	}

	return nil
}

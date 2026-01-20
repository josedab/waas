package streaming

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateBridge(ctx context.Context, bridge *StreamingBridge) error {
	args := m.Called(ctx, bridge)
	return args.Error(0)
}

func (m *MockRepository) GetBridge(ctx context.Context, tenantID, bridgeID string) (*StreamingBridge, error) {
	args := m.Called(ctx, tenantID, bridgeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StreamingBridge), args.Error(1)
}

func (m *MockRepository) GetBridgeByName(ctx context.Context, tenantID, name string) (*StreamingBridge, error) {
	args := m.Called(ctx, tenantID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StreamingBridge), args.Error(1)
}

func (m *MockRepository) UpdateBridge(ctx context.Context, bridge *StreamingBridge) error {
	args := m.Called(ctx, bridge)
	return args.Error(0)
}

func (m *MockRepository) DeleteBridge(ctx context.Context, tenantID, bridgeID string) error {
	args := m.Called(ctx, tenantID, bridgeID)
	return args.Error(0)
}

func (m *MockRepository) ListBridges(ctx context.Context, tenantID string, filters *BridgeFilters) ([]StreamingBridge, int, error) {
	args := m.Called(ctx, tenantID, filters)
	return args.Get(0).([]StreamingBridge), args.Int(1), args.Error(2)
}

func (m *MockRepository) SaveCredentials(ctx context.Context, bridgeID string, credentials map[string]string) error {
	args := m.Called(ctx, bridgeID, credentials)
	return args.Error(0)
}

func (m *MockRepository) GetCredentials(ctx context.Context, bridgeID string) (map[string]string, error) {
	args := m.Called(ctx, bridgeID)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockRepository) DeleteCredentials(ctx context.Context, bridgeID string) error {
	args := m.Called(ctx, bridgeID)
	return args.Error(0)
}

func (m *MockRepository) SaveEvent(ctx context.Context, event *StreamEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetEvent(ctx context.Context, tenantID, eventID string) (*StreamEvent, error) {
	args := m.Called(ctx, tenantID, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StreamEvent), args.Error(1)
}

func (m *MockRepository) ListEvents(ctx context.Context, tenantID, bridgeID string, filters *EventFilters) ([]StreamEvent, int, error) {
	args := m.Called(ctx, tenantID, bridgeID, filters)
	return args.Get(0).([]StreamEvent), args.Int(1), args.Error(2)
}

func (m *MockRepository) UpdateEventStatus(ctx context.Context, eventID, status, errorMsg string) error {
	args := m.Called(ctx, eventID, status, errorMsg)
	return args.Error(0)
}

func (m *MockRepository) SaveMetrics(ctx context.Context, metrics *BridgeMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockRepository) GetLatestMetrics(ctx context.Context, tenantID, bridgeID string) (*BridgeMetrics, error) {
	args := m.Called(ctx, tenantID, bridgeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BridgeMetrics), args.Error(1)
}

func (m *MockRepository) GetMetricsHistory(ctx context.Context, tenantID, bridgeID string, start, end time.Time) ([]BridgeMetrics, error) {
	args := m.Called(ctx, tenantID, bridgeID, start, end)
	return args.Get(0).([]BridgeMetrics), args.Error(1)
}

func (m *MockRepository) IncrementEventCounters(ctx context.Context, bridgeID string, eventsIn, eventsOut, eventsFailed int64) error {
	args := m.Called(ctx, bridgeID, eventsIn, eventsOut, eventsFailed)
	return args.Error(0)
}

// MockPublisher is a mock implementation of queue.PublisherInterface
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishDelivery(ctx context.Context, message interface{}) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockPublisher) PublishDelayedDelivery(ctx context.Context, message interface{}, delay time.Duration) error {
	args := m.Called(ctx, message, delay)
	return args.Error(0)
}

func (m *MockPublisher) PublishToDeadLetter(ctx context.Context, message interface{}, reason string) error {
	args := m.Called(ctx, message, reason)
	return args.Error(0)
}

func (m *MockPublisher) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	args := m.Called(ctx, queueName)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPublisher) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]int64), args.Error(1)
}

func TestKafkaProducer_Init(t *testing.T) {
	producer := NewKafkaProducer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKafka,
		Config: &BridgeConfig{
			KafkaConfig: &KafkaConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		},
	}

	err := producer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer producer.Close()

	assert.True(t, producer.Healthy(context.Background()))
}

func TestKafkaProducer_InvalidConfig(t *testing.T) {
	producer := NewKafkaProducer()

	// Missing config
	bridge := &StreamingBridge{
		ID:       "bridge-1",
		TenantID: "tenant-1",
	}

	err := producer.Init(context.Background(), bridge, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidConfig, err)

	// Missing brokers
	bridge.Config = &BridgeConfig{
		KafkaConfig: &KafkaConfig{
			Topic: "test-topic",
		},
	}

	err = producer.Init(context.Background(), bridge, nil)
	assert.Error(t, err)
}

func TestKafkaProducer_Send(t *testing.T) {
	producer := NewKafkaProducer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKafka,
		Config: &BridgeConfig{
			KafkaConfig: &KafkaConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
			BatchSize: 10,
		},
	}

	err := producer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer producer.Close()

	event := &StreamEvent{
		Value: json.RawMessage(`{"test": "data"}`),
	}

	err = producer.Send(context.Background(), event)
	require.NoError(t, err)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "bridge-1", event.BridgeID)
	assert.Equal(t, "tenant-1", event.TenantID)
}

func TestKafkaProducer_SendBatch(t *testing.T) {
	producer := NewKafkaProducer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKafka,
		Config: &BridgeConfig{
			KafkaConfig: &KafkaConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		},
	}

	err := producer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer producer.Close()

	events := []*StreamEvent{
		{Value: json.RawMessage(`{"test": "1"}`)},
		{Value: json.RawMessage(`{"test": "2"}`)},
		{Value: json.RawMessage(`{"test": "3"}`)},
	}

	err = producer.SendBatch(context.Background(), events)
	require.NoError(t, err)

	for _, event := range events {
		assert.NotEmpty(t, event.ID)
	}
}

func TestKafkaConsumer_Init(t *testing.T) {
	consumer := NewKafkaConsumer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKafka,
		Config: &BridgeConfig{
			KafkaConfig: &KafkaConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test-topic",
				ConsumerGroup: "test-group",
			},
		},
	}

	err := consumer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer consumer.Close()
}

func TestKafkaConsumer_PauseResume(t *testing.T) {
	consumer := NewKafkaConsumer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKafka,
		Config: &BridgeConfig{
			KafkaConfig: &KafkaConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		},
	}

	err := consumer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer consumer.Close()

	err = consumer.Pause()
	require.NoError(t, err)

	err = consumer.Resume()
	require.NoError(t, err)
}

func TestKinesisProducer_Init(t *testing.T) {
	producer := NewKinesisProducer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKinesis,
		Config: &BridgeConfig{
			KinesisConfig: &KinesisConfig{
				StreamName: "test-stream",
				Region:     "us-east-1",
			},
		},
	}

	err := producer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer producer.Close()

	assert.True(t, producer.Healthy(context.Background()))
}

func TestKinesisProducer_Send(t *testing.T) {
	producer := NewKinesisProducer()

	bridge := &StreamingBridge{
		ID:         "bridge-1",
		TenantID:   "tenant-1",
		StreamType: StreamTypeKinesis,
		Config: &BridgeConfig{
			KinesisConfig: &KinesisConfig{
				StreamName: "test-stream",
				Region:     "us-east-1",
			},
		},
	}

	err := producer.Init(context.Background(), bridge, nil)
	require.NoError(t, err)
	defer producer.Close()

	event := &StreamEvent{
		Value: json.RawMessage(`{"test": "data"}`),
	}

	err = producer.Send(context.Background(), event)
	require.NoError(t, err)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "sent", event.Status)
}

func TestSchemaRegistry_RegisterAndGet(t *testing.T) {
	registry := NewConfluentSchemaRegistry("http://localhost:8081", "", "")

	schema := `{"type": "record", "name": "test", "fields": []}`
	schemaID, err := registry.RegisterSchema(context.Background(), "test-subject", schema, SchemaFormatAvro)
	require.NoError(t, err)
	assert.Greater(t, schemaID, 0)

	retrieved, err := registry.GetSchema(context.Background(), schemaID)
	require.NoError(t, err)
	assert.Equal(t, schema, retrieved)
}

func TestSchemaRegistry_Validate(t *testing.T) {
	registry := NewConfluentSchemaRegistry("http://localhost:8081", "", "")

	schema := `{"type": "object"}`
	schemaID, err := registry.RegisterSchema(context.Background(), "test-subject", schema, SchemaFormatJSON)
	require.NoError(t, err)

	// Valid JSON
	err = registry.ValidateSchema(context.Background(), schemaID, []byte(`{"key": "value"}`))
	require.NoError(t, err)

	// Invalid JSON
	err = registry.ValidateSchema(context.Background(), schemaID, []byte(`invalid`))
	assert.Error(t, err)
}

func TestFilterRules(t *testing.T) {
	service := &Service{}

	event := &StreamEvent{
		Value: json.RawMessage(`{"type": "order", "status": "completed", "amount": 100}`),
	}

	tests := []struct {
		name     string
		rules    []FilterRule
		expected bool
	}{
		{
			name:     "No rules",
			rules:    nil,
			expected: true,
		},
		{
			name: "Equals match",
			rules: []FilterRule{
				{Field: "type", Operator: "eq", Value: "order"},
			},
			expected: true,
		},
		{
			name: "Equals no match",
			rules: []FilterRule{
				{Field: "type", Operator: "eq", Value: "payment"},
			},
			expected: false,
		},
		{
			name: "Not equals match",
			rules: []FilterRule{
				{Field: "type", Operator: "neq", Value: "payment"},
			},
			expected: true,
		},
		{
			name: "Exists match",
			rules: []FilterRule{
				{Field: "status", Operator: "exists"},
			},
			expected: true,
		},
		{
			name: "Exists no match",
			rules: []FilterRule{
				{Field: "missing", Operator: "exists"},
			},
			expected: false,
		},
		{
			name: "Contains match",
			rules: []FilterRule{
				{Field: "status", Operator: "contains", Value: "complete"},
			},
			expected: true,
		},
		{
			name: "Multiple rules all match",
			rules: []FilterRule{
				{Field: "type", Operator: "eq", Value: "order"},
				{Field: "status", Operator: "eq", Value: "completed"},
			},
			expected: true,
		},
		{
			name: "Multiple rules partial match",
			rules: []FilterRule{
				{Field: "type", Operator: "eq", Value: "order"},
				{Field: "status", Operator: "eq", Value: "pending"},
			},
			expected: false,
		},
		{
			name: "Negated rule",
			rules: []FilterRule{
				{Field: "type", Operator: "eq", Value: "order", Negate: true},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.passesFilters(event, tt.rules)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCreateRequest(t *testing.T) {
	service := &Service{config: DefaultServiceConfig()}

	tests := []struct {
		name    string
		req     *CreateBridgeRequest
		wantErr bool
	}{
		{
			name:    "Missing name",
			req:     &CreateBridgeRequest{},
			wantErr: true,
		},
		{
			name: "Invalid stream type",
			req: &CreateBridgeRequest{
				Name:       "test",
				StreamType: "invalid",
				Direction:  DirectionOutbound,
			},
			wantErr: true,
		},
		{
			name: "Missing config",
			req: &CreateBridgeRequest{
				Name:       "test",
				StreamType: StreamTypeKafka,
				Direction:  DirectionOutbound,
			},
			wantErr: true,
		},
		{
			name: "Valid Kafka config",
			req: &CreateBridgeRequest{
				Name:       "test",
				StreamType: StreamTypeKafka,
				Direction:  DirectionOutbound,
				Config: &BridgeConfig{
					KafkaConfig: &KafkaConfig{
						Brokers: []string{"localhost:9092"},
						Topic:   "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Kafka missing topic",
			req: &CreateBridgeRequest{
				Name:       "test",
				StreamType: StreamTypeKafka,
				Direction:  DirectionOutbound,
				Config: &BridgeConfig{
					KafkaConfig: &KafkaConfig{
						Brokers: []string{"localhost:9092"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Kinesis config",
			req: &CreateBridgeRequest{
				Name:       "test",
				StreamType: StreamTypeKinesis,
				Direction:  DirectionOutbound,
				Config: &BridgeConfig{
					KinesisConfig: &KinesisConfig{
						StreamName: "test-stream",
						Region:     "us-east-1",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCreateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

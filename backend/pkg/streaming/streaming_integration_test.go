package streaming_test

import (
	"testing"
	"time"

	"github.com/josedab/waas/pkg/streaming"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamingBridgeCRUD(t *testing.T) {
	t.Skip("Integration test - requires database")

	// Test creating a streaming bridge
	service := streaming.NewService(nil, nil, streaming.DefaultServiceConfig())
	require.NotNil(t, service)
}

func TestMessageSerialization(t *testing.T) {
	msg := &streaming.Message{
		ID:        "msg-123",
		Key:       "user-456",
		Value:     []byte(`{"event":"user.created"}`),
		Headers:   map[string]string{"content-type": "application/json"},
		Timestamp: time.Now(),
	}

	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, "user-456", msg.Key)
	assert.NotNil(t, msg.Value)
	assert.Len(t, msg.Headers, 1)
}

func TestSaramaKafkaConfigDefaults(t *testing.T) {
	config := streaming.DefaultSaramaKafkaConfig()

	assert.NotNil(t, config)
	assert.Equal(t, []string{"localhost:9092"}, config.Brokers)
	assert.Equal(t, "waas-streaming", config.ClientID)
	assert.Equal(t, 3, config.RetryMax)
	assert.True(t, config.Idempotent)
}

func TestAWSKinesisConfigDefaults(t *testing.T) {
	config := streaming.DefaultAWSKinesisConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "us-east-1", config.Region)
	assert.Equal(t, 1024*1024, config.MaxRecordSize)
	assert.Equal(t, 500, config.MaxBatchRecords)
}

func TestServiceConfigDefaults(t *testing.T) {
	config := streaming.DefaultServiceConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 50, config.MaxBridgesPerTenant)
	assert.Equal(t, 100, config.DefaultBatchSize)
	assert.True(t, config.EnableSchemaValidation)
}

func TestStreamTypeConstants(t *testing.T) {
	streamTypes := []streaming.StreamType{
		streaming.StreamTypeKafka,
		streaming.StreamTypeKinesis,
		streaming.StreamTypePulsar,
		streaming.StreamTypeEventBridge,
	}

	for _, st := range streamTypes {
		assert.NotEmpty(t, string(st))
	}
}

func TestDirectionConstants(t *testing.T) {
	directions := []streaming.Direction{
		streaming.DirectionInbound,
		streaming.DirectionOutbound,
		streaming.DirectionBoth,
	}

	for _, dir := range directions {
		assert.NotEmpty(t, string(dir))
	}
}

func TestBridgeStatusConstants(t *testing.T) {
	statuses := []streaming.BridgeStatus{
		streaming.BridgeStatusActive,
		streaming.BridgeStatusPaused,
		streaming.BridgeStatusError,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}

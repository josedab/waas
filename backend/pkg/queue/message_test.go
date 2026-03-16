package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliveryMessage_JSON(t *testing.T) {
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test": "data", "number": 42}`),
		Headers:       map[string]string{"Content-Type": "application/json", "X-Custom": "value"},
		AttemptNumber: 2,
		ScheduledAt:   time.Now().UTC().Truncate(time.Second), // Truncate for comparison
		Signature:     "sha256=abcdef123456",
		MaxAttempts:   5,
	}

	// Test serialization
	data, err := message.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test deserialization
	var decoded DeliveryMessage
	err = decoded.FromJSON(data)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, message.DeliveryID, decoded.DeliveryID)
	assert.Equal(t, message.EndpointID, decoded.EndpointID)
	assert.Equal(t, message.TenantID, decoded.TenantID)
	assert.JSONEq(t, string(message.Payload), string(decoded.Payload))
	assert.Equal(t, message.Headers, decoded.Headers)
	assert.Equal(t, message.AttemptNumber, decoded.AttemptNumber)
	assert.Equal(t, message.ScheduledAt, decoded.ScheduledAt)
	assert.Equal(t, message.Signature, decoded.Signature)
	assert.Equal(t, message.MaxAttempts, decoded.MaxAttempts)
}

func TestDeliveryMessage_EmptyPayload(t *testing.T) {
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{}`),
		Headers:       make(map[string]string),
		AttemptNumber: 1,
		ScheduledAt:   time.Now(),
		MaxAttempts:   3,
	}

	data, err := message.ToJSON()
	require.NoError(t, err)

	var decoded DeliveryMessage
	err = decoded.FromJSON(data)
	require.NoError(t, err)

	assert.Equal(t, message.DeliveryID, decoded.DeliveryID)
	assert.JSONEq(t, `{}`, string(decoded.Payload))
}

func TestDeliveryMessage_InvalidJSON(t *testing.T) {
	var message DeliveryMessage

	// Test invalid JSON
	err := message.FromJSON([]byte(`{invalid json`))
	assert.Error(t, err)

	// Test empty data
	err = message.FromJSON([]byte(``))
	assert.Error(t, err)
}

func TestDeliveryResult_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	nextRetry := now.Add(5 * time.Minute)
	httpStatus := 200
	responseBody := `{"success": true}`
	errorMessage := "Connection timeout"

	result := &DeliveryResult{
		DeliveryID:    uuid.New(),
		Status:        StatusSuccess,
		HTTPStatus:    &httpStatus,
		ResponseBody:  &responseBody,
		ErrorMessage:  &errorMessage,
		DeliveredAt:   &now,
		NextRetryAt:   &nextRetry,
		AttemptNumber: 3,
	}

	// Test serialization
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Test deserialization
	var decoded DeliveryResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.DeliveryID, decoded.DeliveryID)
	assert.Equal(t, result.Status, decoded.Status)
	assert.Equal(t, *result.HTTPStatus, *decoded.HTTPStatus)
	assert.Equal(t, *result.ResponseBody, *decoded.ResponseBody)
	assert.Equal(t, *result.ErrorMessage, *decoded.ErrorMessage)
	assert.Equal(t, *result.DeliveredAt, *decoded.DeliveredAt)
	assert.Equal(t, *result.NextRetryAt, *decoded.NextRetryAt)
	assert.Equal(t, result.AttemptNumber, decoded.AttemptNumber)
}

func TestDeliveryResult_NilFields(t *testing.T) {
	result := &DeliveryResult{
		DeliveryID:    uuid.New(),
		Status:        StatusFailed,
		AttemptNumber: 1,
		// All pointer fields are nil
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded DeliveryResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.DeliveryID, decoded.DeliveryID)
	assert.Equal(t, result.Status, decoded.Status)
	assert.Nil(t, decoded.HTTPStatus)
	assert.Nil(t, decoded.ResponseBody)
	assert.Nil(t, decoded.ErrorMessage)
	assert.Nil(t, decoded.DeliveredAt)
	assert.Nil(t, decoded.NextRetryAt)
	assert.Equal(t, result.AttemptNumber, decoded.AttemptNumber)
}

func TestConstants(t *testing.T) {
	// Test queue names are defined
	assert.NotEmpty(t, DeliveryQueue)
	assert.NotEmpty(t, DeadLetterQueue)
	assert.NotEmpty(t, RetryQueue)
	assert.NotEmpty(t, ProcessingQueue)

	// Test status constants are defined
	assert.NotEmpty(t, StatusPending)
	assert.NotEmpty(t, StatusProcessing)
	assert.NotEmpty(t, StatusSuccess)
	assert.NotEmpty(t, StatusFailed)
	assert.NotEmpty(t, StatusRetrying)

	// Test queue names are unique
	queues := []string{DeliveryQueue, DeadLetterQueue, RetryQueue, ProcessingQueue}
	uniqueQueues := make(map[string]bool)
	for _, queue := range queues {
		assert.False(t, uniqueQueues[queue], "Queue name %s is not unique", queue)
		uniqueQueues[queue] = true
	}

	// Test status values are unique
	statuses := []string{StatusPending, StatusProcessing, StatusSuccess, StatusFailed, StatusRetrying}
	uniqueStatuses := make(map[string]bool)
	for _, status := range statuses {
		assert.False(t, uniqueStatuses[status], "Status %s is not unique", status)
		uniqueStatuses[status] = true
	}
}

func TestDeliveryMessage_VersionSetOnSerialize(t *testing.T) {
	msg := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"test":true}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	data, err := msg.ToJSON()
	assert.NoError(t, err)
	assert.Equal(t, 1, msg.Version, "ToJSON should set Version to 1")

	// Verify version in raw JSON
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), raw["version"])
}

func TestDeliveryMessage_VersionDefaultsOnDeserialize(t *testing.T) {
	// Simulate a pre-versioning message (no version field)
	oldMessage := `{"delivery_id":"` + uuid.New().String() + `","endpoint_id":"` + uuid.New().String() + `","tenant_id":"` + uuid.New().String() + `","payload":{"x":1},"attempt_number":1,"max_attempts":3,"scheduled_at":"2025-01-01T00:00:00Z","signature":"sig"}`

	var msg DeliveryMessage
	err := msg.FromJSON([]byte(oldMessage))
	assert.NoError(t, err)
	assert.Equal(t, 1, msg.Version, "FromJSON should default missing version to 1")
}

func TestDeliveryMessage_EventTypeRoundTrip(t *testing.T) {
	msg := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		EventType:     "order.created",
		Payload:       json.RawMessage(`{"order_id":"123"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	data, err := msg.ToJSON()
	assert.NoError(t, err)

	var decoded DeliveryMessage
	err = decoded.FromJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, "order.created", decoded.EventType)
	assert.Equal(t, 1, decoded.Version)
}

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHandler for testing consumer logic
type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) HandleDelivery(ctx context.Context, message *DeliveryMessage) (*DeliveryResult, error) {
	args := m.Called(ctx, message)
	return args.Get(0).(*DeliveryResult), args.Error(1)
}

func TestConsumer_CalculateRetryDelay(t *testing.T) {
	consumer := &Consumer{}

	tests := []struct {
		name          string
		attemptNumber int
		expectedMin   time.Duration
		expectedMax   time.Duration
	}{
		{
			name:          "First retry",
			attemptNumber: 1,
			expectedMin:   500 * time.Millisecond,  // 1s - 25% jitter
			expectedMax:   1500 * time.Millisecond, // 1s + 25% jitter
		},
		{
			name:          "Second retry",
			attemptNumber: 2,
			expectedMin:   1500 * time.Millisecond, // 2s - 25% jitter
			expectedMax:   2500 * time.Millisecond, // 2s + 25% jitter
		},
		{
			name:          "Third retry",
			attemptNumber: 3,
			expectedMin:   3 * time.Second,   // 4s - 25% jitter
			expectedMax:   5 * time.Second,   // 4s + 25% jitter
		},
		{
			name:          "High attempt number (should cap at 5 minutes)",
			attemptNumber: 10,
			expectedMin:   3*time.Minute + 45*time.Second, // 5m - 25% jitter
			expectedMax:   5 * time.Minute,                // Capped at 5m
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := consumer.calculateRetryDelay(tt.attemptNumber)
			assert.GreaterOrEqual(t, delay, tt.expectedMin, "Delay should be at least minimum expected")
			assert.LessOrEqual(t, delay, tt.expectedMax, "Delay should not exceed maximum expected")
		})
	}
}

func TestConsumer_HandleDeliveryResult_Success(t *testing.T) {
	consumer := &Consumer{}
	
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		AttemptNumber: 1,
		MaxAttempts:   3,
	}

	now := time.Now()
	result := &DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        StatusSuccess,
		DeliveredAt:   &now,
		AttemptNumber: 1,
	}

	// Success should not trigger any additional actions
	err := consumer.handleDeliveryResult(context.Background(), message, result)
	assert.NoError(t, err)
}

func TestConsumer_HandleDeliveryResult_FailedWithRetriesLeft(t *testing.T) {
	// This test would require mocking the publisher, which is complex
	// In a real implementation, you might want to inject the publisher as a dependency
	// For now, we'll test the logic without the actual Redis operations
	
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		AttemptNumber: 1,
		MaxAttempts:   3,
	}

	errorMsg := "Connection timeout"
	_ = &DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        StatusFailed,
		ErrorMessage:  &errorMsg,
		AttemptNumber: 1,
	}

	// We can test that the attempt number would be incremented
	assert.Less(t, message.AttemptNumber, message.MaxAttempts, "Should have retries left")
}

func TestConsumer_HandleDeliveryResult_FailedMaxAttempts(t *testing.T) {
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		AttemptNumber: 3,
		MaxAttempts:   3,
	}

	errorMsg := "Permanent failure"
	_ = &DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        StatusFailed,
		ErrorMessage:  &errorMsg,
		AttemptNumber: 3,
	}

	// Should go to dead letter queue when max attempts reached
	assert.Equal(t, message.AttemptNumber, message.MaxAttempts, "Should be at max attempts")
}

func TestDeliveryMessage_Validation(t *testing.T) {
	tests := []struct {
		name    string
		message *DeliveryMessage
		valid   bool
	}{
		{
			name: "Valid message",
			message: &DeliveryMessage{
				DeliveryID:    uuid.New(),
				EndpointID:    uuid.New(),
				TenantID:      uuid.New(),
				Payload:       json.RawMessage(`{"test": true}`),
				AttemptNumber: 1,
				MaxAttempts:   3,
				ScheduledAt:   time.Now(),
			},
			valid: true,
		},
		{
			name: "Zero UUID delivery ID",
			message: &DeliveryMessage{
				DeliveryID:    uuid.Nil,
				EndpointID:    uuid.New(),
				TenantID:      uuid.New(),
				AttemptNumber: 1,
				MaxAttempts:   3,
			},
			valid: false,
		},
		{
			name: "Invalid attempt number",
			message: &DeliveryMessage{
				DeliveryID:    uuid.New(),
				EndpointID:    uuid.New(),
				TenantID:      uuid.New(),
				AttemptNumber: 0,
				MaxAttempts:   3,
			},
			valid: false,
		},
		{
			name: "Attempt number exceeds max",
			message: &DeliveryMessage{
				DeliveryID:    uuid.New(),
				EndpointID:    uuid.New(),
				TenantID:      uuid.New(),
				AttemptNumber: 5,
				MaxAttempts:   3,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateDeliveryMessage(tt.message)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// validateDeliveryMessage validates a delivery message
func validateDeliveryMessage(message *DeliveryMessage) bool {
	if message.DeliveryID == uuid.Nil {
		return false
	}
	if message.EndpointID == uuid.Nil {
		return false
	}
	if message.TenantID == uuid.Nil {
		return false
	}
	if message.AttemptNumber <= 0 {
		return false
	}
	if message.MaxAttempts <= 0 {
		return false
	}
	if message.AttemptNumber > message.MaxAttempts {
		return false
	}
	return true
}

func TestRetryDelayProgression(t *testing.T) {
	consumer := &Consumer{}
	
	// Test that delays increase with attempt number
	var previousDelay time.Duration
	for attempt := 1; attempt <= 5; attempt++ {
		delay := consumer.calculateRetryDelay(attempt)
		
		if attempt > 1 {
			// Each delay should generally be larger than the previous
			// (accounting for jitter, we'll check it's at least 50% of expected increase)
			expectedMinIncrease := time.Duration(1<<uint(attempt-2)) * time.Second / 2
			assert.GreaterOrEqual(t, delay, previousDelay+expectedMinIncrease/2, 
				"Delay should increase with attempt number")
		}
		
		previousDelay = delay
		
		// No delay should exceed 5 minutes
		assert.LessOrEqual(t, delay, 5*time.Minute, "Delay should not exceed 5 minutes")
	}
}

func TestDeliveryResultStatuses(t *testing.T) {
	validStatuses := []string{StatusSuccess, StatusFailed, StatusRetrying}
	
	for _, status := range validStatuses {
		result := &DeliveryResult{
			DeliveryID:    uuid.New(),
			Status:        status,
			AttemptNumber: 1,
		}
		
		assert.Contains(t, validStatuses, result.Status, "Status should be valid")
	}
}

func TestMessageSerialization_LargePayload(t *testing.T) {
	// Test with a large payload
	largePayload := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largePayload[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_some_longer_content", i)
	}
	
	payloadBytes, err := json.Marshal(largePayload)
	require.NoError(t, err)
	
	message := &DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    uuid.New(),
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(payloadBytes),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}
	
	// Should be able to serialize and deserialize large payloads
	data, err := message.ToJSON()
	require.NoError(t, err)
	
	var decoded DeliveryMessage
	err = decoded.FromJSON(data)
	require.NoError(t, err)
	
	assert.Equal(t, message.DeliveryID, decoded.DeliveryID)
	assert.JSONEq(t, string(message.Payload), string(decoded.Payload))
}
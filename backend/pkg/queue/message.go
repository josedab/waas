package queue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DeliveryMessage represents a webhook delivery message in the queue
type DeliveryMessage struct {
	Version       int               `json:"version"`
	DeliveryID    uuid.UUID         `json:"delivery_id"`
	EndpointID    uuid.UUID         `json:"endpoint_id"`
	TenantID      uuid.UUID         `json:"tenant_id"`
	EventType     string            `json:"event_type,omitempty"`
	Payload       json.RawMessage   `json:"payload"`
	Headers       map[string]string `json:"headers"`
	AttemptNumber int               `json:"attempt_number"`
	ScheduledAt   time.Time         `json:"scheduled_at"`
	Signature     string            `json:"signature"`
	MaxAttempts   int               `json:"max_attempts"`
}

const currentMessageVersion = 1

// ToJSON serializes the message to JSON, setting the version field.
func (dm *DeliveryMessage) ToJSON() ([]byte, error) {
	dm.Version = currentMessageVersion
	return json.Marshal(dm)
}

// FromJSON deserializes JSON to DeliveryMessage. Messages without a version
// field (pre-versioning) are treated as version 1.
func (dm *DeliveryMessage) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, dm); err != nil {
		return err
	}
	if dm.Version == 0 {
		dm.Version = 1
	}
	return nil
}

// DeliveryResult represents the result of a webhook delivery attempt
type DeliveryResult struct {
	DeliveryID    uuid.UUID  `json:"delivery_id"`
	Status        string     `json:"status"` // success, failed, retrying
	HTTPStatus    *int       `json:"http_status,omitempty"`
	ResponseBody  *string    `json:"response_body,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty"`
	AttemptNumber int        `json:"attempt_number"`
}

// QueueNames defines the Redis queue names
const (
	DeliveryQueue   = "webhook:delivery"
	DeadLetterQueue = "webhook:dlq"
	RetryQueue      = "webhook:retry"
	ProcessingQueue = "webhook:processing"
)

// MessageStatus constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusRetrying   = "retrying"
)

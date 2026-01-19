package models

import (
	"time"
	"github.com/google/uuid"
)

type WebhookEndpoint struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	TenantID      uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	URL           string                 `json:"url" db:"url"`
	SecretHash    string                 `json:"-" db:"secret_hash"`
	IsActive      bool                   `json:"is_active" db:"is_active"`
	RetryConfig   RetryConfiguration     `json:"retry_config" db:"retry_config"`
	CustomHeaders map[string]string      `json:"custom_headers" db:"custom_headers"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

type RetryConfiguration struct {
	MaxAttempts        int `json:"max_attempts"`
	InitialDelayMs     int `json:"initial_delay_ms"`
	MaxDelayMs         int `json:"max_delay_ms"`
	BackoffMultiplier  int `json:"backoff_multiplier"`
}

type DeliveryRequest struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	EndpointID  uuid.UUID         `json:"endpoint_id" db:"endpoint_id"`
	Payload     interface{}       `json:"payload" db:"payload"`
	Headers     map[string]string `json:"headers" db:"headers"`
	ScheduledAt time.Time         `json:"scheduled_at" db:"scheduled_at"`
	Attempts    int               `json:"attempts" db:"attempts"`
	MaxAttempts int               `json:"max_attempts" db:"max_attempts"`
}

type DeliveryAttempt struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	EndpointID     uuid.UUID  `json:"endpoint_id" db:"endpoint_id"`
	PayloadHash    string     `json:"payload_hash" db:"payload_hash"`
	PayloadSize    int        `json:"payload_size" db:"payload_size"`
	Status         string     `json:"status" db:"status"`
	HTTPStatus     *int       `json:"http_status" db:"http_status"`
	ResponseBody   *string    `json:"response_body" db:"response_body"`
	ErrorMessage   *string    `json:"error_message" db:"error_message"`
	AttemptNumber  int        `json:"attempt_number" db:"attempt_number"`
	ScheduledAt    time.Time  `json:"scheduled_at" db:"scheduled_at"`
	DeliveredAt    *time.Time `json:"delivered_at" db:"delivered_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}
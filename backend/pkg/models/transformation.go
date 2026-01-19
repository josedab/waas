package models

import (
	"time"

	"github.com/google/uuid"
)

// Transformation represents a payload transformation script
type Transformation struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	TenantID    uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description,omitempty" db:"description"`
	Script      string            `json:"script" db:"script"`
	Enabled     bool              `json:"enabled" db:"enabled"`
	IsActive    bool              `json:"is_active" db:"is_active"` // Alias for Enabled used in delivery
	Version     int               `json:"version" db:"version"`
	Config      TransformConfig   `json:"config" db:"config"`
	EnableLogging bool            `json:"-"` // Convenience accessor
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// TransformConfig holds configuration for transformation execution
type TransformConfig struct {
	TimeoutMs     int  `json:"timeout_ms"`      // Max execution time
	MaxMemoryMB   int  `json:"max_memory_mb"`   // Max memory usage
	AllowHTTP     bool `json:"allow_http"`      // Allow HTTP requests in scripts
	EnableLogging bool `json:"enable_logging"`  // Enable script logging
}

// TransformationLog records transformation execution
type TransformationLog struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TransformationID uuid.UUID  `json:"transformation_id" db:"transformation_id"`
	EndpointID       *uuid.UUID `json:"endpoint_id,omitempty" db:"endpoint_id"`
	DeliveryID       *uuid.UUID `json:"delivery_id,omitempty" db:"delivery_id"`
	InputPayload     string     `json:"input_payload" db:"input_payload"`
	OutputPayload    *string    `json:"output_payload,omitempty" db:"output_payload"`
	OutputPreview    *string    `json:"output_preview,omitempty" db:"output_preview"`
	Success          bool       `json:"success" db:"success"`
	ErrorMessage     *string    `json:"error_message,omitempty" db:"error_message"`
	ExecutionTimeMs  int64      `json:"execution_time_ms" db:"execution_time_ms"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// EndpointTransformation links transformations to endpoints
type EndpointTransformation struct {
	ID               uuid.UUID `json:"id" db:"id"`
	EndpointID       uuid.UUID `json:"endpoint_id" db:"endpoint_id"`
	TransformationID uuid.UUID `json:"transformation_id" db:"transformation_id"`
	Priority         int       `json:"priority" db:"priority"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// DefaultTransformConfig returns default transformation configuration
func DefaultTransformConfig() TransformConfig {
	return TransformConfig{
		TimeoutMs:     5000,   // 5 seconds
		MaxMemoryMB:   64,     // 64 MB
		AllowHTTP:     false,
		EnableLogging: true,
	}
}

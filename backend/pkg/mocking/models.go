package mocking

import (
	"time"
)

// MockEndpoint represents a mock webhook endpoint
type MockEndpoint struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description,omitempty" db:"description"`
	URL         string                 `json:"url" db:"url"`
	EventType   string                 `json:"event_type" db:"event_type"`
	Template    *PayloadTemplate       `json:"template,omitempty" db:"template"`
	Schedule    *MockSchedule          `json:"schedule,omitempty" db:"schedule"`
	Settings    MockSettings           `json:"settings" db:"settings"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// PayloadTemplate defines the structure for mock payloads
type PayloadTemplate struct {
	Type     string                 `json:"type"` // static, faker, template
	Content  map[string]interface{} `json:"content"`
	Fields   []TemplateField        `json:"fields,omitempty"`
	Includes []string               `json:"includes,omitempty"` // Include other templates
}

// TemplateField defines a dynamic field in a template
type TemplateField struct {
	Path    string       `json:"path"`
	Type    string       `json:"type"` // string, number, boolean, date, uuid, email, etc.
	Value   interface{}  `json:"value,omitempty"`
	Faker   string       `json:"faker,omitempty"` // Faker function name
	Options FieldOptions `json:"options,omitempty"`
}

// FieldOptions defines options for field generation
type FieldOptions struct {
	Min      float64  `json:"min,omitempty"`
	Max      float64  `json:"max,omitempty"`
	Length   int      `json:"length,omitempty"`
	Format   string   `json:"format,omitempty"`
	Choices  []string `json:"choices,omitempty"`
	Nullable bool     `json:"nullable,omitempty"`
	NullProb float64  `json:"null_prob,omitempty"` // Probability of null
}

// MockSchedule defines when to send mock webhooks
type MockSchedule struct {
	Type     string     `json:"type"`               // once, interval, cron
	Interval string     `json:"interval,omitempty"` // e.g., "5m", "1h"
	Cron     string     `json:"cron,omitempty"`
	StartAt  *time.Time `json:"start_at,omitempty"`
	EndAt    *time.Time `json:"end_at,omitempty"`
	MaxRuns  int        `json:"max_runs,omitempty"`
	RunCount int        `json:"run_count,omitempty"`
}

// MockSettings defines behavior settings
type MockSettings struct {
	Headers       map[string]string `json:"headers,omitempty"`
	DelayMs       int               `json:"delay_ms,omitempty"`
	BatchSize     int               `json:"batch_size,omitempty"`
	BatchInterval string            `json:"batch_interval,omitempty"`
	Signature     bool              `json:"signature,omitempty"`
	SignatureKey  string            `json:"signature_key,omitempty"`
}

// MockDelivery represents a mock webhook delivery
type MockDelivery struct {
	ID           string                 `json:"id" db:"id"`
	EndpointID   string                 `json:"endpoint_id" db:"endpoint_id"`
	TenantID     string                 `json:"tenant_id" db:"tenant_id"`
	Payload      map[string]interface{} `json:"payload" db:"payload"`
	Headers      map[string]string      `json:"headers" db:"headers"`
	Status       string                 `json:"status" db:"status"`
	StatusCode   int                    `json:"status_code,omitempty" db:"status_code"`
	ResponseBody string                 `json:"response_body,omitempty" db:"response_body"`
	Error        string                 `json:"error,omitempty" db:"error"`
	LatencyMs    int                    `json:"latency_ms" db:"latency_ms"`
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty" db:"scheduled_at"`
	SentAt       *time.Time             `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// MockTemplate represents a reusable mock template
type MockTemplate struct {
	ID          string                   `json:"id" db:"id"`
	TenantID    string                   `json:"tenant_id" db:"tenant_id"`
	Name        string                   `json:"name" db:"name"`
	Description string                   `json:"description,omitempty" db:"description"`
	EventType   string                   `json:"event_type" db:"event_type"`
	Category    string                   `json:"category,omitempty" db:"category"`
	Template    PayloadTemplate          `json:"template" db:"template"`
	Examples    []map[string]interface{} `json:"examples,omitempty" db:"examples"`
	IsPublic    bool                     `json:"is_public" db:"is_public"`
	CreatedAt   time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at" db:"updated_at"`
}

// CreateMockEndpointRequest represents a request to create a mock endpoint
type CreateMockEndpointRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description,omitempty"`
	URL         string                 `json:"url" binding:"required,url"`
	EventType   string                 `json:"event_type" binding:"required"`
	Template    *PayloadTemplate       `json:"template,omitempty"`
	Schedule    *MockSchedule          `json:"schedule,omitempty"`
	Settings    MockSettings           `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMockEndpointRequest represents a request to update a mock endpoint
type UpdateMockEndpointRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	URL         *string                `json:"url,omitempty"`
	EventType   *string                `json:"event_type,omitempty"`
	Template    *PayloadTemplate       `json:"template,omitempty"`
	Schedule    *MockSchedule          `json:"schedule,omitempty"`
	Settings    *MockSettings          `json:"settings,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TriggerMockRequest represents a request to trigger mock deliveries
type TriggerMockRequest struct {
	Count    int                    `json:"count,omitempty"`   // Number of mocks to send
	Payload  map[string]interface{} `json:"payload,omitempty"` // Custom payload override
	DelayMs  int                    `json:"delay_ms,omitempty"`
	Interval string                 `json:"interval,omitempty"` // Delay between multiple
}

// CreateTemplateRequest represents a request to create a mock template
type CreateTemplateRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description,omitempty"`
	EventType   string                   `json:"event_type" binding:"required"`
	Category    string                   `json:"category,omitempty"`
	Template    PayloadTemplate          `json:"template" binding:"required"`
	Examples    []map[string]interface{} `json:"examples,omitempty"`
	IsPublic    bool                     `json:"is_public,omitempty"`
}

// FakerType represents available faker types
type FakerType string

const (
	FakerUUID       FakerType = "uuid"
	FakerEmail      FakerType = "email"
	FakerName       FakerType = "name"
	FakerFirstName  FakerType = "first_name"
	FakerLastName   FakerType = "last_name"
	FakerPhone      FakerType = "phone"
	FakerAddress    FakerType = "address"
	FakerCity       FakerType = "city"
	FakerCountry    FakerType = "country"
	FakerCompany    FakerType = "company"
	FakerURL        FakerType = "url"
	FakerIPv4       FakerType = "ipv4"
	FakerIPv6       FakerType = "ipv6"
	FakerTimestamp  FakerType = "timestamp"
	FakerDate       FakerType = "date"
	FakerNumber     FakerType = "number"
	FakerFloat      FakerType = "float"
	FakerBoolean    FakerType = "boolean"
	FakerWord       FakerType = "word"
	FakerSentence   FakerType = "sentence"
	FakerParagraph  FakerType = "paragraph"
	FakerCreditCard FakerType = "credit_card"
	FakerCurrency   FakerType = "currency"
	FakerPrice      FakerType = "price"
	FakerUsername   FakerType = "username"
	FakerPassword   FakerType = "password"
	FakerSlug       FakerType = "slug"
	FakerHexColor   FakerType = "hex_color"
	FakerUserAgent  FakerType = "user_agent"
)

// GetAvailableFakerTypes returns all available faker types
func GetAvailableFakerTypes() []FakerType {
	return []FakerType{
		FakerUUID, FakerEmail, FakerName, FakerFirstName, FakerLastName,
		FakerPhone, FakerAddress, FakerCity, FakerCountry, FakerCompany,
		FakerURL, FakerIPv4, FakerIPv6, FakerTimestamp, FakerDate,
		FakerNumber, FakerFloat, FakerBoolean, FakerWord, FakerSentence,
		FakerParagraph, FakerCreditCard, FakerCurrency, FakerPrice,
		FakerUsername, FakerPassword, FakerSlug, FakerHexColor, FakerUserAgent,
	}
}

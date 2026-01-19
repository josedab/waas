package models

import (
	"time"

	"github.com/google/uuid"
)

// GraphQL Schema status constants
const (
	GraphQLSchemaStatusActive   = "active"
	GraphQLSchemaStatusInactive = "inactive"
	GraphQLSchemaStatusDraft    = "draft"
)

// GraphQL Subscription status constants
const (
	GraphQLSubscriptionActive  = "active"
	GraphQLSubscriptionPaused  = "paused"
	GraphQLSubscriptionDeleted = "deleted"
)

// Federation health status constants
const (
	FederationHealthUnknown   = "unknown"
	FederationHealthHealthy   = "healthy"
	FederationHealthDegraded  = "degraded"
	FederationHealthUnhealthy = "unhealthy"
)

// GraphQLSchema represents a registered GraphQL schema
type GraphQLSchema struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	TenantID              uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name                  string     `json:"name" db:"name"`
	Description           string     `json:"description,omitempty" db:"description"`
	SchemaSDL             string     `json:"schema_sdl" db:"schema_sdl"`
	Version               string     `json:"version" db:"version"`
	Status                string     `json:"status" db:"status"`
	IntrospectionEndpoint string     `json:"introspection_endpoint,omitempty" db:"introspection_endpoint"`
	FederationEnabled     bool       `json:"federation_enabled" db:"federation_enabled"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// GraphQLSubscription represents a subscription that delivers to webhooks
type GraphQLSubscription struct {
	ID                uuid.UUID               `json:"id" db:"id"`
	TenantID          uuid.UUID               `json:"tenant_id" db:"tenant_id"`
	SchemaID          uuid.UUID               `json:"schema_id" db:"schema_id"`
	EndpointID        uuid.UUID               `json:"endpoint_id" db:"endpoint_id"`
	Name              string                  `json:"name" db:"name"`
	Description       string                  `json:"description,omitempty" db:"description"`
	SubscriptionQuery string                  `json:"subscription_query" db:"subscription_query"`
	Variables         map[string]interface{}  `json:"variables" db:"variables"`
	FilterExpression  string                  `json:"filter_expression,omitempty" db:"filter_expression"`
	FieldSelection    []string                `json:"field_selection" db:"field_selection"`
	TransformJS       string                  `json:"transform_js,omitempty" db:"transform_js"`
	Status            string                  `json:"status" db:"status"`
	DeliveryConfig    *SubscriptionDelivery   `json:"delivery_config" db:"delivery_config"`
	CreatedAt         time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at" db:"updated_at"`
}

// SubscriptionDelivery contains delivery configuration for subscriptions
type SubscriptionDelivery struct {
	BatchSize       int    `json:"batch_size,omitempty"`
	BatchWindowMS   int    `json:"batch_window_ms,omitempty"`
	MaxRetries      int    `json:"max_retries,omitempty"`
	RetryDelayMS    int    `json:"retry_delay_ms,omitempty"`
	DeduplicationID string `json:"deduplication_id,omitempty"`
}

// GraphQLSubscriptionEvent represents an event received via subscription
type GraphQLSubscriptionEvent struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	SubscriptionID  uuid.UUID              `json:"subscription_id" db:"subscription_id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EventType       string                 `json:"event_type" db:"event_type"`
	Payload         map[string]interface{} `json:"payload" db:"payload"`
	FilteredPayload map[string]interface{} `json:"filtered_payload,omitempty" db:"filtered_payload"`
	Delivered       bool                   `json:"delivered" db:"delivered"`
	DeliveryID      *uuid.UUID             `json:"delivery_id,omitempty" db:"delivery_id"`
	ReceivedAt      time.Time              `json:"received_at" db:"received_at"`
	ProcessedAt     *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
}

// GraphQLFederationSource represents a federated subgraph source
type GraphQLFederationSource struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	SchemaID        uuid.UUID              `json:"schema_id" db:"schema_id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name            string                 `json:"name" db:"name"`
	EndpointURL     string                 `json:"endpoint_url" db:"endpoint_url"`
	SubgraphSDL     string                 `json:"subgraph_sdl,omitempty" db:"subgraph_sdl"`
	AuthConfig      map[string]interface{} `json:"auth_config" db:"auth_config"`
	HealthStatus    string                 `json:"health_status" db:"health_status"`
	LastHealthCheck *time.Time             `json:"last_health_check,omitempty" db:"last_health_check"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// GraphQLTypeMapping maps GraphQL types to webhook event types
type GraphQLTypeMapping struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	SchemaID         uuid.UUID              `json:"schema_id" db:"schema_id"`
	TenantID         uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	GraphQLType      string                 `json:"graphql_type" db:"graphql_type"`
	WebhookEventType string                 `json:"webhook_event_type" db:"webhook_event_type"`
	FieldMappings    map[string]string      `json:"field_mappings" db:"field_mappings"`
	AutoGenerated    bool                   `json:"auto_generated" db:"auto_generated"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// CreateGraphQLSchemaRequest represents a request to create a GraphQL schema
type CreateGraphQLSchemaRequest struct {
	Name                  string `json:"name" binding:"required"`
	Description           string `json:"description"`
	SchemaSDL             string `json:"schema_sdl" binding:"required"`
	Version               string `json:"version"`
	IntrospectionEndpoint string `json:"introspection_endpoint"`
	FederationEnabled     bool   `json:"federation_enabled"`
}

// CreateGraphQLSubscriptionRequest represents a request to create a subscription
type CreateGraphQLSubscriptionRequest struct {
	SchemaID          string                 `json:"schema_id" binding:"required"`
	EndpointID        string                 `json:"endpoint_id" binding:"required"`
	Name              string                 `json:"name" binding:"required"`
	Description       string                 `json:"description"`
	SubscriptionQuery string                 `json:"subscription_query" binding:"required"`
	Variables         map[string]interface{} `json:"variables"`
	FilterExpression  string                 `json:"filter_expression"`
	FieldSelection    []string               `json:"field_selection"`
	TransformJS       string                 `json:"transform_js"`
	DeliveryConfig    *SubscriptionDelivery  `json:"delivery_config"`
}

// AddFederationSourceRequest represents a request to add a federation source
type AddFederationSourceRequest struct {
	SchemaID    string                 `json:"schema_id" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	EndpointURL string                 `json:"endpoint_url" binding:"required"`
	AuthConfig  map[string]interface{} `json:"auth_config"`
}

// CreateTypeMappingRequest represents a request to create a type mapping
type CreateTypeMappingRequest struct {
	SchemaID         string            `json:"schema_id" binding:"required"`
	GraphQLType      string            `json:"graphql_type" binding:"required"`
	WebhookEventType string            `json:"webhook_event_type" binding:"required"`
	FieldMappings    map[string]string `json:"field_mappings"`
}

// GraphQL parsed types for schema analysis
type GraphQLParsedSchema struct {
	Types         []GraphQLTypeInfo         `json:"types"`
	Queries       []GraphQLOperationInfo    `json:"queries"`
	Mutations     []GraphQLOperationInfo    `json:"mutations"`
	Subscriptions []GraphQLSubscriptionInfo `json:"subscriptions"`
}

type GraphQLTypeInfo struct {
	Name   string                `json:"name"`
	Kind   string                `json:"kind"`
	Fields []GraphQLFieldInfo    `json:"fields,omitempty"`
}

type GraphQLFieldInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type GraphQLOperationInfo struct {
	Name      string               `json:"name"`
	Arguments []GraphQLArgumentInfo `json:"arguments"`
	ReturnType string              `json:"return_type"`
}

type GraphQLArgumentInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type GraphQLSubscriptionInfo struct {
	Name      string               `json:"name"`
	Arguments []GraphQLArgumentInfo `json:"arguments"`
	ReturnType string              `json:"return_type"`
	Description string             `json:"description,omitempty"`
}

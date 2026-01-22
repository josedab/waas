package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TestingService handles testing-related API operations
type TestingService struct {
	client *Client
}

// TestWebhookRequest represents a webhook test request
type TestWebhookRequest struct {
	URL     string            `json:"url"`
	Payload interface{}       `json:"payload"`
	Headers map[string]string `json:"headers,omitempty"`
	Method  string            `json:"method,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

// TestWebhookResponse represents the response from a webhook test
type TestWebhookResponse struct {
	TestID       uuid.UUID `json:"test_id"`
	URL          string    `json:"url"`
	Status       string    `json:"status"`
	HTTPStatus   *int      `json:"http_status,omitempty"`
	ResponseBody *string   `json:"response_body,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	Latency      *int64    `json:"latency_ms,omitempty"`
	RequestID    string    `json:"request_id"`
	TestedAt     time.Time `json:"tested_at"`
}

// CreateTestEndpointRequest represents a request to create a temporary test endpoint
type CreateTestEndpointRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	TTL         int               `json:"ttl,omitempty"`
}

// TestEndpointResponse represents a temporary test endpoint
type TestEndpointResponse struct {
	ID          uuid.UUID         `json:"id"`
	URL         string            `json:"url"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
}

// DeliveryInspectionResponse represents detailed delivery information for debugging
type DeliveryInspectionResponse struct {
	DeliveryID   uuid.UUID                `json:"delivery_id"`
	EndpointID   uuid.UUID                `json:"endpoint_id"`
	Status       string                   `json:"status"`
	AttemptNumber int                     `json:"attempt_number"`
	Request      *DeliveryRequestDetails  `json:"request"`
	Response     *DeliveryResponseDetails `json:"response,omitempty"`
	Timeline     []DeliveryTimelineEvent  `json:"timeline"`
	ErrorDetails *DeliveryErrorDetails    `json:"error_details,omitempty"`
}

// DeliveryRequestDetails represents request details for debugging
type DeliveryRequestDetails struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	PayloadHash string            `json:"payload_hash"`
	PayloadSize int               `json:"payload_size"`
	Signature   string            `json:"signature"`
	ScheduledAt time.Time         `json:"scheduled_at"`
}

// DeliveryResponseDetails represents response details for debugging
type DeliveryResponseDetails struct {
	HTTPStatus  int               `json:"http_status"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	BodySize    int               `json:"body_size"`
	DeliveredAt time.Time         `json:"delivered_at"`
	Latency     int64             `json:"latency_ms"`
}

// DeliveryTimelineEvent represents an event in the delivery timeline
type DeliveryTimelineEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Event       string                 `json:"event"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// DeliveryErrorDetails represents error details for debugging
type DeliveryErrorDetails struct {
	ErrorType    string     `json:"error_type"`
	ErrorMessage string     `json:"error_message"`
	HTTPStatus   *int       `json:"http_status,omitempty"`
	RetryCount   int        `json:"retry_count"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	Suggestions  []string   `json:"suggestions,omitempty"`
}

// TestWebhook tests a webhook endpoint with a custom payload
func (s *TestingService) TestWebhook(ctx context.Context, req *TestWebhookRequest) (*TestWebhookResponse, error) {
	var result TestWebhookResponse
	err := s.client.post(ctx, "/webhooks/test", req, &result)
	return &result, err
}

// CreateTestEndpoint creates a temporary test endpoint for webhook testing
func (s *TestingService) CreateTestEndpoint(ctx context.Context, req *CreateTestEndpointRequest) (*TestEndpointResponse, error) {
	var result TestEndpointResponse
	err := s.client.post(ctx, "/webhooks/test/endpoints", req, &result)
	return &result, err
}

// InspectDelivery provides detailed debugging information for a webhook delivery
func (s *TestingService) InspectDelivery(ctx context.Context, deliveryID uuid.UUID) (*DeliveryInspectionResponse, error) {
	var result DeliveryInspectionResponse
	path := fmt.Sprintf("/webhooks/deliveries/%s/inspect", deliveryID.String())
	err := s.client.get(ctx, path, &result)
	return &result, err
}

// GetDeliveryLogs retrieves logs for a specific delivery
func (s *TestingService) GetDeliveryLogs(ctx context.Context, deliveryID uuid.UUID) (map[string]interface{}, error) {
	var result map[string]interface{}
	path := fmt.Sprintf("/webhooks/deliveries/%s/logs", deliveryID.String())
	err := s.client.get(ctx, path, &result)
	return result, err
}
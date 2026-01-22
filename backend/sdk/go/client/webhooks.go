package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WebhookService handles webhook-related API operations
type WebhookService struct {
	client *Client
}

// WebhookEndpoint represents a webhook endpoint
type WebhookEndpoint struct {
	ID            uuid.UUID              `json:"id"`
	URL           string                 `json:"url"`
	Secret        string                 `json:"secret,omitempty"`
	IsActive      bool                   `json:"is_active"`
	RetryConfig   RetryConfiguration     `json:"retry_config"`
	CustomHeaders map[string]string      `json:"custom_headers"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// RetryConfiguration defines retry behavior for webhook deliveries
type RetryConfiguration struct {
	MaxAttempts       int `json:"max_attempts"`
	InitialDelayMs    int `json:"initial_delay_ms"`
	MaxDelayMs        int `json:"max_delay_ms"`
	BackoffMultiplier int `json:"backoff_multiplier"`
}

// CreateEndpointRequest represents a webhook endpoint creation request
type CreateEndpointRequest struct {
	URL           string                  `json:"url"`
	CustomHeaders map[string]string       `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfigurationReq  `json:"retry_config,omitempty"`
}

// RetryConfigurationReq represents retry configuration in requests
type RetryConfigurationReq struct {
	MaxAttempts       int `json:"max_attempts,omitempty"`
	InitialDelayMs    int `json:"initial_delay_ms,omitempty"`
	MaxDelayMs        int `json:"max_delay_ms,omitempty"`
	BackoffMultiplier int `json:"backoff_multiplier,omitempty"`
}

// UpdateEndpointRequest represents a webhook endpoint update request
type UpdateEndpointRequest struct {
	URL           *string                 `json:"url,omitempty"`
	CustomHeaders map[string]string       `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfigurationReq  `json:"retry_config,omitempty"`
	IsActive      *bool                   `json:"is_active,omitempty"`
}

// SendWebhookRequest represents a webhook send request
type SendWebhookRequest struct {
	EndpointID *uuid.UUID        `json:"endpoint_id,omitempty"`
	Payload    interface{}       `json:"payload"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// SendWebhookResponse represents the response for webhook send requests
type SendWebhookResponse struct {
	DeliveryID  uuid.UUID `json:"delivery_id"`
	EndpointID  uuid.UUID `json:"endpoint_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// BatchSendWebhookRequest represents a batch webhook send request
type BatchSendWebhookRequest struct {
	EndpointIDs []uuid.UUID       `json:"endpoint_ids,omitempty"`
	Payload     interface{}       `json:"payload"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// BatchSendWebhookResponse represents the response for batch webhook send requests
type BatchSendWebhookResponse struct {
	Deliveries []SendWebhookResponse `json:"deliveries"`
	Total      int                   `json:"total"`
	Queued     int                   `json:"queued"`
	Failed     int                   `json:"failed"`
}

// ListEndpointsOptions represents options for listing endpoints
type ListEndpointsOptions struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ListEndpointsResponse represents the response for listing endpoints
type ListEndpointsResponse struct {
	Endpoints  []WebhookEndpoint `json:"endpoints"`
	Pagination PaginationInfo    `json:"pagination"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

// CreateEndpoint creates a new webhook endpoint
func (s *WebhookService) CreateEndpoint(ctx context.Context, req *CreateEndpointRequest) (*WebhookEndpoint, error) {
	var result WebhookEndpoint
	err := s.client.post(ctx, "/webhooks/endpoints", req, &result)
	return &result, err
}

// GetEndpoint retrieves a webhook endpoint by ID
func (s *WebhookService) GetEndpoint(ctx context.Context, id uuid.UUID) (*WebhookEndpoint, error) {
	var result WebhookEndpoint
	path := fmt.Sprintf("/webhooks/endpoints/%s", id.String())
	err := s.client.get(ctx, path, &result)
	return &result, err
}

// ListEndpoints retrieves all webhook endpoints with optional pagination
func (s *WebhookService) ListEndpoints(ctx context.Context, opts *ListEndpointsOptions) (*ListEndpointsResponse, error) {
	path := "/webhooks/endpoints"
	
	if opts != nil {
		params := url.Values{}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", strconv.Itoa(opts.Offset))
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	var result ListEndpointsResponse
	err := s.client.get(ctx, path, &result)
	return &result, err
}

// UpdateEndpoint updates an existing webhook endpoint
func (s *WebhookService) UpdateEndpoint(ctx context.Context, id uuid.UUID, req *UpdateEndpointRequest) (*WebhookEndpoint, error) {
	var result WebhookEndpoint
	path := fmt.Sprintf("/webhooks/endpoints/%s", id.String())
	err := s.client.put(ctx, path, req, &result)
	return &result, err
}

// DeleteEndpoint deletes a webhook endpoint
func (s *WebhookService) DeleteEndpoint(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/webhooks/endpoints/%s", id.String())
	return s.client.delete(ctx, path)
}

// Send sends a webhook to one or all endpoints
func (s *WebhookService) Send(ctx context.Context, req *SendWebhookRequest) (*SendWebhookResponse, error) {
	var result SendWebhookResponse
	err := s.client.post(ctx, "/webhooks/send", req, &result)
	return &result, err
}

// BatchSend sends a webhook to multiple endpoints
func (s *WebhookService) BatchSend(ctx context.Context, req *BatchSendWebhookRequest) (*BatchSendWebhookResponse, error) {
	var result BatchSendWebhookResponse
	err := s.client.post(ctx, "/webhooks/send/batch", req, &result)
	return &result, err
}

// DeliveryHistoryOptions represents options for getting delivery history
type DeliveryHistoryOptions struct {
	EndpointIDs []uuid.UUID `json:"endpoint_ids,omitempty"`
	Statuses    []string    `json:"statuses,omitempty"`
	StartDate   *time.Time  `json:"start_date,omitempty"`
	EndDate     *time.Time  `json:"end_date,omitempty"`
	Limit       int         `json:"limit,omitempty"`
	Offset      int         `json:"offset,omitempty"`
}

// DeliveryAttempt represents a webhook delivery attempt
type DeliveryAttempt struct {
	ID            uuid.UUID  `json:"id"`
	EndpointID    uuid.UUID  `json:"endpoint_id"`
	PayloadHash   string     `json:"payload_hash"`
	PayloadSize   int        `json:"payload_size"`
	Status        string     `json:"status"`
	HTTPStatus    *int       `json:"http_status"`
	ResponseBody  *string    `json:"response_body,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	AttemptNumber int        `json:"attempt_number"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	DeliveredAt   *time.Time `json:"delivered_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DeliveryHistoryResponse represents the response for delivery history
type DeliveryHistoryResponse struct {
	Deliveries []DeliveryAttempt `json:"deliveries"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse represents pagination information in responses
type PaginationResponse struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}

// GetDeliveryHistory retrieves webhook delivery history with filtering
func (s *WebhookService) GetDeliveryHistory(ctx context.Context, opts *DeliveryHistoryOptions) (*DeliveryHistoryResponse, error) {
	path := "/webhooks/deliveries"
	
	if opts != nil {
		params := url.Values{}
		
		if len(opts.EndpointIDs) > 0 {
			var ids []string
			for _, id := range opts.EndpointIDs {
				ids = append(ids, id.String())
			}
			params.Set("endpoint_ids", strings.Join(ids, ","))
		}
		
		if len(opts.Statuses) > 0 {
			params.Set("statuses", strings.Join(opts.Statuses, ","))
		}
		
		if opts.StartDate != nil {
			params.Set("start_date", opts.StartDate.Format(time.RFC3339))
		}
		
		if opts.EndDate != nil {
			params.Set("end_date", opts.EndDate.Format(time.RFC3339))
		}
		
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		
		if opts.Offset > 0 {
			params.Set("offset", strconv.Itoa(opts.Offset))
		}
		
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	var result DeliveryHistoryResponse
	err := s.client.get(ctx, path, &result)
	return &result, err
}

// DeliveryDetails represents detailed delivery information
type DeliveryDetails struct {
	DeliveryID uuid.UUID       `json:"delivery_id"`
	Attempts   []DeliveryAttempt `json:"attempts"`
	Summary    DeliverySummary `json:"summary"`
}

// DeliverySummary represents a summary of delivery attempts
type DeliverySummary struct {
	TotalAttempts   int        `json:"total_attempts"`
	Status          string     `json:"status"`
	FirstAttemptAt  time.Time  `json:"first_attempt_at"`
	LastAttemptAt   *time.Time `json:"last_attempt_at"`
	NextRetryAt     *time.Time `json:"next_retry_at,omitempty"`
	FinalHTTPStatus *int       `json:"final_http_status,omitempty"`
	FinalErrorMsg   *string    `json:"final_error_message,omitempty"`
}

// GetDeliveryDetails retrieves detailed information about a specific delivery
func (s *WebhookService) GetDeliveryDetails(ctx context.Context, deliveryID uuid.UUID) (*DeliveryDetails, error) {
	var result DeliveryDetails
	path := fmt.Sprintf("/webhooks/deliveries/%s", deliveryID.String())
	err := s.client.get(ctx, path, &result)
	return &result, err
}
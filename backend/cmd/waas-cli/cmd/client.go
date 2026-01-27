package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client is the WAAS API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new WAAS API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Tenant represents a WAAS tenant
type Tenant struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	SubscriptionTier string    `json:"subscription_tier"`
	RateLimitPerMin  int       `json:"rate_limit_per_minute"`
	MonthlyQuota     int       `json:"monthly_quota"`
	CreatedAt        time.Time `json:"created_at"`
}

// Endpoint represents a webhook endpoint
type Endpoint struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	URL           string            `json:"url"`
	IsActive      bool              `json:"is_active"`
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfig      `json:"retry_config,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts       int `json:"max_attempts"`
	InitialDelay      int `json:"initial_delay_ms"`
	MaxDelay          int `json:"max_delay_ms"`
	BackoffMultiplier int `json:"backoff_multiplier"`
}

// Delivery represents a webhook delivery
type Delivery struct {
	ID             string    `json:"id"`
	EndpointID     string    `json:"endpoint_id"`
	Status         string    `json:"status"`
	AttemptCount   int       `json:"attempt_count"`
	LastAttemptAt  time.Time `json:"last_attempt_at,omitempty"`
	NextAttemptAt  time.Time `json:"next_attempt_at,omitempty"`
	LastHTTPStatus int       `json:"last_http_status,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// DeliveryAttempt represents a single delivery attempt
type DeliveryAttempt struct {
	ID            string    `json:"id"`
	DeliveryID    string    `json:"delivery_id"`
	AttemptNumber int       `json:"attempt_number"`
	HTTPStatus    int       `json:"http_status"`
	ResponseBody  string    `json:"response_body,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// parseResponse parses the response body into the target struct
func parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return &apiErr
	}

	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// GetTenant retrieves the current tenant information
func (c *Client) GetTenant() (*Tenant, error) {
	resp, err := c.doRequest("GET", "/api/v1/tenant", nil)
	if err != nil {
		return nil, err
	}

	var tenant Tenant
	if err := parseResponse(resp, &tenant); err != nil {
		return nil, err
	}

	return &tenant, nil
}

// ListEndpoints retrieves all webhook endpoints
func (c *Client) ListEndpoints() ([]Endpoint, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/endpoints", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Endpoints []Endpoint `json:"endpoints"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Endpoints, nil
}

// GetEndpoint retrieves a single endpoint by ID
func (c *Client) GetEndpoint(id string) (*Endpoint, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/endpoints/"+id, nil)
	if err != nil {
		return nil, err
	}

	var endpoint Endpoint
	if err := parseResponse(resp, &endpoint); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

// CreateEndpointRequest represents a request to create an endpoint
type CreateEndpointRequest struct {
	URL           string            `json:"url"`
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfig      `json:"retry_config,omitempty"`
}

// CreateEndpoint creates a new webhook endpoint
func (c *Client) CreateEndpoint(req *CreateEndpointRequest) (*Endpoint, error) {
	resp, err := c.doRequest("POST", "/api/v1/webhooks/endpoints", req)
	if err != nil {
		return nil, err
	}

	var endpoint Endpoint
	if err := parseResponse(resp, &endpoint); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

// DeleteEndpoint deletes a webhook endpoint
func (c *Client) DeleteEndpoint(id string) error {
	resp, err := c.doRequest("DELETE", "/api/v1/webhooks/endpoints/"+id, nil)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// SendWebhookRequest represents a request to send a webhook
type SendWebhookRequest struct {
	EndpointID string            `json:"endpoint_id"`
	Payload    json.RawMessage   `json:"payload"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// SendWebhookResponse represents the response from sending a webhook
type SendWebhookResponse struct {
	DeliveryID string `json:"delivery_id"`
	Status     string `json:"status"`
}

// SendWebhook sends a webhook to an endpoint
func (c *Client) SendWebhook(req *SendWebhookRequest) (*SendWebhookResponse, error) {
	body := map[string]interface{}{
		"endpoint_id": req.EndpointID,
		"payload":     req.Payload,
	}
	if req.Headers != nil {
		body["headers"] = req.Headers
	}

	resp, err := c.doRequest("POST", "/api/v1/webhooks/send", body)
	if err != nil {
		return nil, err
	}

	var result SendWebhookResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListDeliveries retrieves webhook deliveries
func (c *Client) ListDeliveries(endpointID string, limit int) ([]Delivery, error) {
	path := "/api/v1/webhooks/deliveries"
	if endpointID != "" {
		path = fmt.Sprintf("/api/v1/webhooks/endpoints/%s/deliveries", endpointID)
	}
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Deliveries []Delivery `json:"deliveries"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Deliveries, nil
}

// GetDelivery retrieves a single delivery by ID
func (c *Client) GetDelivery(id string) (*Delivery, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/deliveries/"+id, nil)
	if err != nil {
		return nil, err
	}

	var delivery Delivery
	if err := parseResponse(resp, &delivery); err != nil {
		return nil, err
	}

	return &delivery, nil
}

// GetDeliveryLogs retrieves delivery attempt logs
func (c *Client) GetDeliveryLogs(deliveryID string) ([]DeliveryAttempt, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/deliveries/"+deliveryID+"/logs", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Attempts []DeliveryAttempt `json:"attempts"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Attempts, nil
}

// ReplayDelivery replays a webhook delivery
func (c *Client) ReplayDelivery(deliveryID string) (*SendWebhookResponse, error) {
	resp, err := c.doRequest("POST", "/api/v1/webhooks/deliveries/"+deliveryID+"/replay", nil)
	if err != nil {
		return nil, err
	}

	var result SendWebhookResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateTenantRequest represents a request to create a tenant
type CreateTenantRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateTenantResponse represents the response from creating a tenant
type CreateTenantResponse struct {
	Tenant Tenant `json:"tenant"`
	APIKey string `json:"api_key"`
}

// CreateTenant creates a new tenant
func (c *Client) CreateTenant(name, email string) (*CreateTenantResponse, error) {
	req := &CreateTenantRequest{Name: name, Email: email}
	resp, err := c.doRequest("POST", "/api/v1/tenants", req)
	if err != nil {
		return nil, err
	}

	var result CreateTenantResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RegenerateAPIKeyResponse represents the response from regenerating an API key
type RegenerateAPIKeyResponse struct {
	APIKey string `json:"api_key"`
}

// RegenerateAPIKey regenerates the tenant's API key
func (c *Client) RegenerateAPIKey() (*RegenerateAPIKeyResponse, error) {
	resp, err := c.doRequest("POST", "/api/v1/tenant/regenerate-key", nil)
	if err != nil {
		return nil, err
	}

	var result RegenerateAPIKeyResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateEndpointRequest represents a request to update an endpoint
type UpdateEndpointRequest struct {
	URL           string            `json:"url,omitempty"`
	IsActive      *bool             `json:"is_active,omitempty"`
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfig      `json:"retry_config,omitempty"`
}

// UpdateEndpoint updates a webhook endpoint
func (c *Client) UpdateEndpoint(id string, req *UpdateEndpointRequest) (*Endpoint, error) {
	resp, err := c.doRequest("PUT", "/api/v1/webhooks/endpoints/"+id, req)
	if err != nil {
		return nil, err
	}

	var endpoint Endpoint
	if err := parseResponse(resp, &endpoint); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

// BatchRequest represents a single request in a batch send
type BatchRequest struct {
	EndpointID string          `json:"endpoint_id"`
	Payload    json.RawMessage `json:"payload"`
}

// BatchResponse represents the response from a batch send
type BatchResponse struct {
	Results []BatchResult `json:"results"`
}

// BatchResult represents the result of a single batch item
type BatchResult struct {
	EndpointID string `json:"endpoint_id"`
	DeliveryID string `json:"delivery_id"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

// BatchSend sends webhooks to multiple endpoints
func (c *Client) BatchSend(requests []BatchRequest) (*BatchResponse, error) {
	body := map[string]interface{}{"requests": requests}
	resp, err := c.doRequest("POST", "/api/v1/webhooks/send/batch", body)
	if err != nil {
		return nil, err
	}

	var result BatchResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeliveryDetail represents detailed delivery inspection data
type DeliveryDetail struct {
	Delivery Delivery          `json:"delivery"`
	Attempts []DeliveryAttempt `json:"attempts"`
	Request  *DeliveryRequest  `json:"request,omitempty"`
}

// DeliveryRequest represents the original request data
type DeliveryRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

// InspectDelivery returns detailed delivery inspection data
func (c *Client) InspectDelivery(id string) (*DeliveryDetail, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/deliveries/"+id+"/inspect", nil)
	if err != nil {
		return nil, err
	}

	var result DeliveryDetail
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RetryDelivery retries a failed delivery (alias for replay)
func (c *Client) RetryDelivery(deliveryID string) (*SendWebhookResponse, error) {
	return c.ReplayDelivery(deliveryID)
}

// StartMigration starts a live migration from the given platform
func (c *Client) StartMigration(platform, filePath string, dryRun bool) (map[string]interface{}, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	body := map[string]interface{}{
		"platform": platform,
		"data":     string(fileContent),
	}

	var path string
	if dryRun {
		path = "/api/v1/livemigration/dry-run"
	} else {
		path = "/api/v1/livemigration/import/" + platform
	}

	resp, err := c.doRequest("POST", path, body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMigrationStatus retrieves the status of migration jobs
func (c *Client) GetMigrationStatus() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/livemigration/jobs", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Jobs []map[string]interface{} `json:"jobs"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Jobs, nil
}

// RollbackMigration rolls back a migration job by ID
func (c *Client) RollbackMigration(jobID string) (map[string]interface{}, error) {
	resp, err := c.doRequest("POST", "/api/v1/livemigration/jobs/"+jobID+"/rollback", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ExportEndpoints retrieves all endpoints for export
func (c *Client) ExportEndpoints() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/endpoints", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Endpoints []map[string]interface{} `json:"endpoints"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Endpoints, nil
}

// ExportDeliveries retrieves all deliveries for export
func (c *Client) ExportDeliveries() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/webhooks/deliveries", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Deliveries []map[string]interface{} `json:"deliveries"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Deliveries, nil
}

// ImportEndpoints imports endpoints, optionally as a dry run
func (c *Client) ImportEndpoints(endpoints []map[string]interface{}, dryRun bool) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"endpoints": endpoints,
		"dry_run":   dryRun,
	}

	resp, err := c.doRequest("POST", "/api/v1/webhooks/endpoints/import", body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GenerateFromOpenAPI generates webhook config from an OpenAPI spec
func (c *Client) GenerateFromOpenAPI(specContent string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"spec": specContent,
	}

	resp, err := c.doRequest("POST", "/api/v1/openapi/generate", body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GenerateSDK generates SDK types from an OpenAPI spec
func (c *Client) GenerateSDK(specContent, language string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"spec":     specContent,
		"language": language,
	}

	resp, err := c.doRequest("POST", "/api/v1/openapi/generate-sdk", body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GenerateTests generates contract tests from an OpenAPI spec
func (c *Client) GenerateTests(specContent string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"spec": specContent,
	}

	resp, err := c.doRequest("POST", "/api/v1/openapi/generate-tests", body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Package client provides a Go SDK for the Webhook Platform API
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultBaseURL is the default API base URL
	DefaultBaseURL = "https://api.webhook-platform.com/api/v1"
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
	// UserAgent is the SDK user agent string
	UserAgent = "webhook-platform-go-sdk/1.0.0"
)

// Client is the main SDK client
type Client struct {
	config   *Config
	http     *http.Client
	Webhooks *WebhookService
	Tenants  *TenantService
	Testing  *TestingService
}

// Config holds the client configuration
type Config struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// APIError represents an API error response
type APIError struct {
	StatusCode int                    `json:"-"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s - %s", e.StatusCode, e.Code, e.Message)
}

// New creates a new client with the provided API key
func New(apiKey string) *Client {
	return NewWithConfig(&Config{
		APIKey:  apiKey,
		BaseURL: DefaultBaseURL,
		Timeout: DefaultTimeout,
	})
}

// NewFromEnv creates a new client using the WEBHOOK_PLATFORM_API_KEY environment variable
func NewFromEnv() (*Client, error) {
	apiKey := os.Getenv("WEBHOOK_PLATFORM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("WEBHOOK_PLATFORM_API_KEY environment variable is required")
	}
	return New(apiKey), nil
}

// NewWithConfig creates a new client with the provided configuration
func NewWithConfig(config *Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return NewWithHTTPClient(config.APIKey, httpClient, config.BaseURL)
}

// NewWithHTTPClient creates a new client with a custom HTTP client
func NewWithHTTPClient(apiKey string, httpClient *http.Client, baseURL ...string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}

	url := DefaultBaseURL
	if len(baseURL) > 0 && baseURL[0] != "" {
		url = baseURL[0]
	}

	config := &Config{
		APIKey:  apiKey,
		BaseURL: url,
		Timeout: httpClient.Timeout,
	}

	c := &Client{
		config: config,
		http:   httpClient,
	}

	// Initialize services
	c.Webhooks = &WebhookService{client: c}
	c.Tenants = &TenantService{client: c}
	c.Testing = &TestingService{client: c}

	return c
}

// makeRequest performs an HTTP request and handles the response
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Build URL
	u, err := url.Parse(c.config.BaseURL + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-API-Key", c.config.APIKey)

	// Make request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var apiErr APIError
		apiErr.StatusCode = resp.StatusCode

		// Try to parse error response
		var errorResp struct {
			Error APIError `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			apiErr = errorResp.Error
			apiErr.StatusCode = resp.StatusCode
		} else {
			// Fallback for non-JSON error responses
			apiErr.Code = "UNKNOWN_ERROR"
			apiErr.Message = string(respBody)
		}

		return &apiErr
	}

	// Parse successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// get performs a GET request
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	return c.makeRequest(ctx, "GET", path, nil, result)
}

// post performs a POST request
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.makeRequest(ctx, "POST", path, body, result)
}

// put performs a PUT request
func (c *Client) put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.makeRequest(ctx, "PUT", path, body, result)
}

// delete performs a DELETE request
func (c *Client) delete(ctx context.Context, path string) error {
	return c.makeRequest(ctx, "DELETE", path, nil, nil)
}
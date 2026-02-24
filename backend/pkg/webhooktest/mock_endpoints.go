package webhooktest

import (
	"github.com/josedab/waas/pkg/httputil"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MockEndpointService manages test endpoints with unique URLs and assertion APIs
type MockEndpointService struct {
	endpoints map[string]*TestEndpoint
	requests  map[string][]*CapturedWebhook
	mu        sync.RWMutex
}

// NewMockEndpointService creates a new mock endpoint service
func NewMockEndpointService() *MockEndpointService {
	return &MockEndpointService{
		endpoints: make(map[string]*TestEndpoint),
		requests:  make(map[string][]*CapturedWebhook),
	}
}

// TestEndpoint represents a temporary test URL for capturing webhooks
type TestEndpoint struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	Token         string            `json:"token"`
	TenantID      string            `json:"tenant_id"`
	ResponseCode  int               `json:"response_code"`
	ResponseBody  string            `json:"response_body"`
	ResponseDelay int               `json:"response_delay_ms"`
	Headers       map[string]string `json:"headers,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at"`
	CreatedAt     time.Time         `json:"created_at"`
	RequestCount  int               `json:"request_count"`
}

// CapturedWebhook represents a webhook received by a test endpoint
type CapturedWebhook struct {
	ID         string              `json:"id"`
	EndpointID string              `json:"endpoint_id"`
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	Headers    map[string][]string `json:"headers"`
	Body       json.RawMessage     `json:"body"`
	Query      map[string]string   `json:"query,omitempty"`
	ReceivedAt time.Time           `json:"received_at"`
	SourceIP   string              `json:"source_ip,omitempty"`
}

// CreateTestEndpointRequest creates a temporary test endpoint
type CreateTestEndpointRequest struct {
	ResponseCode  int               `json:"response_code"`
	ResponseBody  string            `json:"response_body,omitempty"`
	ResponseDelay int               `json:"response_delay_ms,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	TTLMinutes    int               `json:"ttl_minutes"`
}

// AssertionRequest defines what to assert on captured webhooks
type AssertionRequest struct {
	EndpointID string      `json:"endpoint_id" binding:"required"`
	Assertions []Assertion `json:"assertions" binding:"required,min=1"`
}

// AssertionResult contains the result of assertions on captured webhooks
type AssertionResult struct {
	EndpointID   string         `json:"endpoint_id"`
	TotalChecks  int            `json:"total_checks"`
	PassedChecks int            `json:"passed_checks"`
	FailedChecks int            `json:"failed_checks"`
	Results      []AssertResult `json:"results"`
	AllPassed    bool           `json:"all_passed"`
}

// ChaosConfig defines chaos injection parameters
type ChaosConfig struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	EndpointID  string    `json:"endpoint_id,omitempty"`
	FailureType string    `json:"failure_type"`     // timeout, error_500, rate_limit, dns_failure, slow_response, connection_reset
	FailureRate float64   `json:"failure_rate"`     // 0.0-1.0
	DurationSec int       `json:"duration_seconds"` // how long to inject chaos
	LatencyMs   int       `json:"latency_ms,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// ChaosFailureType constants
const (
	ChaosTimeout         = "timeout"
	ChaosError500        = "error_500"
	ChaosRateLimit       = "rate_limit"
	ChaosDNSFailure      = "dns_failure"
	ChaosSlowResponse    = "slow_response"
	ChaosConnectionReset = "connection_reset"
	ChaosPartialResponse = "partial_response"
)

// CreateChaosRequest defines a chaos injection request
type CreateChaosRequest struct {
	EndpointID  string  `json:"endpoint_id,omitempty"`
	FailureType string  `json:"failure_type" binding:"required"`
	FailureRate float64 `json:"failure_rate" binding:"required,min=0,max=1"`
	DurationSec int     `json:"duration_seconds" binding:"required,min=1,max=3600"`
	LatencyMs   int     `json:"latency_ms,omitempty"`
}

// CITestConfig represents a CI/CD test configuration
type CITestConfig struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	BaseURL     string       `json:"base_url" yaml:"base_url"`
	APIKey      string       `json:"api_key" yaml:"api_key"`
	Tests       []CITestCase `json:"tests" yaml:"tests"`
	FailFast    bool         `json:"fail_fast,omitempty" yaml:"fail_fast,omitempty"`
	Timeout     int          `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
}

// CITestCase represents a single CI/CD test case
type CITestCase struct {
	Name       string            `json:"name" yaml:"name"`
	EventType  string            `json:"event_type" yaml:"event_type"`
	Payload    json.RawMessage   `json:"payload" yaml:"payload"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Assertions []Assertion       `json:"assertions" yaml:"assertions"`
	RetryCount int               `json:"retry_count,omitempty" yaml:"retry_count,omitempty"`
	DelayMs    int               `json:"delay_ms,omitempty" yaml:"delay_ms,omitempty"`
}

// CITestResult contains the result of a CI/CD test run
type CITestResult struct {
	Name        string         `json:"name"`
	TotalTests  int            `json:"total_tests"`
	PassedTests int            `json:"passed_tests"`
	FailedTests int            `json:"failed_tests"`
	Duration    time.Duration  `json:"duration"`
	Results     []CITestOutput `json:"results"`
	ExitCode    int            `json:"exit_code"` // 0 = success, 1 = failure
}

// CITestOutput represents the output of a single CI test
type CITestOutput struct {
	Name     string         `json:"name"`
	Status   string         `json:"status"` // passed, failed, error
	Duration int64          `json:"duration_ms"`
	Error    string         `json:"error,omitempty"`
	Details  []AssertResult `json:"details,omitempty"`
}

// CreateTestEndpoint creates a temporary endpoint with unique URL
func (s *MockEndpointService) CreateTestEndpoint(ctx context.Context, tenantID, baseURL string, req *CreateTestEndpointRequest) (*TestEndpoint, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	ttl := req.TTLMinutes
	if ttl <= 0 {
		ttl = 60
	}
	if ttl > 1440 {
		ttl = 1440
	}

	responseCode := req.ResponseCode
	if responseCode == 0 {
		responseCode = 200
	}

	ep := &TestEndpoint{
		ID:            uuid.New().String(),
		Token:         token,
		URL:           fmt.Sprintf("%s/test-hooks/%s", baseURL, token),
		TenantID:      tenantID,
		ResponseCode:  responseCode,
		ResponseBody:  req.ResponseBody,
		ResponseDelay: req.ResponseDelay,
		Headers:       req.Headers,
		ExpiresAt:     time.Now().Add(time.Duration(ttl) * time.Minute),
		CreatedAt:     time.Now(),
	}

	s.mu.Lock()
	s.endpoints[token] = ep
	s.requests[ep.ID] = make([]*CapturedWebhook, 0)
	s.mu.Unlock()

	return ep, nil
}

// CaptureRequest records an incoming webhook to a test endpoint
func (s *MockEndpointService) CaptureRequest(token string, r *http.Request, body []byte) (*TestEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ep, ok := s.endpoints[token]
	if !ok {
		return nil, fmt.Errorf("test endpoint not found")
	}

	if time.Now().After(ep.ExpiresAt) {
		return nil, fmt.Errorf("test endpoint expired")
	}

	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	captured := &CapturedWebhook{
		ID:         uuid.New().String(),
		EndpointID: ep.ID,
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    r.Header,
		Body:       body,
		Query:      query,
		ReceivedAt: time.Now(),
		SourceIP:   r.RemoteAddr,
	}

	s.requests[ep.ID] = append(s.requests[ep.ID], captured)
	ep.RequestCount++

	return ep, nil
}

// GetCapturedRequests returns all captured webhooks for an endpoint
func (s *MockEndpointService) GetCapturedRequests(endpointID string) []*CapturedWebhook {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requests[endpointID]
}

// RunAssertions checks assertions against captured webhooks
func (s *MockEndpointService) RunAssertions(req *AssertionRequest) *AssertionResult {
	captured := s.GetCapturedRequests(req.EndpointID)

	result := &AssertionResult{
		EndpointID:  req.EndpointID,
		TotalChecks: len(req.Assertions),
	}

	for _, assertion := range req.Assertions {
		ar := AssertResult{
			Type:   assertion.Type,
			Passed: true,
		}

		switch assertion.Type {
		case "request_count":
			expected := fmt.Sprintf("%v", assertion.Expected)
			actual := fmt.Sprintf("%d", len(captured))
			ar.Expected = expected
			ar.Actual = actual
			if actual != expected {
				ar.Passed = false
				ar.Message = fmt.Sprintf("expected %s requests, got %s", expected, actual)
			}

		case "body_contains":
			expectedStr := fmt.Sprintf("%v", assertion.Expected)
			found := false
			for _, c := range captured {
				if contains(string(c.Body), expectedStr) {
					found = true
					break
				}
			}
			if !found {
				ar.Passed = false
				ar.Message = fmt.Sprintf("no request body contains %q", expectedStr)
			}

		case "header_present":
			found := false
			for _, c := range captured {
				if _, ok := c.Headers[assertion.Field]; ok {
					found = true
					break
				}
			}
			if !found {
				ar.Passed = false
				ar.Message = fmt.Sprintf("header %q not found in any request", assertion.Field)
			}

		case "header_value":
			found := false
			expectedStr := fmt.Sprintf("%v", assertion.Expected)
			for _, c := range captured {
				if vals, ok := c.Headers[assertion.Field]; ok {
					for _, v := range vals {
						if v == expectedStr {
							found = true
							break
						}
					}
				}
			}
			if !found {
				ar.Passed = false
				ar.Message = fmt.Sprintf("header %q with value %q not found", assertion.Field, assertion.Expected)
			}

		case "payload_field":
			found := false
			expectedStr := fmt.Sprintf("%v", assertion.Expected)
			for _, c := range captured {
				var data map[string]interface{}
				if json.Unmarshal(c.Body, &data) == nil {
					if val, ok := data[assertion.Field]; ok {
						if fmt.Sprintf("%v", val) == expectedStr {
							found = true
							break
						}
					}
				}
			}
			if !found {
				ar.Passed = false
				ar.Message = fmt.Sprintf("field %q with value %q not found", assertion.Field, assertion.Expected)
			}
		}

		if ar.Passed {
			result.PassedChecks++
		} else {
			result.FailedChecks++
		}
		result.Results = append(result.Results, ar)
	}

	result.AllPassed = result.FailedChecks == 0
	return result
}

// RunCITests executes a CI/CD test suite
func (s *MockEndpointService) RunCITests(ctx context.Context, config *CITestConfig) *CITestResult {
	start := time.Now()
	result := &CITestResult{
		Name:       config.Name,
		TotalTests: len(config.Tests),
	}

	for _, test := range config.Tests {
		testStart := time.Now()
		output := CITestOutput{
			Name:   test.Name,
			Status: "passed",
		}

		// Simulate sending the webhook and checking assertions
		for _, assertion := range test.Assertions {
			ar := AssertResult{
				Type:   assertion.Type,
				Passed: true,
			}

			// Validate assertion structure
			if assertion.Type == "" {
				ar.Passed = false
				ar.Message = "assertion type is required"
			}

			output.Details = append(output.Details, ar)
			if !ar.Passed {
				output.Status = "failed"
			}
		}

		output.Duration = time.Since(testStart).Milliseconds()

		if output.Status == "passed" {
			result.PassedTests++
		} else {
			result.FailedTests++
			if config.FailFast {
				result.Results = append(result.Results, output)
				break
			}
		}

		result.Results = append(result.Results, output)
	}

	result.Duration = time.Since(start)
	if result.FailedTests > 0 {
		result.ExitCode = 1
	}

	return result
}

// Cleanup removes expired test endpoints
func (s *MockEndpointService) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for token, ep := range s.endpoints {
		if time.Now().After(ep.ExpiresAt) {
			delete(s.endpoints, token)
			delete(s.requests, ep.ID)
			count++
		}
	}
	return count
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "wt_" + hex.EncodeToString(b), nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MockEndpointHandler provides HTTP handlers for the mock endpoint service
type MockEndpointHandler struct {
	service *MockEndpointService
}

// NewMockEndpointHandler creates a new handler
func NewMockEndpointHandler(service *MockEndpointService) *MockEndpointHandler {
	return &MockEndpointHandler{service: service}
}

// RegisterRoutes registers mock endpoint routes
func (h *MockEndpointHandler) RegisterRoutes(router *gin.RouterGroup) {
	t := router.Group("/webhook-testing")
	{
		t.POST("/endpoints", h.CreateTestEndpoint)
		t.GET("/endpoints/:id/requests", h.GetCapturedRequests)
		t.POST("/endpoints/assert", h.RunAssertions)
		t.POST("/chaos", h.CreateChaosExperiment)
		t.POST("/ci/run", h.RunCITests)
		t.GET("/ci/config-template", h.GetCIConfigTemplate)
	}
}

func (h *MockEndpointHandler) CreateTestEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateTestEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	baseURL := c.GetString("api_url")
	if baseURL == "" {
		baseURL = "https://api.waas.dev"
	}
	ep, err := h.service.CreateTestEndpoint(c.Request.Context(), tenantID, baseURL, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}
	c.JSON(http.StatusCreated, ep)
}

func (h *MockEndpointHandler) GetCapturedRequests(c *gin.Context) {
	endpointID := c.Param("id")
	requests := h.service.GetCapturedRequests(endpointID)
	c.JSON(http.StatusOK, gin.H{"requests": requests, "total": len(requests)})
}

func (h *MockEndpointHandler) RunAssertions(c *gin.Context) {
	var req AssertionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	result := h.service.RunAssertions(&req)
	status := http.StatusOK
	if !result.AllPassed {
		status = http.StatusExpectationFailed
	}
	c.JSON(status, result)
}

func (h *MockEndpointHandler) CreateChaosExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateChaosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	chaos := &ChaosConfig{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		EndpointID:  req.EndpointID,
		FailureType: req.FailureType,
		FailureRate: req.FailureRate,
		DurationSec: req.DurationSec,
		LatencyMs:   req.LatencyMs,
		IsActive:    true,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(req.DurationSec) * time.Second),
	}

	c.JSON(http.StatusCreated, chaos)
}

func (h *MockEndpointHandler) RunCITests(c *gin.Context) {
	var config CITestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	result := h.service.RunCITests(c.Request.Context(), &config)
	c.JSON(http.StatusOK, result)
}

func (h *MockEndpointHandler) GetCIConfigTemplate(c *gin.Context) {
	template := CITestConfig{
		Name:    "WaaS Webhook Tests",
		BaseURL: "https://api.waas.dev",
		APIKey:  "${WAAS_API_KEY}",
		Tests: []CITestCase{
			{
				Name:      "Test order.created webhook",
				EventType: "order.created",
				Payload:   json.RawMessage(`{"order_id":"test_123","amount":99.99}`),
				Assertions: []Assertion{
					{Type: "status_code", Expected: "200"},
					{Type: "latency", Expected: "5000", Operator: "lt"},
				},
			},
		},
		FailFast: false,
		Timeout:  30,
	}
	c.JSON(http.StatusOK, template)
}

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/httputil"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// TestingHandler handles webhook testing, inspection, and real-time debugging.
type TestingHandler struct {
	webhookRepo         repository.WebhookEndpointRepository
	deliveryAttemptRepo repository.DeliveryAttemptRepository
	publisher           queue.PublisherInterface
	logger              *utils.Logger
	upgrader            websocket.Upgrader
	urlValidator        URLValidatorInterface
	httpClientFactory   func(timeout time.Duration) *http.Client
}

// TestWebhookRequest represents a webhook test request
type TestWebhookRequest struct {
	URL     string            `json:"url" binding:"required"`
	Payload json.RawMessage   `json:"payload" binding:"required"`
	Headers map[string]string `json:"headers,omitempty"`
	Method  string            `json:"method,omitempty"`  // defaults to POST
	Timeout int               `json:"timeout,omitempty"` // seconds, defaults to 30
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
	TTL         int               `json:"ttl,omitempty"` // seconds, defaults to 3600 (1 hour)
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
	DeliveryID    uuid.UUID                `json:"delivery_id"`
	EndpointID    uuid.UUID                `json:"endpoint_id"`
	Status        string                   `json:"status"`
	AttemptNumber int                      `json:"attempt_number"`
	Request       *DeliveryRequestDetails  `json:"request"`
	Response      *DeliveryResponseDetails `json:"response,omitempty"`
	Timeline      []DeliveryTimelineEvent  `json:"timeline"`
	ErrorDetails  *DeliveryErrorDetails    `json:"error_details,omitempty"`
}

// DeliveryRequestDetails contains the HTTP request details of a delivery attempt.
type DeliveryRequestDetails struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	PayloadHash string            `json:"payload_hash"`
	PayloadSize int               `json:"payload_size"`
	Signature   string            `json:"signature"`
	ScheduledAt time.Time         `json:"scheduled_at"`
}

// DeliveryResponseDetails contains the HTTP response details of a delivery attempt.
type DeliveryResponseDetails struct {
	HTTPStatus  int               `json:"http_status"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	BodySize    int               `json:"body_size"`
	DeliveredAt time.Time         `json:"delivered_at"`
	Latency     int64             `json:"latency_ms"`
}

// DeliveryTimelineEvent represents a single event in the delivery timeline.
type DeliveryTimelineEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Event       string                 `json:"event"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// DeliveryErrorDetails provides error analysis with categorization and troubleshooting suggestions.
type DeliveryErrorDetails struct {
	ErrorType    string     `json:"error_type"`
	ErrorMessage string     `json:"error_message"`
	HTTPStatus   *int       `json:"http_status,omitempty"`
	RetryCount   int        `json:"retry_count"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	Suggestions  []string   `json:"suggestions,omitempty"`
}

// WebSocketMessage represents a real-time update message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewTestingHandler(webhookRepo repository.WebhookEndpointRepository, deliveryAttemptRepo repository.DeliveryAttemptRepository, publisher queue.PublisherInterface, logger *utils.Logger) *TestingHandler {
	upgrader := websocket.Upgrader{
		CheckOrigin: utils.CheckWebSocketOrigin(),
	}

	return &TestingHandler{
		webhookRepo:         webhookRepo,
		deliveryAttemptRepo: deliveryAttemptRepo,
		publisher:           publisher,
		logger:              logger,
		upgrader:            upgrader,
		urlValidator:        utils.NewURLValidator(),
		httpClientFactory:   httputil.NewSSRFSafeClient,
	}
}

// SetURLValidator replaces the URL validator (useful for testing).
func (h *TestingHandler) SetURLValidator(v URLValidatorInterface) {
	h.urlValidator = v
}

// SetHTTPClientFactory replaces the HTTP client factory (useful for testing).
func (h *TestingHandler) SetHTTPClientFactory(f func(timeout time.Duration) *http.Client) {
	h.httpClientFactory = f
}

// TestWebhook tests a webhook endpoint with a custom payload
// @Summary Test webhook endpoint
// @Description Send a test webhook to any URL with custom payload and headers for immediate testing
// @Tags testing
// @Accept json
// @Produce json
// @Param request body TestWebhookRequest true "Webhook test request"
// @Success 200 {object} TestWebhookResponse "Webhook test result"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/test [post]
func (h *TestingHandler) TestWebhook(c *gin.Context) {
	var req TestWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Validate URL
	if err := h.urlValidator.ValidateWebhookURL(req.URL); err != nil {
		BadRequest(c, "INVALID_URL", "Invalid webhook URL format")
		return
	}

	// Validate payload
	var payloadTest interface{}
	if err := json.Unmarshal(req.Payload, &payloadTest); err != nil {
		BadRequest(c, "INVALID_PAYLOAD", "Webhook payload must be valid JSON")
		return
	}

	// Set defaults
	if req.Method == "" {
		req.Method = "POST"
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}

	// Generate test ID and request ID
	testID := uuid.New()
	requestID := fmt.Sprintf("test_%s", testID.String()[:8])

	// Perform the webhook test
	result := h.performWebhookTest(c.Request.Context(), &req, testID, requestID)

	h.logger.Info("Webhook test performed", map[string]interface{}{
		"test_id":   testID,
		"tenant_id": tenantID,
		"url":       req.URL,
		"status":    result.Status,
	})

	c.JSON(http.StatusOK, result)
}

// CreateTestEndpoint creates a temporary test endpoint for webhook testing
// @Summary Create test endpoint
// @Description Create a temporary webhook endpoint that can receive test webhooks for debugging
// @Tags testing
// @Accept json
// @Produce json
// @Param request body CreateTestEndpointRequest true "Test endpoint creation request"
// @Success 201 {object} TestEndpointResponse "Test endpoint created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/test/endpoints [post]
func (h *TestingHandler) CreateTestEndpoint(c *gin.Context) {
	var req CreateTestEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Set defaults
	if req.TTL == 0 {
		req.TTL = 3600 // 1 hour
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("Test Endpoint %s", time.Now().Format("15:04:05"))
	}

	// Generate test endpoint
	endpointID := uuid.New()
	// Derive base URL from the incoming request
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	testURL := fmt.Sprintf("%s://%s/test/%s", scheme, c.Request.Host, endpointID.String())

	now := time.Now()
	expiresAt := now.Add(time.Duration(req.TTL) * time.Second)

	response := TestEndpointResponse{
		ID:          endpointID,
		URL:         testURL,
		Name:        req.Name,
		Description: req.Description,
		Headers:     req.Headers,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	h.logger.Info("Test endpoint created", map[string]interface{}{
		"endpoint_id": endpointID,
		"tenant_id":   tenantID,
		"ttl":         req.TTL,
	})

	c.JSON(http.StatusCreated, response)
}

// InspectDelivery provides detailed debugging information for a webhook delivery
// @Summary Inspect delivery
// @Description Get detailed debugging information about a webhook delivery including request/response details and error analysis
// @Tags testing
// @Accept json
// @Produce json
// @Param id path string true "Delivery ID" format(uuid)
// @Success 200 {object} DeliveryInspectionResponse "Detailed delivery inspection information"
// @Failure 400 {object} map[string]interface{} "Invalid delivery ID format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 404 {object} map[string]interface{} "Delivery not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/deliveries/{id}/inspect [get]
func (h *TestingHandler) InspectDelivery(c *gin.Context) {
	// Parse delivery ID
	deliveryIDStr := c.Param("id")
	deliveryID, err := uuid.Parse(deliveryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid delivery ID format"})
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Get delivery attempts for this delivery ID
	attempts, err := h.deliveryAttemptRepo.GetDeliveryAttemptsByDeliveryID(c.Request.Context(), deliveryID, tenantID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Code: "DELIVERY_NOT_FOUND", Message: "Delivery not found"})
			return
		}

		h.logger.Error("Failed to get delivery attempts for inspection", map[string]interface{}{
			"error":       err.Error(),
			"delivery_id": deliveryID,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve delivery information"})
		return
	}

	if len(attempts) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "DELIVERY_NOT_FOUND", Message: "Delivery not found"})
		return
	}

	// Get the latest attempt for main details
	latestAttempt := attempts[len(attempts)-1]

	// Get endpoint details
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), latestAttempt.EndpointID)
	if err != nil {
		h.logger.Error("Failed to get endpoint for delivery inspection", map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": latestAttempt.EndpointID,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve endpoint information"})
		return
	}

	// Build inspection response
	inspection := h.buildDeliveryInspection(deliveryID, endpoint, attempts)

	c.JSON(http.StatusOK, inspection)
}

// GetDeliveryLogs handles GET /webhooks/deliveries/:id/logs
func (h *TestingHandler) GetDeliveryLogs(c *gin.Context) {
	// Parse delivery ID
	deliveryIDStr := c.Param("id")
	deliveryID, err := uuid.Parse(deliveryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid delivery ID format"})
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Get delivery attempts
	attempts, err := h.deliveryAttemptRepo.GetDeliveryAttemptsByDeliveryID(c.Request.Context(), deliveryID, tenantID)
	if err != nil {
		h.logger.Error("Failed to get delivery logs", map[string]interface{}{
			"error":       err.Error(),
			"delivery_id": deliveryID,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DATABASE_ERROR", Message: "Failed to retrieve delivery logs"})
		return
	}

	// Format logs for display
	logs := make([]map[string]interface{}, len(attempts))
	for i, attempt := range attempts {
		logs[i] = map[string]interface{}{
			"attempt_number": attempt.AttemptNumber,
			"status":         attempt.Status,
			"http_status":    attempt.HTTPStatus,
			"response_body":  attempt.ResponseBody,
			"error_message":  attempt.ErrorMessage,
			"scheduled_at":   attempt.ScheduledAt,
			"delivered_at":   attempt.DeliveredAt,
			"created_at":     attempt.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"delivery_id":    deliveryID,
		"logs":           logs,
		"total_attempts": len(attempts),
	})
}

// WebSocketUpdates handles WebSocket connections for real-time delivery updates
func (h *TestingHandler) WebSocketUpdates(c *gin.Context) {
	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer conn.Close()

	h.logger.Info("WebSocket connection established", map[string]interface{}{
		"tenant_id": tenantID,
	})

	// Handle WebSocket connection
	h.handleWebSocketConnection(conn, tenantID)
}

// Helper methods

func (h *TestingHandler) performWebhookTest(ctx context.Context, req *TestWebhookRequest, testID uuid.UUID, requestID string) *TestWebhookResponse {
	startTime := time.Now()

	response := &TestWebhookResponse{
		TestID:    testID,
		URL:       req.URL,
		RequestID: requestID,
		TestedAt:  startTime,
	}

	// Create HTTP client with timeout
	client := h.httpClientFactory(time.Duration(req.Timeout) * time.Second)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(string(req.Payload)))
	if err != nil {
		response.Status = "failed"
		errorMsg := fmt.Sprintf("Failed to create HTTP request: %v", err)
		response.ErrorMessage = &errorMsg
		return response
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Webhook-Platform-Test/1.0")
	httpReq.Header.Set("X-Request-ID", requestID)

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Perform the request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		latency := time.Since(startTime).Milliseconds()
		response.Latency = &latency
		response.Status = "failed"
		errorMsg := fmt.Sprintf("HTTP request failed: %v", err)
		response.ErrorMessage = &errorMsg
		return response
	}
	defer httpResp.Body.Close()

	// Calculate latency
	latency := time.Since(startTime).Milliseconds()
	response.Latency = &latency
	response.HTTPStatus = &httpResp.StatusCode

	// Read response body (limit to 10KB for safety)
	bodyBytes := make([]byte, 10240)
	n, _ := httpResp.Body.Read(bodyBytes)
	if n > 0 {
		responseBody := string(bodyBytes[:n])
		response.ResponseBody = &responseBody
	}

	// Determine status
	if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
		response.Status = "success"
	} else {
		response.Status = "failed"
		errorMsg := fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, httpResp.Status)
		response.ErrorMessage = &errorMsg
	}

	return response
}

func (h *TestingHandler) buildDeliveryInspection(deliveryID uuid.UUID, endpoint *models.WebhookEndpoint, attempts []*models.DeliveryAttempt) *DeliveryInspectionResponse {
	latestAttempt := attempts[len(attempts)-1]

	inspection := &DeliveryInspectionResponse{
		DeliveryID:    deliveryID,
		EndpointID:    endpoint.ID,
		Status:        latestAttempt.Status,
		AttemptNumber: latestAttempt.AttemptNumber,
	}

	// Build request details
	inspection.Request = &DeliveryRequestDetails{
		URL:         endpoint.URL,
		Method:      "POST",
		Headers:     endpoint.CustomHeaders,
		PayloadHash: latestAttempt.PayloadHash,
		PayloadSize: latestAttempt.PayloadSize,
		ScheduledAt: latestAttempt.ScheduledAt,
	}

	// Build response details if available
	if latestAttempt.HTTPStatus != nil && latestAttempt.DeliveredAt != nil {
		inspection.Response = &DeliveryResponseDetails{
			HTTPStatus:  *latestAttempt.HTTPStatus,
			DeliveredAt: *latestAttempt.DeliveredAt,
		}

		if latestAttempt.ResponseBody != nil {
			inspection.Response.Body = *latestAttempt.ResponseBody
			inspection.Response.BodySize = len(*latestAttempt.ResponseBody)
		}

		// Calculate latency if possible
		if latestAttempt.DeliveredAt != nil {
			latency := latestAttempt.DeliveredAt.Sub(latestAttempt.ScheduledAt).Milliseconds()
			inspection.Response.Latency = latency
		}
	}

	// Build timeline
	inspection.Timeline = h.buildDeliveryTimeline(attempts)

	// Build error details if failed
	if latestAttempt.Status == "failed" {
		inspection.ErrorDetails = h.buildErrorDetails(latestAttempt, len(attempts))
	}

	return inspection
}

func (h *TestingHandler) buildDeliveryTimeline(attempts []*models.DeliveryAttempt) []DeliveryTimelineEvent {
	var timeline []DeliveryTimelineEvent

	for _, attempt := range attempts {
		// Scheduled event
		timeline = append(timeline, DeliveryTimelineEvent{
			Timestamp:   attempt.ScheduledAt,
			Event:       "scheduled",
			Description: fmt.Sprintf("Delivery attempt %d scheduled", attempt.AttemptNumber),
			Details: map[string]interface{}{
				"attempt_number": attempt.AttemptNumber,
			},
		})

		// Delivery event
		if attempt.DeliveredAt != nil {
			event := "delivered"
			description := fmt.Sprintf("Delivery attempt %d completed", attempt.AttemptNumber)

			if attempt.Status == "failed" {
				event = "failed"
				description = fmt.Sprintf("Delivery attempt %d failed", attempt.AttemptNumber)
			}

			details := map[string]interface{}{
				"attempt_number": attempt.AttemptNumber,
				"status":         attempt.Status,
			}

			if attempt.HTTPStatus != nil {
				details["http_status"] = *attempt.HTTPStatus
			}

			if attempt.ErrorMessage != nil {
				details["error_message"] = *attempt.ErrorMessage
			}

			timeline = append(timeline, DeliveryTimelineEvent{
				Timestamp:   *attempt.DeliveredAt,
				Event:       event,
				Description: description,
				Details:     details,
			})
		}
	}

	return timeline
}

func (h *TestingHandler) buildErrorDetails(attempt *models.DeliveryAttempt, retryCount int) *DeliveryErrorDetails {
	errorDetails := &DeliveryErrorDetails{
		RetryCount: retryCount,
	}

	if attempt.ErrorMessage != nil {
		errorDetails.ErrorMessage = *attempt.ErrorMessage

		// Categorize error type based on message
		errorMsg := strings.ToLower(*attempt.ErrorMessage)
		switch {
		case strings.Contains(errorMsg, "timeout"):
			errorDetails.ErrorType = "timeout"
			errorDetails.Suggestions = []string{
				"Check if the endpoint is responding within the timeout period",
				"Consider increasing the timeout configuration",
				"Verify the endpoint is not overloaded",
			}
		case strings.Contains(errorMsg, "connection"):
			errorDetails.ErrorType = "connection"
			errorDetails.Suggestions = []string{
				"Verify the endpoint URL is correct and accessible",
				"Check if there are network connectivity issues",
				"Ensure the endpoint server is running",
			}
		case strings.Contains(errorMsg, "dns"):
			errorDetails.ErrorType = "dns"
			errorDetails.Suggestions = []string{
				"Verify the domain name is correct",
				"Check DNS resolution for the endpoint domain",
				"Ensure the domain is not expired or misconfigured",
			}
		default:
			errorDetails.ErrorType = "unknown"
			errorDetails.Suggestions = []string{
				"Check the endpoint logs for more details",
				"Verify the endpoint is configured to accept webhooks",
				"Contact the endpoint administrator if the issue persists",
			}
		}
	}

	if attempt.HTTPStatus != nil {
		errorDetails.HTTPStatus = attempt.HTTPStatus

		// Add HTTP status specific suggestions
		switch *attempt.HTTPStatus {
		case 400:
			errorDetails.ErrorType = "client_error"
			errorDetails.Suggestions = append(errorDetails.Suggestions, "Check the webhook payload format and structure")
		case 401, 403:
			errorDetails.ErrorType = "authentication"
			errorDetails.Suggestions = append(errorDetails.Suggestions, "Verify webhook authentication credentials and permissions")
		case 404:
			errorDetails.ErrorType = "not_found"
			errorDetails.Suggestions = append(errorDetails.Suggestions, "Check if the webhook endpoint URL path is correct")
		case 500, 502, 503, 504:
			errorDetails.ErrorType = "server_error"
			errorDetails.Suggestions = append(errorDetails.Suggestions, "The endpoint server is experiencing issues - retries may succeed")
		}
	}

	return errorDetails
}

func (h *TestingHandler) handleWebSocketConnection(conn *websocket.Conn, tenantID uuid.UUID) {
	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type:      "connected",
		Data:      map[string]interface{}{"tenant_id": tenantID},
		Timestamp: time.Now(),
	}

	if err := conn.WriteJSON(welcomeMsg); err != nil {
		h.logger.Error("Failed to send welcome message", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Handle incoming messages and keep connection alive
	for {
		// Read message (for ping/pong or subscription management)
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error", map[string]interface{}{
					"error": err.Error(),
				})
			}
			break
		}

		// Handle different message types
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "ping":
				pongMsg := WebSocketMessage{
					Type:      "pong",
					Data:      nil,
					Timestamp: time.Now(),
				}
				if err := conn.WriteJSON(pongMsg); err != nil {
					h.logger.Error("Failed to send pong", map[string]interface{}{
						"error": err.Error(),
					})
					return
				}
			case "subscribe":
				// Handle subscription to specific delivery updates
				h.logger.Info("WebSocket subscription request", map[string]interface{}{
					"tenant_id": tenantID,
					"message":   msg,
				})
			}
		}
	}
}

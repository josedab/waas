package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TestEndpointHandler struct {
	logger *utils.Logger
}

// TestEndpointReceive represents a received webhook on a test endpoint
type TestEndpointReceive struct {
	ID          uuid.UUID         `json:"id"`
	EndpointID  uuid.UUID         `json:"endpoint_id"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Body        json.RawMessage   `json:"body"`
	QueryParams map[string]string `json:"query_params"`
	ReceivedAt  time.Time         `json:"received_at"`
	SourceIP    string            `json:"source_ip"`
	UserAgent   string            `json:"user_agent"`
}

func NewTestEndpointHandler(logger *utils.Logger) *TestEndpointHandler {
	return &TestEndpointHandler{
		logger: logger,
	}
}

// ReceiveTestWebhook handles incoming webhooks to test endpoints
// This would typically be at a route like /test/:endpoint_id
func (h *TestEndpointHandler) ReceiveTestWebhook(c *gin.Context) {
	// Parse endpoint ID from URL
	endpointIDStr := c.Param("endpoint_id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid endpoint ID format",
		})
		return
	}

	// Read request body
	bodyBytes, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read request body", map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// Extract headers
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Extract query parameters
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Create receive record
	receive := TestEndpointReceive{
		ID:          uuid.New(),
		EndpointID:  endpointID,
		Method:      c.Request.Method,
		Headers:     headers,
		Body:        json.RawMessage(bodyBytes),
		QueryParams: queryParams,
		ReceivedAt:  time.Now(),
		SourceIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	}

	h.logger.Info("Test webhook received", map[string]interface{}{
		"receive_id":  receive.ID,
		"endpoint_id": endpointID,
		"method":      receive.Method,
		"source_ip":   receive.SourceIP,
		"body_size":   len(bodyBytes),
	})

	// In a real implementation, you would:
	// 1. Store this receive record in a database or cache
	// 2. Broadcast to WebSocket connections for real-time updates
	// 3. Check if the endpoint exists and is not expired

	// For now, return a success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Webhook received successfully",
		"receive_id": receive.ID,
		"timestamp":  receive.ReceivedAt,
	})
}

// GetTestEndpointReceives handles GET /test/:endpoint_id/receives
func (h *TestEndpointHandler) GetTestEndpointReceives(c *gin.Context) {
	// Parse endpoint ID from URL
	endpointIDStr := c.Param("endpoint_id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid endpoint ID format",
		})
		return
	}

	// In a real implementation, you would fetch from database/cache
	// For now, return a mock response
	receives := []TestEndpointReceive{
		{
			ID:         uuid.New(),
			EndpointID: endpointID,
			Method:     "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "Webhook-Test/1.0",
			},
			Body:       json.RawMessage(`{"test": "data", "timestamp": "2024-01-01T12:00:00Z"}`),
			ReceivedAt: time.Now().Add(-5 * time.Minute),
			SourceIP:   "192.168.1.100",
			UserAgent:  "Webhook-Test/1.0",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoint_id": endpointID,
		"receives":    receives,
		"total":       len(receives),
	})
}

// GetTestEndpointReceive handles GET /test/:endpoint_id/receives/:receive_id
func (h *TestEndpointHandler) GetTestEndpointReceive(c *gin.Context) {
	// Parse endpoint ID from URL
	endpointIDStr := c.Param("endpoint_id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid endpoint ID format",
		})
		return
	}

	// Parse receive ID from URL
	receiveIDStr := c.Param("receive_id")
	receiveID, err := uuid.Parse(receiveIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid receive ID format",
		})
		return
	}

	// In a real implementation, you would fetch from database/cache
	// For now, return a mock response
	receive := TestEndpointReceive{
		ID:         receiveID,
		EndpointID: endpointID,
		Method:     "POST",
		Headers: map[string]string{
			"Content-Type":     "application/json",
			"User-Agent":       "Webhook-Test/1.0",
			"X-Webhook-ID":     "test-123",
			"X-Signature":      "sha256=abc123",
			"Accept":           "*/*",
			"Content-Length":   "45",
		},
		Body: json.RawMessage(`{"test": "data", "timestamp": "2024-01-01T12:00:00Z"}`),
		QueryParams: map[string]string{
			"source": "test",
			"version": "1.0",
		},
		ReceivedAt: time.Now().Add(-5 * time.Minute),
		SourceIP:   "192.168.1.100",
		UserAgent:  "Webhook-Test/1.0",
	}

	c.JSON(http.StatusOK, receive)
}

// ClearTestEndpointReceives handles DELETE /test/:endpoint_id/receives
func (h *TestEndpointHandler) ClearTestEndpointReceives(c *gin.Context) {
	// Parse endpoint ID from URL
	endpointIDStr := c.Param("endpoint_id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid endpoint ID format",
		})
		return
	}

	h.logger.Info("Test endpoint receives cleared", map[string]interface{}{
		"endpoint_id": endpointID,
	})

	// In a real implementation, you would clear from database/cache

	c.JSON(http.StatusOK, gin.H{
		"message":     "Test endpoint receives cleared",
		"endpoint_id": endpointID,
		"cleared_at":  time.Now(),
	})
}
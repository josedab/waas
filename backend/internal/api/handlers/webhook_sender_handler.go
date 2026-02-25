package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SendWebhookRequest represents a single webhook send request
type SendWebhookRequest struct {
	EndpointID *uuid.UUID        `json:"endpoint_id,omitempty"`
	Payload    json.RawMessage   `json:"payload" binding:"required"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// BatchSendWebhookRequest represents a batch webhook send request
type BatchSendWebhookRequest struct {
	EndpointIDs []uuid.UUID       `json:"endpoint_ids,omitempty"` // If empty, send to all active endpoints
	Payload     json.RawMessage   `json:"payload" binding:"required"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// SendWebhookResponse represents the response for webhook send requests
type SendWebhookResponse struct {
	DeliveryID  uuid.UUID `json:"delivery_id"`
	EndpointID  uuid.UUID `json:"endpoint_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// BatchSendWebhookResponse represents the response for batch webhook send requests
type BatchSendWebhookResponse struct {
	Deliveries []SendWebhookResponse `json:"deliveries"`
	Total      int                   `json:"total"`
	Queued     int                   `json:"queued"`
	Failed     int                   `json:"failed"`
}

// SendWebhook sends a webhook to one or all endpoints
// @Summary Send webhook
// @Description Send a webhook payload to a specific endpoint or all active endpoints
// @Tags webhooks
// @Accept json
// @Produce json
// @Param request body SendWebhookRequest true "Webhook send request"
// @Success 202 {object} SendWebhookResponse "Webhook queued for delivery successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format, payload too large, or no active endpoints"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 403 {object} map[string]interface{} "Forbidden - access denied to specified endpoint"
// @Failure 404 {object} map[string]interface{} "Webhook endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/send [post]
func (h *WebhookHandler) SendWebhook(c *gin.Context) {
	var req SendWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Validate payload size (1MB limit)
	const maxPayloadSize = 1024 * 1024 // 1MB
	if len(req.Payload) > maxPayloadSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "PAYLOAD_TOO_LARGE",
				"message": "Webhook payload exceeds maximum size limit",
				"details": map[string]interface{}{
					"maxSizeBytes":    maxPayloadSize,
					"actualSizeBytes": len(req.Payload),
				},
			},
		})
		return
	}

	// Validate payload is valid JSON
	var payloadTest interface{}
	if err := json.Unmarshal(req.Payload, &payloadTest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_PAYLOAD",
				"message": "Webhook payload must be valid JSON",
				"details": err.Error(),
			},
		})
		return
	}

	var endpoints []*models.WebhookEndpoint
	var err error

	if req.EndpointID != nil {
		// Send to specific endpoint
		endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), *req.EndpointID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": map[string]interface{}{
						"code":    "ENDPOINT_NOT_FOUND",
						"message": "Webhook endpoint not found",
					},
				})
				return
			}

			h.logger.Error("Failed to get webhook endpoint", map[string]interface{}{
				"error":       err.Error(),
				"endpoint_id": *req.EndpointID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"code":    "DATABASE_ERROR",
					"message": "Failed to retrieve webhook endpoint",
				},
			})
			return
		}

		// Verify tenant ownership
		if endpoint.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": map[string]interface{}{
					"code":    "FORBIDDEN",
					"message": "Access denied to this webhook endpoint",
				},
			})
			return
		}

		// Check if endpoint is active
		if !endpoint.IsActive {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]interface{}{
					"code":    "ENDPOINT_INACTIVE",
					"message": "Webhook endpoint is not active",
				},
			})
			return
		}

		endpoints = []*models.WebhookEndpoint{endpoint}
	} else {
		// Send to all active endpoints for the tenant
		endpoints, err = h.webhookRepo.GetActiveByTenantID(c.Request.Context(), tenantID)
		if err != nil {
			h.logger.Error("Failed to get active webhook endpoints", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"code":    "DATABASE_ERROR",
					"message": "Failed to retrieve webhook endpoints",
				},
			})
			return
		}

		if len(endpoints) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]interface{}{
					"code":    "NO_ACTIVE_ENDPOINTS",
					"message": "No active webhook endpoints found for tenant",
				},
			})
			return
		}
	}

	// Process each endpoint
	var responses []SendWebhookResponse
	for _, endpoint := range endpoints {
		response, err := h.queueWebhookDelivery(c.Request.Context(), endpoint, req.Payload, req.Headers)
		if err != nil {
			h.logger.Error("Failed to queue webhook delivery", map[string]interface{}{
				"error":       err.Error(),
				"endpoint_id": endpoint.ID,
				"tenant_id":   tenantID,
			})
			// Continue with other endpoints, don't fail the entire request
			continue
		}
		responses = append(responses, *response)
	}

	if len(responses) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "DELIVERY_QUEUE_ERROR",
				"message": "Failed to queue webhook deliveries",
			},
		})
		return
	}

	// If single endpoint, return single response
	if req.EndpointID != nil {
		c.JSON(http.StatusAccepted, responses[0])
		return
	}

	// Multiple endpoints, return array
	c.JSON(http.StatusAccepted, gin.H{
		"deliveries": responses,
		"total":      len(responses),
	})
}

// BatchSendWebhook sends a webhook to multiple endpoints
// @Summary Batch send webhook
// @Description Send a webhook payload to multiple endpoints in a single request
// @Tags webhooks
// @Accept json
// @Produce json
// @Param request body BatchSendWebhookRequest true "Batch webhook send request"
// @Success 202 {object} BatchSendWebhookResponse "Webhooks queued for delivery successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format, payload too large, or no active endpoints"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/send/batch [post]
func (h *WebhookHandler) BatchSendWebhook(c *gin.Context) {
	var req BatchSendWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Get tenant from context
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Validate payload size (1MB limit)
	const maxPayloadSize = 1024 * 1024 // 1MB
	if len(req.Payload) > maxPayloadSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "PAYLOAD_TOO_LARGE",
				"message": "Webhook payload exceeds maximum size limit",
				"details": map[string]interface{}{
					"maxSizeBytes":    maxPayloadSize,
					"actualSizeBytes": len(req.Payload),
				},
			},
		})
		return
	}

	// Validate payload is valid JSON
	var payloadTest interface{}
	if err := json.Unmarshal(req.Payload, &payloadTest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_PAYLOAD",
				"message": "Webhook payload must be valid JSON",
				"details": err.Error(),
			},
		})
		return
	}

	var endpoints []*models.WebhookEndpoint
	var err error

	if len(req.EndpointIDs) > 0 {
		// Send to specific endpoints
		for _, endpointID := range req.EndpointIDs {
			endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
			if err != nil {
				h.logger.Warn("Failed to get webhook endpoint for batch send", map[string]interface{}{
					"error":       err.Error(),
					"endpoint_id": endpointID,
				})
				continue // Skip invalid endpoints
			}

			// Verify tenant ownership
			if endpoint.TenantID != tenantID {
				h.logger.Warn("Skipping endpoint not owned by tenant", map[string]interface{}{
					"endpoint_id": endpointID,
					"tenant_id":   tenantID,
				})
				continue
			}

			// Only include active endpoints
			if endpoint.IsActive {
				endpoints = append(endpoints, endpoint)
			}
		}
	} else {
		// Send to all active endpoints for the tenant
		endpoints, err = h.webhookRepo.GetActiveByTenantID(c.Request.Context(), tenantID)
		if err != nil {
			h.logger.Error("Failed to get active webhook endpoints for batch send", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"code":    "DATABASE_ERROR",
					"message": "Failed to retrieve webhook endpoints",
				},
			})
			return
		}
	}

	if len(endpoints) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "NO_ACTIVE_ENDPOINTS",
				"message": "No active webhook endpoints found for batch send",
			},
		})
		return
	}

	// Process each endpoint
	var deliveries []SendWebhookResponse
	var queued, failed int

	for _, endpoint := range endpoints {
		response, err := h.queueWebhookDelivery(c.Request.Context(), endpoint, req.Payload, req.Headers)
		if err != nil {
			h.logger.Error("Failed to queue webhook delivery in batch", map[string]interface{}{
				"error":       err.Error(),
				"endpoint_id": endpoint.ID,
				"tenant_id":   tenantID,
			})
			failed++
			continue
		}
		deliveries = append(deliveries, *response)
		queued++
	}

	response := BatchSendWebhookResponse{
		Deliveries: deliveries,
		Total:      len(endpoints),
		Queued:     queued,
		Failed:     failed,
	}

	c.JSON(http.StatusAccepted, response)
}

// Helper methods

// queueWebhookDelivery queues a webhook delivery for processing
func (h *WebhookHandler) queueWebhookDelivery(ctx context.Context, endpoint *models.WebhookEndpoint, payload json.RawMessage, headers map[string]string) (*SendWebhookResponse, error) {
	// Generate delivery ID
	deliveryID := uuid.New()

	// Create delivery attempt record
	attempt := &models.DeliveryAttempt{
		ID:            deliveryID,
		EndpointID:    endpoint.ID,
		PayloadHash:   h.calculatePayloadHash(payload),
		PayloadSize:   len(payload),
		Status:        queue.StatusPending,
		AttemptNumber: 1,
		ScheduledAt:   time.Now(),
		CreatedAt:     time.Now(),
	}

	// Save delivery attempt to database
	if err := h.deliveryAttemptRepo.Create(ctx, attempt); err != nil {
		return nil, fmt.Errorf("failed to create delivery attempt: %w", err)
	}

	// Generate webhook signature
	signature, err := h.generateWebhookSignature(payload, endpoint.SecretHash)
	if err != nil {
		return nil, fmt.Errorf("failed to generate webhook signature: %w", err)
	}

	// Create delivery message
	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpoint.ID,
		TenantID:      endpoint.TenantID,
		Payload:       payload,
		Headers:       headers,
		AttemptNumber: 1,
		ScheduledAt:   time.Now(),
		Signature:     signature,
		MaxAttempts:   endpoint.RetryConfig.MaxAttempts,
	}

	// Publish to delivery queue
	if err := h.publisher.PublishDelivery(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to publish delivery message: %w", err)
	}

	h.logger.Info("Webhook delivery queued", map[string]interface{}{
		"delivery_id": deliveryID,
		"endpoint_id": endpoint.ID,
		"tenant_id":   endpoint.TenantID,
	})

	return &SendWebhookResponse{
		DeliveryID:  deliveryID,
		EndpointID:  endpoint.ID,
		Status:      queue.StatusPending,
		ScheduledAt: attempt.ScheduledAt,
	}, nil
}

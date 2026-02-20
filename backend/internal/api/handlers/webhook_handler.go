// Package handlers provides HTTP handlers for the webhook service platform API
package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebhookHandler handles webhook endpoint CRUD and webhook delivery operations.
type WebhookHandler struct {
	webhookRepo         repository.WebhookEndpointRepository
	deliveryAttemptRepo repository.DeliveryAttemptRepository
	publisher           queue.PublisherInterface
	logger              *utils.Logger
	urlValidator        *utils.URLValidator
}

// CreateWebhookEndpointRequest is the request payload for registering a new webhook endpoint.
type CreateWebhookEndpointRequest struct {
	URL           string              `json:"url" binding:"required"`
	CustomHeaders map[string]string   `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfigRequest `json:"retry_config,omitempty"`
}

// UpdateWebhookEndpointRequest is the request payload for updating an existing webhook endpoint.
type UpdateWebhookEndpointRequest struct {
	URL           *string             `json:"url,omitempty"`
	CustomHeaders map[string]string   `json:"custom_headers,omitempty"`
	RetryConfig   *RetryConfigRequest `json:"retry_config,omitempty"`
	IsActive      *bool               `json:"is_active,omitempty"`
}

// RetryConfigRequest holds retry parameters for webhook endpoint configuration.
type RetryConfigRequest struct {
	MaxAttempts       int `json:"max_attempts,omitempty"`
	InitialDelayMs    int `json:"initial_delay_ms,omitempty"`
	MaxDelayMs        int `json:"max_delay_ms,omitempty"`
	BackoffMultiplier int `json:"backoff_multiplier,omitempty"`
}

// WebhookEndpointResponse is the API representation of a webhook endpoint.
type WebhookEndpointResponse struct {
	ID            uuid.UUID                 `json:"id"`
	URL           string                    `json:"url"`
	Secret        string                    `json:"secret,omitempty"` // Only returned on creation
	IsActive      bool                      `json:"is_active"`
	RetryConfig   models.RetryConfiguration `json:"retry_config"`
	CustomHeaders map[string]string         `json:"custom_headers"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

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

func NewWebhookHandler(webhookRepo repository.WebhookEndpointRepository, deliveryAttemptRepo repository.DeliveryAttemptRepository, publisher queue.PublisherInterface, logger *utils.Logger) *WebhookHandler {
	return &WebhookHandler{
		webhookRepo:         webhookRepo,
		deliveryAttemptRepo: deliveryAttemptRepo,
		publisher:           publisher,
		logger:              logger,
		urlValidator:        utils.NewURLValidator(),
	}
}

// CreateWebhookEndpoint creates a new webhook endpoint
// @Summary Create webhook endpoint
// @Description Create a new webhook endpoint for receiving webhook notifications
// @Tags webhooks
// @Accept json
// @Produce json
// @Param request body CreateWebhookEndpointRequest true "Webhook endpoint creation request"
// @Success 201 {object} WebhookEndpointResponse "Webhook endpoint created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/endpoints [post]
func (h *WebhookHandler) CreateWebhookEndpoint(c *gin.Context) {
	var req CreateWebhookEndpointRequest
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

	// Get tenant from context (set by auth middleware)
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
		return
	}

	// Validate URL format and accessibility
	if err := h.urlValidator.ValidateWebhookURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_URL",
				"message": "Invalid webhook URL",
				"details": err.Error(),
			},
		})
		return
	}

	// Check URL accessibility (optional - can be disabled for testing)
	if err := h.urlValidator.CheckURLAccessibility(c.Request.Context(), req.URL); err != nil {
		h.logger.Warn("Webhook URL accessibility check failed", map[string]interface{}{
			"url":   req.URL,
			"error": err.Error(),
		})
		// Note: We log the warning but don't fail the request
		// This allows for URLs that might be temporarily unreachable during setup
	}

	// Generate secret for webhook signing
	secret, secretHash, err := h.generateSecret()
	if err != nil {
		h.logger.Error("Failed to generate webhook secret", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate webhook secret",
			},
		})
		return
	}

	// Set default retry configuration if not provided
	retryConfig := models.RetryConfiguration{
		MaxAttempts:       5,
		InitialDelayMs:    1000,
		MaxDelayMs:        300000, // 5 minutes
		BackoffMultiplier: 2,
	}

	if req.RetryConfig != nil {
		if req.RetryConfig.MaxAttempts > 0 {
			retryConfig.MaxAttempts = req.RetryConfig.MaxAttempts
		}
		if req.RetryConfig.InitialDelayMs > 0 {
			retryConfig.InitialDelayMs = req.RetryConfig.InitialDelayMs
		}
		if req.RetryConfig.MaxDelayMs > 0 {
			retryConfig.MaxDelayMs = req.RetryConfig.MaxDelayMs
		}
		if req.RetryConfig.BackoffMultiplier > 0 {
			retryConfig.BackoffMultiplier = req.RetryConfig.BackoffMultiplier
		}
	}

	// Create webhook endpoint
	endpoint := &models.WebhookEndpoint{
		ID:            uuid.New(),
		TenantID:      tenantID.(uuid.UUID),
		URL:           req.URL,
		SecretHash:    secretHash,
		IsActive:      true,
		RetryConfig:   retryConfig,
		CustomHeaders: req.CustomHeaders,
	}

	// Validate the endpoint
	if err := models.ValidateWebhookEndpoint(endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid webhook endpoint data",
				"details": err.Error(),
			},
		})
		return
	}

	// Save to database
	if err := h.webhookRepo.Create(c.Request.Context(), endpoint); err != nil {
		h.logger.Error("Failed to create webhook endpoint", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenantID,
			"url":       req.URL,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "DATABASE_ERROR",
				"message": "Failed to create webhook endpoint",
			},
		})
		return
	}

	h.logger.Info("Webhook endpoint created", map[string]interface{}{
		"endpoint_id": endpoint.ID,
		"tenant_id":   tenantID,
		"url":         req.URL,
	})

	// Return response with secret (only returned on creation)
	response := WebhookEndpointResponse{
		ID:            endpoint.ID,
		URL:           endpoint.URL,
		Secret:        secret,
		IsActive:      endpoint.IsActive,
		RetryConfig:   endpoint.RetryConfig,
		CustomHeaders: endpoint.CustomHeaders,
		CreatedAt:     endpoint.CreatedAt,
		UpdatedAt:     endpoint.UpdatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// GetWebhookEndpoints retrieves all webhook endpoints for the authenticated tenant
// @Summary List webhook endpoints
// @Description Get a paginated list of webhook endpoints for the authenticated tenant
// @Tags webhooks
// @Accept json
// @Produce json
// @Param limit query int false "Number of results to return (max 100)" default(50)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} map[string]interface{} "List of webhook endpoints with pagination info"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/endpoints [get]
func (h *WebhookHandler) GetWebhookEndpoints(c *gin.Context) {
	// Get tenant from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get endpoints from database
	endpoints, err := h.webhookRepo.GetByTenantID(c.Request.Context(), tenantID.(uuid.UUID), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get webhook endpoints", map[string]interface{}{
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

	// Convert to response format (without secrets)
	responses := make([]WebhookEndpointResponse, len(endpoints))
	for i, endpoint := range endpoints {
		responses[i] = WebhookEndpointResponse{
			ID:            endpoint.ID,
			URL:           endpoint.URL,
			IsActive:      endpoint.IsActive,
			RetryConfig:   endpoint.RetryConfig,
			CustomHeaders: endpoint.CustomHeaders,
			CreatedAt:     endpoint.CreatedAt,
			UpdatedAt:     endpoint.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoints": responses,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(responses),
		},
	})
}

// GetWebhookEndpoint retrieves a specific webhook endpoint by ID
// @Summary Get webhook endpoint
// @Description Get details of a specific webhook endpoint by its ID
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook endpoint ID" format(uuid)
// @Success 200 {object} WebhookEndpointResponse "Webhook endpoint details"
// @Failure 400 {object} map[string]interface{} "Invalid endpoint ID format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 403 {object} map[string]interface{} "Forbidden - access denied to this endpoint"
// @Failure 404 {object} map[string]interface{} "Webhook endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/endpoints/{id} [get]
func (h *WebhookHandler) GetWebhookEndpoint(c *gin.Context) {
	// Parse endpoint ID
	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_ID",
				"message": "Invalid endpoint ID format",
			},
		})
		return
	}

	// Get tenant from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
		return
	}

	// Get endpoint from database
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
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
			"endpoint_id": endpointID,
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
	if endpoint.TenantID != tenantID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": map[string]interface{}{
				"code":    "FORBIDDEN",
				"message": "Access denied to this webhook endpoint",
			},
		})
		return
	}

	// Return response (without secret)
	response := WebhookEndpointResponse{
		ID:            endpoint.ID,
		URL:           endpoint.URL,
		IsActive:      endpoint.IsActive,
		RetryConfig:   endpoint.RetryConfig,
		CustomHeaders: endpoint.CustomHeaders,
		CreatedAt:     endpoint.CreatedAt,
		UpdatedAt:     endpoint.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateWebhookEndpoint updates an existing webhook endpoint
// @Summary Update webhook endpoint
// @Description Update configuration of an existing webhook endpoint
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook endpoint ID" format(uuid)
// @Param request body UpdateWebhookEndpointRequest true "Webhook endpoint update request"
// @Success 200 {object} WebhookEndpointResponse "Webhook endpoint updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 403 {object} map[string]interface{} "Forbidden - access denied to this endpoint"
// @Failure 404 {object} map[string]interface{} "Webhook endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/endpoints/{id} [put]
func (h *WebhookHandler) UpdateWebhookEndpoint(c *gin.Context) {
	// Parse endpoint ID
	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_ID",
				"message": "Invalid endpoint ID format",
			},
		})
		return
	}

	var req UpdateWebhookEndpointRequest
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
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
		return
	}

	// Get existing endpoint
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]interface{}{
					"code":    "ENDPOINT_NOT_FOUND",
					"message": "Webhook endpoint not found",
				},
			})
			return
		}

		h.logger.Error("Failed to get webhook endpoint for update", map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID,
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
	if endpoint.TenantID != tenantID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": map[string]interface{}{
				"code":    "FORBIDDEN",
				"message": "Access denied to this webhook endpoint",
			},
		})
		return
	}

	// Update fields if provided
	if req.URL != nil {
		if err := h.urlValidator.ValidateWebhookURL(*req.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]interface{}{
					"code":    "INVALID_URL",
					"message": "Invalid webhook URL",
					"details": err.Error(),
				},
			})
			return
		}

		// Check URL accessibility (optional - log warning only)
		if err := h.urlValidator.CheckURLAccessibility(c.Request.Context(), *req.URL); err != nil {
			h.logger.Warn("Updated webhook URL accessibility check failed", map[string]interface{}{
				"url":   *req.URL,
				"error": err.Error(),
			})
		}

		endpoint.URL = *req.URL
	}

	if req.IsActive != nil {
		endpoint.IsActive = *req.IsActive
	}

	if req.CustomHeaders != nil {
		endpoint.CustomHeaders = req.CustomHeaders
	}

	if req.RetryConfig != nil {
		if req.RetryConfig.MaxAttempts > 0 {
			endpoint.RetryConfig.MaxAttempts = req.RetryConfig.MaxAttempts
		}
		if req.RetryConfig.InitialDelayMs > 0 {
			endpoint.RetryConfig.InitialDelayMs = req.RetryConfig.InitialDelayMs
		}
		if req.RetryConfig.MaxDelayMs > 0 {
			endpoint.RetryConfig.MaxDelayMs = req.RetryConfig.MaxDelayMs
		}
		if req.RetryConfig.BackoffMultiplier > 0 {
			endpoint.RetryConfig.BackoffMultiplier = req.RetryConfig.BackoffMultiplier
		}
	}

	// Validate updated endpoint
	if err := models.ValidateWebhookEndpoint(endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid webhook endpoint data",
				"details": err.Error(),
			},
		})
		return
	}

	// Update in database
	if err := h.webhookRepo.Update(c.Request.Context(), endpoint); err != nil {
		h.logger.Error("Failed to update webhook endpoint", map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID,
			"tenant_id":   tenantID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "DATABASE_ERROR",
				"message": "Failed to update webhook endpoint",
			},
		})
		return
	}

	h.logger.Info("Webhook endpoint updated", map[string]interface{}{
		"endpoint_id": endpointID,
		"tenant_id":   tenantID,
	})

	// Return updated endpoint (without secret)
	response := WebhookEndpointResponse{
		ID:            endpoint.ID,
		URL:           endpoint.URL,
		IsActive:      endpoint.IsActive,
		RetryConfig:   endpoint.RetryConfig,
		CustomHeaders: endpoint.CustomHeaders,
		CreatedAt:     endpoint.CreatedAt,
		UpdatedAt:     endpoint.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteWebhookEndpoint deletes a webhook endpoint
// @Summary Delete webhook endpoint
// @Description Delete a webhook endpoint and stop all future deliveries to it
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook endpoint ID" format(uuid)
// @Success 204 "Webhook endpoint deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid endpoint ID format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 403 {object} map[string]interface{} "Forbidden - access denied to this endpoint"
// @Failure 404 {object} map[string]interface{} "Webhook endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /webhooks/endpoints/{id} [delete]
func (h *WebhookHandler) DeleteWebhookEndpoint(c *gin.Context) {
	// Parse endpoint ID
	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "INVALID_ID",
				"message": "Invalid endpoint ID format",
			},
		})
		return
	}

	// Get tenant from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
		return
	}

	// Get existing endpoint to verify ownership
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]interface{}{
					"code":    "ENDPOINT_NOT_FOUND",
					"message": "Webhook endpoint not found",
				},
			})
			return
		}

		h.logger.Error("Failed to get webhook endpoint for deletion", map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID,
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
	if endpoint.TenantID != tenantID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": map[string]interface{}{
				"code":    "FORBIDDEN",
				"message": "Access denied to this webhook endpoint",
			},
		})
		return
	}

	// Delete from database
	if err := h.webhookRepo.Delete(c.Request.Context(), endpointID); err != nil {
		h.logger.Error("Failed to delete webhook endpoint", map[string]interface{}{
			"error":       err.Error(),
			"endpoint_id": endpointID,
			"tenant_id":   tenantID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "DATABASE_ERROR",
				"message": "Failed to delete webhook endpoint",
			},
		})
		return
	}

	h.logger.Info("Webhook endpoint deleted", map[string]interface{}{
		"endpoint_id": endpointID,
		"tenant_id":   tenantID,
	})

	c.JSON(http.StatusNoContent, nil)
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
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
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
			if strings.Contains(err.Error(), "not found") {
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
		if endpoint.TenantID != tenantID.(uuid.UUID) {
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
		endpoints, err = h.webhookRepo.GetActiveByTenantID(c.Request.Context(), tenantID.(uuid.UUID))
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
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant not found in context",
			},
		})
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
			if endpoint.TenantID != tenantID.(uuid.UUID) {
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
		endpoints, err = h.webhookRepo.GetActiveByTenantID(c.Request.Context(), tenantID.(uuid.UUID))
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

// calculatePayloadHash calculates SHA256 hash of the payload
func (h *WebhookHandler) calculatePayloadHash(payload json.RawMessage) string {
	hasher := sha256.New()
	hasher.Write(payload)
	return hex.EncodeToString(hasher.Sum(nil))
}

// generateWebhookSignature generates HMAC signature for webhook payload
func (h *WebhookHandler) generateWebhookSignature(payload json.RawMessage, secretHash string) (string, error) {
	// For now, we'll use the secret hash directly as the signing key
	// In a production system, you'd want to retrieve the actual secret
	// This is a simplified implementation for the webhook sending API
	signature := utils.GenerateWebhookSignature(payload, secretHash, "sha256")
	return fmt.Sprintf("sha256=%s", signature), nil
}

// generateSecret generates a random secret and its hash for webhook signing
func (h *WebhookHandler) generateSecret() (secret, hash string, err error) {
	// Generate 32 random bytes
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	// Convert to hex string
	secret = hex.EncodeToString(secretBytes)

	// Create hash for storage
	hasher := sha256.New()
	hasher.Write([]byte(secret))
	hash = hex.EncodeToString(hasher.Sum(nil))

	return secret, hash, nil
}

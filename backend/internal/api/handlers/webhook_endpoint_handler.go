// Package handlers provides HTTP handlers for the webhook service platform API
package handlers

import (
	"errors"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"strconv"
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
	tenantID, ok := RequireTenantID(c)
	if !ok {
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
		TenantID:      tenantID,
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
	tenantID, ok := RequireTenantID(c)
	if !ok {
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
	endpoints, err := h.webhookRepo.GetByTenantID(c.Request.Context(), tenantID, limit, offset)
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
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Get endpoint from database
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
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
	if endpoint.TenantID != tenantID {
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
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Get existing endpoint
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
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
	if endpoint.TenantID != tenantID {
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
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	// Get existing endpoint to verify ownership
	endpoint, err := h.webhookRepo.GetByID(c.Request.Context(), endpointID)
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
	if endpoint.TenantID != tenantID {
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

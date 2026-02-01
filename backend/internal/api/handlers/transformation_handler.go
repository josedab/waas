package handlers

import (
	"net/http"
	"strconv"

	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/transform"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TransformationHandler handles transformation API requests
type TransformationHandler struct {
	transformRepo repository.TransformationRepository
	engine        *transform.Engine
	logger        *utils.Logger
}

// CreateTransformationRequest represents a transformation creation request
type CreateTransformationRequest struct {
	Name        string                  `json:"name" binding:"required,min=1,max=255"`
	Description string                  `json:"description,omitempty"`
	Script      string                  `json:"script" binding:"required"`
	Enabled     *bool                   `json:"enabled,omitempty"`
	Config      *TransformConfigRequest `json:"config,omitempty"`
}

// UpdateTransformationRequest represents a transformation update request
type UpdateTransformationRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Script      *string                 `json:"script,omitempty"`
	Enabled     *bool                   `json:"enabled,omitempty"`
	Config      *TransformConfigRequest `json:"config,omitempty"`
}

// TransformConfigRequest represents transformation configuration
type TransformConfigRequest struct {
	TimeoutMs     *int  `json:"timeout_ms,omitempty"`
	MaxMemoryMB   *int  `json:"max_memory_mb,omitempty"`
	AllowHTTP     *bool `json:"allow_http,omitempty"`
	EnableLogging *bool `json:"enable_logging,omitempty"`
}

// TestTransformationRequest represents a dry-run transformation test
type TestTransformationRequest struct {
	Script       string      `json:"script" binding:"required"`
	InputPayload interface{} `json:"input_payload" binding:"required"`
}

// TestTransformationResponse represents the test result
type TestTransformationResponse struct {
	Success         bool        `json:"success"`
	OutputPayload   interface{} `json:"output_payload,omitempty"`
	Error           string      `json:"error,omitempty"`
	ExecutionTimeMs int64       `json:"execution_time_ms"`
	Logs            []string    `json:"logs,omitempty"`
}

// LinkEndpointRequest links a transformation to an endpoint
type LinkEndpointRequest struct {
	TransformationID string `json:"transformation_id" binding:"required,uuid"`
	Priority         int    `json:"priority"`
}

// NewTransformationHandler creates a new transformation handler
func NewTransformationHandler(
	transformRepo repository.TransformationRepository,
	logger *utils.Logger,
) *TransformationHandler {
	return &TransformationHandler{
		transformRepo: transformRepo,
		engine:        transform.NewEngine(transform.DefaultEngineConfig()),
		logger:        logger,
	}
}

// RegisterRoutes registers transformation routes
func (h *TransformationHandler) RegisterRoutes(r *gin.RouterGroup) {
	transformations := r.Group("/transformations")
	{
		transformations.POST("", h.CreateTransformation)
		transformations.GET("", h.ListTransformations)
		transformations.GET("/:id", h.GetTransformation)
		transformations.PATCH("/:id", h.UpdateTransformation)
		transformations.DELETE("/:id", h.DeleteTransformation)
		transformations.POST("/test", h.TestTransformation)
		transformations.GET("/:id/logs", h.GetTransformationLogs)
	}

	// Endpoint transformation links
	r.POST("/endpoints/:endpoint_id/transformations", h.LinkTransformation)
	r.DELETE("/endpoints/:endpoint_id/transformations/:transformation_id", h.UnlinkTransformation)
	r.GET("/endpoints/:endpoint_id/transformations", h.GetEndpointTransformations)
}

// CreateTransformation creates a new transformation
// @Summary Create a new transformation
// @Tags Transformations
// @Accept json
// @Produce json
// @Param request body CreateTransformationRequest true "Transformation details"
// @Success 201 {object} models.Transformation
// @Failure 400 {object} ErrorResponse
// @Router /transformations [post]
func (h *TransformationHandler) CreateTransformation(c *gin.Context) {
	var req CreateTransformationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// Validate script syntax
	if err := h.engine.ValidateScript(req.Script); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_SCRIPT",
			Message: err.Error(),
		})
		return
	}

	tenantID, _ := c.Get("tenant_id")

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	config := models.DefaultTransformConfig()
	if req.Config != nil {
		if req.Config.TimeoutMs != nil {
			config.TimeoutMs = *req.Config.TimeoutMs
		}
		if req.Config.MaxMemoryMB != nil {
			config.MaxMemoryMB = *req.Config.MaxMemoryMB
		}
		if req.Config.AllowHTTP != nil {
			config.AllowHTTP = *req.Config.AllowHTTP
		}
		if req.Config.EnableLogging != nil {
			config.EnableLogging = *req.Config.EnableLogging
		}
	}

	transformation := &models.Transformation{
		TenantID:    tenantID.(uuid.UUID),
		Name:        req.Name,
		Description: req.Description,
		Script:      req.Script,
		Enabled:     enabled,
		Config:      config,
	}

	if err := h.transformRepo.Create(c.Request.Context(), transformation); err != nil {
		h.logger.Error("Failed to create transformation", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "CREATE_FAILED",
			Message: "Failed to create transformation",
		})
		return
	}

	c.JSON(http.StatusCreated, transformation)
}

// ListTransformations lists all transformations for the tenant
// @Summary List transformations
// @Tags Transformations
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {array} models.Transformation
// @Router /transformations [get]
func (h *TransformationHandler) ListTransformations(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	transformations, err := h.transformRepo.GetByTenantID(c.Request.Context(), tenantID.(uuid.UUID), perPage, offset)
	if err != nil {
		h.logger.Error("Failed to list transformations", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "LIST_FAILED",
			Message: "Failed to list transformations",
		})
		return
	}

	c.JSON(http.StatusOK, transformations)
}

// GetTransformation gets a transformation by ID
// @Summary Get a transformation
// @Tags Transformations
// @Produce json
// @Param id path string true "Transformation ID"
// @Success 200 {object} models.Transformation
// @Failure 404 {object} ErrorResponse
// @Router /transformations/{id} [get]
func (h *TransformationHandler) GetTransformation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid transformation ID",
		})
		return
	}

	transformation, err := h.transformRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    "NOT_FOUND",
				Message: "Transformation not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
		})
		return
	}

	c.JSON(http.StatusOK, transformation)
}

// UpdateTransformation updates a transformation
// @Summary Update a transformation
// @Tags Transformations
// @Accept json
// @Produce json
// @Param id path string true "Transformation ID"
// @Param request body UpdateTransformationRequest true "Update details"
// @Success 200 {object} models.Transformation
// @Router /transformations/{id} [patch]
func (h *TransformationHandler) UpdateTransformation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid transformation ID",
		})
		return
	}

	var req UpdateTransformationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	transformation, err := h.transformRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    "NOT_FOUND",
				Message: "Transformation not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
		})
		return
	}

	// Apply updates
	if req.Name != nil {
		transformation.Name = *req.Name
	}
	if req.Description != nil {
		transformation.Description = *req.Description
	}
	if req.Script != nil {
		if err := h.engine.ValidateScript(*req.Script); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_SCRIPT",
				Message: err.Error(),
			})
			return
		}
		transformation.Script = *req.Script
	}
	if req.Enabled != nil {
		transformation.Enabled = *req.Enabled
	}
	if req.Config != nil {
		if req.Config.TimeoutMs != nil {
			transformation.Config.TimeoutMs = *req.Config.TimeoutMs
		}
		if req.Config.MaxMemoryMB != nil {
			transformation.Config.MaxMemoryMB = *req.Config.MaxMemoryMB
		}
		if req.Config.AllowHTTP != nil {
			transformation.Config.AllowHTTP = *req.Config.AllowHTTP
		}
		if req.Config.EnableLogging != nil {
			transformation.Config.EnableLogging = *req.Config.EnableLogging
		}
	}

	if err := h.transformRepo.Update(c.Request.Context(), transformation); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "UPDATE_FAILED",
			Message: "Failed to update transformation",
		})
		return
	}

	c.JSON(http.StatusOK, transformation)
}

// DeleteTransformation deletes a transformation
// @Summary Delete a transformation
// @Tags Transformations
// @Param id path string true "Transformation ID"
// @Success 204
// @Router /transformations/{id} [delete]
func (h *TransformationHandler) DeleteTransformation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid transformation ID",
		})
		return
	}

	if err := h.transformRepo.Delete(c.Request.Context(), id); err != nil {
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    "NOT_FOUND",
				Message: "Transformation not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "DELETE_FAILED",
			Message: "Failed to delete transformation",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestTransformation performs a dry-run of a transformation script
// @Summary Test a transformation script
// @Tags Transformations
// @Accept json
// @Produce json
// @Param request body TestTransformationRequest true "Test request"
// @Success 200 {object} TestTransformationResponse
// @Router /transformations/test [post]
func (h *TransformationHandler) TestTransformation(c *gin.Context) {
	var req TestTransformationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// Validate script
	if err := h.engine.ValidateScript(req.Script); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_SCRIPT",
			Message: err.Error(),
		})
		return
	}

	// Execute transformation
	result, err := h.engine.Transform(c.Request.Context(), req.Script, req.InputPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "EXECUTION_FAILED",
			Message: err.Error(),
		})
		return
	}

	response := TestTransformationResponse{
		Success:         result.Success,
		OutputPayload:   result.Output,
		Error:           result.Error,
		ExecutionTimeMs: result.ExecutionTimeMs,
		Logs:            result.Logs,
	}

	c.JSON(http.StatusOK, response)
}

// GetTransformationLogs gets execution logs for a transformation
// @Summary Get transformation logs
// @Tags Transformations
// @Param id path string true "Transformation ID"
// @Param limit query int false "Max logs to return" default(50)
// @Success 200 {array} models.TransformationLog
// @Router /transformations/{id}/logs [get]
func (h *TransformationHandler) GetTransformationLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid transformation ID",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	logs, err := h.transformRepo.GetLogsByTransformationID(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "GET_LOGS_FAILED",
			Message: "Failed to get transformation logs",
		})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// LinkTransformation links a transformation to an endpoint
// @Summary Link transformation to endpoint
// @Tags Transformations
// @Accept json
// @Param endpoint_id path string true "Endpoint ID"
// @Param request body LinkEndpointRequest true "Link request"
// @Success 204
// @Router /endpoints/{endpoint_id}/transformations [post]
func (h *TransformationHandler) LinkTransformation(c *gin.Context) {
	endpointID, err := uuid.Parse(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid endpoint ID",
		})
		return
	}

	var req LinkEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	transformationID, err := uuid.Parse(req.TransformationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid transformation ID",
		})
		return
	}

	if err := h.transformRepo.LinkToEndpoint(c.Request.Context(), endpointID, transformationID, req.Priority); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "LINK_FAILED",
			Message: "Failed to link transformation to endpoint",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// UnlinkTransformation unlinks a transformation from an endpoint
// @Summary Unlink transformation from endpoint
// @Tags Transformations
// @Param endpoint_id path string true "Endpoint ID"
// @Param transformation_id path string true "Transformation ID"
// @Success 204
// @Router /endpoints/{endpoint_id}/transformations/{transformation_id} [delete]
func (h *TransformationHandler) UnlinkTransformation(c *gin.Context) {
	endpointID, err := uuid.Parse(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid endpoint ID",
		})
		return
	}

	transformationID, err := uuid.Parse(c.Param("transformation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid transformation ID",
		})
		return
	}

	if err := h.transformRepo.UnlinkFromEndpoint(c.Request.Context(), endpointID, transformationID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "UNLINK_FAILED",
			Message: "Failed to unlink transformation from endpoint",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetEndpointTransformations gets all transformations linked to an endpoint
// @Summary Get endpoint transformations
// @Tags Transformations
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200 {array} models.Transformation
// @Router /endpoints/{endpoint_id}/transformations [get]
func (h *TransformationHandler) GetEndpointTransformations(c *gin.Context) {
	endpointID, err := uuid.Parse(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "Invalid endpoint ID",
		})
		return
	}

	transformations, err := h.transformRepo.GetByEndpointID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Failed to get endpoint transformations",
		})
		return
	}

	c.JSON(http.StatusOK, transformations)
}

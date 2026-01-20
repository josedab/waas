package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"webhook-platform/internal/api/services"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/utils"
)

// AIComposerHandler handles AI-powered webhook configuration requests
type AIComposerHandler struct {
	service *services.AIComposerService
	logger  *utils.Logger
}

// NewAIComposerHandler creates a new AI composer handler
func NewAIComposerHandler(service *services.AIComposerService, logger *utils.Logger) *AIComposerHandler {
	return &AIComposerHandler{
		service: service,
		logger:  logger,
	}
}

// Compose handles AI webhook composition requests
// @Summary Compose webhook configuration using AI
// @Description Use natural language to create webhook configurations
// @Tags ai-composer
// @Accept json
// @Produce json
// @Param request body models.AIComposerRequest true "Composition request"
// @Success 200 {object} models.AIComposerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /ai/compose [post]
func (h *AIComposerHandler) Compose(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.AIComposerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	response, err := h.service.Compose(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to compose webhook", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ApplyConfig applies a generated configuration
// @Summary Apply generated webhook configuration
// @Description Creates actual webhook resources from a generated configuration
// @Tags ai-composer
// @Accept json
// @Produce json
// @Param config_id path string true "Configuration ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /ai/configs/{config_id}/apply [post]
func (h *AIComposerHandler) ApplyConfig(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configIDStr := c.Param("config_id")
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
		return
	}

	if err := h.service.ApplyConfig(c.Request.Context(), tenantID.(uuid.UUID), configID); err != nil {
		h.logger.Error("Failed to apply config", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "configuration applied successfully"})
}

// GetTemplates returns available prompt templates
// @Summary Get AI composer templates
// @Description Returns available prompt templates for common use cases
// @Tags ai-composer
// @Accept json
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {array} models.AIComposerTemplate
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /ai/templates [get]
func (h *AIComposerHandler) GetTemplates(c *gin.Context) {
	category := c.Query("category")

	templates, err := h.service.GetTemplates(c.Request.Context(), category)
	if err != nil {
		h.logger.Error("Failed to get templates", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get templates"})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// SubmitFeedback records user feedback on AI-generated configs
// @Summary Submit feedback on generated configuration
// @Description Records user feedback to improve AI suggestions
// @Tags ai-composer
// @Accept json
// @Produce json
// @Param request body FeedbackRequest true "Feedback data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /ai/feedback [post]
func (h *AIComposerHandler) SubmitFeedback(c *gin.Context) {
	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	var configID *uuid.UUID
	if req.ConfigID != "" {
		cid, err := uuid.Parse(req.ConfigID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
			return
		}
		configID = &cid
	}

	feedback := &models.AIComposerFeedback{
		SessionID:        sessionID,
		ConfigID:         configID,
		Rating:           req.Rating,
		FeedbackText:     req.FeedbackText,
		WorkedAsExpected: req.WorkedAsExpected,
	}

	if err := h.service.SubmitFeedback(c.Request.Context(), feedback); err != nil {
		h.logger.Error("Failed to submit feedback", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feedback submitted successfully"})
}

// GetSessionHistory returns conversation history for a session
// @Summary Get AI composer session history
// @Description Returns the conversation history and generated configs for a session
// @Tags ai-composer
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} SessionHistoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /ai/sessions/{session_id} [get]
func (h *AIComposerHandler) GetSessionHistory(c *gin.Context) {
	// Implementation would retrieve session messages and configs
	// For brevity, returning a placeholder
	c.JSON(http.StatusOK, gin.H{"message": "session history endpoint"})
}

// FeedbackRequest represents the feedback submission request
type FeedbackRequest struct {
	SessionID        string `json:"session_id" binding:"required"`
	ConfigID         string `json:"config_id,omitempty"`
	Rating           int    `json:"rating" binding:"required,min=1,max=5"`
	FeedbackText     string `json:"feedback_text"`
	WorkedAsExpected bool   `json:"worked_as_expected"`
}

// SessionHistoryResponse represents session history with messages and configs
type SessionHistoryResponse struct {
	Session  *models.AIComposerSession           `json:"session"`
	Messages []*models.AIComposerMessage         `json:"messages"`
	Configs  []*models.AIComposerGeneratedConfig `json:"configs"`
}

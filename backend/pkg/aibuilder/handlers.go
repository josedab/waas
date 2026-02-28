package aibuilder

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the AI conversational webhook builder.
type Handler struct {
	service *Service
}

// NewHandler creates a new AI builder handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers AI builder routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/ai-builder")
	{
		g.POST("/message", h.SendMessage)
		g.GET("/conversations", h.ListConversations)
		g.GET("/conversations/:id", h.GetConversation)
		g.GET("/conversations/:id/messages", h.GetMessages)
		g.DELETE("/conversations/:id", h.DeleteConversation)
		g.POST("/debug", h.DebugDelivery)
	}
}

// SendMessage handles a user message in the AI conversation.
// @Summary Send message to AI webhook builder
// @Tags ai-builder
// @Accept json
// @Produce json
// @Param request body SendMessageRequest true "Message"
// @Success 200 {object} SendMessageResponse
// @Security ApiKeyAuth
// @Router /ai-builder/message [post]
func (h *Handler) SendMessage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id required"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.SendMessage(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListConversations returns all conversations for the tenant.
// @Summary List AI builder conversations
// @Tags ai-builder
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} ConversationSummary
// @Security ApiKeyAuth
// @Router /ai-builder/conversations [get]
func (h *Handler) ListConversations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	convs, err := h.service.ListConversations(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversations": convs})
}

// GetConversation returns a specific conversation.
// @Summary Get AI builder conversation
// @Tags ai-builder
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} Conversation
// @Security ApiKeyAuth
// @Router /ai-builder/conversations/{id} [get]
func (h *Handler) GetConversation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	conv, err := h.service.GetConversation(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conv)
}

// GetMessages returns messages for a conversation.
// @Summary Get conversation messages
// @Tags ai-builder
// @Produce json
// @Param id path string true "Conversation ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} Message
// @Security ApiKeyAuth
// @Router /ai-builder/conversations/{id}/messages [get]
func (h *Handler) GetMessages(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	convID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	msgs, err := h.service.GetMessages(c.Request.Context(), tenantID, convID, limit, offset)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

// DeleteConversation removes a conversation.
// @Summary Delete AI builder conversation
// @Tags ai-builder
// @Param id path string true "Conversation ID"
// @Success 204
// @Security ApiKeyAuth
// @Router /ai-builder/conversations/{id} [delete]
func (h *Handler) DeleteConversation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.DeleteConversation(c.Request.Context(), tenantID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// DebugDelivery asks the AI to diagnose a delivery issue.
// @Summary AI-powered delivery debugging
// @Tags ai-builder
// @Accept json
// @Produce json
// @Param request body DebugRequest true "Debug request"
// @Success 200 {object} DebugResponse
// @Security ApiKeyAuth
// @Router /ai-builder/debug [post]
func (h *Handler) DebugDelivery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req DebugRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.DebugDelivery(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

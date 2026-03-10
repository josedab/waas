package nlbuilder

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the NL webhook builder.
type Handler struct {
	service *Service
}

// NewHandler creates a new NL builder handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all NL builder routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/nl-builder")
	{
		group.POST("/conversations", h.StartConversation)
		group.GET("/conversations", h.ListConversations)
		group.GET("/conversations/:id", h.GetConversation)
		group.POST("/chat", h.Chat)
		group.POST("/conversations/:id/apply", h.ApplyConfig)
		group.POST("/conversations/:id/routing-rules", h.GenerateRoutingRules)
		group.POST("/validate-config", h.ValidateConfig)
		group.POST("/refinement-suggestions", h.GetRefinementSuggestions)
	}
}

// StartConversation begins a new builder chat session.
// @Summary Start NL builder conversation
// @Tags nl-builder
// @Produce json
// @Success 201 {object} Conversation
// @Router /nl-builder/conversations [post]
func (h *Handler) StartConversation(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	conv, err := h.service.StartConversation(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, conv)
}

// ListConversations lists all conversations for the tenant.
// @Summary List NL builder conversations
// @Tags nl-builder
// @Produce json
// @Success 200 {array} Conversation
// @Router /nl-builder/conversations [get]
func (h *Handler) ListConversations(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	convs, err := h.service.ListConversations(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, convs)
}

// GetConversation retrieves a specific conversation.
// @Summary Get NL builder conversation
// @Tags nl-builder
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} Conversation
// @Router /nl-builder/conversations/{id} [get]
func (h *Handler) GetConversation(c *gin.Context) {
	conv, err := h.service.GetConversation(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, conv)
}

// Chat processes a chat message.
// @Summary Send message to NL builder
// @Tags nl-builder
// @Accept json
// @Produce json
// @Param request body ChatRequest true "Chat message"
// @Success 200 {object} ChatResponse
// @Router /nl-builder/chat [post]
func (h *Handler) Chat(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Chat(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ApplyConfig finalizes and applies the generated configuration.
// @Summary Apply generated webhook config
// @Tags nl-builder
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} GeneratedConfig
// @Router /nl-builder/conversations/{id}/apply [post]
func (h *Handler) ApplyConfig(c *gin.Context) {
	config, err := h.service.ApplyConfig(c.Param("id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

// GenerateRoutingRules creates routing rules from natural language.
func (h *Handler) GenerateRoutingRules(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rules, err := h.service.GenerateRoutingRules(tenantID, c.Param("id"), req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"routing_rules": rules})
}

// ValidateConfig validates a generated webhook configuration.
func (h *Handler) ValidateConfig(c *gin.Context) {
	var config GeneratedConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.service.ValidateConfig(&config)
	c.JSON(http.StatusOK, result)
}

// GetRefinementSuggestions returns AI improvement suggestions for a config.
func (h *Handler) GetRefinementSuggestions(c *gin.Context) {
	var config GeneratedConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	suggestions := h.service.GetRefinementSuggestions(&config)
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

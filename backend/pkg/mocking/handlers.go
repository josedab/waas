package mocking

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for webhook mocking
type Handler struct {
	service *Service
}

// NewHandler creates a new mocking handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers mocking routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	mocks := router.Group("/mocks")
	{
		mocks.POST("/endpoints", h.CreateEndpoint)
		mocks.GET("/endpoints", h.ListEndpoints)
		mocks.GET("/endpoints/:id", h.GetEndpoint)
		mocks.PUT("/endpoints/:id", h.UpdateEndpoint)
		mocks.DELETE("/endpoints/:id", h.DeleteEndpoint)
		mocks.POST("/endpoints/:id/trigger", h.TriggerMock)
		mocks.GET("/endpoints/:id/deliveries", h.ListDeliveries)
	}

	templates := router.Group("/mocks/templates")
	{
		templates.POST("", h.CreateTemplate)
		templates.GET("", h.ListTemplates)
		templates.GET("/:id", h.GetTemplate)
		templates.DELETE("/:id", h.DeleteTemplate)
	}

	utils := router.Group("/mocks")
	{
		utils.POST("/preview", h.PreviewPayload)
		utils.GET("/faker-types", h.GetFakerTypes)
	}
}

// CreateEndpoint godoc
//
//	@Summary		Create mock endpoint
//	@Description	Create a new mock webhook endpoint
//	@Tags			mocking
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateMockEndpointRequest	true	"Mock endpoint request"
//	@Success		201		{object}	MockEndpoint
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints [post]
func (h *Handler) CreateEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateMockEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	endpoint, err := h.service.CreateMockEndpoint(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, endpoint)
}

// ListEndpoints godoc
//
//	@Summary		List mock endpoints
//	@Description	Get a list of mock webhook endpoints
//	@Tags			mocking
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints [get]
func (h *Handler) ListEndpoints(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	endpoints, total, err := h.service.ListMockEndpoints(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpoints,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetEndpoint godoc
//
//	@Summary		Get mock endpoint
//	@Description	Get details of a specific mock endpoint
//	@Tags			mocking
//	@Produce		json
//	@Param			id	path		string	true	"Endpoint ID"
//	@Success		200	{object}	MockEndpoint
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints/{id} [get]
func (h *Handler) GetEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	endpoint, err := h.service.GetMockEndpoint(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if endpoint == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mock endpoint not found"})
		return
	}

	c.JSON(http.StatusOK, endpoint)
}

// UpdateEndpoint godoc
//
//	@Summary		Update mock endpoint
//	@Description	Update a mock endpoint
//	@Tags			mocking
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Endpoint ID"
//	@Param			request	body		UpdateMockEndpointRequest	true	"Update request"
//	@Success		200		{object}	MockEndpoint
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints/{id} [put]
func (h *Handler) UpdateEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	var req UpdateMockEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	endpoint, err := h.service.UpdateMockEndpoint(c.Request.Context(), tenantID, endpointID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, endpoint)
}

// DeleteEndpoint godoc
//
//	@Summary		Delete mock endpoint
//	@Description	Delete a mock endpoint
//	@Tags			mocking
//	@Produce		json
//	@Param			id	path	string	true	"Endpoint ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints/{id} [delete]
func (h *Handler) DeleteEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	if err := h.service.DeleteMockEndpoint(c.Request.Context(), tenantID, endpointID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// TriggerMock godoc
//
//	@Summary		Trigger mock webhooks
//	@Description	Trigger one or more mock webhook deliveries
//	@Tags			mocking
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Endpoint ID"
//	@Param			request	body		TriggerMockRequest	true	"Trigger request"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints/{id}/trigger [post]
func (h *Handler) TriggerMock(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	var req TriggerMockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = TriggerMockRequest{Count: 1}
	}

	deliveries, err := h.service.TriggerMock(c.Request.Context(), tenantID, endpointID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deliveries": deliveries,
		"count":      len(deliveries),
	})
}

// ListDeliveries godoc
//
//	@Summary		List mock deliveries
//	@Description	Get delivery history for a mock endpoint
//	@Tags			mocking
//	@Produce		json
//	@Param			id		path		string	true	"Endpoint ID"
//	@Param			limit	query		int		false	"Limit"		default(20)
//	@Param			offset	query		int		false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/endpoints/{id}/deliveries [get]
func (h *Handler) ListDeliveries(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	deliveries, total, err := h.service.ListDeliveries(c.Request.Context(), tenantID, endpointID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deliveries": deliveries,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// CreateTemplate godoc
//
//	@Summary		Create mock template
//	@Description	Create a reusable mock template
//	@Tags			mocking
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateTemplateRequest	true	"Template request"
//	@Success		201		{object}	MockTemplate
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/templates [post]
func (h *Handler) CreateTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	template, err := h.service.CreateTemplate(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// ListTemplates godoc
//
//	@Summary		List mock templates
//	@Description	Get a list of mock templates
//	@Tags			mocking
//	@Produce		json
//	@Param			include_public	query		bool	false	"Include public templates"	default(true)
//	@Param			limit			query		int		false	"Limit"						default(20)
//	@Param			offset			query		int		false	"Offset"					default(0)
//	@Success		200				{object}	map[string]interface{}
//	@Failure		401				{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/templates [get]
func (h *Handler) ListTemplates(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	includePublic := c.DefaultQuery("include_public", "true") == "true"
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	templates, total, err := h.service.ListTemplates(c.Request.Context(), tenantID, includePublic, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetTemplate godoc
//
//	@Summary		Get mock template
//	@Description	Get details of a mock template
//	@Tags			mocking
//	@Produce		json
//	@Param			id	path		string	true	"Template ID"
//	@Success		200	{object}	MockTemplate
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/templates/{id} [get]
func (h *Handler) GetTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	templateID := c.Param("id")
	template, err := h.service.GetTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if template == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// DeleteTemplate godoc
//
//	@Summary		Delete mock template
//	@Description	Delete a mock template
//	@Tags			mocking
//	@Produce		json
//	@Param			id	path	string	true	"Template ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/templates/{id} [delete]
func (h *Handler) DeleteTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	templateID := c.Param("id")
	if err := h.service.DeleteTemplate(c.Request.Context(), tenantID, templateID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// PreviewPayload godoc
//
//	@Summary		Preview mock payload
//	@Description	Generate a preview of mock payloads from a template
//	@Tags			mocking
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PayloadTemplate	true	"Template"
//	@Param			count	query		int				false	"Number of previews"	default(1)
//	@Success		200		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/preview [post]
func (h *Handler) PreviewPayload(c *gin.Context) {
	var template PayloadTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	count := 1
	if cnt := c.Query("count"); cnt != "" {
		fmt.Sscanf(cnt, "%d", &count)
	}

	previews, err := h.service.GeneratePreview(&template, count)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"previews": previews})
}

// GetFakerTypes godoc
//
//	@Summary		Get faker types
//	@Description	Get available faker data types
//	@Tags			mocking
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/mocks/faker-types [get]
func (h *Handler) GetFakerTypes(c *gin.Context) {
	types := h.service.GetFakerTypes()
	c.JSON(http.StatusOK, gin.H{"faker_types": types})
}

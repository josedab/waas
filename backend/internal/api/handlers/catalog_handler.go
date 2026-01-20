package handlers

import (
	"fmt"
	"net/http"

	"webhook-platform/pkg/catalog"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CatalogHandler handles event catalog HTTP requests
type CatalogHandler struct {
	service *catalog.Service
	logger  *utils.Logger
}

// NewCatalogHandler creates a new catalog handler
func NewCatalogHandler(service *catalog.Service, logger *utils.Logger) *CatalogHandler {
	return &CatalogHandler{
		service: service,
		logger:  logger,
	}
}

// CreateEventType creates a new event type
// @Summary Create event type
// @Tags catalog
// @Accept json
// @Produce json
// @Param request body catalog.CreateEventTypeRequest true "Event type"
// @Success 201 {object} catalog.EventType
// @Router /catalog/events [post]
func (h *CatalogHandler) CreateEventType(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var req catalog.CreateEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	et, err := h.service.CreateEventType(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to create event type", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "CREATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, et)
}

// GetEventType retrieves an event type
// @Summary Get event type
// @Tags catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {object} catalog.EventType
// @Router /catalog/events/{id} [get]
func (h *CatalogHandler) GetEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	et, err := h.service.GetEventType(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "NOT_FOUND", Message: "Event type not found"})
		return
	}

	c.JSON(http.StatusOK, et)
}

// GetEventTypeBySlug retrieves an event type by slug
// @Summary Get event type by slug
// @Tags catalog
// @Produce json
// @Param slug path string true "Event type slug"
// @Success 200 {object} catalog.EventType
// @Router /catalog/events/slug/{slug} [get]
func (h *CatalogHandler) GetEventTypeBySlug(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	slug := c.Param("slug")
	et, err := h.service.GetEventTypeBySlug(c.Request.Context(), tenantID, slug)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "NOT_FOUND", Message: "Event type not found"})
		return
	}

	c.JSON(http.StatusOK, et)
}

// UpdateEventType updates an event type
// @Summary Update event type
// @Tags catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Param request body catalog.UpdateEventTypeRequest true "Update data"
// @Success 200 {object} catalog.EventType
// @Router /catalog/events/{id} [patch]
func (h *CatalogHandler) UpdateEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	var req catalog.UpdateEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	et, err := h.service.UpdateEventType(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "UPDATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, et)
}

// DeleteEventType deletes an event type
// @Summary Delete event type
// @Tags catalog
// @Param id path string true "Event type ID"
// @Success 204
// @Router /catalog/events/{id} [delete]
func (h *CatalogHandler) DeleteEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	if err := h.service.DeleteEventType(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DELETE_FAILED", Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeprecateEventType marks an event type as deprecated
// @Summary Deprecate event type
// @Tags catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Param request body catalog.DeprecateEventTypeRequest true "Deprecation info"
// @Success 200 {object} catalog.EventType
// @Router /catalog/events/{id}/deprecate [post]
func (h *CatalogHandler) DeprecateEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	var req catalog.DeprecateEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	et, err := h.service.DeprecateEventType(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "DEPRECATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, et)
}

// SearchEventTypes searches the event catalog
// @Summary Search event types
// @Tags catalog
// @Produce json
// @Param q query string false "Search query"
// @Param category query string false "Category filter"
// @Param status query string false "Status filter"
// @Param tags query []string false "Tags filter"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} catalog.CatalogSearchResult
// @Router /catalog/events [get]
func (h *CatalogHandler) SearchEventTypes(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	params := &catalog.CatalogSearchParams{
		TenantID:  tenantID,
		Query:     c.Query("q"),
		Category:  c.Query("category"),
		Status:    c.DefaultQuery("status", "active"),
		SortBy:    c.DefaultQuery("sort_by", "name"),
		SortOrder: c.DefaultQuery("sort_order", "asc"),
	}

	if tags := c.QueryArray("tags"); len(tags) > 0 {
		params.Tags = tags
	}

	if limit := c.Query("limit"); limit != "" {
		var l int
		fmt.Sscanf(limit, "%d", &l)
		params.Limit = l
	}

	if offset := c.Query("offset"); offset != "" {
		var o int
		fmt.Sscanf(offset, "%d", &o)
		params.Offset = o
	}

	result, err := h.service.SearchEventTypes(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "SEARCH_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// PublishVersion publishes a new version of an event type
// @Summary Publish new version
// @Tags catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Param request body catalog.PublishVersionRequest true "Version info"
// @Success 201 {object} catalog.EventVersion
// @Router /catalog/events/{id}/versions [post]
func (h *CatalogHandler) PublishVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	var req catalog.PublishVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	ev, err := h.service.PublishVersion(c.Request.Context(), id, &req, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "PUBLISH_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ev)
}

// GetVersions returns all versions of an event type
// @Summary Get event versions
// @Tags catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {array} catalog.EventVersion
// @Router /catalog/events/{id}/versions [get]
func (h *CatalogHandler) GetVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	versions, err := h.service.GetVersions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "GET_VERSIONS_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// ListCategories lists all event categories
// @Summary List categories
// @Tags catalog
// @Produce json
// @Success 200 {array} catalog.EventCategory
// @Router /catalog/categories [get]
func (h *CatalogHandler) ListCategories(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	categories, err := h.service.ListCategories(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "LIST_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// CreateCategory creates a new event category
// @Summary Create category
// @Tags catalog
// @Accept json
// @Produce json
// @Param request body catalog.CreateCategoryRequest true "Category data"
// @Success 201 {object} catalog.EventCategory
// @Router /catalog/categories [post]
func (h *CatalogHandler) CreateCategory(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var req catalog.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	cat, err := h.service.CreateCategory(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "CREATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cat)
}

// SubscribeToEvent subscribes an endpoint to an event type
// @Summary Subscribe to event
// @Tags catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Param request body catalog.SubscribeRequest true "Subscription data"
// @Success 201 {object} catalog.EventSubscription
// @Router /catalog/events/{id}/subscribe [post]
func (h *CatalogHandler) SubscribeToEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	var req catalog.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	sub, err := h.service.Subscribe(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "SUBSCRIBE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// UnsubscribeFromEvent unsubscribes an endpoint from an event type
// @Summary Unsubscribe from event
// @Tags catalog
// @Param id path string true "Event type ID"
// @Param endpoint_id path string true "Endpoint ID"
// @Success 204
// @Router /catalog/events/{id}/subscribe/{endpoint_id} [delete]
func (h *CatalogHandler) UnsubscribeFromEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	endpointID, err := uuid.Parse(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid endpoint ID"})
		return
	}

	if err := h.service.Unsubscribe(c.Request.Context(), eventID, endpointID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "UNSUBSCRIBE_FAILED", Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetOpenAPISpec generates an OpenAPI spec for the catalog
// @Summary Get OpenAPI spec
// @Tags catalog
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /catalog/openapi [get]
func (h *CatalogHandler) GetOpenAPISpec(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	spec, err := h.service.GenerateOpenAPISpec(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "SPEC_FAILED", Message: err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", spec)
}

// GetDocumentation returns documentation for an event type
// @Summary Get event documentation
// @Tags catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {array} catalog.EventDocumentation
// @Router /catalog/events/{id}/docs [get]
func (h *CatalogHandler) GetDocumentation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid event type ID"})
		return
	}

	docs, err := h.service.GetDocumentation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "GET_DOCS_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// RegisterCatalogRoutes registers catalog API routes
func RegisterCatalogRoutes(r *gin.RouterGroup, h *CatalogHandler) {
	cat := r.Group("/catalog")
	{
		// Event types
		cat.GET("/events", h.SearchEventTypes)
		cat.POST("/events", h.CreateEventType)
		cat.GET("/events/:id", h.GetEventType)
		cat.GET("/events/slug/:slug", h.GetEventTypeBySlug)
		cat.PATCH("/events/:id", h.UpdateEventType)
		cat.DELETE("/events/:id", h.DeleteEventType)
		cat.POST("/events/:id/deprecate", h.DeprecateEventType)

		// Versions
		cat.GET("/events/:id/versions", h.GetVersions)
		cat.POST("/events/:id/versions", h.PublishVersion)

		// Subscriptions
		cat.POST("/events/:id/subscribe", h.SubscribeToEvent)
		cat.DELETE("/events/:id/subscribe/:endpoint_id", h.UnsubscribeFromEvent)

		// Documentation
		cat.GET("/events/:id/docs", h.GetDocumentation)

		// Categories
		cat.GET("/categories", h.ListCategories)
		cat.POST("/categories", h.CreateCategory)

		// OpenAPI spec
		cat.GET("/openapi", h.GetOpenAPISpec)
	}
}

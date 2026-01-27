package catalog

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler provides HTTP handlers for the event catalog
type Handler struct {
	service *Service
}

// NewHandler creates a new catalog handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers catalog routes on the given router group
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	catalog := router.Group("/catalog")
	{
		catalog.POST("/event-types", h.CreateEventType)
		catalog.GET("/event-types", h.ListEventTypes)
		catalog.GET("/event-types/:id", h.GetEventType)
		catalog.PUT("/event-types/:id", h.UpdateEventType)
		catalog.POST("/event-types/:id/versions", h.PublishVersion)
		catalog.GET("/event-types/:id/versions", h.GetVersions)
		catalog.POST("/event-types/:id/validate", h.ValidatePayload)
		catalog.POST("/event-types/:id/deprecate", h.DeprecateEventType)
		catalog.POST("/event-types/:id/subscribe", h.SubscribeEndpoint)
		catalog.GET("/event-types/:id/subscribers", h.ListSubscribers)
		catalog.POST("/event-types/:id/generate/:language", h.GenerateSDKTypes)
		catalog.GET("/search", h.SearchCatalog)
		catalog.GET("/event-types/:id/changelog", h.GetChangelog)
		catalog.GET("/event-types/:id/portal", h.GetDocPortal)
		catalog.POST("/event-types/:id/validate-with-mode", h.ValidatePayloadWithMode)
		catalog.GET("/notifications/breaking-changes", h.GetBreakingChangeNotifications)
		catalog.GET("/config/validation", h.GetValidationConfig)
		catalog.PUT("/config/validation", h.UpdateValidationConfig)
		catalog.DELETE("/event-types/:id", h.DeleteEventType)
		catalog.GET("/categories", h.ListCategories)
		catalog.POST("/categories", h.CreateCategory)
		catalog.DELETE("/event-types/:id/subscribe/:endpointId", h.UnsubscribeEndpoint)
		catalog.GET("/event-types/:id/documentation", h.GetDocumentation)
		catalog.PUT("/event-types/:id/documentation/:section", h.SaveDocumentation)
		catalog.POST("/openapi-spec", h.GenerateOpenAPISpec)
	}
}

// @Summary Create event type
// @Tags Catalog
// @Accept json
// @Produce json
// @Success 201 {object} EventType
// @Router /catalog/event-types [post]
func (h *Handler) CreateEventType(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}

	var req CreateEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	et, err := h.service.CreateEventType(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, et)
}

// @Summary List event types
// @Tags Catalog
// @Produce json
// @Success 200 {object} CatalogSearchResult
// @Router /catalog/event-types [get]
func (h *Handler) ListEventTypes(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}

	params := &CatalogSearchParams{
		TenantID:  tenantID,
		Query:     c.Query("q"),
		Category:  c.Query("category"),
		Status:    c.DefaultQuery("status", ""),
		SortBy:    c.DefaultQuery("sort_by", "name"),
		SortOrder: c.DefaultQuery("sort_order", "asc"),
	}

	if tags := c.QueryArray("tags"); len(tags) > 0 {
		params.Tags = tags
	}

	result, err := h.service.SearchEventTypes(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Get event type
// @Tags Catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {object} EventType
// @Router /catalog/event-types/{id} [get]
func (h *Handler) GetEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	et, err := h.service.GetEventType(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Event type not found"}})
		return
	}

	c.JSON(http.StatusOK, et)
}

// @Summary Update event type
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {object} EventType
// @Router /catalog/event-types/{id} [put]
func (h *Handler) UpdateEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	var req UpdateEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	et, err := h.service.UpdateEventType(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, et)
}

// @Summary Publish new version
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Success 201 {object} EventVersion
// @Router /catalog/event-types/{id}/versions [post]
func (h *Handler) PublishVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	var req PublishVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	ev, err := h.service.PublishVersion(c.Request.Context(), id, &req, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "PUBLISH_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, ev)
}

// @Summary List versions
// @Tags Catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {array} EventVersion
// @Router /catalog/event-types/{id}/versions [get]
func (h *Handler) GetVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	versions, err := h.service.GetVersions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "GET_VERSIONS_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// @Summary Validate payload
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {object} map[string]interface{}
// @Router /catalog/event-types/{id}/validate [post]
func (h *Handler) ValidatePayload(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	var payload json.RawMessage
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	valid, issues := h.service.ValidatePayload(c.Request.Context(), id, payload)
	c.JSON(http.StatusOK, gin.H{
		"valid":  valid,
		"issues": issues,
	})
}

// @Summary Deprecate event type
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {object} EventType
// @Router /catalog/event-types/{id}/deprecate [post]
func (h *Handler) DeprecateEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	var req DeprecateEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	et, err := h.service.DeprecateEventType(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DEPRECATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, et)
}

// @Summary Subscribe endpoint to event type
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Event type ID"
// @Success 201 {object} EventSubscription
// @Router /catalog/event-types/{id}/subscribe [post]
func (h *Handler) SubscribeEndpoint(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	sub, err := h.service.Subscribe(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SUBSCRIBE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// @Summary List subscribers
// @Tags Catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Success 200 {array} EventSubscription
// @Router /catalog/event-types/{id}/subscribers [get]
func (h *Handler) ListSubscribers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	subs, err := h.service.ListSubscribers(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscribers": subs})
}

// @Summary Generate SDK types
// @Tags Catalog
// @Produce json
// @Param id path string true "Event type ID"
// @Param language path string true "Language (go, python, typescript)"
// @Success 200 {object} map[string]string
// @Router /catalog/event-types/{id}/generate/{language} [post]
func (h *Handler) GenerateSDKTypes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}

	language := c.Param("language")
	code, err := h.service.GenerateSDKTypes(c.Request.Context(), id, language)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "GENERATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"language": language,
		"code":     code,
	})
}

// @Summary Search catalog
// @Tags Catalog
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {object} CatalogSearchResult
// @Router /catalog/search [get]
func (h *Handler) SearchCatalog(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}

	params := &CatalogSearchParams{
		TenantID: tenantID,
		Query:    c.Query("q"),
	}

	result, err := h.service.SearchEventTypes(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SEARCH_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetChangelog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	changelog, err := h.service.GenerateChangelog(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, changelog)
}

func (h *Handler) GetDocPortal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	portal, err := h.service.GenerateDocPortal(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, portal)
}

func (h *Handler) ValidatePayloadWithMode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	var req struct {
		Payload json.RawMessage `json:"payload" binding:"required"`
		Mode    string          `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	mode := ValidationMode(req.Mode)
	if mode == "" {
		mode = ValidationModeStrict
	}
	result := h.service.ValidatePayloadWithMode(c.Request.Context(), id, req.Payload, mode)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetBreakingChangeNotifications(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}
	notifications, err := h.service.GetBreakingChangeNotifications(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "FETCH_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (h *Handler) GetValidationConfig(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}
	config := h.service.GetValidationConfig(c.Request.Context(), tenantID)
	c.JSON(http.StatusOK, config)
}

// @Summary Update validation config
// @Tags Catalog
// @Accept json
// @Produce json
// @Success 200 {object} SchemaValidationConfig
// @Router /catalog/config/validation [put]
func (h *Handler) UpdateValidationConfig(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}

	var config SchemaValidationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	config.TenantID = tenantID

	if err := h.service.UpdateValidationConfig(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, &config)
}

func (h *Handler) DeleteEventType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	if err := h.service.DeleteEventType(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DELETE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "event type deleted"})
}

func (h *Handler) ListCategories(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}
	categories, err := h.service.ListCategories(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *Handler) CreateCategory(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	cat, err := h.service.CreateCategory(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

func (h *Handler) UnsubscribeEndpoint(c *gin.Context) {
	eventTypeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	endpointID, err := uuid.Parse(c.Param("endpointId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid endpoint ID"}})
		return
	}
	if err := h.service.Unsubscribe(c.Request.Context(), eventTypeID, endpointID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UNSUBSCRIBE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unsubscribed"})
}

func (h *Handler) GetDocumentation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	docs, err := h.service.GetDocumentation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "GET_DOCS_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"documentation": docs})
}

func (h *Handler) SaveDocumentation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event type ID"}})
		return
	}
	section := c.Param("section")
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	if err := h.service.SaveDocumentation(c.Request.Context(), id, section, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SAVE_DOCS_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "documentation saved"})
}

func (h *Handler) GenerateOpenAPISpec(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant"}})
		return
	}
	spec, err := h.service.GenerateOpenAPISpec(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "GENERATE_FAILED", "message": err.Error()}})
		return
	}
	c.Data(http.StatusOK, "application/json", spec)
}

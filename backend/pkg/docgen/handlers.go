package docgen

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// Handler provides HTTP handlers for API documentation generation
type Handler struct {
	service *Service
}

// NewHandler creates a new docgen handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers docgen routes on the given router group
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	docgen := router.Group("/docgen")
	{
		docgen.POST("/docs", h.CreateDoc)
		docgen.GET("/docs", h.ListDocs)
		docgen.GET("/docs/:id", h.GetDoc)
		docgen.PUT("/docs/:id", h.UpdateDoc)
		docgen.DELETE("/docs/:id", h.DeleteDoc)

		docgen.POST("/docs/:id/events", h.AddEventType)
		docgen.GET("/docs/:id/events", h.ListEventTypes)
		docgen.PUT("/events/:eventId", h.UpdateEventType)
		docgen.DELETE("/events/:eventId", h.DeleteEventType)

		docgen.POST("/generate-code", h.GenerateCode)

		docgen.GET("/docs/:id/catalog", h.GetEventCatalog)

		docgen.POST("/widgets", h.CreateWidget)
		docgen.GET("/widgets/:id", h.GetWidget)

		docgen.GET("/docs/:id/analytics", h.GetDocAnalytics)
	}
}

// @Summary Create webhook doc
// @Tags DocGen
// @Accept json
// @Produce json
// @Success 201 {object} WebhookDoc
// @Router /docgen/docs [post]
func (h *Handler) CreateDoc(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req CreateDocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	doc, err := h.service.CreateDoc(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, doc)
}

// @Summary List webhook docs
// @Tags DocGen
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /docgen/docs [get]
func (h *Handler) ListDocs(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		pkgerrors.AbortWithUnauthorized(c)
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

	docs, total, err := h.service.ListDocs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"docs":   docs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// @Summary Get webhook doc
// @Tags DocGen
// @Produce json
// @Param id path string true "Doc ID"
// @Success 200 {object} WebhookDoc
// @Router /docgen/docs/{id} [get]
func (h *Handler) GetDoc(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	// Record view
	h.service.RecordDocView(c.Request.Context(), id)

	doc, err := h.service.GetDoc(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "doc")
		return
	}

	c.JSON(http.StatusOK, doc)
}

// @Summary Update webhook doc
// @Tags DocGen
// @Accept json
// @Produce json
// @Param id path string true "Doc ID"
// @Success 200 {object} WebhookDoc
// @Router /docgen/docs/{id} [put]
func (h *Handler) UpdateDoc(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	var req CreateDocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	doc, err := h.service.UpdateDoc(c.Request.Context(), id, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, doc)
}

// @Summary Delete webhook doc
// @Tags DocGen
// @Param id path string true "Doc ID"
// @Success 204 "No content"
// @Router /docgen/docs/{id} [delete]
func (h *Handler) DeleteDoc(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	if err := h.service.DeleteDoc(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Add event type to doc
// @Tags DocGen
// @Accept json
// @Produce json
// @Param id path string true "Doc ID"
// @Success 201 {object} EventTypeDoc
// @Router /docgen/docs/{id}/events [post]
func (h *Handler) AddEventType(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	var req AddEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	et, err := h.service.AddEventType(c.Request.Context(), docID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, et)
}

// @Summary List event types for a doc
// @Tags DocGen
// @Produce json
// @Param id path string true "Doc ID"
// @Success 200 {object} map[string]interface{}
// @Router /docgen/docs/{id}/events [get]
func (h *Handler) ListEventTypes(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	eventTypes, err := h.service.ListEventTypes(c.Request.Context(), docID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"event_types": eventTypes})
}

// @Summary Update event type
// @Tags DocGen
// @Accept json
// @Produce json
// @Param eventId path string true "Event type ID"
// @Success 200 {object} EventTypeDoc
// @Router /docgen/events/{eventId} [put]
func (h *Handler) UpdateEventType(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid event type ID")
		return
	}

	var req AddEventTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	et, err := h.service.UpdateEventType(c.Request.Context(), eventID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, et)
}

// @Summary Delete event type
// @Tags DocGen
// @Param eventId path string true "Event type ID"
// @Success 204 "No content"
// @Router /docgen/events/{eventId} [delete]
func (h *Handler) DeleteEventType(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid event type ID")
		return
	}

	if err := h.service.DeleteEventType(c.Request.Context(), eventID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Generate code sample
// @Tags DocGen
// @Accept json
// @Produce json
// @Success 200 {object} CodeSample
// @Router /docgen/generate-code [post]
func (h *Handler) GenerateCode(c *gin.Context) {
	var req GenerateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	sample, err := h.service.GenerateCodeSample(c.Request.Context(), &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, sample)
}

// @Summary Get event catalog
// @Tags DocGen
// @Produce json
// @Param id path string true "Doc ID"
// @Success 200 {object} EventCatalog
// @Router /docgen/docs/{id}/catalog [get]
func (h *Handler) GetEventCatalog(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	catalog, err := h.service.GetEventCatalog(c.Request.Context(), docID, tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, catalog)
}

// @Summary Create doc widget
// @Tags DocGen
// @Accept json
// @Produce json
// @Success 201 {object} DocWidget
// @Router /docgen/widgets [post]
func (h *Handler) CreateWidget(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		pkgerrors.AbortWithUnauthorized(c)
		return
	}

	var req CreateWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	widget, err := h.service.CreateWidget(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, widget)
}

// @Summary Get doc widget
// @Tags DocGen
// @Produce json
// @Param id path string true "Widget ID"
// @Success 200 {object} DocWidget
// @Router /docgen/widgets/{id} [get]
func (h *Handler) GetWidget(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid widget ID")
		return
	}

	widget, err := h.service.GetWidget(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "widget")
		return
	}

	c.JSON(http.StatusOK, widget)
}

// @Summary Get doc analytics
// @Tags DocGen
// @Produce json
// @Param id path string true "Doc ID"
// @Success 200 {object} DocAnalytics
// @Router /docgen/docs/{id}/analytics [get]
func (h *Handler) GetDocAnalytics(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid doc ID")
		return
	}

	analytics, err := h.service.GetDocAnalytics(c.Request.Context(), docID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, analytics)
}

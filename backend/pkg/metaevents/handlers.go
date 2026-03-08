package metaevents

import (
	"fmt"
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for meta-events
type Handler struct {
	service *Service
}

// NewHandler creates a new meta-events handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers meta-events routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	subscriptions := router.Group("/meta-events/subscriptions")
	{
		subscriptions.POST("", h.CreateSubscription)
		subscriptions.GET("", h.ListSubscriptions)
		subscriptions.GET("/:id", h.GetSubscription)
		subscriptions.PUT("/:id", h.UpdateSubscription)
		subscriptions.DELETE("/:id", h.DeleteSubscription)
		subscriptions.POST("/:id/rotate-secret", h.RotateSecret)
		subscriptions.GET("/:id/secret", h.GetSecret)
		subscriptions.POST("/:id/test", h.TestSubscription)
		subscriptions.GET("/:id/deliveries", h.ListDeliveries)
	}

	events := router.Group("/meta-events")
	{
		events.GET("/events", h.ListEvents)
		events.GET("/events/:id", h.GetEvent)
		events.GET("/event-types", h.GetEventTypes)
	}
}

// CreateSubscription godoc
//
//	@Summary		Create meta-event subscription
//	@Description	Create a new subscription to receive meta-events
//	@Tags			meta-events
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateSubscriptionRequest	true	"Subscription request"
//	@Success		201		{object}	Subscription
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions [post]
func (h *Handler) CreateSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	sub, err := h.service.CreateSubscription(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// ListSubscriptions godoc
//
//	@Summary		List meta-event subscriptions
//	@Description	Get a list of meta-event subscriptions
//	@Tags			meta-events
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions [get]
func (h *Handler) ListSubscriptions(c *gin.Context) {
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

	subs, total, err := h.service.ListSubscriptions(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subs,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// GetSubscription godoc
//
//	@Summary		Get subscription details
//	@Description	Get details of a specific meta-event subscription
//	@Tags			meta-events
//	@Produce		json
//	@Param			id	path		string	true	"Subscription ID"
//	@Success		200	{object}	Subscription
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id} [get]
func (h *Handler) GetSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	sub, err := h.service.GetSubscription(c.Request.Context(), tenantID, subID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// UpdateSubscription godoc
//
//	@Summary		Update subscription
//	@Description	Update a meta-event subscription
//	@Tags			meta-events
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Subscription ID"
//	@Param			request	body		UpdateSubscriptionRequest	true	"Update request"
//	@Success		200		{object}	Subscription
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id} [put]
func (h *Handler) UpdateSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	sub, err := h.service.UpdateSubscription(c.Request.Context(), tenantID, subID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// DeleteSubscription godoc
//
//	@Summary		Delete subscription
//	@Description	Delete a meta-event subscription
//	@Tags			meta-events
//	@Produce		json
//	@Param			id	path	string	true	"Subscription ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id} [delete]
func (h *Handler) DeleteSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	if err := h.service.DeleteSubscription(c.Request.Context(), tenantID, subID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RotateSecret godoc
//
//	@Summary		Rotate subscription secret
//	@Description	Generate a new signing secret for the subscription
//	@Tags			meta-events
//	@Produce		json
//	@Param			id	path		string	true	"Subscription ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id}/rotate-secret [post]
func (h *Handler) RotateSecret(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	_, newSecret, err := h.service.RotateSecret(c.Request.Context(), tenantID, subID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":  newSecret,
		"message": "Secret rotated successfully. Store this secret securely - it won't be shown again.",
	})
}

// GetSecret godoc
//
//	@Summary		Get subscription secret
//	@Description	Get the signing secret for a subscription (sensitive operation)
//	@Tags			meta-events
//	@Produce		json
//	@Param			id	path		string	true	"Subscription ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id}/secret [get]
func (h *Handler) GetSecret(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	secret, err := h.service.GetSecret(c.Request.Context(), tenantID, subID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

// TestSubscription godoc
//
//	@Summary		Test subscription
//	@Description	Send a test event to the subscription endpoint
//	@Tags			meta-events
//	@Produce		json
//	@Param			id	path		string	true	"Subscription ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id}/test [post]
func (h *Handler) TestSubscription(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	if err := h.service.TestSubscription(c.Request.Context(), tenantID, subID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test event sent successfully"})
}

// ListDeliveries godoc
//
//	@Summary		List deliveries
//	@Description	Get delivery history for a subscription
//	@Tags			meta-events
//	@Produce		json
//	@Param			id		path		string	true	"Subscription ID"
//	@Param			limit	query		int		false	"Limit"		default(20)
//	@Param			offset	query		int		false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/subscriptions/{id}/deliveries [get]
func (h *Handler) ListDeliveries(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subID := c.Param("id")
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	deliveries, total, err := h.service.ListDeliveries(c.Request.Context(), tenantID, subID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deliveries": deliveries,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// ListEvents godoc
//
//	@Summary		List meta-events
//	@Description	Get a list of meta-events
//	@Tags			meta-events
//	@Produce		json
//	@Param			type	query		string	false	"Filter by event type"
//	@Param			limit	query		int		false	"Limit"		default(20)
//	@Param			offset	query		int		false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
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

	var eventType *EventType
	if et := c.Query("type"); et != "" {
		t := EventType(et)
		eventType = &t
	}

	events, total, err := h.service.ListEvents(c.Request.Context(), tenantID, eventType, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetEvent godoc
//
//	@Summary		Get event details
//	@Description	Get details of a specific meta-event
//	@Tags			meta-events
//	@Produce		json
//	@Param			id	path		string	true	"Event ID"
//	@Success		200	{object}	MetaEvent
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/events/{id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	eventID := c.Param("id")
	event, err := h.service.GetEvent(c.Request.Context(), tenantID, eventID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetEventTypes godoc
//
//	@Summary		Get available event types
//	@Description	Get all supported meta-event types
//	@Tags			meta-events
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/meta-events/event-types [get]
func (h *Handler) GetEventTypes(c *gin.Context) {
	types := h.service.GetAvailableEventTypes()
	c.JSON(http.StatusOK, gin.H{"event_types": types})
}

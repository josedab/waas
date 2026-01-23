package fanout

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles fan-out HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new fanout handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers fan-out routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	fanout := r.Group("/fanout")
	{
		fanout.POST("/topics", h.CreateTopic)
		fanout.GET("/topics", h.ListTopics)
		fanout.GET("/topics/:id", h.GetTopic)
		fanout.PUT("/topics/:id", h.UpdateTopic)
		fanout.DELETE("/topics/:id", h.DeleteTopic)
		fanout.POST("/topics/:id/subscribe", h.Subscribe)
		fanout.DELETE("/topics/:id/subscriptions/:subId", h.Unsubscribe)
		fanout.GET("/topics/:id/subscriptions", h.ListSubscriptions)
		fanout.POST("/topics/:id/publish", h.PublishEvent)
		fanout.GET("/topics/:id/events", h.ListEvents)
	}
}

func (h *Handler) CreateTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	var req CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	topic, err := h.service.CreateTopic(c.Request.Context(), tid, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, topic)
}

func (h *Handler) ListTopics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	limit, offset := parsePagination(c)

	topics, total, err := h.service.ListTopics(c.Request.Context(), tid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"topics": topics,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	topic, err := h.service.GetTopic(c.Request.Context(), tid, topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, topic)
}

func (h *Handler) UpdateTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	var req UpdateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	topic, err := h.service.UpdateTopic(c.Request.Context(), tid, topicID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, topic)
}

func (h *Handler) DeleteTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	if err := h.service.DeleteTopic(c.Request.Context(), tid, topicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DELETE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "topic deleted"})
}

func (h *Handler) Subscribe(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	endpointID, err := uuid.Parse(req.EndpointID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ENDPOINT", "message": "invalid endpoint ID"}})
		return
	}

	sub, err := h.service.Subscribe(c.Request.Context(), tid, topicID, endpointID, req.FilterExpression)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "SUBSCRIBE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *Handler) Unsubscribe(c *gin.Context) {
	subID, err := uuid.Parse(c.Param("subId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid subscription ID"}})
		return
	}

	if err := h.service.Unsubscribe(c.Request.Context(), subID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UNSUBSCRIBE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unsubscribed"})
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	limit, offset := parsePagination(c)

	subs, total, err := h.service.GetTopicSubscribers(c.Request.Context(), topicID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subs,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

func (h *Handler) PublishEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "invalid tenant ID"}})
		return
	}

	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	// Look up the topic to get its name
	topic, err := h.service.GetTopic(c.Request.Context(), tid, topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "TOPIC_NOT_FOUND", "message": err.Error()}})
		return
	}

	result, err := h.service.Publish(c.Request.Context(), tid, topic.Name, req.EventType, req.Payload, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "PUBLISH_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListEvents(c *gin.Context) {
	topicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "invalid topic ID"}})
		return
	}

	limit, offset := parsePagination(c)

	events, total, err := h.service.GetTopicEvents(c.Request.Context(), topicID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func parsePagination(c *gin.Context) (int, int) {
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}
	return limit, offset
}

package streaming

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GAHandler provides HTTP handlers for GA streaming features
type GAHandler struct {
	groups     *ConsumerGroupManager
	dlq        *DeadLetterRouter
	topicAdmin *TopicAdmin
	monitoring *MonitoringService
}

// NewGAHandler creates a new GA handler
func NewGAHandler(groups *ConsumerGroupManager, dlq *DeadLetterRouter, topicAdmin *TopicAdmin, monitoring *MonitoringService) *GAHandler {
	return &GAHandler{
		groups:     groups,
		dlq:        dlq,
		topicAdmin: topicAdmin,
		monitoring: monitoring,
	}
}

// RegisterRoutes registers GA streaming routes
func (h *GAHandler) RegisterRoutes(r *gin.RouterGroup) {
	streaming := r.Group("/streaming")
	{
		// Consumer Groups
		groups := streaming.Group("/consumer-groups")
		{
			groups.POST("", h.CreateConsumerGroup)
			groups.GET("", h.ListConsumerGroups)
			groups.GET("/:groupId", h.GetConsumerGroup)
			groups.DELETE("/:groupId", h.DeleteConsumerGroup)
			groups.POST("/:groupId/join", h.JoinConsumerGroup)
			groups.POST("/:groupId/leave", h.LeaveConsumerGroup)
			groups.POST("/:groupId/offsets", h.CommitOffset)
		}

		// Dead-Letter Queues
		dlq := streaming.Group("/dead-letter")
		{
			dlq.POST("/policies", h.CreateDeadLetterPolicy)
			dlq.GET("/policies/:bridgeId", h.GetDeadLetterPolicy)
			dlq.GET("/events/:bridgeId", h.ListDeadLetterEvents)
			dlq.POST("/events/:eventId/reprocess", h.ReprocessDeadLetterEvent)
			dlq.GET("/stats/:bridgeId", h.GetDeadLetterStats)
		}

		// Topic Admin
		topics := streaming.Group("/topics")
		{
			topics.POST("", h.CreateTopic)
			topics.GET("", h.ListTopics)
			topics.GET("/:topicId", h.GetTopic)
			topics.PUT("/:topicId", h.UpdateTopic)
			topics.DELETE("/:topicId", h.DeleteTopic)
		}

		// Monitoring
		monitoring := streaming.Group("/monitoring")
		{
			monitoring.GET("/dashboard", h.GetMonitoringDashboard)
			monitoring.GET("/alerts", h.GetAlerts)
			monitoring.POST("/alerts/:alertId/resolve", h.ResolveAlert)
			monitoring.POST("/health-check", h.RunHealthCheck)
		}
	}
}

// CreateConsumerGroup creates a consumer group
// @Summary Create consumer group
// @Tags streaming
// @Accept json
// @Produce json
// @Param request body CreateConsumerGroupRequest true "Consumer group configuration"
// @Success 201 {object} ConsumerGroup
// @Router /streaming/consumer-groups [post]
func (h *GAHandler) CreateConsumerGroup(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateConsumerGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.groups.CreateGroup(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// ListConsumerGroups lists consumer groups
// @Summary List consumer groups
// @Tags streaming
// @Produce json
// @Param bridge_id query string false "Filter by bridge ID"
// @Success 200 {array} ConsumerGroup
// @Router /streaming/consumer-groups [get]
func (h *GAHandler) ListConsumerGroups(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Query("bridge_id")
	groups, err := h.groups.ListGroups(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// GetConsumerGroup retrieves a consumer group
// @Summary Get consumer group
// @Tags streaming
// @Produce json
// @Param groupId path string true "Group ID"
// @Success 200 {object} ConsumerGroup
// @Router /streaming/consumer-groups/{groupId} [get]
func (h *GAHandler) GetConsumerGroup(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID := c.Param("groupId")
	group, err := h.groups.GetGroup(c.Request.Context(), tenantID, groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "consumer group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteConsumerGroup deletes a consumer group
// @Summary Delete consumer group
// @Tags streaming
// @Param groupId path string true "Group ID"
// @Success 204 "No content"
// @Router /streaming/consumer-groups/{groupId} [delete]
func (h *GAHandler) DeleteConsumerGroup(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID := c.Param("groupId")
	if err := h.groups.DeleteGroup(c.Request.Context(), tenantID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// JoinConsumerGroup adds a member to a consumer group
// @Summary Join consumer group
// @Tags streaming
// @Accept json
// @Produce json
// @Param groupId path string true "Group ID"
// @Success 200 {object} ConsumerMember
// @Router /streaming/consumer-groups/{groupId}/join [post]
func (h *GAHandler) JoinConsumerGroup(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID := c.Param("groupId")
	var req struct {
		ClientID string `json:"client_id" binding:"required"`
		Host     string `json:"host" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.groups.JoinGroup(c.Request.Context(), tenantID, groupID, req.ClientID, req.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}

// LeaveConsumerGroup removes a member from a consumer group
// @Summary Leave consumer group
// @Tags streaming
// @Accept json
// @Param groupId path string true "Group ID"
// @Success 204 "No content"
// @Router /streaming/consumer-groups/{groupId}/leave [post]
func (h *GAHandler) LeaveConsumerGroup(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID := c.Param("groupId")
	var req struct {
		MemberID string `json:"member_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.groups.LeaveGroup(c.Request.Context(), tenantID, groupID, req.MemberID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CommitOffset commits an offset for a partition
// @Summary Commit offset
// @Tags streaming
// @Accept json
// @Param groupId path string true "Group ID"
// @Success 204 "No content"
// @Router /streaming/consumer-groups/{groupId}/offsets [post]
func (h *GAHandler) CommitOffset(c *gin.Context) {
	groupID := c.Param("groupId")
	var req struct {
		Partition int   `json:"partition" binding:"min=0"`
		Offset    int64 `json:"offset" binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.groups.CommitOffset(c.Request.Context(), groupID, req.Partition, req.Offset); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateDeadLetterPolicy creates a dead-letter policy
// @Summary Create dead-letter policy
// @Tags streaming
// @Accept json
// @Produce json
// @Success 201 {object} DeadLetterPolicy
// @Router /streaming/dead-letter/policies [post]
func (h *GAHandler) CreateDeadLetterPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateDeadLetterPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.dlq.CreatePolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// GetDeadLetterPolicy retrieves a dead-letter policy
// @Summary Get dead-letter policy
// @Tags streaming
// @Produce json
// @Param bridgeId path string true "Bridge ID"
// @Success 200 {object} DeadLetterPolicy
// @Router /streaming/dead-letter/policies/{bridgeId} [get]
func (h *GAHandler) GetDeadLetterPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("bridgeId")
	policy, err := h.dlq.GetPolicy(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// ListDeadLetterEvents lists dead-letter events
// @Summary List dead-letter events
// @Tags streaming
// @Produce json
// @Param bridgeId path string true "Bridge ID"
// @Success 200 {object} map[string]interface{}
// @Router /streaming/dead-letter/events/{bridgeId} [get]
func (h *GAHandler) ListDeadLetterEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("bridgeId")
	filters := &DLQFilters{}
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		filters.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil && pageSize > 0 {
		filters.PageSize = pageSize
	}

	events, total, err := h.dlq.ListDeadLetterEvents(c.Request.Context(), tenantID, bridgeID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
	})
}

// ReprocessDeadLetterEvent reprocesses a dead-letter event
// @Summary Reprocess dead-letter event
// @Tags streaming
// @Param eventId path string true "Event ID"
// @Success 202 {object} map[string]interface{}
// @Router /streaming/dead-letter/events/{eventId}/reprocess [post]
func (h *GAHandler) ReprocessDeadLetterEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	eventID := c.Param("eventId")
	if err := h.dlq.ReprocessEvent(c.Request.Context(), tenantID, eventID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "reprocessing"})
}

// GetDeadLetterStats returns dead-letter statistics
// @Summary Get dead-letter stats
// @Tags streaming
// @Produce json
// @Param bridgeId path string true "Bridge ID"
// @Success 200 {object} map[string]interface{}
// @Router /streaming/dead-letter/stats/{bridgeId} [get]
func (h *GAHandler) GetDeadLetterStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("bridgeId")
	stats, err := h.dlq.GetDeadLetterStats(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateTopic creates a managed topic
// @Summary Create managed topic
// @Tags streaming
// @Accept json
// @Produce json
// @Success 201 {object} ManagedTopic
// @Router /streaming/topics [post]
func (h *GAHandler) CreateTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.topicAdmin.CreateTopic(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, topic)
}

// ListTopics lists managed topics
// @Summary List managed topics
// @Tags streaming
// @Produce json
// @Param bridge_id query string false "Filter by bridge ID"
// @Success 200 {array} ManagedTopic
// @Router /streaming/topics [get]
func (h *GAHandler) ListTopics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Query("bridge_id")
	topics, err := h.topicAdmin.ListTopics(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topics)
}

// GetTopic retrieves a managed topic
// @Summary Get managed topic
// @Tags streaming
// @Produce json
// @Param topicId path string true "Topic ID"
// @Success 200 {object} ManagedTopic
// @Router /streaming/topics/{topicId} [get]
func (h *GAHandler) GetTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	topicID := c.Param("topicId")
	topic, err := h.topicAdmin.GetTopic(c.Request.Context(), tenantID, topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
		return
	}

	c.JSON(http.StatusOK, topic)
}

// UpdateTopic updates a managed topic
// @Summary Update managed topic
// @Tags streaming
// @Accept json
// @Produce json
// @Param topicId path string true "Topic ID"
// @Success 200 {object} ManagedTopic
// @Router /streaming/topics/{topicId} [put]
func (h *GAHandler) UpdateTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	topicID := c.Param("topicId")
	var req UpdateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.topicAdmin.UpdateTopic(c.Request.Context(), tenantID, topicID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topic)
}

// DeleteTopic deletes a managed topic
// @Summary Delete managed topic
// @Tags streaming
// @Param topicId path string true "Topic ID"
// @Success 204 "No content"
// @Router /streaming/topics/{topicId} [delete]
func (h *GAHandler) DeleteTopic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	topicID := c.Param("topicId")
	if err := h.topicAdmin.DeleteTopic(c.Request.Context(), tenantID, topicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMonitoringDashboard retrieves the monitoring dashboard
// @Summary Get monitoring dashboard
// @Tags streaming
// @Produce json
// @Success 200 {object} MonitoringDashboard
// @Router /streaming/monitoring/dashboard [get]
func (h *GAHandler) GetMonitoringDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.monitoring.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetAlerts retrieves monitoring alerts
// @Summary Get monitoring alerts
// @Tags streaming
// @Produce json
// @Param active_only query bool false "Show only active alerts"
// @Success 200 {array} StreamingAlert
// @Router /streaming/monitoring/alerts [get]
func (h *GAHandler) GetAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activeOnly := c.Query("active_only") == "true"
	alerts, err := h.monitoring.GetAlerts(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// ResolveAlert resolves a monitoring alert
// @Summary Resolve alert
// @Tags streaming
// @Param alertId path string true "Alert ID"
// @Success 204 "No content"
// @Router /streaming/monitoring/alerts/{alertId}/resolve [post]
func (h *GAHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("alertId")
	if err := h.monitoring.ResolveAlert(c.Request.Context(), alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RunHealthCheck runs a health check on all bridges
// @Summary Run health check
// @Tags streaming
// @Produce json
// @Success 200 {array} StreamingAlert
// @Router /streaming/monitoring/health-check [post]
func (h *GAHandler) RunHealthCheck(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	alerts, err := h.monitoring.CheckBridgeHealth(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts_generated": len(alerts),
		"alerts":           alerts,
	})
}

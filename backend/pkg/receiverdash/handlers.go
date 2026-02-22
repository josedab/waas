package receiverdash

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the receiver dashboard.
type Handler struct {
	service *Service
}

// NewHandler creates a new receiver dashboard handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all receiver dashboard routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	// Admin routes — token management (requires tenant auth)
	admin := router.Group("/receiver-dashboard/admin")
	{
		admin.POST("/tokens", h.CreateToken)
		admin.GET("/tokens", h.ListTokens)
		admin.GET("/tokens/:id", h.GetToken)
		admin.DELETE("/tokens/:id", h.RevokeToken)
	}

	// Public receiver routes — authenticated via X-Receiver-Token header
	receiver := router.Group("/receiver-dashboard")
	receiver.Use(h.receiverTokenAuth())
	{
		receiver.GET("/deliveries", h.GetDeliveryHistory)
		receiver.GET("/deliveries/:id/payload", h.InspectPayload)
		receiver.GET("/retries", h.GetRetryStatus)
		receiver.GET("/health", h.GetHealthSummary)
		receiver.GET("/health/:endpoint_id", h.GetEndpointHealth)
	}
}

// receiverTokenAuth validates the X-Receiver-Token header.
func (h *Handler) receiverTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := c.GetHeader("X-Receiver-Token")
		if tokenValue == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Receiver-Token header required"})
			c.Abort()
			return
		}
		token, err := h.service.ValidateToken(tokenValue)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.Set("receiver_token", token)
		c.Next()
	}
}

func getReceiverToken(c *gin.Context) *ReceiverToken {
	val, exists := c.Get("receiver_token")
	if !exists {
		return nil
	}
	return val.(*ReceiverToken)
}

// CreateToken creates a new receiver-scoped token.
// @Summary Create receiver token
// @Tags receiver-dashboard
// @Accept json
// @Produce json
// @Param request body CreateTokenRequest true "Token request"
// @Success 201 {object} ReceiverToken
// @Router /receiver-dashboard/admin/tokens [post]
func (h *Handler) CreateToken(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.service.CreateToken(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, token)
}

// ListTokens lists all tokens for the tenant.
// @Summary List receiver tokens
// @Tags receiver-dashboard
// @Produce json
// @Success 200 {array} ReceiverToken
// @Router /receiver-dashboard/admin/tokens [get]
func (h *Handler) ListTokens(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	tokens, err := h.service.ListTokens(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

// GetToken retrieves a specific token.
// @Summary Get receiver token
// @Tags receiver-dashboard
// @Produce json
// @Param id path string true "Token ID"
// @Success 200 {object} ReceiverToken
// @Router /receiver-dashboard/admin/tokens/{id} [get]
func (h *Handler) GetToken(c *gin.Context) {
	token, err := h.service.GetToken(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, token)
}

// RevokeToken revokes a token.
// @Summary Revoke receiver token
// @Tags receiver-dashboard
// @Param id path string true "Token ID"
// @Success 204
// @Router /receiver-dashboard/admin/tokens/{id} [delete]
func (h *Handler) RevokeToken(c *gin.Context) {
	if err := h.service.RevokeToken(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetDeliveryHistory returns delivery history for authorized endpoints.
// @Summary Get delivery history
// @Tags receiver-dashboard
// @Produce json
// @Param endpoint_id query string false "Filter by endpoint"
// @Param event_type query string false "Filter by event type"
// @Param status query string false "Filter: success, failed, all"
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /receiver-dashboard/deliveries [get]
func (h *Handler) GetDeliveryHistory(c *gin.Context) {
	token := getReceiverToken(c)
	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req DeliveryHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deliveries, total, err := h.service.GetDeliveryHistory(token, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deliveries": deliveries, "total": total})
}

// InspectPayload returns the payload for a specific delivery.
// @Summary Inspect delivery payload
// @Tags receiver-dashboard
// @Produce json
// @Param id path string true "Delivery ID"
// @Success 200 {object} PayloadInspection
// @Router /receiver-dashboard/deliveries/{id}/payload [get]
func (h *Handler) InspectPayload(c *gin.Context) {
	token := getReceiverToken(c)
	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	payload, err := h.service.InspectPayload(token, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, payload)
}

// GetRetryStatus returns retry status for authorized endpoints.
// @Summary Get retry status
// @Tags receiver-dashboard
// @Produce json
// @Param active_only query bool false "Show only active retries"
// @Success 200 {array} RetryStatus
// @Router /receiver-dashboard/retries [get]
func (h *Handler) GetRetryStatus(c *gin.Context) {
	token := getReceiverToken(c)
	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activeOnly := c.Query("active_only") == "true"
	retries, err := h.service.GetRetryStatus(token, activeOnly)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, retries)
}

// GetHealthSummary returns aggregate health across all authorized endpoints.
// @Summary Get health summary
// @Tags receiver-dashboard
// @Produce json
// @Param period query string false "Time period: 1h, 24h, 7d"
// @Success 200 {object} HealthSummary
// @Router /receiver-dashboard/health [get]
func (h *Handler) GetHealthSummary(c *gin.Context) {
	token := getReceiverToken(c)
	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.DefaultQuery("period", "24h")
	summary, err := h.service.GetHealthSummary(token, period)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetEndpointHealth returns health for a specific endpoint.
// @Summary Get endpoint health
// @Tags receiver-dashboard
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Param period query string false "Time period: 1h, 24h, 7d"
// @Success 200 {object} EndpointHealth
// @Router /receiver-dashboard/health/{endpoint_id} [get]
func (h *Handler) GetEndpointHealth(c *gin.Context) {
	token := getReceiverToken(c)
	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	period := c.DefaultQuery("period", "24h")
	health, err := h.service.GetEndpointHealth(token, c.Param("endpoint_id"), period)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, health)
}

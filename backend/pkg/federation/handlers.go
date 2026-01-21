package federation

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles federation HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new federation handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers federation routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	fed := r.Group("/federation")
	{
		// Members
		fed.POST("/members", h.RegisterMember)
		fed.GET("/members", h.ListMembers)
		fed.GET("/members/:id", h.GetMember)
		fed.POST("/members/:id/activate", h.ActivateMember)
		fed.POST("/members/:id/suspend", h.SuspendMember)
		fed.GET("/members/:id/health", h.HealthCheck)

		// Trust
		fed.POST("/trust/request", h.RequestTrust)
		fed.GET("/trust/requests", h.ListTrustRequests)
		fed.POST("/trust/requests/:id/approve", h.ApproveTrust)
		fed.POST("/trust/requests/:id/reject", h.RejectTrust)
		fed.GET("/trust/relationships/:member_id", h.ListTrustRelationships)

		// Catalogs
		fed.POST("/catalogs", h.CreateCatalog)
		fed.GET("/catalogs", h.ListCatalogs)
		fed.GET("/catalogs/:id", h.GetCatalog)
		fed.GET("/catalogs/discover", h.DiscoverCatalogs)

		// Subscriptions
		fed.POST("/subscriptions", h.Subscribe)
		fed.GET("/subscriptions", h.ListSubscriptions)
		fed.GET("/subscriptions/:id", h.GetSubscription)
		fed.POST("/subscriptions/:id/pause", h.PauseSubscription)
		fed.POST("/subscriptions/:id/resume", h.ResumeSubscription)

		// Events
		fed.POST("/events", h.PublishEvent)

		// Policy
		fed.GET("/policy", h.GetPolicy)
		fed.PUT("/policy", h.UpdatePolicy)

		// Metrics
		fed.GET("/metrics", h.GetMetrics)

		// Keys
		fed.POST("/keys/generate", h.GenerateKeyPair)
	}
}

// RegisterMember godoc
// @Summary Register member
// @Description Register a new federation member
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body RegisterMemberRequest true "Member configuration"
// @Success 201 {object} FederationMember
// @Router /federation/members [post]
func (h *Handler) RegisterMember(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req RegisterMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.service.RegisterMember(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, member)
}

// ListMembers godoc
// @Summary List members
// @Description List federation members
// @Tags Federation
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {array} FederationMember
// @Router /federation/members [get]
func (h *Handler) ListMembers(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var status *MemberStatus
	if s := c.Query("status"); s != "" {
		st := MemberStatus(s)
		status = &st
	}

	members, err := h.service.ListMembers(c.Request.Context(), tenantID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// GetMember godoc
// @Summary Get member
// @Description Get federation member by ID
// @Tags Federation
// @Produce json
// @Param id path string true "Member ID"
// @Success 200 {object} FederationMember
// @Router /federation/members/{id} [get]
func (h *Handler) GetMember(c *gin.Context) {
	memberID := c.Param("id")

	member, err := h.service.GetMember(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}

// ActivateMember godoc
// @Summary Activate member
// @Description Activate a federation member
// @Tags Federation
// @Param id path string true "Member ID"
// @Success 200 {object} FederationMember
// @Router /federation/members/{id}/activate [post]
func (h *Handler) ActivateMember(c *gin.Context) {
	memberID := c.Param("id")

	member, err := h.service.ActivateMember(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}

// SuspendMember godoc
// @Summary Suspend member
// @Description Suspend a federation member
// @Tags Federation
// @Param id path string true "Member ID"
// @Success 200 {object} FederationMember
// @Router /federation/members/{id}/suspend [post]
func (h *Handler) SuspendMember(c *gin.Context) {
	memberID := c.Param("id")

	member, err := h.service.SuspendMember(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}

// HealthCheck godoc
// @Summary Health check
// @Description Check health of a federation member
// @Tags Federation
// @Produce json
// @Param id path string true "Member ID"
// @Success 200 {object} HealthCheck
// @Router /federation/members/{id}/health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	memberID := c.Param("id")

	health, err := h.service.HealthCheck(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// RequestTrust godoc
// @Summary Request trust
// @Description Request trust with another member
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body CreateTrustRequest true "Trust request"
// @Success 201 {object} TrustRequest
// @Router /federation/trust/request [post]
func (h *Handler) RequestTrust(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateTrustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trustReq, err := h.service.RequestTrust(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, trustReq)
}

// ListTrustRequests godoc
// @Summary List trust requests
// @Description List trust requests
// @Tags Federation
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {array} TrustRequest
// @Router /federation/trust/requests [get]
func (h *Handler) ListTrustRequests(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var status *TrustReqStatus
	if s := c.Query("status"); s != "" {
		st := TrustReqStatus(s)
		status = &st
	}

	requests, err := h.service.GetTrustRequests(c.Request.Context(), tenantID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// ApproveTrust godoc
// @Summary Approve trust
// @Description Approve a trust request
// @Tags Federation
// @Accept json
// @Produce json
// @Param id path string true "Request ID"
// @Param request body map[string]string false "Response message"
// @Success 200 {object} TrustRequest
// @Router /federation/trust/requests/{id}/approve [post]
func (h *Handler) ApproveTrust(c *gin.Context) {
	reqID := c.Param("id")

	var body struct {
		Response string `json:"response"`
	}
	c.ShouldBindJSON(&body)

	trustReq, err := h.service.ApproveTrust(c.Request.Context(), reqID, body.Response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trustReq)
}

// RejectTrust godoc
// @Summary Reject trust
// @Description Reject a trust request
// @Tags Federation
// @Accept json
// @Param id path string true "Request ID"
// @Param request body map[string]string false "Response message"
// @Success 200 {object} map[string]interface{}
// @Router /federation/trust/requests/{id}/reject [post]
func (h *Handler) RejectTrust(c *gin.Context) {
	reqID := c.Param("id")

	var body struct {
		Response string `json:"response"`
	}
	c.ShouldBindJSON(&body)

	if err := h.service.RejectTrust(c.Request.Context(), reqID, body.Response); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

// ListTrustRelationships godoc
// @Summary List trust relationships
// @Description List trust relationships for a member
// @Tags Federation
// @Produce json
// @Param member_id path string true "Member ID"
// @Success 200 {array} TrustRelationship
// @Router /federation/trust/relationships/{member_id} [get]
func (h *Handler) ListTrustRelationships(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	memberID := c.Param("member_id")

	relationships, err := h.service.GetTrustRelationships(c.Request.Context(), tenantID, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"relationships": relationships})
}

// CreateCatalog godoc
// @Summary Create catalog
// @Description Create an event catalog
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body CreateCatalogRequest true "Catalog configuration"
// @Success 201 {object} EventCatalog
// @Router /federation/catalogs [post]
func (h *Handler) CreateCatalog(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	catalog, err := h.service.CreateCatalog(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, catalog)
}

// ListCatalogs godoc
// @Summary List catalogs
// @Description List event catalogs
// @Tags Federation
// @Produce json
// @Param public query bool false "Filter by public"
// @Success 200 {array} EventCatalog
// @Router /federation/catalogs [get]
func (h *Handler) ListCatalogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	public := c.Query("public") == "true"

	catalogs, err := h.service.ListCatalogs(c.Request.Context(), tenantID, public)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"catalogs": catalogs})
}

// GetCatalog godoc
// @Summary Get catalog
// @Description Get event catalog by ID
// @Tags Federation
// @Produce json
// @Param id path string true "Catalog ID"
// @Success 200 {object} EventCatalog
// @Router /federation/catalogs/{id} [get]
func (h *Handler) GetCatalog(c *gin.Context) {
	catalogID := c.Param("id")

	catalog, err := h.service.GetCatalog(c.Request.Context(), catalogID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, catalog)
}

// DiscoverCatalogs godoc
// @Summary Discover catalogs
// @Description Discover public event catalogs
// @Tags Federation
// @Produce json
// @Success 200 {array} EventCatalog
// @Router /federation/catalogs/discover [get]
func (h *Handler) DiscoverCatalogs(c *gin.Context) {
	catalogs, err := h.service.DiscoverCatalogs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"catalogs": catalogs})
}

// Subscribe godoc
// @Summary Subscribe
// @Description Subscribe to events from another member
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body CreateSubscriptionRequest true "Subscription configuration"
// @Success 201 {object} FederatedSubscription
// @Router /federation/subscriptions [post]
func (h *Handler) Subscribe(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.Subscribe(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// ListSubscriptions godoc
// @Summary List subscriptions
// @Description List federated subscriptions
// @Tags Federation
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {array} FederatedSubscription
// @Router /federation/subscriptions [get]
func (h *Handler) ListSubscriptions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var status *SubStatus
	if s := c.Query("status"); s != "" {
		st := SubStatus(s)
		status = &st
	}

	subs, err := h.service.ListSubscriptions(c.Request.Context(), tenantID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscriptions": subs})
}

// GetSubscription godoc
// @Summary Get subscription
// @Description Get subscription by ID
// @Tags Federation
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 200 {object} FederatedSubscription
// @Router /federation/subscriptions/{id} [get]
func (h *Handler) GetSubscription(c *gin.Context) {
	subID := c.Param("id")

	sub, err := h.service.GetSubscription(c.Request.Context(), subID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// PauseSubscription godoc
// @Summary Pause subscription
// @Description Pause a subscription
// @Tags Federation
// @Param id path string true "Subscription ID"
// @Success 200 {object} map[string]interface{}
// @Router /federation/subscriptions/{id}/pause [post]
func (h *Handler) PauseSubscription(c *gin.Context) {
	subID := c.Param("id")

	if err := h.service.PauseSubscription(c.Request.Context(), subID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "paused"})
}

// ResumeSubscription godoc
// @Summary Resume subscription
// @Description Resume a paused subscription
// @Tags Federation
// @Param id path string true "Subscription ID"
// @Success 200 {object} map[string]interface{}
// @Router /federation/subscriptions/{id}/resume [post]
func (h *Handler) ResumeSubscription(c *gin.Context) {
	subID := c.Param("id")

	if err := h.service.ResumeSubscription(c.Request.Context(), subID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "active"})
}

// PublishEvent godoc
// @Summary Publish event
// @Description Publish an event to federation
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body FederationEvent true "Event to publish"
// @Success 202 {object} map[string]interface{}
// @Router /federation/events [post]
func (h *Handler) PublishEvent(c *gin.Context) {
	var event FederationEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.PublishEvent(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}

// GetPolicy godoc
// @Summary Get policy
// @Description Get federation policy
// @Tags Federation
// @Produce json
// @Success 200 {object} FederationPolicy
// @Router /federation/policy [get]
func (h *Handler) GetPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	policy, err := h.service.GetPolicy(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy godoc
// @Summary Update policy
// @Description Update federation policy
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body UpdatePolicyRequest true "Policy updates"
// @Success 200 {object} FederationPolicy
// @Router /federation/policy [put]
func (h *Handler) UpdatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.UpdatePolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// GetMetrics godoc
// @Summary Get metrics
// @Description Get federation metrics
// @Tags Federation
// @Produce json
// @Success 200 {object} FederationMetrics
// @Router /federation/metrics [get]
func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	metrics, err := h.service.GetMetrics(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GenerateKeyPair godoc
// @Summary Generate key pair
// @Description Generate a new key pair for federation
// @Tags Federation
// @Produce json
// @Success 200 {object} KeyPair
// @Router /federation/keys/generate [post]
func (h *Handler) GenerateKeyPair(c *gin.Context) {
	keyPair, err := h.service.GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keyPair)
}

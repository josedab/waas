package federation

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MeshHandler provides HTTP handlers for the federation mesh
type MeshHandler struct {
	mesh *FederationMesh
}

// NewMeshHandler creates a new handler
func NewMeshHandler(mesh *FederationMesh) *MeshHandler {
	return &MeshHandler{mesh: mesh}
}

// RegisterMeshRoutes registers federation mesh routes
func (h *MeshHandler) RegisterMeshRoutes(r *gin.RouterGroup) {
	fed := r.Group("/federation")
	{
		fed.POST("/events/route", h.RouteEvent)
		fed.GET("/events", h.ListEvents)
		fed.POST("/events/:eventId/verify", h.VerifyAttestation)

		schemas := fed.Group("/schemas")
		{
			schemas.POST("", h.PublishSchema)
			schemas.GET("", h.ListSchemas)
		}

		gov := fed.Group("/governance")
		{
			gov.POST("/policy", h.SetGovernancePolicy)
			gov.GET("/policy", h.GetGovernancePolicy)
		}
	}
}

// RouteEvent routes an event across federation
// @Summary Route federated event
// @Tags federation
// @Accept json
// @Produce json
// @Success 201 {object} FederatedEvent
// @Router /federation/events/route [post]
func (h *MeshHandler) RouteEvent(c *gin.Context) {
	var req RouteEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.mesh.RouteEvent(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusInternalServerError
		if event != nil && event.Status == "rejected" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error(), "event": event})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// ListEvents lists federated events
// @Summary List federated events
// @Tags federation
// @Produce json
// @Param peer_id query string false "Peer ID"
// @Param limit query int false "Limit"
// @Success 200 {array} FederatedEvent
// @Router /federation/events [get]
func (h *MeshHandler) ListEvents(c *gin.Context) {
	peerID := c.Query("peer_id")
	if peerID == "" {
		peerID = h.mesh.selfPeerID
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	events, err := h.mesh.ListFederatedEvents(c.Request.Context(), peerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// VerifyAttestation verifies a federated event's attestation
// @Summary Verify event attestation
// @Tags federation
// @Param eventId path string true "Event ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /federation/events/{eventId}/verify [post]
func (h *MeshHandler) VerifyAttestation(c *gin.Context) {
	eventID := c.Param("eventId")
	event, err := h.mesh.repo.GetFederatedEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	verified, err := h.mesh.VerifyAttestation(c.Request.Context(), event)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "verified": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified})
}

// PublishSchema publishes an event schema
// @Summary Publish event schema
// @Tags federation
// @Accept json
// @Produce json
// @Success 201 {object} SharedEventSchema
// @Router /federation/schemas [post]
func (h *MeshHandler) PublishSchema(c *gin.Context) {
	var req PublishSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schema, err := h.mesh.PublishSchema(c.Request.Context(), h.mesh.selfPeerID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schema)
}

// ListSchemas lists event schemas
// @Summary List event schemas
// @Tags federation
// @Produce json
// @Param public_only query bool false "Show only public schemas"
// @Success 200 {array} SharedEventSchema
// @Router /federation/schemas [get]
func (h *MeshHandler) ListSchemas(c *gin.Context) {
	publicOnly := c.Query("public_only") == "true"
	schemas, err := h.mesh.ListSchemas(c.Request.Context(), publicOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schemas)
}

// SetGovernancePolicy sets governance rules
// @Summary Set governance policy
// @Tags federation
// @Accept json
// @Produce json
// @Success 201 {object} FederationGovernancePolicy
// @Router /federation/governance/policy [post]
func (h *MeshHandler) SetGovernancePolicy(c *gin.Context) {
	var policy FederationGovernancePolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy.PeerID = h.mesh.selfPeerID
	if err := h.mesh.SetGovernancePolicy(c.Request.Context(), &policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// GetGovernancePolicy retrieves governance policy
// @Summary Get governance policy
// @Tags federation
// @Produce json
// @Success 200 {object} FederationGovernancePolicy
// @Router /federation/governance/policy [get]
func (h *MeshHandler) GetGovernancePolicy(c *gin.Context) {
	policy, err := h.mesh.GetGovernancePolicy(c.Request.Context(), h.mesh.selfPeerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

package policyengine

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the OPA policy engine.
type Handler struct {
	service *Service
}

// NewHandler creates a new policy engine handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers policy engine routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/policy-engine")
	{
		g.POST("/policies", h.CreatePolicy)
		g.GET("/policies", h.ListPolicies)
		g.GET("/policies/:id", h.GetPolicy)
		g.PUT("/policies/:id", h.UpdatePolicy)
		g.DELETE("/policies/:id", h.DeletePolicy)

		g.GET("/policies/:id/versions", h.ListPolicyVersions)

		g.POST("/evaluate", h.Evaluate)
		g.POST("/validate", h.ValidateRego)

		g.GET("/evaluations", h.ListEvaluationLogs)
	}
}

func (h *Handler) CreatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.CreatePolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

func (h *Handler) GetPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policyID := c.Param("id")

	policy, err := h.service.GetPolicy(c.Request.Context(), tenantID, policyID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

func (h *Handler) ListPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	policies, err := h.service.ListPolicies(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (h *Handler) UpdatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policyID := c.Param("id")

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.UpdatePolicy(c.Request.Context(), tenantID, policyID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

func (h *Handler) DeletePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policyID := c.Param("id")

	if err := h.service.DeletePolicy(c.Request.Context(), tenantID, policyID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) Evaluate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Evaluate(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) ValidateRego(c *gin.Context) {
	var req ValidateRegoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.service.ValidateRegoSource(req.RegoSource)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListPolicyVersions(c *gin.Context) {
	policyID := c.Param("id")

	versions, err := h.service.ListPolicyVersions(c.Request.Context(), policyID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func (h *Handler) ListEvaluationLogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.service.ListEvaluationLogs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"evaluations": logs})
}

package routingpolicy

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for routing policies.
type Handler struct {
	service *Service
}

// NewHandler creates a new routing policy handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all routing policy routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/routing-policies")
	{
		group.POST("", h.CreatePolicy)
		group.GET("", h.ListPolicies)
		group.GET("/:id", h.GetPolicy)
		group.PUT("/:id", h.UpdatePolicy)
		group.DELETE("/:id", h.DeletePolicy)
		group.POST("/:id/toggle", h.TogglePolicy)
		group.GET("/:id/versions", h.GetVersions)
		group.GET("/:id/audit", h.GetAuditLog)
		group.POST("/evaluate", h.Evaluate)
		group.POST("/what-if", h.WhatIf)
	}
}

// CreatePolicy creates a new routing policy.
// @Summary Create routing policy
// @Tags routing-policies
// @Accept json
// @Produce json
// @Success 201 {object} Policy
// @Router /routing-policies [post]
func (h *Handler) CreatePolicy(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.CreatePolicy(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, policy)
}

// ListPolicies lists all policies for the tenant.
// @Summary List routing policies
// @Tags routing-policies
// @Produce json
// @Success 200 {array} Policy
// @Router /routing-policies [get]
func (h *Handler) ListPolicies(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	policies, err := h.service.ListPolicies(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, policies)
}

// GetPolicy retrieves a specific policy.
// @Summary Get routing policy
// @Tags routing-policies
// @Param id path string true "Policy ID"
// @Produce json
// @Success 200 {object} Policy
// @Router /routing-policies/{id} [get]
func (h *Handler) GetPolicy(c *gin.Context) {
	policy, err := h.service.GetPolicy(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates a policy.
// @Summary Update routing policy
// @Tags routing-policies
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {object} Policy
// @Router /routing-policies/{id} [put]
func (h *Handler) UpdatePolicy(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.UpdatePolicy(c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, policy)
}

// DeletePolicy deletes a policy.
// @Summary Delete routing policy
// @Tags routing-policies
// @Param id path string true "Policy ID"
// @Success 204
// @Router /routing-policies/{id} [delete]
func (h *Handler) DeletePolicy(c *gin.Context) {
	if err := h.service.DeletePolicy(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// TogglePolicy enables or disables a policy.
// @Summary Toggle routing policy
// @Tags routing-policies
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {object} Policy
// @Router /routing-policies/{id}/toggle [post]
func (h *Handler) TogglePolicy(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.TogglePolicy(c.Param("id"), req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, policy)
}

// GetVersions returns version history.
// @Summary Get policy version history
// @Tags routing-policies
// @Param id path string true "Policy ID"
// @Produce json
// @Success 200 {array} PolicyVersion
// @Router /routing-policies/{id}/versions [get]
func (h *Handler) GetVersions(c *gin.Context) {
	versions, err := h.service.GetVersions(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

// GetAuditLog returns the audit log.
// @Summary Get policy audit log
// @Tags routing-policies
// @Param id path string true "Policy ID"
// @Produce json
// @Success 200 {array} AuditEntry
// @Router /routing-policies/{id}/audit [get]
func (h *Handler) GetAuditLog(c *gin.Context) {
	entries, err := h.service.GetAuditLog(c.Param("id"), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

// Evaluate evaluates policies against a context.
// @Summary Evaluate routing policies
// @Tags routing-policies
// @Accept json
// @Produce json
// @Success 200 {array} EvaluationResult
// @Router /routing-policies/evaluate [post]
func (h *Handler) Evaluate(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var ctx EvaluationContext
	if err := c.ShouldBindJSON(&ctx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.service.Evaluate(tenantID, &ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

// WhatIf simulates policy evaluation.
// @Summary Simulate policy evaluation
// @Tags routing-policies
// @Accept json
// @Produce json
// @Success 200 {object} EvaluationResult
// @Router /routing-policies/what-if [post]
func (h *Handler) WhatIf(c *gin.Context) {
	var req WhatIfRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.WhatIf(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

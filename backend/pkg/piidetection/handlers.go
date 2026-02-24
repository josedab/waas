package piidetection

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for PII detection.
type Handler struct {
	service *Service
}

// NewHandler creates a new PII detection handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers PII detection routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/pii-detection")
	{
		// Policies
		g.POST("/policies", h.CreatePolicy)
		g.GET("/policies", h.ListPolicies)
		g.GET("/policies/:id", h.GetPolicy)
		g.PUT("/policies/:id", h.UpdatePolicy)
		g.DELETE("/policies/:id", h.DeletePolicy)

		// Scanning
		g.POST("/scan", h.ScanPayload)

		// Scan results
		g.GET("/scans", h.ListScanResults)
		g.GET("/scans/:id", h.GetScanResult)

		// Dashboard
		g.GET("/dashboard", h.GetDashboardStats)
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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

func (h *Handler) ScanPayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ScanPayload(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetScanResult(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	resultID := c.Param("id")

	result, err := h.service.GetScanResult(c.Request.Context(), tenantID, resultID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListScanResults(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	results, err := h.service.ListScanResults(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"scans": results})
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetDashboardStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

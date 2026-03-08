package compliancevault

import (
	"encoding/json"
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the compliance vault.
type Handler struct {
	service *Service
}

// NewHandler creates a new compliance vault handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers compliance vault routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/compliance-vault")
	{
		// Vault entries
		g.POST("/entries", h.StorePayload)
		g.GET("/entries", h.ListEntries)
		g.GET("/entries/:id", h.GetEntry)
		g.GET("/entries/:id/decrypt", h.DecryptPayload)

		// Retention policies
		g.POST("/retention-policies", h.CreateRetentionPolicy)
		g.GET("/retention-policies", h.ListRetentionPolicies)
		g.DELETE("/retention-policies/:id", h.DeleteRetentionPolicy)

		// Erasure requests (GDPR)
		g.POST("/erasure-requests", h.RequestErasure)
		g.GET("/erasure-requests", h.ListErasureRequests)

		// Compliance reports
		g.POST("/reports", h.GenerateComplianceReport)

		// Audit trail
		g.GET("/audit-trail", h.ListAuditTrail)

		// Stats
		g.GET("/stats", h.GetVaultStats)
	}
}

func (h *Handler) StorePayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req StorePayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.service.StorePayload(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func (h *Handler) GetEntry(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	entryID := c.Param("id")
	actorID := c.GetString("user_id")

	entry, err := h.service.GetEntry(c.Request.Context(), tenantID, entryID, actorID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (h *Handler) DecryptPayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	entryID := c.Param("id")
	actorID := c.GetString("user_id")

	plaintext, err := h.service.DecryptPayload(c.Request.Context(), tenantID, entryID, actorID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"payload": json.RawMessage(plaintext)})
}

func (h *Handler) ListEntries(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	entries, err := h.service.ListEntries(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func (h *Handler) CreateRetentionPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRetentionPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.service.CreateRetentionPolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

func (h *Handler) ListRetentionPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	policies, err := h.service.ListRetentionPolicies(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (h *Handler) DeleteRetentionPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policyID := c.Param("id")

	if err := h.service.DeleteRetentionPolicy(c.Request.Context(), tenantID, policyID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) RequestErasure(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateErasureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	erasure, err := h.service.RequestErasure(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, erasure)
}

func (h *Handler) ListErasureRequests(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	requests, err := h.service.ListErasureRequests(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"erasure_requests": requests})
}

func (h *Handler) GenerateComplianceReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.GenerateComplianceReport(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *Handler) ListAuditTrail(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	trail, err := h.service.ListAuditTrail(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_trail": trail})
}

func (h *Handler) GetVaultStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetVaultStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

package mtls

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for mTLS management
type Handler struct {
	service *Service
}

// NewHandler creates a new mTLS handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers mTLS routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	tls := router.Group("/mtls")
	{
		// Certificates
		tls.POST("/certificates", h.IssueCertificate)
		tls.GET("/certificates", h.ListCertificates)
		tls.GET("/certificates/:id", h.GetCertificate)
		tls.POST("/certificates/:id/renew", h.RenewCertificate)
		tls.POST("/certificates/:id/revoke", h.RevokeCertificate)
		tls.GET("/inventory", h.GetInventory)

		// TLS Policies
		tls.POST("/policies", h.CreateTLSPolicy)
		tls.GET("/policies", h.ListTLSPolicies)
		tls.GET("/policies/:endpoint_id", h.GetTLSPolicy)
		tls.PUT("/policies/:id", h.UpdateTLSPolicy)
		tls.DELETE("/policies/:id", h.DeleteTLSPolicy)

		// Maintenance
		tls.POST("/check-expiring", h.CheckExpiring)
	}
}

// @Summary Issue a new certificate
// @Tags mTLS
// @Accept json
// @Produce json
// @Param body body CertificateRequest true "Certificate request"
// @Success 201 {object} Certificate
// @Router /mtls/certificates [post]
func (h *Handler) IssueCertificate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	cert, err := h.service.IssueCertificate(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "ISSUE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, cert)
}

// @Summary List certificates
// @Tags mTLS
// @Produce json
// @Success 200 {object} map[string][]Certificate
// @Router /mtls/certificates [get]
func (h *Handler) ListCertificates(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	certs, err := h.service.ListCertificates(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"certificates": certs})
}

// @Summary Get a certificate
// @Tags mTLS
// @Produce json
// @Param id path string true "Certificate ID"
// @Success 200 {object} Certificate
// @Router /mtls/certificates/{id} [get]
func (h *Handler) GetCertificate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	certID := c.Param("id")

	cert, err := h.service.GetCertificate(c.Request.Context(), tenantID, certID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, cert)
}

// @Summary Renew a certificate
// @Tags mTLS
// @Produce json
// @Param id path string true "Certificate ID"
// @Success 200 {object} Certificate
// @Router /mtls/certificates/{id}/renew [post]
func (h *Handler) RenewCertificate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	certID := c.Param("id")

	cert, err := h.service.RenewCertificate(c.Request.Context(), tenantID, certID)
	if err != nil {
		httputil.InternalError(c, "RENEW_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, cert)
}

// @Summary Revoke a certificate
// @Tags mTLS
// @Param id path string true "Certificate ID"
// @Success 204
// @Router /mtls/certificates/{id}/revoke [post]
func (h *Handler) RevokeCertificate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	certID := c.Param("id")

	if err := h.service.RevokeCertificate(c.Request.Context(), tenantID, certID); err != nil {
		httputil.InternalError(c, "REVOKE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Get certificate inventory
// @Tags mTLS
// @Produce json
// @Success 200 {object} CertificateInventory
// @Router /mtls/inventory [get]
func (h *Handler) GetInventory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	inv, err := h.service.GetInventory(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "INVENTORY_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, inv)
}

// @Summary Create a TLS policy
// @Tags mTLS
// @Accept json
// @Produce json
// @Param body body TLSPolicyRequest true "TLS policy"
// @Success 201 {object} TLSPolicy
// @Router /mtls/policies [post]
func (h *Handler) CreateTLSPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req TLSPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	policy, err := h.service.CreateTLSPolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// @Summary List TLS policies
// @Tags mTLS
// @Produce json
// @Success 200 {object} map[string][]TLSPolicy
// @Router /mtls/policies [get]
func (h *Handler) ListTLSPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	policies, err := h.service.ListTLSPolicies(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// @Summary Get TLS policy for an endpoint
// @Tags mTLS
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200 {object} TLSPolicy
// @Router /mtls/policies/{endpoint_id} [get]
func (h *Handler) GetTLSPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpoint_id")

	policy, err := h.service.GetTLSPolicy(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// @Summary Update a TLS policy
// @Tags mTLS
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param body body TLSPolicyRequest true "Updated policy"
// @Success 200 {object} TLSPolicy
// @Router /mtls/policies/{id} [put]
func (h *Handler) UpdateTLSPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policyID := c.Param("id")

	var req TLSPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	policy, err := h.service.UpdateTLSPolicy(c.Request.Context(), tenantID, policyID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// @Summary Delete a TLS policy
// @Tags mTLS
// @Param id path string true "Policy ID"
// @Success 204
// @Router /mtls/policies/{id} [delete]
func (h *Handler) DeleteTLSPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policyID := c.Param("id")

	if err := h.service.DeleteTLSPolicy(c.Request.Context(), tenantID, policyID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Check for expiring certificates and auto-renew
// @Tags mTLS
// @Produce json
// @Success 200 {object} map[string]int
// @Router /mtls/check-expiring [post]
func (h *Handler) CheckExpiring(c *gin.Context) {
	renewed, err := h.service.CheckExpiringCerts(c.Request.Context())
	if err != nil {
		httputil.InternalError(c, "CHECK_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"renewed_count": renewed})
}

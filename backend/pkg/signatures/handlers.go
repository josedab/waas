package signatures

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for signature operations
type Handlers struct {
	service *Service
}

// NewHandlers creates new signature handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers signature routes
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	signatures := r.Group("/signatures")
	{
		signatures.GET("/schemes", h.ListSchemes)
		signatures.POST("/schemes", h.CreateScheme)
		signatures.GET("/schemes/:id", h.GetScheme)
		signatures.PUT("/schemes/:id", h.UpdateScheme)
		signatures.DELETE("/schemes/:id", h.DeleteScheme)

		signatures.GET("/schemes/:id/keys", h.GetKeys)
		signatures.POST("/schemes/:id/rotate", h.RotateKey)
		signatures.DELETE("/schemes/:id/keys/:keyId", h.RevokeKey)

		signatures.GET("/schemes/:id/rotations", h.GetRotations)
		signatures.GET("/schemes/:id/stats", h.GetStats)

		signatures.POST("/sign", h.Sign)
		signatures.POST("/verify", h.Verify)

		signatures.GET("/supported-types", h.GetSupportedTypes)
	}
}

// ListSchemes lists signature schemes
// @Summary List signature schemes
// @Tags Signatures
// @Produce json
// @Success 200 {array} SignatureScheme
// @Router /signatures/schemes [get]
func (h *Handlers) ListSchemes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	schemes, err := h.service.ListSchemes(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, schemes)
}

// CreateScheme creates a new signature scheme
// @Summary Create signature scheme
// @Tags Signatures
// @Accept json
// @Produce json
// @Param request body CreateSchemeRequest true "Scheme configuration"
// @Success 201 {object} SignatureScheme
// @Router /signatures/schemes [post]
func (h *Handlers) CreateScheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scheme, err := h.service.CreateScheme(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, scheme)
}

// GetScheme gets a signature scheme
// @Summary Get signature scheme
// @Tags Signatures
// @Produce json
// @Param id path string true "Scheme ID"
// @Success 200 {object} SignatureScheme
// @Router /signatures/schemes/{id} [get]
func (h *Handlers) GetScheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	scheme, err := h.service.GetScheme(c.Request.Context(), tenantID, schemeID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, scheme)
}

// UpdateScheme updates a signature scheme
// @Summary Update signature scheme
// @Tags Signatures
// @Accept json
// @Produce json
// @Param id path string true "Scheme ID"
// @Param request body UpdateSchemeRequest true "Update configuration"
// @Success 200 {object} SignatureScheme
// @Router /signatures/schemes/{id} [put]
func (h *Handlers) UpdateScheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	var req UpdateSchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scheme, err := h.service.UpdateScheme(c.Request.Context(), tenantID, schemeID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, scheme)
}

// DeleteScheme deletes a signature scheme
// @Summary Delete signature scheme
// @Tags Signatures
// @Param id path string true "Scheme ID"
// @Success 204
// @Router /signatures/schemes/{id} [delete]
func (h *Handlers) DeleteScheme(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	if err := h.service.DeleteScheme(c.Request.Context(), tenantID, schemeID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetKeys lists keys for a scheme
// @Summary List signing keys
// @Tags Signatures
// @Produce json
// @Param id path string true "Scheme ID"
// @Success 200 {array} SigningKey
// @Router /signatures/schemes/{id}/keys [get]
func (h *Handlers) GetKeys(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	keys, err := h.service.GetKeys(c.Request.Context(), tenantID, schemeID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	// Don't expose secret keys
	for i := range keys {
		keys[i].SecretKey = ""
		keys[i].PrivateKey = ""
	}

	c.JSON(http.StatusOK, keys)
}

// RotateKey initiates key rotation
// @Summary Rotate signing key
// @Tags Signatures
// @Accept json
// @Produce json
// @Param id path string true "Scheme ID"
// @Param request body RotateKeyRequest true "Rotation configuration"
// @Success 201 {object} KeyRotation
// @Router /signatures/schemes/{id}/rotate [post]
func (h *Handlers) RotateKey(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	var req RotateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RotateKeyRequest{} // Default values
	}

	rotation, err := h.service.RotateKey(c.Request.Context(), tenantID, schemeID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, rotation)
}

// RevokeKey revokes a signing key
// @Summary Revoke signing key
// @Tags Signatures
// @Param id path string true "Scheme ID"
// @Param keyId path string true "Key ID"
// @Success 204
// @Router /signatures/schemes/{id}/keys/{keyId} [delete]
func (h *Handlers) RevokeKey(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")
	keyID := c.Param("keyId")

	if err := h.service.RevokeKey(c.Request.Context(), tenantID, schemeID, keyID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRotations lists key rotations
// @Summary List key rotations
// @Tags Signatures
// @Produce json
// @Param id path string true "Scheme ID"
// @Success 200 {array} KeyRotation
// @Router /signatures/schemes/{id}/rotations [get]
func (h *Handlers) GetRotations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	rotations, err := h.service.GetRotations(c.Request.Context(), tenantID, schemeID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, rotations)
}

// GetStats gets scheme statistics
// @Summary Get scheme statistics
// @Tags Signatures
// @Produce json
// @Param id path string true "Scheme ID"
// @Success 200 {object} SchemeStats
// @Router /signatures/schemes/{id}/stats [get]
func (h *Handlers) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	schemeID := c.Param("id")

	stats, err := h.service.GetStats(c.Request.Context(), tenantID, schemeID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Sign signs a payload
// @Summary Sign payload
// @Tags Signatures
// @Accept json
// @Produce json
// @Param request body SignatureRequest true "Signing request"
// @Success 200 {object} SignatureResult
// @Router /signatures/sign [post]
func (h *Handlers) Sign(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req SignatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Sign(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Verify verifies a signature
// @Summary Verify signature
// @Tags Signatures
// @Accept json
// @Produce json
// @Param request body VerifyRequest true "Verification request"
// @Success 200 {object} VerifyResult
// @Router /signatures/verify [post]
func (h *Handlers) Verify(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Verify(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSupportedTypes returns supported signature types
// @Summary Get supported signature types
// @Tags Signatures
// @Produce json
// @Success 200 {array} SignatureSchemeInfo
// @Router /signatures/supported-types [get]
func (h *Handlers) GetSupportedTypes(c *gin.Context) {
	types := h.service.GetSupportedSchemes()
	c.JSON(http.StatusOK, types)
}

package e2ee

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for E2EE.
type Handler struct {
	service *Service
}

// NewHandler creates a new E2EE handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all E2EE routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/e2ee")
	{
		group.POST("/keys", h.GenerateKeyPair)
		group.GET("/keys/:endpoint_id", h.GetPublicKey)
		group.GET("/keys/:endpoint_id/all", h.ListKeyPairs)
		group.POST("/keys/rotate", h.RotateKey)
		group.POST("/encrypt", h.Encrypt)
		group.POST("/decrypt", h.Decrypt)
		group.GET("/health/:endpoint_id", h.CheckHealth)
		group.GET("/audit/:endpoint_id", h.GetAuditLog)
	}
}

// GenerateKeyPair creates a new key pair for an endpoint.
// @Summary Generate E2EE key pair
// @Tags e2ee
// @Accept json
// @Produce json
// @Success 201 {object} KeyPair
// @Router /e2ee/keys [post]
func (h *Handler) GenerateKeyPair(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req RegisterKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kp, err := h.service.GenerateKeyPair(tenantID, req.EndpointID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, kp)
}

// GetPublicKey returns the active public key for an endpoint.
// @Summary Get endpoint public key
// @Tags e2ee
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200
// @Router /e2ee/keys/{endpoint_id} [get]
func (h *Handler) GetPublicKey(c *gin.Context) {
	pubKey, version, err := h.service.GetPublicKey(c.Param("endpoint_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_key": pubKey, "version": version, "algorithm": "x25519"})
}

// ListKeyPairs returns all key pairs for an endpoint.
// @Summary List endpoint key pairs
// @Tags e2ee
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200 {array} KeyPair
// @Router /e2ee/keys/{endpoint_id}/all [get]
func (h *Handler) ListKeyPairs(c *gin.Context) {
	pairs, err := h.service.repo.ListKeyPairs(c.Param("endpoint_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, pairs)
}

// RotateKey rotates the key pair for an endpoint.
// @Summary Rotate E2EE key
// @Tags e2ee
// @Accept json
// @Produce json
// @Success 200 {object} KeyRotationResult
// @Router /e2ee/keys/rotate [post]
func (h *Handler) RotateKey(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req KeyRotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.RotateKey(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Encrypt encrypts a payload for an endpoint.
// @Summary Encrypt payload
// @Tags e2ee
// @Accept json
// @Produce json
// @Success 200 {object} EncryptedPayload
// @Router /e2ee/encrypt [post]
func (h *Handler) Encrypt(c *gin.Context) {
	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		Payload    string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encrypted, err := h.service.Encrypt(req.EndpointID, []byte(req.Payload))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, encrypted)
}

// Decrypt decrypts a payload.
// @Summary Decrypt payload
// @Tags e2ee
// @Accept json
// @Produce json
// @Success 200 {object} DecryptedPayload
// @Router /e2ee/decrypt [post]
func (h *Handler) Decrypt(c *gin.Context) {
	var req struct {
		EndpointID string            `json:"endpoint_id" binding:"required"`
		Encrypted  *EncryptedPayload `json:"encrypted" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	decrypted, err := h.service.Decrypt(req.EndpointID, req.Encrypted)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plaintext": string(decrypted.Plaintext), "verified": decrypted.Verified, "key_version": decrypted.KeyVersion})
}

// CheckHealth verifies encryption health for an endpoint.
// @Summary Check E2EE health
// @Tags e2ee
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Success 200 {object} HealthCheck
// @Router /e2ee/health/{endpoint_id} [get]
func (h *Handler) CheckHealth(c *gin.Context) {
	hc, err := h.service.CheckHealth(c.Param("endpoint_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, hc)
}

// GetAuditLog returns the audit log for an endpoint.
// @Summary Get E2EE audit log
// @Tags e2ee
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Param limit query int false "Max entries"
// @Success 200 {array} AuditEntry
// @Router /e2ee/audit/{endpoint_id} [get]
func (h *Handler) GetAuditLog(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	entries, err := h.service.GetAuditLog(c.Param("endpoint_id"), limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

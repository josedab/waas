package sdkgen

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for SDK generation.
type Handler struct {
	service *Service
}

// NewHandler creates a new SDK generation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers SDK generation routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/sdk-gen")
	{
		g.POST("/generate", h.GenerateSDK)
	}
}

func (h *Handler) GenerateSDK(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req GenerateSDKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sdk, err := h.service.GenerateSDK(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sdk)
}

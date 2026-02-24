package deliveryreceipt

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for delivery receipts.
type Handler struct {
	service *Service
}

// NewHandler creates a new delivery receipt handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers delivery receipt routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/delivery-receipts")
	{
		g.POST("", h.CreateReceipt)
		g.GET("", h.ListReceipts)
		g.GET("/:id", h.GetReceipt)
		g.POST("/:id/confirm", h.ConfirmReceipt)
		g.GET("/stats", h.GetStats)
	}
}

func (h *Handler) CreateReceipt(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receipt, err := h.service.CreateReceipt(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, receipt)
}

func (h *Handler) GetReceipt(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	receiptID := c.Param("id")

	receipt, err := h.service.GetReceipt(c.Request.Context(), tenantID, receiptID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, receipt)
}

func (h *Handler) ListReceipts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	receipts, err := h.service.ListReceipts(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"receipts": receipts})
}

func (h *Handler) ConfirmReceipt(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	receiptID := c.Param("id")

	var req ConfirmReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receipt, err := h.service.ConfirmReceipt(c.Request.Context(), tenantID, receiptID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, receipt)
}

func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

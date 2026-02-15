package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
)

// BidirectionalSyncHandler handles bi-directional sync HTTP endpoints
type BidirectionalSyncHandler struct {
	service *services.BidirectionalSyncService
	logger  *utils.Logger
}

// NewBidirectionalSyncHandler creates a new bi-directional sync handler
func NewBidirectionalSyncHandler(service *services.BidirectionalSyncService, logger *utils.Logger) *BidirectionalSyncHandler {
	return &BidirectionalSyncHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all bi-directional sync routes
func (h *BidirectionalSyncHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sync := rg.Group("/sync")
	{
		// Configurations
		sync.POST("/configs", h.CreateConfig)
		sync.GET("/configs", h.GetConfigs)
		sync.GET("/configs/:id", h.GetConfig)

		// Transactions (Request-Response mode)
		sync.POST("/transactions", h.SendSyncRequest)
		sync.POST("/transactions/:id/response", h.ReceiveSyncResponse)
		sync.GET("/transactions/:id", h.GetTransaction)
		sync.GET("/configs/:id/transactions", h.GetTransactions)

		// Acknowledgments (Event Acknowledgment mode)
		sync.POST("/acknowledgments", h.SendAcknowledgment)
		sync.POST("/acknowledgments/confirm", h.ConfirmAcknowledgment)

		// State Sync mode
		sync.POST("/state", h.UpdateState)
		sync.POST("/state/remote", h.ReceiveRemoteState)

		// Conflict Resolution
		sync.GET("/conflicts", h.GetConflicts)
		sync.POST("/conflicts/resolve", h.ResolveConflict)

		// Dashboard
		sync.GET("/dashboard", h.GetDashboard)
	}
}

// CreateConfig creates a new sync configuration
func (h *BidirectionalSyncHandler) CreateConfig(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.CreateSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfigs retrieves all sync configurations
func (h *BidirectionalSyncHandler) GetConfigs(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	configs, err := h.service.GetConfigs(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, configs)
}

// GetConfig retrieves a sync configuration
func (h *BidirectionalSyncHandler) GetConfig(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	config, err := h.service.GetConfig(c.Request.Context(), tenantID.(uuid.UUID), configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// SendSyncRequest sends a synchronous webhook request
func (h *BidirectionalSyncHandler) SendSyncRequest(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.SendSyncRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.service.SendSyncRequest(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

// ReceiveSyncResponse processes an incoming sync response
func (h *BidirectionalSyncHandler) ReceiveSyncResponse(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.ReceiveSyncResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.service.ReceiveSyncResponse(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// GetTransaction retrieves a sync transaction
func (h *BidirectionalSyncHandler) GetTransaction(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	txID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}

	tx, err := h.service.GetTransaction(c.Request.Context(), tenantID.(uuid.UUID), txID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// GetTransactions retrieves transactions for a config
func (h *BidirectionalSyncHandler) GetTransactions(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	limit := 20
	transactions, err := h.service.GetTransactions(c.Request.Context(), tenantID.(uuid.UUID), configID, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// SendAcknowledgment sends an acknowledgment for an event
func (h *BidirectionalSyncHandler) SendAcknowledgment(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.SendAcknowledgmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ack, err := h.service.SendAcknowledgment(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ack)
}

// ConfirmAcknowledgment confirms receipt of an acknowledgment
func (h *BidirectionalSyncHandler) ConfirmAcknowledgment(c *gin.Context) {
	var req struct {
		CorrelationID string `json:"correlation_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ConfirmAcknowledgment(c.Request.Context(), req.CorrelationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}

// UpdateState updates local state for state sync
func (h *BidirectionalSyncHandler) UpdateState(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.UpdateStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record, err := h.service.UpdateState(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// ReceiveRemoteState receives state update from remote
func (h *BidirectionalSyncHandler) ReceiveRemoteState(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req struct {
		ConfigID     string                 `json:"config_id" binding:"required"`
		ResourceType string                 `json:"resource_type" binding:"required"`
		ResourceID   string                 `json:"resource_id" binding:"required"`
		RemoteState  map[string]interface{} `json:"remote_state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
		return
	}

	record, err := h.service.ReceiveRemoteState(c.Request.Context(), tenantID.(uuid.UUID), configID, req.ResourceType, req.ResourceID, req.RemoteState)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// GetConflicts retrieves conflicted state records
func (h *BidirectionalSyncHandler) GetConflicts(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	conflicts, err := h.service.GetConflicts(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, conflicts)
}

// ResolveConflict resolves a state conflict
func (h *BidirectionalSyncHandler) ResolveConflict(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	var req models.ResolveConflictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record, err := h.service.ResolveConflict(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// GetDashboard retrieves the sync dashboard
func (h *BidirectionalSyncHandler) GetDashboard(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

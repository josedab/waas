package contracts

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for contract testing
type Handler struct {
	service *Service
}

// NewHandler creates a new contract testing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers contract testing routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	contracts := router.Group("/contracts")
	{
		contracts.POST("", h.CreateContract)
		contracts.GET("", h.ListContracts)
		contracts.GET("/status", h.GetContractStatus)
		contracts.GET("/:id", h.GetContract)
		contracts.PUT("/:id", h.UpdateContract)
		contracts.DELETE("/:id", h.DeleteContract)

		// Validation
		contracts.POST("/validate", h.ValidatePayload)
		contracts.POST("/:id/diff", h.DiffContracts)

		// Test results
		contracts.GET("/:id/results", h.GetTestResults)
	}
}

// @Summary Create a webhook contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param body body CreateContractRequest true "Contract definition"
// @Success 201 {object} Contract
// @Router /contracts [post]
func (h *Handler) CreateContract(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	contract, err := h.service.CreateContract(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, contract)
}

// @Summary List contracts
// @Tags Contracts
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /contracts [get]
func (h *Handler) ListContracts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	contracts, total, err := h.service.ListContracts(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"contracts": contracts, "total": total})
}

// @Summary Get a contract
// @Tags Contracts
// @Produce json
// @Param id path string true "Contract ID"
// @Success 200 {object} Contract
// @Router /contracts/{id} [get]
func (h *Handler) GetContract(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contractID := c.Param("id")

	contract, err := h.service.GetContract(c.Request.Context(), tenantID, contractID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, contract)
}

// @Summary Update a contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param id path string true "Contract ID"
// @Param body body CreateContractRequest true "Updated contract"
// @Success 200 {object} Contract
// @Router /contracts/{id} [put]
func (h *Handler) UpdateContract(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contractID := c.Param("id")

	var req CreateContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	contract, err := h.service.UpdateContract(c.Request.Context(), tenantID, contractID, &req)
	if err != nil {
		httputil.InternalError(c, "UPDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// @Summary Delete a contract
// @Tags Contracts
// @Param id path string true "Contract ID"
// @Success 204
// @Router /contracts/{id} [delete]
func (h *Handler) DeleteContract(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contractID := c.Param("id")

	if err := h.service.DeleteContract(c.Request.Context(), tenantID, contractID); err != nil {
		httputil.InternalError(c, "DELETE_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Validate a payload against a contract
// @Tags Contracts
// @Accept json
// @Produce json
// @Param body body ValidatePayloadRequest true "Payload to validate"
// @Success 200 {object} ContractTestResult
// @Router /contracts/validate [post]
func (h *Handler) ValidatePayload(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req ValidatePayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.ValidatePayload(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "VALIDATION_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Diff contract versions
// @Tags Contracts
// @Accept json
// @Produce json
// @Param id path string true "Contract ID"
// @Success 200 {object} SchemaDiff
// @Router /contracts/{id}/diff [post]
func (h *Handler) DiffContracts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contractID := c.Param("id")

	var req struct {
		OldVersion string `json:"old_version" binding:"required"`
		NewVersion string `json:"new_version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	diff, err := h.service.DiffContracts(c.Request.Context(), tenantID, contractID, req.OldVersion, req.NewVersion)
	if err != nil {
		httputil.InternalError(c, "DIFF_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, diff)
}

// @Summary Get contract status overview
// @Tags Contracts
// @Produce json
// @Success 200 {object} ContractStatus
// @Router /contracts/status [get]
func (h *Handler) GetContractStatus(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	status, err := h.service.GetContractStatus(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "STATUS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// @Summary Get test results for a contract
// @Tags Contracts
// @Produce json
// @Param id path string true "Contract ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string][]ContractTestResult
// @Router /contracts/{id}/results [get]
func (h *Handler) GetTestResults(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contractID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	results, err := h.service.GetTestResults(c.Request.Context(), tenantID, contractID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

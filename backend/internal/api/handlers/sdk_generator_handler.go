package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"webhook-platform/internal/api/services"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/utils"
)

// SDKGeneratorHandler handles SDK generator endpoints
type SDKGeneratorHandler struct {
	service *services.SDKGeneratorService
	logger  *utils.Logger
}

// NewSDKGeneratorHandler creates a new SDK generator handler
func NewSDKGeneratorHandler(service *services.SDKGeneratorService, logger *utils.Logger) *SDKGeneratorHandler {
	return &SDKGeneratorHandler{
		service: service,
		logger:  logger,
	}
}

// CreateConfig creates a new SDK configuration
// @Summary Create SDK configuration
// @Tags SDK Generator
// @Accept json
// @Produce json
// @Param request body models.CreateSDKConfigRequest true "SDK config"
// @Success 201 {object} models.SDKConfiguration
// @Router /sdk/configs [post]
func (h *SDKGeneratorHandler) CreateConfig(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var req models.CreateSDKConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to create SDK config", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfigs retrieves all SDK configurations for the tenant
// @Summary List SDK configurations
// @Tags SDK Generator
// @Produce json
// @Success 200 {array} models.SDKConfiguration
// @Router /sdk/configs [get]
func (h *SDKGeneratorHandler) GetConfigs(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	configs, err := h.service.GetConfigs(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		h.logger.Error("Failed to get SDK configs", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// GetConfig retrieves a specific SDK configuration
// @Summary Get SDK configuration
// @Tags SDK Generator
// @Produce json
// @Param config_id path string true "Config ID"
// @Success 200 {object} models.SDKConfiguration
// @Router /sdk/configs/{config_id} [get]
func (h *SDKGeneratorHandler) GetConfig(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	configID, err := uuid.Parse(c.Param("config_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
		return
	}

	config, err := h.service.GetConfig(c.Request.Context(), tenantID.(uuid.UUID), configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GenerateSDK triggers SDK generation
// @Summary Generate SDK
// @Tags SDK Generator
// @Accept json
// @Produce json
// @Param request body models.GenerateSDKRequest true "Generation request"
// @Success 202 {object} map[string]interface{}
// @Router /sdk/generate [post]
func (h *SDKGeneratorHandler) GenerateSDK(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var req models.GenerateSDKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.service.GenerateSDK(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to generate SDK", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "SDK generation started",
		"generations": results,
	})
}

// GetGenerations lists all generations for a config
// @Summary List SDK generations
// @Tags SDK Generator
// @Produce json
// @Param config_id path string true "Config ID"
// @Success 200 {array} models.SDKGeneration
// @Router /sdk/configs/{config_id}/generations [get]
func (h *SDKGeneratorHandler) GetGenerations(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	configID, err := uuid.Parse(c.Param("config_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
		return
	}

	generations, err := h.service.GetGenerations(c.Request.Context(), tenantID.(uuid.UUID), configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"generations": generations})
}

// GetGeneration retrieves a specific SDK generation
// @Summary Get SDK generation
// @Tags SDK Generator
// @Produce json
// @Param generation_id path string true "Generation ID"
// @Success 200 {object} models.SDKGeneration
// @Router /sdk/generations/{generation_id} [get]
func (h *SDKGeneratorHandler) GetGeneration(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	generationID, err := uuid.Parse(c.Param("generation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation_id"})
		return
	}

	generation, err := h.service.GetGeneration(c.Request.Context(), tenantID.(uuid.UUID), generationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
		return
	}

	c.JSON(http.StatusOK, generation)
}

// DownloadSDK records a download and returns the artifact URL
// @Summary Download SDK
// @Tags SDK Generator
// @Produce json
// @Param generation_id path string true "Generation ID"
// @Success 200 {object} map[string]interface{}
// @Router /sdk/generations/{generation_id}/download [get]
func (h *SDKGeneratorHandler) DownloadSDK(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	generationID, err := uuid.Parse(c.Param("generation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid generation_id"})
		return
	}

	generation, err := h.service.GetGeneration(c.Request.Context(), tenantID.(uuid.UUID), generationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "generation not found"})
		return
	}

	if generation.Status != models.SDKStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "SDK generation not completed",
			"status": generation.Status,
		})
		return
	}

	// Record download
	_ = h.service.RecordDownload(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		generationID,
		"direct",
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"artifact_url": generation.ArtifactURL,
		"package_name": generation.PackageName,
		"version":      generation.Version,
		"language":     generation.Language,
	})
}

// GetSupportedLanguages returns supported SDK languages
// @Summary Get supported languages
// @Tags SDK Generator
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /sdk/languages [get]
func (h *SDKGeneratorHandler) GetSupportedLanguages(c *gin.Context) {
	languages := []map[string]interface{}{
		{"code": models.SDKLanguageGo, "name": "Go", "registry": "pkg.go.dev"},
		{"code": models.SDKLanguageTypeScript, "name": "TypeScript/Node.js", "registry": "npmjs.com"},
		{"code": models.SDKLanguagePython, "name": "Python", "registry": "pypi.org"},
		{"code": models.SDKLanguageJava, "name": "Java", "registry": "maven.org"},
		{"code": models.SDKLanguageRuby, "name": "Ruby", "registry": "rubygems.org"},
		{"code": models.SDKLanguagePHP, "name": "PHP", "registry": "packagist.org"},
	}

	c.JSON(http.StatusOK, gin.H{"languages": languages})
}

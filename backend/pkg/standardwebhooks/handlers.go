package standardwebhooks

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for Standard Webhooks and CloudEvents.
type Handler struct {
	service *Service
}

// NewHandler creates a new Standard Webhooks handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers Standard Webhooks and CloudEvents routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/standard-webhooks")
	{
		g.POST("/detect", h.DetectFormat)
		g.POST("/sign", h.Sign)
		g.POST("/verify", h.Verify)
		g.POST("/convert", h.Convert)
		g.GET("/conformance/:format", h.RunConformance)
	}
}

func (h *Handler) DetectFormat(c *gin.Context) {
	var req DetectFormatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.service.DetectFormat(req.Headers, req.Payload)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Sign(c *gin.Context) {
	var req SignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Sign(req.WebhookID, req.Payload, req.Secret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Verify(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Verify(req.Headers, req.Payload, req.Secret, req.Tolerance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Convert(c *gin.Context) {
	var req ConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result *ConversionResult
	var err error

	switch req.TargetFormat {
	case FormatCloudEvents:
		result, err = h.service.ConvertToCloudEvents(req.Headers, req.Payload, req.Source, req.EventType)
	case FormatStandardWebhooks:
		result, err = h.service.ConvertToStandardWebhooks(req.Headers, req.Payload)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_format must be 'cloudevents' or 'standard_webhooks'"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) RunConformance(c *gin.Context) {
	format := c.Param("format")

	result := h.service.RunConformanceTests(format)
	c.JSON(http.StatusOK, result)
}

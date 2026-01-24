package callback

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pkgerrors "webhook-platform/pkg/errors"
)

// Handler handles callback HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new callback handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers callback routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	callbacks := r.Group("/callbacks")
	{
		callbacks.POST("/send", h.SendWithCallback)
		callbacks.GET("/requests", h.ListCallbackRequests)
		callbacks.GET("/requests/:id", h.GetCallbackRequest)
		callbacks.POST("/receive/:correlationId", h.ReceiveCallback)
		callbacks.POST("/wait/:correlationId", h.WaitForCallback)
		callbacks.POST("/long-poll", h.CreateLongPollSession)
		callbacks.GET("/long-poll/:id/events", h.PollForEvents)
		callbacks.POST("/patterns", h.RegisterPattern)
		callbacks.GET("/patterns", h.GetPatterns)
		callbacks.GET("/metrics", h.GetMetrics)
	}
}

func (h *Handler) SendWithCallback(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	var req CreateCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	cbReq, err := h.service.SendWithCallback(c.Request.Context(), tid, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, cbReq)
}

func (h *Handler) ListCallbackRequests(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	limit, offset := parsePagination(c)

	requests, total, err := h.service.ListCallbackRequests(c.Request.Context(), tid, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *Handler) GetCallbackRequest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid request ID")
		return
	}

	cbReq, err := h.service.GetCallbackRequest(c.Request.Context(), tid, requestID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "callback request")
		return
	}

	c.JSON(http.StatusOK, cbReq)
}

func (h *Handler) ReceiveCallback(c *gin.Context) {
	correlationID := c.Param("correlationId")
	if correlationID == "" {
		pkgerrors.AbortWithValidationError(c, "correlation_id", "correlation ID is required")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pkgerrors.HandleBindError(c, fmt.Errorf("failed to read request body"))
		return
	}

	headersJSON, _ := json.Marshal(c.Request.Header)
	statusCode := http.StatusOK
	if sc := c.Query("status_code"); sc != "" {
		if parsed, err := strconv.Atoi(sc); err == nil {
			statusCode = parsed
		}
	}

	resp, err := h.service.ReceiveCallback(c.Request.Context(), correlationID, statusCode, body, headersJSON)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) WaitForCallback(c *gin.Context) {
	correlationID := c.Param("correlationId")
	if correlationID == "" {
		pkgerrors.AbortWithValidationError(c, "correlation_id", "correlation ID is required")
		return
	}

	timeoutMs := 30000
	if t := c.Query("timeout_ms"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			timeoutMs = parsed
		}
	}

	resp, err := h.service.WaitForCallback(c.Request.Context(), correlationID, timeoutMs)
	if err != nil {
		pkgerrors.AbortWithRateLimit(c, 30)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateLongPollSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	var req CreateLongPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	session, err := h.service.CreateLongPollSession(c.Request.Context(), tid, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *Handler) PollForEvents(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "id", "invalid session ID")
		return
	}

	events, err := h.service.PollForEvents(c.Request.Context(), sessionID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *Handler) RegisterPattern(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	var req RegisterPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	pattern, err := h.service.RegisterPattern(c.Request.Context(), tid, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, pattern)
}

func (h *Handler) GetPatterns(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	limit, offset := parsePagination(c)

	patterns, total, err := h.service.GetPatterns(c.Request.Context(), tid, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patterns": patterns,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		pkgerrors.AbortWithValidationError(c, "tenant_id", "invalid tenant ID")
		return
	}

	metrics, err := h.service.GetMetrics(c.Request.Context(), tid)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func parsePagination(c *gin.Context) (int, int) {
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}
	return limit, offset
}

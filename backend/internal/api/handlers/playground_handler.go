package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"webhook-platform/pkg/playground"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlaygroundHandler handles webhook playground/IDE HTTP requests
type PlaygroundHandler struct {
	service *playground.Service
	logger  *utils.Logger
}

// NewPlaygroundHandler creates a new playground handler
func NewPlaygroundHandler(service *playground.Service, logger *utils.Logger) *PlaygroundHandler {
	return &PlaygroundHandler{
		service: service,
		logger:  logger,
	}
}

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	Name string `json:"name,omitempty"`
}

// ExecuteTransformRequest represents a transformation execution request
type ExecuteTransformRequest struct {
	Code  string          `json:"code" binding:"required"`
	Input json.RawMessage `json:"input" binding:"required"`
}

// SaveSessionRequest represents a request to save a session
type SaveSessionRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateSnippetRequest represents a request to create a snippet
type CreateSnippetRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"`
	SnippetType string   `json:"snippet_type" binding:"required"`
	Content     string   `json:"content" binding:"required"`
	Language    string   `json:"language,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsPublic    bool     `json:"is_public"`
}

// CaptureRequestRequest represents a captured request
type CaptureRequestRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        json.RawMessage   `json:"body,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// CreateSession creates a new playground session
// @Summary Create playground session
// @Tags playground
// @Accept json
// @Produce json
// @Param request body CreateSessionRequest false "Session name"
// @Success 201 {object} playground.Session
// @Router /playground/sessions [post]
func (h *PlaygroundHandler) CreateSession(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var req CreateSessionRequest
	c.ShouldBindJSON(&req)

	session, err := h.service.CreateSession(c.Request.Context(), tenantID, req.Name)
	if err != nil {
		h.logger.Error("Failed to create session", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "CREATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession retrieves a session
// @Summary Get playground session
// @Tags playground
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} playground.Session
// @Router /playground/sessions/{id} [get]
func (h *PlaygroundHandler) GetSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid session ID"})
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "NOT_FOUND", Message: "Session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// SaveSession saves a session permanently
// @Summary Save playground session
// @Tags playground
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param request body SaveSessionRequest true "Session name"
// @Success 200 {object} playground.Session
// @Router /playground/sessions/{id}/save [post]
func (h *PlaygroundHandler) SaveSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid session ID"})
		return
	}

	var req SaveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	session, err := h.service.SaveSession(c.Request.Context(), id, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "SAVE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ExecuteTransformation executes a transformation in the playground
// @Summary Execute transformation
// @Tags playground
// @Accept json
// @Produce json
// @Param id path string false "Session ID"
// @Param request body ExecuteTransformRequest true "Transformation code and input"
// @Success 200 {object} playground.TransformationExecution
// @Router /playground/execute [post]
func (h *PlaygroundHandler) ExecuteTransformation(c *gin.Context) {
	var sessionID *uuid.UUID
	if idStr := c.Param("id"); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			sessionID = &id
		}
	}

	var req ExecuteTransformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	execution, err := h.service.ExecuteTransformation(c.Request.Context(), sessionID, req.Code, req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "EXECUTION_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// GetExecutionHistory returns execution history for a session
// @Summary Get execution history
// @Tags playground
// @Produce json
// @Param id path string true "Session ID"
// @Param limit query int false "Limit"
// @Success 200 {array} playground.TransformationExecution
// @Router /playground/sessions/{id}/history [get]
func (h *PlaygroundHandler) GetExecutionHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid session ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	history, err := h.service.GetExecutionHistory(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "GET_HISTORY_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// CaptureRequest captures a request for inspection
// @Summary Capture request
// @Tags playground
// @Accept json
// @Produce json
// @Param request body CaptureRequestRequest true "Request data"
// @Success 201 {object} playground.RequestCapture
// @Router /playground/capture [post]
func (h *PlaygroundHandler) CaptureRequest(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var req CaptureRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	headersJSON, _ := json.Marshal(req.Headers)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)

	capture := &playground.RequestCapture{
		TenantID:    tenantID,
		Method:      req.Method,
		URL:         req.URL,
		Headers:     headersJSON,
		Body:        req.Body,
		QueryParams: queryParamsJSON,
		Tags:        req.Tags,
		Source:      "manual",
	}

	if err := h.service.CaptureRequest(c.Request.Context(), capture); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "CAPTURE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, capture)
}

// ListCaptures lists captured requests
// @Summary List captured requests
// @Tags playground
// @Produce json
// @Param session_id query string false "Session ID"
// @Param limit query int false "Limit"
// @Success 200 {array} playground.RequestCapture
// @Router /playground/captures [get]
func (h *PlaygroundHandler) ListCaptures(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var sessionID *uuid.UUID
	if sidStr := c.Query("session_id"); sidStr != "" {
		if sid, err := uuid.Parse(sidStr); err == nil {
			sessionID = &sid
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	captures, err := h.service.ListCaptures(c.Request.Context(), tenantID, sessionID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "LIST_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, captures)
}

// ReplayRequest replays a captured request
// @Summary Replay captured request
// @Tags playground
// @Produce json
// @Param id path string true "Capture ID"
// @Success 200 {object} playground.RequestCapture
// @Router /playground/captures/{id}/replay [post]
func (h *PlaygroundHandler) ReplayRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_ID", Message: "Invalid capture ID"})
		return
	}

	replay, err := h.service.ReplayRequest(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "REPLAY_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, replay)
}

// CreateSnippet creates a new code snippet
// @Summary Create snippet
// @Tags playground
// @Accept json
// @Produce json
// @Param request body CreateSnippetRequest true "Snippet data"
// @Success 201 {object} playground.Snippet
// @Router /playground/snippets [post]
func (h *PlaygroundHandler) CreateSnippet(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var req CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	language := req.Language
	if language == "" {
		language = "javascript"
	}

	snippet := &playground.Snippet{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		SnippetType: req.SnippetType,
		Content:     req.Content,
		Language:    language,
		Tags:        req.Tags,
		IsPublic:    req.IsPublic,
	}

	if err := h.service.CreateSnippet(c.Request.Context(), snippet); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "CREATE_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snippet)
}

// ListSnippets lists saved snippets
// @Summary List snippets
// @Tags playground
// @Produce json
// @Param type query string false "Snippet type filter"
// @Success 200 {array} playground.Snippet
// @Router /playground/snippets [get]
func (h *PlaygroundHandler) ListSnippets(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	snippetType := c.Query("type")

	snippets, err := h.service.ListSnippets(c.Request.Context(), tenantID, snippetType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "LIST_FAILED", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, snippets)
}

// RegisterPlaygroundRoutes registers playground API routes
func RegisterPlaygroundRoutes(r *gin.RouterGroup, h *PlaygroundHandler) {
	pg := r.Group("/playground")
	{
		// Sessions
		pg.POST("/sessions", h.CreateSession)
		pg.GET("/sessions/:id", h.GetSession)
		pg.POST("/sessions/:id/save", h.SaveSession)
		pg.GET("/sessions/:id/history", h.GetExecutionHistory)

		// Transformation execution
		pg.POST("/execute", h.ExecuteTransformation)
		pg.POST("/sessions/:id/execute", h.ExecuteTransformation)

		// Request capture
		pg.POST("/capture", h.CaptureRequest)
		pg.GET("/captures", h.ListCaptures)
		pg.POST("/captures/:id/replay", h.ReplayRequest)

		// Snippets
		pg.POST("/snippets", h.CreateSnippet)
		pg.GET("/snippets", h.ListSnippets)
	}
}

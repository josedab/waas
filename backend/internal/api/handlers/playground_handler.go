package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/josedab/waas/pkg/playground"
	"github.com/josedab/waas/pkg/utils"

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
		InternalError(c, "CREATE_FAILED", err)
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
		InternalError(c, "SAVE_FAILED", err)
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
		InternalError(c, "EXECUTION_FAILED", err)
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
		InternalError(c, "GET_HISTORY_FAILED", err)
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
		InternalError(c, "CAPTURE_FAILED", err)
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
		InternalError(c, "LIST_FAILED", err)
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
		InternalError(c, "REPLAY_FAILED", err)
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
		InternalError(c, "CREATE_FAILED", err)
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
		InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, snippets)
}

// CreateSharedScenarioRequest represents a request to create a shared scenario
type CreateSharedScenarioRequest struct {
	SessionID   string            `json:"session_id" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Payload     string            `json:"payload" binding:"required"`
	TargetURL   string            `json:"target_url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// CreateSharedScenario creates a shareable test scenario
// @Summary Create shared scenario
// @Tags playground
// @Accept json
// @Produce json
// @Param request body CreateSharedScenarioRequest true "Scenario data"
// @Success 201 {object} playground.SharedScenario
// @Router /playground/scenarios [post]
func (h *PlaygroundHandler) CreateSharedScenario(c *gin.Context) {
	var req CreateSharedScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	scenario, err := h.service.CreateSharedScenario(c.Request.Context(), req.SessionID, req.Name, req.Description, req.Payload, req.TargetURL, req.Headers)
	if err != nil {
		h.logger.Error("Failed to create shared scenario", map[string]interface{}{"error": err.Error()})
		InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, scenario)
}

// GetSharedScenario retrieves a shared scenario by token
// @Summary Get shared scenario
// @Tags playground
// @Produce json
// @Param token path string true "Scenario token"
// @Success 200 {object} playground.SharedScenario
// @Router /playground/scenarios/{token} [get]
func (h *PlaygroundHandler) GetSharedScenario(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_TOKEN", Message: "Token is required"})
		return
	}

	scenario, err := h.service.GetSharedScenario(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "NOT_FOUND", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, scenario)
}

// CreateTestSuiteRequest represents a request to create a test suite
type CreateTestSuiteRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public"`
}

// CreateTestSuite creates a new test suite
// @Summary Create test suite
// @Tags playground
// @Accept json
// @Produce json
// @Param request body CreateTestSuiteRequest true "Test suite data"
// @Success 201 {object} playground.TestSuite
// @Router /playground/test-suites [post]
func (h *PlaygroundHandler) CreateTestSuite(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	var req CreateTestSuiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	suite, err := h.service.CreateTestSuite(c.Request.Context(), tenantID, req.Name, req.Description, req.IsPublic)
	if err != nil {
		h.logger.Error("Failed to create test suite", map[string]interface{}{"error": err.Error()})
		InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, suite)
}

// ListTestSuites lists test suites for a tenant
// @Summary List test suites
// @Tags playground
// @Produce json
// @Success 200 {array} playground.TestSuite
// @Router /playground/test-suites [get]
func (h *PlaygroundHandler) ListTestSuites(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant"})
		return
	}

	suites, err := h.service.ListTestSuites(c.Request.Context(), tenantID)
	if err != nil {
		InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, suites)
}

// ReplayProductionEventRequest represents a request to replay a production event
type ReplayProductionEventRequest struct {
	EventID         string   `json:"event_id" binding:"required"`
	SensitiveFields []string `json:"sensitive_fields,omitempty"`
}

// ReplayProductionEvent replays a production event with redacted fields
// @Summary Replay production event
// @Tags playground
// @Accept json
// @Produce json
// @Param request body ReplayProductionEventRequest true "Replay data"
// @Success 200 {object} playground.SanitizedReplay
// @Router /playground/replay [post]
func (h *PlaygroundHandler) ReplayProductionEvent(c *gin.Context) {
	var req ReplayProductionEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	replay, err := h.service.ReplayProductionEvent(c.Request.Context(), req.EventID, req.SensitiveFields)
	if err != nil {
		InternalError(c, "REPLAY_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, replay)
}

// ComputeDiffRequest represents a request to compute a diff between two request/response pairs
type ComputeDiffRequest struct {
	OldRequest  map[string]interface{} `json:"old_request"`
	NewRequest  map[string]interface{} `json:"new_request"`
	OldResponse map[string]interface{} `json:"old_response"`
	NewResponse map[string]interface{} `json:"new_response"`
}

// ComputeDiff computes the diff between old and new request/response pairs
// @Summary Compute diff
// @Tags playground
// @Accept json
// @Produce json
// @Param request body ComputeDiffRequest true "Diff data"
// @Success 200 {object} playground.DiffResult
// @Router /playground/diff [post]
func (h *PlaygroundHandler) ComputeDiff(c *gin.Context) {
	var req ComputeDiffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	result := h.service.ComputeDiff(req.OldRequest, req.NewRequest, req.OldResponse, req.NewResponse)
	c.JSON(http.StatusOK, result)
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

		// Playground v2: Shareable scenarios
		pg.POST("/scenarios", h.CreateSharedScenario)
		pg.GET("/scenarios/:token", h.GetSharedScenario)

		// Playground v2: Test suites
		pg.POST("/test-suites", h.CreateTestSuite)
		pg.GET("/test-suites", h.ListTestSuites)

		// Playground v2: Production replay & diff
		pg.POST("/replay", h.ReplayProductionEvent)
		pg.POST("/diff", h.ComputeDiff)
	}
}

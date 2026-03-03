package playground

import (
	"github.com/josedab/waas/pkg/httputil"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles playground HTTP requests
type Handler struct {
	service *Service
	hub     *WebSocketHub
}

// NewHandler creates a new playground handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service, hub: NewWebSocketHub()}
}

// RegisterRoutes registers playground routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	pg := r.Group("/playground")
	{
		// Sessions
		pg.POST("/sessions", h.CreateSession)
		pg.GET("/sessions/:id", h.GetSession)
		pg.DELETE("/sessions/:id", h.DeleteSession)

		// Transformations
		pg.POST("/sessions/:id/execute", h.ExecuteTransformation)

		// Request capture
		pg.POST("/sessions/:id/capture", h.CaptureRequest)
		pg.GET("/sessions/:id/captures", h.ListCaptures)
		pg.POST("/sessions/:id/captures/:captureId/replay", h.ReplayCapture)

		// Snippets
		pg.POST("/snippets", h.SaveSnippet)
		pg.GET("/snippets", h.ListSnippets)
		pg.GET("/snippets/:snippetId", h.GetSnippet)

		// V2: Shared scenarios
		pg.POST("/sessions/:id/share", h.ShareScenario)
		pg.GET("/shared/:token", h.GetSharedScenario)

		// V2: Test suites
		pg.POST("/suites", h.CreateTestSuite)
		pg.GET("/suites", h.ListTestSuites)
		pg.GET("/suites/:suiteId", h.GetTestSuite)
		pg.POST("/suites/:suiteId/scenarios", h.AddScenarioToSuite)
		pg.POST("/suites/:suiteId/run", h.RunTestSuite)
	}
}

// CreateSession creates a new playground session
// @Summary Create playground session
// @Tags Playground
// @Accept json
// @Produce json
// @Success 201 {object} Session
// @Router /playground/sessions [post]
func (h *Handler) CreateSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Name = "Untitled Session"
	}

	session, err := h.service.CreateSession(c.Request.Context(), tid, req.Name)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, session)
}

// GetSession retrieves a playground session
// @Summary Get playground session
// @Tags Playground
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} Session
// @Router /playground/sessions/{id} [get]
func (h *Handler) GetSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, session)
}

// DeleteSession deletes a playground session
// @Summary Delete playground session
// @Tags Playground
// @Param id path string true "Session ID"
// @Success 204
// @Router /playground/sessions/{id} [delete]
func (h *Handler) DeleteSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	if err := h.service.DeleteSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ExecuteTransformation runs a transformation in a session
// @Summary Execute transformation
// @Tags Playground
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} TransformationExecution
// @Router /playground/sessions/{id}/execute [post]
func (h *Handler) ExecuteTransformation(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req struct {
		Code    string `json:"code" binding:"required"`
		Payload string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ExecuteTransformation(c.Request.Context(), &sessionID, req.Code, json.RawMessage(req.Payload))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CaptureRequest captures an HTTP request in a session
// @Summary Capture HTTP request
// @Tags Playground
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 201 {object} RequestCapture
// @Router /playground/sessions/{id}/capture [post]
func (h *Handler) CaptureRequest(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	var req struct {
		Method  string            `json:"method" binding:"required"`
		URL     string            `json:"url" binding:"required"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	headersJSON, err := json.Marshal(req.Headers)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid headers format"})
		return
	}
	capture := &RequestCapture{
		TenantID:  tid,
		SessionID: &sessionID,
		Method:    req.Method,
		URL:       req.URL,
		Headers:   headersJSON,
		Body:      json.RawMessage(req.Body),
		Source:    "playground",
	}

	if err := h.service.CaptureRequest(c.Request.Context(), capture); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, capture)
}

// ListCaptures lists request captures for a session
// @Summary List captures
// @Tags Playground
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {array} RequestCapture
// @Router /playground/sessions/{id}/captures [get]
func (h *Handler) ListCaptures(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	captures, err := h.service.ListCaptures(c.Request.Context(), tid, &sessionID, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"captures": captures})
}

// ReplayCapture replays a captured request
// @Summary Replay captured request
// @Tags Playground
// @Produce json
// @Param id path string true "Session ID"
// @Param captureId path string true "Capture ID"
// @Success 200 {object} RequestCapture
// @Router /playground/sessions/{id}/captures/{captureId}/replay [post]
func (h *Handler) ReplayCapture(c *gin.Context) {
	captureID, err := uuid.Parse(c.Param("captureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid capture ID"})
		return
	}

	capture, err := h.service.ReplayRequest(c.Request.Context(), captureID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, capture)
}

// SaveSnippet saves a code snippet
// @Summary Save snippet
// @Tags Playground
// @Accept json
// @Produce json
// @Success 201 {object} Snippet
// @Router /playground/snippets [post]
func (h *Handler) SaveSnippet(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Code        string `json:"code" binding:"required"`
		Language    string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snippet := &Snippet{
		TenantID:    tid,
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Code,
		Language:    req.Language,
		SnippetType: "transformation",
	}
	if err := h.service.CreateSnippet(c.Request.Context(), snippet); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, snippet)
}

// ListSnippets lists saved snippets
// @Summary List snippets
// @Tags Playground
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} Snippet
// @Router /playground/snippets [get]
func (h *Handler) ListSnippets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	snippetType := c.DefaultQuery("type", "")
	snippets, err := h.service.ListSnippets(c.Request.Context(), tid, snippetType)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"snippets": snippets})
}

// GetSnippet retrieves a snippet by ID
// @Summary Get snippet
// @Tags Playground
// @Produce json
// @Param snippetId path string true "Snippet ID"
// @Success 200 {object} Snippet
// @Router /playground/snippets/{snippetId} [get]
func (h *Handler) GetSnippet(c *gin.Context) {
	snippetID, err := uuid.Parse(c.Param("snippetId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snippet ID"})
		return
	}

	snippet, err := h.service.GetSnippet(c.Request.Context(), snippetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snippet not found"})
		return
	}
	c.JSON(http.StatusOK, snippet)
}

// ShareScenario creates a shareable link for a session scenario
// @Summary Share scenario
// @Tags Playground
// @Produce json
// @Param id path string true "Session ID"
// @Success 201 {object} SharedScenario
// @Router /playground/sessions/{id}/share [post]
func (h *Handler) ShareScenario(c *gin.Context) {
	sessionID := c.Param("id")

	var req struct {
		Title       string            `json:"title"`
		Description string            `json:"description"`
		Payload     string            `json:"payload"`
		TargetURL   string            `json:"target_url"`
		Headers     map[string]string `json:"headers"`
	}
	c.ShouldBindJSON(&req)

	scenario, err := h.service.CreateSharedScenario(c.Request.Context(), sessionID, req.Title, req.Description, req.Payload, req.TargetURL, req.Headers)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, scenario)
}

// GetSharedScenario retrieves a shared scenario by token
// @Summary Get shared scenario
// @Tags Playground
// @Produce json
// @Param token path string true "Share token"
// @Success 200 {object} SharedScenario
// @Router /playground/shared/{token} [get]
func (h *Handler) GetSharedScenario(c *gin.Context) {
	token := c.Param("token")
	scenario, err := h.service.GetSharedScenario(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shared scenario not found or expired"})
		return
	}
	c.JSON(http.StatusOK, scenario)
}

// CreateTestSuite creates a new test suite
// @Summary Create test suite
// @Tags Playground
// @Accept json
// @Produce json
// @Success 201 {object} TestSuite
// @Router /playground/suites [post]
func (h *Handler) CreateTestSuite(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	suite, err := h.service.CreateTestSuite(c.Request.Context(), tenantID, req.Name, req.Description, req.IsPublic)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, suite)
}

// ListTestSuites lists test suites
// @Summary List test suites
// @Tags Playground
// @Produce json
// @Success 200 {array} TestSuite
// @Router /playground/suites [get]
func (h *Handler) ListTestSuites(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	suites, err := h.service.ListTestSuites(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suites": suites})
}

// GetTestSuite retrieves a test suite
// @Summary Get test suite
// @Tags Playground
// @Produce json
// @Param suiteId path string true "Suite ID"
// @Success 200 {object} TestSuite
// @Router /playground/suites/{suiteId} [get]
func (h *Handler) GetTestSuite(c *gin.Context) {
	suiteID := c.Param("suiteId")
	suite, err := h.service.GetTestSuite(c.Request.Context(), suiteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test suite not found"})
		return
	}
	c.JSON(http.StatusOK, suite)
}

// AddScenarioToSuite adds a scenario to a test suite
// @Summary Add scenario to suite
// @Tags Playground
// @Accept json
// @Produce json
// @Param suiteId path string true "Suite ID"
// @Success 200 {object} TestSuite
// @Router /playground/suites/{suiteId}/scenarios [post]
func (h *Handler) AddScenarioToSuite(c *gin.Context) {
	suiteID := c.Param("suiteId")

	var req struct {
		ScenarioToken string `json:"scenario_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	suite, err := h.service.AddScenarioToSuite(c.Request.Context(), suiteID, req.ScenarioToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, suite)
}

// RunTestSuite runs all scenarios in a test suite
// @Summary Run test suite
// @Tags Playground
// @Produce json
// @Param suiteId path string true "Suite ID"
// @Success 200 {object} map[string]interface{}
// @Router /playground/suites/{suiteId}/run [post]
func (h *Handler) RunTestSuite(c *gin.Context) {
	suiteID := c.Param("suiteId")
	results, err := h.service.RunTestSuite(c.Request.Context(), suiteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suite_id": suiteID, "results": results})
}

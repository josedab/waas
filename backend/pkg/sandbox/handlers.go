package sandbox

import (
	"github.com/josedab/waas/pkg/httputil"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for sandbox management
type Handler struct {
	service *Service
}

// NewHandler creates a new sandbox handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers sandbox routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	sb := router.Group("/sandbox")
	{
		sb.POST("", h.CreateSandbox)
		sb.GET("", h.ListSandboxes)
		sb.GET("/:id", h.GetSandbox)
		sb.POST("/:id/replay", h.ReplayEvents)
		sb.GET("/:id/compare", h.GetComparisonReport)
		sb.POST("/:id/terminate", h.TerminateSandbox)
		sb.POST("/cleanup", h.CleanupExpired)

		// Mock endpoint routes
		sb.POST("/environments/:id/mock", h.CreateMockEndpoint)
		sb.POST("/environments/:id/mock/:endpointId", h.HandleMockRequest)
		sb.GET("/environments/:id/requests", h.GetCapturedRequests)
		sb.POST("/environments/:id/chaos", h.InjectChaos)

		// Test scenario routes
		sb.POST("/scenarios", h.CreateTestScenario)
		sb.GET("/scenarios", h.ListTestScenarios)
		sb.POST("/scenarios/:id/run", h.RunTestScenario)
		sb.GET("/scenarios/:id/results", h.GetScenarioResults)
	}
}

// @Summary Create a sandbox environment
// @Tags Sandbox
// @Accept json
// @Produce json
// @Param body body CreateSandboxRequest true "Sandbox configuration"
// @Success 201 {object} SandboxEnvironment
// @Router /sandbox [post]
func (h *Handler) CreateSandbox(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateSandboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	sandbox, err := h.service.CreateSandbox(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, sandbox)
}

// @Summary List sandbox environments
// @Tags Sandbox
// @Produce json
// @Success 200 {object} map[string][]SandboxEnvironment
// @Router /sandbox [get]
func (h *Handler) ListSandboxes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	sandboxes, err := h.service.ListSandboxes(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"sandboxes": sandboxes})
}

// @Summary Get a sandbox environment
// @Tags Sandbox
// @Produce json
// @Param id path string true "Sandbox ID"
// @Success 200 {object} SandboxEnvironment
// @Router /sandbox/{id} [get]
func (h *Handler) GetSandbox(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sandboxID := c.Param("id")

	sandbox, err := h.service.GetSandbox(c.Request.Context(), tenantID, sandboxID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, sandbox)
}

// @Summary Replay events in a sandbox
// @Tags Sandbox
// @Accept json
// @Produce json
// @Param id path string true "Sandbox ID"
// @Param body body ReplayRequest true "Replay configuration"
// @Success 200 {object} map[string][]ReplaySession
// @Router /sandbox/{id}/replay [post]
func (h *Handler) ReplayEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sandboxID := c.Param("id")

	var req ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	sessions, err := h.service.ReplayEvents(c.Request.Context(), tenantID, sandboxID, &req)
	if err != nil {
		httputil.InternalError(c, "REPLAY_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions, "count": len(sessions)})
}

// @Summary Get comparison report for a sandbox
// @Tags Sandbox
// @Produce json
// @Param id path string true "Sandbox ID"
// @Success 200 {object} ComparisonReport
// @Router /sandbox/{id}/compare [get]
func (h *Handler) GetComparisonReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sandboxID := c.Param("id")

	report, err := h.service.GetComparisonReport(c.Request.Context(), tenantID, sandboxID)
	if err != nil {
		httputil.InternalError(c, "COMPARISON_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, report)
}

// @Summary Terminate a sandbox environment
// @Tags Sandbox
// @Produce json
// @Param id path string true "Sandbox ID"
// @Success 200 {object} SandboxEnvironment
// @Router /sandbox/{id}/terminate [post]
func (h *Handler) TerminateSandbox(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sandboxID := c.Param("id")

	sandbox, err := h.service.TerminateSandbox(c.Request.Context(), tenantID, sandboxID)
	if err != nil {
		httputil.InternalError(c, "TERMINATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, sandbox)
}

// @Summary Cleanup expired sandbox environments
// @Tags Sandbox
// @Produce json
// @Success 200 {object} map[string]int64
// @Router /sandbox/cleanup [post]
func (h *Handler) CleanupExpired(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	count, err := h.service.CleanupExpired(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "CLEANUP_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted_count": count})
}

// @Summary Create a mock endpoint for a sandbox
// @Tags Sandbox
// @Accept json
// @Produce json
// @Param id path string true "Sandbox ID"
// @Param body body CreateMockEndpointRequest true "Mock endpoint configuration"
// @Success 201 {object} MockEndpointConfig
// @Router /sandbox/environments/{id}/mock [post]
func (h *Handler) CreateMockEndpoint(c *gin.Context) {
	sandboxID := c.Param("id")

	var req CreateMockEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	endpoint, err := h.service.CreateMockEndpoint(c.Request.Context(), sandboxID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_MOCK_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, endpoint)
}

// @Summary Handle a mock endpoint request
// @Tags Sandbox
// @Accept json
// @Produce json
// @Param id path string true "Sandbox ID"
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{}
// @Router /sandbox/environments/{id}/mock/{endpointId} [post]
func (h *Handler) HandleMockRequest(c *gin.Context) {
	sandboxID := c.Param("id")
	endpointID := c.Param("endpointId")

	var body string
	if c.Request.Body != nil {
		rawBody, _ := c.GetRawData()
		body = string(rawBody)
	}

	headers := make(map[string]string)
	for key, vals := range c.Request.Header {
		if len(vals) > 0 {
			headers[key] = vals[0]
		}
	}

	captured, statusCode, respBody, err := h.service.HandleMockRequest(
		c.Request.Context(), sandboxID, endpointID,
		c.Request.Method, c.Request.URL.String(), headers, body,
	)
	if err != nil {
		httputil.InternalError(c, "MOCK_REQUEST_FAILED", err)
		return
	}

	c.JSON(statusCode, gin.H{
		"response_body": respBody,
		"request_id":    captured.ID,
		"latency_ms":    captured.LatencyMs,
		"failure":       captured.FailureInjected,
	})
}

// @Summary Get captured requests for a sandbox
// @Tags Sandbox
// @Produce json
// @Param id path string true "Sandbox ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string][]CapturedRequest
// @Router /sandbox/environments/{id}/requests [get]
func (h *Handler) GetCapturedRequests(c *gin.Context) {
	sandboxID := c.Param("id")
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	requests, err := h.service.GetCapturedRequests(c.Request.Context(), sandboxID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "GET_REQUESTS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests, "count": len(requests)})
}

// @Summary Inject chaos into a sandbox
// @Tags Sandbox
// @Accept json
// @Produce json
// @Param id path string true "Sandbox ID"
// @Param body body InjectChaosRequest true "Chaos configuration"
// @Success 200 {object} map[string]string
// @Router /sandbox/environments/{id}/chaos [post]
func (h *Handler) InjectChaos(c *gin.Context) {
	sandboxID := c.Param("id")

	var req InjectChaosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	if err := h.service.InjectChaos(c.Request.Context(), sandboxID, req.FailureType, req.Probability); err != nil {
		httputil.InternalError(c, "CHAOS_INJECTION_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "chaos injected", "failure_type": req.FailureType})
}

// @Summary Create a test scenario
// @Tags Sandbox
// @Accept json
// @Produce json
// @Param body body CreateTestScenarioRequest true "Test scenario configuration"
// @Success 201 {object} TestScenario
// @Router /sandbox/scenarios [post]
func (h *Handler) CreateTestScenario(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateTestScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	scenario, err := h.service.CreateTestScenario(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_SCENARIO_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, scenario)
}

// @Summary List test scenarios
// @Tags Sandbox
// @Produce json
// @Success 200 {object} map[string][]TestScenario
// @Router /sandbox/scenarios [get]
func (h *Handler) ListTestScenarios(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	scenarios, err := h.service.ListTestScenarios(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_SCENARIOS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"scenarios": scenarios, "count": len(scenarios)})
}

// @Summary Run a test scenario
// @Tags Sandbox
// @Produce json
// @Param id path string true "Scenario ID"
// @Success 200 {object} ScenarioResult
// @Router /sandbox/scenarios/{id}/run [post]
func (h *Handler) RunTestScenario(c *gin.Context) {
	scenarioID := c.Param("id")

	result, err := h.service.RunTestScenario(c.Request.Context(), scenarioID)
	if err != nil {
		httputil.InternalError(c, "RUN_SCENARIO_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Get scenario results
// @Tags Sandbox
// @Produce json
// @Param id path string true "Scenario ID"
// @Success 200 {object} ScenarioResult
// @Router /sandbox/scenarios/{id}/results [get]
func (h *Handler) GetScenarioResults(c *gin.Context) {
	scenarioID := c.Param("id")

	result, err := h.service.GetScenarioResults(c.Request.Context(), scenarioID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "RESULTS_NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

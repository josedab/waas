package edge

import (
	"errors"
	"github.com/josedab/waas/pkg/httputil"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for edge functions
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	edge := r.Group("/edge")
	{
		// Functions
		edge.POST("/functions", h.CreateFunction)
		edge.GET("/functions", h.ListFunctions)
		edge.GET("/functions/:id", h.GetFunction)
		edge.PUT("/functions/:id", h.UpdateFunction)
		edge.DELETE("/functions/:id", h.DeleteFunction)

		// Deployments
		edge.POST("/functions/:id/deploy", h.DeployFunction)
		edge.GET("/functions/:id/deployments", h.GetDeployments)

		// Invocations
		edge.POST("/functions/:id/invoke", h.InvokeFunction)

		// Logs and Metrics
		edge.GET("/functions/:id/logs", h.GetLogs)
		edge.GET("/functions/:id/metrics", h.GetMetrics)

		// Runtime providers
		edge.GET("/runtimes", h.ListRuntimes)
		edge.GET("/regions", h.ListRegions)

		// Edge Delivery Network
		edge.POST("/dispatch/config", h.CreateEdgeDispatchConfig)
		edge.GET("/dispatch/config", h.GetEdgeDispatchConfig)
		edge.POST("/dispatch", h.DispatchWebhookEdge)
		edge.POST("/dispatch/record", h.RecordEdgeDelivery)
		edge.GET("/dispatch/overview", h.GetEdgeNetworkOverview)
	}
}

// CreateFunction creates a new edge function
// @Summary Create edge function
// @Tags edge
// @Accept json
// @Produce json
// @Param request body CreateFunctionRequest true "Function request"
// @Success 201 {object} EdgeFunction
// @Failure 400 {object} ErrorResponse
// @Router /edge/functions [post]
func (h *Handler) CreateFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreateFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	fn, err := h.service.CreateFunction(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, fn)
}

// GetFunction retrieves an edge function
// @Summary Get edge function
// @Tags edge
// @Produce json
// @Param id path string true "Function ID"
// @Success 200 {object} EdgeFunction
// @Failure 404 {object} ErrorResponse
// @Router /edge/functions/{id} [get]
func (h *Handler) GetFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	fn, err := h.service.GetFunction(c.Request.Context(), tenantID, functionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, fn)
}

// UpdateFunction updates an edge function
// @Summary Update edge function
// @Tags edge
// @Accept json
// @Produce json
// @Param id path string true "Function ID"
// @Param request body UpdateFunctionRequest true "Update request"
// @Success 200 {object} EdgeFunction
// @Failure 400 {object} ErrorResponse
// @Router /edge/functions/{id} [put]
func (h *Handler) UpdateFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	var req UpdateFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	fn, err := h.service.UpdateFunction(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, fn)
}

// DeleteFunction deletes an edge function
// @Summary Delete edge function
// @Tags edge
// @Param id path string true "Function ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /edge/functions/{id} [delete]
func (h *Handler) DeleteFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	err := h.service.DeleteFunction(c.Request.Context(), tenantID, functionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListFunctions lists edge functions
// @Summary List edge functions
// @Tags edge
// @Produce json
// @Param status query string false "Filter by status"
// @Param runtime query string false "Filter by runtime"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListFunctionsResponse
// @Router /edge/functions [get]
func (h *Handler) ListFunctions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	filters := &FunctionFilters{
		Page:     1,
		PageSize: 20,
	}

	if status := c.Query("status"); status != "" {
		s := FunctionStatus(status)
		filters.Status = &s
	}
	if runtime := c.Query("runtime"); runtime != "" {
		r := RuntimeType(runtime)
		filters.Runtime = &r
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.ListFunctions(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeployFunction deploys a function to edge locations
// @Summary Deploy edge function
// @Tags edge
// @Produce json
// @Param id path string true "Function ID"
// @Success 200 {object} FunctionDeployment
// @Failure 400 {object} ErrorResponse
// @Router /edge/functions/{id}/deploy [post]
func (h *Handler) DeployFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	deployment, err := h.service.DeployFunction(c.Request.Context(), tenantID, functionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// InvokeFunction invokes an edge function
// @Summary Invoke edge function
// @Tags edge
// @Accept json
// @Produce json
// @Param id path string true "Function ID"
// @Param request body InvokeFunctionRequest true "Invocation request"
// @Success 200 {object} InvokeFunctionResponse
// @Failure 400 {object} ErrorResponse
// @Router /edge/functions/{id}/invoke [post]
func (h *Handler) InvokeFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	var req InvokeFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow raw JSON body as input
		body, _ := c.GetRawData()
		req.Input = json.RawMessage(body)
	}

	response, err := h.service.InvokeFunction(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetLogs retrieves function logs
// @Summary Get function logs
// @Tags edge
// @Produce json
// @Param id path string true "Function ID"
// @Param since query string false "Logs since timestamp"
// @Param limit query int false "Maximum logs to return"
// @Success 200 {array} LogEntry
// @Router /edge/functions/{id}/logs [get]
func (h *Handler) GetLogs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	since := time.Now().Add(-1 * time.Hour)
	if sinceStr := c.Query("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	limit := 100
	if l, _ := strconv.Atoi(c.Query("limit")); l > 0 && l <= 1000 {
		limit = l
	}

	logs, err := h.service.GetLogs(c.Request.Context(), tenantID, functionID, since, limit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetMetrics retrieves function metrics
// @Summary Get function metrics
// @Tags edge
// @Produce json
// @Param id path string true "Function ID"
// @Param period query string false "Metrics period (hour, day, week)"
// @Success 200 {object} FunctionMetrics
// @Router /edge/functions/{id}/metrics [get]
func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	period := c.DefaultQuery("period", "hour")

	metrics, err := h.service.GetMetrics(c.Request.Context(), tenantID, functionID, period)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetDeployments retrieves function deployments
// @Summary Get function deployments
// @Tags edge
// @Produce json
// @Param id path string true "Function ID"
// @Param limit query int false "Maximum deployments to return"
// @Success 200 {array} FunctionDeployment
// @Router /edge/functions/{id}/deployments [get]
func (h *Handler) GetDeployments(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	functionID := c.Param("id")

	limit := 10
	if l, _ := strconv.Atoi(c.Query("limit")); l > 0 && l <= 100 {
		limit = l
	}

	deployments, err := h.service.GetDeployments(c.Request.Context(), tenantID, functionID, limit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, deployments)
}

// ListRuntimes lists available runtimes
// @Summary List available runtimes
// @Tags edge
// @Produce json
// @Success 200 {array} RuntimeInfo
// @Router /edge/runtimes [get]
func (h *Handler) ListRuntimes(c *gin.Context) {
	runtimes := []RuntimeInfo{
		{
			Type:        RuntimeCloudflare,
			Name:        "Cloudflare Workers",
			Description: "Deploy to Cloudflare's global edge network",
			Features:    []string{"KV Storage", "Durable Objects", "R2 Storage", "Queues"},
			Regions:     []EdgeRegion{RegionUSEast, RegionUSWest, RegionEUWest, RegionEUCentral, RegionAPSoutheast, RegionAPNortheast},
		},
		{
			Type:        RuntimeDeno,
			Name:        "Deno Deploy",
			Description: "Deploy to Deno's globally distributed edge runtime",
			Features:    []string{"TypeScript Native", "Web Standard APIs", "BroadcastChannel"},
			Regions:     []EdgeRegion{RegionUSEast, RegionUSWest, RegionEUWest, RegionAPSoutheast},
		},
		{
			Type:        RuntimeFastly,
			Name:        "Fastly Compute",
			Description: "Deploy to Fastly's high-performance edge compute platform",
			Features:    []string{"WASM Runtime", "Edge Dictionary", "Backend Origins"},
			Regions:     []EdgeRegion{RegionUSEast, RegionUSWest, RegionEUWest, RegionEUCentral},
		},
		{
			Type:        RuntimeVercel,
			Name:        "Vercel Edge Functions",
			Description: "Deploy to Vercel's edge infrastructure",
			Features:    []string{"Streaming", "Edge Config", "Middleware"},
			Regions:     []EdgeRegion{RegionUSEast, RegionUSWest, RegionEUWest, RegionAPNortheast},
		},
		{
			Type:        RuntimeLambdaEdge,
			Name:        "AWS Lambda@Edge",
			Description: "Deploy to AWS CloudFront edge locations",
			Features:    []string{"CloudFront Integration", "Origin Request/Response", "Viewer Request/Response"},
			Regions:     []EdgeRegion{RegionUSEast}, // Lambda@Edge deploys from us-east-1
		},
		{
			Type:        RuntimeLocal,
			Name:        "Local Runtime",
			Description: "Execute locally for testing and development",
			Features:    []string{"No deployment required", "Fast iteration", "Debug support"},
			Regions:     AllEdgeRegions(),
		},
	}

	c.JSON(http.StatusOK, runtimes)
}

// RuntimeInfo describes a runtime
type RuntimeInfo struct {
	Type        RuntimeType  `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Features    []string     `json:"features"`
	Regions     []EdgeRegion `json:"regions"`
}

// ListRegions lists available edge regions
// @Summary List available regions
// @Tags edge
// @Produce json
// @Success 200 {array} RegionInfo
// @Router /edge/regions [get]
func (h *Handler) ListRegions(c *gin.Context) {
	regions := []RegionInfo{
		{Region: RegionUSEast, Name: "US East", Location: "Virginia, USA", Latency: "Low for North America"},
		{Region: RegionUSWest, Name: "US West", Location: "California, USA", Latency: "Low for West Coast"},
		{Region: RegionEUWest, Name: "EU West", Location: "Ireland", Latency: "Low for Western Europe"},
		{Region: RegionEUCentral, Name: "EU Central", Location: "Frankfurt, Germany", Latency: "Low for Central Europe"},
		{Region: RegionAPSoutheast, Name: "Asia Pacific Southeast", Location: "Singapore", Latency: "Low for Southeast Asia"},
		{Region: RegionAPNortheast, Name: "Asia Pacific Northeast", Location: "Tokyo, Japan", Latency: "Low for Northeast Asia"},
		{Region: RegionSAEast, Name: "South America East", Location: "São Paulo, Brazil", Latency: "Low for South America"},
		{Region: RegionMECentral, Name: "Middle East Central", Location: "Dubai, UAE", Latency: "Low for Middle East"},
		{Region: RegionAFSouth, Name: "Africa South", Location: "Cape Town, South Africa", Latency: "Low for Africa"},
		{Region: RegionOCEast, Name: "Oceania East", Location: "Sydney, Australia", Latency: "Low for Australia/NZ"},
	}

	c.JSON(http.StatusOK, regions)
}

// RegionInfo describes a region
type RegionInfo struct {
	Region   EdgeRegion `json:"region"`
	Name     string     `json:"name"`
	Location string     `json:"location"`
	Latency  string     `json:"latency"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

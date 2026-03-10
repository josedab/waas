package webhooktest

import (
"context"
"fmt"
"net/http"
"time"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
)

// TestSuiteRecord represents a stored test suite.
type TestSuiteRecord struct {
ID          string        `json:"id" db:"id"`
TenantID    string        `json:"tenant_id" db:"tenant_id"`
Name        string        `json:"name" db:"name"`
Description string        `json:"description,omitempty" db:"description"`
TestCount   int           `json:"test_count"`
Config      SuiteRunConfig `json:"config"`
Status      string        `json:"status" db:"status"`
LastRunAt   *time.Time    `json:"last_run_at,omitempty" db:"last_run_at"`
CreatedAt   time.Time     `json:"created_at" db:"created_at"`
UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

// RunSuiteRequest represents a request to run a test suite.
type RunSuiteRequest struct {
Suite  TestSuite      `json:"suite" binding:"required"`
Config SuiteRunConfig `json:"config,omitempty"`
}

// TestService provides test suite management.
type TestService struct{}

// NewTestService creates a new test service.
func NewTestService() *TestService { return &TestService{} }

// RunSuite executes a test suite.
func (s *TestService) RunSuite(_ context.Context, _ string, req *RunSuiteRequest) (*SuiteRunResult, error) {
runner := NewSuiteRunner(&req.Suite, req.Config)
return runner.Run(), nil
}

// TestHandler provides HTTP handlers for the testing framework.
type TestHandler struct {
service *TestService
}

// NewTestHandler creates a new test handler.
func NewTestHandler(service *TestService) *TestHandler {
return &TestHandler{service: service}
}

// RegisterTestRoutes registers testing framework routes.
func (h *TestHandler) RegisterTestRoutes(router *gin.RouterGroup) {
tests := router.Group("/webhook-tests")
{
tests.POST("/run", h.RunSuite)
tests.POST("/validate", h.ValidateSuite)
tests.GET("/simulations", h.ListSimulations)
}
}

func (h *TestHandler) RunSuite(c *gin.Context) {
tenantID := c.GetString("tenant_id")
if tenantID == "" {
c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
return
}
var req RunSuiteRequest
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
return
}
result, err := h.service.RunSuite(c.Request.Context(), tenantID, &req)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
return
}
if req.Config.CI || req.Config.OutputFormat == "junit" {
xml, _ := result.ToJUnitXML()
c.JSON(http.StatusOK, gin.H{"result": result, "junit_xml": xml})
return
}
c.JSON(http.StatusOK, result)
}

func (h *TestHandler) ValidateSuite(c *gin.Context) {
var suite TestSuite
if err := c.ShouldBindJSON(&suite); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"valid": false, "errors": []string{err.Error()}})
return
}
errors := validateSuite(&suite)
c.JSON(http.StatusOK, gin.H{"valid": len(errors) == 0, "errors": errors, "test_count": len(suite.Tests)})
}

func (h *TestHandler) ListSimulations(c *gin.Context) {
sims := []FailureSimulation{
{Name: "Connection Timeout", Type: "timeout", Duration: 30000},
{Name: "Server Error (500)", Type: "5xx", StatusCode: 500},
{Name: "Rate Limited (429)", Type: "rate_limit", StatusCode: 429},
{Name: "TLS Certificate Error", Type: "tls_error"},
{Name: "Connection Refused", Type: "connection_refused"},
}
c.JSON(http.StatusOK, gin.H{"simulations": sims})
}

func validateSuite(suite *TestSuite) []string {
var errs []string
if suite.Name == "" {
errs = append(errs, "suite name is required")
}
if len(suite.Tests) == 0 {
errs = append(errs, "at least one test case is required")
}
for i, tc := range suite.Tests {
if tc.Name == "" {
errs = append(errs, fmt.Sprintf("test[%d]: name is required", i))
}
}
return errs
}

var _ = uuid.New

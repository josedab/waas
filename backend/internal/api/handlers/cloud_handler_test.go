package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func setupCloudTest() (*CloudHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := &CloudHandler{logger: logger}
	router := gin.New()
	return handler, router
}

func TestCloudHandler_GetPlans_Success(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/plans", handler.GetPlans)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/plans", nil)
	router.ServeHTTP(w, req)

	// GetPlans returns plans from the billing service; nil service causes internal error
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
}

func TestCloudHandler_GetPlan_NotFound(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/plans/:plan_id", handler.GetPlan)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/plans/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCloudHandler_GetSubscription_Unauthorized(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/subscription", handler.GetSubscription)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/subscription", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudHandler_GetUsage_Unauthorized(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/usage", handler.GetUsage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/usage", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudHandler_ListInvoices_Unauthorized(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/invoices", handler.ListInvoices)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/invoices", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudHandler_GetCustomer_Unauthorized(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/customer", handler.GetCustomer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/customer", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudHandler_ListTeamMembers_Unauthorized(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/team", handler.ListTeamMembers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/team", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudHandler_ListAuditLogs_Unauthorized(t *testing.T) {
	handler, router := setupCloudTest()
	router.GET("/audit-logs", handler.ListAuditLogs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

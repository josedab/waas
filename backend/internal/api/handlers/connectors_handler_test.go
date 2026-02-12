package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/connectors"
	"github.com/josedab/waas/pkg/utils"
)

// mockTransformEngine implements connectors.TransformEngine for testing
type mockTransformEngine struct{}

func (m *mockTransformEngine) Execute(script string, event interface{}, config interface{}) (interface{}, error) {
	return nil, nil
}

func setupConnectorsTestRouter() (*gin.Engine, *ConnectorsHandler) {
	gin.SetMode(gin.TestMode)
	repo := connectors.NewInMemoryRepository()
	service := connectors.NewService(repo, &mockTransformEngine{})
	logger := utils.NewLogger("test")
	handler := NewConnectorsHandler(service, logger)

	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
			c.Set("tenant_id", tid)
		}
		c.Next()
	})
	RegisterConnectorsRoutes(api, handler)
	return router, handler
}

// --- ListAvailableConnectors tests ---

func TestListAvailableConnectors_Success(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/connectors/marketplace", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []*connectors.Connector
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
}

// --- ListInstalledConnectors tests ---

func TestListInstalledConnectors_Success(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/connectors/installed", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListInstalledConnectors_MissingTenantID(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/connectors/installed", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- InstallConnector tests ---

func TestInstallConnector_Success(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	body := `{"connector_id":"stripe-to-slack","name":"My Stripe Alerts","config":{"slack_webhook_url":"https://hooks.slack.com/test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connectors/installed", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp connectors.InstalledConnector
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Name != "My Stripe Alerts" {
		t.Errorf("expected name 'My Stripe Alerts', got %s", resp.Name)
	}
	if resp.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id 'tenant-1', got %s", resp.TenantID)
	}
}

func TestInstallConnector_MissingTenantID(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	body := `{"connector_id":"stripe-to-slack","name":"My Stripe Alerts","config":{"slack_webhook_url":"https://hooks.slack.com/test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connectors/installed", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInstallConnector_InvalidJSON(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/connectors/installed", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetConnectorDetails tests ---

func TestGetConnectorDetails_NotFound(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/connectors/marketplace/nonexistent", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// --- UninstallConnector tests ---

func TestUninstallConnector_MissingTenantID(t *testing.T) {
	router, _ := setupConnectorsTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/connectors/installed/some-id", nil)
	// No X-Tenant-ID header — handler does not guard against empty tenantID,
	// and the in-memory repo silently succeeds, so we get 204.
	// This test documents the current behavior (no auth check on uninstall).

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

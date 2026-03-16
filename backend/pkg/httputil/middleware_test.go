package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		id, exists := c.Get("request_id")
		if !exists || id == "" {
			t.Error("request_id should be set in context")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get(HeaderRequestID) == "" {
		t.Error("X-Request-ID header should be set in response")
	}
}

func TestRequestIDMiddleware_PreservesClientID(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	clientID := "client-provided-id-123"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(HeaderRequestID, clientID)
	router.ServeHTTP(w, req)

	got := w.Header().Get(HeaderRequestID)
	if got != clientID {
		t.Errorf("expected preserved client ID %q, got %q", clientID, got)
	}
}

func TestAPIVersionMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(APIVersionMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Header().Get(HeaderAPIVersion) == "" {
		t.Error("X-API-Version header should be set")
	}
}

type mockLogger struct {
	lastMsg    string
	lastFields map[string]interface{}
}

func (m *mockLogger) Info(msg string, fields map[string]interface{}) {
	m.lastMsg = msg
	m.lastFields = fields
}

func TestRequestLoggerMiddleware(t *testing.T) {
	logger := &mockLogger{}
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(RequestLoggerMiddleware(logger))
	router.GET("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test?foo=bar", nil)
	router.ServeHTTP(w, req)

	if logger.lastMsg != "request" {
		t.Errorf("expected log msg 'request', got %q", logger.lastMsg)
	}
	if logger.lastFields["status"] != 200 {
		t.Errorf("expected status 200, got %v", logger.lastFields["status"])
	}
	if logger.lastFields["method"] != "GET" {
		t.Errorf("expected method GET, got %v", logger.lastFields["method"])
	}
	if logger.lastFields["path"] != "/api/test?foo=bar" {
		t.Errorf("expected path /api/test?foo=bar, got %v", logger.lastFields["path"])
	}
	if _, ok := logger.lastFields["request_id"]; !ok {
		t.Error("expected request_id in log fields")
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestServerRouter(t *testing.T) {
	t.Run("Router returns the gin engine", func(t *testing.T) {
		router := gin.New()
		s := &Server{router: router}

		got := s.Router()
		require.NotNil(t, got)
		assert.Equal(t, router, got, "Router() should return the same engine")
	})
}

func TestServerStruct(t *testing.T) {
	t.Run("zero-value server has nil fields", func(t *testing.T) {
		s := &Server{}
		assert.Nil(t, s.router)
		assert.Nil(t, s.db)
		assert.Nil(t, s.sqlxDB)
		assert.Nil(t, s.redisClient)
		assert.Nil(t, s.logger)
		assert.Nil(t, s.config)
	})

	t.Run("server with router responds to registered routes", func(t *testing.T) {
		router := gin.New()
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		s := &Server{router: router}

		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/health", nil)
		require.NoError(t, err)

		s.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("unregistered route returns 404", func(t *testing.T) {
		router := gin.New()
		s := &Server{router: router}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
		s.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestServerRouteGroups(t *testing.T) {
	// Verify that a minimal server with a Gin engine can register route groups
	// and that protected groups apply middleware correctly.
	router := gin.New()

	// Simulate a simplified version of setupRoutes with a public health endpoint
	// and a protected group that requires a header.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	protected := router.Group("/api/v1")
	protected.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	})
	protected.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "test"})
	})

	s := &Server{router: router}

	t.Run("health endpoint is public", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		s.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("protected endpoint rejects unauthenticated requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		s.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("protected endpoint accepts authenticated requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		s.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

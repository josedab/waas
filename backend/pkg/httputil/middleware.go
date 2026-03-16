package httputil

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// HeaderRequestID is the header name for request IDs.
	HeaderRequestID = "X-Request-ID"
	// HeaderAPIVersion is the header name for the API version.
	HeaderAPIVersion = "X-API-Version"

	apiVersion = "2025-06-01"
)

// RequestIDMiddleware ensures every request has a unique X-Request-ID header.
// If the client already sent one it is preserved; otherwise a new UUID is
// generated. The value is stored in the Gin context under "request_id" and
// echoed back in the response.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("request_id", id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// APIVersionMiddleware attaches the current API version to every response.
func APIVersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(HeaderAPIVersion, apiVersion)
		c.Next()
	}
}

// RequestLoggerMiddleware logs method, path, status, latency and request ID
// for every request. It uses the structured logger passed in.
type RequestLogger interface {
	Info(msg string, fields map[string]interface{})
}

func RequestLoggerMiddleware(logger RequestLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := map[string]interface{}{
			"status":     status,
			"method":     c.Request.Method,
			"path":       path,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
		}
		if id, ok := c.Get("request_id"); ok {
			fields["request_id"] = id
		}

		logger.Info("request", fields)
	}
}

package idempotency

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// IdempotencyKeyHeader is the HTTP header for idempotency keys
	IdempotencyKeyHeader = "Idempotency-Key"
	
	// ContextKeyIdempotency is the context key for idempotency data
	ContextKeyIdempotency = "idempotency"
)

// Middleware returns a Gin middleware for idempotency handling
func Middleware(service *Service, getTenantID func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to non-GET methods
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
		if idempotencyKey == "" {
			c.Next()
			return
		}

		tenantID := getTenantID(c)
		if tenantID == "" {
			c.Next()
			return
		}

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Check idempotency
		result, err := service.Check(c.Request.Context(), tenantID, idempotencyKey, requestBody)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{
				"code":    "IDEMPOTENCY_ERROR",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// If request is still being processed
		if result.IsProcessing {
			c.JSON(http.StatusConflict, gin.H{
				"code":    "REQUEST_IN_PROGRESS",
				"message": "A request with this idempotency key is still being processed",
			})
			c.Abort()
			return
		}

		// If we have a cached response
		if !result.IsNew && result.CachedResponse != nil {
			c.Header("Idempotent-Replayed", "true")
			c.Data(result.CachedStatusCode, "application/json", result.CachedResponse)
			c.Abort()
			return
		}

		// Store idempotency data in context
		c.Set(ContextKeyIdempotency, &MiddlewareData{
			Key:      idempotencyKey,
			TenantID: tenantID,
			Service:  service,
		})

		// Use a custom response writer to capture the response
		writer := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// After handler completes, save the response
		if idempotencyKey != "" && writer.body.Len() > 0 {
			err := service.Complete(
				c.Request.Context(),
				tenantID,
				idempotencyKey,
				writer.status,
				writer.body.Bytes(),
			)
			if err != nil {
				// Log error but don't fail the request
				// The response has already been sent
			}
		}
	}
}

// MiddlewareData holds idempotency data for the request
type MiddlewareData struct {
	Key      string
	TenantID string
	Service  *Service
}

// GetFromContext retrieves idempotency data from the context
func GetFromContext(c *gin.Context) *MiddlewareData {
	data, exists := c.Get(ContextKeyIdempotency)
	if !exists {
		return nil
	}
	return data.(*MiddlewareData)
}

// AbortIdempotency aborts the idempotency key if an error occurs
func AbortIdempotency(c *gin.Context) {
	data := GetFromContext(c)
	if data != nil {
		data.Service.Abort(c.Request.Context(), data.TenantID, data.Key)
	}
}

// responseCapture captures the response body
type responseCapture struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseCapture) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// ManualComplete allows manual completion of idempotency for async operations
func ManualComplete(c *gin.Context, statusCode int, response interface{}) error {
	data := GetFromContext(c)
	if data == nil {
		return nil
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return data.Service.Complete(c.Request.Context(), data.TenantID, data.Key, statusCode, respBytes)
}

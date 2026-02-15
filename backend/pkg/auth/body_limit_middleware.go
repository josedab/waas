package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMaxBodySize is 1 MB, suitable for most webhook payloads.
	DefaultMaxBodySize int64 = 1 << 20
)

// MaxBodySize returns middleware that limits request body size to prevent
// memory exhaustion from arbitrarily large payloads. When a handler reads
// beyond the limit, http.MaxBytesReader returns an error that Gin translates
// to a 413 or binding error.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"code":    "REQUEST_BODY_TOO_LARGE",
					"message": "Request body exceeds maximum allowed size",
				},
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

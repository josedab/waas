package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var errorLogger = utils.NewLogger("api-handlers")

// InternalError logs the full error server-side and returns a generic error
// message with a correlation ID to the client. This prevents leaking internal
// details (stack traces, SQL errors, etc.) to API consumers.
func InternalError(c *gin.Context, code string, err error) {
	correlationID := uuid.New().String()
	errorLogger.ErrorWithCorrelation("Internal error", correlationID, map[string]interface{}{
		"code":  code,
		"error": err.Error(),
	})
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    code,
		Message: "An internal error occurred. Correlation ID: " + correlationID,
	})
}

// RequireTenantID extracts and validates the tenant_id from the Gin context.
// Returns the tenant UUID and true on success. On failure it writes a 401
// response and returns uuid.Nil, false so the caller can simply return.
func RequireTenantID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	tid, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant context"})
		return uuid.Nil, false
	}
	return tid, true
}

// InternalErrorGeneric is like InternalError but uses gin.H for handlers that
// don't use the ErrorResponse struct.
func InternalErrorGeneric(c *gin.Context, err error) {
	correlationID := uuid.New().String()
	errorLogger.ErrorWithCorrelation("Internal error", correlationID, map[string]interface{}{
		"error": err.Error(),
	})
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "An internal error occurred. Correlation ID: " + correlationID,
	})
}

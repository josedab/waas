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

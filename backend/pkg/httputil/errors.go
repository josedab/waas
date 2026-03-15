package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var errorLogger = utils.NewLogger("httputil")

// APIErrorResponse is the standard error response format.
type APIErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// InternalErrorGeneric logs the full error server-side and returns a generic
// error message with a correlation ID. This prevents leaking internal details
// (stack traces, SQL errors, etc.) to API consumers.
func InternalErrorGeneric(c *gin.Context, err error) {
	correlationID := uuid.New().String()
	errorLogger.ErrorWithCorrelation("Internal error", correlationID, map[string]interface{}{
		"error": err.Error(),
		"path":  c.Request.URL.Path,
	})
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "An internal error occurred. Correlation ID: " + correlationID,
	})
}

// InternalError logs the full error server-side with a structured code and
// returns a generic message with a correlation ID.
func InternalError(c *gin.Context, code string, err error) {
	correlationID := uuid.New().String()
	errorLogger.ErrorWithCorrelation("Internal error", correlationID, map[string]interface{}{
		"code":  code,
		"error": err.Error(),
		"path":  c.Request.URL.Path,
	})
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": APIErrorResponse{Code: code, Message: "An internal error occurred. Correlation ID: " + correlationID},
	})
}

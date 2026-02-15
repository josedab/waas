package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InternalError logs the full error server-side and returns a generic error
// message with a correlation ID to the client. This prevents leaking internal
// details (stack traces, SQL errors, etc.) to API consumers.
func InternalError(c *gin.Context, code string, err error) {
	correlationID := uuid.New().String()
	log.Printf("[error] correlation_id=%s code=%s err=%v", correlationID, code, err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    code,
		Message: "An internal error occurred. Correlation ID: " + correlationID,
	})
}

// InternalErrorGeneric is like InternalError but uses gin.H for handlers that
// don't use the ErrorResponse struct.
func InternalErrorGeneric(c *gin.Context, err error) {
	correlationID := uuid.New().String()
	log.Printf("[error] correlation_id=%s err=%v", correlationID, err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "An internal error occurred. Correlation ID: " + correlationID,
	})
}

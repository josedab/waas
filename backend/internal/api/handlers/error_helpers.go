package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	webhookerrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/utils"
)

var errorLogger = utils.NewLogger("api-handlers")

// ErrorResponse represents a lightweight error response with a code and message.
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// RespondWithError writes a structured error response using a WebhookError.
// It derives the HTTP status from the error and renders the standard shape.
func RespondWithError(c *gin.Context, err *webhookerrors.WebhookError) {
	c.JSON(err.GetHTTPStatus(), err)
}

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
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Missing authentication"})
		return uuid.Nil, false
	}
	tid, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid tenant context"})
		return uuid.Nil, false
	}
	return tid, true
}

// InternalErrorGeneric is like InternalError but accepts a bare error without
// a domain code. Produces the same ErrorResponse shape for consistency.
func InternalErrorGeneric(c *gin.Context, err error) {
	correlationID := uuid.New().String()
	errorLogger.ErrorWithCorrelation("Internal error", correlationID, map[string]interface{}{
		"error": err.Error(),
	})
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred. Correlation ID: " + correlationID,
	})
}

// ParseQueryInt parses an integer query parameter. If the raw value cannot be
// parsed, it logs a debug warning and returns defaultVal.
func ParseQueryInt(c *gin.Context, param string, defaultVal int) int {
	raw := c.DefaultQuery(param, strconv.Itoa(defaultVal))
	val, err := strconv.Atoi(raw)
	if err != nil {
		errorLogger.Debug("Invalid query parameter, using default", map[string]interface{}{
			"param":   param,
			"value":   raw,
			"default": defaultVal,
		})
		return defaultVal
	}
	return val
}

package portal

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

const (
	// PortalTokenKey is the context key for the validated portal token
	PortalTokenKey = "portal_token"
	// PortalTenantIDKey is the context key for the tenant ID from the portal token
	PortalTenantIDKey = "portal_tenant_id"
)

// PortalTokenMiddleware validates portal embed tokens from the Authorization header
func (h *Handler) PortalTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "missing authorization header"}})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid authorization format"}})
			c.Abort()
			return
		}

		tokenValue := parts[1]
		if !strings.HasPrefix(tokenValue, "wpt_") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid portal token format"}})
			c.Abort()
			return
		}

		token, err := h.service.ValidatePortalToken(c.Request.Context(), tokenValue)
		if err != nil {
			httputil.InternalError(c, "UNAUTHORIZED", err)
			c.Abort()
			return
		}

		c.Set(PortalTokenKey, token)
		c.Set(PortalTenantIDKey, token.TenantID)
		c.Next()
	}
}

// RequireScope creates middleware that checks for a specific scope on the portal token
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenVal, exists := c.Get(PortalTokenKey)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "no portal token in context"}})
			c.Abort()
			return
		}

		embedToken, ok := tokenVal.(*EmbedToken)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "invalid token in context"}})
			c.Abort()
			return
		}

		if !HasScope(embedToken, scope) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "INSUFFICIENT_SCOPE", "message": "token missing required scope: " + scope}})
			c.Abort()
			return
		}

		c.Next()
	}
}

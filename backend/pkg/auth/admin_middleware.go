package auth

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware restricts access to admin-only endpoints.
// It checks if the authenticated tenant is in the configured admin list
// (ADMIN_TENANT_IDS env var, comma-separated) or has the "admin" subscription tier.
type AdminMiddleware struct {
	adminTenantIDs map[string]bool
}

// NewAdminMiddleware creates a new AdminMiddleware.
// Admin tenant IDs are loaded from the ADMIN_TENANT_IDS environment variable.
func NewAdminMiddleware() *AdminMiddleware {
	ids := make(map[string]bool)
	if raw := os.Getenv("ADMIN_TENANT_IDS"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				ids[trimmed] = true
			}
		}
	}
	return &AdminMiddleware{adminTenantIDs: ids}
}

// RequireAdmin returns middleware that rejects non-admin tenants with 403.
func (am *AdminMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant, exists := GetTenantFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, authErrorResponse{Code: "NO_TENANT_CONTEXT", Message: "No authenticated tenant found"})
			c.Abort()
			return
		}

		// Allow if tenant ID is in the admin list
		if am.adminTenantIDs[tenant.ID.String()] {
			c.Next()
			return
		}

		// Allow if tenant has admin subscription tier
		if tenant.SubscriptionTier == "admin" {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, authErrorResponse{Code: "ADMIN_ACCESS_REQUIRED", Message: "This endpoint requires admin privileges"})
		c.Abort()
	}
}

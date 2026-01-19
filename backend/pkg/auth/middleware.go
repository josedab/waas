package auth

import (
	"context"
	"net/http"
	"strings"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	AuthHeaderName = "Authorization"
	BearerPrefix   = "Bearer "
	TenantKey      = "tenant"
	TenantIDKey    = "tenant_id"
)

type AuthMiddleware struct {
	tenantRepo repository.TenantRepository
}

func NewAuthMiddleware(tenantRepo repository.TenantRepository) *AuthMiddleware {
	return &AuthMiddleware{
		tenantRepo: tenantRepo,
	}
}

// RequireAuth validates API key and sets tenant context
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthHeaderName)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "MISSING_AUTH_HEADER",
					"message": "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		// Extract API key from Bearer token
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_AUTH_FORMAT",
					"message": "Authorization header must use Bearer format",
				},
			})
			c.Abort()
			return
		}

		apiKey := strings.TrimPrefix(authHeader, BearerPrefix)
		if !IsValidAPIKeyFormat(apiKey) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_API_KEY_FORMAT",
					"message": "API key format is invalid",
				},
			})
			c.Abort()
			return
		}

		// Find tenant by API key
		ctx := context.Background()
		tenant, err := am.tenantRepo.FindByAPIKey(ctx, apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_API_KEY",
					"message": "API key is invalid or expired",
				},
			})
			c.Abort()
			return
		}

		// Set tenant context
		c.Set(TenantKey, tenant)
		c.Set(TenantIDKey, tenant.ID.String())
		
		c.Next()
	}
}

// GetTenantFromContext retrieves the authenticated tenant from Gin context
func GetTenantFromContext(c *gin.Context) (*models.Tenant, bool) {
	tenant, exists := c.Get(TenantKey)
	if !exists {
		return nil, false
	}
	
	t, ok := tenant.(*models.Tenant)
	return t, ok
}

// GetTenantIDFromContext retrieves the authenticated tenant ID from Gin context
func GetTenantIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	tenantIDStr, exists := c.Get(TenantIDKey)
	if !exists {
		return uuid.Nil, false
	}
	
	tenantIDString, ok := tenantIDStr.(string)
	if !ok {
		return uuid.Nil, false
	}
	
	tenantID, err := uuid.Parse(tenantIDString)
	if err != nil {
		return uuid.Nil, false
	}
	
	return tenantID, true
}
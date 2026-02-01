package security

import (
	"net/http"
	"strings"
	"github.com/josedab/waas/pkg/auth"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecureAuthMiddleware extends the basic auth middleware with security features
type SecureAuthMiddleware struct {
	tenantRepo  repository.TenantRepository
	auditLogger *AuditLogger
}

// NewSecureAuthMiddleware creates a new secure authentication middleware
func NewSecureAuthMiddleware(tenantRepo repository.TenantRepository, auditLogger *AuditLogger) *SecureAuthMiddleware {
	return &SecureAuthMiddleware{
		tenantRepo:  tenantRepo,
		auditLogger: auditLogger,
	}
}

// RequireAuth validates API key and logs authentication events
func (sam *SecureAuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(auth.AuthHeaderName)
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		if authHeader == "" {
			sam.logAuthFailure(c, nil, "missing_auth_header", "Authorization header is required", clientIP, userAgent)
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
		if !strings.HasPrefix(authHeader, auth.BearerPrefix) {
			sam.logAuthFailure(c, nil, "invalid_auth_format", "Authorization header must use Bearer format", clientIP, userAgent)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_AUTH_FORMAT",
					"message": "Authorization header must use Bearer format",
				},
			})
			c.Abort()
			return
		}

		apiKey := strings.TrimPrefix(authHeader, auth.BearerPrefix)
		if !auth.IsValidAPIKeyFormat(apiKey) {
			sam.logAuthFailure(c, nil, "invalid_api_key_format", "API key format is invalid", clientIP, userAgent)
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
		tenant, err := sam.tenantRepo.FindByAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			sam.logAuthFailure(c, nil, "invalid_api_key", "API key is invalid or expired", clientIP, userAgent)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_API_KEY",
					"message": "API key is invalid or expired",
				},
			})
			c.Abort()
			return
		}

		// Log successful authentication
		sam.logAuthSuccess(c, tenant, clientIP, userAgent)

		// Set tenant context
		c.Set(auth.TenantKey, tenant)
		c.Set(auth.TenantIDKey, tenant.ID.String())
		
		c.Next()
	}
}

// RequireTenantAccess ensures the authenticated tenant can access the specified resource
func (sam *SecureAuthMiddleware) RequireTenantAccess(resourceTenantID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant, exists := auth.GetTenantFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "NO_TENANT_CONTEXT",
					"message": "No authenticated tenant found",
				},
			})
			c.Abort()
			return
		}

		if tenant.ID != resourceTenantID {
			sam.logAccessViolation(c, tenant, resourceTenantID)
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "TENANT_ACCESS_DENIED",
					"message": "Access denied to resource",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityHeaders adds security-related HTTP headers
func (sam *SecureAuthMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Enforce HTTPS in production
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		// Prevent caching of sensitive responses
		if strings.Contains(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// SuspiciousActivityDetection detects and blocks suspicious requests
func (sam *SecureAuthMiddleware) SuspiciousActivityDetection() gin.HandlerFunc {
	suspiciousUserAgents := []string{
		"sqlmap", "nikto", "nmap", "masscan", "dirb", "gobuster",
		"wfuzz", "burp", "zap", "w3af", "skipfish",
	}

	return func(c *gin.Context) {
		userAgent := strings.ToLower(c.GetHeader("User-Agent"))
		
		// Check for suspicious user agents
		for _, suspicious := range suspiciousUserAgents {
			if strings.Contains(userAgent, suspicious) {
				sam.logSuspiciousActivity(c, "suspicious_user_agent", map[string]interface{}{
					"user_agent": c.GetHeader("User-Agent"),
					"path":       c.Request.URL.Path,
				})
				
				c.JSON(http.StatusForbidden, gin.H{
					"error": gin.H{
						"code":    "SUSPICIOUS_ACTIVITY",
						"message": "Request blocked due to suspicious activity",
					},
				})
				c.Abort()
				return
			}
		}

		// Check for suspicious request patterns
		path := strings.ToLower(c.Request.URL.Path)
		suspiciousPatterns := []string{
			"../", "..\\", "/etc/passwd", "/proc/", "cmd=", "exec(",
			"<script", "javascript:", "vbscript:", "onload=", "onerror=",
		}

		for _, pattern := range suspiciousPatterns {
			if strings.Contains(path, pattern) || strings.Contains(c.Request.URL.RawQuery, pattern) {
				sam.logSuspiciousActivity(c, "suspicious_request_pattern", map[string]interface{}{
					"pattern": pattern,
					"path":    c.Request.URL.Path,
					"query":   c.Request.URL.RawQuery,
				})
				
				c.JSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"code":    "INVALID_REQUEST",
						"message": "Invalid request format",
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// Helper methods for audit logging
func (sam *SecureAuthMiddleware) logAuthSuccess(c *gin.Context, tenant *models.Tenant, clientIP, userAgent string) {
	if sam.auditLogger != nil {
		sam.auditLogger.LogAuthAction(
			c.Request.Context(),
			&tenant.ID,
			nil, // User ID not implemented yet
			ActionAuthAPIKeyUsed,
			map[string]interface{}{
				"tenant_name": tenant.Name,
				"path":        c.Request.URL.Path,
				"method":      c.Request.Method,
			},
			clientIP,
			userAgent,
			true,
			nil,
		)
	}
}

func (sam *SecureAuthMiddleware) logAuthFailure(c *gin.Context, tenantID *uuid.UUID, reason, message, clientIP, userAgent string) {
	if sam.auditLogger != nil {
		sam.auditLogger.LogAuthAction(
			c.Request.Context(),
			tenantID,
			nil,
			ActionAuthAPIKeyInvalid,
			map[string]interface{}{
				"reason": reason,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			},
			clientIP,
			userAgent,
			false,
			&message,
		)
	}
}

func (sam *SecureAuthMiddleware) logAccessViolation(c *gin.Context, tenant *models.Tenant, resourceTenantID uuid.UUID) {
	if sam.auditLogger != nil {
		sam.auditLogger.LogAuthAction(
			c.Request.Context(),
			&tenant.ID,
			nil,
			"auth.access_violation",
			map[string]interface{}{
				"tenant_name":        tenant.Name,
				"resource_tenant_id": resourceTenantID.String(),
				"path":               c.Request.URL.Path,
				"method":             c.Request.Method,
			},
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			false,
			stringPtr("Attempted to access resource belonging to different tenant"),
		)
	}
}

func (sam *SecureAuthMiddleware) logSuspiciousActivity(c *gin.Context, activityType string, details map[string]interface{}) {
	if sam.auditLogger != nil {
		// Try to get tenant from context if available
		var tenantID *uuid.UUID
		if tenant, exists := auth.GetTenantFromContext(c); exists {
			tenantID = &tenant.ID
		}

		sam.auditLogger.LogAuthAction(
			c.Request.Context(),
			tenantID,
			nil,
			"security.suspicious_activity",
			map[string]interface{}{
				"activity_type": activityType,
				"details":       details,
				"path":          c.Request.URL.Path,
				"method":        c.Request.Method,
			},
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			false,
			stringPtr("Suspicious activity detected"),
		)
	}
}

func stringPtr(s string) *string {
	return &s
}
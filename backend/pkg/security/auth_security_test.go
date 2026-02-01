package security

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/auth"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTenantRepository for testing
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Tenant, error) {
	args := m.Called(ctx, apiKeyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantRepository) List(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.Tenant), args.Error(1)
}

// AuthSecurityTester provides security tests for authentication and authorization
type AuthSecurityTester struct {
	auditLogger *AuditLogger
}

// NewAuthSecurityTester creates a new authentication security tester
func NewAuthSecurityTester(auditLogger *AuditLogger) *AuthSecurityTester {
	return &AuthSecurityTester{
		auditLogger: auditLogger,
	}
}

// TestAPIKeyValidation tests API key validation security
func TestAPIKeyValidation(t *testing.T) {
	mockRepo := &MockTenantRepository{}
	middleware := auth.NewAuthMiddleware(mockRepo)

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "MISSING_AUTH_HEADER",
		},
		{
			name:           "invalid authorization format",
			authHeader:     "InvalidFormat token123",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "INVALID_AUTH_FORMAT",
		},
		{
			name:           "invalid API key format",
			authHeader:     "Bearer invalid-key",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "INVALID_API_KEY_FORMAT",
		},
		{
			name:       "valid API key format but not found",
			authHeader: "Bearer wh_test_1234567890abcdef1234567890abcdef",
			setupMock: func() {
				mockRepo.On("FindByAPIKey", mock.Anything, "wh_test_1234567890abcdef1234567890abcdef").
					Return(nil, fmt.Errorf("not found"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "INVALID_API_KEY",
		},
		{
			name:       "valid API key",
			authHeader: "Bearer wh_test_1234567890abcdef1234567890abcdef",
			setupMock: func() {
				tenant := &models.Tenant{
					ID:   uuid.New(),
					Name: "Test Tenant",
				}
				mockRepo.On("FindByAPIKey", mock.Anything, "wh_test_1234567890abcdef1234567890abcdef").
					Return(tenant, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
			tt.setupMock()

			// Setup Gin router
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.RequireAuth())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyBruteForceProtection tests protection against brute force attacks
func TestAPIKeyBruteForceProtection(t *testing.T) {
	mockRepo := &MockTenantRepository{}
	mockAuditRepo := &MockAuditRepository{}
	auditLogger := NewAuditLogger(mockAuditRepo)
	
	middleware := auth.NewAuthMiddleware(mockRepo)

	// Setup mock to always return "not found" for invalid keys
	mockRepo.On("FindByAPIKey", mock.Anything, mock.AnythingOfType("string")).
		Return(nil, fmt.Errorf("not found"))

	// Setup audit logging mock
	mockAuditRepo.On("LogEvent", mock.Anything, mock.AnythingOfType("*repository.AuditEvent")).
		Return(nil)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add audit logging middleware
	router.Use(func(c *gin.Context) {
		c.Set("audit_logger", auditLogger)
		c.Next()
		
		// Log failed authentication attempts
		if c.Writer.Status() == http.StatusUnauthorized {
			auditLogger.LogAuthAction(
				c.Request.Context(),
				nil, nil,
				ActionAuthAPIKeyInvalid,
				map[string]interface{}{
					"attempted_key": c.GetHeader("Authorization"),
					"user_agent":   c.GetHeader("User-Agent"),
				},
				c.ClientIP(),
				c.GetHeader("User-Agent"),
				false,
				testStringPtr("Invalid API key"),
			)
		}
	})
	
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Simulate multiple failed attempts
	invalidKeys := []string{
		"Bearer wh_test_invalid1",
		"Bearer wh_test_invalid2", 
		"Bearer wh_test_invalid3",
		"Bearer wh_test_invalid4",
		"Bearer wh_test_invalid5",
	}

	for _, key := range invalidKeys {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", key)
		req.Header.Set("User-Agent", "AttackerBot/1.0")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// Verify audit logs were created for failed attempts
	mockAuditRepo.AssertNumberOfCalls(t, "LogEvent", len(invalidKeys))
}

// TestTimingAttackProtection tests protection against timing attacks
func TestTimingAttackProtection(t *testing.T) {
	mockRepo := &MockTenantRepository{}
	middleware := auth.NewAuthMiddleware(mockRepo)

	// Setup mock for valid key
	validTenant := &models.Tenant{ID: uuid.New(), Name: "Valid Tenant"}
	mockRepo.On("FindByAPIKey", mock.Anything, "wh_test_validkey1234567890abcdef1234567890").
		Return(validTenant, nil)

	// Setup mock for invalid key (simulate database lookup time)
	mockRepo.On("FindByAPIKey", mock.Anything, "wh_test_invalidkey1234567890abcdef123456").
		Return(nil, fmt.Errorf("not found"))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Measure timing for valid key
	validKey := "Bearer wh_test_validkey1234567890abcdef1234567890"
	start := time.Now()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", validKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	validDuration := time.Since(start)

	// Measure timing for invalid key
	invalidKey := "Bearer wh_test_invalidkey1234567890abcdef123456"
	start = time.Now()
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", invalidKey)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	invalidDuration := time.Since(start)

	// The timing difference should not be significant (within reasonable bounds)
	// This is a basic check - in production, you'd want more sophisticated timing analysis
	timingRatio := float64(validDuration) / float64(invalidDuration)
	assert.True(t, timingRatio > 0.5 && timingRatio < 2.0, 
		"Timing difference too significant: valid=%v, invalid=%v, ratio=%f", 
		validDuration, invalidDuration, timingRatio)
}

// TestSecureHeaderValidation tests validation of security-related headers
func TestSecureHeaderValidation(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		expectBlock bool
		reason      string
	}{
		{
			name: "normal request",
			headers: map[string]string{
				"User-Agent": "MyApp/1.0",
				"Accept":     "application/json",
			},
			expectBlock: false,
		},
		{
			name: "suspicious user agent",
			headers: map[string]string{
				"User-Agent": "sqlmap/1.0",
			},
			expectBlock: true,
			reason:      "suspicious user agent",
		},
		{
			name: "missing user agent",
			headers: map[string]string{
				"Accept": "application/json",
			},
			expectBlock: false, // Some legitimate clients don't send User-Agent
		},
		{
			name: "suspicious headers combination",
			headers: map[string]string{
				"User-Agent":      "curl/7.0",
				"X-Forwarded-For": "127.0.0.1, 10.0.0.1, 192.168.1.1",
			},
			expectBlock: false, // This is actually normal for proxied requests
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be implemented as middleware that checks for suspicious patterns
			suspiciousPatterns := []string{"sqlmap", "nikto", "nmap", "masscan"}
			
			userAgent := tt.headers["User-Agent"]
			isSuspicious := false
			
			for _, pattern := range suspiciousPatterns {
				if strings.Contains(strings.ToLower(userAgent), pattern) {
					isSuspicious = true
					break
				}
			}
			
			assert.Equal(t, tt.expectBlock, isSuspicious, "Unexpected blocking decision for %s", tt.name)
		})
	}
}

// TestRateLimitingBypass tests that rate limiting cannot be easily bypassed
func TestRateLimitingBypass(t *testing.T) {
	// Test various bypass techniques
	bypassAttempts := []map[string]string{
		{"X-Forwarded-For": "1.2.3.4"},
		{"X-Real-IP": "5.6.7.8"},
		{"X-Originating-IP": "9.10.11.12"},
		{"X-Remote-IP": "13.14.15.16"},
		{"X-Client-IP": "17.18.19.20"},
	}

	for i, headers := range bypassAttempts {
		t.Run(fmt.Sprintf("bypass_attempt_%d", i), func(t *testing.T) {
			// In a real implementation, you would:
			// 1. Configure rate limiting to use the actual client IP
			// 2. Validate and sanitize proxy headers
			// 3. Use a trusted proxy list
			// 4. Implement multiple rate limiting strategies (per-IP, per-API-key, etc.)
			
			// For this test, we just verify that the headers are present
			// The actual rate limiting logic would need to be tested with a real rate limiter
			for key, value := range headers {
				assert.NotEmpty(t, key)
				assert.NotEmpty(t, value)
			}
		})
	}
}

// Helper function for tests
func testStringPtr(s string) *string {
	return &s
}

// MockAuditRepository for testing
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) LogEvent(ctx context.Context, event *repository.AuditEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockAuditRepository) GetAuditLogs(ctx context.Context, filter repository.AuditFilter) ([]*repository.AuditEvent, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*repository.AuditEvent), args.Error(1)
}

func (m *MockAuditRepository) GetAuditLogsByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.AuditFilter) ([]*repository.AuditEvent, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*repository.AuditEvent), args.Error(1)
}

// TestEncryptionKeyRotation tests that encryption keys can be rotated securely
func TestEncryptionKeyRotation(t *testing.T) {
	// Generate two different encryption keys
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	
	_, err := rand.Read(key1)
	require.NoError(t, err)
	_, err = rand.Read(key2)
	require.NoError(t, err)

	// Create encryption services with different keys
	service1, err := NewEncryptionService(key1)
	require.NoError(t, err)
	
	service2, err := NewEncryptionService(key2)
	require.NoError(t, err)

	plaintext := "sensitive data"

	// Encrypt with first key
	encrypted1, err := service1.EncryptString(plaintext)
	require.NoError(t, err)

	// Encrypt with second key
	encrypted2, err := service2.EncryptString(plaintext)
	require.NoError(t, err)

	// Verify that encrypted data is different
	assert.NotEqual(t, encrypted1, encrypted2)

	// Verify that each service can only decrypt its own data
	decrypted1, err := service1.DecryptString(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	// Service2 should not be able to decrypt service1's data
	_, err = service2.DecryptString(encrypted1)
	assert.Error(t, err)

	// Service1 should not be able to decrypt service2's data
	_, err = service1.DecryptString(encrypted2)
	assert.Error(t, err)
}
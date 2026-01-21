# Security Package

This package implements comprehensive security hardening and data protection features for the webhook service platform.

## Components

### 1. Data Encryption at Rest (`encryption.go`)

- **AES-256-GCM encryption** for sensitive data
- **Base64 encoding** for storage compatibility
- **Random nonces** for each encryption operation
- **Authenticated encryption** to prevent tampering

**Usage:**
```go
key := make([]byte, 32) // 256-bit key
service, err := NewEncryptionService(key)
encrypted, err := service.EncryptString("sensitive data")
decrypted, err := service.DecryptString(encrypted)
```

### 2. Secure Secret Storage and Rotation (`secret_manager.go`)

- **Versioned secrets** with automatic rotation
- **Grace period support** for gradual migration
- **Encrypted storage** of all secret values
- **Multi-secret validation** for backward compatibility

**Features:**
- Generate cryptographically secure secrets
- Rotate secrets with configurable grace periods
- Validate against multiple active secret versions
- Automatic cleanup of expired secrets

### 3. Audit Logging (`audit_logger.go`)

- **Comprehensive event tracking** for all administrative operations
- **Structured logging** with correlation IDs
- **Tenant isolation** in audit logs
- **Filtering and search** capabilities

**Tracked Events:**
- Authentication attempts (success/failure)
- Tenant management operations
- Webhook endpoint modifications
- Secret management actions
- Suspicious activity detection

### 4. Enhanced Authentication Middleware (`auth_middleware.go`)

- **Security headers** injection
- **Suspicious activity detection**
- **Audit logging integration**
- **Tenant access control**

**Security Features:**
- XSS protection headers
- Clickjacking prevention
- MIME type sniffing protection
- HTTPS enforcement (production)
- Cache control for sensitive data

### 5. Tenant Data Isolation (`isolation_test.go`)

- **Comprehensive isolation testing** framework
- **Cross-tenant access prevention**
- **Data segregation verification**
- **Repository-level isolation checks**

## Database Schema

### Secret Versions Table
```sql
CREATE TABLE secret_versions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    secret_id VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL,
    encrypted_value TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, secret_id, version)
);
```

### Audit Logs Table
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    user_id UUID,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    resource_id VARCHAR(255),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    timestamp TIMESTAMP DEFAULT NOW()
);
```

## Security Best Practices Implemented

### 1. Encryption
- AES-256-GCM for authenticated encryption
- Unique nonces for each encryption operation
- Secure key management practices
- Protection against timing attacks

### 2. Authentication & Authorization
- API key format validation
- Brute force protection through audit logging
- Tenant-based access control
- Suspicious activity detection

### 3. Data Protection
- Encryption at rest for sensitive payloads
- Secure secret storage with rotation
- Audit trails for all administrative actions
- Tenant data isolation verification

### 4. Security Headers
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Strict-Transport-Security (production)
- Cache-Control for sensitive endpoints

## Testing

The package includes comprehensive test coverage:

- **Unit tests** for all encryption operations
- **Security tests** for authentication flows
- **Integration tests** for tenant isolation
- **Mock-based testing** for repository interactions

Run tests with:
```bash
go test ./pkg/security/... -v
```

## Usage Examples

### Setting up Security Components

```go
// Initialize encryption service
encryptionKey := make([]byte, 32)
rand.Read(encryptionKey)
encryption, err := security.NewEncryptionService(encryptionKey)

// Initialize repositories
secretRepo := repository.NewSecretRepository(db)
auditRepo := repository.NewAuditRepository(db)

// Initialize security services
secretManager := security.NewSecretManager(encryption, secretRepo)
auditLogger := security.NewAuditLogger(auditRepo)

// Initialize secure middleware
authMiddleware := security.NewSecureAuthMiddleware(tenantRepo, auditLogger)
```

### Using in Gin Router

```go
router := gin.New()

// Add security middleware
router.Use(authMiddleware.SecurityHeaders())
router.Use(authMiddleware.SuspiciousActivityDetection())
router.Use(authMiddleware.RequireAuth())

// Protected routes
api := router.Group("/api/v1")
api.POST("/webhooks", webhookHandler.Create)
```

## Security Considerations

1. **Key Management**: Encryption keys should be stored securely (e.g., AWS KMS, HashiCorp Vault)
2. **Secret Rotation**: Implement regular secret rotation policies
3. **Audit Log Retention**: Configure appropriate retention policies for audit logs
4. **Monitoring**: Set up alerts for suspicious activities and failed authentications
5. **Access Control**: Ensure proper tenant isolation at all levels

## Compliance

This implementation supports compliance with:
- **GDPR**: Data encryption and audit trails
- **SOC 2**: Security controls and monitoring
- **PCI DSS**: Data protection requirements
- **HIPAA**: Audit logging and access controls
package security

import (
	"context"
	"time"
	"webhook-platform/pkg/repository"

	"github.com/google/uuid"
)

// AuditLogger handles audit logging for security events
type AuditLogger struct {
	repository repository.AuditRepository
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(repo repository.AuditRepository) *AuditLogger {
	return &AuditLogger{
		repository: repo,
	}
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, event *repository.AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return al.repository.LogEvent(ctx, event)
}

// LogTenantAction logs a tenant-related administrative action
func (al *AuditLogger) LogTenantAction(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, action, resourceID string, details map[string]interface{}, ipAddress, userAgent string, success bool, errorMsg *string) error {
	event := &repository.AuditEvent{
		ID:         uuid.New(),
		TenantID:   &tenantID,
		UserID:     userID,
		Action:     action,
		Resource:   "tenant",
		ResourceID: &resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    success,
		ErrorMsg:   errorMsg,
		Timestamp:  time.Now(),
	}

	return al.LogEvent(ctx, event)
}

// LogWebhookAction logs a webhook-related administrative action
func (al *AuditLogger) LogWebhookAction(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, action, webhookID string, details map[string]interface{}, ipAddress, userAgent string, success bool, errorMsg *string) error {
	event := &repository.AuditEvent{
		ID:         uuid.New(),
		TenantID:   &tenantID,
		UserID:     userID,
		Action:     action,
		Resource:   "webhook_endpoint",
		ResourceID: &webhookID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    success,
		ErrorMsg:   errorMsg,
		Timestamp:  time.Now(),
	}

	return al.LogEvent(ctx, event)
}

// LogSecretAction logs a secret management action
func (al *AuditLogger) LogSecretAction(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, action, secretID string, details map[string]interface{}, ipAddress, userAgent string, success bool, errorMsg *string) error {
	event := &repository.AuditEvent{
		ID:         uuid.New(),
		TenantID:   &tenantID,
		UserID:     userID,
		Action:     action,
		Resource:   "secret",
		ResourceID: &secretID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    success,
		ErrorMsg:   errorMsg,
		Timestamp:  time.Now(),
	}

	return al.LogEvent(ctx, event)
}

// LogAuthAction logs an authentication-related action
func (al *AuditLogger) LogAuthAction(ctx context.Context, tenantID *uuid.UUID, userID *uuid.UUID, action string, details map[string]interface{}, ipAddress, userAgent string, success bool, errorMsg *string) error {
	event := &repository.AuditEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		Resource:  "authentication",
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		ErrorMsg:  errorMsg,
		Timestamp: time.Now(),
	}

	return al.LogEvent(ctx, event)
}

// GetAuditLogs retrieves audit logs with filtering
func (al *AuditLogger) GetAuditLogs(ctx context.Context, filter repository.AuditFilter) ([]*repository.AuditEvent, error) {
	return al.repository.GetAuditLogs(ctx, filter)
}

// GetTenantAuditLogs retrieves audit logs for a specific tenant
func (al *AuditLogger) GetTenantAuditLogs(ctx context.Context, tenantID uuid.UUID, filter repository.AuditFilter) ([]*repository.AuditEvent, error) {
	return al.repository.GetAuditLogsByTenant(ctx, tenantID, filter)
}

// Common audit actions
const (
	ActionTenantCreate         = "tenant.create"
	ActionTenantUpdate         = "tenant.update"
	ActionTenantDelete         = "tenant.delete"
	ActionTenantAPIKeyRotate   = "tenant.api_key.rotate"
	
	ActionWebhookCreate        = "webhook.create"
	ActionWebhookUpdate        = "webhook.update"
	ActionWebhookDelete        = "webhook.delete"
	ActionWebhookActivate      = "webhook.activate"
	ActionWebhookDeactivate    = "webhook.deactivate"
	
	ActionSecretGenerate       = "secret.generate"
	ActionSecretRotate         = "secret.rotate"
	ActionSecretDeactivate     = "secret.deactivate"
	ActionSecretAccess         = "secret.access"
	
	ActionAuthLogin            = "auth.login"
	ActionAuthLoginFailed      = "auth.login.failed"
	ActionAuthLogout           = "auth.logout"
	ActionAuthAPIKeyUsed       = "auth.api_key.used"
	ActionAuthAPIKeyInvalid    = "auth.api_key.invalid"
)
package mtls

import "context"

// Repository defines the data access interface for mTLS management
type Repository interface {
	// Certificates
	CreateCertificate(ctx context.Context, cert *Certificate) error
	GetCertificate(ctx context.Context, tenantID, certID string) (*Certificate, error)
	ListCertificates(ctx context.Context, tenantID string) ([]Certificate, error)
	UpdateCertificate(ctx context.Context, cert *Certificate) error
	DeleteCertificate(ctx context.Context, tenantID, certID string) error
	ListExpiringCertificates(ctx context.Context, withinDays int) ([]Certificate, error)

	// TLS Policies
	CreateTLSPolicy(ctx context.Context, policy *TLSPolicy) error
	GetTLSPolicy(ctx context.Context, tenantID, endpointID string) (*TLSPolicy, error)
	ListTLSPolicies(ctx context.Context, tenantID string) ([]TLSPolicy, error)
	UpdateTLSPolicy(ctx context.Context, policy *TLSPolicy) error
	DeleteTLSPolicy(ctx context.Context, tenantID, policyID string) error
}

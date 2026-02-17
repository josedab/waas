package federation

import (
	"context"
	"database/sql"

	"github.com/josedab/waas/pkg/utils"
)

// Repository defines federation data access
type Repository interface {
	// Members
	SaveMember(ctx context.Context, member *FederationMember) error
	GetMember(ctx context.Context, memberID string) (*FederationMember, error)
	GetMemberByDomain(ctx context.Context, domain string) (*FederationMember, error)
	ListMembers(ctx context.Context, tenantID string, status *MemberStatus) ([]FederationMember, error)
	DeleteMember(ctx context.Context, memberID string) error

	// Trust relationships
	SaveTrustRelationship(ctx context.Context, trust *TrustRelationship) error
	GetTrustRelationship(ctx context.Context, trustID string) (*TrustRelationship, error)
	GetTrustBetween(ctx context.Context, sourceID, targetID string) (*TrustRelationship, error)
	ListTrustRelationships(ctx context.Context, tenantID, memberID string) ([]TrustRelationship, error)

	// Trust requests
	SaveTrustRequest(ctx context.Context, req *TrustRequest) error
	GetTrustRequest(ctx context.Context, reqID string) (*TrustRequest, error)
	ListTrustRequests(ctx context.Context, tenantID string, status *TrustReqStatus) ([]TrustRequest, error)
	UpdateTrustRequestStatus(ctx context.Context, reqID string, status TrustReqStatus, response string) error

	// Event catalogs
	SaveCatalog(ctx context.Context, catalog *EventCatalog) error
	GetCatalog(ctx context.Context, catalogID string) (*EventCatalog, error)
	ListCatalogs(ctx context.Context, tenantID string, public bool) ([]EventCatalog, error)
	ListPublicCatalogs(ctx context.Context) ([]EventCatalog, error)

	// Subscriptions
	SaveSubscription(ctx context.Context, sub *FederatedSubscription) error
	GetSubscription(ctx context.Context, subID string) (*FederatedSubscription, error)
	ListSubscriptions(ctx context.Context, tenantID string, status *SubStatus) ([]FederatedSubscription, error)
	ListSubscriptionsByMember(ctx context.Context, memberID string) ([]FederatedSubscription, error)

	// Deliveries
	SaveDelivery(ctx context.Context, delivery *FederatedDelivery) error
	GetDelivery(ctx context.Context, deliveryID string) (*FederatedDelivery, error)
	ListPendingDeliveries(ctx context.Context, limit int) ([]FederatedDelivery, error)
	ListDeliveries(ctx context.Context, tenantID, subID string, limit int) ([]FederatedDelivery, error)
	UpdateDeliveryStatus(ctx context.Context, deliveryID string, status DeliveryStatus, err string, respCode int) error

	// Policy
	SavePolicy(ctx context.Context, policy *FederationPolicy) error
	GetPolicy(ctx context.Context, tenantID string) (*FederationPolicy, error)

	// Keys
	SaveKeys(ctx context.Context, keys *CryptoKeys) error
	GetKeys(ctx context.Context, memberID string) (*CryptoKeys, error)
	GetKeyByID(ctx context.Context, keyID string) (*CryptoKeys, error)

	// Metrics
	GetMetrics(ctx context.Context, tenantID string) (*FederationMetrics, error)
	UpdateMetrics(ctx context.Context, metrics *FederationMetrics) error
}

// PostgresRepository implements Repository
type PostgresRepository struct {
	db     *sql.DB
	logger *utils.Logger
}

// NewPostgresRepository creates repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db, logger: utils.NewLogger("federation")}
}

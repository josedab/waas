package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
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
	db *sql.DB
}

// NewPostgresRepository creates repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveMember saves a federation member
func (r *PostgresRepository) SaveMember(ctx context.Context, member *FederationMember) error {
	endpointsJSON, err := json.Marshal(member.Endpoints)
	if err != nil {
		return fmt.Errorf("failed to marshal member endpoints: %w", err)
	}
	capsJSON, err := json.Marshal(member.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal member capabilities: %w", err)
	}
	metaJSON, err := json.Marshal(member.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal member metadata: %w", err)
	}

	query := `
		INSERT INTO federation_members (
			id, tenant_id, organization_id, name, domain, status, public_key,
			endpoints, capabilities, trust_level, metadata,
			joined_at, last_seen_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			public_key = EXCLUDED.public_key,
			endpoints = EXCLUDED.endpoints,
			capabilities = EXCLUDED.capabilities,
			trust_level = EXCLUDED.trust_level,
			metadata = EXCLUDED.metadata,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		member.ID, member.TenantID, member.OrganizationID, member.Name, member.Domain,
		member.Status, member.PublicKey, endpointsJSON, capsJSON, member.TrustLevel,
		metaJSON, member.JoinedAt, member.LastSeenAt, member.CreatedAt, member.UpdatedAt)

	return err
}

// GetMember retrieves a member
func (r *PostgresRepository) GetMember(ctx context.Context, memberID string) (*FederationMember, error) {
	query := `
		SELECT id, tenant_id, organization_id, name, domain, status, public_key,
			   endpoints, capabilities, trust_level, metadata,
			   joined_at, last_seen_at, created_at, updated_at
		FROM federation_members
		WHERE id = $1`

	var m FederationMember
	var endpointsJSON, capsJSON, metaJSON []byte

	err := r.db.QueryRowContext(ctx, query, memberID).Scan(
		&m.ID, &m.TenantID, &m.OrganizationID, &m.Name, &m.Domain, &m.Status,
		&m.PublicKey, &endpointsJSON, &capsJSON, &m.TrustLevel, &metaJSON,
		&m.JoinedAt, &m.LastSeenAt, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(endpointsJSON, &m.Endpoints); err != nil {
		log.Printf("federation: failed to unmarshal member %s endpoints: %v", memberID, err)
	}
	if err := json.Unmarshal(capsJSON, &m.Capabilities); err != nil {
		log.Printf("federation: failed to unmarshal member %s capabilities: %v", memberID, err)
	}
	if err := json.Unmarshal(metaJSON, &m.Metadata); err != nil {
		log.Printf("federation: failed to unmarshal member %s metadata: %v", memberID, err)
	}

	return &m, nil
}

// GetMemberByDomain retrieves member by domain
func (r *PostgresRepository) GetMemberByDomain(ctx context.Context, domain string) (*FederationMember, error) {
	query := `SELECT id FROM federation_members WHERE domain = $1`
	var memberID string
	err := r.db.QueryRowContext(ctx, query, domain).Scan(&memberID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}
	return r.GetMember(ctx, memberID)
}

// ListMembers lists members
func (r *PostgresRepository) ListMembers(ctx context.Context, tenantID string, status *MemberStatus) ([]FederationMember, error) {
	query := `
		SELECT id, tenant_id, organization_id, name, domain, status, public_key,
			   endpoints, capabilities, trust_level, metadata,
			   joined_at, last_seen_at, created_at, updated_at
		FROM federation_members
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []FederationMember
	for rows.Next() {
		var m FederationMember
		var endpointsJSON, capsJSON, metaJSON []byte

		err := rows.Scan(
			&m.ID, &m.TenantID, &m.OrganizationID, &m.Name, &m.Domain, &m.Status,
			&m.PublicKey, &endpointsJSON, &capsJSON, &m.TrustLevel, &metaJSON,
			&m.JoinedAt, &m.LastSeenAt, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(endpointsJSON, &m.Endpoints); err != nil {
			log.Printf("federation: failed to unmarshal member %s endpoints: %v", m.ID, err)
		}
		if err := json.Unmarshal(capsJSON, &m.Capabilities); err != nil {
			log.Printf("federation: failed to unmarshal member %s capabilities: %v", m.ID, err)
		}
		if err := json.Unmarshal(metaJSON, &m.Metadata); err != nil {
			log.Printf("federation: failed to unmarshal member %s metadata: %v", m.ID, err)
		}

		members = append(members, m)
	}

	return members, nil
}

// DeleteMember deletes a member
func (r *PostgresRepository) DeleteMember(ctx context.Context, memberID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM federation_members WHERE id = $1", memberID)
	return err
}

// SaveTrustRelationship saves a trust relationship
func (r *PostgresRepository) SaveTrustRelationship(ctx context.Context, trust *TrustRelationship) error {
	permsJSON, err := json.Marshal(trust.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal trust permissions: %w", err)
	}

	query := `
		INSERT INTO federation_trust (
			id, tenant_id, source_member_id, target_member_id, status,
			trust_level, permissions, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			trust_level = EXCLUDED.trust_level,
			permissions = EXCLUDED.permissions,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		trust.ID, trust.TenantID, trust.SourceMemberID, trust.TargetMemberID,
		trust.Status, trust.TrustLevel, permsJSON, trust.ExpiresAt,
		trust.CreatedAt, trust.UpdatedAt)

	return err
}

// GetTrustRelationship retrieves a trust relationship
func (r *PostgresRepository) GetTrustRelationship(ctx context.Context, trustID string) (*TrustRelationship, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, status,
			   trust_level, permissions, expires_at, created_at, updated_at
		FROM federation_trust
		WHERE id = $1`

	var t TrustRelationship
	var permsJSON []byte
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, trustID).Scan(
		&t.ID, &t.TenantID, &t.SourceMemberID, &t.TargetMemberID, &t.Status,
		&t.TrustLevel, &permsJSON, &expiresAt, &t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trust relationship not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(permsJSON, &t.Permissions); err != nil {
		log.Printf("federation: failed to unmarshal trust %s permissions: %v", trustID, err)
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}

	return &t, nil
}

// GetTrustBetween gets trust between two members
func (r *PostgresRepository) GetTrustBetween(ctx context.Context, sourceID, targetID string) (*TrustRelationship, error) {
	query := `
		SELECT id FROM federation_trust
		WHERE source_member_id = $1 AND target_member_id = $2`

	var trustID string
	err := r.db.QueryRowContext(ctx, query, sourceID, targetID).Scan(&trustID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trust relationship not found")
	}
	if err != nil {
		return nil, err
	}

	return r.GetTrustRelationship(ctx, trustID)
}

// ListTrustRelationships lists trust relationships
func (r *PostgresRepository) ListTrustRelationships(ctx context.Context, tenantID, memberID string) ([]TrustRelationship, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, status,
			   trust_level, permissions, expires_at, created_at, updated_at
		FROM federation_trust
		WHERE tenant_id = $1 AND (source_member_id = $2 OR target_member_id = $2)
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trusts []TrustRelationship
	for rows.Next() {
		var t TrustRelationship
		var permsJSON []byte
		var expiresAt sql.NullTime

		err := rows.Scan(
			&t.ID, &t.TenantID, &t.SourceMemberID, &t.TargetMemberID, &t.Status,
			&t.TrustLevel, &permsJSON, &expiresAt, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(permsJSON, &t.Permissions); err != nil {
			log.Printf("federation: failed to unmarshal trust %s permissions in list: %v", t.ID, err)
		}
		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Time
		}

		trusts = append(trusts, t)
	}

	return trusts, nil
}

// SaveTrustRequest saves a trust request
func (r *PostgresRepository) SaveTrustRequest(ctx context.Context, req *TrustRequest) error {
	permsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal trust request permissions: %w", err)
	}

	query := `
		INSERT INTO federation_trust_requests (
			id, tenant_id, requester_id, target_member_id, requested_level,
			permissions, message, status, expires_at, responded_at, response, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			responded_at = EXCLUDED.responded_at,
			response = EXCLUDED.response`

	_, err = r.db.ExecContext(ctx, query,
		req.ID, req.TenantID, req.RequesterID, req.TargetMemberID, req.RequestedLevel,
		permsJSON, req.Message, req.Status, req.ExpiresAt, req.RespondedAt,
		req.Response, req.CreatedAt)

	return err
}

// GetTrustRequest retrieves a trust request
func (r *PostgresRepository) GetTrustRequest(ctx context.Context, reqID string) (*TrustRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, target_member_id, requested_level,
			   permissions, message, status, expires_at, responded_at, response, created_at
		FROM federation_trust_requests
		WHERE id = $1`

	var req TrustRequest
	var permsJSON []byte
	var expiresAt, respondedAt sql.NullTime
	var response sql.NullString

	err := r.db.QueryRowContext(ctx, query, reqID).Scan(
		&req.ID, &req.TenantID, &req.RequesterID, &req.TargetMemberID,
		&req.RequestedLevel, &permsJSON, &req.Message, &req.Status,
		&expiresAt, &respondedAt, &response, &req.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trust request not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(permsJSON, &req.Permissions); err != nil {
		log.Printf("federation: failed to unmarshal trust request %s permissions: %v", reqID, err)
	}
	if expiresAt.Valid {
		req.ExpiresAt = &expiresAt.Time
	}
	if respondedAt.Valid {
		req.RespondedAt = &respondedAt.Time
	}
	if response.Valid {
		req.Response = response.String
	}

	return &req, nil
}

// ListTrustRequests lists trust requests
func (r *PostgresRepository) ListTrustRequests(ctx context.Context, tenantID string, status *TrustReqStatus) ([]TrustRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, target_member_id, requested_level,
			   permissions, message, status, expires_at, responded_at, response, created_at
		FROM federation_trust_requests
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []TrustRequest
	for rows.Next() {
		var req TrustRequest
		var permsJSON []byte
		var expiresAt, respondedAt sql.NullTime
		var response sql.NullString

		err := rows.Scan(
			&req.ID, &req.TenantID, &req.RequesterID, &req.TargetMemberID,
			&req.RequestedLevel, &permsJSON, &req.Message, &req.Status,
			&expiresAt, &respondedAt, &response, &req.CreatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(permsJSON, &req.Permissions); err != nil {
			log.Printf("federation: failed to unmarshal trust request %s permissions in list: %v", req.ID, err)
		}
		if expiresAt.Valid {
			req.ExpiresAt = &expiresAt.Time
		}
		if respondedAt.Valid {
			req.RespondedAt = &respondedAt.Time
		}
		if response.Valid {
			req.Response = response.String
		}

		requests = append(requests, req)
	}

	return requests, nil
}

// UpdateTrustRequestStatus updates trust request status
func (r *PostgresRepository) UpdateTrustRequestStatus(ctx context.Context, reqID string, status TrustReqStatus, response string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE federation_trust_requests SET status = $1, responded_at = $2, response = $3 WHERE id = $4",
		status, now, response, reqID)
	return err
}

// SaveCatalog saves an event catalog
func (r *PostgresRepository) SaveCatalog(ctx context.Context, catalog *EventCatalog) error {
	eventsJSON, err := json.Marshal(catalog.EventTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog event types: %w", err)
	}

	query := `
		INSERT INTO federation_catalogs (
			id, tenant_id, member_id, name, description, event_types,
			version, public, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			event_types = EXCLUDED.event_types,
			version = EXCLUDED.version,
			public = EXCLUDED.public,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		catalog.ID, catalog.TenantID, catalog.MemberID, catalog.Name,
		catalog.Description, eventsJSON, catalog.Version, catalog.Public,
		catalog.CreatedAt, catalog.UpdatedAt)

	return err
}

// GetCatalog retrieves a catalog
func (r *PostgresRepository) GetCatalog(ctx context.Context, catalogID string) (*EventCatalog, error) {
	query := `
		SELECT id, tenant_id, member_id, name, description, event_types,
			   version, public, created_at, updated_at
		FROM federation_catalogs
		WHERE id = $1`

	var c EventCatalog
	var eventsJSON []byte

	err := r.db.QueryRowContext(ctx, query, catalogID).Scan(
		&c.ID, &c.TenantID, &c.MemberID, &c.Name, &c.Description,
		&eventsJSON, &c.Version, &c.Public, &c.CreatedAt, &c.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("catalog not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(eventsJSON, &c.EventTypes); err != nil {
		log.Printf("federation: failed to unmarshal catalog %s event types: %v", catalogID, err)
	}

	return &c, nil
}

// ListCatalogs lists catalogs
func (r *PostgresRepository) ListCatalogs(ctx context.Context, tenantID string, public bool) ([]EventCatalog, error) {
	query := `
		SELECT id, tenant_id, member_id, name, description, event_types,
			   version, public, created_at, updated_at
		FROM federation_catalogs
		WHERE tenant_id = $1`

	if public {
		query += " AND public = true"
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalogs []EventCatalog
	for rows.Next() {
		var c EventCatalog
		var eventsJSON []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.MemberID, &c.Name, &c.Description,
			&eventsJSON, &c.Version, &c.Public, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &c.EventTypes); err != nil {
			log.Printf("federation: failed to unmarshal catalog %s event types in list: %v", c.ID, err)
		}
		catalogs = append(catalogs, c)
	}

	return catalogs, nil
}

// ListPublicCatalogs lists all public catalogs
func (r *PostgresRepository) ListPublicCatalogs(ctx context.Context) ([]EventCatalog, error) {
	query := `
		SELECT id, tenant_id, member_id, name, description, event_types,
			   version, public, created_at, updated_at
		FROM federation_catalogs
		WHERE public = true
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalogs []EventCatalog
	for rows.Next() {
		var c EventCatalog
		var eventsJSON []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.MemberID, &c.Name, &c.Description,
			&eventsJSON, &c.Version, &c.Public, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &c.EventTypes); err != nil {
			log.Printf("federation: failed to unmarshal public catalog %s event types: %v", c.ID, err)
		}
		catalogs = append(catalogs, c)
	}

	return catalogs, nil
}

// SaveSubscription saves a subscription
func (r *PostgresRepository) SaveSubscription(ctx context.Context, sub *FederatedSubscription) error {
	eventsJSON, err := json.Marshal(sub.EventTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription event types: %w", err)
	}
	filterJSON, err := json.Marshal(sub.Filter)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription filter: %w", err)
	}
	deliveryJSON, err := json.Marshal(sub.DeliveryConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription delivery config: %w", err)
	}

	query := `
		INSERT INTO federation_subscriptions (
			id, tenant_id, source_member_id, target_member_id, catalog_id,
			event_types, filter, status, delivery_config, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			event_types = EXCLUDED.event_types,
			filter = EXCLUDED.filter,
			status = EXCLUDED.status,
			delivery_config = EXCLUDED.delivery_config,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.SourceMemberID, sub.TargetMemberID, sub.CatalogID,
		eventsJSON, filterJSON, sub.Status, deliveryJSON, sub.CreatedAt, sub.UpdatedAt)

	return err
}

// GetSubscription retrieves a subscription
func (r *PostgresRepository) GetSubscription(ctx context.Context, subID string) (*FederatedSubscription, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, catalog_id,
			   event_types, filter, status, delivery_config, created_at, updated_at
		FROM federation_subscriptions
		WHERE id = $1`

	var s FederatedSubscription
	var eventsJSON, filterJSON, deliveryJSON []byte

	err := r.db.QueryRowContext(ctx, query, subID).Scan(
		&s.ID, &s.TenantID, &s.SourceMemberID, &s.TargetMemberID, &s.CatalogID,
		&eventsJSON, &filterJSON, &s.Status, &deliveryJSON, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(eventsJSON, &s.EventTypes); err != nil {
		log.Printf("federation: failed to unmarshal subscription %s event types: %v", subID, err)
	}
	if err := json.Unmarshal(deliveryJSON, &s.DeliveryConfig); err != nil {
		log.Printf("federation: failed to unmarshal subscription %s delivery config: %v", subID, err)
	}
	if len(filterJSON) > 0 {
		var f EventFilter
		if err := json.Unmarshal(filterJSON, &f); err != nil {
			log.Printf("federation: failed to unmarshal subscription %s filter: %v", subID, err)
		} else {
			s.Filter = &f
		}
	}

	return &s, nil
}

// ListSubscriptions lists subscriptions
func (r *PostgresRepository) ListSubscriptions(ctx context.Context, tenantID string, status *SubStatus) ([]FederatedSubscription, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, catalog_id,
			   event_types, filter, status, delivery_config, created_at, updated_at
		FROM federation_subscriptions
		WHERE tenant_id = $1`
	args := []any{tenantID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []FederatedSubscription
	for rows.Next() {
		var s FederatedSubscription
		var eventsJSON, filterJSON, deliveryJSON []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.SourceMemberID, &s.TargetMemberID, &s.CatalogID,
			&eventsJSON, &filterJSON, &s.Status, &deliveryJSON, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &s.EventTypes); err != nil {
			log.Printf("federation: failed to unmarshal subscription %s event types in list: %v", s.ID, err)
		}
		if err := json.Unmarshal(deliveryJSON, &s.DeliveryConfig); err != nil {
			log.Printf("federation: failed to unmarshal subscription %s delivery config in list: %v", s.ID, err)
		}
		if len(filterJSON) > 0 {
			var f EventFilter
			if err := json.Unmarshal(filterJSON, &f); err != nil {
				log.Printf("federation: failed to unmarshal subscription %s filter in list: %v", s.ID, err)
			} else {
				s.Filter = &f
			}
		}

		subs = append(subs, s)
	}

	return subs, nil
}

// ListSubscriptionsByMember lists subscriptions by member
func (r *PostgresRepository) ListSubscriptionsByMember(ctx context.Context, memberID string) ([]FederatedSubscription, error) {
	query := `
		SELECT id, tenant_id, source_member_id, target_member_id, catalog_id,
			   event_types, filter, status, delivery_config, created_at, updated_at
		FROM federation_subscriptions
		WHERE target_member_id = $1 AND status = 'active'`

	rows, err := r.db.QueryContext(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []FederatedSubscription
	for rows.Next() {
		var s FederatedSubscription
		var eventsJSON, filterJSON, deliveryJSON []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.SourceMemberID, &s.TargetMemberID, &s.CatalogID,
			&eventsJSON, &filterJSON, &s.Status, &deliveryJSON, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(eventsJSON, &s.EventTypes); err != nil {
			log.Printf("federation: failed to unmarshal subscription %s event types by member: %v", s.ID, err)
		}
		if err := json.Unmarshal(deliveryJSON, &s.DeliveryConfig); err != nil {
			log.Printf("federation: failed to unmarshal subscription %s delivery config by member: %v", s.ID, err)
		}
		if len(filterJSON) > 0 {
			var f EventFilter
			if err := json.Unmarshal(filterJSON, &f); err != nil {
				log.Printf("federation: failed to unmarshal subscription %s filter by member: %v", s.ID, err)
			} else {
				s.Filter = &f
			}
		}

		subs = append(subs, s)
	}

	return subs, nil
}

// SaveDelivery saves a delivery
func (r *PostgresRepository) SaveDelivery(ctx context.Context, delivery *FederatedDelivery) error {
	payloadJSON, err := json.Marshal(delivery.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery payload: %w", err)
	}

	query := `
		INSERT INTO federation_deliveries (
			id, tenant_id, subscription_id, source_member_id, target_member_id,
			event_type, event_id, payload, status, attempts, last_attempt_at,
			next_retry_at, error, response_code, response_body, latency,
			delivered_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			attempts = EXCLUDED.attempts,
			last_attempt_at = EXCLUDED.last_attempt_at,
			next_retry_at = EXCLUDED.next_retry_at,
			error = EXCLUDED.error,
			response_code = EXCLUDED.response_code,
			response_body = EXCLUDED.response_body,
			latency = EXCLUDED.latency,
			delivered_at = EXCLUDED.delivered_at`

	_, err = r.db.ExecContext(ctx, query,
		delivery.ID, delivery.TenantID, delivery.SubscriptionID,
		delivery.SourceMemberID, delivery.TargetMemberID, delivery.EventType,
		delivery.EventID, payloadJSON, delivery.Status, delivery.Attempts,
		delivery.LastAttemptAt, delivery.NextRetryAt, delivery.Error,
		delivery.ResponseCode, delivery.ResponseBody, delivery.Latency,
		delivery.DeliveredAt, delivery.CreatedAt)

	return err
}

// GetDelivery retrieves a delivery
func (r *PostgresRepository) GetDelivery(ctx context.Context, deliveryID string) (*FederatedDelivery, error) {
	query := `
		SELECT id, tenant_id, subscription_id, source_member_id, target_member_id,
			   event_type, event_id, payload, status, attempts, last_attempt_at,
			   next_retry_at, error, response_code, response_body, latency,
			   delivered_at, created_at
		FROM federation_deliveries
		WHERE id = $1`

	var d FederatedDelivery
	var payloadJSON []byte
	var lastAttemptAt, nextRetryAt, deliveredAt sql.NullTime
	var errStr, responseBody sql.NullString
	var responseCode sql.NullInt32

	err := r.db.QueryRowContext(ctx, query, deliveryID).Scan(
		&d.ID, &d.TenantID, &d.SubscriptionID, &d.SourceMemberID, &d.TargetMemberID,
		&d.EventType, &d.EventID, &payloadJSON, &d.Status, &d.Attempts,
		&lastAttemptAt, &nextRetryAt, &errStr, &responseCode, &responseBody,
		&d.Latency, &deliveredAt, &d.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("delivery not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadJSON, &d.Payload); err != nil {
		log.Printf("federation: failed to unmarshal delivery %s payload: %v", deliveryID, err)
	}
	if lastAttemptAt.Valid {
		d.LastAttemptAt = &lastAttemptAt.Time
	}
	if nextRetryAt.Valid {
		d.NextRetryAt = &nextRetryAt.Time
	}
	if deliveredAt.Valid {
		d.DeliveredAt = &deliveredAt.Time
	}
	if errStr.Valid {
		d.Error = errStr.String
	}
	if responseBody.Valid {
		d.ResponseBody = responseBody.String
	}
	if responseCode.Valid {
		d.ResponseCode = int(responseCode.Int32)
	}

	return &d, nil
}

// ListPendingDeliveries lists pending deliveries
func (r *PostgresRepository) ListPendingDeliveries(ctx context.Context, limit int) ([]FederatedDelivery, error) {
	query := `
		SELECT id, tenant_id, subscription_id, source_member_id, target_member_id,
			   event_type, event_id, payload, status, attempts, last_attempt_at,
			   next_retry_at, error, response_code, response_body, latency,
			   delivered_at, created_at
		FROM federation_deliveries
		WHERE status IN ('pending', 'retrying')
			AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanDeliveries(rows)
}

// ListDeliveries lists deliveries
func (r *PostgresRepository) ListDeliveries(ctx context.Context, tenantID, subID string, limit int) ([]FederatedDelivery, error) {
	query := `
		SELECT id, tenant_id, subscription_id, source_member_id, target_member_id,
			   event_type, event_id, payload, status, attempts, last_attempt_at,
			   next_retry_at, error, response_code, response_body, latency,
			   delivered_at, created_at
		FROM federation_deliveries
		WHERE tenant_id = $1 AND subscription_id = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, subID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanDeliveries(rows)
}

func (r *PostgresRepository) scanDeliveries(rows *sql.Rows) ([]FederatedDelivery, error) {
	var deliveries []FederatedDelivery
	for rows.Next() {
		var d FederatedDelivery
		var payloadJSON []byte
		var lastAttemptAt, nextRetryAt, deliveredAt sql.NullTime
		var errStr, responseBody sql.NullString
		var responseCode sql.NullInt32

		err := rows.Scan(
			&d.ID, &d.TenantID, &d.SubscriptionID, &d.SourceMemberID, &d.TargetMemberID,
			&d.EventType, &d.EventID, &payloadJSON, &d.Status, &d.Attempts,
			&lastAttemptAt, &nextRetryAt, &errStr, &responseCode, &responseBody,
			&d.Latency, &deliveredAt, &d.CreatedAt)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(payloadJSON, &d.Payload); err != nil {
			log.Printf("federation: failed to unmarshal delivery %s payload in scan: %v", d.ID, err)
		}
		if lastAttemptAt.Valid {
			d.LastAttemptAt = &lastAttemptAt.Time
		}
		if nextRetryAt.Valid {
			d.NextRetryAt = &nextRetryAt.Time
		}
		if deliveredAt.Valid {
			d.DeliveredAt = &deliveredAt.Time
		}
		if errStr.Valid {
			d.Error = errStr.String
		}
		if responseBody.Valid {
			d.ResponseBody = responseBody.String
		}
		if responseCode.Valid {
			d.ResponseCode = int(responseCode.Int32)
		}

		deliveries = append(deliveries, d)
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates delivery status
func (r *PostgresRepository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status DeliveryStatus, errMsg string, respCode int) error {
	now := time.Now()
	var deliveredAt *time.Time
	if status == DeliverySucceeded {
		deliveredAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE federation_deliveries SET 
			status = $1, last_attempt_at = $2, error = $3, 
			response_code = $4, delivered_at = $5, attempts = attempts + 1
		WHERE id = $6`,
		status, now, errMsg, respCode, deliveredAt, deliveryID)
	return err
}

// SavePolicy saves federation policy
func (r *PostgresRepository) SavePolicy(ctx context.Context, policy *FederationPolicy) error {
	allowedJSON, err := json.Marshal(policy.AllowedDomains)
	if err != nil {
		return fmt.Errorf("failed to marshal allowed domains: %w", err)
	}
	blockedJSON, err := json.Marshal(policy.BlockedDomains)
	if err != nil {
		return fmt.Errorf("failed to marshal blocked domains: %w", err)
	}

	query := `
		INSERT INTO federation_policies (
			id, tenant_id, enabled, auto_accept_trust, min_trust_level,
			allowed_domains, blocked_domains, require_encryption, allow_relay,
			max_subscriptions, rate_limit_per_member, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			auto_accept_trust = EXCLUDED.auto_accept_trust,
			min_trust_level = EXCLUDED.min_trust_level,
			allowed_domains = EXCLUDED.allowed_domains,
			blocked_domains = EXCLUDED.blocked_domains,
			require_encryption = EXCLUDED.require_encryption,
			allow_relay = EXCLUDED.allow_relay,
			max_subscriptions = EXCLUDED.max_subscriptions,
			rate_limit_per_member = EXCLUDED.rate_limit_per_member,
			updated_at = EXCLUDED.updated_at`

	_, err = r.db.ExecContext(ctx, query,
		policy.ID, policy.TenantID, policy.Enabled, policy.AutoAcceptTrust,
		policy.MinTrustLevel, allowedJSON, blockedJSON, policy.RequireEncryption,
		policy.AllowRelay, policy.MaxSubscriptions, policy.RateLimitPerMember,
		policy.CreatedAt, policy.UpdatedAt)

	return err
}

// GetPolicy retrieves federation policy
func (r *PostgresRepository) GetPolicy(ctx context.Context, tenantID string) (*FederationPolicy, error) {
	query := `
		SELECT id, tenant_id, enabled, auto_accept_trust, min_trust_level,
			   allowed_domains, blocked_domains, require_encryption, allow_relay,
			   max_subscriptions, rate_limit_per_member, created_at, updated_at
		FROM federation_policies
		WHERE tenant_id = $1`

	var p FederationPolicy
	var allowedJSON, blockedJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Enabled, &p.AutoAcceptTrust, &p.MinTrustLevel,
		&allowedJSON, &blockedJSON, &p.RequireEncryption, &p.AllowRelay,
		&p.MaxSubscriptions, &p.RateLimitPerMember, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(allowedJSON, &p.AllowedDomains); err != nil {
		log.Printf("federation: failed to unmarshal policy allowed domains for tenant %s: %v", tenantID, err)
	}
	if err := json.Unmarshal(blockedJSON, &p.BlockedDomains); err != nil {
		log.Printf("federation: failed to unmarshal policy blocked domains for tenant %s: %v", tenantID, err)
	}

	return &p, nil
}

// SaveKeys saves crypto keys
func (r *PostgresRepository) SaveKeys(ctx context.Context, keys *CryptoKeys) error {
	query := `
		INSERT INTO federation_keys (
			member_id, public_key, algorithm, key_id, created_at, expires_at, revoked
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (member_id) DO UPDATE SET
			public_key = EXCLUDED.public_key,
			algorithm = EXCLUDED.algorithm,
			key_id = EXCLUDED.key_id,
			expires_at = EXCLUDED.expires_at,
			revoked = EXCLUDED.revoked`

	_, err := r.db.ExecContext(ctx, query,
		keys.MemberID, keys.PublicKey, keys.Algorithm, keys.KeyID,
		keys.CreatedAt, keys.ExpiresAt, keys.Revoked)

	return err
}

// GetKeys retrieves keys for a member
func (r *PostgresRepository) GetKeys(ctx context.Context, memberID string) (*CryptoKeys, error) {
	query := `
		SELECT member_id, public_key, algorithm, key_id, created_at, expires_at, revoked
		FROM federation_keys
		WHERE member_id = $1 AND revoked = false`

	var k CryptoKeys
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, memberID).Scan(
		&k.MemberID, &k.PublicKey, &k.Algorithm, &k.KeyID,
		&k.CreatedAt, &expiresAt, &k.Revoked)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("keys not found")
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}

	return &k, nil
}

// GetKeyByID retrieves key by key ID
func (r *PostgresRepository) GetKeyByID(ctx context.Context, keyID string) (*CryptoKeys, error) {
	query := `
		SELECT member_id, public_key, algorithm, key_id, created_at, expires_at, revoked
		FROM federation_keys
		WHERE key_id = $1`

	var k CryptoKeys
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, keyID).Scan(
		&k.MemberID, &k.PublicKey, &k.Algorithm, &k.KeyID,
		&k.CreatedAt, &expiresAt, &k.Revoked)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found")
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}

	return &k, nil
}

// GetMetrics retrieves federation metrics
func (r *PostgresRepository) GetMetrics(ctx context.Context, tenantID string) (*FederationMetrics, error) {
	metrics := &FederationMetrics{TenantID: tenantID, UpdatedAt: time.Now()}

	// Count members
	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM federation_members WHERE tenant_id = $1", tenantID).
		Scan(&metrics.TotalMembers)

	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM federation_members WHERE tenant_id = $1 AND status = 'active'", tenantID).
		Scan(&metrics.ActiveMembers)

	// Count subscriptions
	r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM federation_subscriptions WHERE tenant_id = $1", tenantID).
		Scan(&metrics.TotalSubscriptions)

	// Delivery stats
	r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'succeeded'),
			COUNT(*) FILTER (WHERE status = 'failed')
		FROM federation_deliveries
		WHERE tenant_id = $1`, tenantID).
		Scan(&metrics.TotalDeliveries, &metrics.SuccessfulDeliveries, &metrics.FailedDeliveries)

	// Average latency
	r.db.QueryRowContext(ctx,
		"SELECT COALESCE(AVG(latency), 0) FROM federation_deliveries WHERE tenant_id = $1 AND status = 'succeeded'", tenantID).
		Scan(&metrics.AverageLatency)

	return metrics, nil
}

// UpdateMetrics updates federation metrics (used for caching)
func (r *PostgresRepository) UpdateMetrics(ctx context.Context, metrics *FederationMetrics) error {
	// This would typically update a metrics cache table
	return nil
}

// GenerateMemberID generates a new member ID
func GenerateMemberID() string {
	return uuid.New().String()
}

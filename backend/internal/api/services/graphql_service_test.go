package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Compile-time check that MockGraphQLRepo implements GraphQLRepository.
var _ repository.GraphQLRepository = (*MockGraphQLRepo)(nil)

// --- Mock GraphQLRepository ---
type MockGraphQLRepo struct {
	mock.Mock
}

// Schema operations
func (m *MockGraphQLRepo) CreateSchema(ctx context.Context, schema *models.GraphQLSchema) error {
	return m.Called(ctx, schema).Error(0)
}
func (m *MockGraphQLRepo) GetSchema(ctx context.Context, id uuid.UUID) (*models.GraphQLSchema, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.GraphQLSchema), args.Error(1)
}
func (m *MockGraphQLRepo) GetSchemasByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSchema, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.GraphQLSchema), args.Error(1)
}
func (m *MockGraphQLRepo) UpdateSchema(ctx context.Context, schema *models.GraphQLSchema) error {
	return m.Called(ctx, schema).Error(0)
}
func (m *MockGraphQLRepo) DeleteSchema(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Subscription operations
func (m *MockGraphQLRepo) CreateSubscription(ctx context.Context, sub *models.GraphQLSubscription) error {
	return m.Called(ctx, sub).Error(0)
}
func (m *MockGraphQLRepo) GetSubscription(ctx context.Context, id uuid.UUID) (*models.GraphQLSubscription, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.GraphQLSubscription), args.Error(1)
}
func (m *MockGraphQLRepo) GetSubscriptionsBySchema(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	args := m.Called(ctx, schemaID)
	return args.Get(0).([]*models.GraphQLSubscription), args.Error(1)
}
func (m *MockGraphQLRepo) GetSubscriptionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.GraphQLSubscription), args.Error(1)
}
func (m *MockGraphQLRepo) GetActiveSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.GraphQLSubscription), args.Error(1)
}
func (m *MockGraphQLRepo) UpdateSubscription(ctx context.Context, sub *models.GraphQLSubscription) error {
	return m.Called(ctx, sub).Error(0)
}
func (m *MockGraphQLRepo) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Subscription events
func (m *MockGraphQLRepo) CreateSubscriptionEvent(ctx context.Context, event *models.GraphQLSubscriptionEvent) error {
	return m.Called(ctx, event).Error(0)
}
func (m *MockGraphQLRepo) GetPendingEvents(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]*models.GraphQLSubscriptionEvent, error) {
	args := m.Called(ctx, subscriptionID, limit)
	return args.Get(0).([]*models.GraphQLSubscriptionEvent), args.Error(1)
}
func (m *MockGraphQLRepo) MarkEventDelivered(ctx context.Context, eventID, deliveryID uuid.UUID) error {
	return m.Called(ctx, eventID, deliveryID).Error(0)
}

// Federation sources
func (m *MockGraphQLRepo) AddFederationSource(ctx context.Context, source *models.GraphQLFederationSource) error {
	return m.Called(ctx, source).Error(0)
}
func (m *MockGraphQLRepo) GetFederationSources(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLFederationSource, error) {
	args := m.Called(ctx, schemaID)
	return args.Get(0).([]*models.GraphQLFederationSource), args.Error(1)
}
func (m *MockGraphQLRepo) UpdateFederationSourceHealth(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockGraphQLRepo) DeleteFederationSource(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Type mappings
func (m *MockGraphQLRepo) CreateTypeMapping(ctx context.Context, mapping *models.GraphQLTypeMapping) error {
	return m.Called(ctx, mapping).Error(0)
}
func (m *MockGraphQLRepo) GetTypeMappings(ctx context.Context, schemaID uuid.UUID) ([]*models.GraphQLTypeMapping, error) {
	args := m.Called(ctx, schemaID)
	return args.Get(0).([]*models.GraphQLTypeMapping), args.Error(1)
}
func (m *MockGraphQLRepo) GetTypeMappingByType(ctx context.Context, schemaID uuid.UUID, graphqlType string) (*models.GraphQLTypeMapping, error) {
	args := m.Called(ctx, schemaID, graphqlType)
	return args.Get(0).(*models.GraphQLTypeMapping), args.Error(1)
}
func (m *MockGraphQLRepo) DeleteTypeMapping(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- GraphQL Service Tests ---

func TestGraphQLService_CreateSchema_ValidSDL(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	tenantID := uuid.New()
	req := &models.CreateGraphQLSchemaRequest{
		Name:      "Test Schema",
		SchemaSDL: "type Query { hello: String }",
		Version:   "1.0.0",
	}

	repo.On("CreateSchema", mock.Anything, mock.AnythingOfType("*models.GraphQLSchema")).Return(nil)
	// autoGenerateTypeMappings runs async; allow any calls it might make
	repo.On("CreateTypeMapping", mock.Anything, mock.Anything).Return(nil).Maybe()

	schema, err := svc.CreateSchema(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, schema.TenantID)
	assert.Equal(t, "Test Schema", schema.Name)
	assert.Equal(t, "1.0.0", schema.Version)
	assert.Equal(t, req.SchemaSDL, schema.SchemaSDL)
}

func TestGraphQLService_CreateSchema_EmptySDL(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	_, err := svc.CreateSchema(context.Background(), uuid.New(), &models.CreateGraphQLSchemaRequest{
		Name:      "Empty",
		SchemaSDL: "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schema SDL")
}

func TestGraphQLService_GetSchema_TenantMismatch(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	schemaID := uuid.New()
	ownerTenantID := uuid.New()
	requestingTenantID := uuid.New()

	repo.On("GetSchema", mock.Anything, schemaID).Return(&models.GraphQLSchema{
		ID:       schemaID,
		TenantID: ownerTenantID,
	}, nil)

	_, err := svc.GetSchema(context.Background(), requestingTenantID, schemaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema not found")
}

func TestGraphQLService_ParseSchema_ExtractsAll(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	sdl := `
type User {
  id: ID!
  name: String
}

type Query {
  getUser(id: ID!): User
}

type Mutation {
  createUser(name: String!): User
}

type Subscription {
  onUserCreated: User
}
`

	parsed, err := svc.ParseSchema(context.Background(), sdl)
	require.NoError(t, err)

	// Types: User (Query/Mutation/Subscription are excluded)
	require.Len(t, parsed.Types, 1)
	assert.Equal(t, "User", parsed.Types[0].Name)
	assert.Equal(t, "OBJECT", parsed.Types[0].Kind)
	assert.GreaterOrEqual(t, len(parsed.Types[0].Fields), 2)

	// Queries
	require.Len(t, parsed.Queries, 1)
	assert.Equal(t, "getUser", parsed.Queries[0].Name)
	assert.Equal(t, "User", parsed.Queries[0].ReturnType)

	// Mutations
	require.Len(t, parsed.Mutations, 1)
	assert.Equal(t, "createUser", parsed.Mutations[0].Name)
	assert.Equal(t, "User", parsed.Mutations[0].ReturnType)

	// Subscriptions
	require.Len(t, parsed.Subscriptions, 1)
	assert.Equal(t, "onUserCreated", parsed.Subscriptions[0].Name)
	assert.Equal(t, "User", parsed.Subscriptions[0].ReturnType)
}

func TestGraphQLService_CreateSubscription_ValidQuery(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	tenantID := uuid.New()
	schemaID := uuid.New()
	endpointID := uuid.New()

	repo.On("GetSchema", mock.Anything, schemaID).Return(&models.GraphQLSchema{
		ID:       schemaID,
		TenantID: tenantID,
	}, nil)
	repo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*models.GraphQLSubscription")).Return(nil)

	req := &models.CreateGraphQLSubscriptionRequest{
		SchemaID:          schemaID.String(),
		EndpointID:        endpointID.String(),
		Name:              "Test Subscription",
		SubscriptionQuery: "subscription { onUserCreated { id name } }",
	}

	sub, err := svc.CreateSubscription(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, sub.TenantID)
	assert.Equal(t, schemaID, sub.SchemaID)
	assert.Equal(t, "Test Subscription", sub.Name)
}

func TestGraphQLService_CreateSubscription_InvalidSchemaID(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	req := &models.CreateGraphQLSubscriptionRequest{
		SchemaID:          "not-a-uuid",
		EndpointID:        uuid.New().String(),
		Name:              "Bad Schema",
		SubscriptionQuery: "subscription { x { id } }",
	}

	_, err := svc.CreateSubscription(context.Background(), uuid.New(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schema_id")
}

func TestGraphQLService_ProcessSubscriptionEvent_ActiveSubscription(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	subID := uuid.New()
	tenantID := uuid.New()

	repo.On("GetSubscription", mock.Anything, subID).Return(&models.GraphQLSubscription{
		ID:       subID,
		TenantID: tenantID,
		Status:   models.GraphQLSubscriptionActive,
	}, nil)
	repo.On("CreateSubscriptionEvent", mock.Anything, mock.AnythingOfType("*models.GraphQLSubscriptionEvent")).Return(nil)

	payload := map[string]interface{}{"user": "alice"}
	event, err := svc.ProcessSubscriptionEvent(context.Background(), subID, "user.created", payload)
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, subID, event.SubscriptionID)
	assert.Equal(t, tenantID, event.TenantID)
	assert.Equal(t, "user.created", event.EventType)
	assert.False(t, event.Delivered)
}

func TestGraphQLService_ProcessSubscriptionEvent_InactiveSubscription(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	subID := uuid.New()

	repo.On("GetSubscription", mock.Anything, subID).Return(&models.GraphQLSubscription{
		ID:     subID,
		Status: models.GraphQLSubscriptionPaused,
	}, nil)

	_, err := svc.ProcessSubscriptionEvent(context.Background(), subID, "user.created", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription is not active")
}

func TestGraphQLService_ValidateSchemaSDL_UnbalancedBraces(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	_, err := svc.CreateSchema(context.Background(), uuid.New(), &models.CreateGraphQLSchemaRequest{
		Name:      "Unbalanced",
		SchemaSDL: "type Query { hello: String",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unbalanced braces")
}

func TestGraphQLService_CreateSchema_SchemaNotFoundForOtherTenant(t *testing.T) {
	t.Parallel()
	repo := &MockGraphQLRepo{}
	logger := utils.NewLogger("test")
	svc := NewGraphQLService(repo, logger)

	schemaID := uuid.New()
	repo.On("GetSchema", mock.Anything, schemaID).Return(&models.GraphQLSchema{}, fmt.Errorf("not found"))

	_, err := svc.GetSchema(context.Background(), uuid.New(), schemaID)
	assert.Error(t, err)
}

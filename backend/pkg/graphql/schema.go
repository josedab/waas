package graphql

// Schema defines the GraphQL schema for WAAS
const Schema = `
# Scalars
scalar Time
scalar JSON

# Enums
enum DeliveryStatus {
	PENDING
	DELIVERED
	FAILED
	RETRYING
}

enum SubscriptionTier {
	FREE
	BASIC
	PREMIUM
	ENTERPRISE
}

enum ValidationMode {
	STRICT
	WARN
	NONE
}

# Types
type Tenant {
	id: ID!
	name: String!
	subscriptionTier: SubscriptionTier!
	rateLimitPerMinute: Int!
	monthlyQuota: Int!
	createdAt: Time!
	updatedAt: Time!
	endpoints(first: Int, after: String): EndpointConnection!
}

type Endpoint {
	id: ID!
	url: String!
	isActive: Boolean!
	customHeaders: JSON
	retryConfig: RetryConfig
	createdAt: Time!
	updatedAt: Time!
	deliveries(first: Int, after: String, status: DeliveryStatus): DeliveryConnection!
	schema: EndpointSchema
}

type RetryConfig {
	maxAttempts: Int!
	initialDelayMs: Int!
	maxDelayMs: Int!
	backoffMultiplier: Float!
}

type EndpointSchema {
	schemaId: ID!
	schemaName: String!
	version: String!
	validationMode: ValidationMode!
}

type Delivery {
	id: ID!
	endpointId: ID!
	endpoint: Endpoint
	status: DeliveryStatus!
	attemptCount: Int!
	payload: JSON!
	lastHttpStatus: Int
	lastError: String
	createdAt: Time!
	completedAt: Time
	attempts: [DeliveryAttempt!]!
}

type DeliveryAttempt {
	id: ID!
	attemptNumber: Int!
	httpStatus: Int
	responseBody: String
	errorMessage: String
	createdAt: Time!
}

type Schema {
	id: ID!
	name: String!
	version: String!
	description: String
	jsonSchema: JSON!
	isActive: Boolean!
	isDefault: Boolean!
	createdAt: Time!
	updatedAt: Time!
	versions: [SchemaVersion!]!
}

type SchemaVersion {
	id: ID!
	version: String!
	jsonSchema: JSON!
	changelog: String
	createdAt: Time!
}

# Connections for pagination
type EndpointConnection {
	edges: [EndpointEdge!]!
	pageInfo: PageInfo!
	totalCount: Int!
}

type EndpointEdge {
	node: Endpoint!
	cursor: String!
}

type DeliveryConnection {
	edges: [DeliveryEdge!]!
	pageInfo: PageInfo!
	totalCount: Int!
}

type DeliveryEdge {
	node: Delivery!
	cursor: String!
}

type PageInfo {
	hasNextPage: Boolean!
	hasPreviousPage: Boolean!
	startCursor: String
	endCursor: String
}

# Analytics Types
type DashboardMetrics {
	totalDeliveries: Int!
	successfulDeliveries: Int!
	failedDeliveries: Int!
	successRate: Float!
	avgLatencyMs: Float!
	p95LatencyMs: Float!
	deliveryRatePerMinute: Float!
}

type EndpointMetrics {
	endpointId: ID!
	totalDeliveries: Int!
	successRate: Float!
	avgLatencyMs: Float!
	lastDeliveryAt: Time
}

# Anomaly Types
type Anomaly {
	id: ID!
	metricType: String!
	severity: String!
	currentValue: Float!
	expectedValue: Float!
	deviationPct: Float!
	description: String!
	rootCause: String
	recommendation: String
	status: String!
	detectedAt: Time!
	resolvedAt: Time
}

# Input Types
input CreateEndpointInput {
	url: String!
	customHeaders: JSON
	retryConfig: RetryConfigInput
}

input RetryConfigInput {
	maxAttempts: Int
	initialDelayMs: Int
	maxDelayMs: Int
	backoffMultiplier: Float
}

input UpdateEndpointInput {
	url: String
	isActive: Boolean
	customHeaders: JSON
	retryConfig: RetryConfigInput
}

input SendWebhookInput {
	endpointId: ID!
	payload: JSON!
	headers: JSON
	idempotencyKey: String
}

input CreateSchemaInput {
	name: String!
	version: String!
	description: String
	jsonSchema: JSON!
}

input AssignSchemaInput {
	endpointId: ID!
	schemaId: ID!
	schemaVersion: String
	validationMode: ValidationMode!
}

# Queries
type Query {
	# Tenant
	tenant: Tenant!
	
	# Endpoints
	endpoint(id: ID!): Endpoint
	endpoints(first: Int, after: String): EndpointConnection!
	
	# Deliveries
	delivery(id: ID!): Delivery
	deliveries(
		first: Int
		after: String
		endpointId: ID
		status: DeliveryStatus
	): DeliveryConnection!
	
	# Schemas
	schema(id: ID!): Schema
	schemas(first: Int, after: String): [Schema!]!
	
	# Analytics
	dashboard(startTime: Time, endTime: Time): DashboardMetrics!
	endpointMetrics(endpointId: ID!): EndpointMetrics
	
	# Anomalies
	anomalies(status: String, first: Int, after: String): [Anomaly!]!
}

# Mutations
type Mutation {
	# Endpoint mutations
	createEndpoint(input: CreateEndpointInput!): Endpoint!
	updateEndpoint(id: ID!, input: UpdateEndpointInput!): Endpoint!
	deleteEndpoint(id: ID!): Boolean!
	
	# Webhook sending
	sendWebhook(input: SendWebhookInput!): Delivery!
	replayDelivery(deliveryId: ID!): Delivery!
	
	# Schema mutations
	createSchema(input: CreateSchemaInput!): Schema!
	assignSchema(input: AssignSchemaInput!): Endpoint!
	removeSchema(endpointId: ID!): Endpoint!
	
	# Anomaly mutations
	acknowledgeAnomaly(id: ID!): Anomaly!
	resolveAnomaly(id: ID!): Anomaly!
	
	# API key
	regenerateApiKey: String!
}

# Subscriptions
type Subscription {
	# Real-time delivery updates
	deliveryUpdated(endpointId: ID): Delivery!
	
	# Anomaly alerts
	anomalyDetected: Anomaly!
	
	# Metrics updates
	metricsUpdated: DashboardMetrics!
}
`

// ResolverConfig holds configuration for GraphQL resolvers
type ResolverConfig struct {
	// Add configuration fields as needed
}

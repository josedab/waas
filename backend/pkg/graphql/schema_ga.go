package graphql

// GASchema extends the GraphQL schema with GA features for real-time API
const GASchema = `
# Extended Scalars
scalar DateTime
scalar Duration

# Subscription Types
enum SubscriptionEventType {
	DELIVERY_CREATED
	DELIVERY_SUCCEEDED
	DELIVERY_FAILED
	DELIVERY_RETRYING
	ENDPOINT_HEALTH_CHANGED
	ENDPOINT_CREATED
	ENDPOINT_UPDATED
	ENDPOINT_DELETED
	WORKFLOW_COMPLETED
	WORKFLOW_FAILED
	ALERT_TRIGGERED
}

# Multi-tenant isolation
enum TenantIsolationMode {
	SHARED
	DEDICATED
	FEDERATED
}

# Extended Types
type DeliveryEvent {
	id: ID!
	tenantId: ID!
	endpointId: ID!
	eventType: SubscriptionEventType!
	deliveryId: ID
	status: DeliveryStatus
	payload: JSON
	metadata: JSON
	timestamp: Time!
}

type EndpointHealthEvent {
	endpointId: ID!
	url: String!
	previousStatus: String
	currentStatus: String!
	healthScore: Float!
	latencyMs: Float!
	errorRate: Float!
	timestamp: Time!
}

type AlertEvent {
	id: ID!
	severity: String!
	type: String!
	message: String!
	endpointId: ID
	metadata: JSON
	timestamp: Time!
}

type SubscriptionInfo {
	id: ID!
	type: String!
	query: String!
	active: Boolean!
	eventCount: Int!
	createdAt: Time!
	lastEventAt: Time
}

type TenantConfig {
	id: ID!
	tenantId: ID!
	isolationMode: TenantIsolationMode!
	maxSubscriptions: Int!
	maxConnectionsPerClient: Int!
	rateLimitPerSecond: Int!
	allowedEventTypes: [SubscriptionEventType!]!
	createdAt: Time!
	updatedAt: Time!
}

type SubscriptionStats {
	activeSubscriptions: Int!
	totalEvents: Int!
	eventsPerMinute: Float!
	activeConnections: Int!
	avgLatencyMs: Float!
}

# Input types
input DeliverySubscriptionInput {
	endpointIds: [ID!]
	eventTypes: [SubscriptionEventType!]
	statusFilter: [DeliveryStatus!]
}

input EndpointHealthSubscriptionInput {
	endpointIds: [ID!]
	healthScoreThreshold: Float
}

input AlertSubscriptionInput {
	severities: [String!]
	types: [String!]
	endpointIds: [ID!]
}

input TenantConfigInput {
	isolationMode: TenantIsolationMode
	maxSubscriptions: Int
	maxConnectionsPerClient: Int
	rateLimitPerSecond: Int
	allowedEventTypes: [SubscriptionEventType!]
}

# Extended Query type
extend type Query {
	# Subscription management
	subscriptions: [SubscriptionInfo!]!
	subscription(id: ID!): SubscriptionInfo
	subscriptionStats: SubscriptionStats!

	# Tenant configuration
	tenantConfig: TenantConfig
}

# Extended Mutation type
extend type Mutation {
	# Tenant configuration
	updateTenantConfig(input: TenantConfigInput!): TenantConfig!
}

# Subscription type (real-time)
type Subscription {
	# Delivery events in real-time
	deliveryEvents(input: DeliverySubscriptionInput): DeliveryEvent!

	# Endpoint health changes
	endpointHealth(input: EndpointHealthSubscriptionInput): EndpointHealthEvent!

	# Alert notifications
	alerts(input: AlertSubscriptionInput): AlertEvent!
}
`

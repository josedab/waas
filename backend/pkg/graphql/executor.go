package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QueryExecutor processes GraphQL queries using parsed operation analysis
type QueryExecutor struct {
	resolver *Resolver
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(resolver *Resolver) *QueryExecutor {
	return &QueryExecutor{resolver: resolver}
}

// Execute processes a GraphQL request and returns a response
func (e *QueryExecutor) Execute(ctx context.Context, tenantID string, req GraphQLRequest) GraphQLResponse {
	opType := e.detectOperationType(req.Query)

	switch opType {
	case "query":
		return e.executeQuery(ctx, tenantID, req)
	case "mutation":
		return e.executeMutation(ctx, tenantID, req)
	case "subscription":
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "subscriptions must use WebSocket transport",
				Extensions: map[string]interface{}{"code": "SUBSCRIPTION_OVER_HTTP"},
			}},
		}
	case "introspection":
		return e.executeIntrospection()
	default:
		return GraphQLResponse{
			Errors: []GraphQLError{{Message: "unable to parse query operation type"}},
		}
	}
}

func (e *QueryExecutor) detectOperationType(query string) string {
	q := strings.TrimSpace(query)
	if strings.HasPrefix(q, "mutation") {
		return "mutation"
	}
	if strings.HasPrefix(q, "subscription") {
		return "subscription"
	}
	if strings.Contains(q, "__schema") || strings.Contains(q, "__type") {
		return "introspection"
	}
	return "query"
}

func (e *QueryExecutor) executeQuery(ctx context.Context, tenantID string, req GraphQLRequest) GraphQLResponse {
	fields := extractTopLevelFields(req.Query)
	data := make(map[string]interface{})

	for _, field := range fields {
		switch field {
		case "tenant":
			data["tenant"] = map[string]interface{}{
				"id":                 tenantID,
				"name":               "Tenant " + tenantID,
				"subscriptionTier":   "FREE",
				"rateLimitPerMinute": 100,
				"monthlyQuota":       10000,
				"createdAt":          time.Now().Format(time.RFC3339),
			}
		case "endpoints":
			data["endpoints"] = map[string]interface{}{
				"edges":      []interface{}{},
				"pageInfo":   map[string]interface{}{"hasNextPage": false, "hasPreviousPage": false},
				"totalCount": 0,
			}
		case "endpoint":
			id := extractVariable(req.Variables, "id")
			data["endpoint"] = map[string]interface{}{
				"id":       id,
				"url":      "",
				"isActive": true,
			}
		case "deliveries":
			data["deliveries"] = map[string]interface{}{
				"edges":      []interface{}{},
				"pageInfo":   map[string]interface{}{"hasNextPage": false, "hasPreviousPage": false},
				"totalCount": 0,
			}
		case "delivery":
			id := extractVariable(req.Variables, "id")
			data["delivery"] = map[string]interface{}{
				"id":     id,
				"status": "PENDING",
			}
		case "dashboard":
			data["dashboard"] = map[string]interface{}{
				"totalDeliveries":      0,
				"successfulDeliveries": 0,
				"failedDeliveries":     0,
				"successRate":          0.0,
				"avgLatencyMs":         0.0,
				"p95LatencyMs":         0.0,
				"deliveryRatePerMinute": 0.0,
			}
		case "schemas":
			data["schemas"] = []interface{}{}
		case "anomalies":
			data["anomalies"] = []interface{}{}
		default:
			data[field] = nil
		}
	}

	return GraphQLResponse{Data: data}
}

func (e *QueryExecutor) executeMutation(ctx context.Context, tenantID string, req GraphQLRequest) GraphQLResponse {
	fields := extractTopLevelFields(req.Query)
	data := make(map[string]interface{})

	for _, field := range fields {
		switch field {
		case "createEndpoint":
			input := extractInputVariable(req.Variables, "input")
			data["createEndpoint"] = map[string]interface{}{
				"id":        fmt.Sprintf("ep_%d", time.Now().UnixNano()),
				"url":       input["url"],
				"isActive":  true,
				"createdAt": time.Now().Format(time.RFC3339),
				"updatedAt": time.Now().Format(time.RFC3339),
			}
		case "updateEndpoint":
			id := extractVariable(req.Variables, "id")
			data["updateEndpoint"] = map[string]interface{}{
				"id":        id,
				"isActive":  true,
				"updatedAt": time.Now().Format(time.RFC3339),
			}
		case "deleteEndpoint":
			data["deleteEndpoint"] = true
		case "sendWebhook":
			input := extractInputVariable(req.Variables, "input")
			data["sendWebhook"] = map[string]interface{}{
				"id":           fmt.Sprintf("del_%d", time.Now().UnixNano()),
				"endpointId":   input["endpointId"],
				"status":       "PENDING",
				"attemptCount": 0,
				"payload":      input["payload"],
				"createdAt":    time.Now().Format(time.RFC3339),
			}
		case "replayDelivery":
			data["replayDelivery"] = map[string]interface{}{
				"id":     fmt.Sprintf("del_%d", time.Now().UnixNano()),
				"status": "PENDING",
			}
		case "regenerateApiKey":
			data["regenerateApiKey"] = fmt.Sprintf("wk_%d", time.Now().UnixNano())
		default:
			data[field] = nil
		}
	}

	return GraphQLResponse{Data: data}
}

func (e *QueryExecutor) executeIntrospection() GraphQLResponse {
	types := []map[string]interface{}{
		{"name": "Query", "kind": "OBJECT"},
		{"name": "Mutation", "kind": "OBJECT"},
		{"name": "Subscription", "kind": "OBJECT"},
		{"name": "Tenant", "kind": "OBJECT"},
		{"name": "Endpoint", "kind": "OBJECT"},
		{"name": "Delivery", "kind": "OBJECT"},
		{"name": "DeliveryAttempt", "kind": "OBJECT"},
		{"name": "Schema", "kind": "OBJECT"},
		{"name": "DashboardMetrics", "kind": "OBJECT"},
		{"name": "Anomaly", "kind": "OBJECT"},
		{"name": "DeliveryStatus", "kind": "ENUM"},
		{"name": "SubscriptionTier", "kind": "ENUM"},
		{"name": "ValidationMode", "kind": "ENUM"},
	}

	return GraphQLResponse{
		Data: map[string]interface{}{
			"__schema": map[string]interface{}{
				"types":            types,
				"queryType":        map[string]interface{}{"name": "Query"},
				"mutationType":     map[string]interface{}{"name": "Mutation"},
				"subscriptionType": map[string]interface{}{"name": "Subscription"},
				"directives":       []interface{}{},
			},
		},
	}
}

func extractTopLevelFields(query string) []string {
	var fields []string
	depth := 0
	current := ""
	inField := false

	for _, ch := range query {
		switch ch {
		case '{':
			depth++
			if depth == 2 && !inField {
				inField = true
				current = ""
			}
		case '}':
			depth--
			if depth == 1 && inField {
				inField = false
				if field := strings.TrimSpace(current); field != "" {
					name := strings.SplitN(field, "(", 2)[0]
					name = strings.SplitN(name, "{", 2)[0]
					name = strings.SplitN(name, ":", 2)[0]
					name = strings.TrimSpace(name)
					if name != "" && !strings.HasPrefix(name, "#") {
						fields = append(fields, name)
					}
				}
			}
		case '\n', '\r':
			if depth == 1 && inField {
				if field := strings.TrimSpace(current); field != "" {
					name := strings.SplitN(field, "(", 2)[0]
					name = strings.SplitN(name, "{", 2)[0]
					name = strings.SplitN(name, ":", 2)[0]
					name = strings.TrimSpace(name)
					if name != "" && !strings.HasPrefix(name, "#") {
						fields = append(fields, name)
					}
				}
				current = ""
			}
		default:
			if depth >= 1 {
				current += string(ch)
			}
		}
	}

	return fields
}

func extractVariable(vars map[string]interface{}, key string) string {
	if vars == nil {
		return ""
	}
	if v, ok := vars[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func extractInputVariable(vars map[string]interface{}, key string) map[string]interface{} {
	if vars == nil {
		return map[string]interface{}{}
	}
	if v, ok := vars[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
		// Try JSON conversion
		b, _ := json.Marshal(v)
		var m map[string]interface{}
		json.Unmarshal(b, &m)
		if m != nil {
			return m
		}
	}
	return map[string]interface{}{}
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

const defaultGraphQLJSTimeout = 5 * time.Second

// GraphQLService handles GraphQL subscription to webhook transformations
type GraphQLService struct {
	repo   repository.GraphQLRepository
	logger *utils.Logger
}

// NewGraphQLService creates a new GraphQL service
func NewGraphQLService(repo repository.GraphQLRepository, logger *utils.Logger) *GraphQLService {
	return &GraphQLService{
		repo:   repo,
		logger: logger,
	}
}

// CreateSchema creates a new GraphQL schema
func (s *GraphQLService) CreateSchema(ctx context.Context, tenantID uuid.UUID, req *models.CreateGraphQLSchemaRequest) (*models.GraphQLSchema, error) {
	// Validate schema SDL
	if err := s.validateSchemaSDL(req.SchemaSDL); err != nil {
		return nil, fmt.Errorf("invalid schema SDL: %w", err)
	}

	schema := &models.GraphQLSchema{
		TenantID:              tenantID,
		Name:                  req.Name,
		Description:           req.Description,
		SchemaSDL:             req.SchemaSDL,
		Version:               req.Version,
		IntrospectionEndpoint: req.IntrospectionEndpoint,
		FederationEnabled:     req.FederationEnabled,
	}

	if schema.Version == "" {
		schema.Version = "1.0.0"
	}

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Auto-generate type mappings from subscription types
	go s.autoGenerateTypeMappings(context.Background(), schema)

	return schema, nil
}

// GetSchema retrieves a GraphQL schema
func (s *GraphQLService) GetSchema(ctx context.Context, tenantID, schemaID uuid.UUID) (*models.GraphQLSchema, error) {
	schema, err := s.repo.GetSchema(ctx, schemaID)
	if err != nil {
		return nil, err
	}

	if schema.TenantID != tenantID {
		return nil, fmt.Errorf("schema not found")
	}

	return schema, nil
}

// GetSchemas retrieves all schemas for a tenant
func (s *GraphQLService) GetSchemas(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSchema, error) {
	return s.repo.GetSchemasByTenant(ctx, tenantID)
}

// ParseSchema parses a schema SDL and extracts types, queries, mutations, subscriptions
func (s *GraphQLService) ParseSchema(ctx context.Context, schemaSDL string) (*models.GraphQLParsedSchema, error) {
	parsed := &models.GraphQLParsedSchema{
		Types:         []models.GraphQLTypeInfo{},
		Queries:       []models.GraphQLOperationInfo{},
		Mutations:     []models.GraphQLOperationInfo{},
		Subscriptions: []models.GraphQLSubscriptionInfo{},
	}

	// Parse types
	typeRegex := regexp.MustCompile(`type\s+(\w+)\s*\{([^}]+)\}`)
	typeMatches := typeRegex.FindAllStringSubmatch(schemaSDL, -1)

	for _, match := range typeMatches {
		typeName := match[1]
		fieldsStr := match[2]

		// Skip built-in root types for type listing
		if typeName == "Query" || typeName == "Mutation" || typeName == "Subscription" {
			continue
		}

		typeInfo := models.GraphQLTypeInfo{
			Name:   typeName,
			Kind:   "OBJECT",
			Fields: s.parseFields(fieldsStr),
		}
		parsed.Types = append(parsed.Types, typeInfo)
	}

	// Parse subscriptions
	subscriptionRegex := regexp.MustCompile(`type\s+Subscription\s*\{([^}]+)\}`)
	if subMatch := subscriptionRegex.FindStringSubmatch(schemaSDL); len(subMatch) > 1 {
		fieldRegex := regexp.MustCompile(`(\w+)(?:\(([^)]*)\))?\s*:\s*(\w+!?)`)
		fieldMatches := fieldRegex.FindAllStringSubmatch(subMatch[1], -1)

		for _, fm := range fieldMatches {
			subInfo := models.GraphQLSubscriptionInfo{
				Name:       fm[1],
				Arguments:  s.parseArguments(fm[2]),
				ReturnType: fm[3],
			}
			parsed.Subscriptions = append(parsed.Subscriptions, subInfo)
		}
	}

	// Parse queries
	queryRegex := regexp.MustCompile(`type\s+Query\s*\{([^}]+)\}`)
	if queryMatch := queryRegex.FindStringSubmatch(schemaSDL); len(queryMatch) > 1 {
		fieldRegex := regexp.MustCompile(`(\w+)(?:\(([^)]*)\))?\s*:\s*(\w+!?)`)
		fieldMatches := fieldRegex.FindAllStringSubmatch(queryMatch[1], -1)

		for _, fm := range fieldMatches {
			parsed.Queries = append(parsed.Queries, models.GraphQLOperationInfo{
				Name:       fm[1],
				Arguments:  s.parseArguments(fm[2]),
				ReturnType: fm[3],
			})
		}
	}

	// Parse mutations
	mutationRegex := regexp.MustCompile(`type\s+Mutation\s*\{([^}]+)\}`)
	if mutMatch := mutationRegex.FindStringSubmatch(schemaSDL); len(mutMatch) > 1 {
		fieldRegex := regexp.MustCompile(`(\w+)(?:\(([^)]*)\))?\s*:\s*(\w+!?)`)
		fieldMatches := fieldRegex.FindAllStringSubmatch(mutMatch[1], -1)

		for _, fm := range fieldMatches {
			parsed.Mutations = append(parsed.Mutations, models.GraphQLOperationInfo{
				Name:       fm[1],
				Arguments:  s.parseArguments(fm[2]),
				ReturnType: fm[3],
			})
		}
	}

	return parsed, nil
}

func (s *GraphQLService) parseFields(fieldsStr string) []models.GraphQLFieldInfo {
	var fields []models.GraphQLFieldInfo
	fieldRegex := regexp.MustCompile(`(\w+)\s*:\s*(\[?\w+!?\]?!?)`)
	matches := fieldRegex.FindAllStringSubmatch(fieldsStr, -1)

	for _, m := range matches {
		fields = append(fields, models.GraphQLFieldInfo{
			Name: m[1],
			Type: m[2],
		})
	}

	return fields
}

func (s *GraphQLService) parseArguments(argsStr string) []models.GraphQLArgumentInfo {
	if argsStr == "" {
		return nil
	}

	var args []models.GraphQLArgumentInfo
	argRegex := regexp.MustCompile(`(\w+)\s*:\s*(\w+!?)`)
	matches := argRegex.FindAllStringSubmatch(argsStr, -1)

	for _, m := range matches {
		args = append(args, models.GraphQLArgumentInfo{
			Name: m[1],
			Type: m[2],
		})
	}

	return args
}

// validateSchemaSDL performs basic validation on GraphQL schema SDL
func (s *GraphQLService) validateSchemaSDL(sdl string) error {
	if strings.TrimSpace(sdl) == "" {
		return fmt.Errorf("schema SDL cannot be empty")
	}

	// Check for basic type definitions
	if !strings.Contains(sdl, "type ") {
		return fmt.Errorf("schema must contain at least one type definition")
	}

	// Check for balanced braces
	openBraces := strings.Count(sdl, "{")
	closeBraces := strings.Count(sdl, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces in schema")
	}

	return nil
}

// autoGenerateTypeMappings creates type mappings from subscription return types
func (s *GraphQLService) autoGenerateTypeMappings(ctx context.Context, schema *models.GraphQLSchema) {
	parsed, err := s.ParseSchema(ctx, schema.SchemaSDL)
	if err != nil {
		s.logger.Error("Failed to parse schema for auto-mapping", map[string]interface{}{"error": err})
		return
	}

	for _, sub := range parsed.Subscriptions {
		// Convert camelCase to snake_case for event type
		eventType := s.toSnakeCase(sub.Name)

		mapping := &models.GraphQLTypeMapping{
			SchemaID:         schema.ID,
			TenantID:         schema.TenantID,
			GraphQLType:      sub.ReturnType,
			WebhookEventType: eventType,
			FieldMappings:    make(map[string]string),
			AutoGenerated:    true,
		}

		if err := s.repo.CreateTypeMapping(ctx, mapping); err != nil {
			s.logger.Warn("Failed to create auto type mapping", map[string]interface{}{"error": err, "type": sub.ReturnType})
		}
	}
}

func (s *GraphQLService) toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// CreateSubscription creates a new GraphQL subscription to webhook mapping
func (s *GraphQLService) CreateSubscription(ctx context.Context, tenantID uuid.UUID, req *models.CreateGraphQLSubscriptionRequest) (*models.GraphQLSubscription, error) {
	schemaID, err := uuid.Parse(req.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("invalid schema_id")
	}

	endpointID, err := uuid.Parse(req.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint_id")
	}

	// Verify schema exists and belongs to tenant
	schema, err := s.repo.GetSchema(ctx, schemaID)
	if err != nil || schema.TenantID != tenantID {
		return nil, fmt.Errorf("schema not found")
	}

	// Validate subscription query
	if err := s.validateSubscriptionQuery(req.SubscriptionQuery); err != nil {
		return nil, fmt.Errorf("invalid subscription query: %w", err)
	}

	// Validate transform JS if provided
	if req.TransformJS != "" {
		if err := s.validateTransformJS(req.TransformJS); err != nil {
			return nil, fmt.Errorf("invalid transform_js: %w", err)
		}
	}

	sub := &models.GraphQLSubscription{
		TenantID:          tenantID,
		SchemaID:          schemaID,
		EndpointID:        endpointID,
		Name:              req.Name,
		Description:       req.Description,
		SubscriptionQuery: req.SubscriptionQuery,
		Variables:         req.Variables,
		FilterExpression:  req.FilterExpression,
		FieldSelection:    req.FieldSelection,
		TransformJS:       req.TransformJS,
		DeliveryConfig:    req.DeliveryConfig,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub, nil
}

// validateSubscriptionQuery validates a GraphQL subscription query
func (s *GraphQLService) validateSubscriptionQuery(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("subscription query cannot be empty")
	}

	// Must start with subscription keyword
	if !strings.HasPrefix(strings.ToLower(query), "subscription") {
		return fmt.Errorf("query must be a subscription operation")
	}

	// Check balanced braces
	openBraces := strings.Count(query, "{")
	closeBraces := strings.Count(query, "}")
	if openBraces != closeBraces || openBraces == 0 {
		return fmt.Errorf("invalid subscription query structure")
	}

	return nil
}

// validateTransformJS validates JavaScript transformation code
func (s *GraphQLService) validateTransformJS(code string) error {
	vm := goja.New()

	ctx, cancel := context.WithTimeout(context.Background(), defaultGraphQLJSTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		_, err := vm.RunString(fmt.Sprintf(`
			(function() {
				%s
				if (typeof transform !== 'function') {
					throw new Error('transform function not defined');
				}
			})();
		`, code))
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		vm.Interrupt("timeout")
		return fmt.Errorf("validation timeout")
	}
}

// GetSubscription retrieves a GraphQL subscription
func (s *GraphQLService) GetSubscription(ctx context.Context, tenantID, subID uuid.UUID) (*models.GraphQLSubscription, error) {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return nil, err
	}

	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found")
	}

	return sub, nil
}

// GetSubscriptions retrieves all subscriptions for a tenant
func (s *GraphQLService) GetSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]*models.GraphQLSubscription, error) {
	return s.repo.GetSubscriptionsByTenant(ctx, tenantID)
}

// ProcessSubscriptionEvent processes an incoming subscription event
func (s *GraphQLService) ProcessSubscriptionEvent(ctx context.Context, subscriptionID uuid.UUID, eventType string, payload map[string]interface{}) (*models.GraphQLSubscriptionEvent, error) {
	sub, err := s.repo.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	if sub.Status != models.GraphQLSubscriptionActive {
		return nil, fmt.Errorf("subscription is not active")
	}

	// Apply filter if defined
	if sub.FilterExpression != "" {
		match, err := s.evaluateFilter(sub.FilterExpression, payload)
		if err != nil {
			s.logger.Warn("Filter evaluation failed", map[string]interface{}{"error": err, "subscription_id": subscriptionID})
		} else if !match {
			return nil, nil // Event filtered out
		}
	}

	// Apply field selection
	filteredPayload := payload
	if len(sub.FieldSelection) > 0 {
		filteredPayload = s.selectFields(payload, sub.FieldSelection)
	}

	// Apply transformation
	if sub.TransformJS != "" {
		transformed, err := s.applyTransform(sub.TransformJS, filteredPayload)
		if err != nil {
			s.logger.Warn("Transform failed", map[string]interface{}{"error": err, "subscription_id": subscriptionID})
		} else {
			filteredPayload = transformed
		}
	}

	event := &models.GraphQLSubscriptionEvent{
		SubscriptionID:  subscriptionID,
		TenantID:        sub.TenantID,
		EventType:       eventType,
		Payload:         payload,
		FilteredPayload: filteredPayload,
		Delivered:       false,
	}

	if err := s.repo.CreateSubscriptionEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

// evaluateFilter evaluates a JavaScript filter expression
func (s *GraphQLService) evaluateFilter(expression string, payload map[string]interface{}) (bool, error) {
	vm := goja.New()

	payloadJSON, _ := json.Marshal(payload)
	vm.Set("payload", string(payloadJSON))

	ctx, cancel := context.WithTimeout(context.Background(), defaultGraphQLJSTimeout)
	defer cancel()

	type jsResult struct {
		val goja.Value
		err error
	}
	done := make(chan jsResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- jsResult{nil, fmt.Errorf("panic: %v", r)}
			}
		}()
		result, err := vm.RunString(fmt.Sprintf(`
			var data = JSON.parse(payload);
			(function(data) { return %s; })(data);
		`, expression))
		done <- jsResult{result, err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			return false, res.err
		}
		return res.val.ToBoolean(), nil
	case <-ctx.Done():
		vm.Interrupt("timeout")
		return false, fmt.Errorf("filter evaluation timeout")
	}
}

// selectFields selects specific fields from payload
func (s *GraphQLService) selectFields(payload map[string]interface{}, fields []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, field := range fields {
		parts := strings.Split(field, ".")
		val := s.getNestedValue(payload, parts)
		if val != nil {
			s.setNestedValue(result, parts, val)
		}
	}

	return result
}

func (s *GraphQLService) getNestedValue(data map[string]interface{}, path []string) interface{} {
	current := interface{}(data)

	for _, key := range path {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return nil
		}
	}

	return current
}

func (s *GraphQLService) setNestedValue(data map[string]interface{}, path []string, value interface{}) {
	current := data

	for i, key := range path {
		if i == len(path)-1 {
			current[key] = value
		} else {
			if _, ok := current[key]; !ok {
				current[key] = make(map[string]interface{})
			}
			current = current[key].(map[string]interface{})
		}
	}
}

// applyTransform applies JavaScript transformation to payload
func (s *GraphQLService) applyTransform(code string, payload map[string]interface{}) (map[string]interface{}, error) {
	vm := goja.New()

	payloadJSON, _ := json.Marshal(payload)
	vm.Set("_input", string(payloadJSON))

	ctx, cancel := context.WithTimeout(context.Background(), defaultGraphQLJSTimeout)
	defer cancel()

	type jsResult struct {
		val goja.Value
		err error
	}
	done := make(chan jsResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- jsResult{nil, fmt.Errorf("panic: %v", r)}
			}
		}()
		result, err := vm.RunString(fmt.Sprintf(`
			%s
			var input = JSON.parse(_input);
			JSON.stringify(transform(input));
		`, code))
		done <- jsResult{result, err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			return nil, res.err
		}
		var transformed map[string]interface{}
		if err := json.Unmarshal([]byte(res.val.String()), &transformed); err != nil {
			return nil, err
		}
		return transformed, nil
	case <-ctx.Done():
		vm.Interrupt("timeout")
		return nil, fmt.Errorf("transform execution timeout")
	}
}

// AddFederationSource adds a federation source to a schema
func (s *GraphQLService) AddFederationSource(ctx context.Context, tenantID uuid.UUID, req *models.AddFederationSourceRequest) (*models.GraphQLFederationSource, error) {
	schemaID, err := uuid.Parse(req.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("invalid schema_id")
	}

	schema, err := s.repo.GetSchema(ctx, schemaID)
	if err != nil || schema.TenantID != tenantID {
		return nil, fmt.Errorf("schema not found")
	}

	if !schema.FederationEnabled {
		return nil, fmt.Errorf("federation not enabled for this schema")
	}

	source := &models.GraphQLFederationSource{
		SchemaID:    schemaID,
		TenantID:    tenantID,
		Name:        req.Name,
		EndpointURL: req.EndpointURL,
		AuthConfig:  req.AuthConfig,
	}

	if err := s.repo.AddFederationSource(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to add federation source: %w", err)
	}

	// Trigger async health check
	go s.checkFederationSourceHealth(context.Background(), source)

	return source, nil
}

// checkFederationSourceHealth checks the health of a federation source
func (s *GraphQLService) checkFederationSourceHealth(ctx context.Context, source *models.GraphQLFederationSource) {
	// In production, would actually call the endpoint
	status := models.FederationHealthHealthy

	if err := s.repo.UpdateFederationSourceHealth(ctx, source.ID, status); err != nil {
		s.logger.Error("Failed to update federation source health", map[string]interface{}{"error": err})
	}
}

// GetFederationSources retrieves federation sources for a schema
func (s *GraphQLService) GetFederationSources(ctx context.Context, tenantID, schemaID uuid.UUID) ([]*models.GraphQLFederationSource, error) {
	schema, err := s.repo.GetSchema(ctx, schemaID)
	if err != nil || schema.TenantID != tenantID {
		return nil, fmt.Errorf("schema not found")
	}

	return s.repo.GetFederationSources(ctx, schemaID)
}

// CreateTypeMapping creates a manual type mapping
func (s *GraphQLService) CreateTypeMapping(ctx context.Context, tenantID uuid.UUID, req *models.CreateTypeMappingRequest) (*models.GraphQLTypeMapping, error) {
	schemaID, err := uuid.Parse(req.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("invalid schema_id")
	}

	schema, err := s.repo.GetSchema(ctx, schemaID)
	if err != nil || schema.TenantID != tenantID {
		return nil, fmt.Errorf("schema not found")
	}

	mapping := &models.GraphQLTypeMapping{
		SchemaID:         schemaID,
		TenantID:         tenantID,
		GraphQLType:      req.GraphQLType,
		WebhookEventType: req.WebhookEventType,
		FieldMappings:    req.FieldMappings,
		AutoGenerated:    false,
	}

	if err := s.repo.CreateTypeMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to create type mapping: %w", err)
	}

	return mapping, nil
}

// GetTypeMappings retrieves type mappings for a schema
func (s *GraphQLService) GetTypeMappings(ctx context.Context, tenantID, schemaID uuid.UUID) ([]*models.GraphQLTypeMapping, error) {
	schema, err := s.repo.GetSchema(ctx, schemaID)
	if err != nil || schema.TenantID != tenantID {
		return nil, fmt.Errorf("schema not found")
	}

	return s.repo.GetTypeMappings(ctx, schemaID)
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
)

// GraphQLHandler handles GraphQL gateway endpoints
type GraphQLHandler struct {
	service *services.GraphQLService
	logger  *utils.Logger
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(service *services.GraphQLService, logger *utils.Logger) *GraphQLHandler {
	return &GraphQLHandler{
		service: service,
		logger:  logger,
	}
}

// CreateSchema creates a new GraphQL schema
// @Summary Create GraphQL schema
// @Tags GraphQL Gateway
// @Accept json
// @Produce json
// @Param request body models.CreateGraphQLSchemaRequest true "Schema definition"
// @Success 201 {object} models.GraphQLSchema
// @Router /graphql/schemas [post]
func (h *GraphQLHandler) CreateSchema(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var req models.CreateGraphQLSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schema, err := h.service.CreateSchema(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to create schema", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schema)
}

// GetSchemas retrieves all GraphQL schemas for the tenant
// @Summary List GraphQL schemas
// @Tags GraphQL Gateway
// @Produce json
// @Success 200 {array} models.GraphQLSchema
// @Router /graphql/schemas [get]
func (h *GraphQLHandler) GetSchemas(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	schemas, err := h.service.GetSchemas(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		h.logger.Error("Failed to get schemas", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schemas": schemas})
}

// GetSchema retrieves a specific GraphQL schema
// @Summary Get GraphQL schema
// @Tags GraphQL Gateway
// @Produce json
// @Param schema_id path string true "Schema ID"
// @Success 200 {object} models.GraphQLSchema
// @Router /graphql/schemas/{schema_id} [get]
func (h *GraphQLHandler) GetSchema(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	schemaID, err := uuid.Parse(c.Param("schema_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schema_id"})
		return
	}

	schema, err := h.service.GetSchema(c.Request.Context(), tenantID.(uuid.UUID), schemaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schema not found"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// ParseSchema parses a GraphQL schema SDL and returns structure
// @Summary Parse GraphQL schema
// @Tags GraphQL Gateway
// @Accept json
// @Produce json
// @Param request body map[string]string true "Schema SDL"
// @Success 200 {object} models.GraphQLParsedSchema
// @Router /graphql/parse [post]
func (h *GraphQLHandler) ParseSchema(c *gin.Context) {
	var req struct {
		SchemaSDL string `json:"schema_sdl" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsed, err := h.service.ParseSchema(c.Request.Context(), req.SchemaSDL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, parsed)
}

// CreateSubscription creates a new GraphQL subscription to webhook mapping
// @Summary Create GraphQL subscription
// @Tags GraphQL Gateway
// @Accept json
// @Produce json
// @Param request body models.CreateGraphQLSubscriptionRequest true "Subscription definition"
// @Success 201 {object} models.GraphQLSubscription
// @Router /graphql/subscriptions [post]
func (h *GraphQLHandler) CreateSubscription(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var req models.CreateGraphQLSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.CreateSubscription(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to create subscription", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// GetSubscriptions retrieves all GraphQL subscriptions for the tenant
// @Summary List GraphQL subscriptions
// @Tags GraphQL Gateway
// @Produce json
// @Success 200 {array} models.GraphQLSubscription
// @Router /graphql/subscriptions [get]
func (h *GraphQLHandler) GetSubscriptions(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	subs, err := h.service.GetSubscriptions(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		h.logger.Error("Failed to get subscriptions", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscriptions": subs})
}

// GetSubscription retrieves a specific GraphQL subscription
// @Summary Get GraphQL subscription
// @Tags GraphQL Gateway
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} models.GraphQLSubscription
// @Router /graphql/subscriptions/{subscription_id} [get]
func (h *GraphQLHandler) GetSubscription(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	subID, err := uuid.Parse(c.Param("subscription_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription_id"})
		return
	}

	sub, err := h.service.GetSubscription(c.Request.Context(), tenantID.(uuid.UUID), subID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// IngestEvent ingests an event for a subscription
// @Summary Ingest subscription event
// @Tags GraphQL Gateway
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Param request body map[string]interface{} true "Event payload"
// @Success 202 {object} models.GraphQLSubscriptionEvent
// @Router /graphql/subscriptions/{subscription_id}/events [post]
func (h *GraphQLHandler) IngestEvent(c *gin.Context) {
	subID, err := uuid.Parse(c.Param("subscription_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription_id"})
		return
	}

	var req struct {
		EventType string                 `json:"event_type" binding:"required"`
		Payload   map[string]interface{} `json:"payload" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.service.ProcessSubscriptionEvent(c.Request.Context(), subID, req.EventType, req.Payload)
	if err != nil {
		h.logger.Error("Failed to process event", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if event == nil {
		c.JSON(http.StatusOK, gin.H{"message": "event filtered out"})
		return
	}

	c.JSON(http.StatusAccepted, event)
}

// AddFederationSource adds a federation source to a schema
// @Summary Add federation source
// @Tags GraphQL Gateway
// @Accept json
// @Produce json
// @Param request body models.AddFederationSourceRequest true "Federation source"
// @Success 201 {object} models.GraphQLFederationSource
// @Router /graphql/federation/sources [post]
func (h *GraphQLHandler) AddFederationSource(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var req models.AddFederationSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source, err := h.service.AddFederationSource(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to add federation source", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, source)
}

// GetFederationSources retrieves federation sources for a schema
// @Summary List federation sources
// @Tags GraphQL Gateway
// @Produce json
// @Param schema_id path string true "Schema ID"
// @Success 200 {array} models.GraphQLFederationSource
// @Router /graphql/schemas/{schema_id}/federation/sources [get]
func (h *GraphQLHandler) GetFederationSources(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	schemaID, err := uuid.Parse(c.Param("schema_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schema_id"})
		return
	}

	sources, err := h.service.GetFederationSources(c.Request.Context(), tenantID.(uuid.UUID), schemaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// CreateTypeMapping creates a type mapping for a schema
// @Summary Create type mapping
// @Tags GraphQL Gateway
// @Accept json
// @Produce json
// @Param request body models.CreateTypeMappingRequest true "Type mapping"
// @Success 201 {object} models.GraphQLTypeMapping
// @Router /graphql/type-mappings [post]
func (h *GraphQLHandler) CreateTypeMapping(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var req models.CreateTypeMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mapping, err := h.service.CreateTypeMapping(c.Request.Context(), tenantID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error("Failed to create type mapping", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, mapping)
}

// GetTypeMappings retrieves type mappings for a schema
// @Summary List type mappings
// @Tags GraphQL Gateway
// @Produce json
// @Param schema_id path string true "Schema ID"
// @Success 200 {array} models.GraphQLTypeMapping
// @Router /graphql/schemas/{schema_id}/type-mappings [get]
func (h *GraphQLHandler) GetTypeMappings(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	schemaID, err := uuid.Parse(c.Param("schema_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schema_id"})
		return
	}

	mappings, err := h.service.GetTypeMappings(c.Request.Context(), tenantID.(uuid.UUID), schemaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mappings": mappings})
}

package api

import (
	"github.com/josedab/waas/pkg/costing"
	"github.com/josedab/waas/pkg/embed"
	"github.com/josedab/waas/pkg/flow"
	"github.com/josedab/waas/pkg/georouting"
	"github.com/josedab/waas/pkg/metaevents"
	"github.com/josedab/waas/pkg/mocking"
	"github.com/josedab/waas/pkg/otel"
	"github.com/josedab/waas/pkg/protocols"
	"github.com/josedab/waas/pkg/schema"

	"github.com/gin-gonic/gin"
)

// registerCoreRoutes registers core feature routes (v1): schema, flow,
// meta-events, geo-routing, embed, mocking, costing, OTel, protocols.
func (s *Server) registerCoreRoutes(protected *gin.RouterGroup) {
	// Schema Registry
	schemaRepo := schema.NewPostgresRepository(s.sqlxDB)
	schemaService := schema.NewService(schemaRepo)
	schemaHandler := schema.NewHandler(schemaService)
	schemaHandler.RegisterRoutes(protected)

	// Visual Flow Builder
	flowHandler := flow.NewHandler(s.flowService)
	flowHandler.RegisterRoutes(protected)

	// Meta-Events (Webhooks for Webhooks)
	metaHandler := metaevents.NewHandler(s.metaService)
	metaHandler.RegisterRoutes(protected)

	// Geographic Routing
	geoHandler := georouting.NewHandler(s.geoService)
	geoHandler.RegisterRoutes(protected)

	// Embedded Analytics SDK
	embedHandler := embed.NewHandler(s.embedService)
	embedHandler.RegisterRoutes(protected)

	// Webhook Mocking Service
	mockHandler := mocking.NewHandler(s.mockService)
	mockHandler.RegisterRoutes(protected)

	// Cost Attribution Dashboard
	costHandler := costing.NewHandler(s.costService)
	costHandler.RegisterRoutes(protected)

	// OpenTelemetry Integration
	otelHandler := otel.NewHandler(s.otelService)
	otelHandler.RegisterRoutes(protected)

	// Custom Webhook Protocols
	protocolHandler := protocols.NewHandler(s.protocolService)
	protocolHandler.RegisterRoutes(protected)
}

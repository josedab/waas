package api

import (
	"github.com/josedab/waas/pkg/analyticsembed"
	"github.com/josedab/waas/pkg/autoremediation"
	"github.com/josedab/waas/pkg/blockchain"
	"github.com/josedab/waas/pkg/callback"
	"github.com/josedab/waas/pkg/canary"
	"github.com/josedab/waas/pkg/catalog"
	"github.com/josedab/waas/pkg/cloudmanaged"
	"github.com/josedab/waas/pkg/collabdebug"
	"github.com/josedab/waas/pkg/compliancecenter"
	"github.com/josedab/waas/pkg/costengine"
	"github.com/josedab/waas/pkg/dlq"
	"github.com/josedab/waas/pkg/docgen"
	"github.com/josedab/waas/pkg/edge"
	"github.com/josedab/waas/pkg/fanout"
	"github.com/josedab/waas/pkg/flowbuilder"
	"github.com/josedab/waas/pkg/gitops"
	"github.com/josedab/waas/pkg/graphqlsub"
	"github.com/josedab/waas/pkg/inbound"
	"github.com/josedab/waas/pkg/intelligence"
	"github.com/josedab/waas/pkg/livemigration"
	"github.com/josedab/waas/pkg/mobilesdk"
	"github.com/josedab/waas/pkg/monetization"
	"github.com/josedab/waas/pkg/multicloud"
	"github.com/josedab/waas/pkg/openapigen"
	"github.com/josedab/waas/pkg/pipeline"
	"github.com/josedab/waas/pkg/playground"
	"github.com/josedab/waas/pkg/pluginmarket"
	"github.com/josedab/waas/pkg/prediction"
	"github.com/josedab/waas/pkg/protocolgw"
	"github.com/josedab/waas/pkg/remediation"
	"github.com/josedab/waas/pkg/sandbox"
	"github.com/josedab/waas/pkg/schemaregistry"
	"github.com/josedab/waas/pkg/streaming"
	"github.com/josedab/waas/pkg/timetravel"
	"github.com/josedab/waas/pkg/tracing"
	"github.com/josedab/waas/pkg/waf"
	"github.com/josedab/waas/pkg/whitelabel"

	"github.com/gin-gonic/gin"
)

// registerObservabilityRoutes registers delivery/edge (v3), extended
// platform (v5), inbound/fanout (v6), and ecosystem (v7) feature routes.
func (s *Server) registerObservabilityRoutes(protected *gin.RouterGroup) {
	// ── Delivery & edge services (v3) ───────────────────────────────

	// Event Streaming Bridge
	streamingHandler := streaming.NewHandler(s.streamingService)
	streamingHandler.RegisterRoutes(protected)

	// AI Auto-Remediation
	remediationHandler := remediation.NewHandler(s.remediationService)
	remediationHandler.RegisterRoutes(protected)

	// Edge Function Runtime
	edgeHandler := edge.NewHandler(s.edgeService)
	edgeHandler.RegisterRoutes(protected)

	// Smart Contract Triggers
	blockchainHandler := blockchain.NewHandler(s.blockchainService)
	blockchainHandler.RegisterRoutes(protected)

	// Compliance Center
	complianceHandler := compliancecenter.NewHandler(s.complianceService)
	complianceHandler.RegisterRoutes(protected)

	// Predictive Failure Prevention
	predictionHandler := prediction.NewHandler(s.predictionService)
	predictionHandler.RegisterRoutes(protected)

	// GraphQL Subscriptions Gateway
	graphqlsubHandler := graphqlsub.NewHandler(s.graphqlsubService)
	graphqlsubHandler.RegisterRoutes(protected)

	// Webhook Monetization Platform
	monetizationHandler := monetization.NewHandler(s.monetizationService)
	monetizationHandler.RegisterRoutes(protected)

	// Multi-Cloud Federation
	multicloudHandler := multicloud.NewFederationHandler(s.multicloudService)
	multicloudHandler.RegisterFederationRoutes(protected)

	// ── Extended platform services (v5) ─────────────────────────────

	// OpenTelemetry Distributed Tracing
	tracingHandler := tracing.NewHandler(s.tracingService)
	tracingHandler.RegisterRoutes(protected)

	// Webhook Canary Deployments
	canaryHandler := canary.NewHandler(s.canaryService)
	canaryHandler.RegisterRoutes(protected)

	// AI Auto-Remediation
	autoremediationHandler := autoremediation.NewHandler(s.autoremediationService)
	autoremediationHandler.RegisterRoutes(protected)

	// Event Schema Registry
	schemaregistryHandler := schemaregistry.NewHandler(s.schemaregistryService)
	schemaregistryHandler.RegisterRoutes(protected)

	// Event Catalog & Schema Registry
	catalogHandler := catalog.NewHandler(s.catalogService)
	catalogHandler.RegisterRoutes(protected)

	// Webhook Replay Sandbox
	sandboxHandler := sandbox.NewHandler(s.sandboxService)
	sandboxHandler.RegisterRoutes(protected)

	// Multi-Protocol Gateway
	protocolgwHandler := protocolgw.NewHandler(s.protocolgwService)
	protocolgwHandler.RegisterRoutes(protected)

	// Embeddable Analytics SDK
	analyticsembedHandler := analyticsembed.NewHandler(s.analyticsembedService)
	analyticsembedHandler.RegisterRoutes(protected)

	// Cost Attribution Engine
	costengineHandler := costengine.NewHandler(s.costengineService)
	costengineHandler.RegisterRoutes(protected)

	// GitOps Configuration Management
	gitopsHandler := gitops.NewHandler(s.gitopsService)
	gitopsHandler.RegisterRoutes(protected)

	// Live Migration Toolkit
	livemigrationHandler := livemigration.NewHandler(s.livemigrationService)
	livemigrationHandler.RegisterRoutes(protected)
	livemigrationHandler.RegisterImporterRoutes(protected)

	// ── Inbound & fan-out (v6) ──────────────────────────────────────

	// Inbound Webhook Gateway (management)
	inboundHandler := inbound.NewHandler(s.inboundService)
	inboundHandler.RegisterRoutes(protected)

	// Fan-Out & Topic-Based Routing
	fanoutHandler := fanout.NewHandler(s.fanoutService)
	fanoutHandler.RegisterRoutes(protected)

	// Mobile SDK Management
	mobilesdkHandler := mobilesdk.NewHandler(s.mobilesdkService)
	mobilesdkHandler.RegisterRoutes(protected)

	// ── Ecosystem services (v7) ─────────────────────────────────────

	// Webhook Plugin Marketplace
	pluginmarketHandler := pluginmarket.NewHandler(s.pluginmarketService)
	pluginmarketHandler.RegisterRoutes(protected)
	pluginmarketHandler.RegisterMarketplaceRoutes(protected)

	// AI-Powered Webhook Intelligence
	intelligenceHandler := intelligence.NewHandler(s.intelligenceService)
	intelligenceHandler.RegisterRoutes(protected)

	// Visual Webhook Workflow Builder
	flowbuilderHandler := flowbuilder.NewHandler(s.flowbuilderService)
	flowbuilderHandler.RegisterRoutes(protected)

	// Webhook Replay & Time Travel
	timetravelHandler := timetravel.NewHandler(s.timetravelService)
	timetravelHandler.RegisterRoutes(protected)

	// Managed Cloud Offering
	cloudmanagedHandler := cloudmanaged.NewHandler(s.cloudmanagedService)
	cloudmanagedHandler.RegisterRoutes(protected)

	// Bi-Directional Webhooks & Callbacks
	callbackHandler := callback.NewHandler(s.callbackService)
	callbackHandler.RegisterRoutes(protected)

	// Real-Time Collaborative Debugging
	collabdebugHandler := collabdebug.NewHandler(s.collabdebugService)
	collabdebugHandler.RegisterRoutes(protected)

	// Webhook Security Scanner & WAF
	wafHandler := waf.NewHandler(s.wafService)
	wafHandler.RegisterRoutes(protected)

	// API-First Documentation Generator
	docgenHandler := docgen.NewHandler(s.docgenService)
	docgenHandler.RegisterRoutes(protected)

	// Multi-Tenant Whitelabel
	whitelabelHandler := whitelabel.NewHandler(s.whitelabelService)
	whitelabelHandler.RegisterRoutes(protected)

	// DLQ & Observability Dashboard
	dlqHandler := dlq.NewHandler(s.dlqService)
	dlqHandler.RegisterRoutes(protected)

	// OpenAPI-to-Webhook Generator
	openapigenHandler := openapigen.NewHandler(s.openapigenService)
	openapigenHandler.RegisterRoutes(protected)

	// Interactive Playground v2
	playgroundHandler := playground.NewHandler(s.playgroundService)
	playgroundHandler.RegisterRoutes(protected)

	// Delivery Pipeline Composition
	pipelineHandler := pipeline.NewHandler(s.pipelineService)
	pipelineHandler.RegisterRoutes(protected)
}

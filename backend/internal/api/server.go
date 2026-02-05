package api

import (
	"fmt"

	_ "github.com/josedab/waas/docs"
	"github.com/josedab/waas/internal/api/handlers"
	"github.com/josedab/waas/pkg/analyticsembed"
	"github.com/josedab/waas/pkg/auth"
	"github.com/josedab/waas/pkg/autoremediation"
	"github.com/josedab/waas/pkg/billing"
	"github.com/josedab/waas/pkg/blockchain"
	"github.com/josedab/waas/pkg/callback"
	"github.com/josedab/waas/pkg/canary"
	"github.com/josedab/waas/pkg/catalog"
	"github.com/josedab/waas/pkg/cdc"
	"github.com/josedab/waas/pkg/chaos"
	"github.com/josedab/waas/pkg/cloud"
	"github.com/josedab/waas/pkg/cloudctl"
	"github.com/josedab/waas/pkg/cloudmanaged"
	"github.com/josedab/waas/pkg/collabdebug"
	"github.com/josedab/waas/pkg/compliancecenter"
	"github.com/josedab/waas/pkg/contracts"
	"github.com/josedab/waas/pkg/costengine"
	"github.com/josedab/waas/pkg/costing"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/debugger"
	"github.com/josedab/waas/pkg/dlq"
	"github.com/josedab/waas/pkg/docgen"
	"github.com/josedab/waas/pkg/edge"
	"github.com/josedab/waas/pkg/embed"
	"github.com/josedab/waas/pkg/eventmesh"
	"github.com/josedab/waas/pkg/fanout"
	"github.com/josedab/waas/pkg/federation"
	"github.com/josedab/waas/pkg/flow"
	"github.com/josedab/waas/pkg/flowbuilder"
	"github.com/josedab/waas/pkg/georouting"
	"github.com/josedab/waas/pkg/gitops"
	"github.com/josedab/waas/pkg/graphqlsub"
	"github.com/josedab/waas/pkg/inbound"
	"github.com/josedab/waas/pkg/intelligence"
	"github.com/josedab/waas/pkg/livemigration"
	"github.com/josedab/waas/pkg/marketplacetpl"
	"github.com/josedab/waas/pkg/metaevents"
	"github.com/josedab/waas/pkg/metrics"
	"github.com/josedab/waas/pkg/mobilesdk"
	"github.com/josedab/waas/pkg/mocking"
	"github.com/josedab/waas/pkg/monetization"
	"github.com/josedab/waas/pkg/monitoring"
	"github.com/josedab/waas/pkg/mtls"
	"github.com/josedab/waas/pkg/multicloud"
	"github.com/josedab/waas/pkg/observability"
	"github.com/josedab/waas/pkg/openapigen"
	"github.com/josedab/waas/pkg/otel"
	"github.com/josedab/waas/pkg/pipeline"
	"github.com/josedab/waas/pkg/playground"
	"github.com/josedab/waas/pkg/pluginmarket"
	"github.com/josedab/waas/pkg/portal"
	"github.com/josedab/waas/pkg/prediction"
	"github.com/josedab/waas/pkg/protocolgw"
	"github.com/josedab/waas/pkg/protocols"
	"github.com/josedab/waas/pkg/pushbridge"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/remediation"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/sandbox"
	"github.com/josedab/waas/pkg/schema"
	"github.com/josedab/waas/pkg/schemaregistry"
	"github.com/josedab/waas/pkg/signatures"
	"github.com/josedab/waas/pkg/sla"
	"github.com/josedab/waas/pkg/smartlimit"
	"github.com/josedab/waas/pkg/streaming"
	"github.com/josedab/waas/pkg/tfprovider"
	"github.com/josedab/waas/pkg/timetravel"
	"github.com/josedab/waas/pkg/tracing"
	"github.com/josedab/waas/pkg/transform"
	"github.com/josedab/waas/pkg/utils"
	"github.com/josedab/waas/pkg/versioning"
	"github.com/josedab/waas/pkg/waf"
	"github.com/josedab/waas/pkg/whitelabel"
	"github.com/josedab/waas/pkg/workflow"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	router          *gin.Engine
	db              *database.DB
	sqlxDB          *sqlx.DB
	redisClient     *database.RedisClient
	logger          *utils.Logger
	config          *utils.Config
	healthChecker   *monitoring.HealthChecker
	alertManager    *monitoring.AlertManager
	metricsRecorder *monitoring.MetricsRecorder
	tracer          *monitoring.Tracer
	// Next-gen feature services
	flowService     *flow.Service
	metaService     *metaevents.Service
	geoService      *georouting.Service
	embedService    *embed.Service
	mockService     *mocking.Service
	costService     *costing.Service
	otelService     *otel.Service
	protocolService *protocols.Service
	// Next-gen features v2
	observabilityService *observability.Service
	smartlimitService    *smartlimit.Service
	chaosService         *chaos.Service
	cdcService           *cdc.Service
	workflowService      *workflow.Service
	signaturesService    *signatures.Service
	pushbridgeService    *pushbridge.Service
	billingService       *billing.Service
	versioningService    *versioning.Service
	federationService    *federation.Service
	// Next-gen features v3
	streamingService    *streaming.Service
	remediationService  *remediation.Service
	edgeService         *edge.Service
	blockchainService   *blockchain.Service
	complianceService   *compliancecenter.Service
	predictionService   *prediction.Service
	graphqlsubService   *graphqlsub.Service
	monetizationService *monetization.Service
	multicloudService   *multicloud.FederationService
	// Next-gen features v4
	slaService          *sla.Service
	mtlsService         *mtls.Service
	contractsService    *contracts.Service
	marketplaceService  *marketplacetpl.Service
	eventmeshService    *eventmesh.Service
	debuggerService     *debugger.Service
	cloudService        *cloud.BillingService
	cloudTeamService    *cloud.TeamService
	cloudAuditService   *cloud.AuditService
	cloudOnboardService *cloud.OnboardingService
	cloudctlService     *cloudctl.Service
	tfproviderService   *tfprovider.Service
	portalService       *portal.Service
	// Next-gen features v5
	tracingService         *tracing.Service
	canaryService          *canary.Service
	autoremediationService *autoremediation.Service
	schemaregistryService  *schemaregistry.Service
	catalogService         *catalog.Service
	sandboxService         *sandbox.Service
	protocolgwService      *protocolgw.Service
	analyticsembedService  *analyticsembed.Service
	costengineService      *costengine.Service
	gitopsService          *gitops.Service
	livemigrationService   *livemigration.Service
	inboundService         *inbound.Service
	fanoutService          *fanout.Service
	mobilesdkService       *mobilesdk.Service
	// Next-gen features v7
	pluginmarketService *pluginmarket.Service
	intelligenceService *intelligence.Service
	flowbuilderService  *flowbuilder.Service
	timetravelService   *timetravel.Service
	cloudmanagedService *cloudmanaged.Service
	callbackService     *callback.Service
	collabdebugService  *collabdebug.Service
	wafService          *waf.Service
	docgenService       *docgen.Service
	whitelabelService   *whitelabel.Service
	playgroundService   *playground.Service
	pipelineService     *pipeline.Service
	// DLQ & Observability
	dlqService        *dlq.Service
	openapigenService *openapigen.Service
}

// NewServer constructs and wires the entire API server. Initialization
// proceeds in phases:
//
//  1. Infrastructure — config, logging, PostgreSQL (pgx + sqlx), Redis, migrations
//  2. Observability  — health checker, alert manager, metrics recorder, tracer
//  3. Core services  — flow, meta-events, geo-routing, embed, mocking, costing,
//     OTel, protocols (v1)
//  4. Platform services — observability, smart-limit, chaos, CDC, workflow,
//     signatures, push-bridge, billing, versioning, federation (v2)
//  5. Delivery & edge — streaming, remediation, edge functions, blockchain,
//     compliance, prediction, GraphQL subscriptions, monetization, multi-cloud (v3)
//  6. Enterprise — SLA, mTLS, contracts, marketplace, event-mesh, debugger,
//     cloud (billing/team/audit/onboard), Terraform provider, portal (v4)
//  7. Extended platform — tracing, canary, auto-remediation, schema registry,
//     catalog, sandbox, protocol gateway, analytics embed, cost engine,
//     GitOps, live migration, inbound, fan-out, mobile SDK (v5)
//  8. Ecosystem — DLQ, plugin marketplace, intelligence, flow builder,
//     time-travel, cloud-managed, callback, collab-debug, WAF, docgen,
//     OpenAPI gen, white-label (v7)
//  9. HTTP layer — Gin router with recovery, tracing, and metrics middleware
// 10. Route registration — handler setup and route binding
func NewServer() (*Server, error) {
	// ── Phase 1: Infrastructure (config, databases, migrations) ─────────
	config := utils.LoadConfig()
	logger := utils.NewLogger("api-service")

	// Connect to database
	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Connect to sqlx database for new features
	sqlxDB, err := database.NewSQLxConnection()
	if err != nil {
		logger.Error("Failed to connect to sqlx database", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("sqlx database connection failed: %w", err)
	}

	// Connect to Redis
	redisClient, err := database.NewRedisConnection(config.RedisURL)
	if err != nil {
		logger.Error("Failed to connect to Redis", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	// Run migrations
	if err := database.RunMigrations(config.DatabaseURL); err != nil {
		logger.Error("Failed to run migrations", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("database migrations failed: %w", err)
	}

	// ── Phase 2: Observability (health, alerts, metrics, tracing) ───────
	// Initialize monitoring components
	stdDB, err := database.GetStdDB()
	if err != nil {
		logger.Error("Failed to get std database connection", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("std database connection failed: %w", err)
	}
	healthChecker := monitoring.NewHealthChecker(stdDB, redisClient.Client, logger, "1.0.0")
	alertManager := monitoring.NewAlertManager(logger)
	metricsRecorder := monitoring.NewMetricsRecorder()
	tracer := monitoring.NewTracer("api-service", logger)

	// Setup alert notifiers
	logNotifier := monitoring.NewLogNotifier(logger)
	alertManager.AddNotifier(logNotifier)

	// ── Phase 3: Core services (v1) ─────────────────────────────────────
	// Initialize next-gen feature repositories and services
	flowRepo := flow.NewPostgresRepository(sqlxDB)
	flowService := flow.NewService(flowRepo)

	metaRepo := metaevents.NewPostgresRepository(sqlxDB)
	metaEmitter := metaevents.NewEmitter(metaRepo, 10)
	metaService := metaevents.NewService(metaRepo, metaEmitter)

	geoRepo := georouting.NewPostgresRepository(sqlxDB)
	geoService := georouting.NewService(geoRepo, nil) // nil for GeoIPProvider - optional

	embedRepo := embed.NewPostgresRepository(sqlxDB)
	embedService := embed.NewService(embedRepo)

	mockRepo := mocking.NewPostgresRepository(sqlxDB)
	mockService := mocking.NewService(mockRepo)

	costRepo := costing.NewPostgresRepository(sqlxDB)
	costService := costing.NewService(costRepo)

	otelRepo := otel.NewRepository(sqlxDB)
	otelService := otel.NewService(otelRepo)

	protocolRepo := protocols.NewRepository(sqlxDB)
	protocolService := protocols.NewService(protocolRepo, nil)

	// ── Phase 4: Platform services (v2) ─────────────────────────────────
	// Initialize next-gen features v2
	observabilityRepo := observability.NewPostgresRepository(sqlxDB)
	observabilityService := observability.NewService(observabilityRepo)

	smartlimitRepo := smartlimit.NewPostgresRepository(sqlxDB)
	smartlimitService := smartlimit.NewService(smartlimitRepo, nil)

	chaosRepo := chaos.NewPostgresRepository(sqlxDB)
	chaosService := chaos.NewService(chaosRepo, nil)

	cdcRepo := cdc.NewPostgresRepository(sqlxDB)
	cdcService := cdc.NewService(cdcRepo, nil, nil)

	workflowRepo := workflow.NewPostgresRepository(sqlxDB.DB)
	workflowService := workflow.NewService(workflowRepo, nil)

	signaturesRepo := signatures.NewPostgresRepository(sqlxDB.DB)
	signaturesService := signatures.NewService(signaturesRepo, nil)

	pushbridgeRepo := pushbridge.NewPostgresRepository(sqlxDB.DB)
	pushbridgeService := pushbridge.NewService(pushbridgeRepo, nil)

	billingRepo := billing.NewPostgresRepository(sqlxDB.DB)
	billingService := billing.NewService(billingRepo, nil, nil)

	versioningRepo := versioning.NewPostgresRepository(sqlxDB.DB)
	versioningService := versioning.NewService(versioningRepo)

	federationRepo := federation.NewPostgresRepository(sqlxDB.DB)
	federationService := federation.NewService(federationRepo, nil)

	// ── Phase 5: Delivery & edge services (v3) ──────────────────────────
	// Initialize next-gen features v3
	streamingService := streaming.NewService(nil, nil, streaming.DefaultServiceConfig())
	remediationService := remediation.NewService(nil, nil, nil, nil, remediation.DefaultServiceConfig())
	edgeService := edge.NewService(nil, edge.DefaultServiceConfig())
	blockchainService := blockchain.NewService(nil, blockchain.DefaultServiceConfig())
	complianceService := compliancecenter.NewService(nil, compliancecenter.DefaultServiceConfig())
	predictionService := prediction.NewService(nil, prediction.DefaultServiceConfig())
	graphqlsubService := graphqlsub.NewService(nil, graphqlsub.DefaultConnectionConfig())
	monetizationService := monetization.NewService(nil, monetization.DefaultServiceConfig())
	multicloudService := multicloud.NewFederationService(nil, nil, multicloud.DefaultFederationConfig())

	// ── Phase 6: Enterprise services (v4) ───────────────────────────────
	// Initialize next-gen features v4
	slaService := sla.NewService(nil)
	mtlsService := mtls.NewService(nil)
	contractsService := contracts.NewService(nil)
	marketplaceService := marketplacetpl.NewService(nil)
	eventmeshService := eventmesh.NewService(nil)
	debuggerService := debugger.NewService(nil)
	cloudctlService := cloudctl.NewService(nil)
	cloudBillingService := cloud.NewBillingService(nil, nil)
	cloudTeamService := cloud.NewTeamService(nil)
	cloudAuditService := cloud.NewAuditService(nil)
	cloudOnboardService := cloud.NewOnboardingService(nil, nil, nil)
	tfproviderService := tfprovider.NewService(nil)
	portalService := portal.NewService(nil)

	// ── Phase 7: Extended platform services (v5) ────────────────────────
	// Initialize next-gen features v5
	tracingService := tracing.NewService(nil)
	canaryService := canary.NewService(nil)
	autoremediationService := autoremediation.NewService(nil)
	schemaregistryService := schemaregistry.NewService(nil)
	catalogRepo := catalog.NewRepository(db)
	catalogService := catalog.NewService(catalogRepo)
	sandboxService := sandbox.NewService(nil)
	protocolgwService := protocolgw.NewService(nil)
	analyticsembedService := analyticsembed.NewService(nil)
	costengineService := costengine.NewService(nil)
	gitopsService := gitops.NewService(nil)
	livemigrationService := livemigration.NewService(nil)
	inboundRepo := inbound.NewPostgresRepository(sqlxDB)
	inboundService := inbound.NewService(inboundRepo)
	fanoutService := fanout.NewService(nil)
	mobilesdkService := mobilesdk.NewService()

	// ── Phase 8a: DLQ ───────────────────────────────────────────────────
	// Initialize DLQ service
	dlqService := dlq.NewService()

	// ── Phase 8b: Ecosystem services (v7) ───────────────────────────────
	// Initialize next-gen features v7
	pluginmarketRepo := pluginmarket.NewPostgresRepository(sqlxDB)
	pluginmarketService := pluginmarket.NewService(pluginmarketRepo)

	intelligenceRepo := intelligence.NewPostgresRepository(sqlxDB)
	intelligenceService := intelligence.NewService(intelligenceRepo)

	flowbuilderRepo := flowbuilder.NewPostgresRepository(sqlxDB)
	flowbuilderService := flowbuilder.NewService(flowbuilderRepo)

	timetravelRepo := timetravel.NewPostgresRepository(sqlxDB)
	timetravelService := timetravel.NewService(timetravelRepo)

	cloudmanagedRepo := cloudmanaged.NewPostgresRepository(sqlxDB)
	cloudmanagedService := cloudmanaged.NewService(cloudmanagedRepo)

	callbackRepo := callback.NewPostgresRepository(sqlxDB)
	callbackService := callback.NewService(callbackRepo)

	collabdebugRepo := collabdebug.NewPostgresRepository(sqlxDB)
	collabdebugService := collabdebug.NewService(collabdebugRepo)

	wafRepo := waf.NewPostgresRepository(sqlxDB)
	wafService := waf.NewService(wafRepo)

	docgenRepo := docgen.NewPostgresRepository(sqlxDB)
	docgenService := docgen.NewService(docgenRepo)

	openapigenService := openapigen.NewService()

	whitelabelRepo := whitelabel.NewPostgresRepository(sqlxDB)
	whitelabelService := whitelabel.NewService(whitelabelRepo)

	// Interactive Playground v2
	playgroundRepo := playground.NewRepository(db)
	playgroundEngine := transform.NewEngine(transform.DefaultEngineConfig())
	playgroundService := playground.NewService(playgroundRepo, playgroundEngine)

	// Pipeline Composition
	pipelineRepo := pipeline.NewMemoryRepository()
	pipelineService := pipeline.NewService(pipelineRepo)

	// ── Phase 9: HTTP layer (Gin router + middleware) ───────────────────
	// Setup Gin with monitoring middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(tracer.TracingMiddleware())
	router.Use(metrics.EnhancedMetricsMiddleware(metricsRecorder, alertManager))

	server := &Server{
		router:                 router,
		db:                     db,
		sqlxDB:                 sqlxDB,
		redisClient:            redisClient,
		logger:                 logger,
		config:                 config,
		healthChecker:          healthChecker,
		alertManager:           alertManager,
		metricsRecorder:        metricsRecorder,
		tracer:                 tracer,
		flowService:            flowService,
		metaService:            metaService,
		geoService:             geoService,
		embedService:           embedService,
		mockService:            mockService,
		costService:            costService,
		otelService:            otelService,
		protocolService:        protocolService,
		observabilityService:   observabilityService,
		smartlimitService:      smartlimitService,
		chaosService:           chaosService,
		cdcService:             cdcService,
		workflowService:        workflowService,
		signaturesService:      signaturesService,
		pushbridgeService:      pushbridgeService,
		billingService:         billingService,
		versioningService:      versioningService,
		federationService:      federationService,
		streamingService:       streamingService,
		remediationService:     remediationService,
		edgeService:            edgeService,
		blockchainService:      blockchainService,
		complianceService:      complianceService,
		predictionService:      predictionService,
		graphqlsubService:      graphqlsubService,
		monetizationService:    monetizationService,
		multicloudService:      multicloudService,
		slaService:             slaService,
		mtlsService:            mtlsService,
		contractsService:       contractsService,
		marketplaceService:     marketplaceService,
		eventmeshService:       eventmeshService,
		debuggerService:        debuggerService,
		cloudService:           cloudBillingService,
		cloudTeamService:       cloudTeamService,
		cloudAuditService:      cloudAuditService,
		cloudOnboardService:    cloudOnboardService,
		cloudctlService:        cloudctlService,
		tfproviderService:      tfproviderService,
		portalService:          portalService,
		tracingService:         tracingService,
		canaryService:          canaryService,
		autoremediationService: autoremediationService,
		schemaregistryService:  schemaregistryService,
		catalogService:         catalogService,
		sandboxService:         sandboxService,
		protocolgwService:      protocolgwService,
		analyticsembedService:  analyticsembedService,
		costengineService:      costengineService,
		gitopsService:          gitopsService,
		livemigrationService:   livemigrationService,
		inboundService:         inboundService,
		fanoutService:          fanoutService,
		mobilesdkService:       mobilesdkService,
		pluginmarketService:    pluginmarketService,
		intelligenceService:    intelligenceService,
		flowbuilderService:     flowbuilderService,
		timetravelService:      timetravelService,
		cloudmanagedService:    cloudmanagedService,
		callbackService:        callbackService,
		collabdebugService:     collabdebugService,
		wafService:             wafService,
		docgenService:          docgenService,
		whitelabelService:      whitelabelService,
		playgroundService:      playgroundService,
		pipelineService:        pipelineService,
		dlqService:             dlqService,
		openapigenService:      openapigenService,
	}

	server.setupRoutes()

	return server, nil
}

func (s *Server) setupRoutes() {
	// Initialize repositories
	tenantRepo := repository.NewTenantRepository(s.db)
	webhookRepo := repository.NewWebhookEndpointRepository(s.db)
	deliveryAttemptRepo := repository.NewDeliveryAttemptRepository(s.db)

	// Initialize queue publisher
	publisher := queue.NewPublisher(s.redisClient)

	// Initialize middleware
	authMiddleware := auth.NewAuthMiddleware(tenantRepo)
	rateLimiter := auth.NewRateLimiter(s.redisClient.Client)

	// Initialize handlers
	tenantHandler := handlers.NewTenantHandler(tenantRepo, s.logger)
	webhookHandler := handlers.NewWebhookHandler(webhookRepo, deliveryAttemptRepo, publisher, s.logger)
	testingHandler := handlers.NewTestingHandler(webhookRepo, deliveryAttemptRepo, publisher, s.logger)
	testEndpointHandler := handlers.NewTestEndpointHandler(s.logger)
	monitoringHandler := handlers.NewMonitoringHandler(deliveryAttemptRepo, webhookRepo, s.logger, s.healthChecker, s.alertManager, s.metricsRecorder)

	// Initialize next-gen feature handlers
	schemaRepo := schema.NewPostgresRepository(s.sqlxDB)
	schemaService := schema.NewService(schemaRepo)
	schemaHandler := schema.NewHandler(schemaService)
	flowHandler := flow.NewHandler(s.flowService)
	metaHandler := metaevents.NewHandler(s.metaService)
	geoHandler := georouting.NewHandler(s.geoService)
	embedHandler := embed.NewHandler(s.embedService)
	mockHandler := mocking.NewHandler(s.mockService)
	costHandler := costing.NewHandler(s.costService)
	otelHandler := otel.NewHandler(s.otelService)
	protocolHandler := protocols.NewHandler(s.protocolService)

	// Health and monitoring endpoints (no auth required)
	s.router.GET("/health", monitoringHandler.GetHealthStatus)
	s.router.GET("/ready", monitoringHandler.GetReadinessStatus)
	s.router.GET("/live", monitoringHandler.GetLivenessStatus)
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API Documentation endpoints (no auth required)
	s.router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public endpoints (no auth required)
	public := s.router.Group("/api/v1")
	{
		public.POST("/tenants", tenantHandler.CreateTenant)
	}

	// Test endpoint receivers (no auth required for webhook testing)
	testEndpoints := s.router.Group("/test")
	{
		testEndpoints.Any("/:endpoint_id", testEndpointHandler.ReceiveTestWebhook)
		testEndpoints.GET("/:endpoint_id/receives", testEndpointHandler.GetTestEndpointReceives)
		testEndpoints.GET("/:endpoint_id/receives/:receive_id", testEndpointHandler.GetTestEndpointReceive)
		testEndpoints.DELETE("/:endpoint_id/receives", testEndpointHandler.ClearTestEndpointReceives)
	}

	// Inbound webhook receiver (no auth - called by external providers)
	inboundPublic := s.router.Group("/api/v1")
	{
		inboundPublicHandler := inbound.NewHandler(s.inboundService)
		inboundPublicHandler.RegisterPublicRoutes(inboundPublic)
	}

	// Protected endpoints (require authentication and rate limiting)
	protected := s.router.Group("/api/v1")
	protected.Use(authMiddleware.RequireAuth())
	protected.Use(rateLimiter.RateLimit())
	{
		// Tenant management
		protected.GET("/tenant", tenantHandler.GetTenant)
		protected.PUT("/tenant", tenantHandler.UpdateTenant)
		protected.POST("/tenant/regenerate-key", tenantHandler.RegenerateAPIKey)

		// Webhook endpoint management
		protected.POST("/webhooks/endpoints", webhookHandler.CreateWebhookEndpoint)
		protected.GET("/webhooks/endpoints", webhookHandler.GetWebhookEndpoints)
		protected.GET("/webhooks/endpoints/:id", webhookHandler.GetWebhookEndpoint)
		protected.PUT("/webhooks/endpoints/:id", webhookHandler.UpdateWebhookEndpoint)
		protected.DELETE("/webhooks/endpoints/:id", webhookHandler.DeleteWebhookEndpoint)

		// Webhook sending
		protected.POST("/webhooks/send", webhookHandler.SendWebhook)
		protected.POST("/webhooks/send/batch", webhookHandler.BatchSendWebhook)

		// Webhook testing and debugging tools
		protected.POST("/webhooks/test", testingHandler.TestWebhook)
		protected.POST("/webhooks/test/endpoints", testingHandler.CreateTestEndpoint)
		protected.GET("/webhooks/deliveries/:id/inspect", testingHandler.InspectDelivery)
		protected.GET("/webhooks/deliveries/:id/logs", testingHandler.GetDeliveryLogs)
		protected.GET("/webhooks/realtime", testingHandler.WebSocketUpdates)

		// Monitoring and delivery history
		protected.GET("/webhooks/deliveries", monitoringHandler.GetDeliveryHistory)
		protected.GET("/webhooks/deliveries/:id", monitoringHandler.GetDeliveryDetails)
		protected.GET("/webhooks/endpoints/:id/deliveries", monitoringHandler.GetEndpointDeliveryHistory)

		// ==========================================
		// Next-Gen Feature Routes
		// ==========================================

		// Schema Registry
		schemaHandler.RegisterRoutes(protected)

		// Visual Flow Builder
		flowHandler.RegisterRoutes(protected)

		// Meta-Events (Webhooks for Webhooks)
		metaHandler.RegisterRoutes(protected)

		// Geographic Routing
		geoHandler.RegisterRoutes(protected)

		// Embedded Analytics SDK
		embedHandler.RegisterRoutes(protected)

		// Webhook Mocking Service
		mockHandler.RegisterRoutes(protected)

		// Cost Attribution Dashboard
		costHandler.RegisterRoutes(protected)

		// OpenTelemetry Integration
		otelHandler.RegisterRoutes(protected)

		// Custom Webhook Protocols
		protocolHandler.RegisterRoutes(protected)

		// ==========================================
		// Next-Gen Feature Routes v2
		// ==========================================

		// Webhook Observability Suite
		observabilityHandler := observability.NewHandler(s.observabilityService)
		observabilityHandler.RegisterRoutes(protected)

		// Intelligent Rate Limiting
		smartlimitHandler := smartlimit.NewHandler(s.smartlimitService)
		smartlimitHandler.RegisterRoutes(protected)

		// Webhook Chaos Engineering
		chaosHandler := chaos.NewHandler(s.chaosService)
		chaosHandler.RegisterRoutes(protected)

		// Native CDC Integration
		cdcHandlers := cdc.NewHandlers(s.cdcService)
		cdcHandlers.RegisterRoutes(protected)

		// Visual Workflow Builder v2
		workflowHandlers := workflow.NewHandlers(s.workflowService)
		workflowHandlers.RegisterRoutes(protected)

		// Webhook Signature Standardization
		signaturesHandlers := signatures.NewHandlers(s.signaturesService)
		signaturesHandlers.RegisterRoutes(protected)

		// Mobile SDK & Push Bridge
		pushbridgeHandlers := pushbridge.NewHandlers(s.pushbridgeService)
		pushbridgeHandlers.RegisterRoutes(protected)

		// Real-time Billing Alerts
		billingHandler := billing.NewHandler(s.billingService)
		billingHandler.RegisterRoutes(protected)

		// Webhook Versioning & Deprecation
		versioningHandler := versioning.NewHandler(s.versioningService)
		versioningHandler.RegisterRoutes(protected)

		// Federated Webhook Network
		federationHandler := federation.NewHandler(s.federationService)
		federationHandler.RegisterRoutes(protected)

		// ==========================================
		// Next-Gen Feature Routes v3
		// ==========================================

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

		// ==========================================
		// Next-Gen Feature Routes v4
		// ==========================================

		// SLA Dashboard & Alerting
		slaHandler := sla.NewHandler(s.slaService)
		slaHandler.RegisterRoutes(protected)

		// mTLS Certificate Management
		mtlsHandler := mtls.NewHandler(s.mtlsService)
		mtlsHandler.RegisterRoutes(protected)

		// Webhook Contract Testing
		contractsHandler := contracts.NewHandler(s.contractsService)
		contractsHandler.RegisterRoutes(protected)

		// Webhook Marketplace & Templates
		marketplaceHandler := marketplacetpl.NewHandler(s.marketplaceService)
		marketplaceHandler.RegisterRoutes(protected)

		// Event Mesh Routing Engine
		eventmeshHandler := eventmesh.NewHandler(s.eventmeshService)
		eventmeshHandler.RegisterRoutes(protected)

		// Webhook Debugger & Time-Travel Replay
		debuggerHandler := debugger.NewHandler(s.debuggerService)
		debuggerHandler.RegisterRoutes(protected)

		// Cloud Control Plane
		cloudctlHandler := cloudctl.NewHandler(s.cloudctlService)
		cloudctlHandler.RegisterRoutes(protected)

		// WaaS Cloud Managed Service
		cloudHandler := cloud.NewHandler(s.cloudService, s.cloudTeamService, s.cloudAuditService, s.cloudOnboardService)
		cloudHandler.RegisterRoutes(protected)

		// Terraform Provider API
		tfproviderHandler := tfprovider.NewHandler(s.tfproviderService)
		tfproviderHandler.RegisterRoutes(protected)

		// Embeddable Webhook Portal
		portalHandler := portal.NewHandler(s.portalService)
		portalHandler.RegisterRoutes(protected)

		// ==========================================
		// Next-Gen Feature Routes v5
		// ==========================================

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

		// ==========================================
		// Next-Gen Feature Routes v6
		// ==========================================

		// Inbound Webhook Gateway (management)
		inboundHandler := inbound.NewHandler(s.inboundService)
		inboundHandler.RegisterRoutes(protected)

		// Fan-Out & Topic-Based Routing
		fanoutHandler := fanout.NewHandler(s.fanoutService)
		fanoutHandler.RegisterRoutes(protected)

		// Mobile SDK Management
		mobilesdkHandler := mobilesdk.NewHandler(s.mobilesdkService)
		mobilesdkHandler.RegisterRoutes(protected)

		// ==========================================
		// Next-Gen Feature Routes v7
		// ==========================================

		// Webhook Plugin Marketplace
		pluginmarketHandler := pluginmarket.NewHandler(s.pluginmarketService)
		pluginmarketHandler.RegisterRoutes(protected)

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

	// Admin endpoints (require authentication but no rate limiting for now)
	admin := s.router.Group("/api/v1/admin")
	admin.Use(authMiddleware.RequireAuth())
	{
		admin.GET("/tenants", tenantHandler.ListTenants)
		admin.GET("/alerts/active", monitoringHandler.GetActiveAlerts)
		admin.GET("/alerts/history", monitoringHandler.GetAlertHistory)
	}
}

func (s *Server) Start(addr string) error {
	s.logger.Info("Starting API server", map[string]interface{}{
		"address": addr,
	})

	return s.router.Run(addr)
}

// Router returns the gin router for testing purposes
func (s *Server) Router() *gin.Engine {
	return s.router
}

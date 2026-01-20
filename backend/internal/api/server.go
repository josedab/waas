package api

import (
	"webhook-platform/internal/api/handlers"
	"webhook-platform/pkg/auth"
	"webhook-platform/pkg/billing"
	"webhook-platform/pkg/blockchain"
	"webhook-platform/pkg/cdc"
	"webhook-platform/pkg/chaos"
	"webhook-platform/pkg/compliancecenter"
	"webhook-platform/pkg/costing"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/edge"
	"webhook-platform/pkg/embed"
	"webhook-platform/pkg/federation"
	"webhook-platform/pkg/flow"
	"webhook-platform/pkg/georouting"
	"webhook-platform/pkg/graphqlsub"
	"webhook-platform/pkg/metaevents"
	"webhook-platform/pkg/metrics"
	"webhook-platform/pkg/mocking"
	"webhook-platform/pkg/monetization"
	"webhook-platform/pkg/monitoring"
	"webhook-platform/pkg/multicloud"
	"webhook-platform/pkg/observability"
	"webhook-platform/pkg/otel"
	"webhook-platform/pkg/prediction"
	"webhook-platform/pkg/protocols"
	"webhook-platform/pkg/pushbridge"
	"webhook-platform/pkg/queue"
	"webhook-platform/pkg/remediation"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/schema"
	"webhook-platform/pkg/signatures"
	"webhook-platform/pkg/smartlimit"
	"webhook-platform/pkg/streaming"
	"webhook-platform/pkg/utils"
	"webhook-platform/pkg/versioning"
	"webhook-platform/pkg/workflow"
	_ "webhook-platform/docs"

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
	flowService       *flow.Service
	metaService       *metaevents.Service
	geoService        *georouting.Service
	embedService      *embed.Service
	mockService       *mocking.Service
	costService       *costing.Service
	otelService       *otel.Service
	protocolService   *protocols.Service
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
	streamingService      *streaming.Service
	remediationService    *remediation.Service
	edgeService           *edge.Service
	blockchainService     *blockchain.Service
	complianceService     *compliancecenter.Service
	predictionService     *prediction.Service
	graphqlsubService     *graphqlsub.Service
	monetizationService   *monetization.Service
	multicloudService     *multicloud.FederationService
}

func NewServer() *Server {
	config := utils.LoadConfig()
	logger := utils.NewLogger("api-service")
	
	// Connect to database
	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
		panic(err)
	}

	// Connect to sqlx database for new features
	sqlxDB, err := database.NewSQLxConnection()
	if err != nil {
		logger.Error("Failed to connect to sqlx database", map[string]interface{}{
			"error": err.Error(),
		})
		panic(err)
	}

	// Connect to Redis
	redisClient, err := database.NewRedisConnection(config.RedisURL)
	if err != nil {
		logger.Error("Failed to connect to Redis", map[string]interface{}{
			"error": err.Error(),
		})
		panic(err)
	}

	// Run migrations
	if err := database.RunMigrations(config.DatabaseURL); err != nil {
		logger.Error("Failed to run migrations", map[string]interface{}{
			"error": err.Error(),
		})
		panic(err)
	}

	// Initialize monitoring components
	stdDB, err := database.GetStdDB()
	if err != nil {
		logger.Error("Failed to get std database connection", map[string]interface{}{
			"error": err.Error(),
		})
		panic(err)
	}
	healthChecker := monitoring.NewHealthChecker(stdDB, redisClient.Client, logger, "1.0.0")
	alertManager := monitoring.NewAlertManager(logger)
	metricsRecorder := monitoring.NewMetricsRecorder()
	tracer := monitoring.NewTracer("api-service", logger)
	
	// Setup alert notifiers
	logNotifier := monitoring.NewLogNotifier(logger)
	alertManager.AddNotifier(logNotifier)

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
	
	// Setup Gin with monitoring middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(tracer.TracingMiddleware())
	router.Use(metrics.EnhancedMetricsMiddleware(metricsRecorder, alertManager))
	
	server := &Server{
		router:               router,
		db:                   db,
		sqlxDB:               sqlxDB,
		redisClient:          redisClient,
		logger:               logger,
		config:               config,
		healthChecker:        healthChecker,
		alertManager:         alertManager,
		metricsRecorder:      metricsRecorder,
		tracer:               tracer,
		flowService:          flowService,
		metaService:          metaService,
		geoService:           geoService,
		embedService:         embedService,
		mockService:          mockService,
		costService:          costService,
		otelService:          otelService,
		protocolService:      protocolService,
		observabilityService: observabilityService,
		smartlimitService:    smartlimitService,
		chaosService:         chaosService,
		cdcService:           cdcService,
		workflowService:      workflowService,
		signaturesService:    signaturesService,
		pushbridgeService:    pushbridgeService,
		billingService:       billingService,
		versioningService:    versioningService,
		federationService:    federationService,
		streamingService:     streamingService,
		remediationService:   remediationService,
		edgeService:          edgeService,
		blockchainService:    blockchainService,
		complianceService:    complianceService,
		predictionService:    predictionService,
		graphqlsubService:    graphqlsubService,
		monetizationService:  monetizationService,
		multicloudService:    multicloudService,
	}

	server.setupRoutes()
	
	return server
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
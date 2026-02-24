package api

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	"github.com/josedab/waas/pkg/compliancevault"
	"github.com/josedab/waas/pkg/contracts"
	"github.com/josedab/waas/pkg/costengine"
	"github.com/josedab/waas/pkg/costing"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/dataplane"
	"github.com/josedab/waas/pkg/debugger"
	"github.com/josedab/waas/pkg/deliveryreceipt"
	"github.com/josedab/waas/pkg/dlq"
	"github.com/josedab/waas/pkg/docgen"
	"github.com/josedab/waas/pkg/edge"
	"github.com/josedab/waas/pkg/embed"
	"github.com/josedab/waas/pkg/eventmesh"
	"github.com/josedab/waas/pkg/eventcorrelation"
	"github.com/josedab/waas/pkg/eventlineage"
	"github.com/josedab/waas/pkg/experiment"
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
	"github.com/josedab/waas/pkg/obscodepipeline"
	"github.com/josedab/waas/pkg/onboarding"
	"github.com/josedab/waas/pkg/openapigen"
	"github.com/josedab/waas/pkg/otel"
	"github.com/josedab/waas/pkg/pipeline"
	"github.com/josedab/waas/pkg/playground"
	"github.com/josedab/waas/pkg/pluginmarket"
	"github.com/josedab/waas/pkg/portal"
	"github.com/josedab/waas/pkg/portalsdk"
	"github.com/josedab/waas/pkg/receiverdash"
	"github.com/josedab/waas/pkg/nlbuilder"
	"github.com/josedab/waas/pkg/depgraph"
	"github.com/josedab/waas/pkg/e2ee"
	"github.com/josedab/waas/pkg/selfhealing"
	"github.com/josedab/waas/pkg/loadtest"
	"github.com/josedab/waas/pkg/routingpolicy"
	"github.com/josedab/waas/pkg/schemachangelog"
	"github.com/josedab/waas/pkg/mobileinspector"
	"github.com/josedab/waas/pkg/piidetection"
	"github.com/josedab/waas/pkg/policyengine"
	"github.com/josedab/waas/pkg/standardwebhooks"
	"github.com/josedab/waas/pkg/topologysim"
	"github.com/josedab/waas/pkg/prediction"
	"github.com/josedab/waas/pkg/protocolgw"
	"github.com/josedab/waas/pkg/protocols"
	"github.com/josedab/waas/pkg/pushbridge"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/remediation"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/sandbox"
	"github.com/josedab/waas/pkg/schemaregistry"
	"github.com/josedab/waas/pkg/sdkgen"
	"github.com/josedab/waas/pkg/security"
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

	"github.com/gin-contrib/cors"
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
	dlqService              *dlq.Service
	openapigenService       *openapigen.Service
	obscodepipelineService  *obscodepipeline.Service
	compliancevaultService  *compliancevault.Service
	portalsdkService        *portalsdk.Service
	// Next-gen features v8
	receiverdashService     *receiverdash.Service
	nlbuilderService        *nlbuilder.Service
	depgraphService         *depgraph.Service
	e2eeService             *e2ee.Service
	selfhealingService      *selfhealing.Service
	loadtestService         *loadtest.Service
	routingpolicyService    *routingpolicy.Service
	schemachangelogService  *schemachangelog.Service
	mobileinspectorService  *mobileinspector.Service
	topologysimService      *topologysim.Service
	// Next-gen features v9
	piidetectionService     *piidetection.Service
	standardwebhooksService *standardwebhooks.Service
	policyengineService     *policyengine.Service
	experimentService       *experiment.Service
	// Next-gen features v10
	eventcorrelationService *eventcorrelation.Service
	deliveryreceiptService  *deliveryreceipt.Service
	eventlineageService     *eventlineage.Service
	sdkgenService           *sdkgen.Service
	dataplaneService        *dataplane.Service
	onboardingWizardService *onboarding.Service
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
//
// 10. Route registration — handler setup and route binding
func NewServer() (*Server, error) {
	// ── Phase 1: Infrastructure (config, databases, migrations) ─────────
	config, err := utils.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}
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

	// Observability-as-Code Pipeline
	obscodepipelineService := obscodepipeline.NewService(nil)

	// Compliance Vault
	compliancevaultService := compliancevault.NewService(nil, nil)

	// Portal SDK
	portalsdkService := portalsdk.NewService(nil)

	// Receiver Dashboard
	receiverdashRepo := receiverdash.NewMemoryRepository()
	receiverdashService := receiverdash.NewService(receiverdashRepo, nil)

	// NL Webhook Builder
	nlbuilderRepo := nlbuilder.NewMemoryRepository()
	nlbuilderService := nlbuilder.NewService(nlbuilderRepo, nil, nil)

	// Dependency Graph
	depgraphRepo := depgraph.NewMemoryRepository()
	depgraphService := depgraph.NewService(depgraphRepo, nil)

	// E2EE
	e2eeRepo := e2ee.NewMemoryRepository()
	e2eeService := e2ee.NewService(e2eeRepo, nil)

	// Self-Healing
	selfhealingRepo := selfhealing.NewMemoryRepository()
	selfhealingService := selfhealing.NewService(selfhealingRepo, nil)

	// Load Testing
	loadtestRepo := loadtest.NewMemoryRepository()
	loadtestService := loadtest.NewService(loadtestRepo, nil)

	// Routing Policies
	routingpolicyRepo := routingpolicy.NewMemoryRepository()
	routingpolicyService := routingpolicy.NewService(routingpolicyRepo, nil)

	// Schema Changelog
	schemachangelogRepo := schemachangelog.NewMemoryRepository()
	schemachangelogService := schemachangelog.NewService(schemachangelogRepo)

	// Mobile Inspector
	mobileinspectorRepo := mobileinspector.NewMemoryRepository()
	mobileinspectorService := mobileinspector.NewService(mobileinspectorRepo, nil)

	// Topology Simulator
	topologysimRepo := topologysim.NewMemoryRepository()
	topologysimService := topologysim.NewService(topologysimRepo, nil)

	// PII Detection & Masking Engine
	piidetectionService := piidetection.NewService(nil)

	// Standard Webhooks & CloudEvents
	standardwebhooksService := standardwebhooks.NewService()

	// Programmable Policy Engine (OPA/Rego)
	policyengineService := policyengine.NewService(nil)

	// Webhook A/B Testing & Experimentation
	experimentService := experiment.NewService(nil)

	// Event Correlation & Complex Event Processing
	eventcorrelationService := eventcorrelation.NewService(nil)

	// Delivery Receipt & Processing Confirmation
	deliveryreceiptService := deliveryreceipt.NewService(nil)

	// Event Lineage & Provenance Tracker
	eventlineageService := eventlineage.NewService(nil)

	// Automated Consumer SDK Generation
	sdkgenService := sdkgen.NewService()

	// Multi-Tenant Dedicated Data Planes
	dataplaneService := dataplane.NewService(nil)

	// Interactive Onboarding Wizard
	onboardingWizardService := onboarding.NewService(nil)

	// ── Phase 9: HTTP layer (Gin router + middleware) ───────────────────
	// Setup Gin with monitoring middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(tracer.TracingMiddleware())
	router.Use(metrics.EnhancedMetricsMiddleware(metricsRecorder, alertManager))

	// CORS middleware
	allowedOrigins := []string{}
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Security headers (X-Content-Type-Options, X-Frame-Options, HSTS, etc.)
	secureAuthMiddleware := security.NewSecureAuthMiddleware(nil, nil)
	router.Use(secureAuthMiddleware.SecurityHeaders())

	// Request body size limit (1 MB) to prevent memory exhaustion DoS
	router.Use(auth.MaxBodySize(auth.DefaultMaxBodySize))

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
		dlqService:              dlqService,
		openapigenService:       openapigenService,
		obscodepipelineService:  obscodepipelineService,
		compliancevaultService:  compliancevaultService,
		portalsdkService:        portalsdkService,
		receiverdashService:     receiverdashService,
		nlbuilderService:        nlbuilderService,
		depgraphService:         depgraphService,
		e2eeService:             e2eeService,
		selfhealingService:      selfhealingService,
		loadtestService:         loadtestService,
		routingpolicyService:    routingpolicyService,
		schemachangelogService:  schemachangelogService,
		mobileinspectorService:  mobileinspectorService,
		topologysimService:      topologysimService,
		piidetectionService:     piidetectionService,
		standardwebhooksService: standardwebhooksService,
		policyengineService:     policyengineService,
		experimentService:       experimentService,
		eventcorrelationService: eventcorrelationService,
		deliveryreceiptService:  deliveryreceiptService,
		eventlineageService:     eventlineageService,
		sdkgenService:           sdkgenService,
		dataplaneService:        dataplaneService,
		onboardingWizardService: onboardingWizardService,
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

	// Health and monitoring endpoints (no auth required)
	s.router.GET("/health", monitoringHandler.GetHealthStatus)
	s.router.GET("/ready", monitoringHandler.GetReadinessStatus)
	s.router.GET("/live", monitoringHandler.GetLivenessStatus)

	// Metrics endpoint (requires authentication)
	metricsGroup := s.router.Group("")
	metricsGroup.Use(authMiddleware.RequireAuth())
	metricsGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API Documentation endpoints (only available in development/debug mode)
	if gin.Mode() != gin.ReleaseMode {
		s.router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Public endpoints (no auth required, rate limited)
	public := s.router.Group("/api/v1")
	public.Use(rateLimiter.RateLimit())
	{
		// Tenant creation has stricter IP-based rate limiting (5 req/min per IP)
		public.POST("/tenants", rateLimiter.IPRateLimit(5, time.Minute), tenantHandler.CreateTenant)
	}

	// Test endpoint receivers (no auth required for webhook testing, rate limited)
	testEndpoints := s.router.Group("/test")
	testEndpoints.Use(rateLimiter.RateLimit())
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

		// Register domain-specific routes
		s.registerCoreRoutes(protected)
		s.registerEnterpriseRoutes(protected)
		s.registerObservabilityRoutes(protected)
	}

	// Admin endpoints (require authentication, admin role, and rate limiting)
	adminMiddleware := auth.NewAdminMiddleware()
	admin := s.router.Group("/api/v1/admin")
	admin.Use(authMiddleware.RequireAuth())
	admin.Use(adminMiddleware.RequireAdmin())
	admin.Use(rateLimiter.RateLimit())
	{
		admin.GET("/tenants", tenantHandler.ListTenants)
		admin.PUT("/tenants/:tenant_id", tenantHandler.AdminUpdateTenant)
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

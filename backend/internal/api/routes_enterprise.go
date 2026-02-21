package api

import (
	"github.com/josedab/waas/pkg/billing"
	"github.com/josedab/waas/pkg/cdc"
	"github.com/josedab/waas/pkg/chaos"
	"github.com/josedab/waas/pkg/cloud"
	"github.com/josedab/waas/pkg/cloudctl"
	"github.com/josedab/waas/pkg/contracts"
	"github.com/josedab/waas/pkg/debugger"
	"github.com/josedab/waas/pkg/eventmesh"
	"github.com/josedab/waas/pkg/federation"
	"github.com/josedab/waas/pkg/marketplacetpl"
	"github.com/josedab/waas/pkg/mtls"
	"github.com/josedab/waas/pkg/observability"
	"github.com/josedab/waas/pkg/portal"
	"github.com/josedab/waas/pkg/pushbridge"
	"github.com/josedab/waas/pkg/signatures"
	"github.com/josedab/waas/pkg/sla"
	"github.com/josedab/waas/pkg/smartlimit"
	"github.com/josedab/waas/pkg/tfprovider"
	"github.com/josedab/waas/pkg/versioning"
	"github.com/josedab/waas/pkg/workflow"

	"github.com/gin-gonic/gin"
)

// registerEnterpriseRoutes registers platform (v2) and enterprise (v4)
// feature routes: observability, smart-limit, chaos, CDC, workflow,
// signatures, push-bridge, billing, versioning, federation, SLA, mTLS,
// contracts, marketplace, event-mesh, debugger, cloud, Terraform, portal.
func (s *Server) registerEnterpriseRoutes(protected *gin.RouterGroup) {
	// ── Platform services (v2) ──────────────────────────────────────

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

	// ── Enterprise services (v4) ────────────────────────────────────

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
	eventmeshHandler.RegisterMeshRoutes(protected)

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
}

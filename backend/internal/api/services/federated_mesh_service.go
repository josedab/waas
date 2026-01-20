package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"
)

// FederatedMeshService handles multi-region mesh operations
type FederatedMeshService struct {
	repo   repository.FederatedMeshRepository
	logger *utils.Logger
}

// NewFederatedMeshService creates a new federated mesh service
func NewFederatedMeshService(repo repository.FederatedMeshRepository, logger *utils.Logger) *FederatedMeshService {
	return &FederatedMeshService{
		repo:   repo,
		logger: logger,
	}
}

// GetAllRegions retrieves all regions
func (s *FederatedMeshService) GetAllRegions(ctx context.Context) ([]*models.Region, error) {
	return s.repo.GetAllRegions(ctx)
}

// GetActiveRegions retrieves all active regions
func (s *FederatedMeshService) GetActiveRegions(ctx context.Context) ([]*models.Region, error) {
	return s.repo.GetActiveRegions(ctx)
}

// GetRegion retrieves a region by ID
func (s *FederatedMeshService) GetRegion(ctx context.Context, regionID uuid.UUID) (*models.Region, error) {
	return s.repo.GetRegion(ctx, regionID)
}

// GetRegionByCode retrieves a region by code
func (s *FederatedMeshService) GetRegionByCode(ctx context.Context, code string) (*models.Region, error) {
	return s.repo.GetRegionByCode(ctx, code)
}

// SetupTenantRegion configures regional settings for a tenant
func (s *FederatedMeshService) SetupTenantRegion(ctx context.Context, tenantID uuid.UUID, req *models.CreateTenantRegionRequest) (*models.MeshTenantRegion, error) {
	primaryRegionID, err := uuid.Parse(req.PrimaryRegionID)
	if err != nil {
		return nil, fmt.Errorf("invalid primary_region_id")
	}

	// Verify region exists
	_, err = s.repo.GetRegion(ctx, primaryRegionID)
	if err != nil {
		return nil, fmt.Errorf("primary region not found")
	}

	var allowedRegions []uuid.UUID
	for _, r := range req.AllowedRegions {
		id, err := uuid.Parse(r)
		if err == nil {
			allowedRegions = append(allowedRegions, id)
		}
	}

	// Default policy
	policy := req.DataResidencyPolicy
	if policy == "" {
		policy = models.DataResidencyFlexible
	}

	replicationMode := req.ReplicationMode
	if replicationMode == "" {
		replicationMode = models.ReplicationModeAsync
	}

	tr := &models.MeshTenantRegion{
		TenantID:             tenantID,
		PrimaryRegionID:      primaryRegionID,
		AllowedRegions:       allowedRegions,
		DataResidencyPolicy:  policy,
		ReplicationMode:      replicationMode,
		ComplianceFrameworks: req.ComplianceFrameworks,
	}

	// Check if tenant already has region config
	existing, _ := s.repo.GetTenantRegion(ctx, tenantID)
	if existing != nil {
		tr.ID = existing.ID
		if err := s.repo.UpdateTenantRegion(ctx, tr); err != nil {
			return nil, err
		}
		return s.repo.GetTenantRegion(ctx, tenantID)
	}

	if err := s.repo.CreateTenantRegion(ctx, tr); err != nil {
		return nil, fmt.Errorf("failed to create tenant region: %w", err)
	}

	s.logger.Info("Tenant region configured", map[string]interface{}{"tenant_id": tenantID, "primary_region": primaryRegionID})

	return tr, nil
}

// GetTenantRegion retrieves tenant region configuration
func (s *FederatedMeshService) GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (*models.MeshTenantRegion, error) {
	tr, err := s.repo.GetTenantRegion(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Enrich with primary region details
	if tr.PrimaryRegionID != uuid.Nil {
		region, _ := s.repo.GetRegion(ctx, tr.PrimaryRegionID)
		tr.PrimaryRegion = region
	}

	return tr, nil
}

// CreateRoutingRule creates a geo-routing rule
func (s *FederatedMeshService) CreateRoutingRule(ctx context.Context, tenantID uuid.UUID, req *models.CreateGeoRoutingRuleRequest) (*models.GeoRoutingRule, error) {
	validRuleTypes := map[string]bool{
		models.GeoRuleLatency:     true,
		models.GeoRuleGeofence:    true,
		models.GeoRuleLoadBalance: true,
		models.GeoRuleFailover:    true,
	}

	if !validRuleTypes[req.RuleType] {
		return nil, fmt.Errorf("invalid rule_type: %s", req.RuleType)
	}

	var sourceRegions []uuid.UUID
	for _, r := range req.SourceRegions {
		id, err := uuid.Parse(r)
		if err == nil {
			sourceRegions = append(sourceRegions, id)
		}
	}

	rule := &models.GeoRoutingRule{
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		RuleType:      req.RuleType,
		Priority:      req.Priority,
		SourceRegions: sourceRegions,
		Conditions:    req.Conditions,
		Enabled:       true,
	}

	if req.TargetRegionID != "" {
		targetID, err := uuid.Parse(req.TargetRegionID)
		if err == nil {
			rule.TargetRegionID = &targetID
		}
	}

	if err := s.repo.CreateRoutingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create routing rule: %w", err)
	}

	return rule, nil
}

// GetRoutingRules retrieves routing rules for a tenant
func (s *FederatedMeshService) GetRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error) {
	return s.repo.GetRoutingRulesByTenant(ctx, tenantID)
}

// RouteEvent determines the optimal region for an event
func (s *FederatedMeshService) RouteEvent(ctx context.Context, tenantID uuid.UUID, req *models.RouteEventRequest) (*models.RegionalRoutingDecision, error) {
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event_id")
	}

	sourceRegionID, err := uuid.Parse(req.SourceRegionID)
	if err != nil {
		return nil, fmt.Errorf("invalid source_region_id")
	}

	// Get tenant region config
	tenantRegion, err := s.repo.GetTenantRegion(ctx, tenantID)
	if err != nil {
		// No region config, use source region
		decision := &models.RegionalRoutingDecision{
			TenantID:       tenantID,
			EventID:        eventID,
			SourceRegionID: sourceRegionID,
			TargetRegionID: sourceRegionID,
			DecisionReason: "no_tenant_region_config",
		}
		s.repo.CreateRoutingDecision(ctx, decision)
		return decision, nil
	}

	// Get enabled routing rules
	rules, err := s.repo.GetEnabledRoutingRules(ctx, tenantID)
	if err != nil || len(rules) == 0 {
		// Default to primary region
		decision := &models.RegionalRoutingDecision{
			TenantID:       tenantID,
			EventID:        eventID,
			SourceRegionID: sourceRegionID,
			TargetRegionID: tenantRegion.PrimaryRegionID,
			DecisionReason: "default_primary_region",
		}
		s.repo.CreateRoutingDecision(ctx, decision)
		return decision, nil
	}

	// Evaluate rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	for _, rule := range rules {
		targetRegion, reason, match := s.evaluateRule(ctx, rule, sourceRegionID, tenantRegion)
		if match {
			decision := &models.RegionalRoutingDecision{
				TenantID:       tenantID,
				EventID:        eventID,
				SourceRegionID: sourceRegionID,
				TargetRegionID: targetRegion,
				RoutingRuleID:  &rule.ID,
				DecisionReason: reason,
			}
			s.repo.CreateRoutingDecision(ctx, decision)
			return decision, nil
		}
	}

	// Fallback to primary region
	decision := &models.RegionalRoutingDecision{
		TenantID:       tenantID,
		EventID:        eventID,
		SourceRegionID: sourceRegionID,
		TargetRegionID: tenantRegion.PrimaryRegionID,
		DecisionReason: "no_matching_rule",
	}
	s.repo.CreateRoutingDecision(ctx, decision)

	return decision, nil
}

// evaluateRule evaluates a routing rule
func (s *FederatedMeshService) evaluateRule(ctx context.Context, rule *models.GeoRoutingRule, sourceRegionID uuid.UUID, tenantRegion *models.MeshTenantRegion) (uuid.UUID, string, bool) {
	switch rule.RuleType {
	case models.GeoRuleLatency:
		return s.evaluateLatencyRule(ctx, rule, sourceRegionID, tenantRegion)
	case models.GeoRuleGeofence:
		return s.evaluateGeofenceRule(ctx, rule, sourceRegionID, tenantRegion)
	case models.GeoRuleLoadBalance:
		return s.evaluateLoadBalanceRule(ctx, rule, tenantRegion)
	case models.GeoRuleFailover:
		return s.evaluateFailoverRule(ctx, rule, sourceRegionID, tenantRegion)
	}

	return uuid.Nil, "", false
}

// evaluateLatencyRule finds lowest latency region
func (s *FederatedMeshService) evaluateLatencyRule(ctx context.Context, rule *models.GeoRoutingRule, sourceRegionID uuid.UUID, tenantRegion *models.MeshTenantRegion) (uuid.UUID, string, bool) {
	// Get active regions
	regions, _ := s.repo.GetActiveRegions(ctx)
	if len(regions) == 0 {
		return uuid.Nil, "", false
	}

	// Filter to allowed regions
	allowedMap := make(map[uuid.UUID]bool)
	for _, r := range tenantRegion.AllowedRegions {
		allowedMap[r] = true
	}
	allowedMap[tenantRegion.PrimaryRegionID] = true

	var bestRegion *models.Region
	var bestLatency float64 = math.MaxFloat64

	sourceRegion, _ := s.repo.GetRegion(ctx, sourceRegionID)

	for _, region := range regions {
		if tenantRegion.DataResidencyPolicy == models.DataResidencyStrict && !allowedMap[region.ID] {
			continue
		}

		// Calculate approximate latency based on distance
		latency := s.calculateLatency(sourceRegion, region)
		if latency < bestLatency {
			bestLatency = latency
			bestRegion = region
		}
	}

	if bestRegion != nil {
		return bestRegion.ID, fmt.Sprintf("latency_based:%.0fms", bestLatency), true
	}

	return uuid.Nil, "", false
}

// calculateLatency estimates latency based on geographic distance
func (s *FederatedMeshService) calculateLatency(source, target *models.Region) float64 {
	if source == nil || target == nil || source.Latitude == nil || target.Latitude == nil {
		return 100.0 // Default latency
	}

	// Haversine formula for distance
	lat1 := *source.Latitude * math.Pi / 180
	lat2 := *target.Latitude * math.Pi / 180
	lon1 := *source.Longitude * math.Pi / 180
	lon2 := *target.Longitude * math.Pi / 180

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Earth radius in km
	distanceKm := 6371 * c

	// Approximate latency: ~0.01ms per km (speed of light factor) + 10ms base
	return distanceKm*0.01 + 10
}

// evaluateGeofenceRule checks geofence restrictions
func (s *FederatedMeshService) evaluateGeofenceRule(ctx context.Context, rule *models.GeoRoutingRule, sourceRegionID uuid.UUID, tenantRegion *models.MeshTenantRegion) (uuid.UUID, string, bool) {
	// Check if source region is in allowed list
	for _, allowedID := range rule.SourceRegions {
		if sourceRegionID == allowedID && rule.TargetRegionID != nil {
			return *rule.TargetRegionID, "geofence_match", true
		}
	}

	return uuid.Nil, "", false
}

// evaluateLoadBalanceRule distributes load across regions
func (s *FederatedMeshService) evaluateLoadBalanceRule(ctx context.Context, rule *models.GeoRoutingRule, tenantRegion *models.MeshTenantRegion) (uuid.UUID, string, bool) {
	regions, _ := s.repo.GetActiveRegions(ctx)
	if len(regions) == 0 {
		return uuid.Nil, "", false
	}

	// Find region with lowest load
	var bestRegion *models.Region
	lowestLoad := math.MaxInt

	for _, region := range regions {
		if region.HealthStatus != models.RegionHealthHealthy {
			continue
		}

		loadPercent := 0
		if region.CapacityLimit > 0 {
			loadPercent = region.CurrentLoad * 100 / region.CapacityLimit
		}

		if loadPercent < lowestLoad {
			lowestLoad = loadPercent
			bestRegion = region
		}
	}

	if bestRegion != nil {
		return bestRegion.ID, fmt.Sprintf("load_balance:%d%%", lowestLoad), true
	}

	return uuid.Nil, "", false
}

// evaluateFailoverRule checks for failover conditions
func (s *FederatedMeshService) evaluateFailoverRule(ctx context.Context, rule *models.GeoRoutingRule, sourceRegionID uuid.UUID, tenantRegion *models.MeshTenantRegion) (uuid.UUID, string, bool) {
	// Check if source region is unhealthy
	sourceRegion, _ := s.repo.GetRegion(ctx, sourceRegionID)
	if sourceRegion != nil && sourceRegion.HealthStatus != models.RegionHealthUnhealthy {
		return uuid.Nil, "", false // No failover needed
	}

	// Find healthy region
	regions, _ := s.repo.GetActiveRegions(ctx)
	for _, region := range regions {
		if region.ID != sourceRegionID && region.HealthStatus == models.RegionHealthHealthy {
			return region.ID, fmt.Sprintf("failover_from_%s", sourceRegion.Code), true
		}
	}

	return uuid.Nil, "", false
}

// CreateReplicationStream creates a replication stream
func (s *FederatedMeshService) CreateReplicationStream(ctx context.Context, tenantID uuid.UUID, req *models.CreateReplicationStreamRequest) (*models.ReplicationStream, error) {
	sourceRegionID, err := uuid.Parse(req.SourceRegionID)
	if err != nil {
		return nil, fmt.Errorf("invalid source_region_id")
	}

	targetRegionID, err := uuid.Parse(req.TargetRegionID)
	if err != nil {
		return nil, fmt.Errorf("invalid target_region_id")
	}

	validStreamTypes := map[string]bool{
		models.StreamTypeEvents:  true,
		models.StreamTypeConfigs: true,
		models.StreamTypeState:   true,
	}

	if !validStreamTypes[req.StreamType] {
		return nil, fmt.Errorf("invalid stream_type: %s", req.StreamType)
	}

	stream := &models.ReplicationStream{
		TenantID:       tenantID,
		SourceRegionID: sourceRegionID,
		TargetRegionID: targetRegionID,
		StreamType:     req.StreamType,
		Status:         "active",
		Metadata:       make(map[string]interface{}),
	}

	if err := s.repo.CreateReplicationStream(ctx, stream); err != nil {
		return nil, fmt.Errorf("failed to create replication stream: %w", err)
	}

	s.logger.Info("Replication stream created", map[string]interface{}{"stream_id": stream.ID, "type": req.StreamType})

	return stream, nil
}

// GetReplicationStreams retrieves replication streams for a tenant
func (s *FederatedMeshService) GetReplicationStreams(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplicationStream, error) {
	return s.repo.GetReplicationStreamsByTenant(ctx, tenantID)
}

// InitiateFailover initiates a region failover
func (s *FederatedMeshService) InitiateFailover(ctx context.Context, req *models.InitiateFailoverRequest) (*models.FailoverEvent, error) {
	fromRegionID, err := uuid.Parse(req.FromRegionID)
	if err != nil {
		return nil, fmt.Errorf("invalid from_region_id")
	}

	toRegionID, err := uuid.Parse(req.ToRegionID)
	if err != nil {
		return nil, fmt.Errorf("invalid to_region_id")
	}

	// Verify target region is healthy
	toRegion, err := s.repo.GetRegion(ctx, toRegionID)
	if err != nil {
		return nil, fmt.Errorf("target region not found")
	}

	if toRegion.HealthStatus != models.RegionHealthHealthy {
		return nil, fmt.Errorf("target region is not healthy")
	}

	event := &models.FailoverEvent{
		FromRegionID:    fromRegionID,
		ToRegionID:      toRegionID,
		TriggerReason:   req.TriggerReason,
		Automatic:       false,
		Status:          models.FailoverInProgress,
		AffectedTenants: 0,
		Metadata:        make(map[string]interface{}),
	}

	if err := s.repo.CreateFailoverEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create failover event: %w", err)
	}

	// Mark source region as draining
	s.repo.UpdateRegionHealth(ctx, fromRegionID, models.RegionStatusDraining)

	s.logger.Info("Failover initiated", map[string]interface{}{"from": fromRegionID, "to": toRegionID})

	// In production, would trigger async failover process
	// For now, complete immediately
	s.repo.CompleteFailover(ctx, event.ID, models.FailoverCompleted)

	return s.repo.GetFailoverEvent(ctx, event.ID)
}

// CheckDataResidencyCompliance checks data residency compliance
func (s *FederatedMeshService) CheckDataResidencyCompliance(ctx context.Context, tenantID uuid.UUID, sourceRegionID, targetRegionID uuid.UUID, dataType string) (*models.DataResidencyAudit, error) {
	tenantRegion, err := s.repo.GetTenantRegion(ctx, tenantID)
	if err != nil {
		// No restrictions if no config
		return nil, nil
	}

	audit := &models.DataResidencyAudit{
		TenantID:       tenantID,
		EventType:      "data_transfer",
		SourceRegionID: &sourceRegionID,
		TargetRegionID: &targetRegionID,
		DataType:       dataType,
		Details:        make(map[string]interface{}),
	}

	// Check if target region is allowed
	isAllowed := false
	if targetRegionID == tenantRegion.PrimaryRegionID {
		isAllowed = true
	} else {
		for _, allowed := range tenantRegion.AllowedRegions {
			if targetRegionID == allowed {
				isAllowed = true
				break
			}
		}
	}

	if tenantRegion.DataResidencyPolicy == models.DataResidencyStrict && !isAllowed {
		audit.ComplianceStatus = models.ComplianceViolation
		audit.Details["reason"] = "target_region_not_allowed"
		audit.Details["policy"] = tenantRegion.DataResidencyPolicy
	} else if tenantRegion.DataResidencyPolicy == models.DataResidencyFlexible && !isAllowed {
		audit.ComplianceStatus = models.ComplianceWarning
		audit.Details["reason"] = "target_region_not_preferred"
	} else {
		audit.ComplianceStatus = models.ComplianceCompliant
	}

	s.repo.CreateResidencyAudit(ctx, audit)

	return audit, nil
}

// GetMeshDashboard retrieves the federated mesh dashboard
func (s *FederatedMeshService) GetMeshDashboard(ctx context.Context, tenantID uuid.UUID) (*models.MeshDashboard, error) {
	dashboard := &models.MeshDashboard{
		RegionalHealth: make(map[string]interface{}),
	}

	// Get all regions
	allRegions, _ := s.repo.GetAllRegions(ctx)
	dashboard.TotalRegions = len(allRegions)

	activeCount := 0
	healthyCount := 0
	for _, r := range allRegions {
		if r.Status == models.RegionStatusActive {
			activeCount++
		}
		if r.HealthStatus == models.RegionHealthHealthy {
			healthyCount++
		}
	}
	dashboard.ActiveRegions = activeCount
	dashboard.HealthyRegions = healthyCount

	// Get tenant region config
	tenantRegion, _ := s.repo.GetTenantRegion(ctx, tenantID)
	dashboard.TenantRegion = tenantRegion

	// Get replication streams
	streams, _ := s.repo.GetReplicationStreamsByTenant(ctx, tenantID)
	dashboard.ReplicationStreams = streams

	// Count active routing rules
	rules, _ := s.repo.GetEnabledRoutingRules(ctx, tenantID)
	dashboard.ActiveRoutingRules = len(rules)

	// Get recent failovers
	failovers, _ := s.repo.GetRecentFailovers(ctx, 5)
	dashboard.RecentFailovers = failovers

	// Determine compliance status
	if tenantRegion != nil {
		dashboard.ComplianceStatus = s.determineComplianceStatus(ctx, tenantID, tenantRegion)
	} else {
		dashboard.ComplianceStatus = "not_configured"
	}

	// Regional health summary
	for _, r := range allRegions {
		dashboard.RegionalHealth[r.Code] = map[string]interface{}{
			"status":       r.Status,
			"health":       r.HealthStatus,
			"load_percent": 0,
		}
		if r.CapacityLimit > 0 {
			dashboard.RegionalHealth[r.Code].(map[string]interface{})["load_percent"] = r.CurrentLoad * 100 / r.CapacityLimit
		}
	}

	return dashboard, nil
}

// determineComplianceStatus determines overall compliance status
func (s *FederatedMeshService) determineComplianceStatus(ctx context.Context, tenantID uuid.UUID, tenantRegion *models.MeshTenantRegion) string {
	audits, _ := s.repo.GetResidencyAudits(ctx, tenantID, 100)

	recentViolations := 0
	since := time.Now().Add(-24 * time.Hour)

	for _, audit := range audits {
		if audit.RecordedAt.After(since) && audit.ComplianceStatus == models.ComplianceViolation {
			recentViolations++
		}
	}

	if recentViolations > 0 {
		return "violation"
	}

	return "compliant"
}

// RecordHealthMetric records a health metric for a region
func (s *FederatedMeshService) RecordHealthMetric(ctx context.Context, regionID uuid.UUID, metricType string, value float64) error {
	metric := &models.RegionHealthMetric{
		RegionID:    regionID,
		MetricType:  metricType,
		MetricValue: value,
	}

	return s.repo.CreateHealthMetric(ctx, metric)
}

// GetRegionsWithMetrics retrieves regions with their latest metrics
func (s *FederatedMeshService) GetRegionsWithMetrics(ctx context.Context) ([]*models.RegionWithMetrics, error) {
	regions, err := s.repo.GetActiveRegions(ctx)
	if err != nil {
		return nil, err
	}

	var result []*models.RegionWithMetrics
	for _, region := range regions {
		rwm := &models.RegionWithMetrics{
			Region: region,
		}

		metrics, _ := s.repo.GetLatestHealthMetrics(ctx, region.ID)
		for _, m := range metrics {
			switch m.MetricType {
			case "latency":
				rwm.Latency = m.MetricValue
			case "error_rate":
				rwm.ErrorRate = m.MetricValue
			case "throughput":
				rwm.Throughput = m.MetricValue
			case "capacity":
				if region.CapacityLimit > 0 {
					rwm.Utilization = float64(region.CurrentLoad) / float64(region.CapacityLimit) * 100
				}
			}
		}

		result = append(result, rwm)
	}

	return result, nil
}

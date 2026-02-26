package protocolgw

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Service provides protocol gateway translation functionality
type Service struct {
	repo   Repository
	logger *utils.Logger
}

// NewService creates a new protocol gateway service
func NewService(repo Repository) *Service {
	return &Service{repo: repo, logger: utils.NewLogger("protocolgw-service")}
}

var validProtocols = map[string]bool{
	ProtocolHTTP:    true,
	ProtocolGRPC:    true,
	ProtocolMQTT:    true,
	ProtocolKafka:   true,
	ProtocolKinesis: true,
}

var validOrderingGuarantees = map[string]bool{
	OrderingNone:     true,
	OrderingFIFO:     true,
	OrderingKeyBased: true,
}

var validDeliveryGuarantees = map[string]bool{
	DeliveryAtMostOnce:  true,
	DeliveryAtLeastOnce: true,
	DeliveryExactlyOnce: true,
}

// CreateRoute creates a new protocol translation route
func (s *Service) CreateRoute(ctx context.Context, tenantID string, req *CreateRouteRequest) (*ProtocolRoute, error) {
	if !validProtocols[req.SourceProtocol] {
		return nil, fmt.Errorf("invalid source protocol: %s", req.SourceProtocol)
	}
	if !validProtocols[req.DestProtocol] {
		return nil, fmt.Errorf("invalid destination protocol: %s", req.DestProtocol)
	}

	ordering := req.OrderingGuarantee
	if ordering == "" {
		ordering = OrderingNone
	}
	if !validOrderingGuarantees[ordering] {
		return nil, fmt.Errorf("invalid ordering guarantee: %s", ordering)
	}

	delivery := req.DeliveryGuarantee
	if delivery == "" {
		delivery = DeliveryAtLeastOnce
	}
	if !validDeliveryGuarantees[delivery] {
		return nil, fmt.Errorf("invalid delivery guarantee: %s", delivery)
	}

	if err := s.ValidateProtocolConfig(req.SourceProtocol, req.SourceConfig); err != nil {
		return nil, fmt.Errorf("invalid source config: %w", err)
	}
	if err := s.ValidateProtocolConfig(req.DestProtocol, req.DestConfig); err != nil {
		return nil, fmt.Errorf("invalid destination config: %w", err)
	}

	now := time.Now()
	route := &ProtocolRoute{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		SourceProtocol:    req.SourceProtocol,
		SourceConfig:      req.SourceConfig,
		DestProtocol:      req.DestProtocol,
		DestConfig:        req.DestConfig,
		TransformRule:     req.TransformRule,
		OrderingGuarantee: ordering,
		DeliveryGuarantee: delivery,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateRoute(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to create protocol route: %w", err)
	}

	return route, nil
}

// GetRoute retrieves a protocol route by ID
func (s *Service) GetRoute(ctx context.Context, tenantID, routeID string) (*ProtocolRoute, error) {
	return s.repo.GetRoute(ctx, tenantID, routeID)
}

// ListRoutes retrieves all protocol routes for a tenant
func (s *Service) ListRoutes(ctx context.Context, tenantID string) ([]ProtocolRoute, error) {
	return s.repo.ListRoutes(ctx, tenantID)
}

// UpdateRoute updates an existing protocol route
func (s *Service) UpdateRoute(ctx context.Context, tenantID, routeID string, req *CreateRouteRequest) (*ProtocolRoute, error) {
	route, err := s.repo.GetRoute(ctx, tenantID, routeID)
	if err != nil {
		return nil, err
	}

	if !validProtocols[req.SourceProtocol] {
		return nil, fmt.Errorf("invalid source protocol: %s", req.SourceProtocol)
	}
	if !validProtocols[req.DestProtocol] {
		return nil, fmt.Errorf("invalid destination protocol: %s", req.DestProtocol)
	}

	ordering := req.OrderingGuarantee
	if ordering == "" {
		ordering = OrderingNone
	}
	if !validOrderingGuarantees[ordering] {
		return nil, fmt.Errorf("invalid ordering guarantee: %s", ordering)
	}

	delivery := req.DeliveryGuarantee
	if delivery == "" {
		delivery = DeliveryAtLeastOnce
	}
	if !validDeliveryGuarantees[delivery] {
		return nil, fmt.Errorf("invalid delivery guarantee: %s", delivery)
	}

	route.Name = req.Name
	route.Description = req.Description
	route.SourceProtocol = req.SourceProtocol
	route.SourceConfig = req.SourceConfig
	route.DestProtocol = req.DestProtocol
	route.DestConfig = req.DestConfig
	route.TransformRule = req.TransformRule
	route.OrderingGuarantee = ordering
	route.DeliveryGuarantee = delivery
	route.UpdatedAt = time.Now()

	if err := s.repo.UpdateRoute(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update protocol route: %w", err)
	}

	return route, nil
}

// DeleteRoute deletes a protocol route
func (s *Service) DeleteRoute(ctx context.Context, tenantID, routeID string) error {
	return s.repo.DeleteRoute(ctx, tenantID, routeID)
}

// TranslateMessage translates a message from source to destination protocol
func (s *Service) TranslateMessage(ctx context.Context, tenantID string, req *TranslateMessageRequest) (*TranslationResult, error) {
	route, err := s.repo.GetRoute(ctx, tenantID, req.RouteID)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}

	if !route.IsActive {
		return nil, fmt.Errorf("route %s is not active", route.ID)
	}

	start := time.Now()

	translatedPayload, err := s.applyTransform(route.SourceProtocol, route.DestProtocol, req.Payload, route.TransformRule)
	latencyMs := time.Since(start).Milliseconds()

	result := &TranslationResult{
		RouteID:        route.ID,
		SourceProtocol: route.SourceProtocol,
		DestProtocol:   route.DestProtocol,
		OriginalSize:   len(req.Payload),
		LatencyMs:      latencyMs,
	}

	msg := &ProtocolMessage{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		RouteID:        route.ID,
		SourceProtocol: route.SourceProtocol,
		DestProtocol:   route.DestProtocol,
		Payload:        req.Payload,
		Headers:        req.Headers,
		PartitionKey:   req.PartitionKey,
		LatencyMs:      latencyMs,
		CreatedAt:      time.Now(),
	}

	if err != nil {
		msg.Status = MessageStatusFailed
		msg.ErrorMessage = err.Error()
		result.Success = false
		result.Error = err.Error()
	} else {
		msg.Status = MessageStatusTranslated
		msg.TranslatedPayload = translatedPayload
		result.TranslatedSize = len(translatedPayload)
		result.Success = true
	}

	if err := s.repo.RecordMessage(ctx, msg); err != nil {
		s.logger.Error("failed to record message", map[string]interface{}{"error": err.Error(), "message_id": msg.ID})
	}

	return result, nil
}

// GetRouteStats returns statistics for a specific route
func (s *Service) GetRouteStats(ctx context.Context, tenantID, routeID string) (*ProtocolStats, error) {
	return s.repo.GetRouteStats(ctx, tenantID, routeID)
}

// GetProtocolStats returns aggregate statistics across all routes
func (s *Service) GetProtocolStats(ctx context.Context, tenantID string) ([]ProtocolStats, error) {
	return s.repo.GetAggregateStats(ctx, tenantID)
}

// ValidateProtocolConfig validates configuration for a given protocol type
func (s *Service) ValidateProtocolConfig(protocol, config string) error {
	if config == "" {
		return nil
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		return fmt.Errorf("config must be valid JSON: %w", err)
	}

	switch protocol {
	case ProtocolHTTP:
		if _, ok := parsed["url"]; !ok {
			return fmt.Errorf("HTTP config requires 'url' field")
		}
	case ProtocolGRPC:
		if _, ok := parsed["host"]; !ok {
			return fmt.Errorf("gRPC config requires 'host' field")
		}
	case ProtocolMQTT:
		if _, ok := parsed["broker"]; !ok {
			return fmt.Errorf("MQTT config requires 'broker' field")
		}
		if _, ok := parsed["topic"]; !ok {
			return fmt.Errorf("MQTT config requires 'topic' field")
		}
	case ProtocolKafka:
		if _, ok := parsed["brokers"]; !ok {
			return fmt.Errorf("Kafka config requires 'brokers' field")
		}
		if _, ok := parsed["topic"]; !ok {
			return fmt.Errorf("Kafka config requires 'topic' field")
		}
	case ProtocolKinesis:
		if _, ok := parsed["stream"]; !ok {
			return fmt.Errorf("Kinesis config requires 'stream' field")
		}
		if _, ok := parsed["region"]; !ok {
			return fmt.Errorf("Kinesis config requires 'region' field")
		}
	}

	return nil
}

func (s *Service) applyTransform(sourceProtocol, destProtocol, payload, transformRule string) (string, error) {
	key := sourceProtocol + "->" + destProtocol
	switch key {
	case ProtocolHTTP + "->" + ProtocolKafka:
		return s.translateHTTPToKafka(payload, transformRule)
	case ProtocolKafka + "->" + ProtocolHTTP:
		return s.translateKafkaToHTTP(payload, transformRule)
	case ProtocolHTTP + "->" + ProtocolMQTT:
		return s.translateHTTPToMQTT(payload, transformRule)
	case ProtocolMQTT + "->" + ProtocolHTTP:
		return s.translateMQTTToHTTP(payload, transformRule)
	case ProtocolHTTP + "->" + ProtocolGRPC:
		return s.translateHTTPToGRPC(payload, transformRule)
	case ProtocolGRPC + "->" + ProtocolHTTP:
		return s.translateGRPCToHTTP(payload, transformRule)
	case ProtocolKafka + "->" + ProtocolKinesis:
		return s.translateKafkaToKinesis(payload, transformRule)
	case ProtocolKinesis + "->" + ProtocolKafka:
		return s.translateKinesisToKafka(payload, transformRule)
	default:
		return s.translateGeneric(sourceProtocol, destProtocol, payload, transformRule)
	}
}

func (s *Service) translateHTTPToKafka(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":    "kafka",
		"source":    "http",
		"value":     payload,
		"timestamp": time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateKafkaToHTTP(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":       "http",
		"source":       "kafka",
		"body":         payload,
		"content_type": "application/json",
		"timestamp":    time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateHTTPToMQTT(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":    "mqtt",
		"source":    "http",
		"payload":   payload,
		"qos":       1,
		"timestamp": time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateMQTTToHTTP(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":       "http",
		"source":       "mqtt",
		"body":         payload,
		"content_type": "application/json",
		"timestamp":    time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateHTTPToGRPC(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":    "grpc",
		"source":    "http",
		"message":   payload,
		"encoding":  "proto3",
		"timestamp": time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateGRPCToHTTP(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":       "http",
		"source":       "grpc",
		"body":         payload,
		"content_type": "application/json",
		"timestamp":    time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateKafkaToKinesis(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":    "kinesis",
		"source":    "kafka",
		"data":      payload,
		"timestamp": time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateKinesisToKafka(payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":    "kafka",
		"source":    "kinesis",
		"value":     payload,
		"timestamp": time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

func (s *Service) translateGeneric(sourceProtocol, destProtocol, payload, _ string) (string, error) {
	envelope := map[string]interface{}{
		"format":    destProtocol,
		"source":    sourceProtocol,
		"payload":   payload,
		"timestamp": time.Now().UnixMilli(),
	}
	out, err := json.Marshal(envelope)
	return string(out), err
}

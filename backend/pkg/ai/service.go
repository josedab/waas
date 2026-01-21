package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for AI-related storage
type Repository interface {
	// Analysis storage
	SaveAnalysis(ctx context.Context, analysis *DebugAnalysis) error
	GetAnalysis(ctx context.Context, tenantID, analysisID string) (*DebugAnalysis, error)
	GetAnalysisByDelivery(ctx context.Context, tenantID, deliveryID string) (*DebugAnalysis, error)
	ListAnalyses(ctx context.Context, tenantID string, limit, offset int) ([]DebugAnalysis, int, error)

	// Pattern storage
	SavePattern(ctx context.Context, pattern *ErrorPattern) error
	GetPatterns(ctx context.Context, tenantID string, limit int) ([]ErrorPattern, error)
	IncrementPatternFrequency(ctx context.Context, tenantID, patternID string) error

	// Delivery context retrieval
	GetDeliveryContext(ctx context.Context, tenantID, deliveryID string) (*DeliveryContext, error)
	GetSimilarDeliveries(ctx context.Context, tenantID string, classification ErrorClassification, limit int) ([]DeliveryContext, error)
}

// Service provides AI-powered debugging capabilities
type Service struct {
	repo       Repository
	classifier *Classifier
	llm        LLMClient
	config     *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	EnableLLM          bool
	CacheTTL           time.Duration
	MaxSimilarResults  int
	PatternLearning    bool
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		EnableLLM:         true,
		CacheTTL:          24 * time.Hour,
		MaxSimilarResults: 5,
		PatternLearning:   true,
	}
}

// NewService creates a new AI debugging service
func NewService(repo Repository, llmConfig *LLMConfig, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	var llm LLMClient
	if config.EnableLLM {
		llm = CreateLLMClient(llmConfig)
	} else {
		llm = NewLocalClient()
	}

	return &Service{
		repo:       repo,
		classifier: NewClassifier(),
		llm:        llm,
		config:     config,
	}
}

// AnalyzeDelivery analyzes a failed webhook delivery
func (s *Service) AnalyzeDelivery(ctx context.Context, tenantID string, req *AnalysisRequest) (*DebugAnalysis, error) {
	startTime := time.Now()

	// Check for cached analysis
	if existing, err := s.repo.GetAnalysisByDelivery(ctx, tenantID, req.DeliveryID); err == nil && existing != nil {
		return existing, nil
	}

	// Get delivery context
	deliveryCtx, err := s.repo.GetDeliveryContext(ctx, tenantID, req.DeliveryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery context: %w", err)
	}
	if deliveryCtx == nil {
		return nil, fmt.Errorf("delivery not found: %s", req.DeliveryID)
	}

	// Classify the error
	classification := s.classifier.Classify(deliveryCtx.ErrorMessage, deliveryCtx.HTTPStatus, deliveryCtx.ResponseBody)

	// Get suggestions based on classification
	suggestions := s.classifier.GetSuggestions(classification, deliveryCtx)

	// Build analysis
	analysis := &DebugAnalysis{
		ID:             uuid.New().String(),
		DeliveryID:     req.DeliveryID,
		Classification: classification,
		Suggestions:    suggestions,
		CreatedAt:      time.Now(),
	}

	// Use LLM for deeper analysis if enabled
	if s.config.EnableLLM && s.llm != nil {
		llmAnalysis, err := s.performLLMAnalysis(ctx, deliveryCtx, classification)
		if err == nil && llmAnalysis != nil {
			analysis.RootCause = llmAnalysis.RootCause
			analysis.Explanation = llmAnalysis.Explanation
			analysis.ConfidenceScore = llmAnalysis.ConfidenceScore
			if llmAnalysis.TransformFix != nil && req.GenerateTransform {
				analysis.TransformFix = llmAnalysis.TransformFix
			}
		}
	} else {
		// Use rule-based analysis
		analysis.RootCause = s.getRuleBasedRootCause(classification)
		analysis.Explanation = s.getRuleBasedExplanation(classification, deliveryCtx)
		analysis.ConfidenceScore = classification.Confidence
	}

	// Find similar issues
	if req.IncludeSimilar {
		similar, err := s.findSimilarIssues(ctx, tenantID, classification)
		if err == nil {
			analysis.SimilarIssues = similar
		}
	}

	analysis.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Save analysis
	if err := s.repo.SaveAnalysis(ctx, analysis); err != nil {
		// Log but don't fail
	}

	// Learn pattern if enabled
	if s.config.PatternLearning {
		s.learnPattern(ctx, tenantID, deliveryCtx, classification)
	}

	return analysis, nil
}

// AnalyzeBatch analyzes multiple deliveries and provides summary
func (s *Service) AnalyzeBatch(ctx context.Context, tenantID string, req *BatchAnalysisRequest) (*BatchAnalysisResponse, error) {
	response := &BatchAnalysisResponse{
		Analyses:   make([]DebugAnalysis, 0, len(req.DeliveryIDs)),
		TotalCount: len(req.DeliveryIDs),
	}

	categoryCount := make(map[ErrorCategory]int)
	endpointSet := make(map[string]bool)

	for _, deliveryID := range req.DeliveryIDs {
		analysis, err := s.AnalyzeDelivery(ctx, tenantID, &AnalysisRequest{
			DeliveryID:       deliveryID,
			IncludeSimilar:   req.IncludeSimilar,
			GenerateTransform: req.GenerateTransform,
		})
		if err != nil {
			response.FailedCount++
			continue
		}

		response.Analyses = append(response.Analyses, *analysis)
		categoryCount[analysis.Classification.Category]++

		// Get delivery context for endpoint info
		if ctx, err := s.repo.GetDeliveryContext(ctx, tenantID, deliveryID); err == nil && ctx != nil {
			endpointSet[ctx.EndpointID] = true
		}
	}

	// Build summary
	response.Summary = s.buildSummary(categoryCount, endpointSet, len(req.DeliveryIDs))

	return response, nil
}

// GenerateTransformation generates a transformation script using AI
func (s *Service) GenerateTransformation(ctx context.Context, tenantID string, req *TransformGenerateRequest) (*TransformGenerateResponse, error) {
	var input, output interface{}
	if err := json.Unmarshal(req.InputExample, &input); err != nil {
		return nil, fmt.Errorf("invalid input example: %w", err)
	}
	if err := json.Unmarshal(req.OutputExample, &output); err != nil {
		return nil, fmt.Errorf("invalid output example: %w", err)
	}

	return s.llm.GenerateTransform(ctx, req.Description, input, output)
}

// GetAnalysis retrieves a previously generated analysis
func (s *Service) GetAnalysis(ctx context.Context, tenantID, analysisID string) (*DebugAnalysis, error) {
	return s.repo.GetAnalysis(ctx, tenantID, analysisID)
}

// ListAnalyses lists analyses for a tenant
func (s *Service) ListAnalyses(ctx context.Context, tenantID string, limit, offset int) ([]DebugAnalysis, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListAnalyses(ctx, tenantID, limit, offset)
}

// GetPatterns retrieves learned error patterns
func (s *Service) GetPatterns(ctx context.Context, tenantID string, limit int) ([]ErrorPattern, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetPatterns(ctx, tenantID, limit)
}

func (s *Service) performLLMAnalysis(ctx context.Context, delivery *DeliveryContext, classification ErrorClassification) (*DebugAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze this webhook delivery failure:

URL: %s
HTTP Status: %v
Error: %s
Response: %s
Attempt: %d
Latency: %dms

Classification: %s (%s)

Provide analysis in JSON format:
{
  "root_cause": "specific root cause",
  "explanation": "detailed explanation of what went wrong",
  "confidence_score": 0.0-1.0,
  "transform_fix": {
    "description": "if payload issue, describe the fix",
    "script": "JavaScript transformation code"
  }
}`, delivery.URL, delivery.HTTPStatus, delivery.ErrorMessage, delivery.ResponseBody,
		delivery.AttemptNumber, delivery.Latency, classification.Category, classification.SubCategory)

	response, err := s.llm.Analyze(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		RootCause       string       `json:"root_cause"`
		Explanation     string       `json:"explanation"`
		ConfidenceScore float64      `json:"confidence_score"`
		TransformFix    *TransformFix `json:"transform_fix,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback to using response as explanation
		return &DebugAnalysis{
			RootCause:       "See explanation",
			Explanation:     response,
			ConfidenceScore: 0.7,
		}, nil
	}

	return &DebugAnalysis{
		RootCause:       result.RootCause,
		Explanation:     result.Explanation,
		ConfidenceScore: result.ConfidenceScore,
		TransformFix:    result.TransformFix,
	}, nil
}

func (s *Service) getRuleBasedRootCause(classification ErrorClassification) string {
	causes := map[ErrorCategory]string{
		CategoryNetwork:     "Network connectivity issue preventing webhook delivery",
		CategoryTimeout:     "Request timed out before receiving a response",
		CategoryAuth:        "Authentication or authorization failure",
		CategoryRateLimit:   "Endpoint rate limit exceeded",
		CategoryServerError: "Target server encountered an error",
		CategoryClientError: "Request rejected due to client-side issue",
		CategoryPayload:     "Payload format or content issue",
		CategoryCertificate: "SSL/TLS certificate validation failure",
		CategoryDNS:         "DNS resolution failure for target domain",
		CategoryUnknown:     "Unable to determine specific root cause",
	}

	if cause, ok := causes[classification.Category]; ok {
		return cause
	}
	return "Unknown error occurred"
}

func (s *Service) getRuleBasedExplanation(classification ErrorClassification, ctx *DeliveryContext) string {
	baseExplanation := fmt.Sprintf("The webhook delivery to %s failed ", ctx.URL)

	switch classification.Category {
	case CategoryNetwork:
		return baseExplanation + "due to a network connectivity issue. The target server could not be reached."
	case CategoryTimeout:
		return baseExplanation + fmt.Sprintf("because the request timed out after %dms. The target server did not respond in time.", ctx.Latency)
	case CategoryAuth:
		return baseExplanation + "due to an authentication error. The provided credentials were rejected."
	case CategoryRateLimit:
		return baseExplanation + "because the endpoint rate limit was exceeded. Too many requests were sent in a short period."
	case CategoryServerError:
		if ctx.HTTPStatus != nil {
			return baseExplanation + fmt.Sprintf("with HTTP status %d. The target server encountered an internal error.", *ctx.HTTPStatus)
		}
		return baseExplanation + "due to a server-side error."
	case CategoryClientError:
		if ctx.HTTPStatus != nil {
			return baseExplanation + fmt.Sprintf("with HTTP status %d. The request was rejected by the server.", *ctx.HTTPStatus)
		}
		return baseExplanation + "due to a client-side error in the request."
	case CategoryPayload:
		return baseExplanation + "due to an issue with the payload format or content."
	case CategoryCertificate:
		return baseExplanation + "due to an SSL/TLS certificate issue with the target server."
	case CategoryDNS:
		return baseExplanation + "because the domain name could not be resolved."
	default:
		return baseExplanation + "for an unknown reason. Review the error details for more information."
	}
}

func (s *Service) findSimilarIssues(ctx context.Context, tenantID string, classification ErrorClassification) ([]SimilarIssue, error) {
	deliveries, err := s.repo.GetSimilarDeliveries(ctx, tenantID, classification, s.config.MaxSimilarResults)
	if err != nil {
		return nil, err
	}

	similar := make([]SimilarIssue, 0, len(deliveries))
	for _, d := range deliveries {
		similar = append(similar, SimilarIssue{
			DeliveryID: d.DeliveryID,
			Similarity: 0.9, // Simplified - in production, calculate actual similarity
		})
	}

	return similar, nil
}

func (s *Service) learnPattern(ctx context.Context, tenantID string, delivery *DeliveryContext, classification ErrorClassification) {
	normalized := s.classifier.normalizeError(delivery.ErrorMessage)
	if normalized == "" {
		return
	}

	pattern := &ErrorPattern{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Pattern:   normalized,
		Category:  classification.Category,
		Frequency: 1,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	// Save or increment existing pattern
	s.repo.SavePattern(ctx, pattern)
}

func (s *Service) buildSummary(categoryCount map[ErrorCategory]int, endpoints map[string]bool, total int) AnalysisSummary {
	summary := AnalysisSummary{
		TopCategories:     make([]CategoryCount, 0),
		CommonRootCauses:  make([]string, 0),
		AffectedEndpoints: make([]string, 0, len(endpoints)),
	}

	// Top categories
	for cat, count := range categoryCount {
		summary.TopCategories = append(summary.TopCategories, CategoryCount{
			Category: cat,
			Count:    count,
			Percent:  float64(count) / float64(total) * 100,
		})
	}

	// Affected endpoints
	for ep := range endpoints {
		summary.AffectedEndpoints = append(summary.AffectedEndpoints, ep)
	}

	// Recommended action based on top category
	maxCount := 0
	var topCategory ErrorCategory
	for cat, count := range categoryCount {
		if count > maxCount {
			maxCount = count
			topCategory = cat
		}
	}

	switch topCategory {
	case CategoryNetwork, CategoryDNS:
		summary.RecommendedAction = "Investigate network connectivity to affected endpoints"
	case CategoryTimeout:
		summary.RecommendedAction = "Consider increasing timeout settings or optimizing payload size"
	case CategoryAuth:
		summary.RecommendedAction = "Review and update authentication credentials"
	case CategoryRateLimit:
		summary.RecommendedAction = "Implement request throttling or contact endpoint owner"
	case CategoryServerError:
		summary.RecommendedAction = "Contact endpoint owners about server-side issues"
	case CategoryClientError, CategoryPayload:
		summary.RecommendedAction = "Review and fix payload format or request parameters"
	default:
		summary.RecommendedAction = "Review individual failure details"
	}

	return summary
}

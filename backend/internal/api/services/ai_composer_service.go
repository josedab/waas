package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// AIComposerService handles AI-powered webhook configuration generation
type AIComposerService struct {
	repo           repository.AIComposerRepository
	webhookRepo    repository.WebhookEndpointRepository
	llmClient      LLMClient
	logger         *utils.Logger
	jsValidator    *JavaScriptValidator
}

// LLMClient interface for AI provider abstraction
type LLMClient interface {
	Complete(ctx context.Context, messages []LLMMessage, options LLMOptions) (*LLMResponse, error)
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMOptions struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Model       string  `json:"model"`
}

type LLMResponse struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
	Usage        struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// JavaScriptValidator validates generated transformation code
type JavaScriptValidator struct{}

func NewJavaScriptValidator() *JavaScriptValidator {
	return &JavaScriptValidator{}
}

func (v *JavaScriptValidator) Validate(code string) (bool, []string) {
	var errors []string

	// Basic syntax checks
	if strings.Count(code, "{") != strings.Count(code, "}") {
		errors = append(errors, "Unbalanced curly braces")
	}
	if strings.Count(code, "(") != strings.Count(code, ")") {
		errors = append(errors, "Unbalanced parentheses")
	}
	if strings.Count(code, "[") != strings.Count(code, "]") {
		errors = append(errors, "Unbalanced brackets")
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		`eval\s*\(`, `Function\s*\(`, `require\s*\(`, `import\s+`,
		`__proto__`, `constructor\s*\[`, `process\.`, `child_process`,
	}
	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			errors = append(errors, fmt.Sprintf("Potentially dangerous pattern detected: %s", pattern))
		}
	}

	return len(errors) == 0, errors
}

// NewAIComposerService creates a new AI composer service
func NewAIComposerService(
	repo repository.AIComposerRepository,
	webhookRepo repository.WebhookEndpointRepository,
	llmClient LLMClient,
	logger *utils.Logger,
) *AIComposerService {
	return &AIComposerService{
		repo:        repo,
		webhookRepo: webhookRepo,
		llmClient:   llmClient,
		logger:      logger,
		jsValidator: NewJavaScriptValidator(),
	}
}

const systemPrompt = `You are an expert webhook configuration assistant. Your job is to help users create webhook endpoint configurations and payload transformations.

When generating webhook configurations, output valid JSON with these possible fields:
- url: The webhook destination URL (required)
- method: HTTP method (GET, POST, PUT, PATCH, DELETE) - default POST
- headers: Custom headers as key-value pairs
- retry_config: { max_attempts, initial_delay_ms, max_delay_ms, backoff_multiplier }
- transformation: JavaScript code to transform the payload (use 'payload' variable for input)

For transformations, write clean JavaScript that:
- Uses the 'payload' variable to access incoming data
- Returns the transformed object
- Handles null/undefined gracefully

Example transformation:
return {
  event: payload.type,
  data: {
    id: payload.id,
    timestamp: new Date().toISOString()
  }
};

Always validate URLs and provide sensible defaults. If you need more information, ask specific questions.
Respond with a JSON block wrapped in triple backticks when providing configurations.`

// Compose processes a user prompt and generates webhook configuration
func (s *AIComposerService) Compose(ctx context.Context, tenantID uuid.UUID, req *models.AIComposerRequest) (*models.AIComposerResponse, error) {
	var session *models.AIComposerSession
	var err error

	// Get or create session
	if req.SessionID != nil {
		session, err = s.repo.GetSession(ctx, *req.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
		if session.TenantID != tenantID {
			return nil, fmt.Errorf("session not found")
		}
	} else {
		session = &models.AIComposerSession{
			ID:       uuid.New(),
			TenantID: tenantID,
			Status:   models.AISessionStatusActive,
			Context:  make(map[string]interface{}),
		}
		if err := s.repo.CreateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Add user message to history
	userMessage := &models.AIComposerMessage{
		SessionID: session.ID,
		Role:      models.AIRoleUser,
		Content:   req.Prompt,
		Metadata:  req.Context,
	}
	if err := s.repo.AddMessage(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Build conversation history for LLM
	messages, err := s.repo.GetSessionMessages(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	llmMessages := []LLMMessage{{Role: "system", Content: systemPrompt}}
	for _, msg := range messages {
		llmMessages = append(llmMessages, LLMMessage{Role: msg.Role, Content: msg.Content})
	}

	// Call LLM
	llmResp, err := s.llmClient.Complete(ctx, llmMessages, LLMOptions{
		MaxTokens:   2000,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get AI response: %w", err)
	}

	// Save assistant response
	assistantMessage := &models.AIComposerMessage{
		SessionID: session.ID,
		Role:      models.AIRoleAssistant,
		Content:   llmResp.Content,
	}
	if err := s.repo.AddMessage(ctx, assistantMessage); err != nil {
		s.logger.Error("Failed to save assistant message", map[string]interface{}{"error": err})
	}

	// Parse response and extract configuration
	response := &models.AIComposerResponse{
		SessionID: session.ID,
		Message:   llmResp.Content,
	}

	// Try to extract JSON configuration from response
	config, err := s.extractConfig(llmResp.Content, session.ID, tenantID)
	if err == nil && config != nil {
		// Validate the generated config
		validationErrors := s.validateConfig(config)
		if len(validationErrors) > 0 {
			config.ValidationStatus = models.AIValidationWarnings
			config.ValidationErrors = validationErrors
		} else {
			config.ValidationStatus = models.AIValidationValid
		}

		if err := s.repo.SaveGeneratedConfig(ctx, config); err != nil {
			s.logger.Error("Failed to save generated config", map[string]interface{}{"error": err})
		} else {
			response.GeneratedConfig = config
		}
	}

	// Check if AI is asking for more info
	response.NeedsMoreInfo, response.Questions = s.detectQuestions(llmResp.Content)

	// Generate suggestions based on context
	response.Suggestions = s.generateSuggestions(session.Context, req.Prompt)

	return response, nil
}

// extractConfig extracts JSON configuration from LLM response
func (s *AIComposerService) extractConfig(content string, sessionID, tenantID uuid.UUID) (*models.AIComposerGeneratedConfig, error) {
	// Look for JSON in code blocks
	jsonRegex := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := jsonRegex.FindStringSubmatch(content)

	var jsonStr string
	if len(matches) > 1 {
		jsonStr = strings.TrimSpace(matches[1])
	} else {
		// Try to find raw JSON object
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start != -1 && end > start {
			jsonStr = content[start : end+1]
		}
	}

	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON configuration found")
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &configMap); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	config := &models.AIComposerGeneratedConfig{
		ID:              uuid.New(),
		SessionID:       sessionID,
		TenantID:        tenantID,
		ConfigType:      models.AIConfigTypeEndpoint,
		GeneratedConfig: configMap,
	}

	// Extract transformation code if present
	if transform, ok := configMap["transformation"].(string); ok {
		config.TransformationCode = transform
		config.ConfigType = models.AIConfigTypeTransformation
	}

	return config, nil
}

// validateConfig validates the generated configuration
func (s *AIComposerService) validateConfig(config *models.AIComposerGeneratedConfig) []string {
	var errors []string

	// Validate URL if present
	if urlStr, ok := config.GeneratedConfig["url"].(string); ok {
		if _, err := url.ParseRequestURI(urlStr); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid URL: %s", err.Error()))
		}
	}

	// Validate HTTP method if present
	if method, ok := config.GeneratedConfig["method"].(string); ok {
		validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
		if !validMethods[strings.ToUpper(method)] {
			errors = append(errors, fmt.Sprintf("Invalid HTTP method: %s", method))
		}
	}

	// Validate transformation code if present
	if config.TransformationCode != "" {
		if valid, jsErrors := s.jsValidator.Validate(config.TransformationCode); !valid {
			errors = append(errors, jsErrors...)
		}
	}

	// Validate retry config if present
	if retryConfig, ok := config.GeneratedConfig["retry_config"].(map[string]interface{}); ok {
		if maxAttempts, ok := retryConfig["max_attempts"].(float64); ok {
			if maxAttempts < 1 || maxAttempts > 10 {
				errors = append(errors, "max_attempts must be between 1 and 10")
			}
		}
	}

	return errors
}

// detectQuestions checks if AI is asking for more information
func (s *AIComposerService) detectQuestions(content string) (bool, []string) {
	questionPatterns := []string{
		`\?(?:\s|$)`,
		`(?i)could you (?:please )?(?:provide|specify|clarify|tell)`,
		`(?i)what (?:is|are|would|should)`,
		`(?i)which (?:one|endpoint|service)`,
		`(?i)do you (?:want|need|prefer)`,
	}

	var questions []string
	for _, pattern := range questionPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(content) {
			// Extract sentences that look like questions
			sentences := strings.Split(content, ".")
			for _, sentence := range sentences {
				sentence = strings.TrimSpace(sentence)
				if strings.Contains(sentence, "?") {
					questions = append(questions, sentence)
				}
			}
			break
		}
	}

	return len(questions) > 0, questions
}

// generateSuggestions provides contextual suggestions
func (s *AIComposerService) generateSuggestions(context map[string]interface{}, prompt string) []string {
	suggestions := []string{}
	promptLower := strings.ToLower(prompt)

	if strings.Contains(promptLower, "slack") {
		suggestions = append(suggestions, "Add error handling for Slack rate limits")
		suggestions = append(suggestions, "Include user mention formatting")
	}
	if strings.Contains(promptLower, "email") || strings.Contains(promptLower, "sendgrid") {
		suggestions = append(suggestions, "Add email template support")
		suggestions = append(suggestions, "Configure bounce handling webhook")
	}
	if strings.Contains(promptLower, "payment") || strings.Contains(promptLower, "stripe") {
		suggestions = append(suggestions, "Verify webhook signatures for security")
		suggestions = append(suggestions, "Add idempotency handling for payments")
	}
	if strings.Contains(promptLower, "transform") || strings.Contains(promptLower, "filter") {
		suggestions = append(suggestions, "Test transformation with sample data")
		suggestions = append(suggestions, "Add null checks for optional fields")
	}

	return suggestions
}

// ApplyConfig applies a generated configuration to create actual webhook resources
func (s *AIComposerService) ApplyConfig(ctx context.Context, tenantID, configID uuid.UUID) error {
	config, err := s.repo.GetGeneratedConfig(ctx, configID)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config.TenantID != tenantID {
		return fmt.Errorf("config not found")
	}

	if config.ValidationStatus == models.AIValidationInvalid {
		return fmt.Errorf("cannot apply invalid configuration")
	}

	// Create webhook endpoint from config
	endpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenantID,
		IsActive: true,
	}

	if urlStr, ok := config.GeneratedConfig["url"].(string); ok {
		endpoint.URL = urlStr
	} else {
		return fmt.Errorf("configuration missing required URL")
	}

	if headers, ok := config.GeneratedConfig["headers"].(map[string]interface{}); ok {
		endpoint.CustomHeaders = make(map[string]string)
		for k, v := range headers {
			if vs, ok := v.(string); ok {
				endpoint.CustomHeaders[k] = vs
			}
		}
	}

	if retryConfig, ok := config.GeneratedConfig["retry_config"].(map[string]interface{}); ok {
		endpoint.RetryConfig = models.RetryConfiguration{
			MaxAttempts:       int(getFloatOrDefault(retryConfig, "max_attempts", 5)),
			InitialDelayMs:    int(getFloatOrDefault(retryConfig, "initial_delay_ms", 1000)),
			MaxDelayMs:        int(getFloatOrDefault(retryConfig, "max_delay_ms", 300000)),
			BackoffMultiplier: int(getFloatOrDefault(retryConfig, "backoff_multiplier", 2)),
		}
	} else {
		endpoint.RetryConfig = models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		}
	}

	// Create the endpoint
	if err := s.webhookRepo.Create(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to create webhook endpoint: %w", err)
	}

	// Mark config as applied
	if err := s.repo.MarkConfigApplied(ctx, configID); err != nil {
		s.logger.Error("Failed to mark config as applied", map[string]interface{}{"error": err})
	}

	return nil
}

func getFloatOrDefault(m map[string]interface{}, key string, def float64) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return def
}

// GetTemplates returns available prompt templates
func (s *AIComposerService) GetTemplates(ctx context.Context, category string) ([]*models.AIComposerTemplate, error) {
	return s.repo.GetTemplates(ctx, category)
}

// SubmitFeedback records user feedback on generated configs
func (s *AIComposerService) SubmitFeedback(ctx context.Context, feedback *models.AIComposerFeedback) error {
	return s.repo.SaveFeedback(ctx, feedback)
}

// DefaultLLMClient is a simple HTTP-based LLM client
type DefaultLLMClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewDefaultLLMClient(apiKey, baseURL string) *DefaultLLMClient {
	return &DefaultLLMClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *DefaultLLMClient) Complete(ctx context.Context, messages []LLMMessage, options LLMOptions) (*LLMResponse, error) {
	// This would implement the actual API call to OpenAI/Anthropic/etc.
	// For now, return a placeholder that indicates the service needs configuration
	if c.apiKey == "" {
		return &LLMResponse{
			Content: "AI service not configured. Please provide a valid API key in the configuration. " +
				"Once configured, I can help you create webhook configurations from natural language descriptions.",
			FinishReason: "incomplete",
		}, nil
	}

	reqBody := map[string]interface{}{
		"model":       options.Model,
		"messages":    messages,
		"max_tokens":  options.MaxTokens,
		"temperature": options.Temperature,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	return &LLMResponse{
		Content:      result.Choices[0].Message.Content,
		FinishReason: result.Choices[0].FinishReason,
		Usage:        result.Usage,
	}, nil
}

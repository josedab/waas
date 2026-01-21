package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LLMClient provides integration with language models for AI-powered debugging
type LLMClient interface {
	Analyze(ctx context.Context, prompt string) (string, error)
	GenerateTransform(ctx context.Context, description string, input, output interface{}) (*TransformGenerateResponse, error)
}

// LLMConfig holds configuration for the LLM client
type LLMConfig struct {
	Provider    string // "openai", "anthropic", "local"
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	BaseURL     string
}

// DefaultLLMConfig returns default LLM configuration
func DefaultLLMConfig() *LLMConfig {
	return &LLMConfig{
		Provider:    "openai",
		Model:       "gpt-4-turbo-preview",
		MaxTokens:   2048,
		Temperature: 0.3,
		Timeout:     30 * time.Second,
		BaseURL:     "https://api.openai.com/v1",
	}
}

// OpenAIClient implements LLMClient using OpenAI API
type OpenAIClient struct {
	config     *LLMConfig
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config *LLMConfig) *OpenAIClient {
	if config == nil {
		config = DefaultLLMConfig()
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	
	return &OpenAIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Analyze sends a prompt to the LLM and returns the response
func (c *OpenAIClient) Analyze(ctx context.Context, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: c.config.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}
	
	return result.Choices[0].Message.Content, nil
}

// GenerateTransform generates a transformation script using the LLM
func (c *OpenAIClient) GenerateTransform(ctx context.Context, description string, input, output interface{}) (*TransformGenerateResponse, error) {
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	outputJSON, _ := json.MarshalIndent(output, "", "  ")
	
	prompt := fmt.Sprintf(`Generate a JavaScript transformation function for webhook payloads.

Description: %s

Input Example:
%s

Desired Output:
%s

Requirements:
1. Return a JavaScript function body that transforms 'payload' variable to the desired output
2. Use only the built-in helpers: clone, get, set, pick, omit, merge, formatDate, uuid, hash
3. Handle edge cases (missing fields, null values)
4. Keep the code concise and efficient

Respond in JSON format:
{
  "script": "// your JavaScript code here",
  "explanation": "Brief explanation of the transformation",
  "warnings": ["any potential issues or edge cases"]
}`, description, string(inputJSON), string(outputJSON))
	
	response, err := c.Analyze(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	var result TransformGenerateResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON parsing fails, try to extract the script
		result.Script = response
		result.Explanation = "Generated transformation script"
		result.Confidence = 0.7
	} else {
		result.Confidence = 0.9
	}
	
	return &result, nil
}

// AnthropicClient implements LLMClient using Anthropic Claude API
type AnthropicClient struct {
	config     *LLMConfig
	httpClient *http.Client
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(config *LLMConfig) *AnthropicClient {
	if config == nil {
		config = &LLMConfig{
			Provider:    "anthropic",
			Model:       "claude-3-sonnet-20240229",
			MaxTokens:   2048,
			Temperature: 0.3,
			Timeout:     30 * time.Second,
			BaseURL:     "https://api.anthropic.com/v1",
		}
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	
	return &AnthropicClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

type anthropicRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	System string `json:"system,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Analyze sends a prompt to Claude and returns the response
func (c *AnthropicClient) Analyze(ctx context.Context, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     c.config.Model,
		MaxTokens: c.config.MaxTokens,
		System:    systemPrompt,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		},
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	
	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response content returned")
	}
	
	return result.Content[0].Text, nil
}

// GenerateTransform generates a transformation script using Claude
func (c *AnthropicClient) GenerateTransform(ctx context.Context, description string, input, output interface{}) (*TransformGenerateResponse, error) {
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	outputJSON, _ := json.MarshalIndent(output, "", "  ")
	
	prompt := fmt.Sprintf(`Generate a JavaScript transformation for webhooks.

Description: %s

Input: %s
Output: %s

Return JSON: {"script": "...", "explanation": "...", "warnings": [...]}`, description, string(inputJSON), string(outputJSON))
	
	response, err := c.Analyze(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	var result TransformGenerateResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		result.Script = response
		result.Explanation = "Generated transformation"
		result.Confidence = 0.7
	} else {
		result.Confidence = 0.9
	}
	
	return &result, nil
}

// LocalClient provides a fallback client that uses rule-based analysis
type LocalClient struct{}

// NewLocalClient creates a local (non-LLM) client
func NewLocalClient() *LocalClient {
	return &LocalClient{}
}

// Analyze provides rule-based analysis without LLM
func (c *LocalClient) Analyze(ctx context.Context, prompt string) (string, error) {
	return `{"root_cause": "Analysis performed using local rules", "explanation": "LLM not configured - using pattern-based analysis"}`, nil
}

// GenerateTransform returns a template transformation
func (c *LocalClient) GenerateTransform(ctx context.Context, description string, input, output interface{}) (*TransformGenerateResponse, error) {
	return &TransformGenerateResponse{
		Script:      "// Manual transformation required\nreturn payload;",
		Explanation: "LLM not configured. Please write the transformation manually.",
		Confidence:  0.3,
		Warnings:    []string{"Automatic transformation generation requires LLM configuration"},
	}, nil
}

// CreateLLMClient creates the appropriate LLM client based on configuration
func CreateLLMClient(config *LLMConfig) LLMClient {
	if config == nil {
		config = DefaultLLMConfig()
	}
	
	switch config.Provider {
	case "anthropic":
		return NewAnthropicClient(config)
	case "openai":
		return NewOpenAIClient(config)
	case "local":
		return NewLocalClient()
	default:
		// Check for API keys
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			return NewAnthropicClient(config)
		}
		if os.Getenv("OPENAI_API_KEY") != "" {
			return NewOpenAIClient(config)
		}
		return NewLocalClient()
	}
}

const systemPrompt = `You are an expert webhook debugging assistant. Your role is to analyze webhook delivery failures and provide actionable debugging guidance.

When analyzing errors:
1. Identify the root cause based on error messages, HTTP status codes, and response bodies
2. Classify errors into categories: network, timeout, authentication, rate_limit, server_error, client_error, payload, certificate, dns
3. Determine if the error is retryable
4. Provide specific, actionable suggestions

When generating transformations:
1. Write clean, efficient JavaScript
2. Use available helpers: clone, get, set, pick, omit, merge, formatDate, uuid, hash
3. Handle edge cases and null values
4. Keep transformations simple and maintainable

Always respond with structured JSON when asked for analysis or code generation.`

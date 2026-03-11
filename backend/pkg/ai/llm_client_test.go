package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIClient_Analyze_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		resp := openAIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "analysis result"}},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "test-key",
		Model:   "gpt-4",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := client.Analyze(context.Background(), "test prompt")
	require.NoError(t, err)
	assert.Equal(t, "analysis result", result)
}

func TestOpenAIClient_Analyze_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "invalid api key"},
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "bad-key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := client.Analyze(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
	// Verify API key is NOT leaked in the error message
	assert.NotContains(t, err.Error(), "bad-key")
}

func TestOpenAIClient_Analyze_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{Choices: nil}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := client.Analyze(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no response choices")
}

func TestOpenAIClient_Analyze_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all{{{"))
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := client.Analyze(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestOpenAIClient_Analyze_NetworkTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 100 * time.Millisecond,
	})

	_, err := client.Analyze(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
}

func TestOpenAIClient_Analyze_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Analyze(ctx, "test")
	assert.Error(t, err)
}

func TestOpenAIClient_GenerateTransform_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: `{"script":"return payload;","explanation":"identity transform","warnings":[]}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := client.GenerateTransform(context.Background(), "identity", map[string]string{"a": "b"}, map[string]string{"a": "b"})
	require.NoError(t, err)
	assert.Equal(t, "return payload;", result.Script)
	assert.Equal(t, 0.9, result.Confidence)
}

func TestOpenAIClient_GenerateTransform_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "return payload;"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := client.GenerateTransform(context.Background(), "desc", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "return payload;", result.Script)
	assert.Equal(t, 0.7, result.Confidence)
}

func TestAnthropicClient_Analyze_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		resp := anthropicResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: "claude result"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient(&LLMConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Timeout:  5 * time.Second,
	})

	result, err := client.Analyze(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "claude result", result)
}

func TestAnthropicClient_Analyze_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{Content: nil}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := client.Analyze(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no response content")
}

func TestAnthropicClient_Analyze_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "rate limited"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient(&LLMConfig{
		APIKey:  "key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := client.Analyze(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestLocalClient_Analyze(t *testing.T) {
	client := NewLocalClient()

	result, err := client.Analyze(context.Background(), "anything")
	require.NoError(t, err)
	assert.Contains(t, result, "root_cause")
	assert.Contains(t, result, "local rules")

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)
}

func TestLocalClient_GenerateTransform(t *testing.T) {
	client := NewLocalClient()

	result, err := client.GenerateTransform(context.Background(), "test", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, result.Script, "return payload")
	assert.Equal(t, 0.3, result.Confidence)
	assert.NotEmpty(t, result.Warnings)
}

func TestCreateLLMClient_ExplicitProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"openai provider", "openai", "*ai.OpenAIClient"},
		{"anthropic provider", "anthropic", "*ai.AnthropicClient"},
		{"local provider", "local", "*ai.LocalClient"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := CreateLLMClient(&LLMConfig{Provider: tt.provider})
			assert.NotNil(t, client)
		})
	}
}

func TestCreateLLMClient_EnvVarFallback(t *testing.T) {
	t.Run("falls back to local when no keys set", func(t *testing.T) {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")

		client := CreateLLMClient(&LLMConfig{Provider: "auto"})
		_, isLocal := client.(*LocalClient)
		assert.True(t, isLocal, "should fall back to LocalClient")
	})
}

func TestCreateLLMClient_NilConfig(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")

	client := CreateLLMClient(nil)
	assert.NotNil(t, client)
}

func TestDefaultLLMConfig(t *testing.T) {
	config := DefaultLLMConfig()
	assert.Equal(t, "openai", config.Provider)
	assert.Equal(t, "gpt-4-turbo-preview", config.Model)
	assert.Equal(t, 2048, config.MaxTokens)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

func TestNewOpenAIClient_DefaultConfig(t *testing.T) {
	client := NewOpenAIClient(nil)
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.openai.com/v1", client.config.BaseURL)
}

func TestNewAnthropicClient_DefaultConfig(t *testing.T) {
	client := NewAnthropicClient(nil)
	assert.NotNil(t, client)
	assert.Equal(t, "anthropic", client.config.Provider)
}

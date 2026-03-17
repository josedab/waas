package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeRequest_ParsesFlatErrorResponse(t *testing.T) {
	// API returns flat format: {code, message}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "INVALID_REQUEST",
			"message": "Invalid request body",
		})
	}))
	defer server.Close()

	client := NewWithConfig(&Config{APIKey: "test-key", BaseURL: server.URL})
	err := client.makeRequest(context.Background(), "GET", "/test", nil, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "INVALID_REQUEST" {
		t.Errorf("expected code INVALID_REQUEST, got %q", apiErr.Code)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestMakeRequest_ParsesNestedErrorResponse(t *testing.T) {
	// Legacy nested format: {"error": {code, message}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "Invalid API key",
			},
		})
	}))
	defer server.Close()

	client := NewWithConfig(&Config{APIKey: "bad-key", BaseURL: server.URL})
	err := client.makeRequest(context.Background(), "GET", "/test", nil, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got %q", apiErr.Code)
	}
}

func TestMakeRequest_FallbackForPlainTextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewWithConfig(&Config{APIKey: "test-key", BaseURL: server.URL})
	err := client.makeRequest(context.Background(), "GET", "/test", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "UNKNOWN_ERROR" {
		t.Errorf("expected code UNKNOWN_ERROR, got %q", apiErr.Code)
	}
}

func TestMakeRequest_SuccessfulResponse(t *testing.T) {
	type testResp struct {
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResp{Name: "test"})
	}))
	defer server.Close()

	client := NewWithConfig(&Config{APIKey: "test-key", BaseURL: server.URL})
	var result testResp
	err := client.makeRequest(context.Background(), "GET", "/test", nil, &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("expected name 'test', got %q", result.Name)
	}
}

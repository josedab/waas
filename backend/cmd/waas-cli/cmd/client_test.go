package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================
// Request Construction
// =====================

func TestClient_DoRequest_Headers(t *testing.T) {
	var capturedReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-api-key")
	resp, err := client.doRequest("GET", "/api/v1/tenant", nil)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "Bearer test-api-key", capturedReq.Header.Get("Authorization"))
	assert.Equal(t, "application/json", capturedReq.Header.Get("Content-Type"))
	assert.Equal(t, "application/json", capturedReq.Header.Get("Accept"))
}

func TestClient_DoRequest_WithBody(t *testing.T) {
	var capturedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	body := map[string]string{"url": "https://example.com"}
	resp, err := client.doRequest("POST", "/api/v1/endpoints", body)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "https://example.com", capturedBody["url"])
}

func TestClient_DoRequest_CorrectMethod(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var capturedMethod string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedMethod = r.Method
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			client := NewClient(srv.URL, "key")
			resp, err := client.doRequest(method, "/test", nil)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, method, capturedMethod)
		})
	}
}

// =====================
// Response Parsing
// =====================

func TestParseResponse_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test-tenant"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	tenant, err := client.GetTenant()
	require.NoError(t, err)
	assert.Equal(t, "test-tenant", tenant.Name)
}

func TestParseResponse_ErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{Code: "not_found", Message: "tenant not found"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	_, err := client.GetTenant()
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, "not_found", apiErr.Code)
	assert.Equal(t, "tenant not found", apiErr.Message)
}

func TestParseResponse_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json at all"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	_, err := client.GetTenant()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestParseResponse_ErrorWithMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error text"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	_, err := client.GetTenant()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed with status 500")
}

// =====================
// Client Methods
// =====================

func TestClient_GetTenant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/tenant", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		json.NewEncoder(w).Encode(Tenant{ID: "t-1", Name: "Test"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	tenant, err := client.GetTenant()
	require.NoError(t, err)
	assert.Equal(t, "t-1", tenant.ID)
}

func TestClient_ListEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/webhooks/endpoints", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"endpoints": []Endpoint{
				{ID: "ep-1", URL: "https://example.com/hook"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	endpoints, err := client.ListEndpoints()
	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	assert.Equal(t, "ep-1", endpoints[0].ID)
}

func TestClient_CreateEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/webhooks/endpoints", r.URL.Path)

		var req CreateEndpointRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "https://example.com/hook", req.URL)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Endpoint{ID: "ep-new", URL: req.URL})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	ep, err := client.CreateEndpoint(&CreateEndpointRequest{URL: "https://example.com/hook"})
	require.NoError(t, err)
	assert.Equal(t, "ep-new", ep.ID)
}

func TestClient_DeleteEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v1/webhooks/endpoints/ep-1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	err := client.DeleteEndpoint("ep-1")
	require.NoError(t, err)
}

func TestClient_SendWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		json.NewEncoder(w).Encode(SendWebhookResponse{
			DeliveryID: "del-1",
			Status:     "queued",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	result, err := client.SendWebhook(&SendWebhookRequest{
		EndpointID: "ep-1",
		Payload:    json.RawMessage(`{"event":"test"}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "del-1", result.DeliveryID)
}

func TestClient_ListDeliveries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/deliveries")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deliveries": []Delivery{{ID: "d-1", Status: "delivered"}},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	deliveries, err := client.ListDeliveries("", 10)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, "delivered", deliveries[0].Status)
}

// =====================
// Timeout handling
// =====================

func TestClient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server — the default client timeout is 30s
		// We won't actually wait; just verify the client has a timeout set
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	assert.NotNil(t, client.httpClient.Timeout)
	assert.True(t, client.httpClient.Timeout > 0)
}

// =====================
// APIError
// =====================

func TestAPIError_ErrorString(t *testing.T) {
	err := &APIError{Code: "rate_limited", Message: "too many requests"}
	assert.Equal(t, "rate_limited: too many requests", err.Error())
}

// =====================
// NewClient
// =====================

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com", "my-key")
	assert.Equal(t, "https://api.example.com", client.baseURL)
	assert.Equal(t, "my-key", client.apiKey)
	assert.NotNil(t, client.httpClient)
}

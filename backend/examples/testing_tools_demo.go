package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// This demo shows how to use the webhook testing and debugging tools

func main() {
	fmt.Println("Webhook Testing Tools Demo")
	fmt.Println("==========================")

	// Base URL for the webhook platform API
	baseURL := "http://localhost:8080/api/v1"
	
	// Example API key (in real usage, this would be obtained from tenant registration)
	apiKey := "your-api-key-here"

	// 1. Create a test endpoint for receiving webhooks
	fmt.Println("\n1. Creating a test endpoint...")
	testEndpoint := createTestEndpoint(baseURL, apiKey)
	if testEndpoint != nil {
		fmt.Printf("✓ Test endpoint created: %s\n", testEndpoint.URL)
		fmt.Printf("  ID: %s\n", testEndpoint.ID)
		fmt.Printf("  Expires: %s\n", testEndpoint.ExpiresAt.Format(time.RFC3339))
	}

	// 2. Test a webhook delivery to an external endpoint
	fmt.Println("\n2. Testing webhook delivery...")
	testResult := testWebhookDelivery(baseURL, apiKey, "https://httpbin.org/post")
	if testResult != nil {
		fmt.Printf("✓ Webhook test completed\n")
		fmt.Printf("  Status: %s\n", testResult.Status)
		fmt.Printf("  HTTP Status: %d\n", *testResult.HTTPStatus)
		fmt.Printf("  Latency: %dms\n", *testResult.Latency)
		fmt.Printf("  Request ID: %s\n", testResult.RequestID)
	}

	// 3. Test webhook delivery to our test endpoint
	if testEndpoint != nil {
		fmt.Println("\n3. Testing delivery to test endpoint...")
		testResult2 := testWebhookDelivery(baseURL, apiKey, testEndpoint.URL)
		if testResult2 != nil {
			fmt.Printf("✓ Test endpoint webhook completed\n")
			fmt.Printf("  Status: %s\n", testResult2.Status)
			if testResult2.HTTPStatus != nil {
				fmt.Printf("  HTTP Status: %d\n", *testResult2.HTTPStatus)
			}
			if testResult2.Latency != nil {
				fmt.Printf("  Latency: %dms\n", *testResult2.Latency)
			}
		}
	}

	fmt.Println("\n4. Example: Inspecting a delivery (requires actual delivery ID)")
	fmt.Println("   GET /api/v1/webhooks/deliveries/{delivery-id}/inspect")
	fmt.Println("   This would show detailed request/response information, timeline, and error details")

	fmt.Println("\n5. Example: Real-time updates via WebSocket")
	fmt.Println("   Connect to: ws://localhost:8080/api/v1/webhooks/realtime")
	fmt.Println("   Receive real-time delivery status updates")

	fmt.Println("\nDemo completed! 🎉")
}

type TestEndpointResponse struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
}

type TestWebhookResponse struct {
	TestID      string  `json:"test_id"`
	URL         string  `json:"url"`
	Status      string  `json:"status"`
	HTTPStatus  *int    `json:"http_status"`
	Latency     *int64  `json:"latency_ms"`
	RequestID   string  `json:"request_id"`
	ErrorMessage *string `json:"error_message"`
}

func createTestEndpoint(baseURL, apiKey string) *TestEndpointResponse {
	reqBody := map[string]interface{}{
		"name":        "Demo Test Endpoint",
		"description": "Created by testing tools demo",
		"ttl":         3600, // 1 hour
		"headers": map[string]string{
			"Authorization": "Bearer demo-token",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	
	req, err := http.NewRequest("POST", baseURL+"/webhooks/test/endpoints", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("✗ Error creating request: %v\n", err)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("✗ Error making request: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("✗ Request failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil
	}

	var result TestEndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("✗ Error decoding response: %v\n", err)
		return nil
	}

	return &result
}

func testWebhookDelivery(baseURL, apiKey, targetURL string) *TestWebhookResponse {
	reqBody := map[string]interface{}{
		"url": targetURL,
		"payload": map[string]interface{}{
			"event":     "test.webhook",
			"timestamp": time.Now().Format(time.RFC3339),
			"data": map[string]interface{}{
				"message": "Hello from webhook testing tools!",
				"demo":    true,
			},
		},
		"headers": map[string]string{
			"X-Demo-Header": "testing-tools",
		},
		"method":  "POST",
		"timeout": 30,
	}

	jsonBody, _ := json.Marshal(reqBody)
	
	req, err := http.NewRequest("POST", baseURL+"/webhooks/test", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("✗ Error creating request: %v\n", err)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("✗ Error making request: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("✗ Request failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil
	}

	var result TestWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("✗ Error decoding response: %v\n", err)
		return nil
	}

	return &result
}
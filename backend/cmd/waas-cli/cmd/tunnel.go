package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var (
	tunnelPort    int
	tunnelPath    string
	tunnelVerbose bool
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Create a local development tunnel for webhooks",
	Long: `Creates a public URL that relays webhooks to your local server.
Replaces ngrok for webhook development. Auto-registers an endpoint and streams live logs.

Examples:
  waas tunnel --port 3000                   # Relay to localhost:3000
  waas tunnel --port 8080 --path /webhooks  # Forward to specific path
  waas tunnel --port 3000 --verbose         # Show full request/response details`,
	RunE: runTunnel,
}

func init() {
	tunnelCmd.Flags().IntVarP(&tunnelPort, "port", "p", 3000, "Local port to forward webhooks to")
	tunnelCmd.Flags().StringVar(&tunnelPath, "path", "/", "Local path to forward to")
	tunnelCmd.Flags().BoolVar(&tunnelVerbose, "verbose", false, "Show full request/response details")
	rootCmd.AddCommand(tunnelCmd)
}

func runTunnel(cmd *cobra.Command, args []string) error {
	fmt.Println("🔗 Creating tunnel endpoint...")

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	// Create a temporary test endpoint
	createURL := getAPIURL() + "/api/v1/webhooks/test/endpoints"
	payload := map[string]interface{}{
		"ttl_seconds": 3600,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tunnel payload: %w", err)
	}

	req, err := http.NewRequest("POST", createURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create tunnel request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create tunnel endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read tunnel response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create endpoint (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var endpoint map[string]interface{}
	if err := json.Unmarshal(respBody, &endpoint); err != nil {
		return fmt.Errorf("failed to parse endpoint response: %w", err)
	}

	endpointID, ok := endpoint["id"].(string)
	if !ok {
		return fmt.Errorf("endpoint response missing 'id' field")
	}
	endpointURL, _ := endpoint["url"].(string)
	if endpointURL == "" {
		endpointURL = fmt.Sprintf("%s/test/%s", getAPIURL(), endpointID)
	}

	// Auto-register as a real endpoint for webhook delivery
	_, regErr := client.CreateEndpoint(&CreateEndpointRequest{
		URL: endpointURL,
	})
	if regErr != nil && verbose {
		fmt.Fprintf(os.Stderr, "   ⚠ Auto-registration note: %v\n", regErr)
	}

	localTarget := fmt.Sprintf("http://localhost:%d%s", tunnelPort, tunnelPath)
	fmt.Println()
	out.PrintSuccess("Tunnel active")
	out.PrintKeyValue("Public URL", endpointURL)
	out.PrintKeyValue("Forwarding to", localTarget)
	out.PrintKeyValue("Endpoint ID", endpointID)
	out.PrintKeyValue("Expires", time.Now().Add(time.Hour).Format(time.RFC3339))
	fmt.Println()
	fmt.Println("   Ready! Send webhooks to the public URL above.")
	fmt.Println("   Press Ctrl+C to stop.")

	var delivered uint64
	var failed uint64

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Try WebSocket-based streaming first, fall back to polling
	wsDone := make(chan struct{})
	go func() {
		defer close(wsDone)
		tunnelStreamWS(endpointID, localTarget, &delivered, &failed)
	}()

	// Also run HTTP polling as a backup for received webhooks
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Printf("\n🛑 Tunnel stopped (delivered: %d, failed: %d)\n",
				atomic.LoadUint64(&delivered), atomic.LoadUint64(&failed))
			return nil
		case <-ticker.C:
			tunnelForwardWebhooks(endpointID, localTarget, &delivered, &failed)
		}
	}
}

func tunnelStreamWS(endpointID, localTarget string, delivered, failed *uint64) {
	apiURL := getAPIURL()
	apiKey, err := getAPIKey()
	if err != nil {
		return
	}

	u, err := url.Parse(apiURL)
	if err != nil {
		return
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/api/v1/webhooks/realtime"
	q := u.Query()
	q.Set("endpoint_id", endpointID)
	u.RawQuery = q.Encode()

	header := map[string][]string{
		"Authorization": {"Bearer " + apiKey},
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var event struct {
			DeliveryID string          `json:"delivery_id"`
			Status     string          `json:"status"`
			EventType  string          `json:"event_type"`
			Payload    json.RawMessage `json:"payload"`
			HTTPStatus int             `json:"http_status"`
		}
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		ts := time.Now().Format("15:04:05")
		if len(event.Payload) > 0 {
			localURL := localTarget
			localReq, err := http.NewRequest("POST", localURL, bytes.NewReader(event.Payload))
			if err != nil {
				atomic.AddUint64(failed, 1)
				fmt.Printf("   [%s] \033[31m✗\033[0m failed to create request: %v\n", ts, err)
				continue
			}
			localReq.Header.Set("Content-Type", "application/json")
			localReq.Header.Set("X-WaaS-Delivery-ID", event.DeliveryID)

			localResp, localErr := http.DefaultClient.Do(localReq)
			if localErr != nil {
				atomic.AddUint64(failed, 1)
				fmt.Printf("   [%s] \033[31m✗\033[0m %s → localhost failed: %v\n", ts, out.Truncate(event.DeliveryID, 12), localErr)
			} else {
				localResp.Body.Close()
				atomic.AddUint64(delivered, 1)
				fmt.Printf("   [%s] \033[32m✓\033[0m %s → localhost [%d]\n", ts, out.Truncate(event.DeliveryID, 12), localResp.StatusCode)
			}
		} else {
			fmt.Printf("   [%s] %s %s %s\n", ts, out.ColorStatus(event.Status), event.EventType, out.Truncate(event.DeliveryID, 12))
		}
	}
}

func tunnelForwardWebhooks(endpointID, localTarget string, delivered, failed *uint64) {
	url := fmt.Sprintf("%s/test/%s/receives", getAPIURL(), endpointID)
	apiKey, err := getAPIKey()
	if err != nil {
		return
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var result struct {
		Receives []struct {
			ID      string            `json:"id"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
			Body    string            `json:"body"`
		} `json:"receives"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return
	}

	for _, recv := range result.Receives {
		localReq, err := http.NewRequest(recv.Method, localTarget, bytes.NewBufferString(recv.Body))
		if err != nil {
			atomic.AddUint64(failed, 1)
			fmt.Printf("   [%s] \033[31m✗\033[0m failed to create request: %v\n",
				time.Now().Format("15:04:05"), err)
			continue
		}
		for k, v := range recv.Headers {
			localReq.Header.Set(k, v)
		}
		localReq.Header.Set("X-WaaS-Forwarded", "true")

		ts := time.Now().Format("15:04:05")
		localResp, err := http.DefaultClient.Do(localReq)
		if err != nil {
			atomic.AddUint64(failed, 1)
			fmt.Printf("   [%s] \033[31m✗\033[0m %s %s → failed: %v\n",
				ts, recv.Method, out.Truncate(recv.ID, 8), err)
			if tunnelVerbose {
				fmt.Printf("         Body: %s\n", out.Truncate(recv.Body, 200))
			}
			continue
		}
		localResp.Body.Close()
		atomic.AddUint64(delivered, 1)
		fmt.Printf("   [%s] \033[32m→\033[0m %s %s → localhost [%d]\n",
			ts, recv.Method, out.Truncate(recv.ID, 8), localResp.StatusCode)
		if tunnelVerbose && recv.Body != "" {
			// Indent payload preview
			preview := recv.Body
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("         Payload: %s\n", strings.ReplaceAll(preview, "\n", " "))
		}
	}
}

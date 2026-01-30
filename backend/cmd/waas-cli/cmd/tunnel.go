package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var tunnelPort int

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Create a local development tunnel",
	Long:  "Creates a temporary webhook endpoint and polls for incoming events, forwarding them to a local server",
	RunE:  runTunnel,
}

func init() {
	tunnelCmd.Flags().IntVarP(&tunnelPort, "port", "p", 3000, "Local port to forward webhooks to")
	rootCmd.AddCommand(tunnelCmd)
}

func runTunnel(cmd *cobra.Command, args []string) error {
	fmt.Println("🔗 Creating tunnel endpoint...")

	// Create a test endpoint
	createURL := getAPIURL() + "/api/v1/webhooks/test/endpoints"
	payload := map[string]interface{}{
		"ttl_seconds": 3600,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", createURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", getAPIKey())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create tunnel endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create endpoint (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var endpoint map[string]interface{}
	json.Unmarshal(respBody, &endpoint)

	endpointID, _ := endpoint["id"].(string)
	endpointURL, _ := endpoint["url"].(string)
	if endpointURL == "" {
		endpointURL = fmt.Sprintf("%s/test/%s", getAPIURL(), endpointID)
	}

	fmt.Printf("✅ Tunnel active\n")
	fmt.Printf("   Remote URL:  %s\n", endpointURL)
	fmt.Printf("   Local:       http://localhost:%d\n", tunnelPort)
	fmt.Printf("   Expires:     %s\n", time.Now().Add(time.Hour).Format(time.RFC3339))
	fmt.Println("\n   Waiting for webhooks... (Ctrl+C to stop)")

	// Set up signal handling for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Poll for incoming webhooks and forward to local server
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\n🛑 Tunnel stopped")
			return nil
		case <-ticker.C:
			forwardWebhooks(endpointID, tunnelPort)
		}
	}
}

func forwardWebhooks(endpointID string, port int) {
	url := fmt.Sprintf("%s/test/%s/receives", getAPIURL(), endpointID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-API-Key", getAPIKey())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, _ := io.ReadAll(resp.Body)
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
		localURL := fmt.Sprintf("http://localhost:%d", port)
		localReq, _ := http.NewRequest(recv.Method, localURL, bytes.NewBufferString(recv.Body))
		for k, v := range recv.Headers {
			localReq.Header.Set(k, v)
		}

		localResp, err := http.DefaultClient.Do(localReq)
		if err != nil {
			fmt.Printf("   ⚠ Forward failed: %v\n", err)
			continue
		}
		localResp.Body.Close()
		fmt.Printf("   → %s %s → localhost:%d [%d]\n", recv.Method, recv.ID[:8], port, localResp.StatusCode)
	}
}

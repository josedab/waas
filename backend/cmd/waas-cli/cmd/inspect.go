package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <delivery-id>",
	Short: "Deep inspect a webhook delivery",
	Long:  "Show detailed request/response information for a webhook delivery including headers, payload, timing, and retry history",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	deliveryID := args[0]
	url := getAPIURL() + "/api/v1/webhooks/deliveries/" + deliveryID + "/inspect"

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create inspect request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("failed to inspect delivery: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read inspect response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("inspect failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	if output == "json" {
		fmt.Println(string(body))
		return nil
	}

	var inspection map[string]interface{}
	json.Unmarshal(body, &inspection)

	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  Delivery Inspection: %s\n", deliveryID)
	fmt.Println("═══════════════════════════════════════════")

	if delivery, ok := inspection["delivery"].(map[string]interface{}); ok {
		fmt.Printf("\n  Status:       %v\n", delivery["status"])
		fmt.Printf("  HTTP Status:  %v\n", delivery["http_status_code"])
		fmt.Printf("  Attempt:      %v\n", delivery["attempt_number"])
		fmt.Printf("  Endpoint:     %v\n", delivery["endpoint_id"])
		fmt.Printf("  Created:      %v\n", delivery["created_at"])
	}

	if request, ok := inspection["request"].(map[string]interface{}); ok {
		fmt.Println("\n  ── Request ──")
		fmt.Printf("  Method: %v\n", request["method"])
		fmt.Printf("  URL:    %v\n", request["url"])
		if headers, ok := request["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
		if body, ok := request["body"].(string); ok {
			fmt.Println("\n  Body:")
			prettyPrintJSON(body)
		}
	}

	if response, ok := inspection["response"].(map[string]interface{}); ok {
		fmt.Println("\n  ── Response ──")
		fmt.Printf("  Status:  %v\n", response["status_code"])
		fmt.Printf("  Latency: %vms\n", response["latency_ms"])
		if body, ok := response["body"].(string); ok && body != "" {
			fmt.Println("\n  Body:")
			prettyPrintJSON(body)
		}
	}

	if errMsg, ok := inspection["error"].(string); ok && errMsg != "" {
		fmt.Printf("\n  ⚠ Error: %s\n", errMsg)
	}

	fmt.Println()
	return nil
}

func prettyPrintJSON(s string) {
	var obj interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		fmt.Printf("    %s\n", s)
		return
	}
	pretty, err := json.MarshalIndent(obj, "    ", "  ")
	if err != nil {
		fmt.Printf("    %s\n", s)
		return
	}
	for _, line := range strings.Split(string(pretty), "\n") {
		fmt.Printf("    %s\n", line)
	}
}

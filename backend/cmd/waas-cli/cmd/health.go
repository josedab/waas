package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check API health status",
	Long:  "Perform a health check against the connected WaaS API and display component status",
	RunE:  runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) error {
	url := getAPIURL() + "/health"

	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("❌ API unreachable: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ API unhealthy (HTTP %d)\n", resp.StatusCode)
		return fmt.Errorf("health check failed")
	}

	var health map[string]interface{}
	json.Unmarshal(body, &health)

	fmt.Printf("✅ API healthy (%s, %dms)\n", getAPIURL(), latency.Milliseconds())

	if status, ok := health["status"].(string); ok {
		fmt.Printf("   Status:  %s\n", status)
	}
	if version, ok := health["version"].(string); ok {
		fmt.Printf("   Version: %s\n", version)
	}
	if components, ok := health["components"].(map[string]interface{}); ok {
		for name, comp := range components {
			if compMap, ok := comp.(map[string]interface{}); ok {
				status := compMap["status"]
				fmt.Printf("   %-12s %v\n", name+":", status)
			}
		}
	}

	if output == "json" {
		fmt.Println(string(body))
	}

	return nil
}

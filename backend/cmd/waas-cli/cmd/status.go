package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show account status and usage",
	Long: `Display your WAAS account status, including tenant information,
subscription tier, rate limits, and usage statistics.

Examples:
  waas status
  waas status -o json`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	tenant, err := client.GetTenant()
	if err != nil {
		return fmt.Errorf("failed to get account status: %w", err)
	}

	endpoints, err := client.ListEndpoints()
	if err != nil {
		// Non-fatal, just show what we have
		endpoints = nil
	}

	if output == "json" {
		result := map[string]interface{}{
			"tenant":         tenant,
			"endpoint_count": len(endpoints),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Println("WAAS Account Status")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Tenant:          %s\n", tenant.Name)
	fmt.Printf("Tenant ID:       %s\n", tenant.ID)
	fmt.Printf("Subscription:    %s\n", tenant.SubscriptionTier)
	fmt.Printf("Rate Limit:      %d requests/minute\n", tenant.RateLimitPerMin)
	fmt.Printf("Monthly Quota:   %d webhooks\n", tenant.MonthlyQuota)
	fmt.Printf("Endpoints:       %d\n", len(endpoints))
	fmt.Printf("Created:         %s\n", tenant.CreatedAt.Format("2006-01-02"))
	fmt.Println("═══════════════════════════════════════")

	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("waas version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	out "webhook-platform/cmd/waas-cli/output"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage tenants",
	Long:  `Create, view, and manage your WaaS tenant account.`,
}

var tenantCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tenant",
	Long: `Create a new tenant account and receive an API key.

Examples:
  waas tenant create --name "my-app" --email "dev@example.com"`,
	RunE: runTenantCreate,
}

var tenantShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current tenant details",
	Long: `Display information about the currently authenticated tenant.

Examples:
  waas tenant show
  waas tenant show -o json`,
	RunE: runTenantShow,
}

var tenantRegenerateKeyCmd = &cobra.Command{
	Use:   "regenerate-key",
	Short: "Regenerate API key",
	Long: `Regenerate the API key for the current tenant. The old key will be invalidated.

Examples:
  waas tenant regenerate-key`,
	RunE: runTenantRegenerateKey,
}

var (
	tenantName  string
	tenantEmail string
)

func init() {
	rootCmd.AddCommand(tenantCmd)

	tenantCmd.AddCommand(tenantCreateCmd)
	tenantCmd.AddCommand(tenantShowCmd)
	tenantCmd.AddCommand(tenantRegenerateKeyCmd)

	tenantCreateCmd.Flags().StringVar(&tenantName, "name", "", "Tenant name (required)")
	tenantCreateCmd.Flags().StringVar(&tenantEmail, "email", "", "Contact email (required)")
	tenantCreateCmd.MarkFlagRequired("name")
	tenantCreateCmd.MarkFlagRequired("email")
}

func runTenantCreate(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), "")

	result, err := client.CreateTenant(tenantName, tenantEmail)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	if output == "json" {
		return jsonOutput(result)
	}

	out.PrintSuccess("Tenant created successfully")
	out.PrintKeyValue("Tenant ID", result.Tenant.ID)
	out.PrintKeyValue("Name", result.Tenant.Name)
	out.PrintKeyValue("API Key", result.APIKey)
	out.PrintKeyValue("Subscription", result.Tenant.SubscriptionTier)
	fmt.Println()
	out.PrintWarning("Save your API key! It won't be shown again.")
	fmt.Printf("\nConfigure CLI:  waas config set api-key %s\n", result.APIKey)

	return nil
}

func runTenantShow(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	tenant, err := client.GetTenant()
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if output == "json" {
		return jsonOutput(tenant)
	}

	out.PrintHeader("Tenant Details")
	out.PrintKeyValue("ID", tenant.ID)
	out.PrintKeyValue("Name", tenant.Name)
	out.PrintKeyValue("Subscription", tenant.SubscriptionTier)
	out.PrintKeyValue("Rate Limit", fmt.Sprintf("%d req/min", tenant.RateLimitPerMin))
	out.PrintKeyValue("Monthly Quota", fmt.Sprintf("%d webhooks", tenant.MonthlyQuota))
	out.PrintKeyValue("Created", tenant.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func runTenantRegenerateKey(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	result, err := client.RegenerateAPIKey()
	if err != nil {
		return fmt.Errorf("failed to regenerate API key: %w", err)
	}

	// Update local config with new key
	viper.Set("api_key", result.APIKey)
	home, _ := os.UserHomeDir()
	configPath := home + "/.waas.yaml"
	_ = viper.WriteConfigAs(configPath)

	if output == "json" {
		return jsonOutput(map[string]string{"api_key": result.APIKey})
	}

	out.PrintSuccess("API key regenerated successfully")
	out.PrintKeyValue("New API Key", result.APIKey)
	fmt.Printf("\n  Config updated at: %s\n", configPath)

	return nil
}

// jsonOutput is a helper for JSON output mode
func jsonOutput(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

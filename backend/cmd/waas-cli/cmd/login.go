package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your WAAS API key",
	Long: `Authenticate with the WAAS platform using your API key.

The API key will be saved to your config file for future use.

Examples:
  waas login                           # Interactive login
  waas login --api-key wh_xxx          # Login with API key directly
  waas login --api-url https://api.example.com  # Custom API URL`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	key := viper.GetString("api_key")
	url := viper.GetString("api_url")

	// Interactive mode if no API key provided
	if key == "" {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter your WAAS API key: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		key = strings.TrimSpace(input)
	}

	if key == "" {
		return fmt.Errorf("API key is required")
	}

	// Validate API key format
	if !strings.HasPrefix(key, "wh_") {
		fmt.Fprintln(os.Stderr, "Warning: API key doesn't have expected 'wh_' prefix")
	}

	// Test the API key by making a request to get tenant info
	client := NewClient(url, key)
	tenant, err := client.GetTenant()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save to config file
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	viper.Set("api_key", key)
	viper.Set("api_url", url)

	configPath := home + "/.waas.yaml"
	if err := viper.WriteConfigAs(configPath); err != nil {
		// Try to create the file if it doesn't exist
		if err := viper.SafeWriteConfigAs(configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	fmt.Printf("✓ Successfully authenticated as: %s\n", tenant.Name)
	fmt.Printf("  Subscription: %s\n", tenant.SubscriptionTier)
	fmt.Printf("  Config saved to: %s\n", configPath)

	return nil
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved authentication credentials",
	Long:  `Remove the saved API key from your config file.`,
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := home + "/.waas.yaml"
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	fmt.Println("✓ Successfully logged out")
	return nil
}

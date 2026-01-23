package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	out "webhook-platform/cmd/waas-cli/output"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `View and modify WaaS CLI configuration settings.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a CLI configuration value. The configuration is saved to ~/.waas.yaml.

Available keys:
  api-url     API server URL (default: http://localhost:8080)
  api-key     API key for authentication
  output      Default output format (table, json, yaml)

Examples:
  waas config set api-url http://localhost:8080
  waas config set api-key wh_sk_xxx
  waas config set output json`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long: `Display all current CLI configuration values.

Examples:
  waas config show
  waas config show -o json`,
	RunE: runConfigShow,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		fmt.Println(filepath.Join(home, ".waas.yaml"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
}

// configKeyMap maps user-facing keys to viper keys
var configKeyMap = map[string]string{
	"api-url": "api_url",
	"api-key": "api_key",
	"output":  "output",
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	viperKey, ok := configKeyMap[key]
	if !ok {
		validKeys := make([]string, 0, len(configKeyMap))
		for k := range configKeyMap {
			validKeys = append(validKeys, k)
		}
		sort.Strings(validKeys)
		return fmt.Errorf("unknown config key %q. Valid keys: %s", key, strings.Join(validKeys, ", "))
	}

	viper.Set(viperKey, value)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".waas.yaml")

	// Try to write, create if doesn't exist
	if err := viper.WriteConfigAs(configPath); err != nil {
		if err := viper.SafeWriteConfigAs(configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	out.PrintSuccess(fmt.Sprintf("Set %s = %s", key, maskSensitive(key, value)))
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	config := map[string]string{
		"api-url": viper.GetString("api_url"),
		"api-key": maskSensitive("api-key", viper.GetString("api_key")),
		"output":  viper.GetString("output"),
	}

	// Show config file path
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		home, _ := os.UserHomeDir()
		configFile = filepath.Join(home, ".waas.yaml") + " (not found)"
	}

	if output == "json" {
		return jsonOutput(config)
	}

	out.PrintHeader("CLI Configuration")
	fmt.Printf("  Config file: %s\n\n", configFile)

	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		out.PrintKeyValue(k, config[k])
	}

	return nil
}

// maskSensitive masks sensitive values for display
func maskSensitive(key, value string) string {
	if key == "api-key" && len(value) > 8 {
		return value[:8] + strings.Repeat("*", len(value)-8)
	}
	return value
}

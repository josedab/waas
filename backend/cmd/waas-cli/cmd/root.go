package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	apiURL  string
	apiKey  string
	output  string
)

var rootCmd = &cobra.Command{
	Use:   "waas",
	Short: "WAAS CLI - Webhook-as-a-Service command line tool",
	Long: `WAAS CLI is a command line interface for the Webhook-as-a-Service platform.

It allows you to manage webhook endpoints, send webhooks, view delivery logs,
and test webhook integrations directly from your terminal.

Examples:
  waas login                    # Authenticate with your API key
  waas endpoints list           # List all webhook endpoints
  waas send --endpoint <id>     # Send a webhook to an endpoint
  waas logs --tail              # Stream delivery logs in real-time
  waas replay <delivery-id>     # Replay a failed delivery`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.waas.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "WAAS API URL (default: http://localhost:8080)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table, json, yaml")

	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".waas")
	}

	viper.SetEnvPrefix("WAAS")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("api_url", "http://localhost:8080")

	if err := viper.ReadInConfig(); err == nil {
		// Config file found and successfully parsed
	}
}

func getAPIURL() string {
	return viper.GetString("api_url")
}

func getAPIKey() string {
	key := viper.GetString("api_key")
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: API key not configured. Run 'waas login' first or provide --api-key flag.")
		os.Exit(1)
	}
	return key
}

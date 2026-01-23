package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new WAAS project",
	Long: `Initialize a new WAAS project with configuration files and example payloads.

Creates a .waas/ directory with project configuration, example webhook payloads,
and a Makefile for common operations.

Examples:
  waas init                    # Initialize in current directory
  waas init my-project         # Initialize in a new directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	waasDir := filepath.Join(dir, ".waas")
	if err := os.MkdirAll(waasDir, 0755); err != nil {
		return fmt.Errorf("failed to create .waas directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(waasDir, "payloads"), 0755); err != nil {
		return fmt.Errorf("failed to create payloads directory: %w", err)
	}

	// Write project config
	projectConfig := map[string]interface{}{
		"version":  "1",
		"api_url":  "http://localhost:8080",
		"defaults": map[string]interface{}{
			"max_retries":   5,
			"retry_delay":   "1s",
			"timeout":       "30s",
			"content_type":  "application/json",
		},
		"endpoints": []interface{}{},
	}
	configBytes, _ := json.MarshalIndent(projectConfig, "", "  ")
	configPath := filepath.Join(waasDir, "config.json")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Write example payload
	examplePayload := map[string]interface{}{
		"event": "order.created",
		"data": map[string]interface{}{
			"order_id": "ord_123",
			"amount":   99.99,
			"currency": "USD",
		},
		"timestamp": "2026-01-01T00:00:00Z",
	}
	payloadBytes, _ := json.MarshalIndent(examplePayload, "", "  ")
	payloadPath := filepath.Join(waasDir, "payloads", "example.json")
	if err := os.WriteFile(payloadPath, payloadBytes, 0644); err != nil {
		return fmt.Errorf("failed to write example payload: %w", err)
	}

	// Write .gitignore for secrets
	gitignore := "# WAAS local config\n.waas/secrets/\n.waas/*.key\n"
	gitignorePath := filepath.Join(waasDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	fmt.Printf("✓ WAAS project initialized in %s\n\n", dir)
	fmt.Println("Created:")
	fmt.Printf("  %s          - Project configuration\n", configPath)
	fmt.Printf("  %s  - Example webhook payload\n", payloadPath)
	fmt.Printf("  %s         - Git ignore for secrets\n", gitignorePath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. waas login                              # Authenticate")
	fmt.Println("  2. waas endpoints create --url <your-url>  # Create an endpoint")
	fmt.Println("  3. waas send --endpoint <id> --file .waas/payloads/example.json")

	return nil
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/spf13/cobra"
)

var configExportCmd = &cobra.Command{
	Use:   "config-export",
	Short: "Export current configuration as YAML",
	Long: `Export the current live webhook configuration as a YAML manifest.

Examples:
  waas config-export                           # Export to stdout
  waas config-export -o config.yaml            # Export to file
  waas config-export --env production          # Export specific environment`,
	RunE: runConfigExport,
}

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote configuration between environments",
	Long: `Promote a configuration manifest from one environment to another.
Supports approval gates for production promotions.

Examples:
  waas promote --from dev --to staging         # Promote dev to staging
  waas promote --from staging --to prod        # Promote staging to prod (may require approval)
  waas promote --from staging --to prod --approved-by user@example.com`,
	RunE: runPromote,
}

var (
	configExportOutput string
	configExportEnv    string
	promoteFrom        string
	promoteTo          string
	approvedBy         string
)

func init() {
	rootCmd.AddCommand(configExportCmd)
	rootCmd.AddCommand(promoteCmd)

	configExportCmd.Flags().StringVarP(&configExportOutput, "output", "o", "", "Output file path (default: stdout)")
	configExportCmd.Flags().StringVar(&configExportEnv, "env", "dev", "Environment to export")

	promoteCmd.Flags().StringVar(&promoteFrom, "from", "", "Source environment (required)")
	promoteCmd.Flags().StringVar(&promoteTo, "to", "", "Target environment (required)")
	promoteCmd.Flags().StringVar(&approvedBy, "approved-by", "", "Approver email for gated promotions")
	promoteCmd.MarkFlagRequired("from")
	promoteCmd.MarkFlagRequired("to")
}

func runConfigExport(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	resp, err := client.doRequest("GET", fmt.Sprintf("/api/v1/gitops/export?environment=%s", configExportEnv), nil)
	if err != nil {
		return fmt.Errorf("failed to export configuration: %w", err)
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return fmt.Errorf("failed to export configuration: %w", err)
	}

	content, ok := result["content"].(string)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if configExportOutput != "" {
		dir := filepath.Dir(configExportOutput)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}
		if err := os.WriteFile(configExportOutput, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		out.PrintSuccess(fmt.Sprintf("Configuration exported to %s", configExportOutput))
	} else {
		fmt.Print(content)
	}

	return nil
}

func runPromote(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	resp, err := client.doRequest("POST", "/api/v1/gitops/promote", map[string]interface{}{
		"source_env":  promoteFrom,
		"target_env":  promoteTo,
		"manifest_id": "latest",
		"approved_by": approvedBy,
	})
	if err != nil {
		return fmt.Errorf("promotion failed: %w", err)
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return fmt.Errorf("promotion failed: %w", err)
	}

	if output == "json" {
		return out.PrintJSON(result)
	}

	status, _ := result["status"].(string)
	switch status {
	case "promoted":
		out.PrintSuccess(fmt.Sprintf("Configuration promoted from %s → %s", promoteFrom, promoteTo))
	case "pending_approval":
		out.PrintWarning(fmt.Sprintf("Promotion to %s requires approval. Use --approved-by to provide approver.", promoteTo))
	default:
		out.PrintInfo(fmt.Sprintf("Promotion status: %s", status))
	}

	return nil
}

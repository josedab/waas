package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var contractsCmd = &cobra.Command{
	Use:   "contracts",
	Short: "Manage webhook contracts and run validation",
	Long: `Manage webhook payload contracts for CI/CD-integrated schema validation.

Define expected webhook payload schemas, validate deliveries against contracts,
and detect breaking changes between versions.

Examples:
  waas contracts list
  waas contracts create --name "Order Events" --schema schema.json
  waas contracts validate --contract <id> --payload payload.json
  waas contracts diff --contract <id> --old-version v1 --new-version v2
  waas contracts test --contract <id> --ci`,
}

var contractsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contracts",
	RunE:  runContractsList,
}

var contractsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new contract",
	RunE:  runContractsCreate,
}

var contractsValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a payload against a contract",
	RunE:  runContractsValidate,
}

var contractsDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare contract versions for breaking changes",
	RunE:  runContractsDiff,
}

var contractsTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Run contract validation tests (CI/CD compatible)",
	Long: `Run contract validation tests with exit codes suitable for CI/CD pipelines.

Exit codes:
  0 - All validations passed
  1 - Validation failures detected
  2 - Breaking changes detected`,
	RunE: runContractsTest,
}

var (
	contractName       string
	contractSchema     string
	contractVersion    string
	contractEventType  string
	contractStrictness string
	contractID         string
	contractPayload    string
	contractOldVersion string
	contractNewVersion string
	contractCI         bool
)

func init() {
	rootCmd.AddCommand(contractsCmd)

	contractsCmd.AddCommand(contractsListCmd)
	contractsCmd.AddCommand(contractsCreateCmd)
	contractsCmd.AddCommand(contractsValidateCmd)
	contractsCmd.AddCommand(contractsDiffCmd)
	contractsCmd.AddCommand(contractsTestCmd)

	contractsCreateCmd.Flags().StringVar(&contractName, "name", "", "Contract name")
	contractsCreateCmd.Flags().StringVar(&contractSchema, "schema", "", "Path to JSON schema file")
	contractsCreateCmd.Flags().StringVar(&contractVersion, "version", "v1", "Contract version")
	contractsCreateCmd.Flags().StringVar(&contractEventType, "event-type", "", "Event type")
	contractsCreateCmd.Flags().StringVar(&contractStrictness, "strictness", "standard", "Validation strictness (loose, standard, strict)")

	contractsValidateCmd.Flags().StringVar(&contractID, "contract", "", "Contract ID")
	contractsValidateCmd.Flags().StringVar(&contractPayload, "payload", "", "Path to payload JSON file or '-' for stdin")

	contractsDiffCmd.Flags().StringVar(&contractID, "contract", "", "Contract ID")
	contractsDiffCmd.Flags().StringVar(&contractOldVersion, "old-version", "", "Old version")
	contractsDiffCmd.Flags().StringVar(&contractNewVersion, "new-version", "", "New version")

	contractsTestCmd.Flags().StringVar(&contractID, "contract", "", "Contract ID")
	contractsTestCmd.Flags().StringVar(&contractPayload, "payload", "", "Path to payload JSON file")
	contractsTestCmd.Flags().BoolVar(&contractCI, "ci", false, "CI mode (machine-readable output)")
}

func contractsAPICall(method, path string, body interface{}) ([]byte, error) {
	client := NewClient(getAPIURL(), getAPIKey())
	resp, err := client.doRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func runContractsList(cmd *cobra.Command, args []string) error {
	body, err := contractsAPICall("GET", "/api/v1/contracts", nil)
	if err != nil {
		return fmt.Errorf("failed to list contracts: %w", err)
	}

	if output == "json" {
		fmt.Println(string(body))
		return nil
	}

	var result struct {
		Contracts []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Version   string `json:"version"`
			EventType string `json:"event_type"`
			Status    string `json:"status"`
		} `json:"contracts"`
		Total int `json:"total"`
	}
	json.Unmarshal(body, &result)

	fmt.Printf("%-36s  %-20s  %-8s  %-20s  %s\n", "ID", "NAME", "VERSION", "EVENT TYPE", "STATUS")
	fmt.Println("────────────────────────────────────  ────────────────────  ────────  ────────────────────  ──────")
	for _, c := range result.Contracts {
		fmt.Printf("%-36s  %-20s  %-8s  %-20s  %s\n", c.ID, truncateStr(c.Name, 20), c.Version, truncateStr(c.EventType, 20), c.Status)
	}
	fmt.Printf("\nTotal: %d contracts\n", result.Total)
	return nil
}

func runContractsCreate(cmd *cobra.Command, args []string) error {
	if contractName == "" {
		return fmt.Errorf("--name is required")
	}
	if contractSchema == "" {
		return fmt.Errorf("--schema is required")
	}

	schemaData, err := os.ReadFile(contractSchema)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schemaCheck interface{}
	if err := json.Unmarshal(schemaData, &schemaCheck); err != nil {
		return fmt.Errorf("schema is not valid JSON: %w", err)
	}

	payload := map[string]interface{}{
		"name":       contractName,
		"version":    contractVersion,
		"event_type": contractEventType,
		"schema":     string(schemaData),
		"strictness": contractStrictness,
	}

	body, err := contractsAPICall("POST", "/api/v1/contracts", payload)
	if err != nil {
		return fmt.Errorf("failed to create contract: %w", err)
	}

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(body, &result)

	fmt.Printf("✅ Contract created: %s (%s)\n", result.Name, result.ID)
	return nil
}

func runContractsValidate(cmd *cobra.Command, args []string) error {
	if contractID == "" {
		return fmt.Errorf("--contract is required")
	}
	if contractPayload == "" {
		return fmt.Errorf("--payload is required")
	}

	var payloadData []byte
	var err error

	if contractPayload == "-" {
		payloadData, err = io.ReadAll(os.Stdin)
	} else {
		payloadData, err = os.ReadFile(contractPayload)
	}
	if err != nil {
		return fmt.Errorf("failed to read payload: %w", err)
	}

	req := map[string]interface{}{
		"contract_id": contractID,
		"payload":     string(payloadData),
	}

	body, err := contractsAPICall("POST", "/api/v1/contracts/validate", req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}

	var result struct {
		Passed     bool `json:"passed"`
		Violations []struct {
			Path     string `json:"path"`
			Message  string `json:"message"`
			Severity string `json:"severity"`
		} `json:"violations"`
	}
	json.Unmarshal(body, &result)

	if result.Passed {
		fmt.Println("✅ Payload matches contract schema")
		return nil
	}

	fmt.Println("❌ Validation failed:")
	for _, v := range result.Violations {
		icon := "⚠️"
		if v.Severity == "error" {
			icon = "❌"
		}
		fmt.Printf("  %s [%s] %s: %s\n", icon, v.Severity, v.Path, v.Message)
	}
	os.Exit(1)
	return nil
}

func runContractsDiff(cmd *cobra.Command, args []string) error {
	if contractID == "" {
		return fmt.Errorf("--contract is required")
	}

	path := fmt.Sprintf("/api/v1/contracts/%s/diff?old_version=%s&new_version=%s",
		contractID, contractOldVersion, contractNewVersion)

	body, err := contractsAPICall("GET", path, nil)
	if err != nil {
		return fmt.Errorf("diff request failed: %w", err)
	}

	var result struct {
		IsBreaking bool `json:"is_breaking"`
		Changes    []struct {
			Type        string `json:"type"`
			Path        string `json:"path"`
			Description string `json:"description"`
			IsBreaking  bool   `json:"is_breaking"`
		} `json:"changes"`
	}
	json.Unmarshal(body, &result)

	if len(result.Changes) == 0 {
		fmt.Println("✅ No changes detected between versions")
		return nil
	}

	if result.IsBreaking {
		fmt.Println("🚨 Breaking changes detected:")
	} else {
		fmt.Println("📝 Non-breaking changes detected:")
	}

	for _, c := range result.Changes {
		icon := "  ➕"
		if c.Type == "removed" {
			icon = "  ➖"
		} else if c.Type == "changed" {
			icon = "  🔄"
		}
		breaking := ""
		if c.IsBreaking {
			breaking = " [BREAKING]"
		}
		fmt.Printf("%s %s: %s%s\n", icon, c.Path, c.Description, breaking)
	}

	if result.IsBreaking {
		os.Exit(2)
	}
	return nil
}

func runContractsTest(cmd *cobra.Command, args []string) error {
	if contractID == "" {
		return fmt.Errorf("--contract is required")
	}

	contractBody, err := contractsAPICall("GET", fmt.Sprintf("/api/v1/contracts/%s", contractID), nil)
	if err != nil {
		return fmt.Errorf("failed to get contract: %w", err)
	}

	var contract struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	json.Unmarshal(contractBody, &contract)

	if contractCI {
		fmt.Printf("::group::Contract Test: %s (%s)\n", contract.Name, contract.Version)
	} else {
		fmt.Printf("Testing contract: %s (%s)\n", contract.Name, contract.Version)
	}

	if contractPayload != "" {
		payloadData, err := os.ReadFile(contractPayload)
		if err != nil {
			return fmt.Errorf("failed to read payload: %w", err)
		}

		req := map[string]interface{}{
			"contract_id": contractID,
			"payload":     string(payloadData),
		}

		body, err := contractsAPICall("POST", "/api/v1/contracts/validate", req)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		var result struct {
			Passed     bool `json:"passed"`
			Violations []struct {
				Path     string `json:"path"`
				Message  string `json:"message"`
				Severity string `json:"severity"`
			} `json:"violations"`
		}
		json.Unmarshal(body, &result)

		if result.Passed {
			if contractCI {
				fmt.Println("::notice::✅ Contract validation passed")
			} else {
				fmt.Println("  ✅ Payload validation: PASSED")
			}
		} else {
			if contractCI {
				for _, v := range result.Violations {
					fmt.Printf("::error file=%s::%s: %s\n", contractPayload, v.Path, v.Message)
				}
			} else {
				fmt.Println("  ❌ Payload validation: FAILED")
				for _, v := range result.Violations {
					fmt.Printf("    - [%s] %s: %s\n", v.Severity, v.Path, v.Message)
				}
			}
			if contractCI {
				fmt.Println("::endgroup::")
			}
			os.Exit(1)
		}
	}

	if contractCI {
		fmt.Println("::endgroup::")
	}

	return nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

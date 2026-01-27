package cmd

import (
	"fmt"
	"os"

	out "webhook-platform/cmd/waas-cli/output"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate webhook config from an OpenAPI spec",
	Long: `Generate webhook configuration (event types, endpoints, schemas) from an OpenAPI YAML/JSON file.

Examples:
  waas generate --from openapi.yaml
  waas generate --from openapi.json --sdk --language go
  waas generate --from openapi.yaml --tests`,
	RunE: runGenerate,
}

var (
	generateFrom     string
	generateSDK      bool
	generateTests    bool
	generateLanguage string
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVar(&generateFrom, "from", "", "Path to OpenAPI YAML/JSON file (required)")
	generateCmd.Flags().BoolVar(&generateSDK, "sdk", false, "Also generate SDK types")
	generateCmd.Flags().BoolVar(&generateTests, "tests", false, "Also generate contract tests")
	generateCmd.Flags().StringVar(&generateLanguage, "language", "go", "SDK target language (used with --sdk)")
	generateCmd.MarkFlagRequired("from")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(generateFrom); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", generateFrom)
	}

	specContent, err := os.ReadFile(generateFrom)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	client := NewClient(getAPIURL(), getAPIKey())

	result, err := client.GenerateFromOpenAPI(string(specContent))
	if err != nil {
		return fmt.Errorf("failed to generate from OpenAPI spec: %w", err)
	}

	if output == "json" {
		return jsonOutput(result)
	}

	out.PrintSuccess("Generated webhook config from OpenAPI spec")
	if v, ok := result["event_types"]; ok {
		out.PrintKeyValue("Event Types", fmt.Sprintf("%v", v))
	}
	if v, ok := result["endpoints"]; ok {
		out.PrintKeyValue("Endpoints", fmt.Sprintf("%v", v))
	}
	if v, ok := result["schemas"]; ok {
		out.PrintKeyValue("Schemas", fmt.Sprintf("%v", v))
	}

	if generateSDK {
		sdkResult, err := client.GenerateSDK(string(specContent), generateLanguage)
		if err != nil {
			return fmt.Errorf("failed to generate SDK: %w", err)
		}

		if output == "json" {
			return jsonOutput(sdkResult)
		}

		out.PrintSuccess(fmt.Sprintf("Generated SDK types (%s)", generateLanguage))
		if v, ok := sdkResult["files"]; ok {
			out.PrintKeyValue("Files", fmt.Sprintf("%v", v))
		}
		if v, ok := sdkResult["types"]; ok {
			out.PrintKeyValue("Types", fmt.Sprintf("%v", v))
		}
	}

	if generateTests {
		testsResult, err := client.GenerateTests(string(specContent))
		if err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}

		if output == "json" {
			return jsonOutput(testsResult)
		}

		out.PrintSuccess("Generated contract tests")
		if v, ok := testsResult["tests"]; ok {
			out.PrintKeyValue("Tests", fmt.Sprintf("%v", v))
		}
		if v, ok := testsResult["files"]; ok {
			out.PrintKeyValue("Files", fmt.Sprintf("%v", v))
		}
	}

	return nil
}

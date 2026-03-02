package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initQuickstart bool
	initTemplate   string
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new WAAS project",
	Long: `Initialize a new WAAS project with configuration files and example payloads.

Creates a .waas/ directory with project configuration, example webhook payloads,
and a Makefile for common operations.

Use --quickstart for an interactive wizard that guides you through setup
and achieves time-to-first-webhook in under 5 minutes.

Templates:
  basic      - Minimal configuration (default)
  ecommerce  - E-commerce webhook setup (order events)
  saas       - SaaS platform (user/subscription events)
  payments   - Payment processing (Stripe/PayPal-style events)

Examples:
  waas init                        # Initialize in current directory
  waas init my-project             # Initialize in a new directory
  waas init --quickstart           # Interactive guided setup
  waas init --template ecommerce   # Use e-commerce template`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initQuickstart, "quickstart", false, "Run interactive onboarding wizard")
	initCmd.Flags().StringVar(&initTemplate, "template", "basic", "Project template: basic, ecommerce, saas, payments")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	if initQuickstart {
		return runOnboardingWizard(dir)
	}

	return runStandardInit(dir, initTemplate)
}

func runStandardInit(dir, template string) error {
	waasDir := filepath.Join(dir, ".waas")
	if err := os.MkdirAll(waasDir, 0755); err != nil {
		return fmt.Errorf("failed to create .waas directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(waasDir, "payloads"), 0755); err != nil {
		return fmt.Errorf("failed to create payloads directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(waasDir, "pipelines"), 0755); err != nil {
		return fmt.Errorf("failed to create pipelines directory: %w", err)
	}

	// Write project config
	projectConfig := buildProjectConfig(template, "http://localhost:8080")
	configBytes, err := json.MarshalIndent(projectConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configPath := filepath.Join(waasDir, "config.json")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Write example payloads based on template
	payloads := getTemplatePayloads(template)
	for name, payload := range payloads {
		payloadBytes, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal payload %s: %w", name, err)
		}
		payloadPath := filepath.Join(waasDir, "payloads", name+".json")
		if err := os.WriteFile(payloadPath, payloadBytes, 0644); err != nil {
			return fmt.Errorf("failed to write payload %s: %w", name, err)
		}
	}

	// Write example pipeline YAML
	pipelineYAML := getTemplatePipeline(template)
	pipelinePath := filepath.Join(waasDir, "pipelines", "default.yaml")
	if err := os.WriteFile(pipelinePath, []byte(pipelineYAML), 0644); err != nil {
		return fmt.Errorf("failed to write pipeline: %w", err)
	}

	// Write .gitignore for secrets
	gitignore := "# WAAS local config\n.waas/secrets/\n.waas/*.key\n.waas/.env\n"
	gitignorePath := filepath.Join(waasDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	fmt.Printf("✓ WAAS project initialized in %s (template: %s)\n\n", dir, template)
	fmt.Println("Created:")
	fmt.Printf("  %s\n", configPath)
	for name := range payloads {
		fmt.Printf("  %s\n", filepath.Join(waasDir, "payloads", name+".json"))
	}
	fmt.Printf("  %s\n", pipelinePath)
	fmt.Printf("  %s\n", gitignorePath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. waas login                              # Authenticate")
	fmt.Println("  2. waas endpoints create --url <your-url>  # Create an endpoint")
	fmt.Println("  3. waas send --endpoint <id> --file .waas/payloads/example.json")
	fmt.Println("\n  Or run: waas init --quickstart              # Interactive wizard")

	return nil
}

func runOnboardingWizard(dir string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🚀 WAAS Onboarding Wizard")
	fmt.Println("=========================")
	fmt.Println("This wizard will help you set up your first webhook in under 5 minutes.")

	// Step 1: Project name
	fmt.Print("📋 Project name [my-webhooks]: ")
	projectName := readLine(reader, "my-webhooks")

	// Step 2: API URL
	fmt.Print("🌐 WAAS API URL [http://localhost:8080]: ")
	apiURLInput := readLine(reader, "http://localhost:8080")
	if _, err := url.ParseRequestURI(apiURLInput); err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}

	// Step 3: Template selection
	fmt.Println("\n📦 Choose a template:")
	fmt.Println("  1. basic      - Minimal setup")
	fmt.Println("  2. ecommerce  - E-commerce (order events)")
	fmt.Println("  3. saas       - SaaS platform (user events)")
	fmt.Println("  4. payments   - Payment processing")
	fmt.Print("Select [1]: ")
	templateChoice := readLine(reader, "1")
	templates := map[string]string{"1": "basic", "2": "ecommerce", "3": "saas", "4": "payments"}
	template := templates[templateChoice]
	if template == "" {
		template = "basic"
	}

	// Step 4: Endpoint URL
	fmt.Print("\n🎯 Your webhook endpoint URL (leave empty to skip): ")
	endpointURL := readLine(reader, "")

	// Step 5: Retry configuration
	fmt.Print("🔄 Max retries [5]: ")
	maxRetriesStr := readLine(reader, "5")
	maxRetries, err := strconv.Atoi(maxRetriesStr)
	if err != nil || maxRetries < 0 {
		maxRetries = 5
	}

	// Step 6: Content type
	fmt.Print("📄 Content type [application/json]: ")
	contentType := readLine(reader, "application/json")

	fmt.Println("\n⏳ Setting up your project...")

	// Create project structure
	waasDir := filepath.Join(dir, ".waas")
	for _, subdir := range []string{"payloads", "pipelines", "secrets"} {
		if err := os.MkdirAll(filepath.Join(waasDir, subdir), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Build and write config
	projectConfig := map[string]interface{}{
		"version":      "1",
		"project_name": projectName,
		"api_url":      apiURLInput,
		"template":     template,
		"defaults": map[string]interface{}{
			"max_retries":  maxRetries,
			"retry_delay":  "1s",
			"timeout":      "30s",
			"content_type": contentType,
		},
		"endpoints": []interface{}{},
	}

	if endpointURL != "" {
		projectConfig["endpoints"] = []interface{}{
			map[string]interface{}{
				"url":         endpointURL,
				"description": "Primary webhook endpoint",
			},
		}
	}

	configBytes, err := json.MarshalIndent(projectConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configPath := filepath.Join(waasDir, "config.json")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Write template payloads
	payloads := getTemplatePayloads(template)
	for name, payload := range payloads {
		payloadBytes, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal payload %s: %w", name, err)
		}
		payloadPath := filepath.Join(waasDir, "payloads", name+".json")
		if err := os.WriteFile(payloadPath, payloadBytes, 0644); err != nil {
			return fmt.Errorf("failed to write payload: %w", err)
		}
	}

	// Write pipeline
	pipelineYAML := getTemplatePipeline(template)
	pipelinePath := filepath.Join(waasDir, "pipelines", "default.yaml")
	if err := os.WriteFile(pipelinePath, []byte(pipelineYAML), 0644); err != nil {
		return fmt.Errorf("failed to write pipeline: %w", err)
	}

	// Write .gitignore
	gitignore := "# WAAS local config\n.waas/secrets/\n.waas/*.key\n.waas/.env\n"
	if err := os.WriteFile(filepath.Join(waasDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	// Summary
	fmt.Println("\n✅ Project setup complete!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Project:    %s\n", projectName)
	fmt.Printf("  Template:   %s\n", template)
	fmt.Printf("  API URL:    %s\n", apiURLInput)
	fmt.Printf("  Directory:  %s\n", waasDir)

	fmt.Println("\n📌 Quick Start Commands:")
	fmt.Println("  ┌────────────────────────────────────────────────────────┐")
	fmt.Println("  │ waas login                         # Authenticate     │")
	if endpointURL != "" {
		fmt.Printf("  │ waas endpoints create --url %-25s│\n", endpointURL)
	} else {
		fmt.Println("  │ waas endpoints create --url <url>  # Add endpoint     │")
	}
	fmt.Println("  │ waas send --endpoint <id> \\                           │")
	fmt.Println("  │   --file .waas/payloads/example.json                  │")
	fmt.Println("  │ waas logs --tail                    # Watch deliveries │")
	fmt.Println("  └────────────────────────────────────────────────────────┘")

	return nil
}

func readLine(reader *bufio.Reader, defaultVal string) string {
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func buildProjectConfig(template, apiURL string) map[string]interface{} {
	config := map[string]interface{}{
		"version":  "1",
		"template": template,
		"api_url":  apiURL,
		"defaults": map[string]interface{}{
			"max_retries":  5,
			"retry_delay":  "1s",
			"timeout":      "30s",
			"content_type": "application/json",
		},
		"endpoints": []interface{}{},
	}
	return config
}

func getTemplatePayloads(template string) map[string]interface{} {
	switch template {
	case "ecommerce":
		return map[string]interface{}{
			"example": map[string]interface{}{
				"event": "order.created",
				"data": map[string]interface{}{
					"order_id": "ord_123",
					"amount":   99.99,
					"currency": "USD",
					"items":    []interface{}{map[string]interface{}{"sku": "ITEM-001", "qty": 2}},
				},
				"timestamp": "2026-01-01T00:00:00Z",
			},
			"order_shipped": map[string]interface{}{
				"event": "order.shipped",
				"data": map[string]interface{}{
					"order_id":   "ord_123",
					"carrier":    "fedex",
					"tracking":   "1Z999AA10123456784",
					"shipped_at": "2026-01-02T10:00:00Z",
				},
				"timestamp": "2026-01-02T10:00:00Z",
			},
		}
	case "saas":
		return map[string]interface{}{
			"example": map[string]interface{}{
				"event": "user.created",
				"data": map[string]interface{}{
					"user_id": "usr_abc123",
					"email":   "user@example.com",
					"plan":    "pro",
				},
				"timestamp": "2026-01-01T00:00:00Z",
			},
			"subscription_changed": map[string]interface{}{
				"event": "subscription.updated",
				"data": map[string]interface{}{
					"user_id":      "usr_abc123",
					"old_plan":     "free",
					"new_plan":     "pro",
					"effective_at": "2026-02-01T00:00:00Z",
				},
				"timestamp": "2026-01-15T00:00:00Z",
			},
		}
	case "payments":
		return map[string]interface{}{
			"example": map[string]interface{}{
				"event": "payment.completed",
				"data": map[string]interface{}{
					"payment_id": "pay_xyz789",
					"amount":     49.99,
					"currency":   "USD",
					"method":     "card",
					"status":     "succeeded",
				},
				"timestamp": "2026-01-01T00:00:00Z",
			},
			"payment_failed": map[string]interface{}{
				"event": "payment.failed",
				"data": map[string]interface{}{
					"payment_id": "pay_xyz790",
					"amount":     49.99,
					"currency":   "USD",
					"error_code": "card_declined",
				},
				"timestamp": "2026-01-01T01:00:00Z",
			},
		}
	default:
		return map[string]interface{}{
			"example": map[string]interface{}{
				"event": "order.created",
				"data": map[string]interface{}{
					"order_id": "ord_123",
					"amount":   99.99,
					"currency": "USD",
				},
				"timestamp": "2026-01-01T00:00:00Z",
			},
		}
	}
}

func getTemplatePipeline(template string) string {
	switch template {
	case "ecommerce":
		return `name: ecommerce-pipeline
description: Process and route e-commerce webhook events
version: "1"
stages:
  - id: validate-order
    type: validate
    config:
      schema:
        type: object
        required: [event, data]
      strictness: standard
  - id: enrich
    type: enrich
    config:
      script: "payload.processed_at = new Date().toISOString()"
  - id: route-events
    type: route
    config:
      rules:
        - condition: "payload.event.startsWith('order.')"
          endpoint_ids: [orders-service]
          label: order-events
        - condition: "true"
          endpoint_ids: [default-handler]
          label: catch-all
`
	case "saas":
		return `name: saas-pipeline
description: Process SaaS platform events
version: "1"
stages:
  - id: validate
    type: validate
    config:
      schema:
        type: object
        required: [event, data]
      strictness: standard
  - id: filter-internal
    type: filter
    config:
      condition: "!payload.event.startsWith('internal.')"
      on_reject: log
  - id: deliver
    type: deliver
    config:
      timeout_seconds: 30
      retry_policy:
        max_attempts: 5
        initial_interval_seconds: 1
        backoff_factor: 2
`
	case "payments":
		return `name: payments-pipeline
description: Process payment events with validation
version: "1"
stages:
  - id: validate-payment
    type: validate
    config:
      schema:
        type: object
        required: [event, data]
        properties:
          data:
            type: object
            required: [payment_id, amount]
      strictness: strict
  - id: log-payment
    type: log
    config:
      message: "Processing payment event"
      level: info
  - id: deliver
    type: deliver
    config:
      timeout_seconds: 15
      retry_policy:
        max_attempts: 10
        initial_interval_seconds: 1
        backoff_factor: 2
`
	default:
		return `name: default-pipeline
description: Basic webhook delivery pipeline
version: "1"
stages:
  - id: validate
    type: validate
    config:
      schema:
        type: object
      strictness: loose
  - id: deliver
    type: deliver
    config:
      timeout_seconds: 30
      retry_policy:
        max_attempts: 5
        initial_interval_seconds: 1
        backoff_factor: 2
`
	}
}

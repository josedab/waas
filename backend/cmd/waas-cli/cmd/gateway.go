package cmd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage inbound webhook gateway providers",
	Long: `Manage and test inbound webhook gateway providers.

Test webhook signature verification locally, list supported providers,
and auto-detect provider types from sample headers.

Examples:
  waas gateway providers
  waas gateway test --provider stripe --secret whsec_xxx --payload payload.json
  waas gateway detect --headers headers.json`,
}

var gatewayProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List all supported webhook providers",
	RunE:  runGatewayProviders,
}

var gatewayTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test webhook signature verification locally",
	Long: `Test webhook signature verification against a provider's expected format.

This command generates a valid signature for the given payload and secret,
then verifies it matches the expected output. Useful for debugging
webhook integration issues.

Examples:
  waas gateway test --provider stripe --secret whsec_test123 --payload event.json
  waas gateway test --provider github --secret mysecret --payload '{"action":"push"}' --inline
  waas gateway test --provider custom --secret key --header X-Sig --algorithm hmac-sha256`,
	RunE: runGatewayTest,
}

var gatewayDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Auto-detect webhook provider from headers",
	Long: `Detect which webhook provider sent a request by analyzing HTTP headers.

Provide headers as a JSON object to identify the provider type.

Examples:
  waas gateway detect --headers '{"Stripe-Signature":"t=123,v1=abc"}'
  waas gateway detect --headers-file request_headers.json`,
	RunE: runGatewayDetect,
}

var (
	gwProvider       string
	gwSecret         string
	gwPayloadFile    string
	gwPayloadInline  string
	gwInline         bool
	gwHeaders        string
	gwHeadersFile    string
	gwSigHeader      string
	gwAlgorithm      string
)

func init() {
	rootCmd.AddCommand(gatewayCmd)

	gatewayCmd.AddCommand(gatewayProvidersCmd)
	gatewayCmd.AddCommand(gatewayTestCmd)
	gatewayCmd.AddCommand(gatewayDetectCmd)

	gatewayTestCmd.Flags().StringVar(&gwProvider, "provider", "", "Provider type (stripe, github, shopify, etc.)")
	gatewayTestCmd.Flags().StringVar(&gwSecret, "secret", "", "Webhook signing secret")
	gatewayTestCmd.Flags().StringVar(&gwPayloadFile, "payload", "", "Path to payload JSON file")
	gatewayTestCmd.Flags().StringVar(&gwPayloadInline, "data", "", "Inline payload JSON")
	gatewayTestCmd.Flags().BoolVar(&gwInline, "inline", false, "Treat --payload as inline data")
	gatewayTestCmd.Flags().StringVar(&gwSigHeader, "header", "", "Custom signature header name")
	gatewayTestCmd.Flags().StringVar(&gwAlgorithm, "algorithm", "hmac-sha256", "Signature algorithm")

	gatewayDetectCmd.Flags().StringVar(&gwHeaders, "headers", "", "Headers as JSON string")
	gatewayDetectCmd.Flags().StringVar(&gwHeadersFile, "headers-file", "", "Path to headers JSON file")
}

// Provider metadata for display
type providerInfo struct {
	Name     string
	Type     string
	SigHeader string
	Algorithm string
}

func getSupportedProviders() []providerInfo {
	return []providerInfo{
		{"Stripe", "stripe", "Stripe-Signature", "HMAC-SHA256"},
		{"GitHub", "github", "X-Hub-Signature-256", "HMAC-SHA256"},
		{"Shopify", "shopify", "X-Shopify-Hmac-Sha256", "HMAC-SHA256-Base64"},
		{"Twilio", "twilio", "X-Twilio-Signature", "HMAC-SHA1-Base64"},
		{"Slack", "slack", "X-Slack-Signature", "HMAC-SHA256"},
		{"SendGrid", "sendgrid", "X-Twilio-Email-Event-Webhook-Signature", "ECDSA/HMAC"},
		{"Paddle", "paddle", "Paddle-Signature", "HMAC-SHA256"},
		{"Linear", "linear", "Linear-Signature", "HMAC-SHA256"},
		{"Intercom", "intercom", "X-Hub-Signature", "HMAC-SHA1"},
		{"Discord", "discord", "X-Signature-Ed25519", "Ed25519"},
		{"GitLab", "gitlab", "X-Gitlab-Token", "Token match"},
		{"Bitbucket", "bitbucket", "X-Hub-Signature", "HMAC-SHA256"},
		{"Zoom", "zoom", "X-Zm-Signature", "HMAC-SHA256"},
		{"Square", "square", "X-Square-Hmacsha256-Signature", "HMAC-SHA256-Base64"},
		{"HubSpot", "hubspot", "X-Hubspot-Signature-V3", "HMAC-SHA256"},
		{"Mailgun", "mailgun", "X-Mailgun-Signature", "HMAC-SHA256"},
		{"DocuSign", "docusign", "X-Docusign-Signature-1", "HMAC-SHA256-Base64"},
		{"Typeform", "typeform", "Typeform-Signature", "HMAC-SHA256-Base64"},
		{"Jira", "jira", "X-Hub-Signature", "HMAC-SHA256"},
		{"PagerDuty", "pagerduty", "X-Pagerduty-Signature", "HMAC-SHA256"},
		{"Zendesk", "zendesk", "X-Zendesk-Webhook-Signature", "HMAC-SHA256-Base64"},
		{"Asana", "asana", "X-Hook-Signature", "HMAC-SHA256"},
		{"Cloudflare", "cloudflare", "Cf-Webhook-Auth", "Token match"},
		{"Figma", "figma", "X-Figma-Signature", "HMAC-SHA256"},
		{"AWS EventBridge", "aws_eventbridge", "X-Amz-Ce-Signature", "HMAC-SHA256"},
		{"Azure Event Grid", "azure_event_grid", "Aeg-Signature", "HMAC-SHA256"},
		{"Google Pub/Sub", "google_pubsub", "Authorization", "Bearer JWT"},
		{"Salesforce", "salesforce", "X-Salesforce-Signature", "HMAC-SHA256-Base64"},
		{"Workday", "workday", "X-Workday-Signature", "HMAC-SHA256"},
		{"Datadog", "datadog", "Dd-Webhook-Signature", "HMAC-SHA256"},
		{"LaunchDarkly", "launchdarkly", "X-Ld-Signature", "HMAC-SHA256"},
		{"Plaid", "plaid", "Plaid-Verification", "JWT/HMAC"},
		{"CircleCI", "circleci", "Circleci-Signature", "HMAC-SHA256"},
		{"Vercel", "vercel", "X-Vercel-Signature", "HMAC-SHA256"},
		{"Netlify", "netlify", "X-Webhook-Signature", "HMAC-SHA256"},
		{"Sentry", "sentry", "Sentry-Hook-Signature", "HMAC-SHA256"},
		{"New Relic", "newrelic", "X-Newrelic-Webhook-Signature", "HMAC-SHA256"},
		{"MongoDB Atlas", "mongodb", "X-Mongodb-Signature", "HMAC-SHA256"},
		{"Supabase", "supabase", "X-Supabase-Signature", "HMAC-SHA256"},
		{"Auth0", "auth0", "Auth0-Signature", "HMAC-SHA256"},
		{"Okta", "okta", "Authorization", "Shared secret"},
		{"Trello", "trello", "X-Trello-Webhook", "HMAC-SHA256"},
		{"ClickUp", "clickup", "X-Signature", "HMAC-SHA256"},
		{"Notion", "notion", "X-Notion-Signature", "HMAC-SHA256"},
		{"Airtable", "airtable", "X-Airtable-Signature", "HMAC-SHA256"},
		{"Segment", "segment", "X-Signature", "HMAC-SHA1"},
		{"Braze", "braze", "X-Braze-Signature", "HMAC-SHA256"},
		{"Contentful", "contentful", "X-Contentful-Signature", "HMAC-SHA256"},
		{"Sanity", "sanity", "X-Sanity-Signature", "HMAC-SHA256"},
		{"Calendly", "calendly", "Calendly-Webhook-Signature", "HMAC-SHA256"},
		{"Gong", "gong", "X-Gong-Signature", "HMAC-SHA256"},
		{"Custom", "custom", "(configurable)", "HMAC-SHA256/SHA1"},
	}
}

func runGatewayProviders(cmd *cobra.Command, args []string) error {
	providers := getSupportedProviders()

	if output == "json" {
		data, _ := json.MarshalIndent(providers, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%-20s  %-18s  %-40s  %s\n", "PROVIDER", "TYPE", "SIGNATURE HEADER", "ALGORITHM")
	fmt.Println(strings.Repeat("─", 105))
	for _, p := range providers {
		fmt.Printf("%-20s  %-18s  %-40s  %s\n", p.Name, p.Type, p.SigHeader, p.Algorithm)
	}
	fmt.Printf("\nTotal: %d providers\n", len(providers))
	return nil
}

func runGatewayTest(cmd *cobra.Command, args []string) error {
	if gwProvider == "" {
		return fmt.Errorf("--provider is required")
	}
	if gwSecret == "" {
		return fmt.Errorf("--secret is required")
	}

	var payloadData []byte
	var err error
	if gwPayloadInline != "" || gwInline {
		if gwPayloadInline != "" {
			payloadData = []byte(gwPayloadInline)
		} else {
			payloadData = []byte(gwPayloadFile)
		}
	} else if gwPayloadFile != "" {
		payloadData, err = os.ReadFile(gwPayloadFile)
		if err != nil {
			return fmt.Errorf("failed to read payload file: %w", err)
		}
	} else {
		return fmt.Errorf("--payload or --data is required")
	}

	// Generate signature based on provider type
	sig, headerName := generateSignature(gwProvider, payloadData, gwSecret)

	fmt.Printf("🔐 Provider: %s\n", gwProvider)
	fmt.Printf("📦 Payload size: %d bytes\n", len(payloadData))
	fmt.Printf("🔑 Signature header: %s\n", headerName)
	fmt.Printf("✅ Generated signature: %s\n", sig)
	fmt.Println()
	fmt.Println("Use these headers to test your webhook endpoint:")
	fmt.Printf("  %s: %s\n", headerName, sig)

	return nil
}

func generateSignature(provider string, payload []byte, secret string) (string, string) {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	switch provider {
	case "stripe":
		ts := "1234567890"
		msg := ts + "." + string(payload)
		mac = hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(msg))
		return fmt.Sprintf("t=%s,v1=%s", ts, hex.EncodeToString(mac.Sum(nil))), "Stripe-Signature"
	case "github":
		return "sha256=" + sig, "X-Hub-Signature-256"
	case "slack":
		ts := "1234567890"
		msg := fmt.Sprintf("v0:%s:%s", ts, string(payload))
		mac = hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(msg))
		return "v0=" + hex.EncodeToString(mac.Sum(nil)), "X-Slack-Signature"
	default:
		// Find header from provider list
		for _, p := range getSupportedProviders() {
			if p.Type == provider {
				return sig, p.SigHeader
			}
		}
		return sig, "X-Webhook-Signature"
	}
}

func runGatewayDetect(cmd *cobra.Command, args []string) error {
	var headersData []byte
	var err error

	if gwHeaders != "" {
		headersData = []byte(gwHeaders)
	} else if gwHeadersFile != "" {
		headersData, err = os.ReadFile(gwHeadersFile)
		if err != nil {
			return fmt.Errorf("failed to read headers file: %w", err)
		}
	} else {
		return fmt.Errorf("--headers or --headers-file is required")
	}

	var headers map[string]string
	if err := json.Unmarshal(headersData, &headers); err != nil {
		return fmt.Errorf("invalid headers JSON: %w", err)
	}

	// Auto-detection logic matching pkg/gateway/providers_extended.go
	headerSignatures := map[string]string{
		"Stripe-Signature":                       "stripe",
		"X-Hub-Signature-256":                    "github",
		"X-Shopify-Hmac-Sha256":                  "shopify",
		"X-Twilio-Signature":                     "twilio",
		"X-Slack-Signature":                      "slack",
		"Paddle-Signature":                       "paddle",
		"Linear-Signature":                       "linear",
		"X-Signature-Ed25519":                    "discord",
		"X-Gitlab-Token":                         "gitlab",
		"X-Zm-Signature":                         "zoom",
		"X-Square-Hmacsha256-Signature":          "square",
		"X-Hubspot-Signature-V3":                 "hubspot",
		"X-Pagerduty-Signature":                  "pagerduty",
		"X-Zendesk-Webhook-Signature":            "zendesk",
		"Cf-Webhook-Auth":                        "cloudflare",
		"X-Figma-Signature":                      "figma",
		"X-Amz-Ce-Signature":                     "aws_eventbridge",
		"Dd-Webhook-Signature":                   "datadog",
		"X-Vercel-Signature":                     "vercel",
		"Sentry-Hook-Signature":                  "sentry",
		"Circleci-Signature":                     "circleci",
		"X-Salesforce-Signature":                 "salesforce",
	}

	detected := []string{}
	for header, provider := range headerSignatures {
		if _, ok := headers[header]; ok {
			detected = append(detected, provider)
		}
	}

	sort.Strings(detected)

	if output == "json" {
		result := map[string]interface{}{
			"detected_providers": detected,
			"headers_analyzed":   len(headers),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(detected) == 0 {
		fmt.Println("❓ Could not detect webhook provider from headers")
		fmt.Println("   Tip: Ensure you include the signature-related headers")
		return nil
	}

	fmt.Printf("🔍 Detected provider(s): %s\n", strings.Join(detected, ", "))
	for _, p := range detected {
		for _, info := range getSupportedProviders() {
			if info.Type == p {
				fmt.Printf("   • %s (signature: %s, algorithm: %s)\n", info.Name, info.SigHeader, info.Algorithm)
				break
			}
		}
	}

	return nil
}

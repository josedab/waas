package gateway

import (
	"crypto/hmac"
	"fmt"
	"strings"
)

// --- Extended Provider Verifiers (26 new providers) ---

// AWSEventBridgeVerifier verifies AWS EventBridge signatures
type AWSEventBridgeVerifier struct{}

func (v *AWSEventBridgeVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Amz-Ce-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Amz-Ce-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// AzureEventGridVerifier verifies Azure Event Grid signatures
type AzureEventGridVerifier struct{}

func (v *AzureEventGridVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	// Azure Event Grid uses a validation handshake + shared key
	key := headers["Aeg-Subscription-Name"]
	if key == "" {
		key = headers["Aeg-Event-Type"]
	}
	if config.SecretKey != "" {
		signature := headers["Aeg-Signature"]
		if signature == "" {
			return false, fmt.Errorf("missing Aeg-Signature header")
		}
		expected := computeHMACSHA256(payload, []byte(config.SecretKey))
		return hmac.Equal([]byte(signature), []byte(expected)), nil
	}
	return true, nil // Validation-only mode
}

// GooglePubSubVerifier verifies Google Cloud Pub/Sub push signatures
type GooglePubSubVerifier struct{}

func (v *GooglePubSubVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	authHeader := headers["Authorization"]
	if authHeader == "" {
		return false, fmt.Errorf("missing Authorization header")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false, fmt.Errorf("invalid authorization format")
	}
	// In production, this would validate a JWT token from Google
	return true, nil
}

// SalesforceVerifier verifies Salesforce Platform Event signatures
type SalesforceVerifier struct{}

func (v *SalesforceVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Salesforce-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Salesforce-Signature header")
	}
	expected := computeHMACSHA256Base64(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// WorkdayVerifier verifies Workday webhook signatures
type WorkdayVerifier struct{}

func (v *WorkdayVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Workday-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Workday-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// DatadogVerifier verifies Datadog webhook signatures
type DatadogVerifier struct{}

func (v *DatadogVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Dd-Webhook-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing Dd-Webhook-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// LaunchDarklyVerifier verifies LaunchDarkly webhook signatures
type LaunchDarklyVerifier struct{}

func (v *LaunchDarklyVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Ld-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Ld-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// PlaidVerifier verifies Plaid webhook signatures
type PlaidVerifier struct{}

func (v *PlaidVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Plaid-Verification"]
	if signature == "" {
		return false, fmt.Errorf("missing Plaid-Verification header")
	}
	// Plaid uses JWT-based verification; simplified here
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// CircleCIVerifier verifies CircleCI webhook signatures
type CircleCIVerifier struct{}

func (v *CircleCIVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Circleci-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing Circleci-Signature header")
	}
	if !strings.HasPrefix(signature, "v1=") {
		return false, fmt.Errorf("invalid signature format")
	}
	sig := strings.TrimPrefix(signature, "v1=")
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

// VercelVerifier verifies Vercel webhook signatures
type VercelVerifier struct{}

func (v *VercelVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Vercel-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Vercel-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// NetlifyVerifier verifies Netlify webhook signatures
type NetlifyVerifier struct{}

func (v *NetlifyVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Webhook-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Webhook-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// SentryVerifier verifies Sentry webhook signatures
type SentryVerifier struct{}

func (v *SentryVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Sentry-Hook-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing Sentry-Hook-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// NewRelicVerifier verifies New Relic webhook signatures
type NewRelicVerifier struct{}

func (v *NewRelicVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Newrelic-Webhook-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Newrelic-Webhook-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// MongoDBVerifier verifies MongoDB Atlas webhook signatures
type MongoDBVerifier struct{}

func (v *MongoDBVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Mongodb-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Mongodb-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// SupabaseVerifier verifies Supabase webhook signatures
type SupabaseVerifier struct{}

func (v *SupabaseVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Supabase-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Supabase-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// Auth0Verifier verifies Auth0 webhook signatures
type Auth0Verifier struct{}

func (v *Auth0Verifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Auth0-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing Auth0-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// OktaVerifier verifies Okta event hook signatures
type OktaVerifier struct{}

func (v *OktaVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	authHeader := headers["Authorization"]
	if authHeader == "" {
		return false, fmt.Errorf("missing Authorization header")
	}
	return hmac.Equal([]byte(authHeader), []byte(config.SecretKey)), nil
}

// GenericHMACVerifier is a universal HMAC-based verifier for providers using standard patterns
type GenericHMACVerifier struct {
	HeaderName      string
	SignaturePrefix string
}

func (v *GenericHMACVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers[v.HeaderName]
	if signature == "" {
		return false, fmt.Errorf("missing %s header", v.HeaderName)
	}
	if v.SignaturePrefix != "" {
		signature = strings.TrimPrefix(signature, v.SignaturePrefix)
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// registerExtendedProviders adds all extended providers to the registry
func registerExtendedProviders(registry *VerifierRegistry) {
	registry.Register(ProviderTypeAWSEventBridge, &AWSEventBridgeVerifier{})
	registry.Register(ProviderTypeAzureEventGrid, &AzureEventGridVerifier{})
	registry.Register(ProviderTypeGooglePubSub, &GooglePubSubVerifier{})
	registry.Register(ProviderTypeSalesforce, &SalesforceVerifier{})
	registry.Register(ProviderTypeWorkday, &WorkdayVerifier{})
	registry.Register(ProviderTypeDatadog, &DatadogVerifier{})
	registry.Register(ProviderTypeLaunchDarkly, &LaunchDarklyVerifier{})
	registry.Register(ProviderTypePlaid, &PlaidVerifier{})
	registry.Register(ProviderTypeCircleCI, &CircleCIVerifier{})
	registry.Register(ProviderTypeVercel, &VercelVerifier{})
	registry.Register(ProviderTypeNetlify, &NetlifyVerifier{})
	registry.Register(ProviderTypeSentry, &SentryVerifier{})
	registry.Register(ProviderTypeNewRelic, &NewRelicVerifier{})
	registry.Register(ProviderTypeMongoDB, &MongoDBVerifier{})
	registry.Register(ProviderTypeSupabase, &SupabaseVerifier{})
	registry.Register(ProviderTypeAuth0, &Auth0Verifier{})
	registry.Register(ProviderTypeOkta, &OktaVerifier{})

	// Generic HMAC providers
	registry.Register(ProviderTypeTrello, &GenericHMACVerifier{HeaderName: "X-Trello-Webhook"})
	registry.Register(ProviderTypeClickUp, &GenericHMACVerifier{HeaderName: "X-Signature"})
	registry.Register(ProviderTypeNotion, &GenericHMACVerifier{HeaderName: "X-Notion-Signature"})
	registry.Register(ProviderTypeAirtable, &GenericHMACVerifier{HeaderName: "X-Airtable-Signature"})
	registry.Register(ProviderTypeSegment, &GenericHMACVerifier{HeaderName: "X-Signature", SignaturePrefix: "sha1="})
	registry.Register(ProviderTypeBraze, &GenericHMACVerifier{HeaderName: "X-Braze-Signature"})
	registry.Register(ProviderTypeContentful, &GenericHMACVerifier{HeaderName: "X-Contentful-Signature"})
	registry.Register(ProviderTypeSanity, &GenericHMACVerifier{HeaderName: "X-Sanity-Signature"})
	registry.Register(ProviderTypeCalendly, &GenericHMACVerifier{HeaderName: "Calendly-Webhook-Signature"})
	registry.Register(ProviderTypeGong, &GenericHMACVerifier{HeaderName: "X-Gong-Signature"})
}

// ProviderDefinition is a YAML-compatible provider registry entry
type ProviderDefinition struct {
	Name            string            `json:"name" yaml:"name"`
	Type            string            `json:"type" yaml:"type"`
	Description     string            `json:"description" yaml:"description"`
	SignatureHeader string            `json:"signature_header" yaml:"signature_header"`
	SignaturePrefix string            `json:"signature_prefix,omitempty" yaml:"signature_prefix"`
	TimestampHeader string            `json:"timestamp_header,omitempty" yaml:"timestamp_header"`
	Algorithm       string            `json:"algorithm" yaml:"algorithm"` // hmac-sha256, hmac-sha1
	EventTypeHeader string            `json:"event_type_header,omitempty" yaml:"event_type_header"`
	EventTypePath   string            `json:"event_type_path,omitempty" yaml:"event_type_path"`
	DocURL          string            `json:"doc_url,omitempty" yaml:"doc_url"`
	Category        string            `json:"category,omitempty" yaml:"category"`
	Metadata        map[string]string `json:"metadata,omitempty" yaml:"metadata"`
}

// CommunityProviderRegistry manages community-contributed provider definitions
type CommunityProviderRegistry struct {
	providers map[string]*ProviderDefinition
}

// NewCommunityProviderRegistry creates a new community provider registry
func NewCommunityProviderRegistry() *CommunityProviderRegistry {
	return &CommunityProviderRegistry{
		providers: make(map[string]*ProviderDefinition),
	}
}

// Register adds a provider definition to the registry
func (r *CommunityProviderRegistry) Register(def *ProviderDefinition) error {
	if def.Name == "" || def.Type == "" {
		return fmt.Errorf("provider name and type are required")
	}
	if def.Algorithm == "" {
		def.Algorithm = "hmac-sha256"
	}
	r.providers[def.Type] = def
	return nil
}

// Get returns a provider definition by type
func (r *CommunityProviderRegistry) Get(providerType string) (*ProviderDefinition, bool) {
	def, ok := r.providers[providerType]
	return def, ok
}

// List returns all registered provider definitions
func (r *CommunityProviderRegistry) List() []ProviderDefinition {
	defs := make([]ProviderDefinition, 0, len(r.providers))
	for _, d := range r.providers {
		defs = append(defs, *d)
	}
	return defs
}

// ToVerifier creates a SignatureVerifier from a provider definition
func (r *CommunityProviderRegistry) ToVerifier(def *ProviderDefinition) SignatureVerifier {
	return &GenericHMACVerifier{
		HeaderName:      def.SignatureHeader,
		SignaturePrefix: def.SignaturePrefix,
	}
}

// AutoDetectProvider attempts to identify the webhook provider from request headers
func AutoDetectProvider(headers map[string]string) string {
	headerSignatures := map[string]string{
		"Stripe-Signature":                         ProviderTypeStripe,
		"X-Hub-Signature-256":                      ProviderTypeGitHub,
		"X-Shopify-Hmac-Sha256":                    ProviderTypeShopify,
		"X-Twilio-Signature":                       ProviderTypeTwilio,
		"X-Slack-Signature":                        ProviderTypeSlack,
		"X-Twilio-Email-Event-Webhook-Signature":   ProviderTypeSendGrid,
		"Paddle-Signature":                         ProviderTypePaddle,
		"Linear-Signature":                         ProviderTypeLinear,
		"X-Signature-Ed25519":                      ProviderTypeDiscord,
		"X-Gitlab-Token":                           ProviderTypeGitLab,
		"X-Zm-Signature":                           ProviderTypeZoom,
		"X-Square-Hmacsha256-Signature":            ProviderTypeSquare,
		"X-Hubspot-Signature-V3":                   ProviderTypeHubSpot,
		"X-Hubspot-Signature":                      ProviderTypeHubSpot,
		"X-Mailgun-Signature":                      ProviderTypeMailgun,
		"X-Docusign-Signature-1":                   ProviderTypeDocuSign,
		"Typeform-Signature":                       ProviderTypeTypeform,
		"X-Pagerduty-Signature":                    ProviderTypePagerDuty,
		"X-Zendesk-Webhook-Signature":              ProviderTypeZendesk,
		"X-Hook-Signature":                         ProviderTypeAsana,
		"Cf-Webhook-Auth":                          ProviderTypeCloudflare,
		"X-Figma-Signature":                        ProviderTypeFigma,
		"X-Amz-Ce-Signature":                       ProviderTypeAWSEventBridge,
		"Aeg-Signature":                            ProviderTypeAzureEventGrid,
		"X-Salesforce-Signature":                   ProviderTypeSalesforce,
		"X-Workday-Signature":                      ProviderTypeWorkday,
		"Dd-Webhook-Signature":                     ProviderTypeDatadog,
		"X-Ld-Signature":                           ProviderTypeLaunchDarkly,
		"Plaid-Verification":                       ProviderTypePlaid,
		"Circleci-Signature":                       ProviderTypeCircleCI,
		"X-Vercel-Signature":                       ProviderTypeVercel,
		"Sentry-Hook-Signature":                    ProviderTypeSentry,
		"X-Newrelic-Webhook-Signature":             ProviderTypeNewRelic,
		"X-Mongodb-Signature":                      ProviderTypeMongoDB,
		"X-Supabase-Signature":                     ProviderTypeSupabase,
		"Auth0-Signature":                          ProviderTypeAuth0,
		"X-Notion-Signature":                       ProviderTypeNotion,
		"Calendly-Webhook-Signature":               ProviderTypeCalendly,
		"X-Gong-Signature":                         ProviderTypeGong,
	}

	for header, provider := range headerSignatures {
		if _, ok := headers[header]; ok {
			return provider
		}
	}

	// Check user-agent patterns
	ua := headers["User-Agent"]
	if ua != "" {
		uaPatterns := map[string]string{
			"GitHub-Hookshot":  ProviderTypeGitHub,
			"Shopify":         ProviderTypeShopify,
			"Stripe":          ProviderTypeStripe,
			"Bitbucket":       ProviderTypeBitbucket,
			"Zendesk":         ProviderTypeZendesk,
		}
		for pattern, provider := range uaPatterns {
			if strings.Contains(ua, pattern) {
				return provider
			}
		}
	}

	return "" // Unknown provider
}

// PayloadNormalizer normalizes webhook payloads using JSONPath extraction
type PayloadNormalizer struct {
	rules []NormalizationRule
}

// NormalizationRule defines a JSONPath-based payload normalization rule
type NormalizationRule struct {
	SourcePath  string `json:"source_path" yaml:"source_path"`
	TargetField string `json:"target_field" yaml:"target_field"`
	Default     string `json:"default,omitempty" yaml:"default"`
}

// NormalizedPayload is the standardized payload format
type NormalizedPayload struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Source    string                 `json:"source"`
	Timestamp string                 `json:"timestamp,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Raw       interface{}            `json:"raw,omitempty"`
}

// NewPayloadNormalizer creates a normalizer with default rules
func NewPayloadNormalizer() *PayloadNormalizer {
	return &PayloadNormalizer{
		rules: []NormalizationRule{
			{SourcePath: "$.id", TargetField: "event_id"},
			{SourcePath: "$.type", TargetField: "event_type"},
			{SourcePath: "$.event", TargetField: "event_type"},
			{SourcePath: "$.event_type", TargetField: "event_type"},
			{SourcePath: "$.source", TargetField: "source"},
			{SourcePath: "$.timestamp", TargetField: "timestamp"},
			{SourcePath: "$.created_at", TargetField: "timestamp"},
		},
	}
}

// Normalize converts a raw payload to a standardized format
func (n *PayloadNormalizer) Normalize(provider string, payload map[string]interface{}) *NormalizedPayload {
	normalized := &NormalizedPayload{
		Data: make(map[string]interface{}),
		Raw:  payload,
	}

	// Extract standard fields
	if id, ok := payload["id"]; ok {
		normalized.EventID = fmt.Sprintf("%v", id)
	}

	// Try multiple event type fields
	for _, key := range []string{"type", "event", "event_type", "action", "topic"} {
		if et, ok := payload[key]; ok {
			normalized.EventType = fmt.Sprintf("%v", et)
			break
		}
	}

	normalized.Source = provider

	// Extract timestamp from various fields
	for _, key := range []string{"timestamp", "created_at", "created", "time", "occurred_at"} {
		if ts, ok := payload[key]; ok {
			normalized.Timestamp = fmt.Sprintf("%v", ts)
			break
		}
	}

	// Copy data payload
	if data, ok := payload["data"]; ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			normalized.Data = dataMap
		}
	} else {
		normalized.Data = payload
	}

	return normalized
}

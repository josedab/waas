package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// --- GraphQL Deliverer ---

// GraphQLDeliverer implements webhook delivery via GraphQL mutations/subscriptions
type GraphQLDeliverer struct {
	client *http.Client
}

// NewGraphQLDeliverer creates a new GraphQL deliverer
func NewGraphQLDeliverer() *GraphQLDeliverer {
	return &GraphQLDeliverer{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *GraphQLDeliverer) Protocol() Protocol {
	return ProtocolGraphQL
}

func (d *GraphQLDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("GraphQL endpoint URL is required")
	}
	return nil
}

func (d *GraphQLDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()

	opts := parseGraphQLOptions(config.Options)

	// Build GraphQL request body
	query := opts.Query
	if query == "" {
		// Default: mutation that accepts the webhook payload as a variable
		query = `mutation WebhookDelivery($payload: JSON!) { webhookReceived(payload: $payload) { success } }`
	}

	variables := opts.Variables
	if variables == nil {
		variables = make(map[string]interface{})
	}

	// Inject the webhook payload as a variable
	var payloadData interface{}
	if err := json.Unmarshal(request.Payload, &payloadData); err == nil {
		variables["payload"] = payloadData
	} else {
		variables["payload"] = string(request.Payload)
	}

	gqlBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	if opts.OperationName != "" {
		gqlBody["operationName"] = opts.OperationName
	}

	bodyBytes, err := json.Marshal(gqlBody)
	if err != nil {
		return &DeliveryResponse{
			Success:      false,
			Duration:     time.Since(start),
			Error:        fmt.Sprintf("failed to marshal GraphQL request: %v", err),
			ErrorType:    ErrorTypeProtocol,
			ProtocolInfo: map[string]any{"protocol": "graphql"},
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.Target, bytes.NewReader(bodyBytes))
	if err != nil {
		return &DeliveryResponse{
			Success:      false,
			Duration:     time.Since(start),
			Error:        fmt.Sprintf("failed to create request: %v", err),
			ErrorType:    ErrorTypeConnection,
			ProtocolInfo: map[string]any{"protocol": "graphql"},
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Apply custom headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Apply auth
	applyAuth(req, config.Auth)

	resp, err := d.client.Do(req)
	if err != nil {
		return &DeliveryResponse{
			Success:      false,
			Duration:     time.Since(start),
			Error:        fmt.Sprintf("GraphQL request failed: %v", err),
			ErrorType:    ErrorTypeConnection,
			ProtocolInfo: map[string]any{"protocol": "graphql"},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	// Check for GraphQL errors in the response
	var gqlResp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	hasGQLErrors := false
	if err := json.Unmarshal(body, &gqlResp); err == nil && len(gqlResp.Errors) > 0 {
		hasGQLErrors = true
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300 && !hasGQLErrors

	response := &DeliveryResponse{
		Success:    success,
		StatusCode: resp.StatusCode,
		Body:       body,
		Duration:   time.Since(start),
		ProtocolInfo: map[string]any{
			"protocol":   "graphql",
			"has_errors": hasGQLErrors,
			"query_type": "mutation",
		},
	}

	if !success {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if hasGQLErrors && len(gqlResp.Errors) > 0 {
			errMsg = gqlResp.Errors[0].Message
		}
		response.Error = errMsg
		response.ErrorType = ErrorTypeServer
	}

	return response, nil
}

func (d *GraphQLDeliverer) Close() error {
	return nil
}

func parseGraphQLOptions(options map[string]interface{}) *GraphQLOptions {
	opts := &GraphQLOptions{}
	if options == nil {
		return opts
	}
	if q, ok := options["query"].(string); ok {
		opts.Query = q
	}
	if v, ok := options["variables"].(map[string]interface{}); ok {
		opts.Variables = v
	}
	if o, ok := options["operation_name"].(string); ok {
		opts.OperationName = o
	}
	if s, ok := options["use_sse"].(bool); ok {
		opts.UseSSE = s
	}
	if m, ok := options["use_mutation"].(bool); ok {
		opts.UseMutation = m
	}
	return opts
}

// --- SMTP Deliverer ---

// SMTPDeliverer implements webhook delivery via email
type SMTPDeliverer struct{}

// NewSMTPDeliverer creates a new SMTP deliverer
func NewSMTPDeliverer() *SMTPDeliverer {
	return &SMTPDeliverer{}
}

func (d *SMTPDeliverer) Protocol() Protocol {
	return ProtocolSMTP
}

func (d *SMTPDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("SMTP server address is required")
	}
	opts := parseSMTPOptions(config.Options)
	if opts.From == "" {
		return fmt.Errorf("SMTP 'from' address is required")
	}
	if len(opts.To) == 0 {
		return fmt.Errorf("SMTP 'to' addresses are required")
	}
	return nil
}

func (d *SMTPDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()

	opts := parseSMTPOptions(config.Options)

	// Build email
	subject := opts.Subject
	if subject == "" {
		subject = "Webhook Notification"
	}

	// Format body based on format option
	var body string
	switch opts.BodyFormat {
	case "html":
		body = fmt.Sprintf(`<html><body>
<h2>Webhook Event</h2>
<pre>%s</pre>
</body></html>`, formatJSONForEmail(request.Payload))
	case "json":
		body = string(request.Payload)
	default:
		body = fmt.Sprintf("Webhook Event\n\n%s", formatJSONForEmail(request.Payload))
	}

	// Build MIME message
	contentType := "text/plain"
	if opts.BodyFormat == "html" {
		contentType = "text/html"
	} else if opts.BodyFormat == "json" {
		contentType = "application/json"
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: %s; charset=\"utf-8\"\r\n"+
		"X-WaaS-Delivery-ID: %s\r\n"+
		"X-WaaS-Webhook-ID: %s\r\n"+
		"\r\n%s",
		opts.From,
		strings.Join(opts.To, ", "),
		subject,
		contentType,
		request.ID,
		request.WebhookID,
		body,
	)
	if opts.ReplyTo != "" {
		msg = fmt.Sprintf("Reply-To: %s\r\n%s", opts.ReplyTo, msg)
	}

	// Determine SMTP server address
	addr := config.Target
	port := opts.Port
	if port == 0 {
		port = 587
	}
	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:%d", addr, port)
	}

	// Set up auth if configured
	var auth smtp.Auth
	if opts.Username != "" {
		host := strings.Split(addr, ":")[0]
		auth = smtp.PlainAuth("", opts.Username, opts.Password, host)
	}

	// Send email
	err := smtp.SendMail(addr, auth, opts.From, opts.To, []byte(msg))
	if err != nil {
		return &DeliveryResponse{
			Success:   false,
			Duration:  time.Since(start),
			Error:     fmt.Sprintf("SMTP delivery failed: %v", err),
			ErrorType: ErrorTypeConnection,
			ProtocolInfo: map[string]any{
				"protocol":   "smtp",
				"server":     addr,
				"recipients": len(opts.To),
			},
		}, nil
	}

	return &DeliveryResponse{
		Success:    true,
		StatusCode: 250, // SMTP success
		Duration:   time.Since(start),
		ProtocolInfo: map[string]any{
			"protocol":   "smtp",
			"server":     addr,
			"from":       opts.From,
			"recipients": len(opts.To),
		},
	}, nil
}

func (d *SMTPDeliverer) Close() error {
	return nil
}

func parseSMTPOptions(options map[string]interface{}) *SMTPOptions {
	opts := &SMTPOptions{Port: 587}
	if options == nil {
		return opts
	}
	if f, ok := options["from"].(string); ok {
		opts.From = f
	}
	if t, ok := options["to"].([]interface{}); ok {
		for _, addr := range t {
			if s, ok := addr.(string); ok {
				opts.To = append(opts.To, s)
			}
		}
	}
	if s, ok := options["subject"].(string); ok {
		opts.Subject = s
	}
	if bf, ok := options["body_format"].(string); ok {
		opts.BodyFormat = bf
	}
	if tls, ok := options["use_tls"].(bool); ok {
		opts.UseTLS = tls
	}
	if p, ok := options["port"].(float64); ok {
		opts.Port = int(p)
	}
	if u, ok := options["username"].(string); ok {
		opts.Username = u
	}
	if p, ok := options["password"].(string); ok {
		opts.Password = p
	}
	if r, ok := options["reply_to"].(string); ok {
		opts.ReplyTo = r
	}
	return opts
}

func formatJSONForEmail(payload []byte) string {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, payload, "", "  "); err != nil {
		return string(payload)
	}
	return pretty.String()
}

// applyAuth applies authentication to an HTTP request
func applyAuth(req *http.Request, auth *AuthConfig) {
	if auth == nil {
		return
	}
	switch auth.Type {
	case AuthBearer:
		if token, ok := auth.Credentials["token"]; ok {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case AuthBasic:
		username := auth.Credentials["username"]
		password := auth.Credentials["password"]
		req.SetBasicAuth(username, password)
	case AuthAPIKey:
		header := auth.Credentials["header"]
		value := auth.Credentials["value"]
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, value)
	}
}

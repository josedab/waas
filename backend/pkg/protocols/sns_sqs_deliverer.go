package protocols

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// --- SNS Deliverer ---

// SNSDeliverer implements webhook delivery via AWS SNS
type SNSDeliverer struct {
	clients map[string]*sns.Client
}

// SNSOptions represents SNS-specific delivery options
type SNSOptions struct {
	TopicARN       string            `json:"topic_arn"`
	MessageGroupID string            `json:"message_group_id,omitempty"` // FIFO topics
	Attributes     map[string]string `json:"attributes,omitempty"`
	Region         string            `json:"region,omitempty"`
	Subject        string            `json:"subject,omitempty"`
}

// NewSNSDeliverer creates a new SNS deliverer
func NewSNSDeliverer() *SNSDeliverer {
	return &SNSDeliverer{
		clients: make(map[string]*sns.Client),
	}
}

// Protocol returns the protocol
func (d *SNSDeliverer) Protocol() Protocol {
	return ProtocolSNS
}

// Validate validates the delivery config
func (d *SNSDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("SNS topic ARN is required")
	}
	if !strings.HasPrefix(config.Target, "arn:aws:sns:") {
		return fmt.Errorf("invalid SNS topic ARN format")
	}
	return nil
}

// Deliver performs the SNS delivery
func (d *SNSDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{
		ProtocolInfo: make(map[string]any),
	}

	opts := parseSNSOptions(config.Options)

	client, err := d.getClient(ctx, config, opts)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	topicARN := config.Target
	if opts.TopicARN != "" {
		topicARN = opts.TopicARN
	}

	// Build publish input
	input := &sns.PublishInput{
		TopicArn: &topicARN,
		Message:  strPtr(string(request.Payload)),
	}

	if opts.Subject != "" {
		input.Subject = &opts.Subject
	}

	// FIFO topic support
	if opts.MessageGroupID != "" {
		input.MessageGroupId = &opts.MessageGroupID
	}

	// Build message attributes
	if len(opts.Attributes) > 0 || request.WebhookID != "" {
		input.MessageAttributes = make(map[string]snstypes.MessageAttributeValue)

		for k, v := range opts.Attributes {
			val := v
			input.MessageAttributes[k] = snstypes.MessageAttributeValue{
				DataType:    strPtr("String"),
				StringValue: &val,
			}
		}

		if request.WebhookID != "" {
			input.MessageAttributes["WebhookID"] = snstypes.MessageAttributeValue{
				DataType:    strPtr("String"),
				StringValue: &request.WebhookID,
			}
		}
		if request.ID != "" {
			input.MessageAttributes["DeliveryID"] = snstypes.MessageAttributeValue{
				DataType:    strPtr("String"),
				StringValue: &request.ID,
			}
		}
		if request.ContentType != "" {
			input.MessageAttributes["ContentType"] = snstypes.MessageAttributeValue{
				DataType:    strPtr("String"),
				StringValue: &request.ContentType,
			}
		}
	}

	// Publish message
	result, err := client.Publish(ctx, input)
	response.Duration = time.Since(start)

	if err != nil {
		response.Error = err.Error()
		response.ErrorType = categorizeSNSSQSError(err)
		return response, nil
	}

	response.Success = true
	response.ProtocolInfo["topic_arn"] = topicARN
	if result.MessageId != nil {
		response.ProtocolInfo["message_id"] = *result.MessageId
	}
	if result.SequenceNumber != nil {
		response.ProtocolInfo["sequence_number"] = *result.SequenceNumber
	}

	return response, nil
}

// Close closes the deliverer
func (d *SNSDeliverer) Close() error {
	d.clients = make(map[string]*sns.Client)
	return nil
}

func (d *SNSDeliverer) getClient(ctx context.Context, config *DeliveryConfig, opts SNSOptions) (*sns.Client, error) {
	region := opts.Region
	if region == "" {
		// Extract region from ARN: arn:aws:sns:<region>:<account>:<topic>
		parts := strings.Split(config.Target, ":")
		if len(parts) >= 4 {
			region = parts[3]
		}
	}
	key := region

	if client, ok := d.clients[key]; ok {
		return client, nil
	}

	optFns := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		optFns = append(optFns, awsconfig.WithRegion(region))
	}

	// Static credentials from auth config
	if config.Auth != nil && config.Auth.Type == AuthAPIKey {
		accessKey := config.Auth.Credentials["key"]
		secretKey := config.Auth.Credentials["secret"]
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sns.NewFromConfig(cfg)
	d.clients[key] = client
	return client, nil
}

func parseSNSOptions(opts map[string]interface{}) SNSOptions {
	options := SNSOptions{}
	if opts == nil {
		return options
	}

	if arn, ok := opts["topic_arn"].(string); ok {
		options.TopicARN = arn
	}
	if gid, ok := opts["message_group_id"].(string); ok {
		options.MessageGroupID = gid
	}
	if region, ok := opts["region"].(string); ok {
		options.Region = region
	}
	if subject, ok := opts["subject"].(string); ok {
		options.Subject = subject
	}
	if attrs, ok := opts["attributes"].(map[string]interface{}); ok {
		options.Attributes = make(map[string]string)
		for k, v := range attrs {
			options.Attributes[k] = fmt.Sprintf("%v", v)
		}
	}

	return options
}

// --- SQS Deliverer ---

// SQSDeliverer implements webhook delivery via AWS SQS
type SQSDeliverer struct {
	clients map[string]*sqs.Client
}

// SQSOptions represents SQS-specific delivery options
type SQSOptions struct {
	QueueURL       string            `json:"queue_url"`
	MessageGroupID string            `json:"message_group_id,omitempty"` // FIFO queues
	DelaySeconds   int32             `json:"delay_seconds,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	Region         string            `json:"region,omitempty"`
}

// NewSQSDeliverer creates a new SQS deliverer
func NewSQSDeliverer() *SQSDeliverer {
	return &SQSDeliverer{
		clients: make(map[string]*sqs.Client),
	}
}

// Protocol returns the protocol
func (d *SQSDeliverer) Protocol() Protocol {
	return ProtocolSQS
}

// Validate validates the delivery config
func (d *SQSDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("SQS queue URL is required")
	}
	if !strings.Contains(config.Target, "sqs") || !strings.Contains(config.Target, "amazonaws.com") {
		return fmt.Errorf("invalid SQS queue URL format")
	}
	return nil
}

// Deliver performs the SQS delivery
func (d *SQSDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{
		ProtocolInfo: make(map[string]any),
	}

	opts := parseSQSOptions(config.Options)

	client, err := d.getClient(ctx, config, opts)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	queueURL := config.Target
	if opts.QueueURL != "" {
		queueURL = opts.QueueURL
	}

	// Build the message body as a structured envelope
	envelope := map[string]interface{}{
		"webhook_id":  request.WebhookID,
		"delivery_id": request.ID,
		"attempt":     request.AttemptNumber,
		"payload":     json.RawMessage(request.Payload),
	}
	if request.ContentType != "" {
		envelope["content_type"] = request.ContentType
	}

	bodyBytes, err := json.Marshal(envelope)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = fmt.Sprintf("failed to marshal message: %v", err)
		response.ErrorType = ErrorTypeProtocol
		return response, nil
	}

	// Build send message input
	input := &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: strPtr(string(bodyBytes)),
	}

	// FIFO queue support
	if opts.MessageGroupID != "" {
		input.MessageGroupId = &opts.MessageGroupID
	}

	// Delay seconds
	if opts.DelaySeconds > 0 {
		input.DelaySeconds = opts.DelaySeconds
	}

	// Message attributes
	input.MessageAttributes = make(map[string]sqstypes.MessageAttributeValue)

	if request.WebhookID != "" {
		input.MessageAttributes["WebhookID"] = sqstypes.MessageAttributeValue{
			DataType:    strPtr("String"),
			StringValue: &request.WebhookID,
		}
	}
	if request.ID != "" {
		input.MessageAttributes["DeliveryID"] = sqstypes.MessageAttributeValue{
			DataType:    strPtr("String"),
			StringValue: &request.ID,
		}
	}
	if request.ContentType != "" {
		input.MessageAttributes["ContentType"] = sqstypes.MessageAttributeValue{
			DataType:    strPtr("String"),
			StringValue: &request.ContentType,
		}
	}
	for k, v := range opts.Attributes {
		val := v
		input.MessageAttributes[k] = sqstypes.MessageAttributeValue{
			DataType:    strPtr("String"),
			StringValue: &val,
		}
	}

	// Send message
	result, err := client.SendMessage(ctx, input)
	response.Duration = time.Since(start)

	if err != nil {
		response.Error = err.Error()
		response.ErrorType = categorizeSNSSQSError(err)
		return response, nil
	}

	response.Success = true
	response.ProtocolInfo["queue_url"] = queueURL
	if result.MessageId != nil {
		response.ProtocolInfo["message_id"] = *result.MessageId
	}
	if result.SequenceNumber != nil {
		response.ProtocolInfo["sequence_number"] = *result.SequenceNumber
	}

	return response, nil
}

// Close closes the deliverer
func (d *SQSDeliverer) Close() error {
	d.clients = make(map[string]*sqs.Client)
	return nil
}

func (d *SQSDeliverer) getClient(ctx context.Context, config *DeliveryConfig, opts SQSOptions) (*sqs.Client, error) {
	region := opts.Region
	if region == "" {
		// Extract region from queue URL
		parts := strings.Split(config.Target, ".")
		for i, p := range parts {
			if p == "sqs" && i+1 < len(parts) {
				region = parts[i+1]
				break
			}
		}
	}
	key := region

	if client, ok := d.clients[key]; ok {
		return client, nil
	}

	optFns := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		optFns = append(optFns, awsconfig.WithRegion(region))
	}

	// Static credentials from auth config
	if config.Auth != nil && config.Auth.Type == AuthAPIKey {
		accessKey := config.Auth.Credentials["key"]
		secretKey := config.Auth.Credentials["secret"]
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sqs.NewFromConfig(cfg)
	d.clients[key] = client
	return client, nil
}

func parseSQSOptions(opts map[string]interface{}) SQSOptions {
	options := SQSOptions{}
	if opts == nil {
		return options
	}

	if url, ok := opts["queue_url"].(string); ok {
		options.QueueURL = url
	}
	if gid, ok := opts["message_group_id"].(string); ok {
		options.MessageGroupID = gid
	}
	if delay, ok := opts["delay_seconds"].(float64); ok {
		options.DelaySeconds = int32(delay)
	}
	if region, ok := opts["region"].(string); ok {
		options.Region = region
	}
	if attrs, ok := opts["attributes"].(map[string]interface{}); ok {
		options.Attributes = make(map[string]string)
		for k, v := range attrs {
			options.Attributes[k] = fmt.Sprintf("%v", v)
		}
	}

	return options
}

// categorizeSNSSQSError categorizes AWS SNS/SQS errors
func categorizeSNSSQSError(err error) DeliveryErrorType {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorTypeTimeout
	}
	if strings.Contains(errStr, "AccessDenied") || strings.Contains(errStr, "authorization") ||
		strings.Contains(errStr, "UnauthorizedAccess") {
		return ErrorTypeAuth
	}
	if strings.Contains(errStr, "NotFound") || strings.Contains(errStr, "NonExistentQueue") ||
		strings.Contains(errStr, "NonExistentTopic") {
		return ErrorTypeClientError
	}
	if strings.Contains(errStr, "Throttling") || strings.Contains(errStr, "throttl") {
		return ErrorTypeRateLimit
	}

	return ErrorTypeServer
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}

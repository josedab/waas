package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
)

// AWSKinesisProducer implements a production-ready AWS Kinesis producer
type AWSKinesisProducer struct {
	client  *kinesis.Client
	config  *AWSKinesisConfig
	mu      sync.RWMutex
	closed  bool
	metrics *ProducerMetrics
}

// AWSKinesisConfig holds AWS Kinesis-specific configuration
type AWSKinesisConfig struct {
	Region          string
	StreamName      string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Endpoint        string // For local testing
	MaxRecordSize   int
	MaxBatchSize    int
	MaxBatchRecords int
	FlushInterval   time.Duration
}

// DefaultAWSKinesisConfig returns default AWS Kinesis configuration
func DefaultAWSKinesisConfig() *AWSKinesisConfig {
	return &AWSKinesisConfig{
		Region:          "us-east-1",
		MaxRecordSize:   1024 * 1024,     // 1MB
		MaxBatchSize:    5 * 1024 * 1024, // 5MB
		MaxBatchRecords: 500,
		FlushInterval:   100 * time.Millisecond,
	}
}

// NewAWSKinesisProducer creates a new AWS Kinesis producer
func NewAWSKinesisProducer(kconfig *AWSKinesisConfig) (*AWSKinesisProducer, error) {
	if kconfig == nil {
		kconfig = DefaultAWSKinesisConfig()
	}

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(kconfig.Region))

	// Use explicit credentials if provided
	if kconfig.AccessKeyID != "" && kconfig.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			kconfig.AccessKeyID,
			kconfig.SecretAccessKey,
			kconfig.SessionToken,
		)
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var clientOpts []func(*kinesis.Options)
	if kconfig.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *kinesis.Options) {
			o.BaseEndpoint = aws.String(kconfig.Endpoint)
		})
	}

	client := kinesis.NewFromConfig(cfg, clientOpts...)

	return &AWSKinesisProducer{
		client:  client,
		config:  kconfig,
		metrics: &ProducerMetrics{},
	}, nil
}

// Send sends a message to Kinesis
func (p *AWSKinesisProducer) Send(ctx context.Context, streamName string, event *StreamEvent) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.New("producer is closed")
	}
	p.mu.RUnlock()

	if streamName == "" {
		streamName = p.config.StreamName
	}

	// Serialize message
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Determine partition key
	partitionKey := event.Key
	if partitionKey == "" {
		partitionKey = event.ID
	}

	input := &kinesis.PutRecordInput{
		StreamName:   aws.String(streamName),
		Data:         data,
		PartitionKey: aws.String(partitionKey),
	}

	result, err := p.client.PutRecord(ctx, input)
	if err != nil {
		p.metrics.mu.Lock()
		p.metrics.MessagesFailed++
		p.metrics.LastErrorTime = time.Now()
		p.metrics.LastError = err.Error()
		p.metrics.mu.Unlock()
		return fmt.Errorf("failed to put record: %w", err)
	}

	p.metrics.mu.Lock()
	p.metrics.MessagesSent++
	p.metrics.MessagesAcked++
	p.metrics.BytesSent += int64(len(data))
	p.metrics.LastSendTime = time.Now()
	p.metrics.LastAckTime = time.Now()
	p.metrics.mu.Unlock()

	// Store shard ID and sequence number in event for reference
	_ = result.ShardId
	_ = result.SequenceNumber

	return nil
}

// SendBatch sends multiple messages to Kinesis
func (p *AWSKinesisProducer) SendBatch(ctx context.Context, streamName string, events []*StreamEvent) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.New("producer is closed")
	}
	p.mu.RUnlock()

	if streamName == "" {
		streamName = p.config.StreamName
	}

	// Build records
	var records []types.PutRecordsRequestEntry
	var totalSize int

	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			continue
		}

		partitionKey := evt.Key
		if partitionKey == "" {
			partitionKey = evt.ID
		}

		// Check batch limits
		if totalSize+len(data) > p.config.MaxBatchSize || len(records) >= p.config.MaxBatchRecords {
			// Flush current batch
			if err := p.sendBatch(ctx, streamName, records); err != nil {
				return err
			}
			records = nil
			totalSize = 0
		}

		records = append(records, types.PutRecordsRequestEntry{
			Data:         data,
			PartitionKey: aws.String(partitionKey),
		})
		totalSize += len(data)
	}

	// Send remaining records
	if len(records) > 0 {
		return p.sendBatch(ctx, streamName, records)
	}

	return nil
}

func (p *AWSKinesisProducer) sendBatch(ctx context.Context, streamName string, records []types.PutRecordsRequestEntry) error {
	input := &kinesis.PutRecordsInput{
		StreamName: aws.String(streamName),
		Records:    records,
	}

	result, err := p.client.PutRecords(ctx, input)
	if err != nil {
		p.metrics.mu.Lock()
		p.metrics.MessagesFailed += int64(len(records))
		p.metrics.LastErrorTime = time.Now()
		p.metrics.LastError = err.Error()
		p.metrics.mu.Unlock()
		return fmt.Errorf("failed to put records: %w", err)
	}

	// Count successes and failures
	failed := int64(0)
	if result.FailedRecordCount != nil {
		failed = int64(*result.FailedRecordCount)
	}

	p.metrics.mu.Lock()
	p.metrics.MessagesSent += int64(len(records))
	p.metrics.MessagesAcked += int64(len(records)) - failed
	p.metrics.MessagesFailed += failed
	p.metrics.LastSendTime = time.Now()
	p.metrics.LastAckTime = time.Now()
	p.metrics.mu.Unlock()

	return nil
}

// Close closes the producer
func (p *AWSKinesisProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

// GetMetrics returns current producer metrics
func (p *AWSKinesisProducer) GetMetrics() ProducerMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// AWSKinesisConsumer implements a production-ready AWS Kinesis consumer
type AWSKinesisConsumer struct {
	client         *kinesis.Client
	config         *AWSKinesisConsumerConfig
	handler        func(ctx context.Context, topic string, event *StreamEvent) error
	mu             sync.RWMutex
	closed         bool
	metrics        *ConsumerMetrics
	cancelFunc     context.CancelFunc
	shardIterators map[string]string
}

// AWSKinesisConsumerConfig holds Kinesis consumer configuration
type AWSKinesisConsumerConfig struct {
	Region            string
	StreamName        string
	ConsumerName      string
	AccessKeyID       string
	SecretAccessKey   string
	SessionToken      string
	Endpoint          string
	ShardIteratorType types.ShardIteratorType
	PollInterval      time.Duration
	MaxRecordsPerPoll int32
}

// DefaultAWSKinesisConsumerConfig returns default consumer configuration
func DefaultAWSKinesisConsumerConfig() *AWSKinesisConsumerConfig {
	return &AWSKinesisConsumerConfig{
		Region:            "us-east-1",
		ShardIteratorType: types.ShardIteratorTypeLatest,
		PollInterval:      1 * time.Second,
		MaxRecordsPerPoll: 100,
	}
}

// NewAWSKinesisConsumer creates a new AWS Kinesis consumer
func NewAWSKinesisConsumer(kconfig *AWSKinesisConsumerConfig, handler func(ctx context.Context, topic string, event *StreamEvent) error) (*AWSKinesisConsumer, error) {
	if kconfig == nil {
		kconfig = DefaultAWSKinesisConsumerConfig()
	}

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(kconfig.Region))

	if kconfig.AccessKeyID != "" && kconfig.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			kconfig.AccessKeyID,
			kconfig.SecretAccessKey,
			kconfig.SessionToken,
		)
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var clientOpts []func(*kinesis.Options)
	if kconfig.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *kinesis.Options) {
			o.BaseEndpoint = aws.String(kconfig.Endpoint)
		})
	}

	client := kinesis.NewFromConfig(cfg, clientOpts...)

	return &AWSKinesisConsumer{
		client:         client,
		config:         kconfig,
		handler:        handler,
		metrics:        &ConsumerMetrics{},
		shardIterators: make(map[string]string),
	}, nil
}

// Start starts consuming messages
func (c *AWSKinesisConsumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	// Describe stream to get shards
	describeInput := &kinesis.DescribeStreamInput{
		StreamName: aws.String(c.config.StreamName),
	}

	result, err := c.client.DescribeStream(ctx, describeInput)
	if err != nil {
		return fmt.Errorf("failed to describe stream: %w", err)
	}

	// Start a consumer for each shard
	for _, shard := range result.StreamDescription.Shards {
		shardID := *shard.ShardId
		go c.consumeShard(ctx, shardID)
	}

	return nil
}

func (c *AWSKinesisConsumer) consumeShard(ctx context.Context, shardID string) {
	// Get shard iterator
	iteratorInput := &kinesis.GetShardIteratorInput{
		StreamName:        aws.String(c.config.StreamName),
		ShardId:           aws.String(shardID),
		ShardIteratorType: c.config.ShardIteratorType,
	}

	iteratorResult, err := c.client.GetShardIterator(ctx, iteratorInput)
	if err != nil {
		c.metrics.mu.Lock()
		c.metrics.LastErrorTime = time.Now()
		c.metrics.LastError = fmt.Sprintf("failed to get shard iterator: %v", err)
		c.metrics.mu.Unlock()
		return
	}

	shardIterator := iteratorResult.ShardIterator
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if shardIterator == nil {
				return
			}

			getRecordsInput := &kinesis.GetRecordsInput{
				ShardIterator: shardIterator,
				Limit:         aws.Int32(c.config.MaxRecordsPerPoll),
			}

			result, err := c.client.GetRecords(ctx, getRecordsInput)
			if err != nil {
				c.metrics.mu.Lock()
				c.metrics.LastErrorTime = time.Now()
				c.metrics.LastError = err.Error()
				c.metrics.mu.Unlock()
				continue
			}

			for _, record := range result.Records {
				c.metrics.mu.Lock()
				c.metrics.MessagesReceived++
				c.metrics.LastReceiveTime = time.Now()
				c.metrics.mu.Unlock()

				// Parse event
				var event StreamEvent
				if err := json.Unmarshal(record.Data, &event); err != nil {
					event = StreamEvent{
						ID:        *record.SequenceNumber,
						Key:       *record.PartitionKey,
						Value:     json.RawMessage(record.Data),
						Timestamp: *record.ApproximateArrivalTimestamp,
					}
				}

				// Call handler
				if err := c.handler(ctx, c.config.StreamName, &event); err != nil {
					c.metrics.mu.Lock()
					c.metrics.MessagesFailed++
					c.metrics.LastErrorTime = time.Now()
					c.metrics.LastError = err.Error()
					c.metrics.mu.Unlock()
					continue
				}

				c.metrics.mu.Lock()
				c.metrics.MessagesProcessed++
				c.metrics.LastProcessTime = time.Now()
				c.metrics.mu.Unlock()
			}

			shardIterator = result.NextShardIterator
		}
	}
}

// Stop stops consuming messages
func (c *AWSKinesisConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	return nil
}

// GetMetrics returns current consumer metrics
func (c *AWSKinesisConsumer) GetMetrics() ConsumerMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	return *c.metrics
}

// AWSKinesisAdmin provides Kinesis admin operations
type AWSKinesisAdmin struct {
	client *kinesis.Client
	config *AWSKinesisConfig
}

// NewAWSKinesisAdmin creates a new Kinesis admin client
func NewAWSKinesisAdmin(kconfig *AWSKinesisConfig) (*AWSKinesisAdmin, error) {
	if kconfig == nil {
		kconfig = DefaultAWSKinesisConfig()
	}

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(kconfig.Region))

	if kconfig.AccessKeyID != "" && kconfig.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			kconfig.AccessKeyID,
			kconfig.SecretAccessKey,
			kconfig.SessionToken,
		)
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var clientOpts []func(*kinesis.Options)
	if kconfig.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *kinesis.Options) {
			o.BaseEndpoint = aws.String(kconfig.Endpoint)
		})
	}

	client := kinesis.NewFromConfig(cfg, clientOpts...)

	return &AWSKinesisAdmin{
		client: client,
		config: kconfig,
	}, nil
}

// CreateStream creates a Kinesis stream
func (a *AWSKinesisAdmin) CreateStream(name string, shardCount int32) error {
	input := &kinesis.CreateStreamInput{
		StreamName: aws.String(name),
		ShardCount: aws.Int32(shardCount),
	}

	_, err := a.client.CreateStream(context.Background(), input)
	return err
}

// DeleteStream deletes a Kinesis stream
func (a *AWSKinesisAdmin) DeleteStream(name string) error {
	input := &kinesis.DeleteStreamInput{
		StreamName: aws.String(name),
	}

	_, err := a.client.DeleteStream(context.Background(), input)
	return err
}

// DescribeStream describes a Kinesis stream
func (a *AWSKinesisAdmin) DescribeStream(name string) (*kinesis.DescribeStreamOutput, error) {
	input := &kinesis.DescribeStreamInput{
		StreamName: aws.String(name),
	}

	return a.client.DescribeStream(context.Background(), input)
}

// ListStreams lists all Kinesis streams
func (a *AWSKinesisAdmin) ListStreams() ([]string, error) {
	input := &kinesis.ListStreamsInput{}

	result, err := a.client.ListStreams(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return result.StreamNames, nil
}

// UpdateShardCount updates the shard count of a stream
func (a *AWSKinesisAdmin) UpdateShardCount(name string, targetShardCount int32) error {
	input := &kinesis.UpdateShardCountInput{
		StreamName:       aws.String(name),
		TargetShardCount: aws.Int32(targetShardCount),
		ScalingType:      types.ScalingTypeUniformScaling,
	}

	_, err := a.client.UpdateShardCount(context.Background(), input)
	return err
}

package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kod2ulz/gostart/logr"
)

// RedisStreamsConfig holds Redis Streams configuration
type RedisStreamsConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

// RedisStreamsProducer wraps a Redis client for producing stream messages
type RedisStreamsProducer struct {
	client *redis.Client
	logger *logr.Logger
}

// NewRedisStreamsProducer creates a new Redis Streams producer
func NewRedisStreamsProducer(config RedisStreamsConfig, logger *logr.Logger) (*RedisStreamsProducer, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStreamsProducer{
		client: client,
		logger: logger,
	}, nil
}

// Publish publishes a message to a Redis stream
func (rsp *RedisStreamsProducer) Publish(ctx context.Context, stream string, message interface{}) (string, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	values := map[string]interface{}{
		"data":      string(data),
		"timestamp": time.Now().Unix(),
	}

	messageID, err := rsp.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()

	if err != nil {
		return "", fmt.Errorf("failed to add message to stream: %w", err)
	}

	rsp.logger.Debug("Message published to stream",
		"stream", stream,
		"message_id", messageID,
	)

	return messageID, nil
}

// PublishWithID publishes a message with a specific ID
func (rsp *RedisStreamsProducer) PublishWithID(ctx context.Context, stream string, id string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	values := map[string]interface{}{
		"data":      string(data),
		"timestamp": time.Now().Unix(),
	}

	_, err = rsp.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		ID:     id,
		Values: values,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add message to stream: %w", err)
	}

	return nil
}

// PublishBatch publishes multiple messages in a pipeline
func (rsp *RedisStreamsProducer) PublishBatch(ctx context.Context, stream string, messages []interface{}) ([]string, error) {
	pipe := rsp.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(messages))

	for i, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message %d: %w", i, err)
		}

		values := map[string]interface{}{
			"data":      string(data),
			"timestamp": time.Now().Unix(),
		}

		cmds[i] = pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: values,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to execute batch: %w", err)
	}

	messageIDs := make([]string, len(cmds))
	for i, cmd := range cmds {
		messageIDs[i] = cmd.Val()
	}

	return messageIDs, nil
}

// Trim trims the stream to a maximum length
func (rsp *RedisStreamsProducer) Trim(ctx context.Context, stream string, maxLen int64) error {
	_, err := rsp.client.XTrimMaxLen(ctx, stream, maxLen).Result()
	return err
}

// Close closes the Redis client
func (rsp *RedisStreamsProducer) Close() error {
	return rsp.client.Close()
}

// RedisStreamsConsumer wraps a Redis client for consuming stream messages
type RedisStreamsConsumer struct {
	client       *redis.Client
	logger       *logr.Logger
	consumerGroup string
	consumerName  string
	handlers     map[string]StreamHandler
}

// StreamHandler processes a Redis stream message
type StreamHandler func(ctx context.Context, message redis.XMessage) error

// NewRedisStreamsConsumer creates a new Redis Streams consumer
func NewRedisStreamsConsumer(config RedisStreamsConfig, consumerGroup, consumerName string, logger *logr.Logger) (*RedisStreamsConsumer, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStreamsConsumer{
		client:        client,
		logger:        logger,
		consumerGroup: consumerGroup,
		consumerName:  consumerName,
		handlers:      make(map[string]StreamHandler),
	}, nil
}

// CreateGroup creates a consumer group for a stream
func (rsc *RedisStreamsConsumer) CreateGroup(ctx context.Context, stream string, startID string) error {
	if startID == "" {
		startID = "0" // Start from beginning
	}

	err := rsc.client.XGroupCreateMkStream(ctx, stream, rsc.consumerGroup, startID).Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	rsc.logger.Info("Consumer group created or already exists",
		"stream", stream,
		"group", rsc.consumerGroup,
	)

	return nil
}

// RegisterHandler registers a handler for a specific stream
func (rsc *RedisStreamsConsumer) RegisterHandler(stream string, handler StreamHandler) {
	rsc.handlers[stream] = handler
}

// Start starts consuming messages from streams
func (rsc *RedisStreamsConsumer) Start(ctx context.Context, streams []string, blockDuration time.Duration) error {
	if blockDuration == 0 {
		blockDuration = 5 * time.Second
	}

	rsc.logger.Info("Starting Redis Streams consumer",
		"streams", streams,
		"group", rsc.consumerGroup,
		"consumer", rsc.consumerName,
	)

	// Create consumer groups for all streams
	for _, stream := range streams {
		if err := rsc.CreateGroup(ctx, stream, "$"); err != nil {
			return err
		}
	}

	// Build streams argument
	streamsArg := make([]string, len(streams)*2)
	for i, stream := range streams {
		streamsArg[i] = stream
		streamsArg[len(streams)+i] = ">" // Read new messages
	}

	for {
		select {
		case <-ctx.Done():
			rsc.logger.Info("Stopping Redis Streams consumer")
			return nil
		default:
			messages, err := rsc.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    rsc.consumerGroup,
				Consumer: rsc.consumerName,
				Streams:  streamsArg,
				Count:    10,
				Block:    blockDuration,
			}).Result()

			if err != nil && err != redis.Nil {
				rsc.logger.Error("Error reading from stream", "error", err)
				continue
			}

			// Process messages
			for _, stream := range messages {
				for _, message := range stream.Messages {
					if err := rsc.handleMessage(ctx, stream.Stream, message); err != nil {
						rsc.logger.Error("Error handling message",
							"error", err,
							"stream", stream.Stream,
							"message_id", message.ID,
						)
						// NACK the message
						continue
					}

					// ACK the message
					if err := rsc.Ack(ctx, stream.Stream, message.ID); err != nil {
						rsc.logger.Error("Failed to acknowledge message",
							"error", err,
							"stream", stream.Stream,
							"message_id", message.ID,
						)
					}
				}
			}
		}
	}
}

// handleMessage handles a single message
func (rsc *RedisStreamsConsumer) handleMessage(ctx context.Context, stream string, message redis.XMessage) error {
	rsc.logger.Debug("Received message",
		"stream", stream,
		"message_id", message.ID,
	)

	// Find handler for stream
	handler, exists := rsc.handlers[stream]
	if !exists {
		rsc.logger.Warn("No handler registered for stream", "stream", stream)
		return nil
	}

	// Execute handler
	if err := handler(ctx, message); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	return nil
}

// Ack acknowledges a message
func (rsc *RedisStreamsConsumer) Ack(ctx context.Context, stream string, messageID string) error {
	return rsc.client.XAck(ctx, stream, rsc.consumerGroup, messageID).Err()
}

// Claim claims pending messages
func (rsc *RedisStreamsConsumer) Claim(ctx context.Context, stream string, minIdleTime time.Duration, messageIDs []string) ([]redis.XMessage, error) {
	messages, err := rsc.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   stream,
		Group:    rsc.consumerGroup,
		Consumer: rsc.consumerName,
		MinIdle:  minIdleTime,
		Messages: messageIDs,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to claim messages: %w", err)
	}

	return messages, nil
}

// GetPending gets pending messages for the consumer
func (rsc *RedisStreamsConsumer) GetPending(ctx context.Context, stream string, count int64) ([]redis.XPendingExt, error) {
	pending, err := rsc.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   stream,
		Group:    rsc.consumerGroup,
		Consumer: rsc.consumerName,
		Start:    "-",
		End:      "+",
		Count:    count,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}

	return pending, nil
}

// ProcessPending processes pending messages
func (rsc *RedisStreamsConsumer) ProcessPending(ctx context.Context, stream string, minIdleTime time.Duration) error {
	pending, err := rsc.GetPending(ctx, stream, 100)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	messageIDs := make([]string, len(pending))
	for i, msg := range pending {
		messageIDs[i] = msg.ID
	}

	messages, err := rsc.Claim(ctx, stream, minIdleTime, messageIDs)
	if err != nil {
		return err
	}

	for _, message := range messages {
		if err := rsc.handleMessage(ctx, stream, message); err != nil {
			rsc.logger.Error("Error handling pending message",
				"error", err,
				"stream", stream,
				"message_id", message.ID,
			)
			continue
		}

		rsc.Ack(ctx, stream, message.ID)
	}

	return nil
}

// Close closes the Redis client
func (rsc *RedisStreamsConsumer) Close() error {
	return rsc.client.Close()
}

// Example usage
func ExampleRedisStreams() {
	logger := logr.Log()

	config := RedisStreamsConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	// Producer example
	producer, _ := NewRedisStreamsProducer(config, logger)
	defer producer.Close()

	ctx := context.Background()
	message := map[string]interface{}{
		"user_id": 123,
		"action":  "login",
		"timestamp": time.Now(),
	}

	messageID, _ := producer.Publish(ctx, "user-events", message)
	logger.Info("Published message", "id", messageID)

	// Consumer example
	consumer, _ := NewRedisStreamsConsumer(config, "my-group", "consumer-1", logger)
	defer consumer.Close()

	consumer.RegisterHandler("user-events", func(ctx context.Context, msg redis.XMessage) error {
		dataStr := msg.Values["data"].(string)
		var event map[string]interface{}
		json.Unmarshal([]byte(dataStr), &event)
		logger.Info("Received event", "event", event)
		return nil
	})

	consumer.Start(ctx, []string{"user-events"}, 5*time.Second)
}

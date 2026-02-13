package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kod2ulz/gostart/logr"
)

// KafkaConfig holds Kafka connection configuration
type KafkaConfig struct {
	Brokers       []string
	GroupID       string
	ClientID      string
	SecurityProtocol string
	SASLMechanism string
	SASLUsername  string
	SASLPassword  string
	EnableSSL     bool
	SSLCALocation string
	AutoOffsetReset string
	EnableAutoCommit bool
	SessionTimeout int // milliseconds
}

// KafkaProducer wraps a Kafka producer
type KafkaProducer struct {
	producer *kafka.Producer
	logger   *logr.Logger
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(config KafkaConfig, logger *logr.Logger) (*KafkaProducer, error) {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": joinStrings(config.Brokers, ","),
		"client.id":        config.ClientID,
	}

	// Add security configuration if needed
	if config.SecurityProtocol != "" {
		kafkaConfig.SetKey("security.protocol", config.SecurityProtocol)
	}
	if config.SASLMechanism != "" {
		kafkaConfig.SetKey("sasl.mechanism", config.SASLMechanism)
		kafkaConfig.SetKey("sasl.username", config.SASLUsername)
		kafkaConfig.SetKey("sasl.password", config.SASLPassword)
	}
	if config.EnableSSL {
		kafkaConfig.SetKey("ssl.ca.location", config.SSLCALocation)
	}

	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	kp := &KafkaProducer{
		producer: producer,
		logger:   logger,
	}

	// Start delivery report handler
	go kp.handleDeliveryReports()

	return kp, nil
}

// Publish publishes a message to a Kafka topic
func (kp *KafkaProducer) Publish(ctx context.Context, topic string, key string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          data,
		Headers: []kafka.Header{
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	deliveryChan := make(chan kafka.Event, 1)
	err = kp.producer.Produce(kafkaMsg, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery report
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
		kp.logger.Debug("Message delivered",
			"topic", topic,
			"partition", m.TopicPartition.Partition,
			"offset", m.TopicPartition.Offset,
		)
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// PublishBatch publishes multiple messages in a batch
func (kp *KafkaProducer) PublishBatch(ctx context.Context, topic string, messages []interface{}) error {
	for i, msg := range messages {
		key := fmt.Sprintf("batch-%d", i)
		if err := kp.Publish(ctx, topic, key, msg); err != nil {
			return fmt.Errorf("failed to publish message %d: %w", i, err)
		}
	}
	return nil
}

// handleDeliveryReports handles delivery reports
func (kp *KafkaProducer) handleDeliveryReports() {
	for e := range kp.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				kp.logger.Error("Delivery failed",
					"error", ev.TopicPartition.Error,
					"topic", *ev.TopicPartition.Topic,
				)
			}
		}
	}
}

// Close closes the Kafka producer
func (kp *KafkaProducer) Close() {
	// Flush any remaining messages
	kp.producer.Flush(10 * 1000) // 10 seconds
	kp.producer.Close()
}

// KafkaConsumer wraps a Kafka consumer
type KafkaConsumer struct {
	consumer *kafka.Consumer
	logger   *logr.Logger
	handlers map[string]MessageHandler
}

// MessageHandler processes a Kafka message
type MessageHandler func(ctx context.Context, message *kafka.Message) error

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(config KafkaConfig, logger *logr.Logger) (*KafkaConsumer, error) {
	if config.AutoOffsetReset == "" {
		config.AutoOffsetReset = "earliest"
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 6000
	}

	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":  joinStrings(config.Brokers, ","),
		"group.id":          config.GroupID,
		"client.id":         config.ClientID,
		"auto.offset.reset": config.AutoOffsetReset,
		"enable.auto.commit": config.EnableAutoCommit,
		"session.timeout.ms": config.SessionTimeout,
	}

	// Add security configuration if needed
	if config.SecurityProtocol != "" {
		kafkaConfig.SetKey("security.protocol", config.SecurityProtocol)
	}
	if config.SASLMechanism != "" {
		kafkaConfig.SetKey("sasl.mechanism", config.SASLMechanism)
		kafkaConfig.SetKey("sasl.username", config.SASLUsername)
		kafkaConfig.SetKey("sasl.password", config.SASLPassword)
	}
	if config.EnableSSL {
		kafkaConfig.SetKey("ssl.ca.location", config.SSLCALocation)
	}

	consumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &KafkaConsumer{
		consumer: consumer,
		logger:   logger,
		handlers: make(map[string]MessageHandler),
	}, nil
}

// Subscribe subscribes to Kafka topics
func (kc *KafkaConsumer) Subscribe(topics []string) error {
	err := kc.consumer.SubscribeTopics(topics, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}
	kc.logger.Info("Subscribed to topics", "topics", topics)
	return nil
}

// RegisterHandler registers a handler for a specific topic
func (kc *KafkaConsumer) RegisterHandler(topic string, handler MessageHandler) {
	kc.handlers[topic] = handler
}

// Start starts consuming messages
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	kc.logger.Info("Starting Kafka consumer")

	for {
		select {
		case <-ctx.Done():
			kc.logger.Info("Stopping Kafka consumer")
			return nil
		default:
			msg, err := kc.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Timeout is expected
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				kc.logger.Error("Error reading message", "error", err)
				continue
			}

			if err := kc.handleMessage(ctx, msg); err != nil {
				kc.logger.Error("Error handling message",
					"error", err,
					"topic", *msg.TopicPartition.Topic,
					"partition", msg.TopicPartition.Partition,
					"offset", msg.TopicPartition.Offset,
				)
			}
		}
	}
}

// handleMessage handles a single message
func (kc *KafkaConsumer) handleMessage(ctx context.Context, msg *kafka.Message) error {
	topic := *msg.TopicPartition.Topic

	kc.logger.Debug("Received message",
		"topic", topic,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset,
		"key", string(msg.Key),
	)

	// Find handler for topic
	handler, exists := kc.handlers[topic]
	if !exists {
		kc.logger.Warn("No handler registered for topic", "topic", topic)
		return nil
	}

	// Execute handler
	if err := handler(ctx, msg); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	// Commit offset if auto-commit is disabled
	if !kc.consumer.GetRebalanceProtocol() {
		_, err := kc.consumer.CommitMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to commit offset: %w", err)
		}
	}

	return nil
}

// Close closes the Kafka consumer
func (kc *KafkaConsumer) Close() error {
	return kc.consumer.Close()
}

// KafkaStreamProcessor processes Kafka messages with advanced features
type KafkaStreamProcessor struct {
	consumer    *KafkaConsumer
	producer    *KafkaProducer
	logger      *logr.Logger
	concurrency int
}

// NewKafkaStreamProcessor creates a new stream processor
func NewKafkaStreamProcessor(consumer *KafkaConsumer, producer *KafkaProducer, logger *logr.Logger, concurrency int) *KafkaStreamProcessor {
	if concurrency <= 0 {
		concurrency = 1
	}

	return &KafkaStreamProcessor{
		consumer:    consumer,
		producer:    producer,
		logger:      logger,
		concurrency: concurrency,
	}
}

// Process processes messages with transformation and output
func (ksp *KafkaStreamProcessor) Process(
	ctx context.Context,
	inputTopic string,
	outputTopic string,
	transform func(context.Context, []byte) (interface{}, error),
) error {
	// Subscribe to input topic
	if err := ksp.consumer.Subscribe([]string{inputTopic}); err != nil {
		return err
	}

	// Register handler
	ksp.consumer.RegisterHandler(inputTopic, func(ctx context.Context, msg *kafka.Message) error {
		// Transform message
		output, err := transform(ctx, msg.Value)
		if err != nil {
			ksp.logger.Error("Transformation error", "error", err)
			return err
		}

		// Publish to output topic
		key := string(msg.Key)
		if err := ksp.producer.Publish(ctx, outputTopic, key, output); err != nil {
			ksp.logger.Error("Failed to publish transformed message", "error", err)
			return err
		}

		return nil
	})

	// Start consuming
	return ksp.consumer.Start(ctx)
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// Example usage
func ExampleKafka() {
	logger := logr.Log()

	config := KafkaConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "my-consumer-group",
		ClientID: "my-app",
	}

	// Producer example
	producer, _ := NewKafkaProducer(config, logger)
	defer producer.Close()

	ctx := context.Background()
	message := map[string]interface{}{
		"user_id": 123,
		"action":  "login",
		"timestamp": time.Now(),
	}

	producer.Publish(ctx, "user-events", "user-123", message)

	// Consumer example
	consumer, _ := NewKafkaConsumer(config, logger)
	defer consumer.Close()

	consumer.Subscribe([]string{"user-events"})

	consumer.RegisterHandler("user-events", func(ctx context.Context, msg *kafka.Message) error {
		var event map[string]interface{}
		json.Unmarshal(msg.Value, &event)
		logger.Info("Received event", "event", event)
		return nil
	})

	consumer.Start(ctx)
}

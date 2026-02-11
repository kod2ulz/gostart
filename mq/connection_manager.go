package mq

import (
	"context"
	"fmt"
	"sync"

	"github.com/kod2ulz/gostart/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ConnectionManager manages both publisher and consumer connections with proper retry logic
type ConnectionManager struct {
	config *Conf
	logger *logr.Logger
	ctx    context.Context

	publisherConn *RetryableConnection
	consumerConn  *RetryableConnection

	exchanges map[string]*ManagedExchange
	queues    map[string]*ManagedQueue

	mu sync.RWMutex
}

// ManagedExchange represents an exchange with both connections
type ManagedExchange struct {
	name    string
	kind    string
	options ExchangeOptions

	publisherConn *RetryableConnection
	consumerConn  *RetryableConnection

	logger *logr.Logger
}

// ManagedQueue represents a queue with both connections
type ManagedQueue struct {
	name    string
	options QueueOptions

	publisherConn *RetryableConnection
	consumerConn  *RetryableConnection

	logger *logr.Logger
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(ctx context.Context, config *Conf, logger *logr.Logger) *ConnectionManager {
	return &ConnectionManager{
		config:    config,
		logger:    logger,
		ctx:       ctx,
		exchanges: make(map[string]*ManagedExchange),
		queues:    make(map[string]*ManagedQueue),
	}
}

// InitializeConnections establishes both publisher and consumer connections
func (cm *ConnectionManager) InitializeConnections() error {
	cm.logger.Info("Initializing connection manager")

	// Create publisher connection
	cm.publisherConn = NewRetryableConnection(cm.ctx, cm.config, cm.logger.ExtendWithField("connection", "publisher"))
	if err := cm.publisherConn.Connect(); err != nil {
		return fmt.Errorf("failed to establish publisher connection: %w", err)
	}

	// Create consumer connection
	cm.consumerConn = NewRetryableConnection(cm.ctx, cm.config, cm.logger.ExtendWithField("connection", "consumer"))
	if err := cm.consumerConn.Connect(); err != nil {
		return fmt.Errorf("failed to establish consumer connection: %w", err)
	}

	cm.logger.Info("Connection manager initialized successfully")
	return nil
}

// DeclareExchange creates or gets an exchange
func (cm *ConnectionManager) DeclareExchange(name, kind string, options ExchangeOptions) (*ManagedExchange, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existing, exists := cm.exchanges[name]; exists {
		cm.logger.Warn("Exchange already exists", "exchange", name)
		return existing, nil
	}

	exchange := &ManagedExchange{
		name:          name,
		kind:          kind,
		options:       options,
		publisherConn: cm.publisherConn,
		consumerConn:  cm.consumerConn,
		logger:        cm.logger.ExtendWithField("exchange", name),
	}

	// Declare exchange on both connections
	if err := exchange.declareOnConnection(cm.publisherConn); err != nil {
		return nil, fmt.Errorf("failed to declare exchange on publisher connection: %w", err)
	}

	if err := exchange.declareOnConnection(cm.consumerConn); err != nil {
		return nil, fmt.Errorf("failed to declare exchange on consumer connection: %w", err)
	}

	cm.exchanges[name] = exchange
	cm.logger.Info("Exchange declared", "exchange", name, "type", kind)

	return exchange, nil
}

// DeclareQueue creates or gets a queue
func (cm *ConnectionManager) DeclareQueue(name string, options QueueOptions) (*ManagedQueue, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existing, exists := cm.queues[name]; exists {
		cm.logger.Warn("Queue already exists", "queue", name)
		return existing, nil
	}

	queue := &ManagedQueue{
		name:          name,
		options:       options,
		publisherConn: cm.publisherConn,
		consumerConn:  cm.consumerConn,
		logger:        cm.logger.ExtendWithField("queue", name),
	}

	// Declare queue on both connections
	if err := queue.declareOnConnection(cm.publisherConn); err != nil {
		return nil, fmt.Errorf("failed to declare queue on publisher connection: %w", err)
	}

	if err := queue.declareOnConnection(cm.consumerConn); err != nil {
		return nil, fmt.Errorf("failed to declare queue on consumer connection: %w", err)
	}

	cm.queues[name] = queue
	cm.logger.Info("Queue declared", "queue", name)

	return queue, nil
}

// GetExchange returns an existing exchange
func (cm *ConnectionManager) GetExchange(name string) (*ManagedExchange, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	exchange, exists := cm.exchanges[name]
	if !exists {
		return nil, fmt.Errorf("exchange '%s' not found", name)
	}

	return exchange, nil
}

// GetQueue returns an existing queue
func (cm *ConnectionManager) GetQueue(name string) (*ManagedQueue, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	queue, exists := cm.queues[name]
	if !exists {
		return nil, fmt.Errorf("queue '%s' not found", name)
	}

	return queue, nil
}

// Close gracefully shuts down all connections
func (cm *ConnectionManager) Close() error {
	cm.logger.Info("Closing connection manager")

	var errs []error

	if cm.publisherConn != nil {
		if err := cm.publisherConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("publisher connection close error: %w", err))
		}
	}

	if cm.consumerConn != nil {
		if err := cm.consumerConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("consumer connection close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred while closing: %v", errs)
	}

	cm.logger.Info("Connection manager closed successfully")
	return nil
}

// IsConnected returns whether both connections are active
func (cm *ConnectionManager) IsConnected() bool {
	return cm.publisherConn != nil && cm.publisherConn.IsConnected() &&
		cm.consumerConn != nil && cm.consumerConn.IsConnected()
}

// GetStats returns statistics about the connection manager
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"connected": cm.IsConnected(),
		"exchanges": len(cm.exchanges),
		"queues":    len(cm.queues),
		"publisher": cm.publisherConn.GetStats(),
		"consumer":  cm.consumerConn.GetStats(),
	}
}

// ManagedExchange methods
func (me *ManagedExchange) declareOnConnection(conn *RetryableConnection) error {
	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	return channel.ExchangeDeclare(
		me.name,
		me.kind,
		me.options.Durable,
		me.options.AutoDelete,
		me.options.Internal,
		me.options.NoWait,
		me.options.Arguments,
	)
}

func (me *ManagedExchange) Publish(routingKey string, message Message) error {
	channel, err := me.publisherConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get publisher channel: %w", err)
	}

	return channel.Publish(
		me.name,
		routingKey,
		true,  // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: message.ContentType,
			Headers:     message.Headers,
			Body:        message.Body,
		},
	)
}

func (me *ManagedExchange) Consume(queue, consumer string, options ConsumeOptions) (<-chan amqp.Delivery, error) {
	channel, err := me.consumerConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer channel: %w", err)
	}

	return channel.Consume(
		queue,
		consumer,
		options.AutoAck,
		options.Exclusive,
		options.NoLocal,
		options.NoWait,
		options.Arguments,
	)
}

func (me *ManagedExchange) Name() string {
	return me.name
}

// ManagedQueue methods
func (mq *ManagedQueue) declareOnConnection(conn *RetryableConnection) error {
	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	_, err = channel.QueueDeclare(
		mq.name,
		mq.options.Durable,
		mq.options.AutoDelete,
		mq.options.Exclusive,
		mq.options.NoWait,
		mq.options.Arguments,
	)

	return err
}

func (mq *ManagedQueue) Bind(exchange, routingKey string) error {
	channel, err := mq.consumerConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get consumer channel: %w", err)
	}

	return channel.QueueBind(
		mq.name,
		routingKey,
		exchange,
		false, // noWait
		nil,   // args
	)
}

func (mq *ManagedQueue) Consume(consumer string, options ConsumeOptions) (<-chan amqp.Delivery, error) {
	channel, err := mq.consumerConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer channel: %w", err)
	}

	return channel.Consume(
		mq.name,
		consumer,
		options.AutoAck,
		options.Exclusive,
		options.NoLocal,
		options.NoWait,
		options.Arguments,
	)
}

func (mq *ManagedQueue) Name() string {
	return mq.name
}

package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQConnection implements the Connection interface for RabbitMQ
type RabbitMQConnection struct {
	manager *ConnectionManager
	logger  *logr.Logger
	ctx     context.Context
}

// RabbitMQPublisher implements the Publisher interface
type RabbitMQPublisher struct {
	exchange *ManagedExchange
	logger   *logr.Logger
	defaultKey string
}

// RabbitMQConsumer implements the NewConsumer interface
type RabbitMQConsumer struct {
	queue         *ManagedQueue
	exchange      *ManagedExchange
	bindingKeys   []string
	processor     Processor
	options       WorkerOptions
	logger        *logr.Logger
	ctx           context.Context
	cancel        context.CancelFunc

	messages      <-chan amqp.Delivery
	running       bool
	mu            sync.RWMutex
}

// RabbitMQWorkerManager implements the NewWorkerManager interface
type RabbitMQWorkerManager struct {
	manager *ConnectionManager
	logger  *logr.Logger
	ctx     context.Context
}

// NewRabbitMQConnection creates a new RabbitMQ connection
func NewRabbitMQConnection(ctx context.Context, config *Conf, logger *logr.Logger) (*RabbitMQConnection, error) {
	manager := NewConnectionManager(ctx, config, logger)
	if err := manager.InitializeConnections(); err != nil {
		return nil, fmt.Errorf("failed to initialize connection manager: %w", err)
	}

	return &RabbitMQConnection{
		manager: manager,
		logger:  logger,
		ctx:     ctx,
	}, nil
}

// Connection interface methods
func (r *RabbitMQConnection) Close() error {
	return r.manager.Close()
}

func (r *RabbitMQConnection) IsConnected() bool {
	return r.manager.IsConnected()
}

func (r *RabbitMQConnection) DeclareExchange(name, kind string, opts ExchangeOptions) error {
	_, err := r.manager.DeclareExchange(name, kind, opts)
	return err
}

func (r *RabbitMQConnection) DeleteExchange(name string) error {
	// Implementation needed
	return fmt.Errorf("DeleteExchange not yet implemented")
}

func (r *RabbitMQConnection) DeclareQueue(name string, opts QueueOptions) error {
	_, err := r.manager.DeclareQueue(name, opts)
	return err
}

func (r *RabbitMQConnection) DeleteQueue(name string) error {
	// Implementation needed
	return fmt.Errorf("DeleteQueue not yet implemented")
}

func (r *RabbitMQConnection) BindQueue(queue, exchange, key string) error {
	q, err := r.manager.GetQueue(queue)
	if err != nil {
		return err
	}
	return q.Bind(exchange, key)
}

func (r *RabbitMQConnection) UnbindQueue(queue, exchange, key string) error {
	// Implementation needed
	return fmt.Errorf("UnbindQueue not yet implemented")
}

func (r *RabbitMQConnection) Publish(exchange, key string, message Message) error {
	exc, err := r.manager.GetExchange(exchange)
	if err != nil {
		return err
	}
	return exc.Publish(key, message)
}

func (r *RabbitMQConnection) Consume(queue, consumer string, opts ConsumeOptions) (<-chan Message, error) {
	q, err := r.manager.GetQueue(queue)
	if err != nil {
		return nil, err
	}

	amqpMsgs, err := q.Consume(consumer, opts)
	if err != nil {
		return nil, err
	}

	// Convert amqp.Delivery to Message
	msgChan := make(chan Message)
	go func() {
		for amqpMsg := range amqpMsgs {
			msgChan <- Message{
				Body:        amqpMsg.Body,
				Headers:     amqpMsg.Headers,
				RoutingKey:  amqpMsg.RoutingKey,
				ContentType: amqpMsg.ContentType,
				Redelivered: amqpMsg.Redelivered,
			}
		}
		close(msgChan)
	}()

	return msgChan, nil
}

// Publisher interface methods
func NewRabbitMQPublisher(exchange *ManagedExchange, defaultKey string, logger *logr.Logger) *RabbitMQPublisher {
	return &RabbitMQPublisher{
		exchange:    exchange,
		logger:      logger,
		defaultKey:  defaultKey,
	}
}

func (p *RabbitMQPublisher) Publish(message interface{}, routingKeys ...string) error {
	key := p.defaultKey
	if len(routingKeys) > 0 && routingKeys[0] != "" {
		key = routingKeys[0]
	}

	// Convert message to bytes (simplified - in real implementation, use JSON marshaling)
	var body []byte
	switch msg := message.(type) {
	case []byte:
		body = msg
	case string:
		body = []byte(msg)
	default:
		return fmt.Errorf("unsupported message type: %T", message)
	}

	msg := Message{
		Body:        body,
		ContentType: "application/json",
	}

	return p.exchange.Publish(key, msg)
}

func (p *RabbitMQPublisher) DelayedPublish(message interface{}, delay time.Duration, routingKeys ...string) error {
	// For delayed publishing, we'd need to use the delayed message plugin
	// For now, just publish normally
	return p.Publish(message, routingKeys...)
}

func (p *RabbitMQPublisher) Close() error {
	// Publisher doesn't own the connection, so nothing to close
	return nil
}

// NewConsumer interface methods
func NewRabbitMQConsumer(queue *ManagedQueue, exchange *ManagedExchange, bindingKeys []string, processor Processor, options WorkerOptions, logger *logr.Logger, ctx context.Context) (*RabbitMQConsumer, error) {
	childCtx, cancel := context.WithCancel(ctx)

	consumer := &RabbitMQConsumer{
		queue:       queue,
		exchange:    exchange,
		bindingKeys: bindingKeys,
		processor:   processor,
		options:     options,
		logger:      logger,
		ctx:         childCtx,
		cancel:      cancel,
		running:     false,
	}

	return consumer, nil
}

func (c *RabbitMQConsumer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("consumer is already running")
	}

	// Bind queue to exchange with all binding keys
	for _, key := range c.bindingKeys {
		if err := c.queue.Bind(c.exchange.Name(), key); err != nil {
			return fmt.Errorf("failed to bind queue '%s' to exchange '%s' with key '%s': %w", c.queue.Name(), c.exchange.Name(), key, err)
		}
	}

	// Start consuming messages
	amqpMsgs, err := c.queue.Consume(fmt.Sprintf("%s-consumer", c.queue.Name()), ConsumeOptions{
		AutoAck:   c.options.AutoAck,
		Exclusive: c.options.Exclusive,
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.messages = amqpMsgs
	c.running = true

	// Start processing messages
	go c.processMessages()

	c.logger.Info("Consumer started", "queue", c.queue.Name(), "bindingKeys", c.bindingKeys)
	return nil
}

func (c *RabbitMQConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.cancel()
	c.running = false

	c.logger.Info("Consumer stopped", "queue", c.queue.Name())
	return nil
}

func (c *RabbitMQConsumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *RabbitMQConsumer) processMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return

		case msg, ok := <-c.messages:
			if !ok {
				c.logger.Warn("Message channel closed")
				return
			}

			// Convert amqp.Delivery to Message
			message := Message{
				Body:        msg.Body,
				Headers:     msg.Headers,
				RoutingKey:  msg.RoutingKey,
				ContentType: msg.ContentType,
				Redelivered: msg.Redelivered,
			}

			// Process the message
			if err := c.processor.Process(c.ctx, message, msg.RoutingKey); err != nil {
				action := c.processor.OnError(message, err)
				if action.ShouldRetry {
					c.logger.Warn("Message processing failed, will retry", "error", err, "delay", action.Delay)
					// Retry logic would go here
				} else {
					c.logger.Error("Message processing failed permanently", "error", err)
				}
			}

			// Acknowledge message if not auto-ack
			if !c.options.AutoAck {
				if err := msg.Ack(false); err != nil {
					c.logger.Error("Failed to acknowledge message", "error", err)
				}
			}
		}
	}
}

// WorkerManager interface methods
func NewRabbitMQWorkerManager(manager *ConnectionManager, logger *logr.Logger, ctx context.Context) *RabbitMQWorkerManager {
	return &RabbitMQWorkerManager{
		manager: manager,
		logger:  logger,
		ctx:     ctx,
	}
}

func (wm *RabbitMQWorkerManager) CreateWorker(queue, exchange string, processor Processor, opts WorkerOptions) (NewConsumer, error) {
	// Get or create exchange
	exc, err := wm.manager.DeclareExchange(exchange, "topic", ExchangeOptions{
		Durable: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Get or create queue
	q, err := wm.manager.DeclareQueue(queue, opts.QueueOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Create consumer
	bindingKeys := opts.BindingKeys
	if len(bindingKeys) == 0 {
		bindingKeys = []string{"#"}
	}

	consumer, err := NewRabbitMQConsumer(q, exc, bindingKeys, processor, opts, wm.logger, wm.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return consumer, nil
}

func (wm *RabbitMQWorkerManager) CreatePublisher(exchange string, defaultKey string) (NewPublisher, error) {
	exc, err := wm.manager.GetExchange(exchange)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange '%s': %w", exchange, err)
	}

	return NewRabbitMQPublisher(exc, defaultKey, wm.logger), nil
}

func (wm *RabbitMQWorkerManager) Close() error {
	return wm.manager.Close()
}
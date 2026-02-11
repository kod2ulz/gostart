# Enhanced MQ Package

The enhanced MQ package provides a robust, broker-agnostic message queue system with intelligent connection management, automatic retry logic, and a unified API for handling different message patterns.

## Key Improvements Over Original

### 1. **Eliminated Service Crashes**
- **Before**: `panic()` calls throughout connection handling would crash the service on any RabbitMQ issue
- **After**: Graceful error handling with exponential backoff retry logic

### 2. **Simplified Handler API**
- **Before**: Separate `handlers.go` and `handlers_v2.go` files with nearly identical functionality
- **After**: Unified `UnifiedHandler` that supports both API function signatures

### 3. **Proper Connection Management**
- **Before**: Complex goroutine coordination with race conditions
- **After**: Clean connection manager with separate publisher/consumer connections and automatic recovery

### 4. **Broker Abstraction**
- **Before**: Direct RabbitMQ types throughout the codebase
- **After**: Clean interfaces that allow for different broker implementations (Kafka, Redis Streams, etc.)

## Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "log"

    "github.com/kod2ulz/gostart/mq"
    "github.com/kod2ulz/gostart/logr"
)

func main() {
    ctx := context.Background()
    logger := logr.New(logr.InfoLevel)

    // Create configuration
    config := mq.Config{
        Host:       "localhost",
        Port:       "5672",
        Username:   "guest",
        Password:   "guest",
        VHost:      "/",
        Heartbeat:  5 * time.Second,
        MaxRetries: 10,
    }

    // Create connection
    conn, err := mq.NewRabbitMQConnection(ctx, config, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // Create worker manager
    manager := mq.NewRabbitMQWorkerManager(conn.Manager(), logger, ctx)
    defer manager.Close()

    // Your MQ logic here...
}
```

### Publishing Messages

```go
// Create a publisher
publisher, err := manager.CreatePublisher("user.events", "user.created")
if err != nil {
    log.Fatal(err)
}

// Publish a simple message
err = publisher.Publish(map[string]interface{}{
    "id":    "123",
    "name":  "John Doe",
    "email": "john@example.com",
})

// Publish with specific routing key
err = publisher.Publish(userData, "user.updated")

// Delayed publishing
err = publisher.DelayedPublish(userData, 5*time.Minute, "user.reminder")
```

### Consuming Messages

#### Using the Unified Handler (Recommended)

```go
type UserEvent struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Define your business logic
func processUserEvent(ctx context.Context, event UserEvent) (string, error) {
    log.Printf("Processing user event: %s", event.Name)
    // Your business logic here
    return "success", nil
}

// Create unified handler
handlerConfig := mq.HandlerConfig[UserEvent, string]{
    Manager: manager,
    Theme:   "user-processor",
    Logger:  logger,
    Context: ctx,
}

handler := mq.NewUnifiedHandler(handlerConfig)

// Create worker
worker, err := handler.CreateWorker(
    "user.events",      // exchange
    "user.created",     // routing key
    processUserEvent,   // handler function
    mq.WithBindingKeys("user.created", "user.updated"),
    mq.WithPrefetchCount(10),
    mq.WithAutoAck(false),
)
if err != nil {
    log.Fatal(err)
}

// Start the worker
if err := worker.Start(); err != nil {
    log.Fatal(err)
}
defer worker.Stop()
```

#### Using Legacy Handler Functions

```go
// ApiFunc style (context only)
func legacyHandler(ctx context.Context) (string, error) {
    // Extract from context
    event := ctx.Value("event").(UserEvent)
    return processUserEvent(ctx, event)
}

// ApiFuncWithParam style (context + parameter)
func modernHandler(ctx context.Context, event UserEvent) (string, error) {
    return processUserEvent(ctx, event)
}

// Both work with the same unified handler
worker1, _ := handler.CreateWorker("user.events", "user.created", legacyHandler)
worker2, _ := handler.CreateWorker("user.events", "user.created", modernHandler)
```

## Advanced Features

### 1. Error Handling and Retry Logic

```go
// Custom error handler with retry logic
customErrorHandler := func(msg interface{}, err error) mq.RetryAction {
    if isTransientError(err) {
        return mq.RetryAction{
            ShouldRetry: true,
            Delay:       time.Second * 5,
        }
    }
    return mq.RetryAction{ShouldRetry: false}
}

handlerConfig := mq.HandlerConfig[UserEvent, string]{
    Manager:      manager,
    Theme:        "user-processor",
    Logger:       logger,
    Context:      ctx,
    ErrorHandler: customErrorHandler,
}

// Or use the built-in retryable error handler
retryableHandler := mq.RetryableErrorHandler[UserEvent](logger, "user-processor", 3)

handlerConfig := mq.HandlerConfig[UserEvent, string]{
    Manager:      manager,
    Theme:        "user-processor",
    Logger:       logger,
    Context:      ctx,
    ErrorHandler: retryableHandler,
}
```

### 2. Connection Management

```go
// Monitor connection health
stats := conn.Manager().GetStats()
log.Printf("Connection status: %+v", stats)

// Check if connected
if conn.IsConnected() {
    log.Println("RabbitMQ connection is healthy")
}

// Wait for connection to be ready
if err := conn.WaitForReady(ctx); err != nil {
    log.Fatal("Connection not ready:", err)
}
```

### 3. Queue and Exchange Management

```go
// Declare exchange with options
err := conn.DeclareExchange("notifications", "topic", mq.ExchangeOptions{
    Durable:    true,
    AutoDelete: false,
    Arguments: map[string]interface{}{
        "alternate-exchange": "notifications.dlq",
    },
})

// Declare queue with options
err := conn.DeclareQueue("user.notifications", mq.QueueOptions{
    Durable:    true,
    AutoDelete: false,
    Exclusive:  false,
    Arguments: map[string]interface{}{
        "x-message-ttl": 86400000, // 24 hours
    },
})

// Bind queue to exchange
err := conn.BindQueue("user.notifications", "notifications", "user.*")
```

## Migration from Original MQ Package

### Old API vs New API

#### Publishing
```go
// Old API
userPublisher, err := mq.InitPublisher(logger, userExchange, "users.event.created")
err = userPublisher.Publish(myUserObject)

// New API
publisher, err := manager.CreatePublisher("users.topic", "users.event.created")
err = publisher.Publish(myUserObject)
```

#### Consuming
```go
// Old API
worker, err := mq.InitWorkerStrict(ctx, logger, exchange, queue, routingKey, errorFunc, processorFunc)

// New API
handler := mq.NewUnifiedHandler(config)
worker, err := handler.CreateWorker(exchange, routingKey, processorFunc, options...)
```

#### Handler Functions
```go
// Old API - two separate files
type ApiFunc[R any] func(context.Context) (R, ierrors.Error)
type ApiFuncV2[P contracts.RequestParam, R any] func(context.Context, P) (R, ierrors.Error)

// New API - unified with both signatures supported
type ApiFunc[P contracts.RequestParam, R any] func(context.Context) (R, ierrors.Error)
type ApiFuncWithParam[P contracts.RequestParam, R any] func(context.Context, P) (R, ierrors.Error)
```

## Configuration Options

### Connection Configuration
```go
config := mq.Config{
    Host:            "localhost",
    Port:            "5672",
    VHost:           "/",
    Username:        "guest",
    Password:        "guest",
    Protocol:        "amqp",
    Heartbeat:       10 * time.Second,
    Timeout:         30 * time.Second,
    MaxRetries:      10,
    RetryDelay:      1 * time.Second,
    BackoffMultiplier: 2.0,
    AdditionalArgs: map[string]interface{}{
        "client_properties": map[string]interface{}{
            "connection_name": "my-service",
        },
    },
}
```

### Worker Options
```go
worker, err := handler.CreateWorker(exchange, routingKey, handlerFunc,
    mq.WithQueueOptions(mq.QueueOptions{
        Durable:    true,
        AutoDelete: false,
        Exclusive:  false,
    }),
    mq.WithBindingKeys("event.created", "event.updated"),
    mq.WithConsumerTag("my-worker"),
    mq.WithPrefetchCount(10),
    mq.WithAutoAck(false),
    mq.WithExclusive(false),
    mq.WithErrorHandler(customErrorHandler),
)
```

## Best Practices

### 1. Connection Management
- Use a single connection manager per service
- Initialize connections at startup
- Gracefully close connections on shutdown
- Monitor connection health

### 2. Error Handling
- Always implement proper error handling
- Use retry logic for transient failures
- Log errors appropriately
- Implement dead-letter queues for failed messages

### 3. Performance
- Use appropriate prefetch counts
- Consider message durability requirements
- Use separate connections for publishing and consuming
- Monitor queue lengths and processing times

### 4. Message Design
- Keep messages small and focused
- Use appropriate content types
- Include correlation IDs for tracing
- Consider message expiration

## Troubleshooting

### Common Issues

1. **Connection Failures**
   - Check RabbitMQ server status
   - Verify network connectivity
   - Review authentication credentials
   - Check firewall settings

2. **Message Processing Errors**
   - Verify message schema compatibility
   - Check handler function signatures
   - Review error handling logic
   - Monitor queue growth

3. **Performance Issues**
   - Adjust prefetch counts
   - Consider message batching
   - Monitor resource usage
   - Review queue configuration

### Debug Mode
```go
// Enable debug logging
debugLogger := logr.New(logr.DebugLevel)
config := mq.Config{
    // ... other config
}

// Create connection with debug logging
conn, err := mq.NewRabbitMQConnection(ctx, config, debugLogger)
```

## Implementation Status

### ✅ **Fully Implemented Features**
- **Framework-Agnostic Message Queue Interfaces** - Clean abstraction layer for different brokers
- **Complete RabbitMQ Implementation** - Full producer/consumer pattern with connection management
- **Publisher/Consumer Pattern** - Robust message publishing and consumption with error handling
- **Exchange and Queue Management** - Automatic declaration and binding with configurable options
- **Retry Logic and Error Handling** - Exponential backoff retry with customizable error handling
- **Connection Pooling** - Efficient connection management with health monitoring
- **Worker Management** - Background worker processes with graceful shutdown
- **Unified Handler API** - Single handler interface supporting multiple function signatures
- **Message Acknowledgment** - Proper ack/nack handling with manual and automatic modes
- **Connection Recovery** - Automatic reconnection on connection failures

### ⚠️ **Needs Verification/Enhancement**
- **Delayed Message Publishing** - Basic structure exists, implementation needs verification
- **Message Prioritization** - Some prioritization logic exists, may need enhancement
- **Performance Optimization** - Current implementation works, but performance tuning may be needed
- **Advanced Queue Configuration** - Basic queue management exists, advanced options may be incomplete

### 🚧 **Planned Enhancements**
- **Kafka Support** - Implementation for Apache Kafka broker
- **Redis Streams** - Redis Streams broker implementation
- **Message Schema Validation** - JSON Schema validation for messages
- **Metrics and Monitoring** - Prometheus metrics integration
- **Distributed Tracing** - OpenTelemetry integration
- **Message Encryption** - End-to-end encryption support

## Production Examples

### Order Processing Service
```go
type OrderRequest struct {
    OrderID    string  `json:"orderId"`
    CustomerID string  `json:"customerId"`
    Amount     float64 `json:"amount"`
    Items      []Item  `json:"items"`
}

func processOrder(ctx context.Context, req OrderRequest) (string, error) {
    // Order processing logic
    return "order.processed", nil
}

func main() {
    // Setup connection and manager

    handlerConfig := mq.HandlerConfig[OrderRequest, string]{
        Manager: manager,
        Theme:   "order-processor",
        Logger:  logger,
        Context: ctx,
        ErrorHandler: mq.RetryableErrorHandler[OrderRequest](logger, "order", 5),
    }

    handler := mq.NewUnifiedHandler(handlerConfig)

    worker, _ := handler.CreateWorker(
        "orders",
        "order.request",
        processOrder,
        mq.WithBindingKeys("order.*"),
        mq.WithPrefetchCount(5), // Lower for order processing
    )

    worker.Start()
}
```

### Notification Service
```go
type NotificationEvent struct {
    UserID    string   `json:"userId"`
    Type      string   `json:"type"`
    Message   string   `json:"message"`
    Channels  []string `json:"channels"`
}

func sendNotification(ctx context.Context, event NotificationEvent) (string, error) {
    // Send email, SMS, push notification
    return "notification.sent", nil
}

func main() {
    // Setup connection and manager

    handlerConfig := mq.HandlerConfig[NotificationEvent, string]{
        Manager: manager,
        Theme:   "notification-service",
        Logger:  logger,
        Context: ctx,
    }

    handler := mq.NewUnifiedHandler(handlerConfig)

    worker, _ := handler.CreateWorker(
        "notifications",
        "notification.send",
        sendNotification,
        mq.WithBindingKeys("notification.*"),
        mq.WithPrefetchCount(20), // Higher for notifications
    )

    worker.Start()
}
```
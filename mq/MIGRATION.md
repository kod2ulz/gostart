# MQ Package Migration Guide

This guide helps you migrate from the original MQ package to the enhanced version with improved reliability and unified API.

## Key Changes

### 1. **No More Service Crashes**
The original package used `panic()` calls throughout the connection handling, which would crash your service on any RabbitMQ connection issue. The new version implements graceful error handling with automatic retry logic.

### 2. **Unified Handler API**
The separate `handlers.go` and `handlers_v2.go` files have been unified into a single `handlers.go` with a flexible API that supports both function signatures.

### 3. **Improved Connection Management**
Complex goroutine coordination has been replaced with a clean connection manager that handles both publisher and consumer connections with automatic recovery.

### 4. **Better Abstraction**
The new version provides clean interfaces that make it easier to test and support different message brokers in the future.

## Migration Steps

### Step 1: Update Imports

Most imports remain the same, but you may need to add a few new ones:

```go
// Old imports (no changes needed)
import "github.com/kod2ulz/gostart/mq"

// Additional imports for new features
import "time"
```

### Step 2: Update Connection Setup

#### Old API
```go
rmqHandler := mq.RabbitMQ(ctx, logger, conf)
userExchange := rmqHandler.TopicExchange("users.topic")
userEventsQueue := rmqHandler.Queue("user-events")
```

#### New API
```go
// Create configuration
config := mq.Config{
    Host:       conf.Host,
    Port:       conf.Port,
    Username:   conf.Username,
    Password:   conf.Password,
    VHost:      conf.VHost,
    Heartbeat:  conf.Heartbeat,
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
```

### Step 3: Update Publishing Code

#### Old API
```go
userPublisher, err := mq.InitPublisher(logger, userExchange, "users.event.created")
if err != nil {
    log.Fatal(err)
}

// Publish messages
err = userPublisher.Publish(myUserObject)
err = userPublisher.Publish(myUserObject, "users.event.updated")
err = userPublisher.DelayedPublish(myUserObject, 5*time.Second, "users.event.reminder")
```

#### New API
```go
// Create publisher
publisher, err := manager.CreatePublisher("users.topic", "users.event.created")
if err != nil {
    log.Fatal(err)
}

// Publish messages (same API)
err = publisher.Publish(myUserObject)
err = publisher.Publish(myUserObject, "users.event.updated")
err = publisher.DelayedPublish(myUserObject, 5*time.Second, "users.event.reminder")
```

### Step 4: Update Worker Creation

#### Old API (handlers.go style)
```go
type UserEvent struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

processorFunc := func(msg *UserEvent, routingKey string, redelivered bool) (any, error) {
    log.Printf("Processing user event for user: %s", msg.Name)
    // ... do work ...
    return nil, nil
}

errorFunc := func(msg *UserEvent, err error) (retry bool, delay time.Duration) {
    log.Errorf("Failed to process user event: %v", err)
    return false, 0
}

worker, err := mq.InitWorkerStrict(ctx, logger, userExchange, userEventsQueue.Name(), "users.event.*", errorFunc, processorFunc)
```

#### Old API (handlers_v2.go style)
```go
apiFunc := func(ctx context.Context, msg UserEvent) (string, error) {
    log.Printf("Processing user event for user: %s", msg.Name)
    // ... do work ...
    return "success", nil
}

worker, err := mq.GenericWorkerHandlerV2[UserEvent, string](
    manager, "user-processor", "users.event.*", apiFunc,
)
```

#### New API (Unified)
```go
type UserEvent struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// Your business logic function (either style works)
func processUserEventV1(ctx context.Context) (string, error) {
    // Extract from context
    event := ctx.Value("event").(UserEvent)
    log.Printf("Processing user event for user: %s", event.Name)
    return "success", nil
}

func processUserEventV2(ctx context.Context, event UserEvent) (string, error) {
    log.Printf("Processing user event for user: %s", event.Name)
    return "success", nil
}

// Create unified handler
handlerConfig := mq.HandlerConfig[UserEvent, string]{
    Manager: manager,
    Theme:   "user-processor",
    Logger:  logger,
    Context: ctx,
    // Optional: custom error handler
    ErrorHandler: mq.RetryableErrorHandler[UserEvent](logger, "user-processor", 3),
}

handler := mq.NewUnifiedHandler(handlerConfig)

// Create worker (supports both function styles)
worker1, err := handler.CreateWorker("users.topic", "users.event.*", processUserEventV1)
worker2, err := handler.CreateWorker("users.topic", "users.event.*", processUserEventV2)

// Or with additional options
worker, err := handler.CreateWorker("users.topic", "users.event.*", processUserEventV2,
    mq.WithBindingKeys("users.event.created", "users.event.updated"),
    mq.WithPrefetchCount(10),
    mq.WithAutoAck(false),
    mq.WithQueueOptions(mq.QueueOptions{
        Durable:    true,
        AutoDelete: false,
    }),
)

// Start the worker
if err := worker.Start(); err != nil {
    log.Fatal(err)
}
defer worker.Stop()
```

### Step 5: Update Error Handling

#### Old API
```go
errorFunc := func(msg *UserEvent, err error) (retry bool, delay time.Duration) {
    log.Errorf("Failed to process user event: %v", err)
    if isTransient(err) {
        return true, 5 * time.Second
    }
    return false, 0
}
```

#### New API
```go
customErrorHandler := func(msg interface{}, err error) mq.RetryAction {
    event := msg.(UserEvent)
    logger.Errorf("Failed to process user event for %s: %v", event.Name, err)

    if isTransient(err) {
        return mq.RetryAction{
            ShouldRetry: true,
            Delay:       5 * time.Second,
        }
    }
    return mq.RetryAction{ShouldRetry: false}
}

// Or use built-in error handlers
handlerConfig := mq.HandlerConfig[UserEvent, string]{
    Manager:      manager,
    Theme:        "user-processor",
    Logger:       logger,
    Context:      ctx,
    ErrorHandler: mq.RetryableErrorHandler[UserEvent](logger, "user-processor", 3),
}
```

### Step 6: Remove panic() Dependencies

The old API relied on panic() calls and Docker restarts for connection recovery. The new API handles this automatically:

#### Old Approach (no longer needed)
```go
// This code would panic and rely on Docker to restart the service
func (c *rmqConn) handleChanError(err *amqp.Error) error {
    // ... error handling logic ...
    panic(err) // 😱
}
```

#### New Approach (automatic)
```go
// No panic() calls - automatic retry with exponential backoff
// Connections automatically recover from transient failures
```

### Step 7: Update Queue and Exchange Management

#### Old API
```go
// Direct access to exchanges and queues
exchange := rmqHandler.TopicExchange("users.topic")
queue := rmqHandler.Queue("user-events")
```

#### New API
```go
// Managed through connection manager
err := conn.DeclareExchange("users.topic", "topic", mq.ExchangeOptions{
    Durable: true,
})

err = conn.DeclareQueue("user-events", mq.QueueOptions{
    Durable: true,
})

err = conn.BindQueue("user-events", "users.topic", "users.event.*")
```

## Configuration Migration

### Old Configuration
```go
type Conf struct {
    Host             string
    Port             string
    Vhost            string
    Heartbeat        time.Duration
    HeartbeatTimeout time.Duration
    Username         string
    Password         string
    Protocol         string
    ConsumerExchange ExchangeConfig
    ProducerExchange ExchangeConfig
}
```

### New Configuration
```go
type Config struct {
    Host            string
    Port            string
    VHost           string
    Username        string
    Password        string
    Protocol        string
    Heartbeat       time.Duration
    Timeout         time.Duration
    MaxRetries      int
    RetryDelay      time.Duration
    BackoffMultiplier float64
    AdditionalArgs   map[string]interface{}
}
```

### Configuration Mapping
```go
// Old to new configuration mapping
newConfig := mq.Config{
    Host:       oldConfig.Host,
    Port:       oldConfig.Port,
    VHost:      oldConfig.Vhost,
    Username:   oldConfig.Username,
    Password:   oldConfig.Password,
    Protocol:   oldConfig.Protocol,
    Heartbeat:  oldConfig.Heartbeat,
    Timeout:    oldConfig.HeartbeatTimeout,
    MaxRetries: 10, // New field
    RetryDelay: 1 * time.Second, // New field
    BackoffMultiplier: 2.0, // New field
}
```

## Benefits of Migration

### 1. **No More Service Crashes**
- Automatic connection recovery
- Graceful error handling
- No reliance on external orchestrators for restarts

### 2. **Unified API**
- Single handler system for all use cases
- Flexible function signature support
- Consistent error handling patterns

### 3. **Better Testing**
- Interface-based design
- Mockable components
- Easier unit testing

### 4. **Future-Proof**
- Broker-agnostic interfaces
- Ready for Kafka, Redis Streams support
- Clean architecture for extensions

### 5. **Improved Monitoring**
- Connection health monitoring
- Statistics and metrics
- Better logging and debugging

## Testing Your Migration

### 1. **Unit Tests**
```go
func TestWorkerProcessing(t *testing.T) {
    // Mock the processor
    processor := &MockProcessor{}

    // Create test handler
    handlerConfig := mq.HandlerConfig[TestMessage, string]{
        Manager: mockManager,
        Theme:   "test-worker",
        Logger:  testLogger,
        Context: testCtx,
    }

    handler := mq.NewUnifiedHandler(handlerConfig)

    // Test worker creation and processing
    worker, err := handler.CreateWorker("test.exchange", "test.key", processor.Process)
    assert.NoError(t, err)

    // Start worker and test message processing
}
```

### 2. **Integration Tests**
```go
func TestConnectionRecovery(t *testing.T) {
    // Test connection recovery
    conn, err := mq.NewRabbitMQConnection(testCtx, testConfig, testLogger)
    assert.NoError(t, err)

    // Simulate connection failure
    // Verify automatic reconnection
    // Test message processing after recovery
}
```

### 3. **Load Testing**
```go
func TestHighThroughput(t *testing.T) {
    // Test with high message volumes
    // Verify no message loss
    // Test error handling under load
    // Monitor resource usage
}
```

## Rollback Plan

If you encounter issues during migration, you can temporarily rollback by:

1. **Revert to Old Files**
   ```bash
   git checkout handlers.go handlers_v2.go connection.go
   ```

2. **Use Feature Flag**
   ```go
   if useNewMQ {
       // Use new API
   } else {
       // Use old API
   }
   ```

3. **Parallel Deployment**
   - Deploy new version alongside old version
   - Route a subset of traffic to new version
   - Monitor and compare performance

## Common Migration Issues

### 1. **Import Errors**
- Ensure all imports are correct
- Check for missing dependencies
- Verify Go module versions

### 2. **Type Mismatches**
- Update function signatures
- Check interface implementations
- Verify type assertions

### 3. **Connection Issues**
- Verify configuration parameters
- Check network connectivity
- Review authentication credentials

### 4. **Message Processing Errors**
- Update error handling logic
- Verify message serialization
- Check queue binding configuration

## Support

If you encounter issues during migration:
1. Check this migration guide
2. Review the comprehensive README_NEW.md
3. Look at the provided examples
4. Check test files for reference implementations

## Conclusion

The enhanced MQ package provides significant improvements in reliability, maintainability, and flexibility. While the migration requires some code changes, the benefits far outweigh the costs, especially for production environments where service stability is critical.
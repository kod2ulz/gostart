# MQ Package

This package provides a high-level abstraction for interacting with a RabbitMQ message broker. It simplifies connection management, publishing, and consuming messages.

## Overview

The `mq` package is designed around a central `RMQ` object that manages connections for publishing and consuming. It provides abstractions for creating exchanges, queues, publishers, and workers.

## Initialization

First, initialize the main `RMQ` object with a context, logger, and configuration.

```go
import "github.com/kod2ulz/gostart/mq"

// Assuming conf, logger, and ctx are already defined
rmqHandler := mq.RabbitMQ(ctx, logger, conf)
```

## Exchanges and Queues

You can declare exchanges and queues directly from the `RMQ` handler.

```go
// Declare a topic exchange
userExchange := rmqHandler.TopicExchange("users.topic")

// Declare a queue
userEventsQueue := rmqHandler.Queue("user-events")
```

## Publishing Messages

To publish messages, create a `Publisher` for a specific exchange.

```go
userPublisher, err := mq.InitPublisher(logger, userExchange, "users.event.created")
if err != nil {
    // handle error
}

// Publish a message with the default routing key
err = userPublisher.Publish(myUserObject)

// Publish with a specific routing key
err = userPublisher.Publish(myUserObject, "users.event.updated")

// Publish with a delay
err = userPublisher.DelayedPublish(myUserObject, 5*time.Second, "users.event.reminder")
```

## Consuming Messages (Workers)

To consume messages, create a `Worker`. A worker binds to a queue on an exchange and processes incoming messages.

You must provide a `processorFunc` to handle the message logic and an `errorFunc` to decide what to do on failure.

```go
type UserEvent struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// processorFunc defines what to do with a message
processorFunc := func(msg *UserEvent, routingKey string, redelivered bool) (any, error) {
    log.Printf("Processing user event for user: %s", msg.Name)
    // ... do work ...
    return nil, nil
}

// errorFunc defines what to do when the processor returns an error
errorFunc := func(msg *UserEvent, err error) (retry bool, delay time.Duration) {
    log.Errorf("Failed to process user event: %v", err)
    return false, 0 // Do not retry
}

// Initialize the worker
worker, err := mq.InitWorkerStrict(ctx, logger, userExchange, userEventsQueue.Name(), "users.event.*", errorFunc, processorFunc)
if err != nil {
    // handle error
}

// The worker will now process messages in the background.
// To stop it:
// worker.Stop()
```

## Roadmap

- **Broker-agnostic Interface:** Refine the interfaces to allow for different underlying message brokers.
- **Kafka Support:** Add a Kafka implementation for the `mq` interfaces.
- **Redis Streams Support:** Add a Redis Streams implementation.

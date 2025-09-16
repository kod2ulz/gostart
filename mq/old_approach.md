# MQ Package Refactoring Approach

This document outlines the strategy for refactoring the `mq` package to improve its resilience, testability, and reliability.

## 1. Problems with the Previous Implementation

The original implementation of the `mq` package had several architectural flaws that made it brittle in a production environment:

1.  **Aggressive Crash-on-Failure:** The connection logic was littered with `log.Fatal` and `panic` calls. Any transient disconnection from RabbitMQ would immediately crash the entire service.
2.  **No Exponential Backoff:** The old code would hammer a temporarily unavailable server with connection attempts at a fixed, rapid interval.
3.  **No Startup Timeout:** The constructor would block indefinitely if it couldn't establish an initial connection.
4.  **Poor Testability:** The code directly used concrete types from the `amqp` library, making unit testing nearly impossible.

## 2. The New Refactored Approach

Our new approach addresses each of these problems by introducing a more robust, non-blocking, and testable architecture.

### 2.1. Decoupled Control Loop & Graceful Reconnection

The core of the new design is a `controlLoop()` in a dedicated goroutine for each connection (publisher, consumer). This loop manages the connection lifecycle, including automated reconnection with exponential backoff, preventing the service from crashing or overwhelming the server on transient failures.

### 2.2. Fail-Fast Startup

The `RabbitMQ()` constructor now has a configurable timeout to prevent the application from hanging on startup, returning an error if a connection cannot be established in a timely manner.

### 2.3. Testability via Interfaces

We have introduced `AmqpConnection` and `AmqpChannel` interfaces to abstract away the concrete `amqp` library types. This enables mocking and robust unit testing of the connection logic.

## 3. Implementation Learnings & Challenges

- **Consumer Resubscription:** The logic was updated to track all active consumers. When a connection is re-established, the `controlLoop` now automatically re-subscribes them, ensuring no messages are missed after a recovery.

- **Reconnection Deadlock:** A deadlock was discovered during the reconnection process. The `restoreBindings` function was trying to acquire a channel that was not yet marked as ready, because it was waiting for `restoreBindings` to complete first. This was resolved by creating internal, non-blocking helper methods that are passed the newly-created channel directly during the connection phase.

- **Startup Race Condition:** A hang during testing revealed a race condition in the initial startup logic. The code designed to wait for both publisher and consumer connections could return prematurely, leading to a deadlock on subsequent operations. This needs to be fixed by making the check more robust.

- **Testing Strategy:** The initial testing approach using `testcontainers-go` was abandoned at the user's request. The current, agreed-upon strategy is to use a long-running, named Docker container (`gostart_rabbitmq`). Tests first check for connectivity and the container's health status before running, and skip themselves gracefully if the dependency is not met.

## 4. Next Steps

1.  **Fix Startup Race Condition:** Apply a fix to the `RabbitMQ()` constructor to ensure it correctly waits for both publisher and consumer connections to be ready before returning.
2.  **Verify with Tests:** Run the full test suite to confirm the fix resolves the hang and that the resubscription logic works as expected.
3.  **Contingency:** If the fix is unsuccessful, revert all logic changes in the `mq` package to unblock other development, and move on to other packages.
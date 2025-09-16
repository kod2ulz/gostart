package mq

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RetryableConnection implements a connection with automatic retry and backoff
type RetryableConnection struct {
	config     *Conf
	logger     *logr.Logger
	ctx        context.Context

	connection *amqp.Connection
	channel    *amqp.Channel
	mu         sync.RWMutex

	errorChan   chan *amqp.Error
	readyChan   chan struct{}
	connected  bool

	maxRetries      int
	baseDelay      time.Duration
	maxDelay       time.Duration
	backoffFactor  float64
}

// NewRetryableConnection creates a new connection with retry logic
func NewRetryableConnection(ctx context.Context, config *Conf, logger *logr.Logger) *RetryableConnection {
	return &RetryableConnection{
		config:        config,
		logger:        logger,
		ctx:           ctx,
		errorChan:     make(chan *amqp.Error, 1),
		readyChan:     make(chan struct{}),
		maxRetries:    10,
		baseDelay:     1 * time.Second,
		maxDelay:      30 * time.Second,
		backoffFactor: 2.0,
	}
}

// Connect establishes a connection with retry logic
func (rc *RetryableConnection) Connect() error {
	var lastErr error

	for attempt := 0; attempt < rc.maxRetries; attempt++ {
		if attempt > 0 {
			delay := rc.calculateDelay(attempt)
			rc.logger.Warn("Connection attempt failed, retrying", "attempt", attempt+1, "maxAttempts", rc.maxRetries, "delay", delay, "error", lastErr)

			select {
			case <-time.After(delay):
			case <-rc.ctx.Done():
				return fmt.Errorf("connection cancelled during retry: %w", rc.ctx.Err())
			}
		}

		if err := rc.tryConnect(); err != nil {
			lastErr = err
			continue
		}

		rc.logger.Info("Connection established successfully", "attempt", attempt+1)
		return nil
	}

	return fmt.Errorf("failed to connect after %d attempts, last error: %w", rc.maxRetries, lastErr)
}

// tryConnect attempts to establish a connection once
func (rc *RetryableConnection) tryConnect() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Close existing connection if any
	if rc.connection != nil {
		rc.connection.Close()
	}

	// Try to establish connection
	conn, err := amqp.Dial(rc.config.ConnectionString())
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	// Try to create channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// Set up error notification
	errChan := conn.NotifyClose(make(chan *amqp.Error, 1))

	// Update connection state
	rc.connection = conn
	rc.channel = channel
	rc.errorChan = errChan
	rc.connected = true

	// Signal that we're ready
	select {
	case <-rc.readyChan:
		// Already closed, create new one
		rc.readyChan = make(chan struct{})
	default:
	}
	close(rc.readyChan)

	// Start monitoring for connection issues
	go rc.monitorConnection()

	return nil
}

// monitorConnection watches for connection errors and handles reconnection
func (rc *RetryableConnection) monitorConnection() {
	for {
		select {
		case err, ok := <-rc.errorChan:
			if !ok {
				// Channel closed, exiting
				return
			}

			rc.logger.Error("Connection error detected", "error", err, "code", err.Code, "reason", err.Reason)
			rc.handleConnectionError(err)

		case <-rc.ctx.Done():
			rc.logger.Info("Context cancelled, stopping connection monitor")
			return
		}
	}
}

// handleConnectionError determines if we should reconnect based on the error
func (rc *RetryableConnection) handleConnectionError(err *amqp.Error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.connected = false

	// Determine if this is a recoverable error
	if rc.isRecoverableError(err) {
		rc.logger.Warn("Recoverable connection error, attempting reconnect", "error", err)
		go func() {
			if connectErr := rc.Connect(); connectErr != nil {
				rc.logger.Error("Failed to reconnect", "error", connectErr)
			}
		}()
	} else {
		rc.logger.Error("Non-recoverable connection error", "error", err)
		// For non-recoverable errors, we'll wait for external intervention
	}
}

// isRecoverableError determines if a connection error is recoverable
func (rc *RetryableConnection) isRecoverableError(err *amqp.Error) bool {
	// These error codes typically indicate temporary issues
	recoverableCodes := []int{
		320, // ConnectionForced
		501, // FrameError
		502, // SyntaxError
		503, // CommandInvalid
		504, // ChannelError
		505, // UnexpectedFrame
		506, // ResourceError
		530, // NotAllowed
		540, // NotImplemented
		541, // InternalError
	}

	for _, code := range recoverableCodes {
		if int(err.Code) == code {
			return true
		}
	}

	return false
}

// calculateDelay implements exponential backoff with jitter
func (rc *RetryableConnection) calculateDelay(attempt int) time.Duration {
	delay := float64(rc.baseDelay) * math.Pow(rc.backoffFactor, float64(attempt))

	// Add jitter to prevent thundering herd
	if delay > 0 {
		jitter := delay * 0.1 * (2*rand.Float64() - 1) // ±10% jitter
		delay += jitter
	}

	// Cap at maximum delay
	if delay > float64(rc.maxDelay) {
		delay = float64(rc.maxDelay)
	}

	return time.Duration(delay)
}

// Channel returns a channel, waiting if necessary
func (rc *RetryableConnection) Channel() (*amqp.Channel, error) {
	rc.mu.RLock()

	if !rc.connected || rc.channel == nil {
		rc.mu.RUnlock()
		return nil, errors.New("connection not established")
	}

	channel := rc.channel
	rc.mu.RUnlock()

	return channel, nil
}

// IsConnected returns whether the connection is currently active
func (rc *RetryableConnection) IsConnected() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.connected && rc.channel != nil
}

// Close gracefully shuts down the connection
func (rc *RetryableConnection) Close() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.connected = false

	if rc.channel != nil {
		if err := rc.channel.Close(); err != nil {
			rc.logger.Error("Error closing channel", "error", err)
		}
		rc.channel = nil
	}

	if rc.connection != nil {
		if err := rc.connection.Close(); err != nil {
			rc.logger.Error("Error closing connection", "error", err)
		}
		rc.connection = nil
	}

	close(rc.errorChan)
	close(rc.readyChan)

	return nil
}

// WaitForReady waits until the connection is ready or context is cancelled
func (rc *RetryableConnection) WaitForReady(ctx context.Context) error {
	select {
	case <-rc.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return errors.New("timeout waiting for connection to be ready")
	}
}

// GetStats returns connection statistics
func (rc *RetryableConnection) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return map[string]interface{}{
		"connected": rc.connected,
		"has_channel": rc.channel != nil,
		"config":     rc.config.String(),
	}
}
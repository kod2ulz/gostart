package mq

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/logr"
)

// UnifiedHandler provides a single, flexible API for handling MQ messages
type UnifiedHandler struct {
	manager NewWorkerManager
	theme   string
	logger  *logr.Logger
	ctx     context.Context
}

// HandlerConfig defines handler configuration
type HandlerConfig struct {
	Manager      NewWorkerManager
	Theme        string
	Logger       *logr.Logger
	Context      context.Context
	ErrorHandler ErrorProcessor
	RetryPolicy  RetryPolicy
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// HandlerFunction represents the handler function signature
type HandlerFunction func(context.Context, interface{}) (interface{}, ierrors.Error)

// UnifiedProcessor adapts handler functions to the Processor interface
type UnifiedProcessor struct {
	handler      interface{}
	logger       *logr.Logger
	operation    string
	errorHandler ErrorProcessor
}

// NewUnifiedHandler creates a new unified handler
func NewUnifiedHandler(config HandlerConfig) *UnifiedHandler {
	if config.ErrorHandler == nil {
		config.ErrorHandler = DefaultErrorHandler(config.Logger, config.Theme)
	}

	return &UnifiedHandler{
		manager: config.Manager,
		theme:   config.Theme,
		logger:  config.Logger,
		ctx:     config.Context,
	}
}

// CreateWorker creates a worker with the given handler function
func (h *UnifiedHandler) CreateWorker(
	exchange, routingKey string,
	handlerFunc interface{},
	opts ...WorkerOption,
) (NewConsumer, error) {
	processor := h.createProcessor(handlerFunc)
	workerOpts := h.buildWorkerOptions(routingKey, opts...)

	queueName := fmt.Sprintf("%s-%s-%s", exchange, h.theme, strings.ReplaceAll(routingKey, ".", "-"))

	worker, err := h.manager.CreateWorker(queueName, exchange, processor, workerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker: %w", err)
	}

	return worker, nil
}

// CreatePublisher creates a publisher for the exchange
func (h *UnifiedHandler) CreatePublisher(exchange, defaultRoutingKey string) (NewPublisher, error) {
	return h.manager.CreatePublisher(exchange, defaultRoutingKey)
}

// WorkerOption configures worker behavior
type WorkerOption func(*WorkerOptions)

// WithQueueOptions sets queue creation options
func WithQueueOptions(opts QueueOptions) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.QueueOptions = opts
	}
}

// WithBindingKeys sets the binding keys
func WithBindingKeys(keys ...string) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.BindingKeys = keys
	}
}

// WithConsumerTag sets the consumer tag
func WithConsumerTag(tag string) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.ConsumerTag = tag
	}
}

// WithPrefetchCount sets the prefetch count
func WithPrefetchCount(count int) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.PrefetchCount = count
	}
}

// WithErrorHandler sets a custom error handler
func WithErrorHandler(handler ErrorProcessor) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.ErrorHandler = handler
	}
}

// WithAutoAck enables/disables auto-ack
func WithAutoAck(autoAck bool) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.AutoAck = autoAck
	}
}

// WithExclusive enables/disables exclusive consumer
func WithExclusive(exclusive bool) WorkerOption {
	return func(wo *WorkerOptions) {
		wo.Exclusive = exclusive
	}
}

// createProcessor creates the processor for the handler function
func (h *UnifiedHandler) createProcessor(handlerFunc interface{}) Processor {
	return &UnifiedProcessor{
		handler:      handlerFunc,
		logger:       h.logger,
		operation:    h.theme,
		errorHandler: DefaultErrorHandler(h.logger, h.theme),
	}
}

// buildWorkerOptions creates worker options from configuration
func (h *UnifiedHandler) buildWorkerOptions(routingKey string, opts ...WorkerOption) WorkerOptions {
	workerOpts := WorkerOptions{
		BindingKeys: []string{routingKey},
		QueueOptions: QueueOptions{
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
		},
		PrefetchCount: 1,
		AutoAck:       false,
	}

	for _, opt := range opts {
		opt(&workerOpts)
	}

	return workerOpts
}

// Process implements the Processor interface for UnifiedProcessor
func (p *UnifiedProcessor) Process(ctx context.Context, msg interface{}, routingKey string) error {
	// Use reflection to call the handler function with the appropriate signature
	return p.callHandler(ctx, msg, routingKey)
}

// OnError implements the Processor interface for UnifiedProcessor
func (p *UnifiedProcessor) OnError(msg interface{}, err error) RetryAction {
	if p.errorHandler != nil {
		return p.errorHandler(msg, err)
	}
	return DefaultErrorHandler(p.logger, p.operation)(msg, err)
}

// callHandler uses reflection to call the appropriate handler function signature
func (p *UnifiedProcessor) callHandler(ctx context.Context, msg interface{}, routingKey string) error {
	p.logger.Debug("Processing message", "type", fmt.Sprintf("%T", msg), "routingKey", routingKey)

	// Try to determine the handler function type and call accordingly
	handlerValue := reflect.ValueOf(p.handler)
	if handlerValue.Kind() != reflect.Func {
		return fmt.Errorf("handler is not a function")
	}

	handlerType := handlerValue.Type()

	// Support both function signatures:
	// 1. func(context.Context, T) (R, error)
	// 2. func(context.Context) (R, error) - where T is extracted from context

	if handlerType.NumIn() == 2 {
		// func(context.Context, T) (R, error)
		argType := handlerType.In(1)
		msgValue := reflect.ValueOf(msg)
		if !msgValue.Type().AssignableTo(argType) {
			return fmt.Errorf("message type %T is not assignable to handler parameter type %v", msg, argType)
		}

		args := []reflect.Value{reflect.ValueOf(ctx), msgValue}
		results := handlerValue.Call(args)

		if len(results) == 2 && results[1].Interface() != nil {
			return results[1].Interface().(error)
		}
		return nil
	} else if handlerType.NumIn() == 1 {
		// func(context.Context) (R, error) - extract from context
		if msgWithKey, ok := msg.(interface{ ContextKey() interface{} }); ok {
			ctxWithValue := context.WithValue(ctx, msgWithKey.ContextKey(), msg)
			args := []reflect.Value{reflect.ValueOf(ctxWithValue)}
			results := handlerValue.Call(args)

			if len(results) == 2 && results[1].Interface() != nil {
				return results[1].Interface().(error)
			}
			return nil
		}
		return fmt.Errorf("message does not implement ContextKey() method for context-based handler")
	}

	return fmt.Errorf("unsupported handler function signature: expected 1 or 2 parameters, got %d", handlerType.NumIn())
}

// DefaultErrorHandler provides sensible default error handling
func DefaultErrorHandler(logger *logr.Logger, operation string) ErrorProcessor {
	return func(msg interface{}, err error) RetryAction {
		logger.Error(fmt.Sprintf("%s failed", operation), "error", err, "msg", msg)
		return RetryAction{ShouldRetry: false, Delay: 0}
	}
}

// RetryableErrorHandler provides retry logic for transient errors
func RetryableErrorHandler(logger *logr.Logger, operation string, maxRetries int) ErrorProcessor {
	return func(msg interface{}, err error) RetryAction {
		if isTransientError(err) {
			retryCount := getMessageRetryCount(msg)
			if retryCount < maxRetries {
				delay := calculateBackoff(retryCount)
				logger.Warn(fmt.Sprintf("%s failed transiently, retrying", operation), "error", err, "retry", retryCount, "delay", delay)
				return RetryAction{ShouldRetry: true, Delay: delay}
			}
		}
		logger.Error(fmt.Sprintf("%s failed permanently", operation), "error", err, "msg", msg)
		return RetryAction{ShouldRetry: false, Delay: 0}
	}
}

// Helper functions
func isTransientError(err error) bool {
	// Implement logic to determine if an error is transient
	// This could check for specific error types or messages
	return true // Simplified for now
}

func getMessageRetryCount[P any](msg P) int {
	// Extract retry count from message headers or context
	return 0 // Simplified for now
}

func calculateBackoff(retryCount int) time.Duration {
	// Implement exponential backoff
	baseDelay := time.Second
	delay := baseDelay * time.Duration(1<<uint(retryCount))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}

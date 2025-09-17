package mq_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/mq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MQ Handlers", func() {
	var (
		buffer      *bytes.Buffer
		mockManager *MockWorkerManager
		logger      *logr.Logger
		ctx          context.Context
		handler      *mq.UnifiedHandler
	)

	BeforeEach(func() {
		buffer = &bytes.Buffer{}
		slogHandler := slog.NewJSONHandler(buffer, &slog.HandlerOptions{
			AddSource: true,
		})
		testLogger := slog.New(slogHandler)
		logr.SetUpLogger(testLogger, nil)

		mockManager = &MockWorkerManager{}
		logger = logr.Log()
		ctx = context.Background()

		config := mq.HandlerConfig{
			Manager: mockManager,
			Theme:   "test-theme",
			Logger:  logger,
			Context: ctx,
		}

		handler = mq.NewUnifiedHandler(config)
	})

	Describe("NewUnifiedHandler", func() {
		It("should create a new handler with default error handler", func() {
			Expect(handler).NotTo(BeNil())
		})

		It("should create a new handler with custom error handler", func() {
			customErrorHandler := func(msg interface{}, err error) mq.RetryAction {
				return mq.RetryAction{ShouldRetry: true, Delay: time.Second}
			}

			config := mq.HandlerConfig{
				Manager:      mockManager,
				Theme:        "test-theme",
				Logger:       logger,
				Context:      ctx,
				ErrorHandler: customErrorHandler,
			}

			handler = mq.NewUnifiedHandler(config)
			Expect(handler).NotTo(BeNil())
		})
	})

	Describe("CreateWorker", func() {
		It("should create a worker with default options", func() {
			mockManager.expectedError = nil

			handlerFunc := func(ctx context.Context, msg interface{}) (interface{}, ierrors.Error) {
				return "processed", nil
			}

			worker, err := handler.CreateWorker("test-exchange", "test.routing.key", handlerFunc)

			Expect(err).NotTo(HaveOccurred())
			Expect(worker).NotTo(BeNil())
			Expect(mockManager.lastQueueName).To(Equal("test-exchange-test-theme-test-routing-key"))
			Expect(mockManager.lastExchange).To(Equal("test-exchange"))
			Expect(mockManager.lastProcessor).NotTo(BeNil())
		})

		It("should return error when manager fails to create worker", func() {
			mockManager.expectedError = fmt.Errorf("failed to create worker")

			handlerFunc := func(ctx context.Context, msg interface{}) (interface{}, ierrors.Error) {
				return "processed", nil
			}

			worker, err := handler.CreateWorker("test-exchange", "test.routing.key", handlerFunc)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create worker"))
			Expect(worker).To(BeNil())
		})
	})

	Describe("CreatePublisher", func() {
		It("should create a publisher", func() {
			mockManager.expectedError = nil
			mockManager.expectedPublisher = &MockPublisher{}

			publisher, err := handler.CreatePublisher("test-exchange", "default.key")

			Expect(err).NotTo(HaveOccurred())
			Expect(publisher).NotTo(BeNil())
			Expect(mockManager.lastPublisherExchange).To(Equal("test-exchange"))
			Expect(mockManager.lastPublisherKey).To(Equal("default.key"))
		})

		It("should return error when manager fails to create publisher", func() {
			mockManager.expectedError = fmt.Errorf("failed to create publisher")

			publisher, err := handler.CreatePublisher("test-exchange", "default.key")

			Expect(err).To(HaveOccurred())
			Expect(publisher).To(BeNil())
		})
	})

	Describe("WorkerOptions", func() {
		It("should apply WithQueueOptions", func() {
			opts := mq.WorkerOptions{}
			queueOpts := mq.QueueOptions{Durable: false, AutoDelete: true}

			mq.WithQueueOptions(queueOpts)(&opts)

			Expect(opts.QueueOptions.Durable).To(BeFalse())
			Expect(opts.QueueOptions.AutoDelete).To(BeTrue())
		})

		It("should apply WithBindingKeys", func() {
			opts := mq.WorkerOptions{}

			mq.WithBindingKeys("key1", "key2")(&opts)

			Expect(opts.BindingKeys).To(Equal([]string{"key1", "key2"}))
		})

		It("should apply WithConsumerTag", func() {
			opts := mq.WorkerOptions{}

			mq.WithConsumerTag("test-consumer")(&opts)

			Expect(opts.ConsumerTag).To(Equal("test-consumer"))
		})

		It("should apply WithPrefetchCount", func() {
			opts := mq.WorkerOptions{}

			mq.WithPrefetchCount(10)(&opts)

			Expect(opts.PrefetchCount).To(Equal(10))
		})

		It("should apply WithErrorHandler", func() {
			opts := mq.WorkerOptions{}
			errorHandler := func(msg interface{}, err error) mq.RetryAction {
				return mq.RetryAction{ShouldRetry: false}
			}

			mq.WithErrorHandler(errorHandler)(&opts)

			Expect(opts.ErrorHandler).NotTo(BeNil())
		})

		It("should apply WithAutoAck", func() {
			opts := mq.WorkerOptions{}

			mq.WithAutoAck(true)(&opts)

			Expect(opts.AutoAck).To(BeTrue())
		})

		It("should apply WithExclusive", func() {
			opts := mq.WorkerOptions{}

			mq.WithExclusive(true)(&opts)

			Expect(opts.Exclusive).To(BeTrue())
		})
	})

	// Note: buildWorkerOptions is a private method, so we test it indirectly through CreateWorker

	Describe("DefaultErrorHandler", func() {
		It("should return retry action with no retry", func() {
			errorHandler := mq.DefaultErrorHandler(logger, "test-operation")

			action := errorHandler("test-message", fmt.Errorf("test error"))

			Expect(action.ShouldRetry).To(BeFalse())
			Expect(action.Delay).To(Equal(time.Duration(0)))
		})
	})

	Describe("RetryableErrorHandler", func() {
		It("should retry transient errors within max retries", func() {
			errorHandler := mq.RetryableErrorHandler(logger, "test-operation", 3)

			action := errorHandler(map[string]interface{}{"retry_count": 1}, fmt.Errorf("transient error"))

			Expect(action.ShouldRetry).To(BeTrue())
			Expect(action.Delay).To(BeNumerically(">", 0))
		})
	})
})

// Mock implementations for testing

type MockWorkerManager struct {
	expectedError       error
	lastQueueName       string
	lastExchange        string
	lastProcessor       mq.Processor
	lastWorkerOptions   mq.WorkerOptions
	expectedPublisher   mq.NewPublisher
	lastPublisherExchange string
	lastPublisherKey     string
}

func (m *MockWorkerManager) CreateWorker(queue, exchange string, processor mq.Processor, opts mq.WorkerOptions) (mq.NewConsumer, error) {
	m.lastQueueName = queue
	m.lastExchange = exchange
	m.lastProcessor = processor
	m.lastWorkerOptions = opts

	if m.expectedError != nil {
		return nil, m.expectedError
	}

	return &MockConsumer{}, nil
}

func (m *MockWorkerManager) CreatePublisher(exchange string, defaultKey string) (mq.NewPublisher, error) {
	m.lastPublisherExchange = exchange
	m.lastPublisherKey = defaultKey

	if m.expectedError != nil {
		return nil, m.expectedError
	}

	return m.expectedPublisher, nil
}

func (m *MockWorkerManager) Close() error {
	return nil
}

type MockConsumer struct {
	running bool
}

func (m *MockConsumer) Start() error {
	m.running = true
	return nil
}

func (m *MockConsumer) Stop() error {
	m.running = false
	return nil
}

func (m *MockConsumer) IsRunning() bool {
	return m.running
}

type MockPublisher struct{}

func (m *MockPublisher) Publish(message interface{}, routingKeys ...string) error {
	return nil
}

func (m *MockPublisher) DelayedPublish(message interface{}, delay time.Duration, routingKeys ...string) error {
	return nil
}

func (m *MockPublisher) Close() error {
	return nil
}
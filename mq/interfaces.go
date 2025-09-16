package mq

import (
	"context"
	"time"
)

// Message represents a generic message envelope
type Message struct {
	Body        []byte
	Headers     map[string]interface{}
	RoutingKey  string
	ContentType string
	Redelivered bool
}

// Connection represents a broker connection
type Connection interface {
	// Connection management
	Close() error
	IsConnected() bool

	// Exchange operations
	DeclareExchange(name, kind string, opts ExchangeOptions) error
	DeleteExchange(name string) error

	// Queue operations
	DeclareQueue(name string, opts QueueOptions) error
	DeleteQueue(name string) error
	BindQueue(queue, exchange, key string) error
	UnbindQueue(queue, exchange, key string) error

	// Publishing
	Publish(exchange, key string, message Message) error

	// Consuming
	Consume(queue, consumer string, opts ConsumeOptions) (<-chan Message, error)
}

// ExchangeOptions defines exchange creation options
type ExchangeOptions struct {
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Arguments  map[string]interface{}
}

// QueueOptions defines queue creation options
type QueueOptions struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Arguments  map[string]interface{}
}

// ConsumeOptions defines message consumption options
type ConsumeOptions struct {
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Arguments map[string]interface{}
}

// NewConsumer handles message consumption
type NewConsumer interface {
	Start() error
	Stop() error
	IsRunning() bool
}

// NewPublisher handles message publishing
type NewPublisher interface {
	Publish(message interface{}, routingKeys ...string) error
	DelayedPublish(message interface{}, delay time.Duration, routingKeys ...string) error
	Close() error
}

// Processor defines message processing logic
type Processor interface {
	Process(ctx context.Context, msg interface{}, routingKey string) error
	OnError(msg interface{}, err error) RetryAction
}

// RetryAction defines what to do when processing fails
type RetryAction struct {
	ShouldRetry bool
	Delay       time.Duration
}

// ErrorProcessor provides error handling
type ErrorProcessor func(msg interface{}, err error) RetryAction

// MessageProcessor provides message processing logic
type MessageProcessor func(ctx context.Context, msg interface{}, routingKey string) error

// NewWorkerManager manages worker lifecycle
type NewWorkerManager interface {
	CreateWorker(queue, exchange string, processor Processor, opts WorkerOptions) (NewConsumer, error)
	CreatePublisher(exchange string, defaultKey string) (NewPublisher, error)
	Close() error
}

// WorkerOptions defines worker configuration
type WorkerOptions struct {
	QueueOptions       QueueOptions
	BindingKeys       []string
	ConsumerTag       string
	PrefetchCount     int
	AutoAck           bool
	Exclusive         bool
	ErrorHandler      ErrorProcessor
}

// BrokerFactory creates broker-specific connections
type BrokerFactory func(config BrokerConfig) (Connection, error)

// BrokerConfig represents generic broker configuration
type BrokerConfig struct {
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
package mq

import (
	"context"
	"os"

	"github.com/kod2ulz/gostart/logr"
)

func Load(ctx context.Context, cnf *Conf, log *logr.Logger) *RMQ {
	switch driver := _mqDriver(); driver {
	case "rabbitmq":
		return RabbitMQ(ctx, log, cnf)
	default:
		log.Fatalf("unsupported MQ driver %s", driver)
	}
	return nil
}

func _mqDriver() (d string) {
	if d = os.Getenv("APP_MQ"); d != "" {
		return
	} else if d = os.Getenv("APP_MQ_DRIVER"); d != "" {
		return
	}
	return "rabbitmq"
}

type Provider interface {
	TopicExchange(name string) *rmqExchange
	Queue(name string, temp ...bool) *rmqQueue
}

type Exchange[msg any] interface {
	Name() string
	Consume(tempQueue string, routingKeys ...string) (<-chan msg, error)
	ConsumeShared(tempQueue string, routingKeys ...string) (<-chan msg, error)
	RemoveConsumer(queue string, routingKeys ...string) error
	Publisher() (ExchangePublisherFunc, error)
	PublisherWithDelay() (ExchangePublisherWithDelayFunc, error)
}

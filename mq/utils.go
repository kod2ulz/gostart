package mq

import (
	"github.com/kod2ulz/gostart/logr"
	"github.com/streadway/amqp"
)

type InterExchangeWorkerUtilOnMessageFunc func(*logr.Logger, *amqp.Delivery, ExchangePublisherFunc) error

func InterExchangeWorkerUtil(log *logr.Logger, incoming <-chan amqp.Delivery, publisher ExchangePublisherFunc, onMessageFunc InterExchangeWorkerUtilOnMessageFunc) {
	for msg := range incoming {
		onMessageFunc(log, &msg, publisher)
	}
}

func contentType(mime ...string) string {
	if len(mime) > 0 {
		return mime[0]
	}
	return "text/plain"
}

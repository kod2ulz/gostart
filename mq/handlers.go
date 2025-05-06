package mq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)


// ApiFunc is meant to be compatible with the api handler functions used by the api
type ApiFunc[R any] func(context.Context) (R, api.Error)

func GenericWorkerErrorHandler[P api.RequestParam](log *logr.Logger, operation string) WorkerErrorFunc[P] {
	return func(p *P, err error) (retry bool, delay time.Duration) {
		log.WithError(err).WithField("msg", p).Errorf("%s %T failed", operation, p)
		return false, 0
	}
}

func GenericWorkerProcessHandler[P api.RequestParam, R any](log *logr.Logger, operation string, fn ApiFunc[R]) WorkerProcessorFunc[P, R] {
	return func(msg *P, routingKey string, redelivered bool) (out R, err error) {
		log.Debugf("received payload:[%T] on route:[%s] :: %T", msg, routingKey, fn)
		return fn(context.WithValue(context.TODO(), (*msg).ContextKey(), *msg))
	}
}

func GenericWorkerSuite[P api.RequestParam, R any](
	ctx context.Context, log *logr.Logger, exchange Exchange[amqp.Delivery], theme, routingKey string,
	prcFn WorkerProcessorFunc[P, R], errFn WorkerErrorFunc[P]) (out Worker[P, R], err error) {
	logger := log.ExtendWithField("subject", fmt.Sprintf("%T", new(P)))
	workerQueue := fmt.Sprintf("%s-%s-%s", exchange.Name(), theme, strings.Replace(routingKey, ".", "-", -1))
	return InitWorkerStrict[P, R](ctx, logger, exchange, workerQueue, routingKey, errFn, prcFn)
}

func GenericWorkerHandler[P api.RequestParam, R any](
	manager WorkerManager, operation, routingKey string, opFunc ApiFunc[R],
) (out Worker[P, R], err error) {
	return GenericWorkerSuite[P, R](
		manager.Context(), manager.Logger(),
		manager.Exchange(), manager.Theme(), routingKey,
		GenericWorkerProcessHandler[P](manager.Logger(), operation, opFunc),
		GenericWorkerErrorHandler[P](manager.Logger(), operation),
	)
}
package mq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ApiFuncV2 is meant to be compatible with the api handler functions used by the api
type ApiFuncV2[P contracts.RequestParam, R any] func(context.Context, P) (R, ierrors.Error)

func GenericWorkerErrorHandlerV2[P contracts.RequestParam](log *logr.Logger, operation string) WorkerErrorFunc[P] {
	return func(p *P, err error) (retry bool, delay time.Duration) {
		log.Error(fmt.Sprintf("%s %T failed", operation, p), "msg", p)
		return false, 0
	}
}

func GenericWorkerProcessHandlerV2[P contracts.RequestParam, R any](log *logr.Logger, operation string, fn ApiFuncV2[P, R]) WorkerProcessorFunc[P, R] {
	return func(msg *P, routingKey string, redelivered bool) (out R, err error) {
		log.Debug(fmt.Sprintf("received payload:[%T] on route:[%s] :: %T", msg, routingKey, fn))
		return fn(context.TODO(), *msg)
	}
}

func GenericWorkerSuiteV2[P contracts.RequestParam, R any](
	ctx context.Context, log *logr.Logger, exchange Exchange[amqp.Delivery], theme, routingKey string,
	prcFn WorkerProcessorFunc[P, R], errFn WorkerErrorFunc[P]) (out Worker[P, R], err error) {
	logger := log.ExtendWithField("subject", fmt.Sprintf("%T", new(P)))
	workerQueue := fmt.Sprintf("%s-%s-%s", exchange.Name(), theme, strings.Replace(routingKey, ".", "-", -1))
	return InitWorkerStrict[P, R](ctx, logger, exchange, workerQueue, routingKey, errFn, prcFn)
}

func GenericWorkerHandlerV2[P contracts.RequestParam, R any](
	manager WorkerManager, operation, routingKey string, opFunc ApiFuncV2[P, R],
) (out Worker[P, R], err error) {
	return GenericWorkerSuiteV2[P, R](
		manager.Context(), manager.Logger(),
		manager.Exchange(), manager.Theme(), routingKey,
		GenericWorkerProcessHandlerV2[P](manager.Logger(), operation, opFunc),
		GenericWorkerErrorHandlerV2[P](manager.Logger(), operation),
	)
}

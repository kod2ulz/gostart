package mq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/logr"
	"github.com/streadway/amqp"
)

// InitFunc is a function that is used to initialise something.
// this is meant to be used by service initialisers
type InitFunc[T any] func(*T)

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
		return fn(context.WithValue(context.TODO(), fmt.Sprintf("%T", msg), *msg))
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

// WorkerManager describes the shared initialisation parameters of a queue worker owner.
// Underneath
type WorkerManager interface {

	// Theme is a simple description of the general category of this worker.
	// this forms part of the temporary queue
	Theme() string

	// Logger should return an instance of the logger
	Logger() *logr.Logger

	// Exchange represents the name of the exchange
	Exchange() Exchange[amqp.Delivery]

	// Context should return the parent context.
	// The worker will terminate with the parent context
	Context() context.Context
}

func ManageWithExchange(
	log *logr.Logger,
	ctx context.Context,
	theme string,
	exchange Exchange[amqp.Delivery],
) WorkerManager {
	return &workerManager{
		theme:    theme,
		log:      log,
		ctx:      ctx,
		exchange: exchange,
	}
}

func ManageWithConnection(
	log *logr.Logger,
	ctx context.Context,
	theme string,
	rmq *RMQ,
	exchange string,
) WorkerManager {
	return &workerManager{
		theme:    theme,
		log:      log,
		ctx:      ctx,
		exchange: rmq.TopicExchange(exchange),
	}
}

func ManageWithExclusiveConnection(
	log *logr.Logger,
	ctx context.Context,
	theme string,
	rmqConf *Conf,
	exchange string,
) WorkerManager {
	return &workerManager{
		theme:   theme,
		log:     log,
		ctx:     ctx,
		exn:     exchange,
		rmqConf: rmqConf,
	}
}

type workerManager struct {
	theme    string
	rmqConf  *Conf
	log      *logr.Logger
	ctx      context.Context
	exn      string
	exchange Exchange[amqp.Delivery]
}

func (w *workerManager) Theme() string            { return w.theme }
func (w *workerManager) Logger() *logr.Logger     { return w.log }
func (w *workerManager) Context() context.Context { return w.ctx }
func (w *workerManager) Exchange() Exchange[amqp.Delivery] {
	if w.exchange != nil {
		return w.exchange
	} else if w.rmqConf != nil {
		rmq := Load(w.ctx, w.rmqConf, w.log)
		go func() {
			for {
				select {
				case <-w.ctx.Done():
					w.log.Warnf("%T: %s %s closing connection", w, w.theme, w.exchange)
					time.Sleep(100 * time.Millisecond)
					rmq.Close()
					return
				}
			}
		}()
		return rmq.TopicExchange(w.exn)
	}
	return nil
}

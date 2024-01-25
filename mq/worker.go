package mq

import (
	"context"
	"fmt"
	"time"

	json "github.com/json-iterator/go"
	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type WorkerInitFunc func() error
type WorkerErrorFunc[P any] func(*P, error) (retry bool, delay time.Duration)
type WorkerProcessorFunc[P, R any] func(msg *P, routingKey string, redelivered bool) (R, error)

func WithWorkerBindingKeys[P, R any](keys ...string) func(*worker[P, R]) {
	return func(w *worker[P, R]) { w.bindkeys = keys }
}

func WithWorkerInitFunc[P, R any](initFuncs ...WorkerInitFunc) func(*worker[P, R]) {
	return func(w *worker[P, R]) { w.initFuncs = initFuncs }
}

func WithWorkerErrorFunc[P, R any](errorFunc WorkerErrorFunc[P]) func(*worker[P, R]) {
	return func(w *worker[P, R]) { w.errorFunc = errorFunc }
}

func WithWorkerProcessorFunc[P, R any](processorFunc WorkerProcessorFunc[P, R]) func(*worker[P, R]) {
	return func(w *worker[P, R]) { w.processorFunc = processorFunc }
}

func InitWorker[P, R any](ctx context.Context, log *logr.Logger, exchange Exchange[amqp.Delivery], queue string, bindingKey string, opts ...InitFunc[worker[P, R]]) (out *worker[P, R], err error) {
	wctx, cancel := context.WithCancel(ctx)
	out = &worker[P, R]{
		log: log, ctx: wctx, cancel: cancel,
		queue:    queue,
		bindkeys: []string{bindingKey},
		exchange: exchange,
	}
	if len(opts) > 0 {
		for i := range opts {
			opts[i](out)
		}
	}
	err = out.start()
	return
}

func InitWorkerStrict[P, R any](
	ctx context.Context, log *logr.Logger, exchange Exchange[amqp.Delivery], queue string, bindingKey string, 
	errorFunc WorkerErrorFunc[P], processorFunc WorkerProcessorFunc[P, R]) (out *worker[P, R], err error) {
	wctx, cancel := context.WithCancel(ctx)
	out = &worker[P, R]{
		log: log, ctx: wctx, cancel: cancel,
		queue:    queue,
		bindkeys: []string{bindingKey},
		exchange: exchange,
		errorFunc: errorFunc,
		processorFunc: processorFunc,
	}
	err = out.start()
	log.WithFields(logrus.Fields{
		"exchange": exchange.Name(), "routingKeys": out.bindkeys, 
		"processFn": out.ProcessFn(), "errorFn": out.ErrorFn(), 
	}).Debugf("initialised worker with processor: %T", processorFunc)
	return
}

type Worker[P, R any] interface {
	Stop() error
	ProcessFn() string
	ErrorFn() string
	WithProcessorFunc(WorkerProcessorFunc[P, R])
	WithErrorFunc(WorkerErrorFunc[P])
}

type worker[P, R any] struct {
	log           *logr.Logger
	ctx           context.Context
	errs          chan workerError[P]
	cancel        context.CancelFunc
	queue         string
	bindkeys      []string
	initFuncs     []WorkerInitFunc
	errorFunc     WorkerErrorFunc[P]
	processorFunc WorkerProcessorFunc[P, R]
	exchange      Exchange[amqp.Delivery]
}

func (w *worker[P, R]) error(err error, msg string, args ...interface{}) error {
	w.log.WithError(err).WithFields(logrus.Fields{
		"queue": w.queue, "keys": w.bindkeys, "exchange": w.exchange.Name(),
	}).Errorf(msg, args...)
	return errors.Wrapf(err, msg, args...)
}

func (w *worker[P, R]) init() (err error) {
	if len(w.bindkeys) == 0 {
		return errors.Errorf("worker initialised without rabbitMQ binding keys")
	}
	for i := range w.initFuncs {
		if err = w.initFuncs[i](); err != nil {
			return err
		}
	}
	if w.errorFunc == nil {
		return errors.Errorf("worker initialised without error handler")
	} else if w.processorFunc == nil {
		return errors.Errorf("worker initialised without message processor")
	}
	return
}

func (w *worker[P, R]) start() error {
	if err := w.init(); err != nil {
		return w.error(err, "initialisation failed")
	}
	incoming, err := w.exchange.ConsumeShared(w.queue, w.bindkeys...)
	if err != nil {
		return w.error(err, "failed to bind to exchange %s for message consumption", w.exchange.Name())
	}
	delayedPublisher, err := w.exchange.PublisherWithDelay()
	if err != nil {
		return w.error(err, "failed to bind to exchange %s for message consumption", w.exchange.Name())
	}
	w.errs = make(chan workerError[P])
	go w.handleErrors(delayedPublisher)
	go w.processIncoming(incoming)
	return nil
}

func (w *worker[P, R]) processIncoming(msgs <-chan amqp.Delivery) {
	for {
		select {
		case m, ok := <-msgs:
			var msg P
			if !ok {
				w.log.Warn("receiver message channel was closed")
				return
			} else if err := json.Unmarshal(m.Body, &msg); err != nil {
				w.error(err, "error unmarshalling queue message to %T", msg)
			} else if _, err = w.processorFunc(&msg, m.RoutingKey, m.Redelivered); err != nil {
				go func() { w.errs <- workerError[P]{err: err, data: &msg, route: m.RoutingKey} }()
			}
			m.Ack(false)
		case <-w.ctx.Done():
			w.log.Warn("parent context is done")
			return
		}
	}
}

type workerError[P any] struct {
	err   error
	data  *P
	route string
}

func (w *worker[P, R]) handleErrors(retryFunc ExchangePublisherWithDelayFunc) {
	for {
		select {
		case e, ok := <-w.errs:
			if !ok {
				w.log.Warn("error channel was closed")
				return
			} else if retry, delay := w.errorFunc(e.data, e.err); retry {
				if body, err := json.Marshal(e.data); err != nil {
					w.error(err, "failed to marshall message for requeue")
				} else if err := retryFunc(body, e.route, delay); err != nil {
					w.error(err, "failed to requeue message")
				}
			}
		case <-w.ctx.Done():
			w.log.Warn("parent context is closed")
			return
		}
	}
}

func (w *worker[P, R]) Stop() error {
	w.cancel()
	close(w.errs)
	return nil
}

func (w *worker[P, R]) ProcessFn() string {
	return fmt.Sprintf("%T", w.processorFunc)
}

func (w *worker[P, R]) ErrorFn() string {
	return fmt.Sprintf("%T", w.errorFunc)
}

func (w *worker[P, R]) WithErrorFunc(fn WorkerErrorFunc[P])  {
	w.errorFunc = fn
}

func (w *worker[P, R]) WithProcessorFunc(fn WorkerProcessorFunc[P, R])  {
	w.processorFunc = fn
}

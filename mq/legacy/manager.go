package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/kod2ulz/gostart/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)

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
					w.log.Warn(fmt.Sprintf("%T: %s %s closing connection", w, w.theme, w.exchange))
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
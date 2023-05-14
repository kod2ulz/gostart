package mq

import (
	"context"
	"time"

	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type ExchangePublisherFunc func(data []byte, routingKey string, mime ...string) error

type ExchangePublisherWithDelayFunc func(data []byte, routingKey string, delay time.Duration, mime ...string) error

type QueuePublisherFunc func(data []byte, mime ...string) error

type rmExchangeDeclare struct {
	name       string
	kind       string
	durable    bool
	autoDelete bool
	internal   bool
	noWait     bool
	args       amqp.Table
}

type rmqExchange struct {
	rmExchangeDeclare
	publisher *rmqConn
	consumer  *rmqConn

	in  chan amqp.Delivery
	out chan []byte

	conf *Conf
	ctx  context.Context
	log  *logr.Logger
}

func (e *rmqExchange) RemoveConsumer(queue string, routingKeys ...string) (err error) {
	if len(routingKeys) > 0 {
		for i := range routingKeys {
			if err = e.consumer.Channel().QueueUnbind(queue, routingKeys[i], e.name, e.args); err != nil {
				return errors.Wrapf(err, "error unbinding queue '%s' from exchange '%s' via key '%s'", queue, e.name, routingKeys[i])
			}
		}
	}
	_, err = e.consumer.Channel().QueueDelete(queue, false, false, false)
	return

}

func (e *rmqExchange) Consume(tempQueue string, routingKeys ...string) (in <-chan amqp.Delivery, err error) {
	return e.consumeQueue(tempQueue, false, routingKeys...)
}

func (e *rmqExchange) ConsumeShared(tempQueue string, routingKeys ...string) (in <-chan amqp.Delivery, err error) {
	return e.consumeQueue(tempQueue, true, routingKeys...)
}

func (e *rmqExchange) consumeQueue(tempQueue string, shared bool, routingKeys ...string) (in <-chan amqp.Delivery, err error) {
	var queue amqp.Queue
	var incoming <-chan amqp.Delivery
	var errs chan *amqp.Error
	e.log.Info("initialising listener")
	if _, _, err = e.consumer.createExchangeBindings(e, tempQueue, routingKeys...); err != nil {
		err = errors.Wrapf(err, "failed to declare key bindings for %s exchange: %s", e.kind, e.name)
		return
	}
	if incoming, err = e.consumer.Subscribe(queue.Name, "", false, !shared, false, false, e.args); err != nil {
		err = errors.Wrapf(err, "failed to open consumer for '%s' '%s' exchange", e.name, e.kind)
		return
	}
	go func() {
		defer func() {
			e.log.Warn("exited listener consumer routine")
		}()
		for {
			select {
			case er, ok := <-errs:
				if !ok {
					e.log.Fatal("error channel closed")
					return
				}
				e.log.WithError(er).WithFields(logrus.Fields{
					"exchange": e.name,
					"type":     e.kind,
				}).Error("encountered error while consuming exchange")
				return
			case msg, ok := <-incoming:
				if !ok {
					e.log.Warn("incoming data channel closed")
					return
				}
				e.in <- msg
			}
		}
	}()
	e.log.WithField("routingKeys", routingKeys).Info("listener initialised")
	return e.in, err
}

func (e *rmqExchange) Publisher() (ExchangePublisherFunc, error) {
	e.log.Info("declaring exchange")
	if err := e.publisher.Channel().ExchangeDeclare(
		e.name,       // name
		e.kind,       // type
		e.durable,    // durable
		e.autoDelete, // auto-deleted
		e.internal,   // internal
		e.noWait,     // no-wait
		e.args,       // arguments
	); err != nil {
		err = errors.Wrapf(err, "failed to declare %s exchange: %s", e.kind, e.name)
		return nil, err
	}
	return func(_data []byte, _routingKey string, _mime ...string) error {
		return e.publisher.Channel().Publish(
			e.name, _routingKey, true, false, amqp.Publishing{
				ContentType: contentType(_mime...),
				Body:        _data,
			},
		)
	}, nil
}

func (e *rmqExchange) PublisherWithDelay() (ExchangePublisherWithDelayFunc, error) {
	e.log.Info("declaring exchange")
	if err := e.publisher.Channel().ExchangeDeclare(
		e.name,       // name
		e.kind,       // type
		e.durable,    // durable
		e.autoDelete, // auto-deleted
		e.internal,   // internal
		e.noWait,     // no-wait
		e.args,       // arguments
	); err != nil {
		err = errors.Wrapf(err, "failed to declare %s exchange: %s", e.kind, e.name)
		return nil, err
	}
	return func(_data []byte, _routingKey string, delay time.Duration, _mime ...string) error {
		return e.publisher.Channel().Publish(
			e.name, _routingKey, true, false, amqp.Publishing{
				Headers:     amqp.Table{"x-delay": delay.Milliseconds()},
				ContentType: contentType(_mime...),
				Body:        _data,
			},
		)
	}, nil
}

func (e *rmqExchange) Name() string {
	return e.name
}

package mq

import (
	"context"

	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type rmQueueDeclare struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       amqp.Table
}

type rmqQueue struct {
	rmQueueDeclare
	publisher *rmqConn
	consumer  *rmqConn

	in  chan amqp.Delivery
	out chan []byte

	conf *Conf
	ctx  context.Context
	log  *logr.Logger
}

func (q *rmqQueue) Consume() (in <-chan amqp.Delivery, err error) {
	return q.consume(false)
}

func (q *rmqQueue) ConsumeShared() (in <-chan amqp.Delivery, err error) {
	return q.consume(true)
}

func (q *rmqQueue) consume(shared bool) (in <-chan amqp.Delivery, err error) {
	var queue amqp.Queue
	var incoming <-chan amqp.Delivery
	var errs chan *amqp.Error
	q.log.Info("initialising listener")
	if _, _, err = q.consumer.createQueueBindings(q); err != nil {
		err = errors.Wrapf(err, "failed to declare key bindings for queue: %s", q.name)
		return
	}
	if incoming, err = q.consumer.Subscribe(queue.Name, "", false, !shared, false, false, q.args); err != nil {
		err = errors.Wrapf(err, "failed to open consumer for '%s' queue", q.name)
		return
	}
	go func() {
		defer func() {
			q.log.Warn("exited listener consumer routine")
		}()
		for {
			select {
			case er, ok := <-errs:
				if !ok {
					q.log.Error("error channel closed")
					return
				}
				q.log.WithError(er).WithFields(logrus.Fields{
					"queue": q.name,
				}).Error("encountered error while consuming exchange")
				return
			case msg, ok := <-incoming:
				if !ok {
					q.log.Warn("incoming data channel closed")
					return
				}
				q.in <- msg
			}
		}
	}()
	q.log.Info("listener initialised")
	return q.in, err
}

func (q *rmqQueue) Publisher() (QueuePublisherFunc, error) {
	queue, err := q.publisher.Channel().QueueDeclare(
		q.name,       // name
		q.durable,    // durable
		q.autoDelete, // auto-deleted
		q.exclusive,  // internal
		q.noWait,     // no-wait
		q.args,       // arguments
	)
	if err != nil {
		err = errors.Wrapf(err, "failed to declare queue: %s", q.name)
		return nil, err
	}
	return func(_data []byte, _mime ...string) error {
		return q.publisher.Channel().Publish(
			"", queue.Name, true, false, amqp.Publishing{
				ContentType: contentType(_mime...),
				Body:        _data,
			},
		)
	}, nil
}

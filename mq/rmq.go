package mq

import (
	"context"
	"sync"

	"github.com/kod2ulz/gostart/logr"
	"github.com/sirupsen/logrus"

	"github.com/streadway/amqp"
)

func RabbitMQ(ctx context.Context, log *logr.Logger, conf *Conf) *RMQ {
	q := RMQ{
		conf:      conf,
		ctx:       ctx,
		log:       log,
		queues:    make(map[string]*rmqQueue),
		exchanges: make(map[string]*rmqExchange),
		publisher: &rmqConn{ctx: ctx, conf: conf, err: make(chan *amqp.Error), exchanges: make(map[string]*rmqConnExchange), ready: make(chan struct{}), log: log.ExtendWithTID(conf.Host + ":publisher")},
		consumer:  &rmqConn{ctx: ctx, conf: conf, err: make(chan *amqp.Error), exchanges: make(map[string]*rmqConnExchange), ready: make(chan struct{}), log: log.ExtendWithTID(conf.Host + ":consumer")},
	}
	var wg sync.WaitGroup
	for _, c := range []*rmqConn{q.consumer, q.publisher} {
		wg.Add(1)
		go func(conn *rmqConn) {
			defer wg.Done()
			conn.log.Info("initialising connection")
			go conn.reconnect()
			<-conn.ready
			conn.log.Info("connection ready")
		}(c)
	}
	wg.Wait()
	q.log.Info("initialised rmq handler")
	return &q
}

type rmqMsg struct {
	exchange, queue, mime string
	body                  interface{}
}

type RMQ struct {
	conf *Conf
	ctx  context.Context
	log  *logr.Logger

	queues    map[string]*rmqQueue
	exchanges map[string]*rmqExchange

	publisher *rmqConn
	consumer  *rmqConn
}

func (q *RMQ) Close() error {
	defer func() {
		if err := recover(); err != nil {
			q.log.WithField(
				"conn", q.conf.String(),
			).Errorf("panic occurred while closing instance. %v", err)
		}
	}()
	go func() {
		if err := q.publisher.close(); err != nil {
			q.log.WithError(err).Error("error closing publisher")
		}
	}()
	go func() {
		if err := q.consumer.close(); err != nil {
			q.log.WithError(err).Error("error closing consumer")
		}
	}()
	return nil
}

func (q *RMQ) DeclareExchange(name, kind string, durable bool, autoDelete bool, internal bool, noWait bool, args amqp.Table) (exchange *rmqExchange) {
	var ok bool
	if exchange, ok = q.exchanges[name]; ok {
		q.log.WithField("exchange", name).Warnf("%s exchange already initialised", kind)
		return
	}
	q.log.WithField("exchange", logrus.Fields{
		"name": name, "type": kind,
	}).Info("initiaising exchange")
	exchange = &rmqExchange{
		rmExchangeDeclare: rmExchangeDeclare{name: name, kind: kind, durable: durable, autoDelete: autoDelete, internal: internal, noWait: noWait, args: args},
		publisher:         q.publisher,
		consumer:          q.consumer,
		in:                make(chan amqp.Delivery),
		out:               make(chan []byte),
		ctx:               q.ctx,
		conf:              q.conf,
		log:               q.log.ExtendWithField("exchange", name),
	}
	q.exchanges[name] = exchange
	exchange.log.Info("initiaised")
	return
}

func (q *RMQ) TopicExchange(name string) (exc *rmqExchange) {
	return q.DeclareExchange(name, "topic", true, false, false, false, amqp.Table{})
}

func (q *RMQ) DeclareQueue(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args amqp.Table) (queue *rmqQueue) {
	var ok bool
	if queue, ok = q.queues[name]; ok {
		q.log.WithField("exchange", name).Warnf("queue already initialised")
		return
	}
	q.log.WithField("queue", name).Info("initiaising exchange")
	queue = &rmqQueue{
		rmQueueDeclare: rmQueueDeclare{name: name, durable: durable, autoDelete: autoDelete, exclusive: exclusive, noWait: noWait, args: args},
		publisher:      q.publisher,
		consumer:       q.consumer,
		in:             make(chan amqp.Delivery),
		out:            make(chan []byte),
		ctx:            q.ctx,
		conf:           q.conf,
		log:            q.log.ExtendWithField("exchange", name),
	}
	q.queues[name] = queue
	queue.log.Info("initiaised")
	return
}

func (q *RMQ) Queue(name string, temp ...bool) *rmqQueue {
	var durable bool = false
	if len(temp) > 0 {
		durable = !temp[0]
	}
	return q.DeclareQueue(name, durable, !durable, false, false, amqp.Table{})
}

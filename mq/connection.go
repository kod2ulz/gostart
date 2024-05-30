package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

const (
	EXCHANGE_TEMP_QUEUE_EXPIRY = 60 * time.Second
)

type rmqConnExchange struct {
	keys      []string
	tempQueue string
	exchange  *rmqExchange
}

type rmqConn struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	err        chan *amqp.Error

	ctx   context.Context
	mx    sync.RWMutex
	ready chan struct{}

	conf *Conf
	log  *logr.Logger

	exchanges map[string]*rmqConnExchange
	// queues    map[string]*rmqQueue
}

func (c *rmqConn) Channel() *amqp.Channel {
	<-c.ready
	c.mx.RLock()
	if c.channel != nil {
		c.mx.RUnlock()
		return c.channel
	}
	c.mx.RUnlock()
	c.log.Warn("waiting for connection to initialise")
	// check if we are closing
	<-c.ready
	if c.channel != nil {
		return c.channel
	}
	// panic
	return c.channel
}

func (c *rmqConn) reconnect() (err error) {
	c.connect()
	if err = <-c.err; err != nil {
		c.log.WithError(err).Fatal("connection closed")
	}
	for {
		select {
		case <-c.ctx.Done():
			if err = c.close(); err != nil {
				c.log.WithError(err).Error("error on close() in reconnect() during context shutdown")
				return err
			}
			c.log.Info("reconnect() exiting")
		case e := <-c.err:
			// these don't recover easily
			c.log.WithError(e).Error("error while reconnecting")
			c.connect()
			time.Sleep(c.conf.Heartbeat * time.Millisecond)
		}
	}
}

func (c *rmqConn) connect() (err error) {
	defer func() {
		c.log.Warn("connection broken")
		// if r := recover(); r != nil {
		// 	c.log.Errorf("panic: %v", r)
		// }
	}()
	c.log.Info("establishing connection")
	c.mx.Lock()
	if c.connection, err = amqp.Dial(c.conf.ConnectionString()); err != nil {
		c.log.WithError(err).Fatalf("failed to establish connection '%s'", c.conf.String())
		c.mx.Unlock()
		return
	}
	c.log.Info("rmq connection established")
	c.err = c.connection.NotifyClose(c.err)

	c.log.Info("connecting to channel")
	if c.channel, err = c.connection.Channel(); err != nil {
		c.log.WithError(err).Fatalf("failed to connect to channel on '%s'", c.conf.String())
		c.mx.Unlock()
		return
	}
	c.mx.Unlock()
	c.log.Info("channel connection established")
	go c.restore()
	go func() {
		close(c.ready)
	}()
	defer func() {
		c.ready = make(chan struct{})
	}()
	for {
		select {
		case chErr := <-c.err:
			c.log.WithError(chErr).Fatalf("connection/channel closed unexpectedly")
			close(c.err)
			return c.handleChanError(chErr)
		case <-c.ctx.Done():
			return c.close()
		}
	}
}

func (c *rmqConn) restore() (err error) {
	// apparently, restoration is impossible
	c.log.Fatalf("%T connection terminated unexpectedly", c)
	if len(c.exchanges) == 0 {
		return
	}
	for _, exc := range c.exchanges {
		_, err = exc.exchange.Consume(exc.tempQueue, exc.keys...)
		if err != nil {
			c.log.WithError(err).WithField("exchange", exc.exchange.name).Fatal("error restoring exchange")
		}
	}
	return
}

func (c *rmqConn) close() error {
	defer func() {
		if err := recover(); err != nil {
			c.log.WithField(
				"conn", c.conf.String(),
			).Errorf("panic occurred while closing instance. %v", err)
		}
		c.channel = nil
	}()
	close(c.err)
	return c.closeConnection()
}

func (c *rmqConn) closeChannel() (err error) {
	defer func() {
		if err := recover(); err != nil {
			c.log.WithField(
				"conn", c.conf.String(),
			).Errorf("panic occurred while closing channel. %v", err)
		}
		c.channel = nil
	}()
	if c.channel == nil {
		c.log.WithField(
			"conn", c.conf.String(),
		).Warn("channel.close() called on nil channel")
		return nil
	} else if err = c.channel.Close(); err != nil {
		c.log.WithField(
			"conn", c.conf.String(),
		).WithError(err).Error("failed to close channel")
	}
	c.channel = nil
	return
}

func (c *rmqConn) closeConnection() error {
	defer func() {
		if err := recover(); err != nil {
			c.log.WithField(
				"conn", c.conf.String(),
			).Errorf("panic occurred while closing connection. %s", err)
		}
		c.connection = nil
	}()
	if c.connection == nil {
		c.log.WithField(
			"conn", c.conf.String(),
		).Warn("connection.close() called on uninitialised connection")
		return nil
	} else if c.channel != nil {
		if e := c.closeChannel(); e != nil {
			c.log.WithError(e).WithField(
				"conn", c.conf.String(),
			).Fatal("error closing channel when calling connection.close()")
		}
	}
	return c.connection.Close()
}

func (c *rmqConn) handleChanError(err *amqp.Error) error {
	switch code := err.Code; code {
	case
		amqp.ContentTooLarge,    // 311
		amqp.NoConsumers,        // 313
		amqp.AccessRefused,      // 403
		amqp.NotFound,           // 404
		amqp.ResourceLocked,     // 405
		amqp.PreconditionFailed: // 406
		c.log.WithError(err).
			WithField("reason", err.Reason).WithField("connection", c.conf.String()).
			WithField("code", code).Fatal("error on channel")
	case
		amqp.ConnectionForced, // 320
		amqp.InvalidPath,      // 402
		amqp.FrameError,       // 501
		amqp.SyntaxError,      // 502
		amqp.CommandInvalid,   // 503
		amqp.ChannelError,     // 504
		amqp.UnexpectedFrame,  // 505
		amqp.ResourceError,    // 506
		amqp.NotAllowed,       // 530
		amqp.NotImplemented,   // 540
		amqp.InternalError:    // 541
		c.log.WithError(err).
			WithField("reason", err.Reason).WithField("connection", c.conf.String()).
			WithField("code", code).Fatal("error on connection")

	default:
		c.log.WithError(err).Panicf("connection on '%s' failed", c.conf.String())
	}
	c.log.WithError(err).
		WithField("code", err.Code).
		WithField("reason", err.Reason).
		WithField("recover", err.Recover).
		Fatal("error on connection. sink the ship")
	return c.close()
}

func (c *rmqConn) createExchangeBindings(exchange *rmqExchange, tempQueue string, keys ...string) (queue amqp.Queue, errs chan *amqp.Error, err error) {
	if tempQueue == "" {
		tempQueue = fmt.Sprintf("%s::temp-%d", exchange.name, time.Now().Unix())
	}
	args := amqp.Table{"x-expires": EXCHANGE_TEMP_QUEUE_EXPIRY.Milliseconds()}
	if queue, err = c.Channel().QueueDeclare(tempQueue, true, false, false, false, args); err != nil {
		err = errors.Wrapf(err, "queue declare for queue '%s' failed", tempQueue)
		return
	}
	if len(keys) == 0 {
		keys = []string{"#"}
	}
	for _, key := range keys {
		if err = c.Channel().QueueBind(queue.Name, key, exchange.name, false, exchange.args); err != nil {
			c.log.WithError(err).Errorf("failed to bind exchange %s to queue %s with key %s", exchange.name, queue.Name, key)
		}
	}
	if err != nil {
		c.log.WithError(err).WithFields(logrus.Fields{
			"keys": keys, "exchange": exchange.name, "exchangeType": exchange.kind, "queue": queue.Name,
		}).Panic("failed to create exchange key bindings to queue. ensure that the exchange exists")
	}
	if _, ok := c.exchanges[exchange.name]; !ok {
		c.exchanges[exchange.name] = &rmqConnExchange{
			keys: keys, exchange: exchange, tempQueue: tempQueue,
		}
	}
	return queue, c.err, nil
}

func (c *rmqConn) createQueueBindings(q *rmqQueue) (queue amqp.Queue, errs chan *amqp.Error, err error) {
	if queue, err = c.Channel().QueueDeclare(q.name, q.durable, q.autoDelete, q.exclusive, q.noWait, q.args); err != nil {
		err = errors.Wrapf(err, "queue declare for queue '%s' failed", q.name)
		return
	}
	return queue, c.err, nil
}

func (c *rmqConn) Subscribe(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return c.Channel().Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

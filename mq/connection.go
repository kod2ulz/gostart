package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
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
	<-c.ready
	if c.channel != nil {
		return c.channel
	}
	return c.channel
}

func (c *rmqConn) reconnect() (err error) {
	c.connect()
	if err = <-c.err; err != nil {
		c.log.Error("connection closed", "error", err)
		panic(err)
	}
	for {
		select {
		case <-c.ctx.Done():
			if err = c.close(); err != nil {
				c.log.Error("error on close() in reconnect() during context shutdown", "error", err)
				return err
			}
			c.log.Info("reconnect() exiting")
		case e := <-c.err:
			c.log.Error("error while reconnecting", "error", e)
			c.connect()
			time.Sleep(c.conf.Heartbeat * time.Millisecond)
		}
	}
}

func (c *rmqConn) connect() (err error) {
	defer func() {
		c.log.Warn("connection broken")
	}()
	c.log.Info("establishing connection")
	c.mx.Lock()
	if c.connection, err = amqp.Dial(c.conf.ConnectionString()); err != nil {
		c.log.Error("failed to establish connection", "connection", c.conf.String(), "error", err)
		panic(err)
	}
	c.log.Info("rmq connection established")
	c.err = c.connection.NotifyClose(c.err)

	c.log.Info("connecting to channel")
	if c.channel, err = c.connection.Channel(); err != nil {
		c.log.Error("failed to connect to channel", "connection", c.conf.String(), "error", err)
		panic(err)
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
			c.log.Error("connection/channel closed unexpectedly", "error", chErr)
			close(c.err)
			return c.handleChanError(chErr)
		case <-c.ctx.Done():
			return c.close()
		}
	}
}

func (c *rmqConn) restore() (err error) {
	if len(c.exchanges) == 0 {
		return
	}
	for _, exc := range c.exchanges {
		_, err = exc.exchange.Consume(exc.tempQueue, exc.keys...)
		if err != nil {
			c.log.Error("error restoring exchange", "exchange", exc.exchange.name, "error", err)
			panic(err)
		}
	}
	return
}

func (c *rmqConn) close() error {
	defer func() {
		if err := recover(); err != nil {
			c.log.Error("panic occurred while closing instance", "connection", c.conf.String(), "error", err)
		}
		c.channel = nil
	}()
	close(c.err)
	return c.closeConnection()
}

func (c *rmqConn) closeChannel() (err error) {
	defer func() {
		if err := recover(); err != nil {
			c.log.Error("panic occurred while closing channel", "connection", c.conf.String(), "error", err)
		}
		c.channel = nil
	}()
	if c.channel == nil {
		c.log.Warn("channel.close() called on nil channel", "connection", c.conf.String())
		return nil
	} else if err = c.channel.Close(); err != nil {
		c.log.Error("failed to close channel", "connection", c.conf.String(), "error", err)
	}
	c.channel = nil
	return
}

func (c *rmqConn) closeConnection() error {
	defer func() {
		if err := recover(); err != nil {
			c.log.Error("panic occurred while closing connection", "connection", c.conf.String(), "error", err)
		}
		c.connection = nil
	}()
	if c.connection == nil {
		c.log.Warn("connection.close() called on uninitialised connection", "connection", c.conf.String())
		return nil
	} else if c.channel != nil {
		if e := c.closeChannel(); e != nil {
			c.log.Error("error closing channel when calling connection.close()", "connection", c.conf.String(), "error", e)
			panic(e)
		}
	}
	return c.connection.Close()
}

func (c *rmqConn) handleChanError(err *amqp.Error) error {
	log := c.log.With("reason", err.Reason, "connection", c.conf.String(), "code", err.Code)
	switch err.Code {
	case amqp.ContentTooLarge, amqp.NoConsumers, amqp.AccessRefused, amqp.NotFound, amqp.ResourceLocked, amqp.PreconditionFailed:
		log.Error("error on channel", "error", err)
		panic(err)
	case amqp.ConnectionForced, amqp.InvalidPath, amqp.FrameError, amqp.SyntaxError, amqp.CommandInvalid, amqp.ChannelError, amqp.UnexpectedFrame, amqp.ResourceError, amqp.NotAllowed, amqp.NotImplemented, amqp.InternalError:
		log.Error("error on connection", "error", err)
		panic(err)
	default:
		c.log.Error(fmt.Sprintf("connection on '%s' failed", c.conf.String()), "error", err)
		panic(err)
	}
	log.Error("error on connection. sink the ship", "recover", err.Recover, "error", err)
	panic(err)
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
			c.log.Error("failed to bind exchange to queue", "exchange", exchange.name, "queue", queue.Name, "key", key, "error", err)
		}
	}
	if err != nil {
		c.log.Error("failed to create exchange key bindings to queue. ensure that the exchange exists", "keys", keys, "exchange", exchange.name, "exchangeType", exchange.kind, "queue", queue.Name, "error", err)
		panic(err)
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

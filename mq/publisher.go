package mq

import (
	"fmt"
	"time"

	json "github.com/json-iterator/go"
	colz "github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type Publisher interface {
	Publish(payload any, routingKey ...string) (err error)
	DelayedPublish(payload any, delay time.Duration, routingKey ...string) (err error)
}

func InitPublisher(log *logr.Logger, exchange Exchange[amqp.Delivery], defaultRoutingKey ...string) (out Publisher, err error) {
	p := &publisher{log: log, exchange: exchange}
	err = p.init(defaultRoutingKey...)
	return p, err
}

type publisher struct {
	log               *logr.Logger
	publisher         ExchangePublisherFunc
	delayPublisher    ExchangePublisherWithDelayFunc
	exchange          Exchange[amqp.Delivery]
	defaultRoutingKey string
}

func (p *publisher) getKey(keys ...string) string {
	if len(keys) > 0 && keys[0] != "" {
		return keys[0]
	}
	return p.defaultRoutingKey
}

func (p *publisher) init(routingKey ...string) (err error) {
	if p.defaultRoutingKey = "#"; len(routingKey) > 0 && routingKey[0] != "" {
		p.defaultRoutingKey = routingKey[0]
	}
	p.publisher, err = p.exchange.Publisher()
	if err != nil {
		return p.error(err, nil, "failed to bind to exchange %s for message publishing", p.exchange.Name())
	}
	p.delayPublisher, err = p.exchange.PublisherWithDelay()
	if err != nil {
		return p.error(err, nil, "failed to bind to exchange %s for message delay publisher", p.exchange.Name())
	}
	return nil
}

func (p *publisher) error(err error, params logrus.Fields, msg string, args ...any) error {
	fields := colz.Map[string, any]{
		"exchange": p.exchange.Name(), "worker": fmt.Sprintf("%T", p),
	}
	p.log.WithError(err).WithFields(logrus.Fields(fields.Merge(params))).Errorf(msg, args...)
	return errors.Wrapf(err, msg, args...)
}

func (p *publisher) Publish(payload any, routingKey ...string) (err error) {
	var data []byte
	if data, err = json.Marshal(payload); err != nil {
		return
	} else if err = p.publisher(data, p.getKey(routingKey...)); err != nil {
		return p.error(err, map[string]any{
			"routing-key": routingKey, "payload": payload,
		}, "failed to publish message via route")
	}
	return
}

func (p *publisher) DelayedPublish(payload any, delay time.Duration, routingKey ...string) (err error) {
	var data []byte
	if data, err = json.Marshal(payload); err != nil {
		return
	} else if err = p.delayPublisher(data, p.getKey(routingKey...), delay); err != nil {
		return p.error(err, map[string]any{
			"routing-key": routingKey, "payload": payload,
		}, "failed to publish message via route")
	}
	return
}

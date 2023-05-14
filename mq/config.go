package mq

import (
	"fmt"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/utils"
)

type ExchangeConfig struct {
	Name        string
	BindingKeys ExchangeKeys
	TempQueue   Name
	ErrorKey    Name
}

type Name string

func (n *Name) OrDefault(val string) string {
	if *n != "" {
		return string(*n)
	}
	return val
}

type ExchangeKeys string

func (k *ExchangeKeys) List(_default ...string) (out []string) {
	if len(*k) == 0 {
		if len(_default) > 0 {
			return _default
		}
		return []string{}
	} else if str := string(*k); !strings.Contains(str, ",") {
		return []string{str}
	}
	list := strings.Split(string(*k), ",")
	out = make([]string, 0)
	for i := range list {
		if str := strings.Trim(list[i], " "); str != "" {
			out = append(out, str)
		}
	}
	return
}

func (k ExchangeKeys) Empty() bool {
	return string(k) == ""
}

type Conf struct {
	Host             string
	Port             string
	Vhost            string
	Heartbeat        time.Duration
	HeartbeatTimeout time.Duration
	Username         string
	Password         string
	Protocol         string
	ConsumerExchange ExchangeConfig
	ProducerExchange ExchangeConfig
}

const (
	DefaultHeartbeat        = 5000
	DefaultHeartbeatTimeout = 20000
)

func Config(prefix ...string) *Conf {
	env := utils.Env.Helper(prefix...).OrDefault("MQ")
	heartbeat := env.Get("HEARTBEAT_MILLISECONDS", DefaultHeartbeat).Int()
	heartbeatTimeout := env.Get("HEARTBEAT_TIMEOUT_MILLISECONDS", DefaultHeartbeatTimeout).Int()

	return &Conf{
		Host:             env.GetString("HOST", "127.0.0.1"),
		Port:             env.GetString("PORT", "5672"),
		Vhost:            env.GetString("VHOST", "/"),
		Heartbeat:        time.Duration(heartbeat) * time.Millisecond,
		HeartbeatTimeout: time.Duration(heartbeatTimeout) * time.Millisecond,
		Username:         env.GetString("USERNAME", "guest"),
		Password:         env.GetString("PASSWORD", "guest"),
		Protocol:         env.GetString("PROTOCOL", "amqp"),
		ConsumerExchange: ExchangeConfig{
			Name:        env.GetString("EXCHANGE_CONSUMER", "exchange.in"),
			BindingKeys: ExchangeKeys(env.GetString("EXCHANGE_CONSUMER_BINDING_KEYS", "#")),
			ErrorKey:    Name(env.GetString("EXCHANGE_CONSUMER_ERROR_KEY", "")),
			TempQueue:   Name(env.GetString("EXCHANGE_CONSUMER_TEMP_QUEUE_NAME", "")),
		},
		ProducerExchange: ExchangeConfig{
			Name:        env.GetString("EXCHANGE_PRODUCER", "exchange.out"),
			BindingKeys: ExchangeKeys(env.GetString("EXCHANGE_PRODUCER_BINDING_KEYS", "#")),
			ErrorKey:    Name(env.GetString("EXCHANGE_PRODUCER_ERROR_KEY", "")),
			TempQueue:   Name(env.GetString("EXCHANGE_PRODUCER_TEMP_QUEUE_NAME", "")),
		},
	}
}

func (c *Conf) ConnectionString() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s:%s/",
		c.Protocol,
		c.Username,
		c.Password,
		c.Host,
		c.Port,
	)
}

func (c *Conf) String() string {
	return fmt.Sprintf(
		"%s://%s:%s",
		c.Protocol,
		c.Host,
		c.Port,
	)
}

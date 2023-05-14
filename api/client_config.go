package api

import (
	"time"

	"github.com/kod2ulz/gostart/utils"
)

type HttpClientConfig struct {
	Host           string
	RequestTimeout time.Duration
	RetryTimeout   time.Duration
	RetryTimes     int
	DisableAuth    bool

	prefix string
}

func ClientConfig(prefix ...string) (conf *HttpClientConfig) {
	env := utils.Env.Helper(prefix...).OrDefault("SERVICE_API")

	conf = &HttpClientConfig{
		Host:           env.GetString("URL", "https://api.service.io"),
		RetryTimeout:   env.Get("RETRY_TIMEOUT", "5s").Duration(),
		RequestTimeout: env.Get("RETRY_TIMEOUT", "15s").Duration(),
		RetryTimes:     env.Get("RETRY_TIMES", "3").Int(),
		DisableAuth:    env.Get("DISABLE_AUTH", "false").Bool(),
		prefix:         env.Prefix(),
	}
	return
}

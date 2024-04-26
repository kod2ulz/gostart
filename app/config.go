package app

import (
	"fmt"
	"time"

	"github.com/kod2ulz/gostart/utils"
)

var _config *conf

func Conf() *conf {
	if _config != nil {
		return _config
	}
	var host = utils.Env.GetHost()
	var env = utils.Env.Helper("APP")
	_config = &conf{
		Host:        host,
		Name:        env.Get("NAME", host).String(),
		Version:     env.Get("VERSION", "ver-0.0.0").String(),
		HttpPort:    env.Get("HTTP_PORT", "49080").Int(),
		HttpAddress: env.Get("HTTP_ADDRESS", "0.0.0.0").String(),
		Location:    env.Get("TIME_LOCATION", "Africa/Kampala").Location(),
		Uptime:      UptimeCheckConf(env.Prefix(), "UPTIME_CHECK"),
		Http:        HttpConf(env.Prefix(), "HTTP_SERVER"),
	}
	return _config
}

type conf struct {
	Host        string
	Name        string
	Version     string
	HttpPort    int
	HttpAddress string
	Location    *time.Location
	Uptime      *uptimeCheckConf
	Http        *httpConf
}

func (c conf) Address() string {
	return fmt.Sprintf("%s:%d", c.HttpAddress, c.HttpPort)
}

type httpConf struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	MaxAge           time.Duration
	AllowCredentials bool
}

func HttpConf(prefix ...string) (conf *httpConf) {
	env := utils.Env.Helper(prefix...).OrDefault("HTTP_SERVER")

	return &httpConf{
		AllowOrigins:     env.Get("ALLOW_ORIGINS", "*").StringList(","),
		AllowMethods:     env.Get("ALLOW_METHODS", "GET,POST,PUT,HEAD,OPTIONS").StringList(","),
		AllowHeaders:     env.Get("ALLOW_HEADERS", "Origin,Content-Length,Accept-Encoding,Authorization,Accept-Language,Content-Type").StringList(","),
		ExposeHeaders:    env.Get("EXPOSE_HEADERS", "Content-Length,Host,Content-Type,Connection").StringList(","),
		MaxAge:           env.Get("MAX_AGE", "12h").Duration(),
		AllowCredentials: env.Get("ALLOW_CREDENTIALS", "true").Bool(),
	}
}

type uptimeCheckConf struct {
	Interval time.Duration
	Timeout  time.Duration
}

func UptimeCheckConf(prefix ...string) (conf *uptimeCheckConf) {
	env := utils.Env.Helper(prefix...).OrDefault("UPTIME_CHECK")

	return &uptimeCheckConf{
		Interval: env.Get("INTERVAL", "10s").Duration(),
		Timeout:  env.Get("TIMEOUT", "30s").Duration(),
	}
}

package app

import (
	"fmt"
	"time"

	"github.com/kod2ulz/gostart/config"
)

var _config *conf

func Conf() *conf {
	if _config != nil {
		return _config
	}
	var host = config.Get("host").String()

	// Support both new (SERVER_*) and old (APP_*) config formats
	// SERVER_HTTP_PORT takes precedence over APP_HTTP_PORT
	httpPort := config.Get("SERVER_HTTP_PORT", config.Get("APP_HTTP_PORT", "9025").String()).Int()
	httpAddr := config.Get("SERVER_HTTP_ADDRESS", config.Get("APP_HTTP_ADDRESS", "0.0.0.0").String()).String()

	_config = &conf{
		Host:        host,
		Name:        config.Get("APP_NAME", host).String(),
		Version:     config.Get("APP_VERSION", "ver-0.0.0").String(),
		HttpPort:    httpPort,
		HttpAddress: httpAddr,
		Location:    config.Get("APP_TIME_LOCATION", "Africa/Kampala").Location(),
		Uptime:      UptimeCheckConf(),
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
}

func (c conf) Address() string {
	return fmt.Sprintf("%s:%d", c.HttpAddress, c.HttpPort)
}

type uptimeCheckConf struct {
	Interval time.Duration
	Timeout  time.Duration
}

func UptimeCheckConf(prefix ...string) (conf *uptimeCheckConf) {
	return &uptimeCheckConf{
		Interval: config.Get("UPTIME_CHECK_INTERVAL", "10s").Duration(),
		Timeout:  config.Get("UPTIME_CHECK_TIMEOUT", "30s").Duration(),
	}
}

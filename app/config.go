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
	// var env = config.Env.Helper("APP")
	_config = &conf{
		Host:        host,
		Name:        config.Get("APP_NAME", host).String(),
		Version:     config.Get("APP_VERSION", "ver-0.0.0").String(),
		HttpPort:    config.Get("APP_HTTP_PORT", "9025").Int(),
		HttpAddress: config.Get("APP_HTTP_ADDRESS", "0.0.0.0").String(),
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

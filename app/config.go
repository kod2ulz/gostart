package app

import (
	"fmt"
	"os"
	"time"

	"github.com/kod2ulz/gostart/utils"
)

var _config *conf

func Conf() *conf {
	if _config != nil {
		return _config
	}
	env := utils.Env.GetOrDefault
	host, e := os.Hostname()
	if e != nil {
		host = os.Getenv("HOST")
	}
	_config = &conf{
		Host:        host,
		Name:        env("APP_NAME", host).String(),
		HttpPort:    env("APP_HTTP_PORT", "49080").String(),
		HttpAddress: env("APP_HTTP_ADDRESS", "0.0.0.0").String(),
		Timeout:     time.Duration(env("APP_TIMEOUT", "5").Int()) * time.Second,
		Http:        HttpConf(),
	}
	return _config
}

type conf struct {
	Host        string
	Name        string
	HttpPort    string
	HttpAddress string
	Timeout     time.Duration
	Http        *httpConf
}

func (c conf) Address() string {
	return fmt.Sprintf("%s:%s", c.HttpAddress, c.HttpPort)
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
	env := utils.Env.Helper(prefix...).OrDefault("APP_HTTP_SERVER")

	return &httpConf{
		AllowOrigins:     env.Get("ALLOW_ORIGINS", "*").StringList(","),
		AllowMethods:     env.Get("ALLOW_METHODS", "GET,POST,PUT,HEAD,OPTIONS").StringList(","),
		AllowHeaders:     env.Get("ALLOW_HEADERS", "Origin,Content-Length,Accept-Encoding,Authorization,Accept-Language,Content-Type").StringList(","),
		ExposeHeaders:    env.Get("EXPOSE_HEADERS", "Content-Length,Host,Content-Type,Connection").StringList(","),
		MaxAge:           env.Get("MAX_AGE", "12h").Duration(),
		AllowCredentials: env.Get("ALLOW_CREDENTIALS", "true").Bool(),
	}
}

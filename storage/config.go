package storage

import (
	"fmt"
	"time"

	"github.com/kod2ulz/gostart/utils"
)

type Conf struct {
	Driver           string
	Host             string
	Port             string
	Heartbeat        time.Duration
	HeartbeatTimeout time.Duration
	Username         string
	Password         string
	Database         string
	SSLMode          string
}

const (
	DefaultHeartbeat        = 5000
	DefaultHeartbeatTimeout = 20000
)

var defaults = map[string]map[string]string{
	"postgres": {
		"PORT":     "5432",
		"USERNAME": "postgres",
		"PASSWORD": "postgres",
		"DATABASE": "postgres",
	},
	"redis": {
		"PORT":     "6379",
		"DATABASE": "0",
	},
}

func Config(prefix ...string) (conf *Conf) {
	env := utils.Env.Helper(prefix...)
	heartbeat := env.Get("HEARTBEAT_MILLISECONDS", DefaultHeartbeat).Int()
	heartbeatTimeout := env.Get("HEARTBEAT_TIMEOUT_MILLISECONDS", DefaultHeartbeatTimeout).Int()

	conf = &Conf{
		Heartbeat:        time.Duration(heartbeat) * time.Millisecond,
		HeartbeatTimeout: time.Duration(heartbeatTimeout) * time.Millisecond,
		Driver:           env.MustGet("DRIVER").String(),
	}

	conf.Host = env.GetString("HOST", conf._default("HOST"))
	conf.Port = env.GetString("PORT", conf._default("PORT"))
	conf.Username = env.GetString("USERNAME", conf._default("USERNAME"))
	conf.Password = env.GetString("PASSWORD", conf._default("PASSWORD"))
	conf.Database = env.GetString("DATABASE", conf._default("DATABASE"))
	conf.SSLMode = env.GetString("SSL_MODE", conf._default("disable"))
	return
}

func (c *Conf) ConnectionString() string {
	switch c.Driver {
	case "postgres":
		return c.postgresConnectionString()
	case "redis":
		return c.postgresConnectionString()
	}
	return fmt.Sprintf(
		"%s://%s:%s@%s:%s",
		c.Driver,
		c.Username,
		c.Password,
		c.Host,
		c.Port,
	)
}

func (c *Conf) String() string {
	return fmt.Sprintf(
		"%s://%s:%s",
		c.Driver,
		c.Host,
		c.Port,
	)
}

func (c Conf) _default(prop string) string {
	if _, ok := defaults[c.Driver]; !ok {
		return ""
	}
	return defaults[c.Driver][prop]
}

func (c *Conf) postgresConnectionString() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

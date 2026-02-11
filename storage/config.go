package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/config"
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
	// Helper to construct keys with the given prefix, e.g., buildKey("HOST") -> "POSTGRES_DB_HOST"
	buildKey := func(key string) string {
		return strings.ToUpper(strings.Join(append(prefix, key), "_"))
	}

	// Use the main config.Get, which respects the full hierarchy.
	heartbeat := config.Get(buildKey("HEARTBEAT_MILLISECONDS"), DefaultHeartbeat).Int()
	heartbeatTimeout := config.Get(buildKey("HEARTBEAT_TIMEOUT_MILLISECONDS"), DefaultHeartbeatTimeout).Int()

	driver := config.Get(buildKey("DRIVER")).String()
	if driver == "" {
		panic(fmt.Sprintf("missing required config key: %s", buildKey("DRIVER")))
	}

	conf = &Conf{
		Heartbeat:        time.Duration(heartbeat) * time.Millisecond,
		HeartbeatTimeout: time.Duration(heartbeatTimeout) * time.Millisecond,
		Driver:           driver,
	}

	conf.Host = config.Get(buildKey("HOST"), conf._default("HOST")).String()
	conf.Port = config.Get(buildKey("PORT"), conf._default("PORT")).String()
	conf.Username = config.Get(buildKey("USERNAME"), conf._default("USERNAME")).String()
	conf.Password = config.Get(buildKey("PASSWORD"), conf._default("PASSWORD")).String()
	conf.Database = config.Get(buildKey("DATABASE"), conf._default("DATABASE")).String()
	conf.SSLMode = config.Get(buildKey("SSL_MODE"), "disable").String()
	return
}

func (c *Conf) ConnectionString() string {
	switch c.Driver {
	case "postgres":
		return c.postgresConnectionString()
	case "redis":
		return c.redisConnectionString()
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

func (c *Conf) redisConnectionString() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

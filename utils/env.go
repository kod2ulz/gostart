package utils

import (
	"fmt"
	"os"
	"strings"
)

type EnvUtil interface {
	OrDefault(prx ...string) EnvUtil
	Get(name string, _default ...interface{}) Value
	MustGet(name string) (val Value)
	GetString(name string, _default ...interface{}) string
	Prefix() string
}

var Env envUtils

type envUtils struct {
}

func (envUtils) GetOrDefault(env, _default string) Value {
	if val := os.Getenv(env); val != "" {
		return Value(val)
	}
	return Value(_default)
}

func (envUtils) Helper(prefix ...string) EnvUtil {
	env := _env{}
	return env.setPrx(prefix...)
}

func (envUtils) GetHost() string {
	host, err := os.Hostname()
	if err == nil {
		return host
	}
	return os.Getenv("HOST")
}

type _env struct {
	prx string
}

func (e *_env) OrDefault(prx ...string) EnvUtil {
	if len(prx) == 0 || e.prx != "" {
		return e
	}
	e.prx = strings.Join(prx, "_")
	return e
}

func (e *_env) setPrx(prx ...string) *_env {
	if len(prx) == 0 {
		return e
	}
	e.prx = strings.Trim(strings.Join(prx, "_"), "_")
	return e
}

func (e _env) Get(name string, _default ...interface{}) Value {
	var def string
	if len(_default) > 0 {
		def = fmt.Sprint(_default[0])
	}
	return Env.GetOrDefault(e._prx(name), def)
}

func (e _env) MustGet(name string) (val Value) {
	if val = e.Get(name, ""); !val.Valid() {
		panic("could not find env var " + e.prx + name)
	}
	return
}

func (e _env) GetString(name string, _default ...interface{}) string {
	return e.Get(name, _default...).String()
}

func (e _env) _prx(name string) string {
	return fmt.Sprintf("%s_%s", strings.Trim(e.prx, "_"), name)
}

func (e _env) Prefix() string {
	return e.prx
}

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
)

type EnvUtil interface {
	OrDefault(prx ...string) EnvUtil
	Extend(prx ...string) EnvUtil
	Get(name string, _default ...interface{}) Value
	MustGet(name string) (val Value)
	GetString(name string, _default ...interface{}) string
	Prefix() string
}

var Env EnvSource

type EnvSource struct {
}

func (EnvSource) GetOrDefault(env, _default string) Value {
	if val := os.Getenv(env); val != "" {
		return Value(val)
	}
	return Value(_default)
}

func (EnvSource) Helper(prefix ...string) EnvUtil {
	env := _env{}
	return env.setPrx(prefix...)
}

func (EnvSource) GetHost() string {
	host, err := os.Hostname()
	if err == nil {
		return host
	}
	return os.Getenv("HOST")
}

type _env struct {
	prx string
}

// Extend creates a new EnvHelper with the parent prefix of the curent helper. prefix should not be empty
func (e _env) Extend(prefix ...string) EnvUtil {
	env := _env{}
	return env.setPrx(append([]string{e.prx}, prefix...)...)
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
	// Transform the key to SCREAMING_SNAKE_CASE for the env var lookup.
	lookupKey := strcase.ToScreamingSnake(name)

	if len(_default) == 0 {
		return Env.GetOrDefault(e._prx(lookupKey), def)
	}
	for i := range _default {
		if def = fmt.Sprint(_default[i]); def != "" {
			break
		}
	}
	return Env.GetOrDefault(e._prx(lookupKey), def)
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
	prefix := strings.Trim(e.prx, "_")
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s_%s", prefix, name)
}

func (e _env) Prefix() string {
	return e.prx
}

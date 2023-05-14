package utils

import (
	"context"

	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func LogError(log *logrus.Entry, err error, message string, args ...interface{}) {
	if err == nil {
		return
	}
	log.WithError(err).Errorf(message, args...)
}

type App interface {
	Log() *logr.Logger
	Ctx() context.Context
}

func StopFunc(a App, fn func() error, message string, args ...interface{}) {
	a.Log().Infof("stopping %T", fn)
	if err := fn(); err != nil {
		LogError(a.Log().Entry, err, message, args...)
	} else {
		a.Log().Infof("%T stopped", fn)
	}
}

func StopFuncN(a App, fn func(), message string, args ...interface{}) {
	a.Log().Infof("stopping %T", fn)
	defer func() {
		if r := recover(); r != nil {
			LogError(a.Log().Entry, errors.Errorf("recover: %v", r), message, args...)
		}
	}()
	fn()
	a.Log().Infof("%T stopped", fn)
}

func StopFuncWithCtx(a App, fn func(context.Context) error, message string, args ...interface{}) {
	StopFunc(a, func() error {
		return fn(a.Ctx())
	}, message, args...)
}

type ShFunc1 func()
type ShFunc2 func() error
type ShFunc3 func(context.Context) error
type shutdownFuncs interface {
	ShFunc1 | ShFunc2 | ShFunc3
}

func ErrorFunc[F shutdownFuncs](a App, fn F, message string, args ...interface{}) {
	switch any(fn).(type) {
	case ShFunc1:
		StopFuncN(a, any(fn).(ShFunc1), message, args...)
	case ShFunc2:
		StopFunc(a, any(fn).(ShFunc2), message, args...)
	case ShFunc3:
		StopFuncWithCtx(a, any(fn).(ShFunc3), message, args...)
	}
}

package utils

import (
	"context"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/logr"
	"github.com/sirupsen/logrus"
)

type taskUtils struct{}

var Task taskUtils

func (u taskUtils) WithRetry(log *logr.Logger, tries int, wait time.Duration, fn func() error) (e error) {
	tr := tries
	for {
		if e = fn(); e == nil {
			return
		} else if tries <= 0 {
			return
		}
		tries--
		log.WithError(e).Errorf("attempt %d failed. retrying in %v", tr-tries, wait)
		time.Sleep(wait)
	}
}

func (u taskUtils) WithTimeout(timeout time.Duration, fn func() error) (success bool) {
	timer := Timer().After(timeout)
	for {
		select {
		case <-timer:
			return
		default:
			if err := fn(); err == nil {
				return true
			}
		}
	}
}

func SafeChannelWrite[T any](ctx context.Context, log *logrus.Entry, data T, out chan<- T, closeMessage ...string) (err error) {
	go func(in T) {
		for {
			select {
			case <-ctx.Done():
				var message string = "parent context is done"
				if len(closeMessage) > 0 {
					message = strings.Join(closeMessage, ". ")
				}
				log.Warn(message)
				return
			default:
				if out != nil {
					out <- in
				}
				return
			}
		}
	}(data)
	return
}

func PointerTo[T any](t T) *T {
	return &t
}

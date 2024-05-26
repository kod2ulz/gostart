package utils

import (
	"context"
	"strings"
	"sync"
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

type BatchProcessorFunc[T any, E error] func(context.Context, T) E

func ProcessBatch[T any, E error](ctx context.Context, batchSize int, processor BatchProcessorFunc[T, E], args...T) (err E) {

	if len(args) == 0 {
    return
  }

	ctxwc, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan E, 1)
	semaphore := make(chan struct{}, batchSize)
	var wg sync.WaitGroup

	// Divide Data into batches
	for i := 0; i < len(args); i += batchSize {
		wg.Add(1)
		limit := i+batchSize
		if limit > len(args) {
			limit = len(args)
		}
		go func(start, end int) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			batchProcessor(ctxwc, errCh, processor, args[start:end])
			<-semaphore // Release semaphore
		}(i, limit)
	}

	go func() {
		wg.Wait()
		close(errCh) // Close error channel when all goroutines are done
	}()

	// Collect errors
	for err = range errCh {
		cancel() // Cancel context on first error
		return err
	}
	return
}

func batchProcessor[T any, E error](ctx context.Context, errCh chan<- E, processor BatchProcessorFunc[T, E], ts []T) {
	for _, url := range ts {
		select {
		case <-ctx.Done():
			return
		default:
			if err := processor(ctx, url); any(err) != nil {
				errCh <- err
				return
			}
		}
	}
}

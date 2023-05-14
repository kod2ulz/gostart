package utils

import (
	"time"
)

type T struct {
	cancel chan bool
}

func Timer() *T {
	return &T{
		cancel: make(chan bool),
	}
}

func (c *T) wait(d time.Duration, ch chan bool) {
	select {
	case <-time.After(d):
		ch <- true
	case <-c.cancel:
		ch <- false
	}
}

func (c *T) After(d time.Duration) chan bool {
	ch := make(chan bool)
	go c.wait(d, ch)
	return ch
}

func (c *T) Cancel() {
	close(c.cancel)
}

package core

import (
	"time"
)

type reconnectTimer struct {
	cancel chan bool
}

// NewReconnectTimer builds a new reconnect timer
func newReconnectTimer() *reconnectTimer {
	return &reconnectTimer{
		cancel: make(chan bool),
	}
}

// After returns a channel that will return true/false based on whether the timer was canceled
func (c *reconnectTimer) After(d time.Duration) chan bool {
	ch := make(chan bool)
	go func() {
		select {
		case <-time.After(d):
			ch <- true
		case <-c.cancel:
			ch <- false
		}
	}()
	return ch
}

// Cancel stops the timer
func (c *reconnectTimer) Cancel() {
	close(c.cancel)
}

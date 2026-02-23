package testutil

import (
	"fmt"
	"time"
)

// WaitFor polls condition at the given interval until it returns true or the
// timeout elapses. Returns an error if the timeout is reached.
func WaitFor(condition func() bool, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for condition", timeout)
		}
		time.Sleep(interval)
	}
}

// WaitForValue polls fn until it returns a non-zero value or the timeout
// elapses. Useful for waiting on counters or lengths.
func WaitForValue[T comparable](fn func() T, expected T, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last T
	for {
		last = fn()
		if last == expected {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s: last value %v, expected %v", timeout, last, expected)
		}
		time.Sleep(interval)
	}
}

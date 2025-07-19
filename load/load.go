package load

import "sync/atomic"

// Counter is a thread-safe counter for tracking system load.
type Counter struct {
	value int64
}

// NewCounter creates a new Counter.
func NewCounter() *Counter {
	return &Counter{}
}

// Increment increments the counter by 1.
func (c *Counter) Increment() {
	atomic.AddInt64(&c.value, 1)
}

// Decrement decrements the counter by 1.
func (c *Counter) Decrement() {
	atomic.AddInt64(&c.value, -1)
}

// Get returns the current value of the counter.
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

package observer

import (
	"sync"
	"time"

	"github.com/montanaflynn/stats"
)

// Observer monitors request latency and adjusts the high-load threshold.
type Observer struct {
	mutex             sync.RWMutex
	latencies         []float64
	highLoadThreshold time.Duration
	latencyThreshold  time.Duration
	minDataPoints     int
}

// NewObserver creates a new Observer.
func NewObserver(latencyThreshold time.Duration, minDataPoints int) *Observer {
	return &Observer{
		latencies:        make([]float64, 0, 1000),
		latencyThreshold: latencyThreshold,
		minDataPoints:    minDataPoints,
	}
}

// RecordLatency records a new request latency.
func (o *Observer) RecordLatency(latency time.Duration) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.latencies = append(o.latencies, float64(latency.Milliseconds()))
}

// HighLoadThreshold returns the current high-load threshold.
func (o *Observer) HighLoadThreshold() time.Duration {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	return o.highLoadThreshold
}

// Run starts the observer's monitoring loop.
func (o *Observer) Run(ticker *time.Ticker) {
	for range ticker.C {
		o.updateHighLoadThreshold()
	}
}

// updateHighLoadThreshold calculates the P95 latency and updates the threshold.
func (o *Observer) updateHighLoadThreshold() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if len(o.latencies) < o.minDataPoints {
		return
	}

	p95, err := stats.Percentile(o.latencies, 95)
	if err != nil {
		// Handle error, e.g., log it
		return
	}

	o.highLoadThreshold = time.Duration(p95) * time.Millisecond

	// Reset latencies after calculation
	o.latencies = make([]float64, 0, 1000)
}

package observer

import (
	"context"
	"sync"
	"time"

	"qqbotrouter/interfaces"

	"github.com/montanaflynn/stats"
)

// Ensure Observer implements Observer interface
var _ interfaces.Observer = (*Observer)(nil)

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

// GetCurrentLoad returns the current system load based on recent latencies
func (o *Observer) GetCurrentLoad() float64 {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	if len(o.latencies) == 0 {
		return 0.0
	}

	// Calculate load as the ratio of current average latency to threshold
	var sum float64
	for _, latency := range o.latencies {
		sum += latency
	}
	avgLatency := sum / float64(len(o.latencies))
	thresholdMs := float64(o.latencyThreshold.Milliseconds())

	if thresholdMs == 0 {
		return 0.0
	}

	load := avgLatency / thresholdMs
	if load > 1.0 {
		return 1.0
	}
	return load
}

// Run starts the observer with context support
func (o *Observer) Run(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			o.updateHighLoadThreshold()
		}
	}
}

// GetTickerInterval returns the interval for periodic execution
func (o *Observer) GetTickerInterval() string {
	return "10s"
}

// RunWithContext starts the observer with context support (deprecated, use Run instead)
func (o *Observer) RunWithContext(ctx context.Context, ticker *time.Ticker) {
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.updateHighLoadThreshold()
		}
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

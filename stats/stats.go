package stats

import (
	"context"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"qqbotrouter/interfaces"
)

// Ensure StatsAnalyzer implements StatProvider interface
var _ interfaces.StatProvider = (*StatsAnalyzer)(nil)

// StatsAnalyzer analyzes user message intervals to determine dynamic baselines.
type StatsAnalyzer struct {
	mutex            sync.RWMutex
	messageIntervals []float64
	p50              time.Duration
	p90              time.Duration
	minDataPoints    int
	modeSwitched     chan bool
}

// NewStatsAnalyzer creates a new StatsAnalyzer.
func NewStatsAnalyzer(minDataPoints int) *StatsAnalyzer {
	return &StatsAnalyzer{
		messageIntervals: make([]float64, 0, 10000),
		minDataPoints:    minDataPoints,
		modeSwitched:     make(chan bool, 1),
	}
}

// RecordMessageInterval records a new message interval.
func (s *StatsAnalyzer) RecordMessageInterval(interval time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.messageIntervals = append(s.messageIntervals, float64(interval.Milliseconds()))
}

// P50 returns the 50th percentile of message intervals.
func (s *StatsAnalyzer) P50() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.p50
}

// P90 returns the 90th percentile of message intervals.
func (s *StatsAnalyzer) P90() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.p90
}

// ModeSwitched notifies the StatsAnalyzer that the behavior mode has switched.
func (s *StatsAnalyzer) ModeSwitched() {
	s.modeSwitched <- true
}

// Run starts the stats analyzer with context support
func (s *StatsAnalyzer) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.updateBaselines()
		case <-s.modeSwitched:
			s.reset()
			s.updateBaselines()
		}
	}
}

// GetTickerInterval returns the interval for periodic execution
func (s *StatsAnalyzer) GetTickerInterval() string {
	return "30s"
}

// RunWithContext starts the stats analyzer's processing loop with context support for graceful shutdown.
func (s *StatsAnalyzer) RunWithContext(ctx context.Context, ticker *time.Ticker) {
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateBaselines()
		case <-s.modeSwitched:
			s.reset()
			s.updateBaselines()
		}
	}
}

// updateBaselines calculates the P50 and P90 message intervals.
func (s *StatsAnalyzer) updateBaselines() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.messageIntervals) < s.minDataPoints {
		return
	}

	p50, err := stats.Percentile(s.messageIntervals, 50)
	if err != nil {
		// Handle error
		return
	}
	s.p50 = time.Duration(p50) * time.Millisecond

	p90, err := stats.Percentile(s.messageIntervals, 90)
	if err != nil {
		// Handle error
		return
	}
	s.p90 = time.Duration(p90) * time.Millisecond
}

// GetCurrentBaseline returns the current P50 and P90 baselines in milliseconds.
func (s *StatsAnalyzer) GetCurrentBaseline() (p50Ms, p90Ms int64) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.p50.Milliseconds(), s.p90.Milliseconds()
}

// reset clears the collected message intervals.
func (s *StatsAnalyzer) reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.messageIntervals = make([]float64, 0, 10000)
}

// GetUserRequestCount returns the number of requests for a user in a time window
func (s *StatsAnalyzer) GetUserRequestCount(userID string, window time.Duration) int {
	// Simplified implementation for interface compliance
	return 0
}

// GetAverageResponseTime returns the average response time
func (s *StatsAnalyzer) GetAverageResponseTime() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.p50 // Use P50 as average response time approximation
}

// GetErrorRate returns the current error rate
func (s *StatsAnalyzer) GetErrorRate() float64 {
	// Simplified implementation for interface compliance
	return 0.0
}

// GetTotalRequests returns the total number of requests processed
func (s *StatsAnalyzer) GetTotalRequests() int64 {
	// Simplified implementation for interface compliance
	return 0
}

// GetActiveConnections returns the number of active connections
func (s *StatsAnalyzer) GetActiveConnections() int {
	// Simplified implementation for interface compliance
	return 0
}

// RecordRequest records a request with processing time and success status
func (s *StatsAnalyzer) RecordRequest(userID string, processingTime time.Duration, success bool) {
	// For now, just record the processing time as a message interval
	s.RecordMessageInterval(processingTime)
}

// GetSystemLoad returns the current system load
func (s *StatsAnalyzer) GetSystemLoad() float64 {
	// Simplified implementation for interface compliance
	return 0.0
}

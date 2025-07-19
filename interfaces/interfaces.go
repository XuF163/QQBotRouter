package interfaces

import (
	"context"
	"net/http"
	"time"
)

// StatProvider defines the interface for providing statistical data
type StatProvider interface {
	// GetUserRequestCount returns the number of requests for a user in a time window
	GetUserRequestCount(userID string, window time.Duration) int

	// GetAverageResponseTime returns the average response time
	GetAverageResponseTime() time.Duration

	// GetErrorRate returns the current error rate (0.0 to 1.0)
	GetErrorRate() float64

	// GetTotalRequests returns the total number of requests processed
	GetTotalRequests() int64

	// GetActiveConnections returns the number of active connections
	GetActiveConnections() int

	// RecordRequest records a new request with its processing time and success status
	RecordRequest(userID string, processingTime time.Duration, success bool)

	// GetSystemLoad returns the current system load (0.0 to 1.0)
	GetSystemLoad() float64

	// P50 returns the 50th percentile of message intervals
	P50() time.Duration

	// P90 returns the 90th percentile of message intervals
	P90() time.Duration
}

// QoSProvider defines the interface for QoS management
type QoSProvider interface {
	// ShouldThrottle determines if a request should be throttled
	ShouldThrottle(userID string, priority int) bool

	// UpdateMetrics updates QoS metrics with processing results
	UpdateMetrics(processingTime time.Duration, success bool)

	// RecordUserActivity records user activity for ML training
	RecordUserActivity(userID string, priority int, timestamp time.Time)

	// UpdateUserStats updates user behavior statistics
	UpdateUserStats(userID string, messageSize int)

	// GetCurrentLoad returns the current system load
	GetCurrentLoad() float64
}

// SchedulerProvider defines the interface for request scheduling
type SchedulerProvider interface {
	// Submit submits a request for processing
	Submit(ctx context.Context, body []byte, header http.Header, bot interface{}, logger interface{}) bool

	// GetQueueSize returns the current queue size
	GetQueueSize() int

	// GetProcessingRate returns the current processing rate (requests per second)
	GetProcessingRate() float64
}

// LoadProvider defines the interface for load monitoring
type LoadProvider interface {
	// Get returns the current load count
	Get() int64

	// Increment increments the load counter
	Increment()

	// Decrement decrements the load counter
	Decrement()
}

// Observer defines the interface for observing system metrics
type Observer interface {
	// RecordLatency records a new request latency
	RecordLatency(latency time.Duration)

	// HighLoadThreshold returns the current high-load threshold
	HighLoadThreshold() time.Duration
}

// ConfigProvider defines the interface for configuration management
type ConfigProvider interface {
	// GetQoSConfig returns QoS configuration
	GetQoSConfig() interface{}

	// GetSchedulerConfig returns scheduler configuration
	GetSchedulerConfig() interface{}

	// IsHotReloadEnabled returns whether hot reload is enabled
	IsHotReloadEnabled() bool

	// Reload reloads the configuration
	Reload() error
}

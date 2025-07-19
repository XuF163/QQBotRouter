package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/stats"
)

// AdaptiveThrottler provides adaptive request throttling based on dynamic baselines.
type AdaptiveThrottler struct {
	statsAnalyzer   *stats.StatsAnalyzer
	config          *config.AdaptiveThrottling
	lastRequestTime map[string]time.Time
	mutex           sync.RWMutex
	logger          *zap.Logger
}

// NewAdaptiveThrottler creates a new AdaptiveThrottler.
func NewAdaptiveThrottler(statsAnalyzer *stats.StatsAnalyzer, config *config.AdaptiveThrottling, logger *zap.Logger) *AdaptiveThrottler {
	return &AdaptiveThrottler{
		statsAnalyzer:   statsAnalyzer,
		config:          config,
		lastRequestTime: make(map[string]time.Time),
		logger:          logger,
	}
}

// getUserKey extracts a unique identifier from the request for throttling purposes.
func (at *AdaptiveThrottler) getUserKey(r *http.Request) string {
	// Try to get user identifier from headers or IP
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.Header.Get("X-Forwarded-For")
	}
	if userID == "" {
		userID = r.RemoteAddr
	}

	// Create a hash to anonymize the user identifier
	hash := sha256.Sum256([]byte(userID))
	return hex.EncodeToString(hash[:])
}

// shouldThrottle determines if a request should be throttled based on dynamic baselines.
func (at *AdaptiveThrottler) shouldThrottle(userKey string) bool {
	if !at.config.Enabled {
		return false
	}

	at.mutex.RLock()
	lastTime, exists := at.lastRequestTime[userKey]
	at.mutex.RUnlock()

	if !exists {
		// First request from this user, allow it
		return false
	}

	// Get dynamic baseline from stats analyzer
	p50, p90 := at.statsAnalyzer.GetCurrentBaseline()
	minInterval := time.Duration(at.config.MinRequestInterval) * time.Millisecond

	// Calculate adaptive interval based on current baseline
	adaptiveInterval := minInterval
	if p50 > 0 {
		// Use P50 as the baseline for normal users
		adaptiveInterval = time.Duration(p50) * time.Millisecond
		if adaptiveInterval < minInterval {
			adaptiveInterval = minInterval
		}
	}

	timeSinceLastRequest := time.Since(lastTime)
	return timeSinceLastRequest < adaptiveInterval
}

// Throttle is a middleware that throttles requests based on the dynamic baseline.
func (at *AdaptiveThrottler) Throttle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userKey := at.getUserKey(r)

		if at.shouldThrottle(userKey) {
			at.logger.Debug("Request throttled",
				zap.String("user_key", userKey[:8]), // Only log first 8 chars for privacy
				zap.String("path", r.URL.Path))

			w.Header().Set("Retry-After", "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Update last request time
		at.mutex.Lock()
		at.lastRequestTime[userKey] = time.Now()
		at.mutex.Unlock()

		// Clean up old entries periodically (simple cleanup strategy)
		go at.cleanupOldEntries()

		next.ServeHTTP(w, r)
	})
}

// cleanupOldEntries removes old entries from the lastRequestTime map to prevent memory leaks.
func (at *AdaptiveThrottler) cleanupOldEntries() {
	at.mutex.Lock()
	defer at.mutex.Unlock()

	// Remove entries older than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	for key, lastTime := range at.lastRequestTime {
		if lastTime.Before(cutoff) {
			delete(at.lastRequestTime, key)
		}
	}
}

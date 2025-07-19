package qos

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"qqbotrouter/config"
	"qqbotrouter/interfaces"
)

// QoSManager manages Quality of Service policies
type QoSManager struct {
	qosConfig          *config.QoSConfig
	loadProvider       interfaces.LoadProvider
	statsProvider      interfaces.StatProvider
	observer           interfaces.Observer
	logger             *zap.Logger
	mu                 sync.RWMutex
	throttlingStrategy ThrottlingStrategy // Strategy for adaptive throttling

	// Circuit breaker state
	circuitOpen     bool
	circuitOpenTime time.Time
	failureCount    int

	// Adaptive throttling state
	throttleLevel  float64 // 0.0 to 1.0
	lastAdjustment time.Time

	// Performance metrics
	responseTimeP50 time.Duration
	responseTimeP90 time.Duration
	throughput      float64
}

// NewQoSManager creates a new QoS manager
func NewQoSManager(qosConfig *config.QoSConfig, loadProvider interfaces.LoadProvider, statsProvider interfaces.StatProvider, observer interfaces.Observer, logger *zap.Logger) *QoSManager {
	return &QoSManager{
		qosConfig:          qosConfig,
		loadProvider:       loadProvider,
		statsProvider:      statsProvider,
		observer:           observer,
		logger:             logger,
		throttleLevel:      0.0,
		lastAdjustment:     time.Now(),
		throttlingStrategy: NewAdaptiveThrottlingStrategy(0.7, 0.3), // Default strategy
	}
}

// ShouldThrottle determines if a request should be throttled
func (qm *QoSManager) ShouldThrottle(userID string, priority int) bool {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Check circuit breaker
	if qm.isCircuitOpen() {
		return true
	}

	// Check adaptive throttling
	if qm.shouldAdaptiveThrottle(priority) {
		return true
	}

	return false
}

// isCircuitOpen checks if circuit breaker is open
func (qm *QoSManager) isCircuitOpen() bool {
	if !qm.qosConfig.CircuitBreaker.Enabled {
		return false
	}

	if qm.circuitOpen {
		// Check if we should try to close the circuit
		recoveryTimeout := qm.qosConfig.ParseDuration(qm.qosConfig.CircuitBreaker.RecoveryTimeout)
		if time.Since(qm.circuitOpenTime) > recoveryTimeout {
			qm.circuitOpen = false
			qm.failureCount = 0
			qm.logger.Info("Circuit breaker closed, attempting recovery")
			return false
		}
		return true
	}

	return false
}

// shouldAdaptiveThrottle checks if request should be throttled based on adaptive policy
func (qm *QoSManager) shouldAdaptiveThrottle(priority int) bool {
	if !qm.qosConfig.AdaptiveThrottling.Enabled {
		return false
	}

	currentLoad := qm.loadProvider.Get()
	maxLoad := qm.qosConfig.SystemLimits.MaxLoad

	// Calculate load ratio
	loadRatio := float64(currentLoad) / float64(maxLoad)

	// High priority requests have lower throttle probability
	throttleProbability := qm.throttleLevel * (1.0 - float64(priority)/10.0)

	// Increase throttle probability under high load
	if loadRatio > qm.qosConfig.SystemLimits.HighLoadThreshold {
		throttleProbability *= (1.0 + loadRatio)
	}

	// Simple probability-based throttling
	return throttleProbability > 0.5
}

// UpdateMetrics updates QoS metrics and adjusts policies
func (qm *QoSManager) UpdateMetrics(responseTime time.Duration, success bool) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Update circuit breaker state
	qm.updateCircuitBreaker(success)

	// Update adaptive throttling
	qm.updateAdaptiveThrottling(responseTime)

	// Update performance metrics
	qm.updatePerformanceMetrics(responseTime)
}

// updateCircuitBreaker updates circuit breaker state based on request success
func (qm *QoSManager) updateCircuitBreaker(success bool) {
	if !qm.qosConfig.CircuitBreaker.Enabled {
		return
	}

	if success {
		qm.failureCount = 0
	} else {
		qm.failureCount++
		if qm.failureCount >= qm.qosConfig.CircuitBreaker.FailureThreshold {
			qm.circuitOpen = true
			qm.circuitOpenTime = time.Now()
			qm.logger.Warn("Circuit breaker opened due to high failure rate",
				zap.Int("failure_count", qm.failureCount))
		}
	}
}

// updateAdaptiveThrottling adjusts throttle level using the configured strategy
func (qm *QoSManager) updateAdaptiveThrottling(responseTime time.Duration) {
	if !qm.qosConfig.AdaptiveThrottling.Enabled {
		return
	}

	// Only adjust throttling every few seconds to avoid oscillation
	adjustmentInterval := qm.qosConfig.SystemLimits.AdjustmentInterval
	if adjustmentDuration, err := time.ParseDuration(adjustmentInterval); err == nil {
		if time.Since(qm.lastAdjustment) < adjustmentDuration {
			return
		}
	} else if time.Since(qm.lastAdjustment) < 5*time.Second {
		return
	}

	// Use strategy pattern for throttling calculation
	newInterval, err := qm.throttlingStrategy.UpdateThrottling(qm.qosConfig, qm.observer, qm.statsProvider)
	if err != nil {
		qm.logger.Error("Failed to update throttling", zap.Error(err))
		return
	}

	// Convert interval to throttle level (0.0 to 1.0)
	baseInterval := time.Duration(qm.qosConfig.AdaptiveThrottling.BaseInterval) * time.Millisecond
	maxInterval := time.Duration(qm.qosConfig.AdaptiveThrottling.MaxInterval) * time.Millisecond

	if newInterval <= baseInterval {
		qm.throttleLevel = 0.0
	} else if newInterval >= maxInterval {
		qm.throttleLevel = 1.0
	} else {
		// Linear interpolation between base and max interval
		qm.throttleLevel = float64(newInterval-baseInterval) / float64(maxInterval-baseInterval)
	}

	qm.logger.Info("Updated throttle level using strategy",
		zap.Float64("throttle_level", qm.throttleLevel),
		zap.Duration("new_interval", newInterval),
		zap.Duration("response_time", responseTime))

	qm.lastAdjustment = time.Now()
}

// SetThrottlingStrategy allows changing the throttling strategy
func (qm *QoSManager) SetThrottlingStrategy(strategy ThrottlingStrategy) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.throttlingStrategy = strategy
}

// updatePerformanceMetrics updates performance tracking metrics
func (qm *QoSManager) updatePerformanceMetrics(responseTime time.Duration) {
	// Simple moving average for response times
	// In a real implementation, this would use more sophisticated metrics
	qm.responseTimeP90 = (qm.responseTimeP90*9 + responseTime) / 10
	qm.responseTimeP50 = (qm.responseTimeP50*9 + responseTime) / 10
}

// GetMetrics returns current QoS metrics
func (qm *QoSManager) GetMetrics() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	return map[string]interface{}{
		"throttle_level":    qm.throttleLevel,
		"circuit_open":      qm.circuitOpen,
		"failure_count":     qm.failureCount,
		"current_load":      qm.loadProvider.Get(),
		"response_time_p50": qm.responseTimeP50.Milliseconds(),
		"response_time_p90": qm.responseTimeP90.Milliseconds(),
		"stats_p50":         qm.statsProvider.P50().Milliseconds(),
		"stats_p90":         qm.statsProvider.P90().Milliseconds(),
	}
}

// Run starts the QoS manager background processes
func (qm *QoSManager) Run(ctx context.Context) error {
	return qm.monitoringLoop(ctx)
}

// Start starts the QoS manager background processes (deprecated, use Run instead)
func (qm *QoSManager) Start(ctx context.Context) {
	go qm.monitoringLoop(ctx)
}

// monitoringLoop runs periodic QoS monitoring and adjustments
func (qm *QoSManager) monitoringLoop(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			qm.performPeriodicAdjustments()
		}
	}
}

// performPeriodicAdjustments performs periodic QoS policy adjustments
func (qm *QoSManager) performPeriodicAdjustments() {
	metrics := qm.GetMetrics()
	qm.logger.Debug("QoS metrics update", zap.Any("metrics", metrics))

	// Log important state changes
	if qm.circuitOpen {
		qm.logger.Warn("Circuit breaker is open", zap.Any("metrics", metrics))
	}

	if qm.throttleLevel > 0.5 {
		qm.logger.Warn("High throttle level detected", zap.Any("metrics", metrics))
	}
}

// UpdateConfig updates the QoS configuration during hot reload
func (qm *QoSManager) UpdateConfig(newConfig *config.QoSConfig) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	oldConfig := qm.qosConfig
	qm.qosConfig = newConfig

	// Reset circuit breaker if configuration changed
	if oldConfig.CircuitBreaker.Enabled != newConfig.CircuitBreaker.Enabled ||
		oldConfig.CircuitBreaker.FailureThreshold != newConfig.CircuitBreaker.FailureThreshold {
		qm.circuitOpen = false
		qm.failureCount = 0
		qm.logger.Info("Circuit breaker configuration updated, resetting state")
	}

	// Reset adaptive throttling if configuration changed
	if oldConfig.AdaptiveThrottling.Enabled != newConfig.AdaptiveThrottling.Enabled ||
		oldConfig.AdaptiveThrottling.BaseThreshold != newConfig.AdaptiveThrottling.BaseThreshold ||
		oldConfig.AdaptiveThrottling.MaxThreshold != newConfig.AdaptiveThrottling.MaxThreshold {
		qm.throttleLevel = 0.0
		qm.lastAdjustment = time.Now()
		qm.logger.Info("Adaptive throttling configuration updated, resetting state")
	}

	// Log configuration update
	qm.logger.Info("QoS configuration updated",
		zap.Bool("circuit_breaker_enabled", newConfig.CircuitBreaker.Enabled),
		zap.Bool("adaptive_throttling_enabled", newConfig.AdaptiveThrottling.Enabled),
		zap.Int("max_load", newConfig.SystemLimits.MaxLoad))
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

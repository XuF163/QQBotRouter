package qos

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"qqbotrouter/config"
	"qqbotrouter/load"
	"qqbotrouter/stats"
)

// QoSManager manages Quality of Service policies
type QoSManager struct {
	config        *config.Config
	loadCounter   *load.Counter
	statsAnalyzer *stats.StatsAnalyzer
	logger        *zap.Logger
	mu            sync.RWMutex

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
func NewQoSManager(cfg *config.Config, loadCounter *load.Counter, statsAnalyzer *stats.StatsAnalyzer, logger *zap.Logger) *QoSManager {
	return &QoSManager{
		config:         cfg,
		loadCounter:    loadCounter,
		statsAnalyzer:  statsAnalyzer,
		logger:         logger,
		throttleLevel:  0.0,
		lastAdjustment: time.Now(),
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
	if !qm.config.IntelligentSchedulingPolicy.CircuitBreaker.Enabled {
		return false
	}

	if qm.circuitOpen {
		// Check if we should try to close the circuit
		if time.Since(qm.circuitOpenTime) > time.Duration(qm.config.IntelligentSchedulingPolicy.CircuitBreaker.RecoveryTime)*time.Second {
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
	if !qm.config.IntelligentSchedulingPolicy.AdaptiveThrottling.Enabled {
		return false
	}

	currentLoad := qm.loadCounter.Get()
	maxLoad := qm.config.QoS.SystemLimits.MaxLoad

	// Calculate load ratio
	loadRatio := float64(currentLoad) / float64(maxLoad)

	// High priority requests have lower throttle probability
	throttleProbability := qm.throttleLevel * (1.0 - float64(priority)/10.0)

	// Increase throttle probability under high load
	if loadRatio > qm.config.QoS.SystemLimits.HighLoadThreshold {
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
	if !qm.config.IntelligentSchedulingPolicy.CircuitBreaker.Enabled {
		return
	}

	if success {
		qm.failureCount = 0
	} else {
		qm.failureCount++
		if qm.failureCount >= qm.config.IntelligentSchedulingPolicy.CircuitBreaker.MinRequestsForEvaluation {
			qm.circuitOpen = true
			qm.circuitOpenTime = time.Now()
			qm.logger.Warn("Circuit breaker opened due to high failure rate",
				zap.Int("failure_count", qm.failureCount))
		}
	}
}

// updateAdaptiveThrottling adjusts throttle level based on system performance
func (qm *QoSManager) updateAdaptiveThrottling(responseTime time.Duration) {
	if !qm.config.IntelligentSchedulingPolicy.AdaptiveThrottling.Enabled {
		return
	}

	// Only adjust throttling every few seconds to avoid oscillation
	adjustmentInterval := qm.config.QoS.SystemLimits.AdjustmentInterval
	if adjustmentDuration, err := time.ParseDuration(adjustmentInterval); err == nil {
		if time.Since(qm.lastAdjustment) < adjustmentDuration {
			return
		}
	} else if time.Since(qm.lastAdjustment) < 5*time.Second {
		return
	}

	currentLoad := qm.loadCounter.Get()
	maxLoad := qm.config.QoS.SystemLimits.MaxLoad
	loadRatio := float64(currentLoad) / float64(maxLoad)

	// Get baseline from stats analyzer
	p90Baseline := qm.statsAnalyzer.P90()

	// Adjust throttle level based on load and response time
	throttleAdjustment := qm.config.QoS.SystemLimits.ThrottleAdjustment
	if loadRatio > qm.config.QoS.SystemLimits.HighLoadThreshold || (p90Baseline > 0 && responseTime > p90Baseline*2) {
		// Increase throttling
		qm.throttleLevel = min(1.0, qm.throttleLevel+throttleAdjustment)
		qm.logger.Info("Increased throttle level",
			zap.Float64("throttle_level", qm.throttleLevel),
			zap.Float64("load_ratio", loadRatio))
	} else if loadRatio < qm.config.QoS.SystemLimits.LowLoadThreshold && (p90Baseline == 0 || responseTime < p90Baseline) {
		// Decrease throttling
		qm.throttleLevel = max(0.0, qm.throttleLevel-throttleAdjustment/2)
		qm.logger.Info("Decreased throttle level",
			zap.Float64("throttle_level", qm.throttleLevel),
			zap.Float64("load_ratio", loadRatio))
	}

	qm.lastAdjustment = time.Now()
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
		"current_load":      qm.loadCounter.Get(),
		"response_time_p50": qm.responseTimeP50.Milliseconds(),
		"response_time_p90": qm.responseTimeP90.Milliseconds(),
		"stats_p50":         qm.statsAnalyzer.P50().Milliseconds(),
		"stats_p90":         qm.statsAnalyzer.P90().Milliseconds(),
	}
}

// Start starts the QoS manager background processes
func (qm *QoSManager) Start(ctx context.Context) {
	go qm.monitoringLoop(ctx)
}

// monitoringLoop runs periodic QoS monitoring and adjustments
func (qm *QoSManager) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
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

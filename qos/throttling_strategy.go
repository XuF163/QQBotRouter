package qos

import (
	"qqbotrouter/config"
	"qqbotrouter/interfaces"
	"time"
)

// ThrottlingStrategy defines the interface for adaptive throttling strategies
type ThrottlingStrategy interface {
	UpdateThrottling(config *config.QoSConfig, observer interfaces.Observer, stats interfaces.StatProvider) (time.Duration, error)
}

// LoadBasedThrottlingStrategy adjusts throttling based on system load
type LoadBasedThrottlingStrategy struct{}

func (s *LoadBasedThrottlingStrategy) UpdateThrottling(config *config.QoSConfig, observer interfaces.Observer, statsAnalyzer interfaces.StatProvider) (time.Duration, error) {
	currentLoad := observer.GetCurrentLoad()
	baseInterval := time.Duration(config.AdaptiveThrottling.BaseThreshold) * time.Millisecond
	maxInterval := time.Duration(config.AdaptiveThrottling.MaxThreshold) * time.Millisecond

	// Calculate throttling based on load
	if currentLoad > config.DynamicLoadBalancing.LoadThreshold {
		// High load: increase throttling
		loadFactor := currentLoad / config.DynamicLoadBalancing.LoadThreshold
		newInterval := time.Duration(float64(baseInterval) * loadFactor)
		if newInterval > maxInterval {
			newInterval = maxInterval
		}
		return newInterval, nil
	}

	// Normal load: use base interval
	return baseInterval, nil
}

// ResponseTimeBasedThrottlingStrategy adjusts throttling based on response times
type ResponseTimeBasedThrottlingStrategy struct{}

func (s *ResponseTimeBasedThrottlingStrategy) UpdateThrottling(config *config.QoSConfig, observer interfaces.Observer, statsAnalyzer interfaces.StatProvider) (time.Duration, error) {
	baseInterval := time.Duration(config.AdaptiveThrottling.BaseThreshold) * time.Millisecond
	maxInterval := time.Duration(config.AdaptiveThrottling.MaxThreshold) * time.Millisecond

	// Get average response time from stats
	avgResponseTime := statsAnalyzer.GetAverageResponseTime()

	// Calculate throttling based on response time
	targetResponseTime := 1000 * time.Millisecond // 1 second target
	if avgResponseTime > targetResponseTime {
		// Slow responses: increase throttling
		responseFactor := float64(avgResponseTime) / float64(targetResponseTime)
		newInterval := time.Duration(float64(baseInterval) * responseFactor)
		if newInterval > maxInterval {
			newInterval = maxInterval
		}
		return newInterval, nil
	}

	// Fast responses: use base interval or reduce
	return baseInterval, nil
}

// AdaptiveThrottlingStrategy combines multiple factors for throttling decisions
type AdaptiveThrottlingStrategy struct {
	loadStrategy         *LoadBasedThrottlingStrategy
	responseTimeStrategy *ResponseTimeBasedThrottlingStrategy
	loadWeight           float64
	responseTimeWeight   float64
}

func NewAdaptiveThrottlingStrategy(loadWeight, responseTimeWeight float64) *AdaptiveThrottlingStrategy {
	return &AdaptiveThrottlingStrategy{
		loadStrategy:         &LoadBasedThrottlingStrategy{},
		responseTimeStrategy: &ResponseTimeBasedThrottlingStrategy{},
		loadWeight:           loadWeight,
		responseTimeWeight:   responseTimeWeight,
	}
}

func (s *AdaptiveThrottlingStrategy) UpdateThrottling(config *config.QoSConfig, observer interfaces.Observer, statsAnalyzer interfaces.StatProvider) (time.Duration, error) {
	loadInterval, err := s.loadStrategy.UpdateThrottling(config, observer, statsAnalyzer)
	if err != nil {
		return 0, err
	}

	responseInterval, err := s.responseTimeStrategy.UpdateThrottling(config, observer, statsAnalyzer)
	if err != nil {
		return 0, err
	}

	// Weighted combination - take the higher interval for safety
	weightedInterval := time.Duration(
		float64(loadInterval)*s.loadWeight + float64(responseInterval)*s.responseTimeWeight,
	)

	// Ensure we don't go below base interval
	baseInterval := time.Duration(config.AdaptiveThrottling.BaseThreshold) * time.Millisecond
	if weightedInterval < baseInterval {
		weightedInterval = baseInterval
	}

	return weightedInterval, nil
}

func (s *AdaptiveThrottlingStrategy) CalculateThrottleLevel(load float64, responseTime time.Duration, config *config.QoSConfig, statsAnalyzer interfaces.StatProvider) float64 {
	// Get baseline metrics
	p90Baseline := statsAnalyzer.P90()
	p95HighLoad := time.Duration(float64(p90Baseline) * 1.5) // 50% higher than P90

	// Calculate load factor based on system limits
	loadFactor := load / config.SystemLimits.HighLoadThreshold
	if loadFactor > 1.0 {
		loadFactor = 1.0
	}

	// Calculate response time factor
	responseTimeFactor := 0.0
	if p90Baseline > 0 {
		responseTimeFactor = float64(responseTime) / float64(p90Baseline)
		if responseTime > p95HighLoad {
			responseTimeFactor *= 1.5 // Penalty for high response times
		}
	}

	// Weighted combination
	throttleLevel := s.loadWeight*loadFactor + s.responseTimeWeight*responseTimeFactor

	// Apply error rate consideration using available stats
	errorRate := statsAnalyzer.GetErrorRate()
	if errorRate > 0.05 { // 5% error rate threshold
		throttleLevel *= (1.0 + errorRate)
	}

	// Apply average response time adjustment
	avgResponseTime := statsAnalyzer.GetAverageResponseTime()
	if avgResponseTime > p90Baseline {
		throttleLevel *= 1.2
	}

	// Ensure throttle level is within bounds
	if throttleLevel < 0 {
		throttleLevel = 0
	} else if throttleLevel > 1.0 {
		throttleLevel = 1.0
	}

	return throttleLevel
}

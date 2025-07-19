package config

import "time"

// QoSConfig contains QoS-specific configuration
type QoSConfig struct {
	// Dynamic Load Balancing
	DynamicLoadBalancing struct {
		Enabled            bool    `yaml:"enabled"`
		LoadThreshold      float64 `yaml:"load_threshold"`
		AdjustmentFactor   float64 `yaml:"adjustment_factor"`
		MonitoringInterval string  `yaml:"monitoring_interval"`
	} `yaml:"dynamic_load_balancing"`

	// Adaptive Throttling
	AdaptiveThrottling struct {
		Enabled        bool    `yaml:"enabled"`
		BaseThreshold  int     `yaml:"base_threshold"`
		MaxThreshold   int     `yaml:"max_threshold"`
		AdaptationRate float64 `yaml:"adaptation_rate"`
		CooldownPeriod string  `yaml:"cooldown_period"`
	} `yaml:"adaptive_throttling"`

	// Circuit Breaker
	CircuitBreaker struct {
		Enabled          bool   `yaml:"enabled"`
		FailureThreshold int    `yaml:"failure_threshold"`
		RecoveryTimeout  string `yaml:"recovery_timeout"`
		HalfOpenRequests int    `yaml:"half_open_requests"`
	} `yaml:"circuit_breaker"`

	// Performance Monitoring
	PerformanceMonitoring struct {
		Enabled          bool   `yaml:"enabled"`
		MetricsInterval  string `yaml:"metrics_interval"`
		HistoryRetention string `yaml:"history_retention"`
	} `yaml:"performance_monitoring"`
}

// GetDefaultQoSConfig returns default QoS configuration
func GetDefaultQoSConfig() QoSConfig {
	return QoSConfig{
		DynamicLoadBalancing: struct {
			Enabled            bool    `yaml:"enabled"`
			LoadThreshold      float64 `yaml:"load_threshold"`
			AdjustmentFactor   float64 `yaml:"adjustment_factor"`
			MonitoringInterval string  `yaml:"monitoring_interval"`
		}{
			Enabled:            true,
			LoadThreshold:      0.8,
			AdjustmentFactor:   0.1,
			MonitoringInterval: "30s",
		},
		AdaptiveThrottling: struct {
			Enabled        bool    `yaml:"enabled"`
			BaseThreshold  int     `yaml:"base_threshold"`
			MaxThreshold   int     `yaml:"max_threshold"`
			AdaptationRate float64 `yaml:"adaptation_rate"`
			CooldownPeriod string  `yaml:"cooldown_period"`
		}{
			Enabled:        true,
			BaseThreshold:  100,
			MaxThreshold:   1000,
			AdaptationRate: 0.05,
			CooldownPeriod: "5m",
		},
		CircuitBreaker: struct {
			Enabled          bool   `yaml:"enabled"`
			FailureThreshold int    `yaml:"failure_threshold"`
			RecoveryTimeout  string `yaml:"recovery_timeout"`
			HalfOpenRequests int    `yaml:"half_open_requests"`
		}{
			Enabled:          true,
			FailureThreshold: 5,
			RecoveryTimeout:  "30s",
			HalfOpenRequests: 3,
		},
		PerformanceMonitoring: struct {
			Enabled          bool   `yaml:"enabled"`
			MetricsInterval  string `yaml:"metrics_interval"`
			HistoryRetention string `yaml:"history_retention"`
		}{
			Enabled:          true,
			MetricsInterval:  "10s",
			HistoryRetention: "24h",
		},
	}
}

// ParseDuration safely parses duration strings
func (q *QoSConfig) ParseDuration(durationStr string) time.Duration {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 30 * time.Second // Default fallback
	}
	return duration
}

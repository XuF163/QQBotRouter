package config

import "time"

// QoSConfig contains QoS-specific configuration
type QoSConfig struct {
	// Basic QoS Settings
	MaxConcurrentRequests int     `yaml:"max_concurrent_requests"`
	ThrottleThreshold     float64 `yaml:"throttle_threshold"`
	RecoveryTimeout       int     `yaml:"recovery_timeout"`
	FailureThreshold      int     `yaml:"failure_threshold"`

	// System Limits
	SystemLimits struct {
		MaxLoad            int     `yaml:"max_load"`
		HighLoadThreshold  float64 `yaml:"high_load_threshold"`
		LowLoadThreshold   float64 `yaml:"low_load_threshold"`
		ThrottleAdjustment float64 `yaml:"throttle_adjustment"`
		AdjustmentInterval string  `yaml:"adjustment_interval"`
	} `yaml:"system_limits"`

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
		BaseInterval   int     `yaml:"base_interval"`
		MaxInterval    int     `yaml:"max_interval"`
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

	// Hot Reload
	HotReload struct {
		Enabled       bool   `yaml:"enabled"`
		CheckInterval string `yaml:"check_interval"`
	} `yaml:"hot_reload"`

	// Request Timeouts
	RequestTimeouts struct {
		ForwardTimeout    string `yaml:"forward_timeout"`
		ProcessingTimeout string `yaml:"processing_timeout"`
		IdleCheckInterval string `yaml:"idle_check_interval"`
	} `yaml:"request_timeouts"`
}

// GetDefaultQoSConfig returns default QoS configuration
func GetDefaultQoSConfig() QoSConfig {
	return QoSConfig{
		MaxConcurrentRequests: 100,
		ThrottleThreshold:     0.7,
		RecoveryTimeout:       30,
		FailureThreshold:      5,
		SystemLimits: struct {
			MaxLoad            int     `yaml:"max_load"`
			HighLoadThreshold  float64 `yaml:"high_load_threshold"`
			LowLoadThreshold   float64 `yaml:"low_load_threshold"`
			ThrottleAdjustment float64 `yaml:"throttle_adjustment"`
			AdjustmentInterval string  `yaml:"adjustment_interval"`
		}{
			MaxLoad:            100,
			HighLoadThreshold:  0.8,
			LowLoadThreshold:   0.5,
			ThrottleAdjustment: 0.1,
			AdjustmentInterval: "5s",
		},
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
			BaseInterval   int     `yaml:"base_interval"`
			MaxInterval    int     `yaml:"max_interval"`
			BaseThreshold  int     `yaml:"base_threshold"`
			MaxThreshold   int     `yaml:"max_threshold"`
			AdaptationRate float64 `yaml:"adaptation_rate"`
			CooldownPeriod string  `yaml:"cooldown_period"`
		}{
			Enabled:        true,
			BaseInterval:   100,
			MaxInterval:    1000,
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
		HotReload: struct {
			Enabled       bool   `yaml:"enabled"`
			CheckInterval string `yaml:"check_interval"`
		}{
			Enabled:       false,
			CheckInterval: "30s",
		},
		RequestTimeouts: struct {
			ForwardTimeout    string `yaml:"forward_timeout"`
			ProcessingTimeout string `yaml:"processing_timeout"`
			IdleCheckInterval string `yaml:"idle_check_interval"`
		}{
			ForwardTimeout:    "10s",
			ProcessingTimeout: "12s",
			IdleCheckInterval: "10ms",
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

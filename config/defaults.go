package config

// DefaultValues contains all default configuration values in one place
var DefaultValues = struct {
	// Server settings
	HTTPSPort string
	HTTPPort  string
	LogLevel  string

	// QoS settings
	QoS struct {
		CircuitBreaker struct {
			Enabled          bool
			FailureThreshold int
			RecoveryTimeout  string
		}
		AdaptiveThrottling struct {
			Enabled       bool
			BaseInterval  int
			MaxInterval   int
			BaseThreshold float64
			MaxThreshold  float64
		}
		DynamicLoadBalancing struct {
			Enabled       bool
			LoadThreshold float64
		}
		SystemLimits struct {
			MaxLoad            int
			HighLoadThreshold  float64
			LowLoadThreshold   float64
			ThrottleAdjustment float64
			AdjustmentInterval string
		}
		RequestTimeouts struct {
			ProcessingTimeout string
			ForwardTimeout    string
			IdleCheckInterval string
		}
		HotReload struct {
			Enabled       bool
			CheckInterval string
		}
	}

	// Scheduler settings
	Scheduler struct {
		WorkerPoolSize   int
		PrioritySettings struct {
			BasePriority       int
			MinPriority        int
			MaxPriority        int
			HighLoadAdjustment int
			LowLoadAdjustment  int
			FastUserBonus      int
		}
		MessageClassification struct {
			Enabled          bool
			SpamDetection    bool
			SpamKeywords     []string
			PriorityKeywords []string
		}
		UserBehaviorAnalysis struct {
			Enabled                  bool
			MinDataPointsForBaseline int
			HighActivityThreshold    int
			FastResponseThreshold    int
			HighActivityBonus        int
			FastResponseBonus        int
		}
	}
}{
	// Initialize default values
	HTTPSPort: "8443",
	HTTPPort:  "8080",
	LogLevel:  "development",

	QoS: struct {
		CircuitBreaker struct {
			Enabled          bool
			FailureThreshold int
			RecoveryTimeout  string
		}
		AdaptiveThrottling struct {
			Enabled       bool
			BaseInterval  int
			MaxInterval   int
			BaseThreshold float64
			MaxThreshold  float64
		}
		DynamicLoadBalancing struct {
			Enabled       bool
			LoadThreshold float64
		}
		SystemLimits struct {
			MaxLoad            int
			HighLoadThreshold  float64
			LowLoadThreshold   float64
			ThrottleAdjustment float64
			AdjustmentInterval string
		}
		RequestTimeouts struct {
			ProcessingTimeout string
			ForwardTimeout    string
			IdleCheckInterval string
		}
		HotReload struct {
			Enabled       bool
			CheckInterval string
		}
	}{
		CircuitBreaker: struct {
			Enabled          bool
			FailureThreshold int
			RecoveryTimeout  string
		}{
			Enabled:          true,
			FailureThreshold: 5,
			RecoveryTimeout:  "30s",
		},
		AdaptiveThrottling: struct {
			Enabled       bool
			BaseInterval  int
			MaxInterval   int
			BaseThreshold float64
			MaxThreshold  float64
		}{
			Enabled:       true,
			BaseInterval:  100,
			MaxInterval:   2000,
			BaseThreshold: 0.3,
			MaxThreshold:  0.8,
		},
		DynamicLoadBalancing: struct {
			Enabled       bool
			LoadThreshold float64
		}{
			Enabled:       true,
			LoadThreshold: 80.0,
		},
		SystemLimits: struct {
			MaxLoad            int
			HighLoadThreshold  float64
			LowLoadThreshold   float64
			ThrottleAdjustment float64
			AdjustmentInterval string
		}{
			MaxLoad:            100,
			HighLoadThreshold:  0.8,
			LowLoadThreshold:   0.3,
			ThrottleAdjustment: 0.1,
			AdjustmentInterval: "5s",
		},
		RequestTimeouts: struct {
			ProcessingTimeout string
			ForwardTimeout    string
			IdleCheckInterval string
		}{
			ProcessingTimeout: "30s",
			ForwardTimeout:    "10s",
			IdleCheckInterval: "1s",
		},
		HotReload: struct {
			Enabled       bool
			CheckInterval string
		}{
			Enabled:       true,
			CheckInterval: "2s",
		},
	},

	Scheduler: struct {
		WorkerPoolSize   int
		PrioritySettings struct {
			BasePriority       int
			MinPriority        int
			MaxPriority        int
			HighLoadAdjustment int
			LowLoadAdjustment  int
			FastUserBonus      int
		}
		MessageClassification struct {
			Enabled          bool
			SpamDetection    bool
			SpamKeywords     []string
			PriorityKeywords []string
		}
		UserBehaviorAnalysis struct {
			Enabled                  bool
			MinDataPointsForBaseline int
			HighActivityThreshold    int
			FastResponseThreshold    int
			HighActivityBonus        int
			FastResponseBonus        int
		}
	}{
		WorkerPoolSize: 10,
		PrioritySettings: struct {
			BasePriority       int
			MinPriority        int
			MaxPriority        int
			HighLoadAdjustment int
			LowLoadAdjustment  int
			FastUserBonus      int
		}{
			BasePriority:       50,
			MinPriority:        1,
			MaxPriority:        100,
			HighLoadAdjustment: -10,
			LowLoadAdjustment:  5,
			FastUserBonus:      10,
		},
		MessageClassification: struct {
			Enabled          bool
			SpamDetection    bool
			SpamKeywords     []string
			PriorityKeywords []string
		}{
			Enabled:          true,
			SpamDetection:    true,
			SpamKeywords:     []string{"spam", "advertisement", "promotion"},
			PriorityKeywords: []string{"urgent", "emergency", "important"},
		},
		UserBehaviorAnalysis: struct {
			Enabled                  bool
			MinDataPointsForBaseline int
			HighActivityThreshold    int
			FastResponseThreshold    int
			HighActivityBonus        int
			FastResponseBonus        int
		}{
			Enabled:                  true,
			MinDataPointsForBaseline: 10,
			HighActivityThreshold:    100,
			FastResponseThreshold:    1000,
			HighActivityBonus:        15,
			FastResponseBonus:        10,
		},
	},
}

// GetDefaultValues returns the centralized default values
func GetDefaultValues() *struct {
	// Server settings
	Server struct {
		HTTPSPort string
		HTTPPort  string
		LogLevel  string
	}

	// QoS settings
	QoS struct {
		CircuitBreaker struct {
			Enabled          bool
			FailureThreshold int
			RecoveryTimeout  string
		}
		AdaptiveThrottling struct {
			Enabled       bool
			BaseInterval  int
			MaxInterval   int
			BaseThreshold float64
			MaxThreshold  float64
		}
		DynamicLoadBalancing struct {
			Enabled       bool
			LoadThreshold float64
		}
		SystemLimits struct {
			MaxLoad            int
			HighLoadThreshold  float64
			LowLoadThreshold   float64
			ThrottleAdjustment float64
			AdjustmentInterval string
		}
		RequestTimeouts struct {
			ProcessingTimeout string
			ForwardTimeout    string
			IdleCheckInterval string
		}
		HotReload struct {
			Enabled       bool
			CheckInterval string
		}
	}

	// Scheduler settings
	Scheduler struct {
		WorkerPoolSize   int
		PrioritySettings struct {
			BasePriority       int
			MinPriority        int
			MaxPriority        int
			HighLoadAdjustment int
			LowLoadAdjustment  int
			FastUserBonus      int
		}
		MessageClassification struct {
			Enabled          bool
			SpamDetection    bool
			SpamKeywords     []string
			PriorityKeywords []string
		}
		UserBehaviorAnalysis struct {
			Enabled                  bool
			MinDataPointsForBaseline int
			HighActivityThreshold    int
			FastResponseThreshold    int
			HighActivityBonus        int
			FastResponseBonus        int
		}
	}
} {
	return &struct {
		// Server settings
		Server struct {
			HTTPSPort string
			HTTPPort  string
			LogLevel  string
		}

		// QoS settings
		QoS struct {
			CircuitBreaker struct {
				Enabled          bool
				FailureThreshold int
				RecoveryTimeout  string
			}
			AdaptiveThrottling struct {
				Enabled       bool
				BaseInterval  int
				MaxInterval   int
				BaseThreshold float64
				MaxThreshold  float64
			}
			DynamicLoadBalancing struct {
				Enabled       bool
				LoadThreshold float64
			}
			SystemLimits struct {
				MaxLoad            int
				HighLoadThreshold  float64
				LowLoadThreshold   float64
				ThrottleAdjustment float64
				AdjustmentInterval string
			}
			RequestTimeouts struct {
				ProcessingTimeout string
				ForwardTimeout    string
				IdleCheckInterval string
			}
			HotReload struct {
				Enabled       bool
				CheckInterval string
			}
		}

		// Scheduler settings
		Scheduler struct {
			WorkerPoolSize   int
			PrioritySettings struct {
				BasePriority       int
				MinPriority        int
				MaxPriority        int
				HighLoadAdjustment int
				LowLoadAdjustment  int
				FastUserBonus      int
			}
			MessageClassification struct {
				Enabled          bool
				SpamDetection    bool
				SpamKeywords     []string
				PriorityKeywords []string
			}
			UserBehaviorAnalysis struct {
				Enabled                  bool
				MinDataPointsForBaseline int
				HighActivityThreshold    int
				FastResponseThreshold    int
				HighActivityBonus        int
				FastResponseBonus        int
			}
		}
	}{
		Server: struct {
			HTTPSPort string
			HTTPPort  string
			LogLevel  string
		}{
			HTTPSPort: "8443",
			HTTPPort:  "8080",
			LogLevel:  "development",
		},
		QoS:       DefaultValues.QoS,
		Scheduler: DefaultValues.Scheduler,
	}
}

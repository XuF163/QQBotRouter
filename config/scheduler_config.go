package config

// SchedulerConfig contains scheduler-specific configuration
type SchedulerConfig struct {
	// Worker Pool
	WorkerPoolSize int `yaml:"worker_pool_size"`

	// Priority Settings
	PrioritySettings struct {
		BasePriority       int `yaml:"base_priority"`
		MinPriority        int `yaml:"min_priority"`
		MaxPriority        int `yaml:"max_priority"`
		HighLoadAdjustment int `yaml:"high_load_adjustment"`
		LowLoadAdjustment  int `yaml:"low_load_adjustment"`
		FastUserBonus      int `yaml:"fast_user_bonus"`
	} `yaml:"priority_settings"`

	// Cognitive Scheduling
	CognitiveScheduling struct {
		Enabled             bool    `yaml:"enabled"`
		LearningRate        float64 `yaml:"learning_rate"`
		MemoryWindow        string  `yaml:"memory_window"`
		AdaptationThreshold float64 `yaml:"adaptation_threshold"`
	} `yaml:"cognitive_scheduling"`

	// Priority Queue
	PriorityQueue struct {
		MaxSize           int    `yaml:"max_size"`
		ProcessingTimeout string `yaml:"processing_timeout"`
		BatchSize         int    `yaml:"batch_size"`
	} `yaml:"priority_queue"`

	// User Behavior Analysis
	UserBehaviorAnalysis struct {
		Enabled                  bool   `yaml:"enabled"`
		AnalysisWindow           string `yaml:"analysis_window"`
		BehaviorThreshold        int    `yaml:"behavior_threshold"`
		MinDataPointsForBaseline int    `yaml:"min_data_points_for_baseline"`
	} `yaml:"user_behavior_analysis"`

	// Message Classification
	MessageClassification struct {
		Enabled          bool     `yaml:"enabled"`
		SpamDetection    bool     `yaml:"spam_detection"`
		PriorityKeywords []string `yaml:"priority_keywords"`
		SpamKeywords     []string `yaml:"spam_keywords"`
	} `yaml:"message_classification"`
}

// GetDefaultSchedulerConfig returns default scheduler configuration
func GetDefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		WorkerPoolSize: 10,
		PrioritySettings: struct {
			BasePriority       int `yaml:"base_priority"`
			MinPriority        int `yaml:"min_priority"`
			MaxPriority        int `yaml:"max_priority"`
			HighLoadAdjustment int `yaml:"high_load_adjustment"`
			LowLoadAdjustment  int `yaml:"low_load_adjustment"`
			FastUserBonus      int `yaml:"fast_user_bonus"`
		}{
			BasePriority:       5,
			MinPriority:        1,
			MaxPriority:        10,
			HighLoadAdjustment: -2,
			LowLoadAdjustment:  1,
			FastUserBonus:      2,
		},
		CognitiveScheduling: struct {
			Enabled             bool    `yaml:"enabled"`
			LearningRate        float64 `yaml:"learning_rate"`
			MemoryWindow        string  `yaml:"memory_window"`
			AdaptationThreshold float64 `yaml:"adaptation_threshold"`
		}{
			Enabled:             true,
			LearningRate:        0.01,
			MemoryWindow:        "1h",
			AdaptationThreshold: 0.1,
		},
		PriorityQueue: struct {
			MaxSize           int    `yaml:"max_size"`
			ProcessingTimeout string `yaml:"processing_timeout"`
			BatchSize         int    `yaml:"batch_size"`
		}{
			MaxSize:           10000,
			ProcessingTimeout: "30s",
			BatchSize:         10,
		},
		UserBehaviorAnalysis: struct {
			Enabled                  bool   `yaml:"enabled"`
			AnalysisWindow           string `yaml:"analysis_window"`
			BehaviorThreshold        int    `yaml:"behavior_threshold"`
			MinDataPointsForBaseline int    `yaml:"min_data_points_for_baseline"`
		}{
			Enabled:                  true,
			AnalysisWindow:           "15m",
			BehaviorThreshold:        50,
			MinDataPointsForBaseline: 100,
		},
		MessageClassification: struct {
			Enabled          bool     `yaml:"enabled"`
			SpamDetection    bool     `yaml:"spam_detection"`
			PriorityKeywords []string `yaml:"priority_keywords"`
			SpamKeywords     []string `yaml:"spam_keywords"`
		}{
			Enabled:          true,
			SpamDetection:    true,
			PriorityKeywords: []string{"紧急", "重要", "帮助", "问题", "错误", "urgent", "important", "help", "error", "issue"},
			SpamKeywords:     []string{"重复", "刷屏", "广告", "推广", "spam", "advertisement", "promotion"},
		},
	}
}

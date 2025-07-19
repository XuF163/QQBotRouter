package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	LogLevel                    string                       `yaml:"log_level"`
	HTTPSPort                   string                       `yaml:"https_port"`
	HTTPPort                    string                       `yaml:"http_port"`
	Bots                        map[string]BotConfig         `yaml:"bots"`
	IntelligentSchedulingPolicy *IntelligentSchedulingPolicy `yaml:"intelligent_scheduling_policy"`
}

// BotConfig represents individual bot configuration
type BotConfig struct {
	Secret      string                      `yaml:"secret"`
	ForwardTo   []string                    `yaml:"forward_to"`
	RegexRoutes map[string]RegexRouteConfig `yaml:"regex_routes"`
}

// RegexRouteConfig represents regex route configuration
type RegexRouteConfig struct {
	IsHash    bool     `yaml:"ishash,omitempty"`
	Endpoints []string `yaml:"endpoints,omitempty"`
	URLs      []string `yaml:",flow,omitempty"`
}

// IntelligentSchedulingPolicy represents the QoS policy configuration
type IntelligentSchedulingPolicy struct {
	Enabled                   bool                       `yaml:"enabled"`
	DynamicLoadTuning         *DynamicLoadTuning         `yaml:"dynamic_load_tuning"`
	AdaptiveThrottling        *AdaptiveThrottling        `yaml:"adaptive_throttling"`
	CognitiveScheduling       *CognitiveScheduling       `yaml:"cognitive_scheduling"`
	DynamicBaselineAnalysis   *DynamicBaselineAnalysis   `yaml:"dynamic_baseline_analysis"`
	BehavioralPatternLearning *BehavioralPatternLearning `yaml:"behavioral_pattern_learning"`
	PriorityQueue             *PriorityQueue             `yaml:"priority_queue"`
	PerformanceMonitoring     *PerformanceMonitoring     `yaml:"performance_monitoring"`
	HardwareAdaptive          *HardwareAdaptive          `yaml:"hardware_adaptive"`
	CircuitBreaker            *CircuitBreaker            `yaml:"circuit_breaker"`
	TrafficMirroring          *TrafficMirroring          `yaml:"traffic_mirroring"`
	HotReload                 *HotReload                 `yaml:"hot_reload"`
	GracefulShutdown          *GracefulShutdown          `yaml:"graceful_shutdown"`
	DebuggingAndVisualization *DebuggingAndVisualization `yaml:"debugging_and_visualization"`
	UserReputationSystem      *UserReputationSystem      `yaml:"user_reputation_system"`
	PluginArchitecture        *PluginArchitecture        `yaml:"plugin_architecture"`
	ABTesting                 *ABTesting                 `yaml:"ab_testing"`
	AlertingAndNotification   *AlertingAndNotification   `yaml:"alerting_and_notification"`
	APIVersioning             *APIVersioning             `yaml:"api_versioning"`
	SecurityAndCompliance     *SecurityAndCompliance     `yaml:"security_and_compliance"`
	ExperimentalFeatures      *ExperimentalFeatures      `yaml:"experimental_features"`
}

// DynamicLoadTuning represents the dynamic load tuning configuration
type DynamicLoadTuning struct {
	Enabled          bool `yaml:"enabled"`
	LatencyThreshold int  `yaml:"latency_threshold_ms"`
}

// AdaptiveThrottling represents the adaptive throttling configuration
type AdaptiveThrottling struct {
	Enabled            bool `yaml:"enabled"`
	MinRequestInterval int  `yaml:"min_request_interval_ms"`
}

// CognitiveScheduling represents the cognitive scheduling configuration
type CognitiveScheduling struct {
	WorkerPoolSize       int     `yaml:"worker_pool_size"`
	ModelRetrainInterval int     `yaml:"model_retrain_interval_hours"`
	FastUserSensitivity  float64 `yaml:"fast_user_sensitivity"`
	SpamUserSensitivity  float64 `yaml:"spam_user_sensitivity"`
}

// DynamicBaselineAnalysis represents the dynamic baseline analysis configuration
type DynamicBaselineAnalysis struct {
	Enabled                       bool    `yaml:"enabled"`
	InitialLearningDuration       int     `yaml:"initial_learning_duration_minutes"`
	PatternRecognitionSensitivity float64 `yaml:"pattern_recognition_sensitivity"`
	MinDataPointsForBaseline      int     `yaml:"min_data_points_for_baseline"`
}

// BehavioralPatternLearning represents the behavioral pattern learning configuration
type BehavioralPatternLearning struct {
	Enabled        bool    `yaml:"enabled"`
	LSTMModelPath  string  `yaml:"lstm_model_path"`
	LearningRate   float64 `yaml:"learning_rate"`
	Epochs         int     `yaml:"epochs"`
	BatchSize      int     `yaml:"batch_size"`
	SequenceLength int     `yaml:"sequence_length"`
	NFeatures      int     `yaml:"n_features"`
	HiddenDim      int     `yaml:"hidden_dim"`
	NLayers        int     `yaml:"n_layers"`
	DropoutProb    float64 `yaml:"dropout_prob"`
}

// PriorityQueue represents the priority queue configuration
type PriorityQueue struct {
	HighPriorityWeight   int `yaml:"high_priority_weight"`
	MediumPriorityWeight int `yaml:"medium_priority_weight"`
	LowPriorityWeight    int `yaml:"low_priority_weight"`
	MaxQueueSize         int `yaml:"max_queue_size"`
}

// PerformanceMonitoring represents the performance monitoring configuration
type PerformanceMonitoring struct {
	Enabled         bool `yaml:"enabled"`
	LogInterval     int  `yaml:"log_interval_seconds"`
	DetailedMetrics bool `yaml:"detailed_metrics"`
}

// HardwareAdaptive represents the hardware adaptive configuration
type HardwareAdaptive struct {
	Enabled              bool    `yaml:"enabled"`
	CPUUsageThreshold    float64 `yaml:"cpu_usage_threshold"`
	MemoryUsageThreshold float64 `yaml:"memory_usage_threshold"`
}

// CircuitBreaker represents the circuit breaker configuration
type CircuitBreaker struct {
	Enabled                  bool    `yaml:"enabled"`
	FailureRateThreshold     float64 `yaml:"failure_rate_threshold"`
	RecoveryTime             int     `yaml:"recovery_time_seconds"`
	MinRequestsForEvaluation int     `yaml:"min_requests_for_evaluation"`
}

// TrafficMirroring represents the traffic mirroring configuration
type TrafficMirroring struct {
	Enabled            bool    `yaml:"enabled"`
	MirrorTargetURL    string  `yaml:"mirror_target_url"`
	MirrorSamplingRate float64 `yaml:"mirror_sampling_rate"`
}

// HotReload represents the hot reload configuration
type HotReload struct {
	Enabled             bool `yaml:"enabled"`
	ConfigWatchInterval int  `yaml:"config_watch_interval_seconds"`
}

// GracefulShutdown represents the graceful shutdown configuration
type GracefulShutdown struct {
	Enabled              bool   `yaml:"enabled"`
	ShutdownTimeout      int    `yaml:"shutdown_timeout_seconds"`
	StatePersistencePath string `yaml:"state_persistence_path"`
}

// DebuggingAndVisualization represents the debugging and visualization configuration
type DebuggingAndVisualization struct {
	Enabled                 bool   `yaml:"enabled"`
	PprofPort               int    `yaml:"pprof_port"`
	MetricsExporterType     string `yaml:"metrics_exporter_type"`
	MetricsExporterEndpoint string `yaml:"metrics_exporter_endpoint"`
}

// UserReputationSystem represents the user reputation system configuration
type UserReputationSystem struct {
	Enabled                      bool    `yaml:"enabled"`
	ReputationDecayFactor        float64 `yaml:"reputation_decay_factor"`
	InitialReputation            int     `yaml:"initial_reputation"`
	MinReputationForHighPriority int     `yaml:"min_reputation_for_high_priority"`
	MaxReputationForLowPriority  int     `yaml:"max_reputation_for_low_priority"`
}

// PluginArchitecture represents the plugin architecture configuration
type PluginArchitecture struct {
	Enabled         bool   `yaml:"enabled"`
	PluginDirectory string `yaml:"plugin_directory"`
}

// ABTesting represents the A/B testing configuration
type ABTesting struct {
	Enabled           bool    `yaml:"enabled"`
	ControlGroupRatio float64 `yaml:"control_group_ratio"`
}

// AlertingAndNotification represents the alerting and notification configuration
type AlertingAndNotification struct {
	Enabled              bool                  `yaml:"enabled"`
	AlertManagerURL      string                `yaml:"alert_manager_url"`
	NotificationChannels []NotificationChannel `yaml:"notification_channels"`
}

// NotificationChannel represents a notification channel
type NotificationChannel struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// APIVersioning represents the API versioning configuration
type APIVersioning struct {
	DefaultVersion    string   `yaml:"default_version"`
	SupportedVersions []string `yaml:"supported_versions"`
}

// SecurityAndCompliance represents the security and compliance configuration
type SecurityAndCompliance struct {
	EnableIPWhitelist    bool     `yaml:"enable_ip_whitelist"`
	IPWhitelist          []string `yaml:"ip_whitelist"`
	EnableIPBlacklist    bool     `yaml:"enable_ip_blacklist"`
	IPBlacklist          []string `yaml:"ip_blacklist"`
	MaxRequestBodySizeKB int      `yaml:"max_request_body_size_kb"`
}

// ExperimentalFeatures represents the experimental features configuration
type ExperimentalFeatures struct {
	EnableFeatureX    bool   `yaml:"enable_feature_x"`
	FeatureYParameter string `yaml:"feature_y_parameter"`
}

// Load loads configuration from the specified file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If config file doesn't exist, generate default template
		if os.IsNotExist(err) {
			if err := GenerateDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("failed to generate default config: %w", err)
			}
			// Read the newly generated config
			data, err = os.ReadFile(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read generated config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetDomains extracts unique domains from bot configurations
func (c *Config) GetDomains() []string {
	domainSet := make(map[string]bool)

	for webhookURL := range c.Bots {
		// Parse the webhook URL to extract domain
		if !strings.Contains(webhookURL, "://") {
			// Add https:// prefix if not present
			webhookURL = "https://" + webhookURL
		}

		parsedURL, err := url.Parse(webhookURL)
		if err != nil {
			continue // Skip invalid URLs
		}

		domain := parsedURL.Hostname()
		if domain != "" {
			domainSet[domain] = true
		}
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}

	return domains
}

// Watch watches for configuration file changes and reloads
func Watch(configPath string, reloadFunc func(*Config)) {
	// In a real application, you'd use a library like fsnotify
	// to watch for file changes. For this example, we'll just
	// log that it's not implemented.
	fmt.Printf("File watching is not implemented in this version. Please restart the application to apply configuration changes.")
}

// GetBotConfig returns the bot configuration for a given webhook URL
func (c *Config) GetBotConfig(webhookURL string) (BotConfig, bool) {
	botConfig, exists := c.Bots[webhookURL]
	return botConfig, exists
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Bots) == 0 {
		return fmt.Errorf("no bots configured")
	}

	for webhookURL, botConfig := range c.Bots {
		if botConfig.Secret == "" {
			return fmt.Errorf("bot %s has empty secret", webhookURL)
		}

		if len(botConfig.ForwardTo) == 0 && len(botConfig.RegexRoutes) == 0 {
			return fmt.Errorf("bot %s has no forward_to or regex_routes targets", webhookURL)
		}

		// Validate forward_to URLs
		for _, target := range botConfig.ForwardTo {
			if _, err := url.Parse(target); err != nil {
				return fmt.Errorf("bot %s has invalid forward_to URL %s: %w", webhookURL, target, err)
			}
		}
	}

	return nil
}

// SetDefaults sets default values for missing configuration fields
func (c *Config) SetDefaults() {
	if c.LogLevel == "" {
		c.LogLevel = "development"
	}

	if c.HTTPSPort == "" {
		c.HTTPSPort = "8443"
	}

	if c.HTTPPort == "" {
		c.HTTPPort = "8444"
	}

	if c.IntelligentSchedulingPolicy == nil {
		c.IntelligentSchedulingPolicy = &IntelligentSchedulingPolicy{
			Enabled: true,
		}
	}

	// Set defaults for nested structures
	policy := c.IntelligentSchedulingPolicy
	if policy.DynamicLoadTuning == nil {
		policy.DynamicLoadTuning = &DynamicLoadTuning{
			Enabled:          true,
			LatencyThreshold: 250,
		}
	}

	if policy.AdaptiveThrottling == nil {
		policy.AdaptiveThrottling = &AdaptiveThrottling{
			Enabled:            true,
			MinRequestInterval: 100,
		}
	}

	if policy.CognitiveScheduling == nil {
		policy.CognitiveScheduling = &CognitiveScheduling{
			WorkerPoolSize:       16,
			ModelRetrainInterval: 24,
			FastUserSensitivity:  1.5,
			SpamUserSensitivity:  3.0,
		}
	}

	if policy.DynamicBaselineAnalysis == nil {
		policy.DynamicBaselineAnalysis = &DynamicBaselineAnalysis{
			Enabled:                       true,
			InitialLearningDuration:       60,
			PatternRecognitionSensitivity: 2.0,
			MinDataPointsForBaseline:      100,
		}
	}

	if policy.HotReload == nil {
		policy.HotReload = &HotReload{
			Enabled:             true,
			ConfigWatchInterval: 10,
		}
	}
}

// Global configuration instance
var globalConfig *Config

// SetGlobalConfig sets the global configuration instance
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *Config {
	return globalConfig
}

// GetBotConfigFromRequest returns the bot configuration for a given host and path
func GetBotConfigFromRequest(host, path string) (BotConfig, bool) {
	if globalConfig == nil {
		return BotConfig{}, false
	}

	// Construct the webhook URL from host and path
	webhookURL := host + path

	// Try exact match first
	if botConfig, exists := globalConfig.Bots[webhookURL]; exists {
		return botConfig, true
	}

	// Try with https:// prefix
	httpsURL := "https://" + webhookURL
	if botConfig, exists := globalConfig.Bots[httpsURL]; exists {
		return botConfig, true
	}

	return BotConfig{}, false
}

// GenerateDefaultConfig generates a default configuration template
func GenerateDefaultConfig(configPath string) error {
	defaultConfigTemplate := `# QQ Bot Router Configuration
# 智能QoS路由器配置文件模板
# 请根据实际需求修改以下配置

# 日志级别: production, development
log_level: development

# 服务端口配置
https_port: "8443"
http_port: "8444"

# 机器人配置 - 使用完整 webhook URL 作为键
# 系统会自动从配置中提取域名用于 SSL 证书管理
bots:
  # 示例配置 - 请替换为您的实际配置
  "your-domain.com:8443/webhook":
    secret: "your-bot-secret-here"
    # 默认转发目标
    forward_to:
      - "http://localhost:3000/webhook"
      - "http://localhost:3001/webhook"
    # 正则匹配路由（可选）
    regex_routes:
      "^#help":
        urls:
          - "http://localhost:3002/webhook/help"
      "^#test":
        ishash: true
        endpoints:
          - "http://localhost:3003/webhook/test"
          - "http://localhost:3004/webhook/test"

####################################################
# 智能调度策略 - 全局QoS策略
####################################################
intelligent_scheduling_policy:
  enabled: true

  # 动态负载监控与调节
  dynamic_load_tuning:
    enabled: true
    latency_threshold_ms: 250

  # 自适应入口节流阀
  adaptive_throttling:
    enabled: true
    min_request_interval_ms: 100

  # 自学习优先级调度器
  cognitive_scheduling:
    worker_pool_size: 16
    model_retrain_interval_hours: 24
    fast_user_sensitivity: 1.5
    spam_user_sensitivity: 3.0

  # 动态基线分析
  dynamic_baseline_analysis:
    enabled: true
    initial_learning_duration_minutes: 60
    pattern_recognition_sensitivity: 2.0
    min_data_points_for_baseline: 100

  # 行为模式学习 (LSTM自动编码器)
  behavioral_pattern_learning:
    enabled: false  # 默认关闭，需要模型文件
    lstm_model_path: "./models/lstm_autoencoder.pt"
    learning_rate: 0.001
    epochs: 100
    batch_size: 32
    sequence_length: 60
    n_features: 5
    hidden_dim: 64
    n_layers: 2
    dropout_prob: 0.2

  # 优先级队列配置
  priority_queue:
    high_priority_weight: 10
    medium_priority_weight: 5
    low_priority_weight: 1
    max_queue_size: 10000

  # 性能监控
  performance_monitoring:
    enabled: true
    log_interval_seconds: 60
    detailed_metrics: true

  # 硬件自适应
  hardware_adaptive:
    enabled: true
    cpu_usage_threshold: 0.8
    memory_usage_threshold: 0.85

  # 熔断器
  circuit_breaker:
    enabled: true
    failure_rate_threshold: 0.5
    recovery_time_seconds: 30
    min_requests_for_evaluation: 10

  # 流量镜像（可选）
  traffic_mirroring:
    enabled: false
    mirror_target_url: "http://localhost:9999/mirror"
    mirror_sampling_rate: 0.1

  # 热重载
  hot_reload:
    enabled: true
    config_watch_interval_seconds: 10

  # 优雅关闭
  graceful_shutdown:
    enabled: true
    shutdown_timeout_seconds: 30
    state_persistence_path: "./state/scheduler_state.json"

  # 调试和可视化
  debugging_and_visualization:
    enabled: true
    pprof_port: 6060
    metrics_exporter_type: "prometheus"
    metrics_exporter_endpoint: "http://localhost:9090"

  # 用户声誉系统
  user_reputation_system:
    enabled: true
    reputation_decay_factor: 0.95
    initial_reputation: 100
    min_reputation_for_high_priority: 150
    max_reputation_for_low_priority: 50

  # 插件架构（可选）
  plugin_architecture:
    enabled: false
    plugin_directory: "./plugins"

  # A/B测试（可选）
  ab_testing:
    enabled: false
    control_group_ratio: 0.5

  # 告警和通知（可选）
  alerting_and_notification:
    enabled: false
    alert_manager_url: "http://localhost:9093"
    notification_channels:
      - type: "webhook"
        url: "http://localhost:8080/alerts"

  # API版本控制
  api_versioning:
    default_version: "v1"
    supported_versions: ["v1", "v2"]

  # 安全与合规
  security_and_compliance:
    enable_ip_whitelist: false
    ip_whitelist: []
    enable_ip_blacklist: false
    ip_blacklist: []
    max_request_body_size_kb: 1024

  # 实验性功能
  experimental_features:
    enable_feature_x: false
    feature_y_parameter: "default_value"
`

	return os.WriteFile(configPath, []byte(defaultConfigTemplate), 0644)
}

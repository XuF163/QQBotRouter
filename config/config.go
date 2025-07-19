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
	LogLevel  string               `yaml:"log_level"`
	HTTPSPort string               `yaml:"https_port"`
	HTTPPort  string               `yaml:"http_port"`
	Bots      map[string]BotConfig `yaml:"bots"`

	// QoS Configuration
	QoS QoSConfig `yaml:"qos"`

	// Scheduler Configuration
	Scheduler SchedulerConfig `yaml:"scheduler"`
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

	// Set QoS defaults
	if c.QoS.SystemLimits.MaxLoad == 0 {
		c.QoS = GetDefaultQoSConfig()
	}

	// Set Scheduler defaults
	if c.Scheduler.PrioritySettings.BasePriority == 0 {
		c.Scheduler = GetDefaultSchedulerConfig()
	}
}

// GenerateDefaultConfig generates a default configuration using centralized defaults
func GenerateDefaultConfig(configPath string) error {
	// Create default configuration using centralized default functions
	defaultConfig := Config{
		LogLevel:  "development",
		HTTPSPort: "8443",
		HTTPPort:  "8444",
		QoS:       GetDefaultQoSConfig(),
		Scheduler: GetDefaultSchedulerConfig(),
		Bots: map[string]BotConfig{
			"your-domain.com/webhook": {
				Secret: "your-bot-secret-here",
				ForwardTo: []string{
					"http://localhost:3000/webhook",
					"http://localhost:3001/webhook",
				},
				RegexRoutes: map[string]RegexRouteConfig{
					"^#help": {
						URLs: []string{"http://localhost:3002/webhook/help"},
					},
					"^#test": {
						Endpoints: []string{
							"http://localhost:3003/webhook/test",
							"http://localhost:3004/webhook/test",
						},
					},
				},
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Add header comment
	header := `# QQ Bot Router Configuration
# 智能QoS路由器配置文件
# 此配置使用集中化的默认值管理
# 请根据实际需求修改以下配置

`
	finalContent := header + string(data)

	return os.WriteFile(configPath, []byte(finalContent), 0644)
}

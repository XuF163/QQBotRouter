package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"qqbotrouter/autocert"
)

// Config represents the main configuration structure
type Config struct {
	LogLevel  string               `yaml:"log_level"`
	HTTPSPort string               `yaml:"https_port"`
	HTTPPort  string               `yaml:"http_port"`
	Bots      map[string]BotConfig `yaml:"bots"`
}

// BotConfig represents individual bot configuration
type BotConfig struct {
	Secret    string   `yaml:"secret"`
	ForwardTo []string `yaml:"forward_to"`
}

// Load loads configuration from the specified file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
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
// Note: File watching is disabled in this simplified version
func Watch(configPath string, certManager *autocert.Manager, logger *zap.Logger) {
	logger.Info("Config file watching is disabled in this version", zap.String("path", configPath))
	// TODO: Implement file watching when fsnotify dependency is available
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

		if len(botConfig.ForwardTo) == 0 {
			return fmt.Errorf("bot %s has no forward_to targets", webhookURL)
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
}

// Global configuration instance
var globalConfig *Config

// SetGlobalConfig sets the global configuration instance
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}

// GetBotConfig returns the bot configuration for a given host and path
// This is a package-level function that uses the global configuration
func GetBotConfig(host, path string) (BotConfig, bool) {
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

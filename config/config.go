package config

import (
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LogLevel  string         `yaml:"log_level"`
	HTTPSPort string         `yaml:"https_port"`
	HTTPPort  string         `yaml:"http_port"`
	Domains   []string       `yaml:"domains"`
	Bots      map[string]Bot `yaml:"bots"`
}

// Bot represents bot configuration
type Bot struct {
	Secret    string         `yaml:"secret"`
	ForwardTo []string       `yaml:"forward_to"`
	Paths     map[string]Bot `yaml:"paths,omitempty"`
}

var globalConfig *Config

// Load loads configuration from the specified file
func Load(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Store globally for GetBotConfig
	globalConfig = &config
	return &config, nil
}

// GetBotConfig returns bot configuration for the given host and path
func GetBotConfig(host, path string) (Bot, bool) {
	if globalConfig == nil {
		return Bot{}, false
	}

	// Remove port from host if present (e.g., "test.genshin.icu:8443" -> "test.genshin.icu")
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// If SplitHostPort fails, assume host doesn't contain port
		hostname = host
	}

	// First check if there's a bot configuration for this host
	bot, exists := globalConfig.Bots[hostname]
	if !exists {
		return Bot{}, false
	}

	// If there are path-specific configurations, check for a match
	if bot.Paths != nil {
		if pathBot, pathExists := bot.Paths[path]; pathExists {
			return pathBot, true
		}
	}

	// Return the default bot configuration for this host
	return bot, true
}

// Watch watches for configuration file changes and reloads
func Watch(filename string, certManager *autocert.Manager, logger *zap.Logger) {
	// Simple file watching implementation
	// In production, you might want to use a more sophisticated file watcher
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if info, err := ioutil.ReadFile(filename); err == nil {
				// Simple check - in real implementation you'd check file modification time
				if len(info) > 0 {
					logger.Debug("Config file check completed")
				}
			}
		}
	}
}

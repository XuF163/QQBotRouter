package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v3"
)

// --- Structs for Configuration ---
type Config struct {
	LogLevel  string                  `yaml:"log_level"`
	HTTPSPort string                  `yaml:"https_port"`
	HTTPPort  string                  `yaml:"http_port"`
	Domains   []string                `yaml:"domains"`
	Bots      map[string]DomainConfig `yaml:"bots"`
}

type DomainConfig struct {
	Bot   `yaml:",inline"`
	Paths map[string]Bot `yaml:"paths"`
}

type Bot struct {
	Secret    string   `yaml:"secret"`
	ForwardTo []string `yaml:"forward_to"`
}

// Global config variables
var (
	currentConfig *Config
	configLock    = new(sync.RWMutex)
)

// Load initial configuration
func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(file, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}
	currentConfig = &c
	return currentConfig, nil
}

// Watch for changes and reload
func Watch(path string, m *autocert.Manager, logger *zap.Logger) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal("Failed to create config watcher", zap.Error(err))
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		logger.Fatal("Failed to watch config file", zap.Error(err))
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				logger.Info("Config file changed, reloading...")
				newConfig, err := Load(path)
				if err != nil {
					logger.Error("Error reloading config", zap.Error(err))
				} else {
					configLock.Lock()
					currentConfig = newConfig
					m.HostPolicy = autocert.HostWhitelist(currentConfig.Domains...)
					configLock.Unlock()
					logger.Info("Config reloaded successfully.")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("Watcher error", zap.Error(err))
		}
	}
}

// GetBotConfig finds the correct bot configuration based on host and path.
func GetBotConfig(host, path string) (Bot, bool) {
	configLock.RLock()
	defer configLock.RUnlock()

	if currentConfig == nil {
		return Bot{}, false
	}

	// 去除端口号，只保留域名部分
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	domainConfig, ok := currentConfig.Bots[host]
	if !ok {
		return Bot{}, false
	}

	if bot, ok := domainConfig.Paths[path]; ok {
		return bot, true
	}

	if domainConfig.Secret != "" {
		return domainConfig.Bot, true
	}

	return Bot{}, false
}

func GetDomains() []string {
	configLock.RLock()
	defer configLock.RUnlock()
	if currentConfig == nil {
		return nil
	}
	return currentConfig.Domains
}

package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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

// ConfigWatcher manages configuration file watching and hot reloading
type ConfigWatcher struct {
	mu            sync.RWMutex
	configPath    string
	currentConfig *Config
	watcher       *fsnotify.Watcher
	reloadFunc    func(*Config)
	errorHandler  func(error)
	done          chan struct{}
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(configPath string, reloadFunc func(*Config), errorHandler func(error)) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Load initial configuration
	initialConfig, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	cw := &ConfigWatcher{
		configPath:    configPath,
		currentConfig: initialConfig,
		watcher:       watcher,
		reloadFunc:    reloadFunc,
		errorHandler:  errorHandler,
		done:          make(chan struct{}),
	}

	return cw, nil
}

// Start begins watching the configuration file for changes
func (cw *ConfigWatcher) Start() error {
	// Add the config file to the watcher
	if err := cw.watcher.Add(cw.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	go cw.watchLoop()
	return nil
}

// Stop stops the configuration watcher
func (cw *ConfigWatcher) Stop() {
	close(cw.done)
	cw.watcher.Close()
}

// GetCurrentConfig returns the current configuration (thread-safe)
func (cw *ConfigWatcher) GetCurrentConfig() *Config {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.currentConfig
}

// watchLoop is the main event loop for file watching
func (cw *ConfigWatcher) watchLoop() {
	// Debounce timer to avoid multiple rapid reloads
	var debounceTimer *time.Timer
	const debounceDelay = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Only react to write events (file modifications)
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDelay, func() {
					cw.reloadConfig()
				})
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			if cw.errorHandler != nil {
				cw.errorHandler(fmt.Errorf("file watcher error: %w", err))
			}

		case <-cw.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

// reloadConfig attempts to reload the configuration file
func (cw *ConfigWatcher) reloadConfig() {
	// Load new configuration
	newConfig, err := Load(cw.configPath)
	if err != nil {
		if cw.errorHandler != nil {
			cw.errorHandler(fmt.Errorf("failed to reload config: %w", err))
		}
		return
	}

	// Validate new configuration
	if err := newConfig.Validate(); err != nil {
		if cw.errorHandler != nil {
			cw.errorHandler(fmt.Errorf("invalid config during reload: %w", err))
		}
		return
	}

	// Set defaults for new configuration
	newConfig.SetDefaults()

	// Atomically update current configuration
	cw.mu.Lock()
	oldConfig := cw.currentConfig
	cw.currentConfig = newConfig
	cw.mu.Unlock()

	// Call reload function if provided
	if cw.reloadFunc != nil {
		cw.reloadFunc(newConfig)
	}

	fmt.Printf("Configuration successfully reloaded from %s\n", cw.configPath)
	_ = oldConfig // Prevent unused variable warning
}

// Watch watches for configuration file changes and reloads (legacy function for backward compatibility)
func Watch(configPath string, reloadFunc func(*Config)) {
	errorHandler := func(err error) {
		fmt.Printf("Config watcher error: %v\n", err)
	}

	watcher, err := NewConfigWatcher(configPath, reloadFunc, errorHandler)
	if err != nil {
		fmt.Printf("Failed to create config watcher: %v\n", err)
		return
	}

	if err := watcher.Start(); err != nil {
		fmt.Printf("Failed to start config watcher: %v\n", err)
		return
	}

	// Keep the watcher running (this is a blocking call in the legacy interface)
	select {}
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
	defaults := GetDefaultValues()

	if c.LogLevel == "" {
		c.LogLevel = defaults.Server.LogLevel
	}

	if c.HTTPSPort == "" {
		c.HTTPSPort = defaults.Server.HTTPSPort
	}

	if c.HTTPPort == "" {
		c.HTTPPort = defaults.Server.HTTPPort
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
	// Get centralized default values
	defaults := GetDefaultValues()

	// Create default configuration using centralized default values
	defaultConfig := Config{
		LogLevel:  defaults.Server.LogLevel,
		HTTPSPort: defaults.Server.HTTPSPort,
		HTTPPort:  defaults.Server.HTTPPort,
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

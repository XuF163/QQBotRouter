package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/autocert"
	"qqbotrouter/config"
	"qqbotrouter/handler"
	"qqbotrouter/initialize"
	"qqbotrouter/load"
	"qqbotrouter/ml_trainer"
	"qqbotrouter/observer"
	"qqbotrouter/qos"
	"qqbotrouter/scheduler"
	"qqbotrouter/services"
	"qqbotrouter/stats"
)

var (
	logger        *zap.Logger
	configWatcher *config.ConfigWatcher
	currentConfig *config.Config
	configMutex   sync.RWMutex
)

// handleConfigReload handles configuration reload and updates relevant components
func handleConfigReload(newConfig *config.Config, qosManager *qos.QoSManager, mainScheduler *scheduler.Scheduler) {
	logger.Info("Processing configuration reload...")

	// Update global config atomically
	configMutex.Lock()
	oldConfig := currentConfig
	currentConfig = newConfig
	configMutex.Unlock()

	// Check if logger level changed
	if oldConfig.LogLevel != newConfig.LogLevel {
		logger.Info("Log level changed, reinitializing logger",
			zap.String("old_level", oldConfig.LogLevel),
			zap.String("new_level", newConfig.LogLevel))
		initLogger(newConfig.LogLevel)
	}

	// Update QoS configuration
	if qosManager != nil {
		qosManager.UpdateConfig(&newConfig.QoS)
		logger.Info("QoS configuration updated")
	}

	// Update Scheduler configuration
	if mainScheduler != nil {
		mainScheduler.UpdateConfig(&newConfig.Scheduler)
		logger.Info("Scheduler configuration updated")
	}

	// Log configuration changes summary
	logConfigChanges(oldConfig, newConfig)

	logger.Info("Configuration reload completed successfully")
}

// logConfigChanges logs a summary of configuration changes
func logConfigChanges(oldConfig, newConfig *config.Config) {
	changes := []string{}

	if oldConfig.LogLevel != newConfig.LogLevel {
		changes = append(changes, fmt.Sprintf("log_level: %s -> %s", oldConfig.LogLevel, newConfig.LogLevel))
	}
	if oldConfig.HTTPSPort != newConfig.HTTPSPort {
		changes = append(changes, fmt.Sprintf("https_port: %s -> %s", oldConfig.HTTPSPort, newConfig.HTTPSPort))
	}
	if oldConfig.HTTPPort != newConfig.HTTPPort {
		changes = append(changes, fmt.Sprintf("http_port: %s -> %s", oldConfig.HTTPPort, newConfig.HTTPPort))
	}
	if len(oldConfig.Bots) != len(newConfig.Bots) {
		changes = append(changes, fmt.Sprintf("bots_count: %d -> %d", len(oldConfig.Bots), len(newConfig.Bots)))
	}
	if oldConfig.QoS.HotReload.Enabled != newConfig.QoS.HotReload.Enabled {
		changes = append(changes, fmt.Sprintf("hot_reload: %t -> %t", oldConfig.QoS.HotReload.Enabled, newConfig.QoS.HotReload.Enabled))
	}

	if len(changes) > 0 {
		logger.Info("Configuration changes detected", zap.Strings("changes", changes))
	} else {
		logger.Info("No significant configuration changes detected")
	}
}

func initLogger(logLevel string) {
	var err error

	switch logLevel {
	case "production":
		logger, err = zap.NewProduction()
	case "development":
		logger, err = zap.NewDevelopment()
	default:
		logger, err = zap.NewDevelopment()
		fmt.Fprintf(os.Stderr, "Unknown log_level '%s' in config. Defaulting to development.\n", logLevel)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	// Ensure logger is synced on exit
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()
}

func main() {
	// Check and create config files if they don't exist
	if err := initialize.CheckConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check/create config files: %v\n", err)
		os.Exit(1)
	}

	// 1. Load initial configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load initial config: %v\n", err)
		os.Exit(1)
	}
	cfg.SetDefaults()

	// Store initial config globally
	configMutex.Lock()
	currentConfig = cfg
	configMutex.Unlock()

	// Initialize logger based on config
	initLogger(cfg.LogLevel)

	// 2. Initialize all QoS services
	loadCounter := load.NewCounter()
	statsAnalyzer := stats.NewStatsAnalyzer(cfg.Scheduler.UserBehaviorAnalysis.MinDataPointsForBaseline)
	qosObserver := observer.NewObserver(time.Duration(cfg.QoS.DynamicLoadBalancing.LoadThreshold)*time.Millisecond, 100)
	mlTrainer := ml_trainer.NewMLTrainer(statsAnalyzer)
	qosManager := qos.NewQoSManager(&cfg.QoS, loadCounter, statsAnalyzer, qosObserver, logger)
	mainScheduler := scheduler.NewScheduler(statsAnalyzer, &cfg.Scheduler, &cfg.QoS, loadCounter)

	// 3. Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start all background services using service manager
	serviceManager := services.NewServiceManager(logger)
	serviceManager.AddService(statsAnalyzer)
	serviceManager.AddService(qosObserver)
	serviceManager.AddService(mlTrainer)
	serviceManager.AddService(qosManager)
	serviceManager.AddService(mainScheduler)

	serviceManager.StartAll(ctx)
	logger.Info("All QoS services have been initialized and started.")

	// 4. Configure autocert manager
	domains := cfg.GetDomains()
	logger.Info("Extracted domains for SSL certificates", zap.Strings("domains", domains))
	certManager := autocert.NewManager(domains, "secret-dir")

	// 5. Start config hot-reloader (if enabled)
	if cfg.QoS.HotReload.Enabled {
		// Create config watcher with proper error handling
		errorHandler := func(err error) {
			logger.Error("Configuration watcher error", zap.Error(err))
		}

		reloadHandler := func(newConfig *config.Config) {
			handleConfigReload(newConfig, qosManager, mainScheduler)
		}

		configWatcher, err = config.NewConfigWatcher("config.yaml", reloadHandler, errorHandler)
		if err != nil {
			logger.Error("Failed to create config watcher", zap.Error(err))
		} else {
			if err := configWatcher.Start(); err != nil {
				logger.Error("Failed to start config watcher", zap.Error(err))
			} else {
				logger.Info("Configuration hot-reload enabled",
					zap.String("check_interval", cfg.QoS.HotReload.CheckInterval))
			}
		}
	}

	// 6. Set up HTTP/S servers
	mux := http.NewServeMux()
	mux.Handle("/", handler.NewWebhookHandler(cfg, logger, mainScheduler, qosManager))

	server := &http.Server{
		Addr:      ":" + cfg.HTTPSPort,
		Handler:   mux,
		TLSConfig: certManager.TLSConfig(),
	}

	// 7. Start HTTP server for ACME challenges
	go func() {
		logger.Info("Starting HTTP server for ACME challenges...", zap.String("port", cfg.HTTPPort))
		if err := http.ListenAndServe(":"+cfg.HTTPPort, certManager.HTTPHandler(nil)); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 8. Start HTTPS server in a goroutine
	go func() {
		logger.Info("Starting HTTPS server...", zap.String("port", cfg.HTTPSPort))
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTPS server failed", zap.Error(err))
		}
	}()

	// 9. Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received, starting graceful shutdown...")

	// Cancel context to stop all background services
	cancel()

	// Wait for all background services to complete
	logger.Info("Waiting for background services to complete...")
	serviceManager.WaitForAll()
	logger.Info("All background services stopped")

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	} else {
		logger.Info("Server shutdown completed successfully")
	}

	// Stop config watcher if running
	if configWatcher != nil {
		configWatcher.Stop()
		logger.Info("Configuration watcher stopped")
	}

	logger.Info("Application shutdown completed")
}

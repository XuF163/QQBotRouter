package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/autocert"
	"qqbotrouter/config"
	"qqbotrouter/handler"
	"qqbotrouter/initialize"
	"qqbotrouter/load"
	"qqbotrouter/ml_trainer"
	"qqbotrouter/observer"
	"qqbotrouter/scheduler"
	"qqbotrouter/stats"
)

var logger *zap.Logger

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
	config.SetGlobalConfig(cfg)
	cfg.SetDefaults()

	// Initialize logger based on config
	initLogger(cfg.LogLevel)

	// 2. Initialize all QoS services
	loadCounter := load.NewCounter()
	statsAnalyzer := stats.NewStatsAnalyzer(cfg.IntelligentSchedulingPolicy.DynamicBaselineAnalysis.MinDataPointsForBaseline)
	qosObserver := observer.NewObserver(time.Duration(cfg.IntelligentSchedulingPolicy.DynamicLoadTuning.LatencyThreshold)*time.Millisecond, 100)
	mlTrainer := ml_trainer.NewMLTrainer(statsAnalyzer)
	mainScheduler := scheduler.NewScheduler(statsAnalyzer, cfg.IntelligentSchedulingPolicy.CognitiveScheduling, loadCounter)

	// 3. Start all background services
	go statsAnalyzer.Run(time.NewTicker(1 * time.Minute))
	go qosObserver.Run(time.NewTicker(1 * time.Minute))
	go mlTrainer.Run(time.NewTicker(5 * time.Minute))
	go mainScheduler.Run()

	logger.Info("All QoS services have been initialized and started.")

	// 4. Configure autocert manager
	domains := cfg.GetDomains()
	logger.Info("Extracted domains for SSL certificates", zap.Strings("domains", domains))
	certManager := autocert.NewManager(domains, "secret-dir")

	// 5. Start config hot-reloader (if enabled)
	if cfg.IntelligentSchedulingPolicy.HotReload.Enabled {
		// The actual hot-reloading logic is not implemented in this version.
		// The function call is left here as a placeholder.
		go config.Watch("config.yaml", func(c *config.Config) {
			logger.Info("Configuration reload triggered (not implemented).")
		})
	}

	// 6. Set up HTTP/S servers
	mux := http.NewServeMux()
	mux.Handle("/", handler.NewWebhookHandler(logger, mainScheduler))

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

	// 8. Start HTTPS server
	logger.Info("Starting HTTPS server...", zap.String("port", cfg.HTTPSPort))
	if err := server.ListenAndServeTLS("", ""); err != nil {
		logger.Fatal("HTTPS server failed", zap.Error(err))
	}
}

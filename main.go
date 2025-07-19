package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	cfg.SetDefaults()

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

	// Start all background services with context
	go statsAnalyzer.RunWithContext(ctx, time.NewTicker(30*time.Second))
	go qosObserver.RunWithContext(ctx, time.NewTicker(10*time.Second))
	go mlTrainer.RunWithContext(ctx, time.NewTicker(5*time.Minute))
	go qosManager.Start(ctx)
	go mainScheduler.RunWithContext(ctx)

	logger.Info("All QoS services have been initialized and started.")

	// 4. Configure autocert manager
	domains := cfg.GetDomains()
	logger.Info("Extracted domains for SSL certificates", zap.Strings("domains", domains))
	certManager := autocert.NewManager(domains, "secret-dir")

	// 5. Start config hot-reloader (if enabled)
	if cfg.QoS.HotReload.Enabled {
		// The actual hot-reloading logic is not implemented in this version.
		// The function call is left here as a placeholder.
		go config.Watch("config.yaml", func(c *config.Config) {
			logger.Info("Configuration reload triggered (not implemented).")
		})
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

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	} else {
		logger.Info("Server shutdown completed successfully")
	}

	logger.Info("Application shutdown completed")
}

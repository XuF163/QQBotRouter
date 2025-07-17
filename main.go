package main

import (
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"qqbotrouter/autocert"
	"qqbotrouter/config"
	"qqbotrouter/handler"
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
		// Default to development if not specified or unknown
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
	// 1. Load initial configuration to get log level
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		// Cannot use logger yet, fall back to standard log or fmt
		fmt.Fprintf(os.Stderr, "Failed to load initial config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger based on config
	initLogger(cfg.LogLevel)

	// 2. Configure autocert manager
	certManager := autocert.NewManager(cfg.Domains, "secret-dir")

	// 3. Start config hot-reloader
	go config.Watch("config/config.yaml", certManager, logger)

	// 4. Set up HTTP/S servers
	mux := http.NewServeMux()
	mux.Handle("/", handler.NewWebhookHandler(logger))

	server := &http.Server{
		Addr:      ":443",
		Handler:   mux,
		TLSConfig: certManager.TLSConfig(),
	}

	// 5. Start HTTP server for ACME challenges
	go func() {
		logger.Info("Starting HTTP server on :80 for ACME challenges...")
		if err := http.ListenAndServe(":80", certManager.HTTPHandler(nil)); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 6. Start HTTPS server
	logger.Info("Starting HTTPS server on :443...")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		logger.Fatal("HTTPS server failed", zap.Error(err))
	}
}

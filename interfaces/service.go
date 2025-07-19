package interfaces

import "context"

// BackgroundService defines a unified interface for all background services
type BackgroundService interface {
	// Run starts the background service with the given context
	// The service should stop gracefully when the context is cancelled
	Run(ctx context.Context) error
}

// TickerService defines an interface for services that need periodic execution
type TickerService interface {
	BackgroundService
	// GetTickerInterval returns the interval for periodic execution
	GetTickerInterval() string
}

package services

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"qqbotrouter/interfaces"
)

// ServiceManager manages multiple background services
type ServiceManager struct {
	services []interfaces.BackgroundService
	logger   *zap.Logger
	wg       sync.WaitGroup
}

// NewServiceManager creates a new service manager
func NewServiceManager(logger *zap.Logger) *ServiceManager {
	return &ServiceManager{
		services: make([]interfaces.BackgroundService, 0),
		logger:   logger,
	}
}

// AddService adds a service to be managed
func (sm *ServiceManager) AddService(service interfaces.BackgroundService) {
	sm.services = append(sm.services, service)
}

// StartAll starts all registered services
func (sm *ServiceManager) StartAll(ctx context.Context) {
	for i, service := range sm.services {
		sm.wg.Add(1)
		go func(idx int, svc interfaces.BackgroundService) {
			defer sm.wg.Done()
			if err := svc.Run(ctx); err != nil && err != context.Canceled {
				sm.logger.Error("Service stopped with error",
					zap.Int("service_index", idx),
					zap.Error(err))
			} else {
				sm.logger.Info("Service stopped gracefully",
					zap.Int("service_index", idx))
			}
		}(i, service)
	}
	sm.logger.Info("All services started", zap.Int("service_count", len(sm.services)))
}

// WaitForAll waits for all services to stop
func (sm *ServiceManager) WaitForAll() {
	sm.wg.Wait()
	sm.logger.Info("All services stopped")
}

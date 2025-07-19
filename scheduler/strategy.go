package scheduler

import (
	"qqbotrouter/config"
	"qqbotrouter/interfaces"
)

// PriorityStrategy defines the interface for priority calculation strategies
type PriorityStrategy interface {
	CalculatePriority(userID string, contentType string, statsProvider interfaces.StatProvider, config *config.SchedulerConfig) int
}

// LoadBasedStrategy calculates priority based on load metrics
type LoadBasedStrategy struct{}

func (s *LoadBasedStrategy) CalculatePriority(userID string, contentType string, statsProvider interfaces.StatProvider, config *config.SchedulerConfig) int {
	priority := config.PrioritySettings.BasePriority

	// Adjust priority based on system load
	systemLoad := statsProvider.GetSystemLoad()
	if systemLoad > 0.8 {
		priority += config.PrioritySettings.HighLoadAdjustment
	} else if systemLoad < 0.3 {
		priority += config.PrioritySettings.LowLoadAdjustment
	}

	return priority
}

// ContentBasedStrategy calculates priority based on content type
type ContentBasedStrategy struct{}

func (s *ContentBasedStrategy) CalculatePriority(userID string, contentType string, statsProvider interfaces.StatProvider, config *config.SchedulerConfig) int {
	priority := config.PrioritySettings.BasePriority

	// Adjust priority based on content type
	switch contentType {
	case "text":
		priority += 10
	case "image":
		priority += 5
	case "video":
		priority -= 5
	case "file":
		priority -= 10
	}

	return priority
}

// HybridStrategy combines load-based and content-based strategies
type HybridStrategy struct {
	loadStrategy    *LoadBasedStrategy
	contentStrategy *ContentBasedStrategy
	loadWeight      float64
	contentWeight   float64
}

func NewHybridStrategy(loadWeight, contentWeight float64) *HybridStrategy {
	return &HybridStrategy{
		loadStrategy:    &LoadBasedStrategy{},
		contentStrategy: &ContentBasedStrategy{},
		loadWeight:      loadWeight,
		contentWeight:   contentWeight,
	}
}

func (s *HybridStrategy) CalculatePriority(userID string, contentType string, statsProvider interfaces.StatProvider, config *config.SchedulerConfig) int {
	loadPriority := s.loadStrategy.CalculatePriority(userID, contentType, statsProvider, config)
	contentPriority := s.contentStrategy.CalculatePriority(userID, contentType, statsProvider, config)

	// Weighted combination
	finalPriority := int(float64(loadPriority)*s.loadWeight + float64(contentPriority)*s.contentWeight)
	return finalPriority
}

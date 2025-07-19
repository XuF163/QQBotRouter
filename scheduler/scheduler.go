package scheduler

import (
	"container/heap"
	"context"
	"net/http"
	"regexp"
	"sync"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/forwarder"
	"qqbotrouter/interfaces"
	"qqbotrouter/utils"
)

// Request represents a request to be processed.
type Request struct {
	Context   context.Context
	Body      []byte
	Header    http.Header
	BotConfig config.BotConfig
	Logger    *zap.Logger
	priority  int
	index     int
	userID    string
	message   string
	timestamp time.Time
}

// PriorityQueue implements heap.Interface and holds Requests.
type PriorityQueue []*Request

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	request := x.(*Request)
	request.index = n
	*pq = append(*pq, request)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	request := old[n-1]
	old[n-1] = nil
	request.index = -1
	*pq = old[0 : n-1]
	return request
}

// Scheduler handles asynchronous request processing and priority scheduling.
type Scheduler struct {
	pq               PriorityQueue
	statsProvider    interfaces.StatProvider
	schedulerConfig  *config.SchedulerConfig
	qosConfig        *config.QoSConfig
	loadProvider     interfaces.LoadProvider
	workerPool       chan *Request
	userLastRequest  map[string]time.Time // Track last request time per user
	mu               sync.RWMutex         // Protect userLastRequest map
	priorityStrategy PriorityStrategy     // Strategy for priority calculation
}

// NewScheduler creates a new Scheduler.
func NewScheduler(statsProvider interfaces.StatProvider, schedulerConfig *config.SchedulerConfig, qosConfig *config.QoSConfig, loadProvider interfaces.LoadProvider) *Scheduler {
	s := &Scheduler{
		pq:               make(PriorityQueue, 0),
		statsProvider:    statsProvider,
		schedulerConfig:  schedulerConfig,
		qosConfig:        qosConfig,
		loadProvider:     loadProvider,
		workerPool:       make(chan *Request, schedulerConfig.WorkerPoolSize),
		userLastRequest:  make(map[string]time.Time),
		priorityStrategy: NewHybridStrategy(0.6, 0.4), // Default to hybrid strategy
	}
	heap.Init(&s.pq)
	return s
}

// Submit submits a new request to the scheduler and returns success status.
func (s *Scheduler) Submit(ctx context.Context, body []byte, header http.Header, botConfig config.BotConfig, logger *zap.Logger) bool {
	// Parse message content to extract user info
	userID, message := s.parseMessage(body)

	// Calculate dynamic priority based on stats and config
	priority := s.calculatePriority(userID, message)

	request := &Request{
		Context:   ctx,
		Body:      body,
		Header:    header,
		BotConfig: botConfig,
		Logger:    logger,
		priority:  priority,
		userID:    userID,
		message:   message,
		timestamp: time.Now(),
	}
	heap.Push(&s.pq, request)
	return true // Successfully queued
}

// Run starts the scheduler with context support
func (s *Scheduler) Run(ctx context.Context) error {
	// Start worker goroutines
	for i := 0; i < s.schedulerConfig.WorkerPoolSize; i++ {
		go s.workerWithContext(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			close(s.workerPool)
			return ctx.Err()
		default:
			if len(s.pq) > 0 {
				request := heap.Pop(&s.pq).(*Request)
				select {
				case s.workerPool <- request:
				case <-ctx.Done():
					close(s.workerPool)
					return ctx.Err()
				}
			} else {
				// Prevent busy-waiting when the queue is empty
				idleInterval := s.qosConfig.ParseDuration(s.qosConfig.RequestTimeouts.IdleCheckInterval)
				select {
				case <-time.After(idleInterval):
				case <-ctx.Done():
					close(s.workerPool)
					return ctx.Err()
				}
			}
		}
	}
}

// parseMessage extracts user ID and message content from request body
func (s *Scheduler) parseMessage(body []byte) (userID, message string) {
	return utils.ParseMessage(body)
}

// calculatePriority calculates request priority using the configured strategy
func (s *Scheduler) calculatePriority(userID, message string) int {
	// Determine content type from message
	contentType := s.determineContentType(message)

	// Use strategy pattern for priority calculation
	priority := s.priorityStrategy.CalculatePriority(userID, contentType, s.statsProvider, s.schedulerConfig)

	// Apply anti-spam detection using P50 baseline
	now := time.Now()
	s.mu.Lock()
	lastRequestTime, exists := s.userLastRequest[userID]
	s.userLastRequest[userID] = now
	s.mu.Unlock()

	if exists {
		requestInterval := now.Sub(lastRequestTime)
		p50Baseline := s.statsProvider.P50()

		// If user's request interval is much shorter than P50 baseline, consider it spam
		if p50Baseline > 0 && requestInterval < p50Baseline/3 {
			priority -= s.schedulerConfig.PrioritySettings.HighLoadAdjustment * 2 // Significant penalty for potential spam
		} else if p50Baseline > 0 && requestInterval < p50Baseline/2 {
			priority -= s.schedulerConfig.PrioritySettings.HighLoadAdjustment // Moderate penalty for fast requests
		}
	}

	// Ensure priority is within valid range
	if priority < s.schedulerConfig.PrioritySettings.MinPriority {
		priority = s.schedulerConfig.PrioritySettings.MinPriority
	} else if priority > s.schedulerConfig.PrioritySettings.MaxPriority {
		priority = s.schedulerConfig.PrioritySettings.MaxPriority
	}

	return priority
}

// determineContentType analyzes message content to determine its type
func (s *Scheduler) determineContentType(message string) string {
	// Simple content type detection based on message patterns
	if len(message) == 0 {
		return "empty"
	}

	// Check for common patterns
	if len(message) > 1000 {
		return "long_text"
	}

	// Check for URLs or file references
	if utils.ContainsURL(message) {
		return "url"
	}

	// Default to text
	return "text"
}

// SetPriorityStrategy allows changing the priority calculation strategy
func (s *Scheduler) SetPriorityStrategy(strategy PriorityStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.priorityStrategy = strategy
}

// worker processes requests from the worker pool.
func (s *Scheduler) worker() {
	for request := range s.workerPool {
		// Implement intelligent routing logic
		destinations := s.selectDestinations(request)

		// Forward request and get results
		processingTimeout := s.qosConfig.ParseDuration(s.qosConfig.RequestTimeouts.ProcessingTimeout)
		forwardTimeout := s.qosConfig.ParseDuration(s.qosConfig.RequestTimeouts.ForwardTimeout)
		results := forwarder.ForwardToMultipleDestinations(
			request.Context,
			request.Logger,
			destinations,
			request.Body,
			request.Header,
			processingTimeout,
			s.loadProvider,
			forwardTimeout,
		)

		// Check if any destination succeeded
		success := false
		for _, result := range results {
			if result.Success {
				success = true
				break
			}
		}

		// Log processing result
		if success {
			request.Logger.Debug("Request processed successfully",
				zap.String("user_id", request.userID),
				zap.Int("priority", request.priority),
				zap.Int("successful_destinations", len(results)))
		} else {
			request.Logger.Warn("Request processing failed",
				zap.String("user_id", request.userID),
				zap.Int("priority", request.priority),
				zap.Int("failed_destinations", len(results)))
		}
	}
}

// workerWithContext processes requests from the worker pool with context support for graceful shutdown.
func (s *Scheduler) workerWithContext(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request, ok := <-s.workerPool:
			if !ok {
				return // Channel closed
			}
			// Implement intelligent routing logic
			destinations := s.selectDestinations(request)

			// Forward request and get results
			processingTimeout := s.qosConfig.ParseDuration(s.qosConfig.RequestTimeouts.ProcessingTimeout)
			forwardTimeout := s.qosConfig.ParseDuration(s.qosConfig.RequestTimeouts.ForwardTimeout)
			results := forwarder.ForwardToMultipleDestinations(
				request.Context,
				request.Logger,
				destinations,
				request.Body,
				request.Header,
				processingTimeout,
				s.loadProvider,
				forwardTimeout,
			)

			// Check if any destination succeeded
			success := false
			for _, result := range results {
				if result.Success {
					success = true
					break
				}
			}

			// Log processing result
			if success {
				request.Logger.Debug("Request processed successfully",
					zap.String("user_id", request.userID),
					zap.Int("priority", request.priority),
					zap.Int("successful_destinations", len(results)))
			} else {
				request.Logger.Warn("Request processing failed",
					zap.String("user_id", request.userID),
					zap.Int("priority", request.priority),
					zap.Int("failed_destinations", len(results)))
			}
		}
	}
}

// selectDestinations implements intelligent routing based on message content and config
func (s *Scheduler) selectDestinations(request *Request) []string {
	// First, check regex routes
	if destinations := s.checkRegexRoutes(request); len(destinations) > 0 {
		return destinations
	}

	// Fallback to default forward_to
	return request.BotConfig.ForwardTo
}

// checkRegexRoutes checks if message matches any regex routing rules
func (s *Scheduler) checkRegexRoutes(request *Request) []string {
	for pattern, routeConfig := range request.BotConfig.RegexRoutes {
		matched, err := regexp.MatchString(pattern, request.message)
		if err != nil {
			request.Logger.Warn("Invalid regex pattern", zap.String("pattern", pattern), zap.Error(err))
			continue
		}

		if matched {
			// Return URLs or Endpoints based on configuration
			if len(routeConfig.URLs) > 0 {
				return routeConfig.URLs
			}
			if len(routeConfig.Endpoints) > 0 {
				return routeConfig.Endpoints
			}
		}
	}

	return nil
}

// UpdateConfig updates the scheduler configuration during hot reload
func (s *Scheduler) UpdateConfig(newSchedulerConfig *config.SchedulerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldConfig := s.schedulerConfig
	s.schedulerConfig = newSchedulerConfig

	// Check if worker pool size changed
	if oldConfig.WorkerPoolSize != newSchedulerConfig.WorkerPoolSize {
		// Note: In a production system, you might want to gracefully resize the worker pool
		// For now, we'll just log the change and note that it requires a restart
		zap.L().Warn("Worker pool size changed - restart required for full effect",
			zap.Int("old_size", oldConfig.WorkerPoolSize),
			zap.Int("new_size", newSchedulerConfig.WorkerPoolSize))
	}

	// Clear user request history if user behavior analysis settings changed significantly
	if oldConfig.UserBehaviorAnalysis.Enabled != newSchedulerConfig.UserBehaviorAnalysis.Enabled ||
		oldConfig.UserBehaviorAnalysis.MinDataPointsForBaseline != newSchedulerConfig.UserBehaviorAnalysis.MinDataPointsForBaseline {
		s.userLastRequest = make(map[string]time.Time)
		zap.L().Info("User behavior analysis settings changed, clearing request history")
	}

	// Log configuration update
	zap.L().Info("Scheduler configuration updated",
		zap.Int("worker_pool_size", newSchedulerConfig.WorkerPoolSize),
		zap.Bool("message_classification_enabled", newSchedulerConfig.MessageClassification.Enabled),
		zap.Bool("user_behavior_analysis_enabled", newSchedulerConfig.UserBehaviorAnalysis.Enabled),
		zap.Int("base_priority", newSchedulerConfig.PrioritySettings.BasePriority))
}

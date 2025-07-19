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
	"qqbotrouter/load"
	"qqbotrouter/stats"
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
	pq              PriorityQueue
	statsAnalyzer   *stats.StatsAnalyzer
	schedulerConfig *config.SchedulerConfig
	qosConfig       *config.QoSConfig
	loadCounter     *load.Counter
	workerPool      chan *Request
	userLastRequest map[string]time.Time // Track last request time per user
	mu              sync.RWMutex         // Protect userLastRequest map
}

// NewScheduler creates a new Scheduler.
func NewScheduler(statsAnalyzer *stats.StatsAnalyzer, schedulerConfig *config.SchedulerConfig, qosConfig *config.QoSConfig, loadCounter *load.Counter) *Scheduler {
	s := &Scheduler{
		pq:              make(PriorityQueue, 0),
		statsAnalyzer:   statsAnalyzer,
		schedulerConfig: schedulerConfig,
		qosConfig:       qosConfig,
		loadCounter:     loadCounter,
		workerPool:      make(chan *Request, schedulerConfig.WorkerPoolSize),
		userLastRequest: make(map[string]time.Time),
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

// Run starts the scheduler's processing loop.
func (s *Scheduler) Run() {
	for i := 0; i < s.schedulerConfig.WorkerPoolSize; i++ {
		go s.worker()
	}

	for {
		if len(s.pq) > 0 {
			request := heap.Pop(&s.pq).(*Request)
			s.workerPool <- request
		} else {
			// Prevent busy-waiting when the queue is empty
			idleInterval := s.qosConfig.ParseDuration(s.qosConfig.RequestTimeouts.IdleCheckInterval)
			time.Sleep(idleInterval)
		}
	}
}

// parseMessage extracts user ID and message content from request body
func (s *Scheduler) parseMessage(body []byte) (userID, message string) {
	return utils.ParseMessage(body)
}

// calculatePriority calculates request priority based on user behavior and system load
func (s *Scheduler) calculatePriority(userID, message string) int {
	// 使用配置文件中的优先级设置，而不是硬编码值
	basePriority := s.schedulerConfig.PrioritySettings.BasePriority

	// Factor 1: System load adjustment
	currentLoad := s.loadCounter.Get()
	maxLoad := int64(s.qosConfig.SystemLimits.MaxLoad)
	if currentLoad > maxLoad {
		basePriority += s.schedulerConfig.PrioritySettings.HighLoadAdjustment
	} else if currentLoad < maxLoad/10 {
		basePriority += s.schedulerConfig.PrioritySettings.LowLoadAdjustment
	}

	// Factor 2: Anti-spam detection using P50 baseline
	now := time.Now()
	s.mu.Lock()
	lastRequestTime, exists := s.userLastRequest[userID]
	s.userLastRequest[userID] = now
	s.mu.Unlock()

	if exists {
		requestInterval := now.Sub(lastRequestTime)
		p50Baseline := s.statsAnalyzer.P50()

		// If user's request interval is much shorter than P50 baseline, consider it spam
		if p50Baseline > 0 && requestInterval < p50Baseline/3 {
			basePriority -= s.schedulerConfig.PrioritySettings.HighLoadAdjustment * 2 // Significant penalty for potential spam
		} else if p50Baseline > 0 && requestInterval < p50Baseline/2 {
			basePriority -= s.schedulerConfig.PrioritySettings.HighLoadAdjustment // Moderate penalty for fast requests
		}
	}

	// Factor 3: Message pattern analysis
	if s.schedulerConfig.MessageClassification.Enabled {
		if s.schedulerConfig.MessageClassification.SpamDetection && utils.IsSpamPattern(message, s.schedulerConfig.MessageClassification.SpamKeywords) {
			basePriority = s.schedulerConfig.PrioritySettings.MinPriority // Lowest priority for spam
		} else if utils.IsHighPriorityMessage(message, s.schedulerConfig.MessageClassification.PriorityKeywords) {
			basePriority = s.schedulerConfig.PrioritySettings.MaxPriority // Highest priority for important messages
		}
	}

	// Factor 4: User behavior analysis (simplified)
	if utils.IsFastUser(userID) {
		basePriority += s.schedulerConfig.PrioritySettings.FastUserBonus // Higher priority for active users
	}

	// Ensure priority is within valid range
	if basePriority < s.schedulerConfig.PrioritySettings.MinPriority {
		basePriority = s.schedulerConfig.PrioritySettings.MinPriority
	} else if basePriority > s.schedulerConfig.PrioritySettings.MaxPriority {
		basePriority = s.schedulerConfig.PrioritySettings.MaxPriority
	}

	return basePriority
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
			s.loadCounter,
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

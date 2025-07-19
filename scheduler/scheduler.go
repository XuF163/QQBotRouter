package scheduler

import (
	"container/heap"
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/forwarder"
	"qqbotrouter/load"
	"qqbotrouter/stats"
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
	pq            PriorityQueue
	statsAnalyzer *stats.StatsAnalyzer
	config        *config.CognitiveScheduling
	loadCounter   *load.Counter
	workerPool    chan *Request
}

// NewScheduler creates a new Scheduler.
func NewScheduler(statsAnalyzer *stats.StatsAnalyzer, config *config.CognitiveScheduling, loadCounter *load.Counter) *Scheduler {
	s := &Scheduler{
		pq:            make(PriorityQueue, 0),
		statsAnalyzer: statsAnalyzer,
		config:        config,
		loadCounter:   loadCounter,
		workerPool:    make(chan *Request, config.WorkerPoolSize),
	}
	heap.Init(&s.pq)
	return s
}

// Submit submits a new request to the scheduler.
func (s *Scheduler) Submit(ctx context.Context, body []byte, header http.Header, botConfig config.BotConfig, logger *zap.Logger) {
	// TODO: Calculate priority based on stats and config
	request := &Request{
		Context:   ctx,
		Body:      body,
		Header:    header,
		BotConfig: botConfig,
		Logger:    logger,
		priority:  1, // Default priority
	}
	heap.Push(&s.pq, request)
}

// Run starts the scheduler's processing loop.
func (s *Scheduler) Run() {
	for i := 0; i < s.config.WorkerPoolSize; i++ {
		go s.worker()
	}

	for {
		if len(s.pq) > 0 {
			request := heap.Pop(&s.pq).(*Request)
			s.workerPool <- request
		} else {
			// Prevent busy-waiting when the queue is empty
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// worker processes requests from the worker pool.
func (s *Scheduler) worker() {
	for request := range s.workerPool {
		// TODO: Implement regex and hash-based routing logic here.
		// For now, we only handle the default `forward_to`.
		destinations := request.BotConfig.ForwardTo

		forwarder.ForwardToMultipleDestinations(
			request.Context,
			request.Logger,
			destinations,
			request.Body,
			request.Header,
			12*time.Second, // timeout
			s.loadCounter,
		)
	}
}

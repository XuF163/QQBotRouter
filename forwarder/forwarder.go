package forwarder

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/load"
)

// ForwardResult represents the result of a forward operation
type ForwardResult struct {
	Destination string
	Success     bool
	StatusCode  int
	Error       error
}

// sendResult safely sends a result to the channel or handles context cancellation
func sendResult(ctx context.Context, resultChan chan<- ForwardResult, result ForwardResult) {
	select {
	case resultChan <- result:
	case <-ctx.Done():
	}
}

// ForwardRequestWithResult forwards the request and returns the result via channel
func ForwardRequestWithResult(ctx context.Context, logger *zap.Logger, destination string, body []byte, header http.Header, resultChan chan<- ForwardResult, loadCounter *load.Counter, forwardTimeout time.Duration) {
	loadCounter.Increment()
	defer loadCounter.Decrement()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in ForwardRequestWithResult",
				zap.String("destination", destination),
				zap.Any("panic", r))
			sendResult(ctx, resultChan, ForwardResult{Destination: destination, Success: false, Error: nil})
		}
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", destination, bytes.NewReader(body))
	if err != nil {
		logger.Error("Failed to create forward request",
			zap.String("destination", destination),
			zap.Error(err))
		sendResult(ctx, resultChan, ForwardResult{Destination: destination, Success: false, Error: err})
		return
	}
	req.Header = header.Clone()

	client := &http.Client{Timeout: forwardTimeout}
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("Failed to forward request",
			zap.String("destination", destination),
			zap.Error(err))
		sendResult(ctx, resultChan, ForwardResult{Destination: destination, Success: false, Error: err})
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if success {
		logger.Info("Successfully forwarded request",
			zap.String("destination", destination),
			zap.Int("status_code", resp.StatusCode))
	} else {
		logger.Warn("Forward request returned error status",
			zap.String("destination", destination),
			zap.Int("status_code", resp.StatusCode))
	}

	sendResult(ctx, resultChan, ForwardResult{
		Destination: destination,
		Success:     success,
		StatusCode:  resp.StatusCode,
		Error:       nil,
	})
}

// ForwardToMultipleDestinations forwards to multiple destinations and waits for all results
func ForwardToMultipleDestinations(ctx context.Context, logger *zap.Logger, destinations []string, body []byte, header http.Header, timeout time.Duration, loadCounter *load.Counter, forwardTimeout time.Duration) []ForwardResult {
	if len(destinations) == 0 {
		return []ForwardResult{}
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan ForwardResult, len(destinations))
	var wg sync.WaitGroup

	// Start all forward operations
	for _, dest := range destinations {
		wg.Add(1)
		go func(destination string) {
			defer wg.Done()
			ForwardRequestWithResult(ctxWithTimeout, logger, destination, body, header, resultChan, loadCounter, forwardTimeout)
		}(dest)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []ForwardResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

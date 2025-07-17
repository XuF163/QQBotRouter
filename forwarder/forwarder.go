package forwarder

import (
	"bytes"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ForwardRequest asynchronously forwards the request to a destination.
func ForwardRequest(logger *zap.Logger, destination string, body []byte, header http.Header) {
	req, err := http.NewRequest("POST", destination, bytes.NewReader(body))
	if err != nil {
		logger.Error("Failed to create forward request",
			zap.String("destination", destination),
			zap.Error(err))
		return
	}
	req.Header = header.Clone()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to forward request",
			zap.String("destination", destination),
			zap.Error(err))
		return
	}
	defer resp.Body.Close()
	logger.Info("Successfully forwarded request",
		zap.String("destination", destination),
		zap.Int("status_code", resp.StatusCode))
}
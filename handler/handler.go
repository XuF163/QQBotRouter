package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/qos"
	"qqbotrouter/scheduler"
	"qqbotrouter/utils"
)

// WebhookHandler is the main handler for all incoming webhook requests.
type WebhookHandler struct {
	logger     *zap.Logger
	scheduler  *scheduler.Scheduler
	qosManager *qos.QoSManager
}

// writeJSONResponse writes a JSON response with the given status code and payload
func (h *WebhookHandler) writeJSONResponse(rw http.ResponseWriter, statusCode int, payload []byte) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	if _, err := rw.Write(payload); err != nil {
		h.logger.Error("Failed to write JSON response",
			zap.Int("status_code", statusCode),
			zap.Error(err))
	}
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(logger *zap.Logger, scheduler *scheduler.Scheduler, qosManager *qos.QoSManager) *WebhookHandler {
	return &WebhookHandler{
		logger:     logger,
		scheduler:  scheduler,
		qosManager: qosManager,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *WebhookHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// 1. Read the raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(rw, "Failed to read body", http.StatusInternalServerError)
		return
	}
	// Restore the body so it can be read again later
	r.Body = io.NopCloser(bytes.NewReader(body))

	// 2. Get bot configuration for the requested host and path
	bot, ok := config.GetBotConfigFromRequest(r.Host, r.URL.Path)
	if !ok {
		h.logger.Warn("Unauthorized: No bot configured",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 3. Verify the signature (mandatory for all requests)
	if !VerifySignature(h.logger, r.Header, body, bot.Secret) {
		h.logger.Warn("Unauthorized: Signature verification failed",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 4. Parse the packet to determine the operation
	var packet WebhookPacket
	if err := json.Unmarshal(body, &packet); err != nil {
		h.logger.Error("Failed to parse webhook packet", zap.Error(err))
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	// 5. Handle the request based on the operation code
	switch packet.Op {
	case OpLegacyChallenge, OpCallbackValidation:
		h.logger.Info("Handling challenge request",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		HandleChallenge(h.logger, rw, r, packet.D, bot.Secret)
	case OpEventDispatch:
		startTime := time.Now()
		h.logger.Info("Handling event dispatch",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path),
			zap.String("message_content", string(body)))

		// Extract user information for QoS analysis
		msgInfo := utils.ExtractMessageInfo(body)

		// Calculate priority based on message content and user behavior
		priority := utils.CalculateMessagePriority(msgInfo.UserID, msgInfo.Message)

		// Check if request should be throttled
		if h.qosManager.ShouldThrottle(msgInfo.UserID, priority) {
			h.logger.Warn("Request throttled by QoS",
				zap.String("user_id", msgInfo.UserID),
				zap.Int("priority", priority))

			// Return throttled response
			ackResponse := GenDispatchACK(false) // Indicate processing failed
			h.writeJSONResponse(rw, http.StatusTooManyRequests, ackResponse)

			// Update QoS metrics
			h.qosManager.UpdateMetrics(time.Since(startTime), false)
			return
		}

		// Immediately acknowledge the request
		ackResponse := GenDispatchACK(true)
		h.writeJSONResponse(rw, http.StatusOK, ackResponse)

		// Submit the request to the scheduler for asynchronous processing
		go func() {
			processingStart := time.Now()
			success := h.scheduler.Submit(r.Context(), body, r.Header, bot, h.logger)
			processingTime := time.Since(processingStart)

			// Update QoS metrics with processing results
			h.qosManager.UpdateMetrics(processingTime, success)
		}()

	case OpHeartbeat:
		h.logger.Info("Received Heartbeat",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))

		var seq uint32
		if err := json.Unmarshal(packet.D, &seq); err != nil {
			h.logger.Warn("Failed to parse heartbeat sequence, using 0", zap.Error(err))
			seq = 0
		}

		heartbeatACK := GenHeartbeatACK(seq)
		h.writeJSONResponse(rw, http.StatusOK, heartbeatACK)
	default:
		h.logger.Warn("Received unknown op code",
			zap.Int("op_code", packet.Op),
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		rw.WriteHeader(http.StatusOK) // Acknowledge to be safe
	}
}

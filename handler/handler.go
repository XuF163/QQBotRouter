package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/scheduler"
)

// WebhookHandler is the main handler for all incoming webhook requests.
type WebhookHandler struct {
	logger    *zap.Logger
	scheduler *scheduler.Scheduler
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(logger *zap.Logger, scheduler *scheduler.Scheduler) *WebhookHandler {
	return &WebhookHandler{
		logger:    logger,
		scheduler: scheduler,
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
		h.logger.Info("Handling event dispatch",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path),
			zap.String("message_content", string(body)))

		// Immediately acknowledge the request
		ackResponse := GenDispatchACK(true)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		if _, err := rw.Write(ackResponse); err != nil {
			h.logger.Error("Failed to write ACK response", zap.Error(err))
		}

		// Submit the request to the scheduler for asynchronous processing
		h.scheduler.Submit(r.Context(), body, r.Header, bot, h.logger)

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
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		if _, err := rw.Write(heartbeatACK); err != nil {
			h.logger.Error("Failed to write heartbeat ACK", zap.Error(err))
		}
	default:
		h.logger.Warn("Received unknown op code",
			zap.Int("op_code", packet.Op),
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		rw.WriteHeader(http.StatusOK) // Acknowledge to be safe
	}
}

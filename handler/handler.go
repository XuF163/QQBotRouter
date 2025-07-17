package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/forwarder"
)

// WebhookHandler is the main handler for all incoming webhook requests.
// It now takes a logger instance.
type WebhookHandler struct {
	logger *zap.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{logger: logger}
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
	bot, ok := config.GetBotConfig(r.Host, r.URL.Path)
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
	case 1: // Challenge-Response
		h.logger.Info("Handling challenge request",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		HandleChallenge(h.logger, rw, r, packet.D, bot.Secret)
	case 0: // Event Dispatch
		h.logger.Info("Handling event dispatch",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		rw.WriteHeader(http.StatusOK)
		for _, dest := range bot.ForwardTo {
			go forwarder.ForwardRequest(h.logger, dest, body, r.Header)
		}
	case 11: // Heartbeat
		h.logger.Info("Received Heartbeat",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		rw.WriteHeader(http.StatusOK)
	default:
		h.logger.Warn("Received unknown op code",
			zap.Int("op_code", packet.Op),
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		rw.WriteHeader(http.StatusOK) // Acknowledge to be safe
	}
}
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

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
	case OpLegacyChallenge: // Challenge-Response (legacy)
		h.logger.Info("Handling legacy challenge request",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		HandleChallenge(h.logger, rw, r, packet.D, bot.Secret)
	case OpCallbackValidation: // QQ Official Callback Validation
		h.logger.Info("Handling QQ official callback validation",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		HandleChallenge(h.logger, rw, r, packet.D, bot.Secret)
	case OpEventDispatch: // Event Dispatch
		// Format JSON for better readability
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
			h.logger.Info("Handling event dispatch",
				zap.String("host", r.Host),
				zap.String("path", r.URL.Path),
				zap.String("message_content", string(body)))
		} else {
			h.logger.Info("Handling event dispatch",
				zap.String("host", r.Host),
				zap.String("path", r.URL.Path),
				zap.String("message_content", "\n"+prettyJSON.String()))
		}

		// Forward to multiple destinations and wait for results
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		results := forwarder.ForwardToMultipleDestinations(ctx, h.logger, bot.ForwardTo, body, r.Header, 12*time.Second)

		// Check if at least one forward was successful
		anySuccess := false
		successCount := 0
		for _, result := range results {
			if result.Success {
				anySuccess = true
				successCount++
			}
		}

		h.logger.Info("Forward results summary",
			zap.Int("total_destinations", len(bot.ForwardTo)),
			zap.Int("successful_forwards", successCount),
			zap.Bool("any_success", anySuccess))

		// Generate ACK response based on forward results
		ackResponse := GenDispatchACK(anySuccess)

		// Set content type and write ACK response
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		if anySuccess {
			h.logger.Info("Returning success ACK to platform",
				zap.String("host", r.Host),
				zap.String("path", r.URL.Path),
				zap.Int("successful_forwards", successCount))
		} else {
			h.logger.Warn("All forwards failed, but returning success ACK to prevent platform retry",
				zap.String("host", r.Host),
				zap.String("path", r.URL.Path),
				zap.Strings("failed_destinations", bot.ForwardTo))
		}

		if _, err := rw.Write(ackResponse); err != nil {
			h.logger.Error("Failed to write ACK response",
				zap.String("host", r.Host),
				zap.String("path", r.URL.Path),
				zap.Error(err))
		}
	case OpHeartbeat: // Heartbeat
		h.logger.Info("Received Heartbeat",
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))

		// Extract sequence number from packet data
		var seq uint32
		if err := json.Unmarshal(packet.D, &seq); err != nil {
			h.logger.Warn("Failed to parse heartbeat sequence, using 0",
				zap.Error(err))
			seq = 0
		}

		// Generate and send heartbeat ACK
		heartbeatACK := GenHeartbeatACK(seq)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		if _, err := rw.Write(heartbeatACK); err != nil {
			h.logger.Error("Failed to write heartbeat ACK",
				zap.Error(err))
		}
	default:
		h.logger.Warn("Received unknown op code",
			zap.Int("op_code", packet.Op),
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path))
		rw.WriteHeader(http.StatusOK) // Acknowledge to be safe
	}
}

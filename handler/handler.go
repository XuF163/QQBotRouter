package handler

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"qqbotrouter/config"
	"qqbotrouter/qos"
	"qqbotrouter/scheduler"
)

// WebhookHandler is the main handler for all incoming webhook requests.
type WebhookHandler struct {
	logger     *zap.Logger
	scheduler  *scheduler.Scheduler
	qosManager *qos.QoSManager
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
		userID, message := h.extractUserInfo(body)

		// Calculate priority based on message content and user behavior
		priority := h.calculateMessagePriority(userID, message)

		// Check if request should be throttled
		if h.qosManager.ShouldThrottle(userID, priority) {
			h.logger.Warn("Request throttled by QoS",
				zap.String("user_id", userID),
				zap.Int("priority", priority))

			// Return throttled response
			ackResponse := GenDispatchACK(false) // Indicate processing failed
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusTooManyRequests)
			if _, err := rw.Write(ackResponse); err != nil {
				h.logger.Error("Failed to write throttled ACK response", zap.Error(err))
			}

			// Update QoS metrics
			h.qosManager.UpdateMetrics(time.Since(startTime), false)
			return
		}

		// Immediately acknowledge the request
		ackResponse := GenDispatchACK(true)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		if _, err := rw.Write(ackResponse); err != nil {
			h.logger.Error("Failed to write ACK response", zap.Error(err))
		}

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

// extractUserInfo extracts user ID and message content from request body
func (h *WebhookHandler) extractUserInfo(body []byte) (userID, message string) {
	// Try to parse as JSON (QQ Bot webhook format)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "unknown", string(body)
	}

	// Extract user ID from different possible fields
	if author, ok := payload["author"].(map[string]interface{}); ok {
		if id, ok := author["id"].(string); ok {
			userID = id
		}
	}
	if userID == "" {
		if id, ok := payload["user_id"].(string); ok {
			userID = id
		}
	}
	if userID == "" {
		userID = "unknown"
	}

	// Extract message content
	if content, ok := payload["content"].(string); ok {
		message = content
	} else if msg, ok := payload["message"].(string); ok {
		message = msg
	} else {
		message = string(body)
	}

	return userID, message
}

// calculateMessagePriority calculates message priority based on content and user
func (h *WebhookHandler) calculateMessagePriority(userID, message string) int {
	basePriority := 5 // Default priority (1-10 scale)

	// Factor 1: Message pattern analysis
	if h.isSpamPattern(message) {
		basePriority = 1 // Lowest priority for spam
	} else if h.isHighPriorityMessage(message) {
		basePriority = 10 // Highest priority for important messages
	}

	// Factor 2: User behavior analysis (simplified)
	if h.isFastUser(userID) {
		basePriority += 2 // Higher priority for active users
	}

	// Ensure priority is within valid range
	if basePriority < 1 {
		basePriority = 1
	} else if basePriority > 10 {
		basePriority = 10
	}

	return basePriority
}

// isSpamPattern detects potential spam messages
func (h *WebhookHandler) isSpamPattern(message string) bool {
	// Simple spam detection patterns
	spamPatterns := []string{
		"重复", "刷屏", "广告", "推广",
		"spam", "advertisement", "promotion",
	}

	messageLower := strings.ToLower(message)
	for _, pattern := range spamPatterns {
		if strings.Contains(messageLower, pattern) {
			return true
		}
	}

	// Check for excessive repetition
	if len(message) > 10 {
		repeatedChars := 0
		for i := 1; i < len(message); i++ {
			if message[i] == message[i-1] {
				repeatedChars++
			}
		}
		if float64(repeatedChars)/float64(len(message)) > 0.7 {
			return true
		}
	}

	return false
}

// isHighPriorityMessage detects high priority messages
func (h *WebhookHandler) isHighPriorityMessage(message string) bool {
	highPriorityPatterns := []string{
		"紧急", "重要", "帮助", "问题", "错误",
		"urgent", "important", "help", "error", "issue",
	}

	messageLower := strings.ToLower(message)
	for _, pattern := range highPriorityPatterns {
		if strings.Contains(messageLower, pattern) {
			return true
		}
	}

	return false
}

// isFastUser determines if a user is a fast/active user (simplified implementation)
func (h *WebhookHandler) isFastUser(userID string) bool {
	// This is a simplified implementation
	// In a real system, this would check user behavior history
	hash := md5.Sum([]byte(userID))
	// Use hash to create consistent but pseudo-random classification
	return hash[0]%4 == 0 // 25% of users are considered "fast"
}

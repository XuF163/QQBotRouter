package utils

import (
	"crypto/md5"
	"encoding/json"
	"strings"
)

// MessageInfo contains extracted message information
type MessageInfo struct {
	UserID  string
	Message string
}

// ParseMessage extracts user ID and message content from request body (returns separate values)
func ParseMessage(body []byte) (string, string) {
	msgInfo := ExtractMessageInfo(body)
	return msgInfo.UserID, msgInfo.Message
}

// ExtractMessageInfo extracts user ID and message content from request body
func ExtractMessageInfo(body []byte) MessageInfo {
	// Try to parse as JSON (QQ Bot webhook format)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return MessageInfo{UserID: "unknown", Message: string(body)}
	}

	userID := "unknown"
	message := string(body)

	// Extract user ID from different possible fields
	if author, ok := payload["author"].(map[string]interface{}); ok {
		if id, ok := author["id"].(string); ok {
			userID = id
		}
	}
	if userID == "unknown" {
		if id, ok := payload["user_id"].(string); ok {
			userID = id
		}
	}

	// Extract message content
	if content, ok := payload["content"].(string); ok {
		message = content
	} else if msg, ok := payload["message"].(string); ok {
		message = msg
	}

	return MessageInfo{UserID: userID, Message: message}
}

// IsSpamPattern detects potential spam messages
func IsSpamPattern(message string) bool {
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

// IsHighPriorityMessage detects high priority messages
func IsHighPriorityMessage(message string) bool {
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

// IsFastUser determines if a user is a fast/active user (simplified implementation)
func IsFastUser(userID string) bool {
	// This is a simplified implementation
	// In a real system, this would check user behavior history
	hash := md5.Sum([]byte(userID))
	// Use hash to create consistent but pseudo-random classification
	return hash[0]%4 == 0 // 25% of users are considered "fast"
}

// CalculateMessagePriority calculates message priority based on content and user
func CalculateMessagePriority(userID, message string) int {
	basePriority := 5 // Default priority (1-10 scale)

	// Factor 1: Message pattern analysis
	if IsSpamPattern(message) {
		basePriority = 1 // Lowest priority for spam
	} else if IsHighPriorityMessage(message) {
		basePriority = 10 // Highest priority for important messages
	}

	// Factor 2: User behavior analysis (simplified)
	if IsFastUser(userID) {
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

package handler

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// --- Structs for QQ Bot Webhook Payloads ---
type WebhookPacket struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
}

type ChallengeData struct {
	PlainToken string `json:"plain_token"`
	EventTs    string `json:"event_ts"`
}

type ChallengeResponse struct {
	PlainToken string `json:"plain_token"`
	Signature  string `json:"signature"`
}

// ACK response structure for event dispatch
type ACKResponse struct {
	Op   int    `json:"op"`
	Data uint32 `json:"d"`
}

// OpCode constants
const (
	OpHeartbeat          = 11 // Heartbeat
	OpHeartbeatACK       = 12 // Heartbeat ACK
	OpEventDispatch      = 0  // Event Dispatch
	OpHTTPCallbackACK    = 12 // HTTP Callback ACK
	OpCallbackValidation = 13 // Callback Validation
	OpLegacyChallenge    = 1  // Legacy Challenge
)

// GenDispatchACK generates ACK response for event dispatch
// Always returns d=0 to indicate success, preventing platform from retrying
// even when all forwards fail, to avoid repeated pushes that could crash the project
func GenDispatchACK(success bool) []byte {
	// Always return success (data = 0) to prevent platform retry
	// This avoids repeated pushes when all forward destinations fail
	ack := ACKResponse{
		Op:   OpHTTPCallbackACK,
		Data: 0, // Always success to prevent retry
	}

	response, _ := json.Marshal(ack)
	return response
}

// GenHeartbeatACK generates ACK response for heartbeat
func GenHeartbeatACK(seq uint32) []byte {
	ack := ACKResponse{
		Op:   OpHeartbeatACK,
		Data: seq,
	}

	response, _ := json.Marshal(ack)
	return response
}

// generateSeed generates a 32-byte seed from the secret according to QQ official documentation
func generateSeed(secret string) []byte {
	// 按照QQ官方文档的逻辑：重复secret直到长度达到32字节
	seed := secret
	for len(seed) < ed25519.SeedSize {
		seed = strings.Repeat(seed, 2)
	}
	// 截取前32字节
	return []byte(seed[:ed25519.SeedSize])
}

// VerifySignature checks the ed25519 signature from the request headers.
func VerifySignature(logger *zap.Logger, header http.Header, body []byte, secret string) bool {
	timestamp := header.Get("x-signature-timestamp")
	signatureHex := header.Get("x-signature-ed25519")
	if timestamp == "" || signatureHex == "" {
		logger.Debug("Missing signature headers",
			zap.String("timestamp_header", timestamp),
			zap.String("signature_header", signatureHex))
		return false
	}

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		logger.Error("Error decoding signature", zap.Error(err))
		return false
	}

	// 按照QQ官方文档生成seed
	seed := generateSeed(secret)
	pubKey := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)

	message := []byte(timestamp + string(body))
	return ed25519.Verify(pubKey, message, signature)
}

// HandleChallenge handles the OpCode 13 (QQ official validation) challenge.
func HandleChallenge(logger *zap.Logger, rw http.ResponseWriter, r *http.Request, data json.RawMessage, secret string) {
	var challengeData ChallengeData
	if err := json.Unmarshal(data, &challengeData); err != nil {
		logger.Error("Failed to parse challenge data", zap.Error(err))
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	// 按照QQ官方文档：event_ts + plain_token
	message := []byte(challengeData.EventTs + challengeData.PlainToken)

	// 按照QQ官方文档生成seed
	seed := generateSeed(secret)
	// 使用ed25519.GenerateKey的方式生成私钥
	reader := strings.NewReader(string(seed))
	_, privKey, err := ed25519.GenerateKey(reader)
	if err != nil {
		logger.Error("Failed to generate ed25519 key", zap.Error(err))
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	signature := ed25519.Sign(privKey, message)

	resp := ChallengeResponse{
		PlainToken: challengeData.PlainToken,
		Signature:  hex.EncodeToString(signature),
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(resp)
}

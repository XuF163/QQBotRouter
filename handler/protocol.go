package handler

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// --- Structs for QQ Bot Webhook Payloads ---
type WebhookPacket struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
}

type ChallengeData struct {
	PlainToken string `json:"plain_token"`
}

type ChallengeResponse struct {
	PlainToken string `json:"plain_token"`
	Signature  string `json:"signature"`
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

	seed := []byte(secret)
	if len(seed) != ed25519.SeedSize {
		logger.Error("Invalid secret size for verification",
			zap.Int("expected_size", ed25519.SeedSize),
			zap.Int("actual_size", len(seed)))
		return false
	}
	pubKey := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)

	message := []byte(timestamp + string(body))
	return ed25519.Verify(pubKey, message, signature)
}

// HandleChallenge handles the OpCode 1 (SIGN_VERIFY) challenge.
func HandleChallenge(logger *zap.Logger, rw http.ResponseWriter, r *http.Request, data json.RawMessage, secret string) {
	var challengeData ChallengeData
	if err := json.Unmarshal(data, &challengeData); err != nil {
		logger.Error("Failed to parse challenge data", zap.Error(err))
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	timestamp := r.Header.Get("x-signature-timestamp")
	message := []byte(timestamp + challengeData.PlainToken)

	seed := []byte(secret)
	if len(seed) != ed25519.SeedSize {
		logger.Error("Invalid secret size for signing",
			zap.Int("expected_size", ed25519.SeedSize),
			zap.Int("actual_size", len(seed)))
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	privKey := ed25519.NewKeyFromSeed(seed)

	signature := ed25519.Sign(privKey, message)

	resp := ChallengeResponse{
		PlainToken: challengeData.PlainToken,
		Signature:  hex.EncodeToString(signature),
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(resp)
}
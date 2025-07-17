package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

// Payload 是接收到的请求结构
type Payload struct {
	Data json.RawMessage `json:"d"`
}

// ValidationRequest 是解析出的校验请求结构
type ValidationRequest struct {
	PlainToken string `json:"plain_token"`
	EventTs    string `json:"event_ts"`
}

// ValidationResponse 是返回的校验响应结构
type ValidationResponse struct {
	PlainToken string `json:"plain_token"`
	Signature  string `json:"signature"`
}

// handleValidation 处理回调的验证请求
func handleValidation(rw http.ResponseWriter, r *http.Request, botSecret string) {
	// 打印请求头信息
	log.Println("Request Headers:")
	for name, values := range r.Header {
		log.Printf("%s: %v", name, values)
	}

	// 打印请求方法、URL 和协议版本
	log.Printf("Request Method: %s", r.Method)
	log.Printf("Request URL: %s", r.URL)
	log.Printf("Request Protocol: %s", r.Proto)

	// 读取并打印请求体
	httpBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("read http body err", err)
		http.Error(rw, "Failed to read body", http.StatusInternalServerError)
		return
	}
	// 重置请求体，以便后续使用
	r.Body = io.NopCloser(bytes.NewReader(httpBody))

	log.Printf("Request Body: %s", string(httpBody))

	// 解析请求体
	payload := &Payload{}
	if err = json.Unmarshal(httpBody, payload); err != nil {
		log.Println("parse http payload err", err)
		http.Error(rw, "Failed to parse body", http.StatusInternalServerError)
		return
	}

	// 解析Data部分
	validationPayload := &ValidationRequest{}
	if err = json.Unmarshal(payload.Data, validationPayload); err != nil {
		log.Println("parse validation payload failed:", err)
		http.Error(rw, "Failed to parse validation data", http.StatusInternalServerError)
		return
	}

	// 准备签名所需的私钥
	seed := botSecret
	for len(seed) < ed25519.SeedSize {
		seed = strings.Repeat(seed, 2)
	}
	seed = seed[:ed25519.SeedSize]
	reader := strings.NewReader(seed)
	_, privateKey, err := ed25519.GenerateKey(reader)
	if err != nil {
		log.Println("ed25519 generate key failed:", err)
		http.Error(rw, "Failed to generate private key", http.StatusInternalServerError)
		return
	}

	// 拼接消息，计算签名
	var msg bytes.Buffer
	msg.WriteString(validationPayload.EventTs)
	msg.WriteString(validationPayload.PlainToken)
	signature := hex.EncodeToString(ed25519.Sign(privateKey, msg.Bytes()))

	// 创建响应
	rsp := &ValidationResponse{
		PlainToken: validationPayload.PlainToken,
		Signature:  signature,
	}

	// 返回响应
	rspBytes, err := json.Marshal(rsp)
	if err != nil {
		log.Println("handle validation failed:", err)
		http.Error(rw, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	// 设置响应头部和返回
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(rspBytes)
}

func main() {
	// 设置botSecret，实际使用时需要替换成你的秘钥
	botSecret := "4Qm8VsFczMk8WuIg5UtIh6WwMmCc2TuL"

	// 创建HTTP服务，监听回调请求
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleValidation(w, r, botSecret)
	})

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}



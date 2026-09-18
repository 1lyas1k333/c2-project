// Package jwt предоставляет функции для создания и парсинга JWT-токенов.
// Используется для маскировки передаваемых данных под аутентификационные токены.
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTHeader - заголовок JWT-токена
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// JWTPayload - полезная нагрузка JWT-токена
type JWTPayload struct {
	Sub  string `json:"sub,omitempty"`
	Name string `json:"name,omitempty"`
	Iat  int64  `json:"iat"`
	Data string `json:"data,omitempty"`
	Exp  int64  `json:"exp,omitempty"`
}

// getSecret - возвращает секретный ключ для подписи JWT.
func getSecret() []byte {
	return []byte("your-256-bit-secret-key-for-jwt-signing")
}

// Encode - создаёт JWT-токен с зашифрованными данными в поле Data
func Encode(data string) (string, error) {
	header := JWTHeader{
		Alg: "HS256",
		Typ: "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload := JWTPayload{
		Iat:  time.Now().Unix(),
		Data: data,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signatureInput := headerB64 + "." + payloadB64
	h := hmac.New(sha256.New, getSecret())
	h.Write([]byte(signatureInput))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	token := signatureInput + "." + signature
	return token, nil
}

// Decode - парсит JWT-токен и возвращает данные из поля Data
func Decode(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT token: expected 3 parts")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return "", err
	}

	return payload.Data, nil
}

// ParseBearerToken - извлекает данные из Bearer JWT-токена
func ParseBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Bearer token format")
	}
	return Decode(parts[1])
}

// EncodeClientID - создаёт JWT-токен с client_id в поле Sub.
// Используется для регистрации и опроса задач (маскировка под аутентификацию).
func EncodeClientID(clientID string) (string, error) {
	header := JWTHeader{
		Alg: "HS256",
		Typ: "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload := JWTPayload{
		Sub: clientID,
		Iat: time.Now().Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signatureInput := headerB64 + "." + payloadB64
	h := hmac.New(sha256.New, getSecret())
	h.Write([]byte(signatureInput))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	token := signatureInput + "." + signature
	return token, nil
}

// DecodeClientID - извлекает client_id из JWT-токена (поле Sub).
func DecodeClientID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT token")
	}

	signatureInput := parts[0] + "." + parts[1]
	h := hmac.New(sha256.New, getSecret())
	h.Write([]byte(signatureInput))
	expectedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if expectedSignature != parts[2] {
		return "", fmt.Errorf("invalid JWT signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return "", err
	}

	return payload.Sub, nil
}

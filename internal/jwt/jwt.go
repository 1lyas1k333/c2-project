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
	Data string `json:"data,omitempty"` // Сюда кладём зашифрованные данные
	Exp  int64  `json:"exp,omitempty"`
}

// secret - секретный ключ для подписи JWT
var secret = []byte("your-256-bit-secret-key-for-jwt-signing")

// Encode - создаёт JWT-токен с зашифрованными данными в поле Data
func Encode(data string) (string, error) {
	// Заголовок
	header := JWTHeader{
		Alg: "HS256",
		Typ: "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Полезная нагрузка
	payload := JWTPayload{
		Iat:  time.Now().Unix(),
		Data: data,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Подпись
	signatureInput := headerB64 + "." + payloadB64
	h := hmac.New(sha256.New, secret)
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

	// Декодируем payload (проверку подписи пропускаем)
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

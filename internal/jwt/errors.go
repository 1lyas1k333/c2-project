// Package jwt - ошибки пакета JWT.
package jwt

import "errors"

var (
	// ErrInvalidToken - некорректный формат токена.
	ErrInvalidToken = errors.New("invalid JWT token")

	// ErrInvalidSignature - неверная подпись токена.
	ErrInvalidSignature = errors.New("invalid JWT signature")

	// ErrInvalidBearer - некорректный формат Bearer-заголовка.
	ErrInvalidBearer = errors.New("invalid Bearer token format")
)

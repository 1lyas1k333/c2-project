// Package service - ошибки бизнес-логики.
package service

import "errors"

var (
	// ErrEmptyClientID - пустой client_id.
	ErrEmptyClientID = errors.New("empty client ID")

	// ErrTaskNotFound - задача не найдена.
	ErrTaskNotFound = errors.New("task not found")

	// ErrInvalidToken - некорректный токен.
	ErrInvalidToken = errors.New("invalid token")
)

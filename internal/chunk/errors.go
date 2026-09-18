// Package chunk - ошибки пакета фрагментации.
package chunk

import "errors"

var (
	// ErrSessionNotFound - сессия сборки не найдена.
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidChunk - некорректный чанк.
	ErrInvalidChunk = errors.New("invalid chunk")
)

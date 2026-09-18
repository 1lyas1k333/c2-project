// Package storage - ошибки пакета хранилища.
package storage

import "errors"

var (
	// ErrTaskNotFound - задача не найдена.
	ErrTaskNotFound = errors.New("task not found")

	// ErrClientNotFound - клиент не найден.
	ErrClientNotFound = errors.New("client not found")
)

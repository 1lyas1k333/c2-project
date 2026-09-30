// Package client - ошибки пакета клиента.
package client

import "errors"

var (
	// ErrRegistrationFailed - не удалось зарегистрироваться на сервере.
	ErrRegistrationFailed = errors.New("registration failed")

	// ErrPollFailed - не удалось опросить сервер на задачи.
	ErrPollFailed = errors.New("poll failed")

	// ErrChunkSendFailed - не удалось отправить один или несколько чанков.
	ErrChunkSendFailed = errors.New("chunk send failed")

	// ErrSplitFailed - не удалось разбить данные на чанки.
	ErrSplitFailed = errors.New("failed to split data")

	// ErrConfigLoadFailed - не удалось загрузить конфиг.
	ErrConfigLoadFailed = errors.New("failed to load config")
)

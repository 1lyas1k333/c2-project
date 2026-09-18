// Package compress - ошибки пакета сжатия.
package compress

import "errors"

var (
	// ErrCompressionFailed - ошибка при сжатии данных.
	ErrCompressionFailed = errors.New("compression failed")

	// ErrDecompressionFailed - ошибка при разжатии данных.
	ErrDecompressionFailed = errors.New("decompression failed")
)

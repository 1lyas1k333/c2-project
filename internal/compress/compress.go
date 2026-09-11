// Package compress предоставляет функции для сжатия и разжатия данных.
// Используется для уменьшения объёма передаваемых данных.
package compress

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Compress - сжимает данные с использованием gzip.
// Используется для уменьшения объёма передаваемых данных
// при отправке больших результатов команд.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress - разжимает данные, сжатые gzip.
// Выполняет обратную операцию к Compress.
func Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

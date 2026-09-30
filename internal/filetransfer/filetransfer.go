// Package filetransfer предоставляет функции для передачи файлов
// с подтверждением целостности через SHA-256.
package filetransfer

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrChecksumMismatch - контрольная сумма не совпала.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrFileTooLarge - файл больше допустимого размера.
	ErrFileTooLarge = errors.New("file too large")

	// ErrInvalidBase64 - некорректный base64.
	ErrInvalidBase64 = errors.New("invalid base64")
)

// MaxFileSize - максимальный размер файла (10 MB).
// Ограничение из-за того, что файл передаётся в одном поле JSON.
const MaxFileSize = 10 * 1024 * 1024

// ComputeChecksum - вычисляет SHA-256 hex для данных.
func ComputeChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// VerifyChecksum - проверяет SHA-256 данных.
func VerifyChecksum(data []byte, expected string) bool {
	return ComputeChecksum(data) == expected
}

// ReadFileAsBase64 - читает файл и возвращает base64 + checksum + имя.
func ReadFileAsBase64(path string) (data string, checksum string, name string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", fmt.Errorf("read file: %w", err)
	}

	if len(raw) > MaxFileSize {
		return "", "", "", fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, len(raw), MaxFileSize)
	}

	return base64.StdEncoding.EncodeToString(raw),
		ComputeChecksum(raw),
		filepath.Base(path),
		nil
}

// DecodeBase64 - декодирует base64 и проверяет checksum.
func DecodeBase64(encoded, expectedChecksum string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBase64, err)
	}

	if expectedChecksum != "" && !VerifyChecksum(raw, expectedChecksum) {
		return nil, ErrChecksumMismatch
	}

	return raw, nil
}

// SaveFile - сохраняет данные в файл (создаёт директории).
func SaveFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

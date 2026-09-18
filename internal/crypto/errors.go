// Package crypto - ошибки пакета шифрования.
package crypto

import "errors"

var (
	// ErrEncryptionFailed - ошибка при шифровании данных.
	ErrEncryptionFailed = errors.New("encryption failed")

	// ErrDecryptionFailed - ошибка при расшифровке данных.
	ErrDecryptionFailed = errors.New("decryption failed")

	// ErrInvalidCiphertext - некорректный формат зашифрованных данных.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

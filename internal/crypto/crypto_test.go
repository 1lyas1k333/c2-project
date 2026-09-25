package crypto

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestEncryptDecrypt_RoundTrip — базовый round-trip на разных данных.
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte("")},
		{"single byte", []byte("A")},
		{"short text", []byte("hello")},
		{"russian text", []byte("привет, мир!")},
		{"json", []byte(`{"task_id":"123","cmd":"ls -la"}`)},
		{"newlines", []byte("line1\nline2\r\nline3")},
		{"null bytes", []byte{0x00, 0x01, 0x02, 0x00, 0xFF}},
		{"binary 1KB", bytes.Repeat([]byte{0xAB}, 1024)},
		{"binary 1MB", bytes.Repeat([]byte{0xCD}, 1024*1024)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := Encrypt(tc.data)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if encrypted == "" {
				t.Fatal("Encrypt() returned empty string")
			}

			decrypted, err := Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tc.data) {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tc.data)
			}
		})
	}
}

// TestEncrypt_ProducesDifferentCiphertext — из-за случайного nonce два Encrypt
// на одних данных дают разные строки.
func TestEncrypt_ProducesDifferentCiphertext(t *testing.T) {
	data := []byte("same input")

	first, err := Encrypt(data)
	if err != nil {
		t.Fatalf("Encrypt() #1 error = %v", err)
	}

	second, err := Encrypt(data)
	if err != nil {
		t.Fatalf("Encrypt() #2 error = %v", err)
	}

	if first == second {
		t.Error("Encrypt() returned identical ciphertext for the same input; nonce is not random")
	}

	// Оба должны расшифровываться в одно и то же.
	for i, ct := range []string{first, second} {
		got, err := Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt() #%d error = %v", i+1, err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("Decrypt() #%d = %q, want %q", i+1, got, data)
		}
	}
}

// TestDecrypt_InvalidInput — на мусорных данных Decrypt возвращает ошибку, не паникует.
func TestDecrypt_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"not base64", "!!! not base64 !!!"},
		{"base64 of short bytes", "YWJj"},              // "abc" — короче nonce
		{"base64 of random bytes", "YWJjZGVmZ2hpams="}, // "abcdefghijk"
		{"valid base64, garbage content", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decrypt() panicked: %v", r)
				}
			}()

			_, err := Decrypt(tc.input)
			if err == nil {
				t.Error("Decrypt() expected error, got nil")
			}
		})
	}
}

// TestDecrypt_TooShort — специально проверяем ветку с ErrInvalidCiphertext.
func TestDecrypt_TooShort(t *testing.T) {
	// Валидный base64, но декодированные байты короче размера nonce GCM (12 байт).
	short := "YWJj" // "abc" — 3 байта
	_, err := Decrypt(short)
	if err == nil {
		t.Fatal("Decrypt() expected error on short ciphertext")
	}
	if !errors.Is(err, ErrInvalidCiphertext) {
		t.Errorf("Decrypt() error = %v, want ErrInvalidCiphertext", err)
	}
}

// TestDecrypt_TamperedCiphertext — если подменить байт в шифротексте,
// GCM-тег не сойдётся, и Decrypt вернёт ошибку.
func TestDecrypt_TamperedCiphertext(t *testing.T) {
	original := []byte("secret message")

	encrypted, err := Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Меняем последний символ base64-строки.
	tampered := encrypted[:len(encrypted)-1]
	if encrypted[len(encrypted)-1] == 'A' {
		tampered += "B"
	} else {
		tampered += "A"
	}

	_, err = Decrypt(tampered)
	if err == nil {
		t.Error("Decrypt() expected error on tampered ciphertext, got nil")
	}
}

// TestEncrypt_OutputIsBase64 — результат Encrypt должен быть валидным base64.
func TestEncrypt_OutputIsBase64(t *testing.T) {
	encrypted, err := Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Base64 StdEncoding использует A-Z, a-z, 0-9, +, / и = для паддинга.
	for _, r := range encrypted {
		if !isBase64Char(r) {
			t.Errorf("Encrypt() returned non-base64 char %q in %q", r, encrypted)
		}
	}

	// Длина должна быть кратна 4.
	if len(encrypted)%4 != 0 {
		t.Errorf("Encrypt() output length %d is not multiple of 4", len(encrypted))
	}

	// И не должно быть переносов строк.
	if strings.ContainsAny(encrypted, "\r\n") {
		t.Error("Encrypt() output contains newlines")
	}
}

func isBase64Char(r rune) bool {
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r == '+' || r == '/' || r == '='
}

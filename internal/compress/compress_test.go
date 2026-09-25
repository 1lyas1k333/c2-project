package compress

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

// TestCompressDecompress_RoundTrip — базовый round-trip на разных данных.
func TestCompressDecompress_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte("")},
		{"single byte", []byte("A")},
		{"short text", []byte("hello")},
		{"russian text", []byte("привет, мир!")},
		{"json", []byte(`{"task_id":"123","cmd":"ls -la"}`)},
		{"repeated text", []byte(strings.Repeat("abc", 1000))},
		{"newlines", []byte("line1\nline2\r\nline3")},
		{"null bytes", []byte{0x00, 0x01, 0x02, 0x00, 0xFF}},
		{"1KB random", randomBytes(1024)},
		{"1MB repeated", bytes.Repeat([]byte("AB"), 512*1024)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := Compress(tc.data)
			if err != nil {
				t.Fatalf("Compress() error = %v", err)
			}

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress() error = %v", err)
			}

			if !bytes.Equal(decompressed, tc.data) {
				t.Errorf("Decompress(Compress()) = %d bytes, want %d bytes", len(decompressed), len(tc.data))
			}
		})
	}
}

// TestCompress_RepeatedDataShrinks — на повторяющихся данных gzip реально уменьшает размер.
func TestCompress_RepeatedDataShrinks(t *testing.T) {
	data := []byte(strings.Repeat("hello world ", 10000)) // ~120 KB
	compressed, err := Compress(data)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if len(compressed) >= len(data) {
		t.Errorf("Compress() = %d bytes, want < %d bytes (repetitive data should shrink)",
			len(compressed), len(data))
	}

	t.Logf("original: %d bytes, compressed: %d bytes (%.1f%%)",
		len(data), len(compressed), float64(len(compressed))/float64(len(data))*100)
}

// TestCompress_RandomDataDoesNotShrink — на случайных данных размер может вырасти,
// и это нормально для gzip. Проверяем, что хотя бы round-trip работает.
func TestCompress_RandomDataDoesNotShrink(t *testing.T) {
	data := randomBytes(2048)
	compressed, err := Compress(data)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	// Не проверяем размер — просто что round-trip проходит.
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress() error = %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Error("round-trip failed on random data")
	}
}

// TestDecompress_InvalidInput — на мусорных данных Decompress возвращает ошибку, не паникует.
func TestDecompress_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte("")},
		{"not gzip", []byte("hello world")},
		{"random bytes", randomBytes(100)},
		{"gzip header only", []byte{0x1f, 0x8b, 0x08}}, // начальные байты gzip-заголовка
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decompress() panicked: %v", r)
				}
			}()

			_, err := Decompress(tc.data)
			if err == nil {
				t.Error("Decompress() expected error, got nil")
			}
		})
	}
}

// TestCompress_ProducesValidGzip — результат Compress начинается с gzip-магии 0x1f 0x8b.
func TestCompress_ProducesValidGzip(t *testing.T) {
	compressed, err := Compress([]byte("hello"))
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if len(compressed) < 2 {
		t.Fatalf("Compress() output too short: %d bytes", len(compressed))
	}

	// gzip-файл начинается с двух байт: 0x1f 0x8b.
	if compressed[0] != 0x1f || compressed[1] != 0x8b {
		t.Errorf("Compress() output does not start with gzip magic: got %x %x, want 1f 8b",
			compressed[0], compressed[1])
	}
}

// TestDecompress_TruncatedGzip — если обрезать валидный gzip, Decompress должен вернуть ошибку.
func TestDecompress_TruncatedGzip(t *testing.T) {
	compressed, err := Compress([]byte(strings.Repeat("data", 1000)))
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	// Обрезаем последние 10 байт.
	truncated := compressed[:len(compressed)-10]

	_, err = Decompress(truncated)
	if err == nil {
		t.Error("Decompress() expected error on truncated gzip, got nil")
	}
}

// randomBytes - возвращает n случайных байт (для тестов).
func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

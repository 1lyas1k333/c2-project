package filetransfer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeChecksum(t *testing.T) {
	// Известный SHA-256 для "hello".
	data := []byte("hello")
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	if got := ComputeChecksum(data); got != want {
		t.Errorf("ComputeChecksum() = %q, want %q", got, want)
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("test data")
	checksum := ComputeChecksum(data)

	if !VerifyChecksum(data, checksum) {
		t.Error("VerifyChecksum() = false, want true")
	}

	if VerifyChecksum([]byte("other data"), checksum) {
		t.Error("VerifyChecksum() = true for wrong data, want false")
	}
}

func TestReadFileAsBase64(t *testing.T) {
	// Создаём временный файл.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello, world!")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	data, checksum, name, err := ReadFileAsBase64(path)
	if err != nil {
		t.Fatalf("ReadFileAsBase64() error = %v", err)
	}
	if name != "test.txt" {
		t.Errorf("name = %q, want %q", name, "test.txt")
	}
	if checksum == "" {
		t.Error("checksum is empty")
	}
	if data == "" {
		t.Error("data is empty")
	}
}

func TestReadFileAsBase64_NotFound(t *testing.T) {
	_, _, _, err := ReadFileAsBase64("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("ReadFileAsBase64() expected error on missing file")
	}
}

func TestDecodeBase64_Valid(t *testing.T) {
	raw := []byte("test content")
	checksum := ComputeChecksum(raw)
	encoded := "dGVzdCBjb250ZW50" // base64 от "test content"

	got, err := DecodeBase64(encoded, checksum)
	if err != nil {
		t.Fatalf("DecodeBase64() error = %v", err)
	}
	if string(got) != string(raw) {
		t.Errorf("got %q, want %q", got, raw)
	}
}

func TestDecodeBase64_ChecksumMismatch(t *testing.T) {
	encoded := "dGVzdCBjb250ZW50"
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	_, err := DecodeBase64(encoded, wrongChecksum)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("error = %v, want ErrChecksumMismatch", err)
	}
}

func TestDecodeBase64_InvalidBase64(t *testing.T) {
	_, err := DecodeBase64("!!!not base64!!!", "")
	if !errors.Is(err, ErrInvalidBase64) {
		t.Errorf("error = %v, want ErrInvalidBase64", err)
	}
}

func TestSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.txt")
	content := []byte("saved content")

	if err := SaveFile(path, content); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("file content = %q, want %q", got, content)
	}
}

func TestReadFileAsBase64_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.bin")

	// Создаём файл больше MaxFileSize.
	big := strings.Repeat("x", MaxFileSize+1)
	if err := os.WriteFile(path, []byte(big), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := ReadFileAsBase64(path)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("error = %v, want ErrFileTooLarge", err)
	}
}

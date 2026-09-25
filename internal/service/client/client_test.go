package client

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient создаёт Service, направленный на тестовый сервер.
func newTestClient(serverURL string) *Service {
	return &Service{
		config: Config{
			ServerURL:    serverURL,
			ClientID:     "test-client",
			PollInterval: 1,
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// TestSendResult_SmallData — данные <= 1024 байт отправляются одним запросом.
func TestSendResult_SmallData(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := newTestClient(server.URL)

	err := s.sendResult("task-1", "small output", "completed")
	if err != nil {
		t.Fatalf("sendResult() error = %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server calls = %d, want 1", got)
	}
}

// TestSendResult_LargeData_Parallel — большие данные отправляются несколькими чанками.
func TestSendResult_LargeData_Parallel(t *testing.T) {
	var calls int32
	// Задержка на каждом запросе, чтобы измерить параллельность.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := newTestClient(server.URL)

	// 20 КБ данных → много чанков.
	largeOutput := strings.Repeat("x", 20000)

	start := time.Now()
	err := s.sendResult("task-1", largeOutput, "completed")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("sendResult() error = %v", err)
	}

	totalCalls := atomic.LoadInt32(&calls)
	if totalCalls < 2 {
		t.Fatalf("expected multiple chunks, got %d calls", totalCalls)
	}
	t.Logf("sent %d chunks in %v", totalCalls, elapsed)

	// Если бы отправка была последовательной — было бы >= totalCalls * 50ms.
	// Параллельно — значительно быстрее. Проверим, что быстрее "последовательного худшего случая".
	sequentialWorstCase := time.Duration(totalCalls) * 50 * time.Millisecond
	if elapsed >= sequentialWorstCase {
		t.Errorf("parallel send took %v, want < %v (sequential worst case)",
			elapsed, sequentialWorstCase)
	}
}

// TestSendResult_ErrorPropagation — если сервер отвечает 500 на один чанк,
// sendResult должен вернуть ошибку.
func TestSendResult_ErrorPropagation(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 2 {
			http.Error(w, "simulated error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := newTestClient(server.URL)

	largeOutput := strings.Repeat("x", 20000)
	err := s.sendResult("task-1", largeOutput, "completed")
	if err == nil {
		t.Error("sendResult() expected error on HTTP 500")
	}
}

// TestSendResult_EmptyOutput — пустой вывод отправляется одним запросом.
func TestSendResult_EmptyOutput(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := newTestClient(server.URL)

	if err := s.sendResult("task-1", "", "completed"); err != nil {
		t.Fatalf("sendResult() error = %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server calls = %d, want 1", got)
	}
}

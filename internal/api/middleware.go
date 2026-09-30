package api

import (
	"net/http"
	"time"

	"c2-project/internal/logger"
)

// responseWriter — обёртка над http.ResponseWriter, запоминает статус-код.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader перехватывает статус-код, который пишет хендлер.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write перехватывает запись тела, если статус не был установлен явно.
// По умолчанию Go ставит 200 при первом Write.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware — логирует все HTTP-запросы.
// Формат: method path status duration remote_addr
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Оборачиваем ResponseWriter, чтобы перехватить статус-код.
		rw := &responseWriter{ResponseWriter: w, status: 0}

		next.ServeHTTP(rw, r)

		// Если хендлер ничего не написал — считаем 200.
		if rw.status == 0 {
			rw.status = http.StatusOK
		}

		logger.Info("HTTP request",
			logger.String("method", r.Method),
			logger.String("path", r.URL.Path),
			logger.Int("status", rw.status),
			logger.String("duration", time.Since(start).String()),
			logger.String("remote", r.RemoteAddr),
		)
	})
}

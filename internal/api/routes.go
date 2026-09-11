package api

import (
	"net/http"
)

// RegisterRoutes - регистрирует все HTTP-маршруты сервера.
func RegisterRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("/api/register", handler.RegisterHandler)
	mux.HandleFunc("/api/tasks", handler.TasksHandler)
	mux.HandleFunc("/api/poll", handler.PollHandler)
	mux.HandleFunc("/api/results", handler.ResultsHandler)
	mux.HandleFunc("/api/result", handler.GetResultHandler)
}

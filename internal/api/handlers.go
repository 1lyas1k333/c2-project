package api

import (
	"c2-project/internal/service"
	"encoding/json"
	"log"
	"net/http"
)

// Handler - структура с зависимостями для хендлеров.
type Handler struct {
	taskService   *service.TaskService
	clientService *service.ClientService
}

// NewHandler - создаёт новый Handler.
func NewHandler(taskService *service.TaskService, clientService *service.ClientService) *Handler {
	return &Handler{
		taskService:   taskService,
		clientService: clientService,
	}
}

// RegisterHandler - обрабатывает регистрацию клиента.
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(w, r, http.MethodPost) {
		return
	}

	clientID := GetClientIDFromBearer(r.Header.Get("Authorization"))
	if clientID == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	if err := h.clientService.RegisterClient(clientID, r.RemoteAddr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[OK] Client registered: %s", clientID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// TasksHandler - принимает задачу от терминала.
func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(w, r, http.MethodPost) {
		return
	}

	token := GetTokenFromBearer(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	taskID, err := h.taskService.CreateTaskFromToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	log.Printf("[TASK] Task created: %s", taskID)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "task_id": taskID})
}

func (h *Handler) PollHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(w, r, http.MethodPost) {
		return
	}

	clientID := GetClientIDFromBearer(r.Header.Get("Authorization"))
	if clientID == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	// ОБНОВЛЯЕМ ВРЕМЯ ПОСЛЕДНЕГО ВИЗИТА
	h.clientService.UpdateLastSeen(clientID)

	task, token, err := h.taskService.GetPendingTaskForClient(clientID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if token == "" {
		json.NewEncoder(w).Encode(map[string]string{"status": "no_tasks"})
		return
	}

	log.Printf("[TASK] Task %s found for client %s", task.ID, clientID)
	w.Header().Set("Authorization", "Bearer "+token)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ResultsHandler - принимает результат от клиента.
func (h *Handler) ResultsHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(w, r, http.MethodPost) {
		return
	}

	token := GetTokenFromBearer(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	if err := h.taskService.SaveTaskResult(token); err != nil {
		log.Printf("[ERROR] Failed to save result: %v", err)
		http.Error(w, "Invalid result", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetResultHandler - отдаёт результат задачи по её ID.
func (h *Handler) GetResultHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(w, r, http.MethodGet) {
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	task, err := h.taskService.GetTaskResult(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"status": task.Status,
		"result": task.Result,
	}
	json.NewEncoder(w).Encode(response)
}

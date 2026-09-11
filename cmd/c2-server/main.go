// Package main - точка входа C2-сервера.
// Содержит только инициализацию конфигурации, хранилища и роутера.
// Вся бизнес-логика вынесена в пакеты internal/http и internal/storage.
package main

import (
	"c2-project/internal/api"
	"c2-project/internal/storage"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// ServerConfig - структура конфигурации сервера.
type ServerConfig struct {
	ListenAddress string `json:"listen_address"`
}

// configFile - путь к файлу конфигурации.
var configFile = "configs/server.json"

// loadConfig - загружает конфигурацию из JSON-файла.
func loadConfig() (*ServerConfig, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config ServerConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// main - точка входа сервера.
func main() {
	// Проверяем аргумент -local для локального запуска
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-local" {
			configFile = "configs/server.local.json"
			break
		}
	}

	// Загружаем конфигурацию
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаём хранилище
	store := storage.New()

	// Создаём HTTP-хендлер
	handler := api.NewHandler(store)

	// Регистрируем маршруты
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handler)

	// Запускаем HTTP-сервер
	log.Printf("C2 Server starting on %s", config.ListenAddress)
	log.Fatal(http.ListenAndServe(config.ListenAddress, mux))
}

// Package server содержит бизнес-логику C2-сервера.
// Запускает HTTP-сервер, регистрирует маршруты, управляет жизненным циклом.
package server

import (
	"c2-project/internal/api"
	"c2-project/internal/logger"
	"c2-project/internal/service"
	"c2-project/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Config - конфигурация сервера.
type Config struct {
	ListenAddress string `json:"listen_address"`
}

// Service - сервис сервера.
type Service struct {
	config     Config
	httpServer *http.Server
	store      *storage.Storage
}

// NewService - создаёт новый сервис сервера.
// Загружает конфиг и подготавливает HTTP-сервер.
func NewService(configFile string) (*Service, error) {
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

	// Создаём хранилище
	store := storage.New()

	// Создаём сервисы
	taskService := service.NewTaskService(store)
	clientService := service.NewClientService(store)

	// Создаём HTTP-хендлер
	handler := api.NewHandler(taskService, clientService)

	// Регистрируем маршруты
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handler)

	return &Service{
		config: cfg,
		store:  store,
		httpServer: &http.Server{
			Addr:    cfg.ListenAddress,
			Handler: mux,
		},
	}, nil
}

func (s *Service) Run() error {
	logger.Info("C2 Server starting", logger.String("address", s.config.ListenAddress))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Service) Close(ctx context.Context) error {
	logger.Info("Shutting down C2 Server...")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	logger.Info("C2 Server stopped gracefully")
	return nil
}

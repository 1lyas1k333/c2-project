// Package main - точка входа C2-клиента.
// Содержит только парсинг аргументов и запуск сервиса.
package main

import (
	"c2-project/internal/logger"
	"c2-project/internal/service/client"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Инициализируем логгер
	logger.Init()
	defer logger.Sync()

	// Парсим аргументы командной строки
	localFlag := flag.Bool("local", false, "Use local config (client.local.json)")
	idFlag := flag.String("id", "", "Override client ID")
	flag.Parse()

	// Определяем путь к конфигу
	configFile := "configs/client.json"
	if *localFlag {
		configFile = "configs/client.local.json"
	}

	// Создаём сервис
	srv, err := client.NewService(configFile, *idFlag)
	if err != nil {
		log.Fatalf("Failed to create client service: %v", err)
	}

	// Обработка сигналов Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запускаем клиент в горутине
	done := make(chan struct{})
	go func() {
		if err := srv.Run(); err != nil {
			log.Printf("Client error: %v", err)
		}
		close(done)
	}()

	// Ждём либо сигнала, либо завершения клиента
	select {
	case <-ctx.Done():
		log.Println("[STOP] Received shutdown signal")
	case <-done:
		log.Println("[STOP] Client finished")
	}

	// Штатное завершение
	if err := srv.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

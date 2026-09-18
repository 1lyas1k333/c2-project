// Package main - точка входа C2-сервера.
// Содержит только парсинг аргументов и запуск сервиса.
package main

import (
	"c2-project/internal/service/server"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Парсим аргументы командной строки
	localFlag := flag.Bool("local", false, "Use local config (server.local.json)")
	flag.Parse()

	// Определяем путь к конфигу
	configFile := "configs/server.json"
	if *localFlag {
		configFile = "configs/server.local.json"
	}

	// Создаём сервис
	srv, err := server.NewService(configFile)
	if err != nil {
		log.Fatalf("Failed to create server service: %v", err)
	}

	// Обработка сигналов Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запускаем сервер в горутине
	go func() {
		if err := srv.Run(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Ждём сигнала остановки
	<-ctx.Done()
	log.Println("[STOP] Received shutdown signal")

	// Штатное завершение
	if err := srv.Close(context.Background()); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

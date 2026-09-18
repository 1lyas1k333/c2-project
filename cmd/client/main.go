// Package main - точка входа C2-клиента.
// Содержит только парсинг аргументов и запуск сервиса.
package main

import (
	"c2-project/internal/service/client"
	"flag"
	"log"
)

func main() {
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

	// Запускаем
	if err := srv.Run(); err != nil {
		log.Fatalf("Client error: %v", err)
	}

	// Штатное завершение
	if err := srv.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

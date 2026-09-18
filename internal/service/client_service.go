// Package service - бизнес-логика работы с клиентами.
package service

import (
	"c2-project/internal/storage"
)

// ClientService - сервис для работы с клиентами.
type ClientService struct {
	store *storage.Storage
}

// NewClientService - создаёт новый ClientService.
func NewClientService(store *storage.Storage) *ClientService {
	return &ClientService{store: store}
}

// RegisterClient - регистрирует клиента в хранилище.
func (s *ClientService) RegisterClient(clientID, ip string) error {
	if clientID == "" {
		return ErrEmptyClientID
	}
	s.store.RegisterClient(clientID, ip)
	return nil
}

// ClientExists - проверяет, зарегистрирован ли клиент.
func (s *ClientService) ClientExists(clientID string) bool {
	_, exists := s.store.GetClient(clientID)
	return exists
}

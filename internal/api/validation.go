// Package http содержит HTTP-обработчики сервера C2.
// Включает в себя роутеры, хендлеры и валидацию запросов.
package api

import (
	"net/http"
	"strings"
)

// CheckMethod - проверяет, что HTTP-метод запроса соответствует разрешённому.
// Если метод не совпадает, возвращает false и отправляет ошибку 405.
func CheckMethod(w http.ResponseWriter, r *http.Request, allowedMethod string) bool {
	if r.Method != allowedMethod {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// GetTokenFromBearer - извлекает токен из заголовка Authorization.
// Ожидает формат: "Bearer <token>".
func GetTokenFromBearer(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// GetClientIDFromBearer - извлекает client_id из заголовка Authorization.
// Ожидает формат: "Bearer <client_id>".
func GetClientIDFromBearer(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

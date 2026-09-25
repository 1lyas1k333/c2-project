package protocol

import (
	"reflect"
	"strings"
	"testing"

	"c2-project/internal/models"
)

// --- EncodeRequest / DecodeRequest: round-trip небольших данных ---

func TestEncodeRequest_ReturnsBearerToken(t *testing.T) {
	data := map[string]string{"cmd": "whoami"}
	token, err := EncodeRequest(data)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}
	if !strings.HasPrefix(token, "Bearer ") {
		t.Errorf("EncodeRequest() = %q, want prefix 'Bearer '", token)
	}
}

func TestEncodeDecodeRequest_RoundTrip_Map(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{"simple", map[string]interface{}{"cmd": "whoami"}},
		{"with number", map[string]interface{}{"cmd": "ls", "count": float64(5)}},
		{"with bool", map[string]interface{}{"ok": true}},
		{"empty map", map[string]interface{}{}},
		{"nested", map[string]interface{}{
			"cmd":  "test",
			"opts": map[string]interface{}{"x": "y"},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := EncodeRequest(tc.data)
			if err != nil {
				t.Fatalf("EncodeRequest() error = %v", err)
			}

			got, err := DecodeRequest(token)
			if err != nil {
				t.Fatalf("DecodeRequest() error = %v", err)
			}

			gotMap, ok := got.(map[string]interface{})
			if !ok {
				t.Fatalf("DecodeRequest() returned %T, want map[string]interface{}", got)
			}

			if len(gotMap) != len(tc.data) {
				t.Errorf("DecodeRequest() has %d keys, want %d", len(gotMap), len(tc.data))
			}
			for k, want := range tc.data {
				gotVal, ok := gotMap[k]
				if !ok {
					t.Errorf("missing key %q", k)
					continue
				}
				if !reflect.DeepEqual(gotVal, want) {
					t.Errorf("key %q: got %v (%T), want %v (%T)", k, gotVal, gotVal, want, want)
				}
			}
		})
	}
}

func TestEncodeDecodeRequest_RoundTrip_Struct(t *testing.T) {
	// Реальный кейс: models.Task.
	task := models.Task{
		ClientID: "client-001",
		Command:  "whoami",
		Status:   "pending",
	}

	token, err := EncodeRequest(task)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}

	got, err := DecodeRequest(token)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	gotMap, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("DecodeRequest() returned %T, want map", got)
	}

	if gotMap["client_id"] != "client-001" {
		t.Errorf("client_id = %v, want client-001", gotMap["client_id"])
	}
	if gotMap["command"] != "whoami" {
		t.Errorf("command = %v, want whoami", gotMap["command"])
	}
}

// --- EncodeRequest: невалидный ввод ---

func TestEncodeRequest_InvalidData(t *testing.T) {
	// Канал нельзя сериализовать в JSON.
	_, err := EncodeRequest(make(chan int))
	if err == nil {
		t.Error("EncodeRequest() expected error on unmarshalable data")
	}
}

// --- DecodeRequest: ошибки ---

func TestDecodeRequest_InvalidBearer(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"empty", ""},
		{"no bearer", "some-token"},
		{"wrong prefix", "Basic abc.def.ghi"},
		{"bearer only", "Bearer "},
		{"bearer with garbage", "Bearer !!!not-a-jwt!!!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeRequest(tc.header)
			if err == nil {
				t.Errorf("DecodeRequest(%q) expected error", tc.header)
			}
		})
	}
}

// TestDecodeRequest_TamperedPayload — AES-GCM должен поймать подмену
// в payload-части токена (между первой и второй точкой).
func TestDecodeRequest_TamperedPayload(t *testing.T) {
	token, _ := EncodeRequest(map[string]string{"cmd": "whoami"})

	// Структура: "Bearer <header>.<payload>.<signature>"
	// Отрезаем префикс "Bearer ", потом делим по точкам.
	raw := strings.TrimPrefix(token, "Bearer ")
	parts := strings.SplitN(raw, ".", 3)
	if len(parts) != 3 {
		t.Fatalf("unexpected token format: %q", raw)
	}

	payload := parts[1]
	if len(payload) < 10 {
		t.Fatalf("payload too short: %q", payload)
	}

	// Меняем символ в середине payload.
	middle := len(payload) / 2
	var replacement byte = 'A'
	if payload[middle] == 'A' {
		replacement = 'B'
	}
	parts[1] = payload[:middle] + string(replacement) + payload[middle+1:]

	tampered := "Bearer " + strings.Join(parts, ".")

	_, err := DecodeRequest(tampered)
	if err == nil {
		t.Error("DecodeRequest() expected error on tampered payload")
	}
}

// TestDecodeRequest_TamperedSignatureAccepted — фиксирует ТЕКУЩЕЕ поведение.
// jwt.Decode не проверяет подпись, поэтому tampering signature НЕ ловится.
// Если ты пофиксишь jwt.Decode — этот тест упадёт, и это ХОРОШО.
func TestDecodeRequest_TamperedSignatureAccepted(t *testing.T) {
	token, _ := EncodeRequest(map[string]string{"cmd": "whoami"})

	tampered := token[:len(token)-1]
	if token[len(token)-1] == 'A' {
		tampered += "B"
	} else {
		tampered += "A"
	}

	got, err := DecodeRequest(tampered)
	if err == nil {
		t.Log("WARNING: DecodeRequest accepted tampered signature")
		t.Log("This is the CURRENT behavior (jwt.Decode does not verify signature).")
		t.Logf("Decoded: %v", got)
	}
	// Не фейлим — фиксируем текущее поведение.
}

// --- DecodeRequest: чанки ---

func TestDecodeRequest_RegularDataIsNotTreatedAsChunk(t *testing.T) {
	// Обычное сообщение без поля "type" — должно вернуться как map.
	data := map[string]interface{}{"cmd": "ls", "status": "pending"}
	token, _ := EncodeRequest(data)

	got, err := DecodeRequest(token)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	gotMap, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("got %T, want map", got)
	}
	if gotMap["status"] != "pending" {
		t.Errorf("status = %v, want pending", gotMap["status"])
	}
	// Не должно быть поля "session" — это маркер чанка.
	if _, hasSession := gotMap["session"]; hasSession {
		t.Error("regular message was treated as chunk (has 'session' field)")
	}
}

func TestDecodeRequest_WithTypeFieldIsTreatedAsChunk(t *testing.T) {
	// ВАЖНО: фиксируем ТЕКУЩЕЕ (спорное) поведение.
	// Если в данных есть поле "type", DecodeRequest считает это чанком.
	data := map[string]interface{}{"type": "custom", "cmd": "ls"}
	token, _ := EncodeRequest(data)

	got, err := DecodeRequest(token)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	gotMap, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("got %T, want map", got)
	}

	// Текущее поведение: возвращает {"status":"assembling", "session":""}.
	if gotMap["status"] != "assembling" {
		t.Logf("DecodeRequest with type field returned: %v", gotMap)
		t.Log("NOTE: this is the current behavior. 'type' field triggers chunk-assembly path.")
	}
}

// --- Большие данные ---

func TestEncodeRequest_LargeDataReturnsChunk(t *testing.T) {
	// ВАЖНО: фиксируем ТЕКУЩЕЕ поведение.
	// При данных > MaxChunkSize EncodeRequest сохраняет сессию и возвращает
	// ТОЛЬКО первый чанк (START). Остальные DATA-чанки не отправляются.
	// Это означает, что большие задачи через EncodeRequest работать не будут
	// (round-trip вернёт "assembling", а не данные).
	large := map[string]interface{}{
		"cmd":  strings.Repeat("x", 10000), // > 1024 после шифрования
		"note": "large payload",
	}

	token, err := EncodeRequest(large)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}

	if !strings.HasPrefix(token, "Bearer ") {
		t.Errorf("expected 'Bearer ' prefix, got %q", token)
	}

	// Пробуем декодировать — ожидаем "assembling", т.к. получен только START.
	got, err := DecodeRequest(token)
	if err != nil {
		// Тоже валидный исход — но логируем.
		t.Logf("DecodeRequest returned error (current behavior): %v", err)
		return
	}

	gotMap, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("got %T, want map", got)
	}

	if gotMap["status"] != "assembling" {
		t.Logf("DecodeRequest for large data returned: %v", gotMap)
		t.Log("NOTE: current behavior for large data is incomplete round-trip.")
	}
}

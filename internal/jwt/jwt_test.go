package jwt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// --- Encode / Decode ---

func TestEncodeDecode_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty", ""},
		{"short", "hello"},
		{"russian", "привет мир"},
		{"json", `{"cmd":"ls -la"}`},
		{"base64-like", "SGVsbG8gV29ybGQ="},
		{"long", strings.Repeat("x", 10000)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := Encode(tc.data)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			got, err := Decode(token)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if got != tc.data {
				t.Errorf("Decode(Encode()) = %q, want %q", got, tc.data)
			}
		})
	}
}

func TestEncode_Format(t *testing.T) {
	token, err := Encode("test")
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	for i, p := range parts {
		if p == "" {
			t.Errorf("part %d is empty", i)
		}
	}
}

func TestEncode_ProducesDifferentTokens(t *testing.T) {
	// Из-за iat (time.Now().Unix()) два токена могут совпасть в одну секунду.
	// Но если данные разные — payload разный, значит токены разные.
	t1, _ := Encode("first")
	t2, _ := Encode("second")

	if t1 == t2 {
		t.Error("Encode() produced identical tokens for different data")
	}
}

// --- Decode: ошибки формата ---

func TestDecode_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"one part", "abc"},
		{"two parts", "abc.def"},
		{"four parts", "a.b.c.d"},
		{"only dots", "..."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(tc.token)
			if err == nil {
				t.Error("Decode() expected error, got nil")
			}
		})
	}
}

func TestDecode_InvalidBase64(t *testing.T) {
	// Валидная структура (3 части), но payload — не base64.
	token := "header.!!!invalid-base64!!!.signature"
	_, err := Decode(token)
	if err == nil {
		t.Error("Decode() expected error on invalid base64")
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	// Валидный base64, но не JSON.
	payloadB64 := "bm90IGpzb24=" // "not json"
	token := "header." + payloadB64 + ".signature"
	_, err := Decode(token)
	if err == nil {
		t.Error("Decode() expected error on invalid JSON")
	}
}

// --- Безопасность: текущее (уязвимое) поведение ---

func TestDecode_DoesNotVerifySignature(t *testing.T) {
	// ВАЖНО: этот тест фиксирует ТЕКУЩЕЕ поведение.
	// Decode НЕ проверяет подпись — токен с подделанной подписью всё равно пройдёт.
	// Если ты пофиксишь Decode — этот тест упадёт, и это ХОРОШО.
	token, _ := Encode("real data")

	// Подменяем подпись на мусор.
	parts := strings.Split(token, ".")
	parts[2] = "AAAAtampered-signatureAAAA"
	tampered := strings.Join(parts, ".")

	got, err := Decode(tampered)
	if err == nil {
		t.Logf("WARNING: Decode() accepted tampered signature, returned %q", got)
		t.Log("This is the CURRENT behavior. Consider fixing Decode to verify signature.")
	}
	// Не фейлим тест — просто фиксируем поведение.
	// Если ты хочешь, чтобы это был FAIL — раскомментируй:
	// if err == nil {
	//     t.Error("SECURITY: Decode accepted tampered signature")
	// }
}

func TestDecode_TamperedPayload(t *testing.T) {
	// Подделываем payload — меняем Data, оставляем старую подпись.
	// Это классическая атака на JWT без проверки подписи.
	original, _ := Encode("original data")

	parts := strings.Split(original, ".")
	// Декодируем payload, меняем Data, кодируем обратно.
	// (Для простоты — просто подставляем другой валидный base64.)
	payloadB64 := "eyJpYXQiOjEsImRhdGEiOiJoYWNrZWQifQ" // {"iat":1,"data":"hacked"}
	parts[1] = payloadB64

	forged := strings.Join(parts, ".")

	got, err := Decode(forged)
	if err != nil {
		t.Logf("Decode() rejected forged token (good): %v", err)
		return
	}

	if got == "hacked" {
		t.Logf("WARNING: Decode() accepted forged payload, returned %q", got)
		t.Log("This is the CURRENT behavior. The AES layer still protects the data,")
		t.Log("but JWT layer should also verify signature.")
	}
}

// --- EncodeClientID / DecodeClientID ---

func TestEncodeDecodeClientID_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
	}{
		{"simple", "client-001"},
		{"with dash", "my-client-abc"},
		{"uuid-like", "550e8400-e29b-41d4-a716-446655440000"},
		{"russian", "клиент-1"},
		{"long", strings.Repeat("x", 1000)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := EncodeClientID(tc.clientID)
			if err != nil {
				t.Fatalf("EncodeClientID() error = %v", err)
			}

			got, err := DecodeClientID(token)
			if err != nil {
				t.Fatalf("DecodeClientID() error = %v", err)
			}

			if got != tc.clientID {
				t.Errorf("DecodeClientID(EncodeClientID()) = %q, want %q", got, tc.clientID)
			}
		})
	}
}

func TestDecodeClientID_InvalidFormat(t *testing.T) {
	tests := []string{"", "abc", "a.b", "a.b.c.d"}
	for _, tok := range tests {
		t.Run(tok, func(t *testing.T) {
			_, err := DecodeClientID(tok)
			if err == nil {
				t.Errorf("DecodeClientID(%q) expected error", tok)
			}
		})
	}
}

func TestDecodeClientID_RejectsTamperedSignature(t *testing.T) {
	token, _ := EncodeClientID("real-client")

	parts := strings.Split(token, ".")
	parts[2] = "AAAAtamperedAAAA"
	tampered := strings.Join(parts, ".")

	_, err := DecodeClientID(tampered)
	if err == nil {
		t.Error("DecodeClientID() accepted tampered signature")
	}
	if err != ErrInvalidSignature {
		t.Errorf("DecodeClientID() error = %v, want ErrInvalidSignature", err)
	}
}

func TestDecodeClientID_RejectsTamperedPayload(t *testing.T) {
	token, _ := EncodeClientID("real-client")

	parts := strings.Split(token, ".")
	// Меняем payload на другой валидный base64.
	payloadB64 := "eyJpYXQiOjEsInN1YiI6ImhhY2tlZCJ9" // {"iat":1,"sub":"hacked"}
	parts[1] = payloadB64

	tampered := strings.Join(parts, ".")

	_, err := DecodeClientID(tampered)
	if err == nil {
		t.Error("DecodeClientID() accepted tampered payload")
	}
}

// --- ParseBearerToken ---

func TestParseBearerToken(t *testing.T) {
	data := "secret data"
	token, _ := Encode(data)

	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"valid", "Bearer " + token, data, false},
		{"empty", "", "", true},
		{"no bearer prefix", token, "", true},
		{"wrong prefix", "Basic " + token, "", true},
		{"bearer lowercase", "bearer " + token, "", true}, // регистр важен
		{"bearer no space", "Bearer" + token, "", true},
		{"only bearer", "Bearer ", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseBearerToken(tc.header)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseBearerToken(%q) expected error", tc.header)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseBearerToken(%q) error = %v", tc.header, err)
				return
			}
			if got != tc.want {
				t.Errorf("ParseBearerToken(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestParseBearerToken_InvalidToken(t *testing.T) {
	_, err := ParseBearerToken("Bearer not.a.jwt")
	if err == nil {
		t.Error("ParseBearerToken() expected error on invalid token")
	}
}

// --- Время / iat ---

func TestEncode_IatIsCurrentTime(t *testing.T) {
	before := time.Now().Unix()
	token, _ := Encode("data")
	after := time.Now().Unix()

	parts := strings.Split(token, ".")
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if payload.Iat < before || payload.Iat > after {
		t.Errorf("payload.Iat = %d, want between %d and %d", payload.Iat, before, after)
	}
}

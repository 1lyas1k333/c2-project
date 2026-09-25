package chunk

import (
	"strings"
	"testing"
)

// --- Split ---

func TestSplit_DefaultChunkSize(t *testing.T) {
	// chunkSize <= 0 → должен подставиться 1024.
	data := strings.Repeat("a", 3000) // 3000 байт → base64 ~4000 символов
	chunks, id := Split(data, 0)
	if id == "" {
		t.Fatal("Split() returned empty session ID")
	}
	t.Cleanup(func() { ClearSession(id) })

	if len(chunks) < 2 {
		t.Fatalf("expected at least START+DATA+END, got %d chunks", len(chunks))
	}
	if chunks[0].Type != Start {
		t.Errorf("first chunk type = %q, want START", chunks[0].Type)
	}
	if chunks[len(chunks)-1].Type != End {
		t.Errorf("last chunk type = %q, want END", chunks[len(chunks)-1].Type)
	}
}

func TestSplit_NegativeChunkSize(t *testing.T) {
	data := "hello"
	chunks, id := Split(data, -5)
	t.Cleanup(func() { ClearSession(id) })

	if len(chunks) == 0 {
		t.Fatal("Split() returned no chunks for negative chunkSize")
	}
	// Не должно паниковать и должно использовать дефолт.
	if chunks[0].Type != Start {
		t.Error("expected START chunk")
	}
}

func TestSplit_Structure(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		chunkSize int
		wantData  int // ожидаемое число DATA-чанков
	}{
		{"empty data", "", 1024, 0},
		{"1 byte", "a", 1024, 1},
		{"short text", "hello world", 4, 0},           // base64 от 11 байт = 16 символов → 4 чанка
		{"exact 4 bytes", "abcd", 8, 1},               // base64 8 символов → 1 чанк
		{"large", strings.Repeat("x", 10000), 100, 0}, // посчитаем ниже динамически
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chunks, id := Split(tc.data, tc.chunkSize)
			t.Cleanup(func() { ClearSession(id) })

			if id == "" {
				t.Fatal("Split() returned empty session ID")
			}

			// Все чанки должны иметь один и тот же ID
			for i, c := range chunks {
				if c.ID != id {
					t.Errorf("chunk[%d].ID = %q, want %q", i, c.ID, id)
				}
			}

			// Последний — END
			last := chunks[len(chunks)-1]
			if last.Type != End {
				t.Errorf("last chunk type = %q, want END", last.Type)
			}

			// START (если есть DATA-чанки)
			hasStart := chunks[0].Type == Start
			if tc.wantData > 0 && !hasStart {
				t.Error("expected START chunk as first")
			}
		})
	}
}

func TestSplit_EndAlwaysPresent(t *testing.T) {
	// Пустые данные → только END (START не добавляется).
	chunks, id := Split("", 1024)
	t.Cleanup(func() { ClearSession(id) })

	if len(chunks) == 0 {
		t.Fatal("Split() returned no chunks")
	}
	if chunks[len(chunks)-1].Type != End {
		t.Errorf("last chunk type = %q, want END", chunks[len(chunks)-1].Type)
	}
}

func TestSplit_SeqIsSequential(t *testing.T) {
	data := strings.Repeat("z", 5000)
	chunks, id := Split(data, 100)
	t.Cleanup(func() { ClearSession(id) })

	dataChunks := 0
	for i, c := range chunks {
		if c.Type != Data {
			continue
		}
		dataChunks++
		// DATA-чанки идут Seq=1..N
		if c.Seq != dataChunks {
			t.Errorf("chunk[%d].Seq = %d, want %d", i, c.Seq, dataChunks)
		}
		if c.Total != len(chunks)-2 { // минус START и END
			t.Errorf("chunk[%d].Total = %d, want %d", i, c.Total, len(chunks)-2)
		}
	}
}

// --- Assemble ---

func TestAssemble_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		chunkSize int
	}{
		{"empty", "", 1024},
		{"short", "hello", 1024},
		{"exactly 100 bytes", strings.Repeat("a", 100), 200}, // base64 от 100 байт = 136 символов → 1 чанк
		{"crosses boundary", strings.Repeat("b", 1000), 100},
		{"large 10KB", strings.Repeat("c", 10000), 500},
		{"russian", "привет мир", 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chunks, id := Split(tc.data, tc.chunkSize)
			t.Cleanup(func() { ClearSession(id) })

			var result string
			var done bool
			for _, c := range chunks {
				r, d := Assemble(c)
				if d {
					result = r
					done = true
				}
			}

			if !done {
				t.Fatal("Assemble() never returned done=true")
			}
			if result != tc.data {
				t.Errorf("Assemble() = %q, want %q", result, tc.data)
			}
		})
	}
}

func TestAssemble_StartResetsSession(t *testing.T) {
	data := "hello world"
	chunks, id := Split(data, 1024)
	t.Cleanup(func() { ClearSession(id) })

	// Собираем первый START
	if _, done := Assemble(chunks[0]); done {
		t.Fatal("START should not return done=true")
	}

	// Проверяем, что сессия создалась
	sess := GetSession(id)
	if sess == nil {
		t.Fatal("session was not created")
	}
	if sess.Total != chunks[0].Total {
		t.Errorf("session.Total = %d, want %d", sess.Total, chunks[0].Total)
	}
}

func TestAssemble_EndWithoutAllData(t *testing.T) {
	data := strings.Repeat("a", 2000)
	chunks, id := Split(data, 100)
	t.Cleanup(func() { ClearSession(id) })

	// Отправляем START и только первый DATA, потом END.
	Assemble(chunks[0]) // START
	Assemble(chunks[1]) // первый DATA

	// Ищем END
	var endChunk Chunk
	for _, c := range chunks {
		if c.Type == End {
			endChunk = c
			break
		}
	}

	_, done := Assemble(endChunk)
	if done {
		t.Error("END should not complete when not all DATA received")
	}
}

func TestAssemble_OutOfOrder(t *testing.T) {
	data := strings.Repeat("x", 5000)
	chunks, id := Split(data, 200)
	t.Cleanup(func() { ClearSession(id) })

	// Перемешиваем DATA-чанки: сначала START, потом DATA в обратном порядке, потом END.
	var start, end *Chunk
	var dataChunks []Chunk
	for i := range chunks {
		switch chunks[i].Type {
		case Start:
			start = &chunks[i]
		case End:
			end = &chunks[i]
		case Data:
			dataChunks = append(dataChunks, chunks[i])
		}
	}

	if start == nil || end == nil {
		t.Fatal("START or END missing")
	}

	Assemble(*start)
	// DATA в обратном порядке
	for i := len(dataChunks) - 1; i >= 0; i-- {
		Assemble(dataChunks[i])
	}
	result, done := Assemble(*end)

	if !done {
		t.Fatal("Assemble() did not complete after all chunks")
	}
	if result != data {
		t.Errorf("Assemble() = %q (len %d), want %q (len %d)", result[:50], len(result), data[:50], len(data))
	}
}

func TestAssemble_UnknownID(t *testing.T) {
	// Чанк с неизвестным ID — должен создать новую сессию, не паниковать.
	c := Chunk{
		Type:  Start,
		ID:    "nonexistent-session-id-" + t.Name(),
		Seq:   0,
		Total: 1,
	}
	t.Cleanup(func() { ClearSession(c.ID) })

	_, done := Assemble(c)
	if done {
		t.Error("Assemble() should not complete on START")
	}
}

func TestAssemble_DataSeqOutOfBounds(t *testing.T) {
	c := Chunk{
		Type:  Data,
		ID:    "oob-" + t.Name(),
		Seq:   999,
		Total: 2,
		Data:  "AAAA",
	}
	t.Cleanup(func() { ClearSession(c.ID) })

	// Сначала создаём сессию
	Assemble(Chunk{Type: Start, ID: c.ID, Total: 2})

	// Затем DATA с Seq вне диапазона — не должно паниковать
	_, done := Assemble(c)
	if done {
		t.Error("out-of-bounds DATA should not complete")
	}
}

// --- Sessions ---

func TestGetSession_NotFound(t *testing.T) {
	if s := GetSession("does-not-exist-xyz"); s != nil {
		t.Errorf("GetSession() = %v, want nil", s)
	}
}

func TestClearSession(t *testing.T) {
	chunks, id := Split("data", 10)
	if len(chunks) == 0 {
		t.Fatal("Split() returned no chunks")
	}

	// Сначала создаём сессию через START-чанк
	Assemble(chunks[0])

	if GetSession(id) == nil {
		t.Fatal("session should exist after Assemble(START)")
	}

	ClearSession(id)
	if GetSession(id) != nil {
		t.Error("session should be gone after ClearSession")
	}
}

func TestSaveSession(t *testing.T) {
	id := "save-session-test"
	t.Cleanup(func() { ClearSession(id) })

	SaveSession(id, nil)
	s := GetSession(id)
	if s == nil {
		t.Fatal("SaveSession() did not create session")
	}
	if s.ID != id {
		t.Errorf("session.ID = %q, want %q", s.ID, id)
	}
}

package storage

import (
	"strings"
	"testing"
	"time"

	"c2-project/internal/models"
)

// --- New ---

func TestNew_Empty(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.tasks == nil {
		t.Error("tasks map is nil")
	}
	if s.clients == nil {
		t.Error("clients map is nil")
	}
	if len(s.tasks) != 0 {
		t.Errorf("tasks map not empty: %d entries", len(s.tasks))
	}
	if len(s.clients) != 0 {
		t.Errorf("clients map not empty: %d entries", len(s.clients))
	}
}

// --- Clients ---

func TestRegisterClient(t *testing.T) {
	s := New()
	s.RegisterClient("client-001", "192.168.1.1")

	client, ok := s.GetClient("client-001")
	if !ok {
		t.Fatal("GetClient() returned ok=false after RegisterClient")
	}
	if client.ID != "client-001" {
		t.Errorf("client.ID = %q, want %q", client.ID, "client-001")
	}
	if client.IP != "192.168.1.1" {
		t.Errorf("client.IP = %q, want %q", client.IP, "192.168.1.1")
	}
	if !client.IsActive {
		t.Error("client.IsActive = false, want true")
	}
	if client.LastSeen.IsZero() {
		t.Error("client.LastSeen is zero")
	}
	if client.CreatedAt.IsZero() {
		t.Error("client.CreatedAt is zero")
	}
}

func TestRegisterClient_Overwrite(t *testing.T) {
	s := New()
	s.RegisterClient("client-001", "192.168.1.1")
	time.Sleep(2 * time.Millisecond)
	s.RegisterClient("client-001", "10.0.0.1") // перезапись

	client, ok := s.GetClient("client-001")
	if !ok {
		t.Fatal("client not found after re-register")
	}
	if client.IP != "10.0.0.1" {
		t.Errorf("client.IP = %q, want %q (should be overwritten)", client.IP, "10.0.0.1")
	}
}

func TestGetClient_NotFound(t *testing.T) {
	s := New()
	_, ok := s.GetClient("does-not-exist")
	if ok {
		t.Error("GetClient() returned ok=true for non-existent client")
	}
}

func TestUpdateClientLastSeen(t *testing.T) {
	s := New()
	s.RegisterClient("client-001", "192.168.1.1")

	before, _ := s.GetClient("client-001")
	time.Sleep(2 * time.Millisecond)
	s.UpdateClientLastSeen("client-001")
	after, _ := s.GetClient("client-001")

	if !after.LastSeen.After(before.LastSeen) {
		t.Errorf("LastSeen not updated: before=%v, after=%v", before.LastSeen, after.LastSeen)
	}
}

func TestUpdateClientLastSeen_NotFound(t *testing.T) {
	s := New()
	// Не должно паниковать.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("UpdateClientLastSeen() panicked: %v", r)
		}
	}()
	s.UpdateClientLastSeen("does-not-exist")
}

// --- Tasks ---

func TestCreateTask(t *testing.T) {
	s := New()
	task := models.Task{
		ClientID: "client-001",
		Command:  "whoami",
	}

	id := s.CreateTask(task)
	if id == "" {
		t.Fatal("CreateTask() returned empty ID")
	}

	got, ok := s.GetTask(id)
	if !ok {
		t.Fatal("GetTask() returned ok=false after CreateTask")
	}
	if got.ID != id {
		t.Errorf("task.ID = %q, want %q", got.ID, id)
	}
	if got.ClientID != "client-001" {
		t.Errorf("task.ClientID = %q, want %q", got.ClientID, "client-001")
	}
	if got.Command != "whoami" {
		t.Errorf("task.Command = %q, want %q", got.Command, "whoami")
	}
	if got.Status != "pending" {
		t.Errorf("task.Status = %q, want %q", got.Status, "pending")
	}
	if got.CreatedAt.IsZero() {
		t.Error("task.CreatedAt is zero")
	}
}

func TestCreateTask_UniqueIDs(t *testing.T) {
	s := New()

	// Создаём 100 задач, все ID должны быть разными.
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		task := models.Task{ClientID: "client-001", Command: "cmd"}
		id := s.CreateTask(task)
		if ids[id] {
			t.Errorf("duplicate ID: %q", id)
		}
		ids[id] = true
		// time.Now().UnixNano() зависит от разрешения таймера — на Windows
		// может быть ~15 мс. Спим чуть-чуть, чтобы ID отличались.
		time.Sleep(time.Microsecond)
	}

	if len(ids) != 100 {
		t.Errorf("got %d unique IDs, want 100", len(ids))
	}
}

func TestGetTask_NotFound(t *testing.T) {
	s := New()
	_, ok := s.GetTask("does-not-exist")
	if ok {
		t.Error("GetTask() returned ok=true for non-existent task")
	}
}

func TestGetPendingTask(t *testing.T) {
	s := New()
	task := models.Task{ClientID: "client-001", Command: "whoami"}
	id := s.CreateTask(task)

	got, ok := s.GetPendingTask("client-001")
	if !ok {
		t.Fatal("GetPendingTask() returned ok=false")
	}
	if got.ID != id {
		t.Errorf("got.ID = %q, want %q", got.ID, id)
	}
	if got.Status != "in_progress" {
		t.Errorf("got.Status = %q, want %q", got.Status, "in_progress")
	}

	// Статус в хранилище тоже должен измениться.
	stored, _ := s.GetTask(id)
	if stored.Status != "in_progress" {
		t.Errorf("stored task Status = %q, want in_progress", stored.Status)
	}
}

func TestGetPendingTask_SecondCallReturnsNothing(t *testing.T) {
	s := New()
	s.CreateTask(models.Task{ClientID: "client-001", Command: "whoami"})

	// Первый вызов забирает задачу.
	_, ok := s.GetPendingTask("client-001")
	if !ok {
		t.Fatal("first GetPendingTask() returned ok=false")
	}

	// Второй вызов не должен вернуть ту же задачу.
	_, ok = s.GetPendingTask("client-001")
	if ok {
		t.Error("second GetPendingTask() returned ok=true, want false")
	}
}

func TestGetPendingTask_FiltersByClient(t *testing.T) {
	s := New()
	id1 := s.CreateTask(models.Task{ClientID: "client-001", Command: "cmd1"})
	id2 := s.CreateTask(models.Task{ClientID: "client-002", Command: "cmd2"})

	// Запрос от client-001 должен вернуть именно его задачу.
	got, ok := s.GetPendingTask("client-001")
	if !ok {
		t.Fatal("GetPendingTask(client-001) returned ok=false")
	}
	if got.ID != id1 {
		t.Errorf("got task %q, want %q", got.ID, id1)
	}

	// Запрос от client-002 — свою.
	got2, ok := s.GetPendingTask("client-002")
	if !ok {
		t.Fatal("GetPendingTask(client-002) returned ok=false")
	}
	if got2.ID != id2 {
		t.Errorf("got task %q, want %q", got2.ID, id2)
	}
}

func TestGetPendingTask_NoTasks(t *testing.T) {
	s := New()
	_, ok := s.GetPendingTask("client-001")
	if ok {
		t.Error("GetPendingTask() returned ok=true with no tasks")
	}
}

func TestGetPendingTask_AlreadyInProgress(t *testing.T) {
	s := New()
	id := s.CreateTask(models.Task{ClientID: "client-001", Command: "whoami"})
	s.GetPendingTask("client-001") // теперь in_progress

	// Ещё раз — не должно вернуть.
	_, ok := s.GetPendingTask("client-001")
	if ok {
		t.Error("GetPendingTask() returned in_progress task")
	}

	// Но задача в хранилище есть.
	if _, ok := s.GetTask(id); !ok {
		t.Error("task disappeared from storage")
	}
}

func TestUpdateTaskResult(t *testing.T) {
	s := New()
	id := s.CreateTask(models.Task{ClientID: "client-001", Command: "whoami"})

	s.UpdateTaskResult(id, "output text", "completed")

	got, _ := s.GetTask(id)
	if got.Result != "output text" {
		t.Errorf("task.Result = %q, want %q", got.Result, "output text")
	}
	if got.Status != "completed" {
		t.Errorf("task.Status = %q, want %q", got.Status, "completed")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("task.UpdatedAt is zero")
	}
}

func TestUpdateTaskResult_NotFound(t *testing.T) {
	s := New()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("UpdateTaskResult() panicked: %v", r)
		}
	}()
	s.UpdateTaskResult("does-not-exist", "output", "completed")
}

// --- generateID ---

func TestGenerateID_Format(t *testing.T) {
	id := generateID()
	if id == "" {
		t.Fatal("generateID() returned empty string")
	}
	// Формат: timestamp-counter-random, разделённые дефисом.
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		t.Errorf("generateID() = %q, want 3 parts separated by dash", id)
	}
	if len(parts[0]) != 14 {
		t.Errorf("timestamp part length = %d, want 14 (%q)", len(parts[0]), parts[0])
	}
}

func TestGenerateID_ChangesOverTime(t *testing.T) {
	id1 := generateID()
	time.Sleep(2 * time.Millisecond)
	id2 := generateID()

	if id1 == id2 {
		t.Errorf("generateID() returned same ID twice: %q", id1)
	}
}

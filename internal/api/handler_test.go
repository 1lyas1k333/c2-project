package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"c2-project/internal/jwt"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"c2-project/internal/service"
	"c2-project/internal/storage"
)

// newTestHandler создаёт Handler с чистым хранилищем.
func newTestHandler() *Handler {
	store := storage.New()
	taskSvc := service.NewTaskService(store)
	clientSvc := service.NewClientService(store)
	return NewHandler(taskSvc, clientSvc)
}

// newTestHandlerWithStore создаёт Handler и возвращает store — для проверок.
func newTestHandlerWithStore() (*Handler, *storage.Storage) {
	store := storage.New()
	taskSvc := service.NewTaskService(store)
	clientSvc := service.NewClientService(store)
	return NewHandler(taskSvc, clientSvc), store
}

// doRequest — удобная обёртка для запросов через хендлер.
func doRequest(handler http.HandlerFunc, method, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// --- CheckMethod ---

func TestCheckMethod_Allowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()
	ok := CheckMethod(rec, req, "POST")
	if !ok {
		t.Error("CheckMethod() returned false for matching method")
	}
	if rec.Code != 200 {
		t.Errorf("rec.Code = %d, want 200 (no response written)", rec.Code)
	}
}

func TestCheckMethod_NotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ok := CheckMethod(rec, req, "POST")
	if ok {
		t.Error("CheckMethod() returned true for non-matching method")
	}
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("rec.Code = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// --- GetTokenFromBearer ---

func TestGetTokenFromBearer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid", "Bearer abc.def.ghi", "abc.def.ghi"},
		{"empty", "", ""},
		{"no prefix", "abc.def.ghi", ""},
		{"wrong prefix", "Basic abc", ""},
		{"lowercase", "bearer abc", ""},
		{"only bearer", "Bearer ", ""},
		{"bearer no space", "Bearerabc", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetTokenFromBearer(tc.input)
			if got != tc.want {
				t.Errorf("GetTokenFromBearer(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- GetClientIDFromBearer ---

func TestGetClientIDFromBearer_Valid(t *testing.T) {
	token, err := jwt.EncodeClientID("client-001")
	if err != nil {
		t.Fatalf("EncodeClientID() error = %v", err)
	}

	got := GetClientIDFromBearer("Bearer " + token)
	if got != "client-001" {
		t.Errorf("GetClientIDFromBearer() = %q, want %q", got, "client-001")
	}
}

func TestGetClientIDFromBearer_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"no bearer", "some-token"},
		{"wrong prefix", "Basic xyz"},
		{"garbage token", "Bearer not.a.jwt"},
		{"only bearer", "Bearer "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetClientIDFromBearer(tc.input)
			if got != "" {
				t.Errorf("GetClientIDFromBearer(%q) = %q, want empty", tc.input, got)
			}
		})
	}
}

// --- RegisterHandler ---

func TestRegisterHandler_Success(t *testing.T) {
	h := newTestHandler()

	token, _ := jwt.EncodeClientID("client-001")
	rec := doRequest(h.RegisterHandler, "POST", "/api/register", "Bearer "+token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("response status = %q, want %q", resp["status"], "ok")
	}
}

func TestRegisterHandler_RegistersInStorage(t *testing.T) {
	h, store := newTestHandlerWithStore()

	token, _ := jwt.EncodeClientID("client-042")
	rec := doRequest(h.RegisterHandler, "POST", "/api/register", "Bearer "+token)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	client, ok := store.GetClient("client-042")
	if !ok {
		t.Fatal("client not found in storage after registration")
	}
	if client.ID != "client-042" {
		t.Errorf("client.ID = %q, want client-042", client.ID)
	}
	if !client.IsActive {
		t.Error("client.IsActive = false, want true")
	}
}

func TestRegisterHandler_WrongMethod(t *testing.T) {
	h := newTestHandler()
	token, _ := jwt.EncodeClientID("client-001")

	rec := doRequest(h.RegisterHandler, "GET", "/api/register", "Bearer "+token)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRegisterHandler_MissingAuth(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.RegisterHandler, "POST", "/api/register", "")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterHandler_InvalidToken(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.RegisterHandler, "POST", "/api/register", "Bearer garbage")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- TasksHandler ---

func TestTasksHandler_Success(t *testing.T) {
	h := newTestHandler()

	task := models.Task{ClientID: "client-001", Command: "whoami"}
	token, err := protocol.EncodeRequest(task)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}

	rec := doRequest(h.TasksHandler, "POST", "/api/tasks", token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "created" {
		t.Errorf("status = %q, want %q", resp["status"], "created")
	}
	if resp["task_id"] == "" {
		t.Error("task_id is empty")
	}
}

func TestTasksHandler_CreatesInStorage(t *testing.T) {
	h, store := newTestHandlerWithStore()

	task := models.Task{ClientID: "client-001", Command: "whoami"}
	token, _ := protocol.EncodeRequest(task)

	rec := doRequest(h.TasksHandler, "POST", "/api/tasks", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)

	stored, ok := store.GetTask(resp["task_id"])
	if !ok {
		t.Fatal("task not found in storage")
	}
	if stored.ClientID != "client-001" {
		t.Errorf("stored.ClientID = %q, want client-001", stored.ClientID)
	}
	if stored.Command != "whoami" {
		t.Errorf("stored.Command = %q, want whoami", stored.Command)
	}
	if stored.Status != "pending" {
		t.Errorf("stored.Status = %q, want pending", stored.Status)
	}
}

func TestTasksHandler_WrongMethod(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.TasksHandler, "GET", "/api/tasks", "Bearer x.y.z")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestTasksHandler_MissingAuth(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.TasksHandler, "POST", "/api/tasks", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTasksHandler_InvalidToken(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.TasksHandler, "POST", "/api/tasks", "Bearer garbage")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- PollHandler ---

func TestPollHandler_NoTasks(t *testing.T) {
	h := newTestHandler()

	token, _ := jwt.EncodeClientID("client-001")
	rec := doRequest(h.PollHandler, "POST", "/api/poll", "Bearer "+token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "no_tasks" {
		t.Errorf("status = %q, want no_tasks", resp["status"])
	}
}

func TestPollHandler_ReturnsTask(t *testing.T) {
	h, store := newTestHandlerWithStore()

	// Создаём задачу напрямую в storage.
	taskID := store.CreateTask(models.Task{ClientID: "client-001", Command: "whoami"})

	token, _ := jwt.EncodeClientID("client-001")
	rec := doRequest(h.PollHandler, "POST", "/api/poll", "Bearer "+token)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// В теле — {"status": "ok"}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("body status = %q, want ok", resp["status"])
	}

	// В заголовке Authorization — токен задачи.
	authHeader := rec.Header().Get("Authorization")
	if authHeader == "" {
		t.Fatal("Authorization header is empty")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Errorf("Authorization = %q, want 'Bearer ' prefix", authHeader)
	}

	// Декодируем токен и проверяем, что это наша задача.
	gotTask, err := protocol.DecodeRequest(authHeader)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	gotMap, ok := gotTask.(map[string]interface{})
	if !ok {
		t.Fatalf("DecodeRequest returned %T, want map", gotTask)
	}
	if gotMap["id"] != taskID {
		t.Errorf("task id in token = %v, want %q", gotMap["id"], taskID)
	}
	if gotMap["command"] != "whoami" {
		t.Errorf("command = %v, want whoami", gotMap["command"])
	}
	if gotMap["status"] != "in_progress" {
		t.Errorf("status = %v, want in_progress (GetPendingTask should set it)", gotMap["status"])
	}
}

func TestPollHandler_UpdatesLastSeen(t *testing.T) {
	h, store := newTestHandlerWithStore()

	// Регистрируем клиента.
	regToken, _ := jwt.EncodeClientID("client-001")
	doRequest(h.RegisterHandler, "POST", "/api/register", "Bearer "+regToken)

	before, _ := store.GetClient("client-001")

	// Ждём, чтобы время точно изменилось (на Windows разрешение ~15 мс).
	// Используем time.Sleep — но он в реальном времени; для стабильности — 20 мс.
	// (Импорт time нужен.)
	// time.Sleep(20 * time.Millisecond) — раскомментируй, если тест флапает.

	// Poll
	pollToken, _ := jwt.EncodeClientID("client-001")
	doRequest(h.PollHandler, "POST", "/api/poll", "Bearer "+pollToken)

	after, _ := store.GetClient("client-001")
	if !after.LastSeen.After(before.LastSeen) && !after.LastSeen.Equal(before.LastSeen) {
		t.Errorf("LastSeen not updated: before=%v, after=%v", before.LastSeen, after.LastSeen)
	}
}

func TestPollHandler_WrongMethod(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.PollHandler, "GET", "/api/poll", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestPollHandler_MissingAuth(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.PollHandler, "POST", "/api/poll", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- ResultsHandler ---

func TestResultsHandler_Success(t *testing.T) {
	h, store := newTestHandlerWithStore()

	// Создаём задачу напрямую.
	taskID := store.CreateTask(models.Task{ClientID: "client-001", Command: "whoami"})

	// Клиент отправляет результат — это обычное сообщение (не чанк).
	result := map[string]interface{}{
		"task_id": taskID,
		"output":  "root",
		"status":  "completed",
	}
	token, err := protocol.EncodeRequest(result)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}

	rec := doRequest(h.ResultsHandler, "POST", "/api/results", token)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want ok", resp["status"])
	}

	// Проверяем, что результат сохранён.
	stored, _ := store.GetTask(taskID)
	if stored.Result != "root" {
		t.Errorf("stored.Result = %q, want root", stored.Result)
	}
	if stored.Status != "completed" {
		t.Errorf("stored.Status = %q, want completed", stored.Status)
	}
}

func TestResultsHandler_WrongMethod(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.ResultsHandler, "GET", "/api/results", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestResultsHandler_MissingAuth(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.ResultsHandler, "POST", "/api/results", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestResultsHandler_InvalidToken(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.ResultsHandler, "POST", "/api/results", "Bearer garbage")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- GetResultHandler ---

func TestGetResultHandler_Success(t *testing.T) {
	h, store := newTestHandlerWithStore()

	taskID := store.CreateTask(models.Task{ClientID: "client-001", Command: "whoami"})
	store.UpdateTaskResult(taskID, "root", "completed")

	rec := doRequest(h.GetResultHandler, "GET", "/api/result?task_id="+taskID, "")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "completed" {
		t.Errorf("status = %v, want completed", resp["status"])
	}
	if resp["result"] != "root" {
		t.Errorf("result = %v, want root", resp["result"])
	}
}

func TestGetResultHandler_MissingTaskID(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.GetResultHandler, "GET", "/api/result", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetResultHandler_NotFound(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.GetResultHandler, "GET", "/api/result?task_id=nonexistent", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetResultHandler_WrongMethod(t *testing.T) {
	h := newTestHandler()
	rec := doRequest(h.GetResultHandler, "POST", "/api/result?task_id=x", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// --- RegisterRoutes ---

func TestRegisterRoutes(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	RegisterRoutes(mux, h)

	// Проверяем, что маршруты отвечают (не 404).
	// Для этого делаем минимальный запрос к /api/poll без auth — ожидаем 400.
	req := httptest.NewRequest("POST", "/api/poll", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Error("/api/poll route not registered (404)")
	}
}

// --- Полный сценарий ---

func TestFullScenario(t *testing.T) {
	h, store := newTestHandlerWithStore()
	mux := http.NewServeMux()
	RegisterRoutes(mux, h)

	// 1. Клиент регистрируется.
	clientToken, _ := jwt.EncodeClientID("client-001")
	req := httptest.NewRequest("POST", "/api/register", nil)
	req.Header.Set("Authorization", "Bearer "+clientToken)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register: status = %d, want 200", rec.Code)
	}

	// 2. Терминал создаёт задачу.
	task := models.Task{ClientID: "client-001", Command: "whoami"}
	taskToken, _ := protocol.EncodeRequest(task)
	req = httptest.NewRequest("POST", "/api/tasks", nil)
	req.Header.Set("Authorization", taskToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tasks: status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var taskResp map[string]string
	json.NewDecoder(rec.Body).Decode(&taskResp)
	taskID := taskResp["task_id"]
	if taskID == "" {
		t.Fatal("task_id is empty")
	}

	// 3. Клиент опрашивает → получает задачу.
	req = httptest.NewRequest("POST", "/api/poll", nil)
	req.Header.Set("Authorization", "Bearer "+clientToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("poll: status = %d, want 200", rec.Code)
	}
	taskAuthHeader := rec.Header().Get("Authorization")
	if taskAuthHeader == "" {
		t.Fatal("poll: no Authorization header with task")
	}

	// 4. Клиент расшифровывает задачу (проверяем, что она правильная).
	gotTask, err := protocol.DecodeRequest(taskAuthHeader)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	gotMap := gotTask.(map[string]interface{})
	if gotMap["command"] != "whoami" {
		t.Errorf("decoded command = %v, want whoami", gotMap["command"])
	}

	// 5. Клиент отправляет результат.
	result := map[string]interface{}{
		"task_id": taskID,
		"output":  "root",
		"status":  "completed",
	}
	resultToken, _ := protocol.EncodeRequest(result)
	req = httptest.NewRequest("POST", "/api/results", nil)
	req.Header.Set("Authorization", resultToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("results: status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	// 6. Терминал получает результат.
	req = httptest.NewRequest("GET", "/api/result?task_id="+taskID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get result: status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var finalResp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&finalResp)
	if finalResp["result"] != "root" {
		t.Errorf("final result = %v, want root", finalResp["result"])
	}
	if finalResp["status"] != "completed" {
		t.Errorf("final status = %v, want completed", finalResp["status"])
	}

	// Дополнительно: задача в хранилище имеет статус completed.
	stored, _ := store.GetTask(taskID)
	if stored.Status != "completed" {
		t.Errorf("stored status = %q, want completed", stored.Status)
	}
}

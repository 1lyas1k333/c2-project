// Package main - терминал оператора с TUI интерфейсом.
// Поддерживает выбор клиента, историю команд и красивый вывод.
package main

import (
	"bytes"
	"c2-project/internal/compress"
	"c2-project/internal/models"
	"c2-project/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
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
var configFile = "configs/terminal.json"

// resultMsg - сообщение с результатом выполнения команды.
type resultMsg struct {
	result string
	status string
	err    error
}

// model - состояние приложения.
type model struct {
	input    textinput.Model
	history  []string // Последние 5 команд
	result   string   // Результат последней команды
	status   string
	clientID string
	taskID   string
	waiting  bool
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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B93A7")).
			Italic(true)

	historyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B93A7"))

	resultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	clientStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	historyBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#3A3A3A")).
			Padding(0, 1)

	resultBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Padding(0, 1)

	inputBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
)

func main() {
	// ОТКЛЮЧАЕМ ЛОГИ — они ломают TUI в Windows CMD
	log.SetOutput(io.Discard)
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-local" {
			configFile = "configs/terminal.local.json"
			break
		}
	}

	if err := loadConfig(); err != nil {
		fmt.Printf("[WARN] Failed to load config: %v\n", err)
		fmt.Println("[INFO] Using default server URL: http://localhost:8080")
		config.ServerURL = "http://localhost:8080"
	}

	loadHistory()
	m := initialModel()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

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

			// Сохраняем команду в историю (только последние 5)
			m.history = append(m.history, fmt.Sprintf("> %s [%s]", cmd, m.clientID))
			if len(m.history) > 5 {
				m.history = m.history[1:]
			}

			m.status = fmt.Sprintf("Sending command to %s...", m.clientID)
			m.waiting = true
			m.result = ""

			addHistory(m.clientID, cmd)
			m.input.SetValue("")

			return m, m.sendCommandWithResult(cmd)

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// sendCommandWithResult - отправляет команду и возвращает tea.Cmd с результатом.
func (m *model) sendCommandWithResult(cmd string) tea.Cmd {
	return func() tea.Msg {
		task := models.Task{
			ClientID: m.clientID,
			Command:  cmd,
			Status:   "pending",
		}

		token, err := protocol.EncodeRequest(task)
		if err != nil {
			return resultMsg{err: fmt.Errorf("encryption error: %v", err)}
		}

		jsonData, _ := json.Marshal(map[string]interface{}{})

		client := &http.Client{}
		reqHTTP, err := http.NewRequest("POST", config.ServerURL+"/api/tasks", bytes.NewReader(jsonData))
		if err != nil {
			return resultMsg{err: fmt.Errorf("request error: %v", err)}
		}
		reqHTTP.Header.Set("Authorization", "Bearer "+token)
		reqHTTP.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(reqHTTP)
		if err != nil {
			return resultMsg{err: fmt.Errorf("send error: %v", err)}
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		taskID, _ := result["task_id"].(string)

		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)

			resp2, err := http.Get(fmt.Sprintf("%s/api/result?task_id=%s", config.ServerURL, taskID))
			if err != nil {
				continue
			}

			var result2 map[string]interface{}
			json.NewDecoder(resp2.Body).Decode(&result2)
			resp2.Body.Close()

			status, _ := result2["status"].(string)

			if status == "completed" {
				resultText, _ := result2["result"].(string)
				resultText = strings.TrimSpace(resultText)
				resultText = strings.ReplaceAll(resultText, "\r\n", "\n")

				isCompressed := len(resultText) > 0 && (resultText[0] == '\x1f' || resultText[0] == 0x1f)
				if isCompressed {
					decompressed, err := compress.Decompress([]byte(resultText))
					if err == nil {
						return resultMsg{result: string(decompressed), status: fmt.Sprintf("Command executed on %s", m.clientID)}
					}
				}

				return resultMsg{result: resultText, status: fmt.Sprintf("Command executed on %s", m.clientID)}
			} else if status == "failed" {
				return resultMsg{status: "Command execution failed"}
			}
		}
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
	clientInfo := clientStyle.Render("Client: " + m.clientID)
	statusInfo := statusStyle.Render(m.status)
	content.WriteString(title + "  " + clientInfo + "  " + statusInfo + "\n")

	// Разделитель
	content.WriteString(strings.Repeat("─", m.width-6) + "\n")

	// История (последние 5 команд)
	content.WriteString(historyStyle.Render("История команд:") + "\n")
	if len(m.history) > 0 {
		for _, h := range m.history {
			content.WriteString("  " + historyStyle.Render(h) + "\n")
		}
	} else {
		content.WriteString("  " + helpStyle.Render("(пока пусто)") + "\n")
	}

	content.WriteString("\n")

	// Результат (большая область) — адаптивно под высоту окна
	content.WriteString(statusStyle.Render("Результат:") + "\n")
	if m.result != "" {
		// Оставляем место для заголовка (2), истории (7), ввода (3), помощи (2), подвала (1)
		maxResultLines := m.height - 20
		if maxResultLines < 5 {
			maxResultLines = 5
		}
		if maxResultLines > 40 {
			maxResultLines = 40
		}

		displayResult := m.result
		lines := strings.Split(displayResult, "\n")
		if len(lines) > maxResultLines {
			lines = lines[:maxResultLines]
			displayResult = strings.Join(lines, "\n") + "\n... (ещё строки обрезаны)"
		}
		displayResult = truncateLines(displayResult, m.width-12)
		resultBox := resultBoxStyle.Width(m.width - 8).Render(displayResult)
		content.WriteString(resultBox + "\n")
	} else {
		content.WriteString(helpStyle.Render("(пока пусто)") + "\n")
	}

	content.WriteString("\n")

	// Поле ввода
	inputBox := inputBoxStyle.Width(m.width - 8).Render(m.input.View())
	content.WriteString(inputBox + "\n")

	// Помощь
	if m.showHelp {
		help := helpStyle.Render(`/help - помощь | /clients - список клиентов | /select <id> - выбрать | /clear - очистить | q - выход`)
		content.WriteString(help + "\n")
	} else {
		content.WriteString(helpStyle.Render("Enter /help для справки | q - выход") + "\n")
	}

	// Подвал
	footer := fmt.Sprintf("История: %d | Результат: %d строк", len(historyData), len(strings.Split(m.result, "\n")))
	content.WriteString(helpStyle.Render(footer))

	return content.String()
}

// truncateLines - обрезает строки по максимальной ширине.
// Предотвращает "съезд" рамок при длинных строках.
func truncateLines(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > maxWidth {
			lines[i] = line[:maxWidth-3] + "..."
		}
	}
	return strings.Join(lines, "\n")
}

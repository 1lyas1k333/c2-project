# C2 Project - Удаленный доступ к устройству

## Описание
Комплект ПО для удаленного доступа к устройству, состоящий из трех компонентов:
- Терминал оператора (TUI) — интерфейс для отправки команд
- C2-сервер — промежуточный компонент для агрегации задач
- Клиент — бэкдор, выполняющий команды

## Требования
- Go 1.21 или новее (для локальной сборки)
- Операционная система: Windows, Linux или macOS
- Для выполнения Linux-команд на Windows: WSL (Windows Subsystem for Linux)
- Docker (опционально, для запуска через контейнеры)
- Windows Terminal (рекомендуется для корректного отображения TUI)

## Установка Go
Скачайте и установите Go с официального сайта: https://go.dev/dl/

## Структура проекта
```
c2-project/
├── bin/                 # Собранные исполняемые файлы
├── cmd/                 # Исходный код компонентов
│   ├── c2-server/       # C2-сервер
│   ├── client/          # Клиент (бэкдор)
│   └── terminal/        # Терминал оператора (TUI)
├── configs/             # Конфигурационные файлы
│   ├── server.json      # Настройки сервера (Docker)
│   ├── client.json      # Настройки клиента (Docker)
│   ├── terminal.json    # Настройки терминала (Docker)
│   ├── server.local.json    # Настройки сервера (локально)
│   ├── client.local.json    # Настройки клиента (локально)
│   └── terminal.local.json  # Настройки терминала (локально)
├── internal/            # Внутренние пакеты
│   ├── api/             # HTTP-хендлеры, роутеры, валидация
│   ├── compress/        # Сжатие gzip
│   ├── crypto/          # Шифрование AES-GCM
│   ├── chunk/           # Фрагментация данных
│   ├── jwt/             # JWT-токены
│   ├── models/          # Структуры данных
│   ├── protocol/        # Протокол передачи
│   └── storage/         # Хранилище задач и клиентов
├── Dockerfile
├── docker-compose.yml
├── .dockerignore
├── go.mod
├── go.sum
└── README.md
```

## Конфигурация

### Конфигурация для Docker

**configs/server.json:**
```json
{
    "listen_address": ":8080"
}
```

**configs/client.json:**
```json
{
    "server_url": "http://server:8080",
    "client_id": "client-001",
    "poll_interval": 5
}
```

**configs/terminal.json:**
```json
{
    "server_url": "http://server:8080"
}
```

### Конфигурация для локального запуска

**configs/server.local.json:**
```json
{
    "listen_address": ":8080"
}
```

**configs/client.local.json:**
```json
{
    "server_url": "http://localhost:8080",
    "client_id": "client-001",
    "poll_interval": 5
}
```

**configs/terminal.local.json:**
```json
{
    "server_url": "http://localhost:8080"
}
```

## Сборка проекта (локально)

### 1. Откройте терминал в папке проекта
```bash
cd c2-project
```

### 2. Установите зависимости
```bash
go mod tidy
```

### 3. Соберите все компоненты
```bash
go build -o bin/c2-server.exe ./cmd/c2-server
go build -o bin/client.exe ./cmd/client
go build -o bin/terminal.exe ./cmd/terminal
```

### 4. Сборка для Linux (если нужно)
```bash
GOOS=linux GOARCH=amd64 go build -o bin/c2-server ./cmd/c2-server
GOOS=linux GOARCH=amd64 go build -o bin/client ./cmd/client
GOOS=linux GOARCH=amd64 go build -o bin/terminal ./cmd/terminal
```

## Запуск через Docker (рекомендуемый способ)

### 1. Установите Docker Desktop
https://www.docker.com/products/docker-desktop/

### 2. Соберите образы
```bash
docker-compose build
```

### 3. Запустите все компоненты
```bash
docker-compose up
```

### 4. Подключитесь к терминалу
```bash
docker attach c2-terminal
```

### 5. Остановка
```bash
docker-compose down
```

## Запуск (локально, без Docker)

Важно! Запускайте все компоненты в отдельных окнах терминала.
Для корректного отображения TUI в Windows рекомендуется использовать Windows Terminal.

Перед запуском перейдите в папку проекта:
```bash
cd c2-project
```

### 1. Запуск C2-сервера
```bash
bin\c2-server.exe -local
```
Ожидаемый вывод:
```
C2 Server starting on :8080
```

### 2. Запуск клиента (на атакуемом устройстве)
```bash
bin\client.exe -local -id client-001
```
Ожидаемый вывод:
```
[START] Client client-001 starting...
[OK] Registration successful
```

Для запуска нескольких клиентов используйте аргумент `-id`:
```bash
bin\client.exe -local -id client-001
bin\client.exe -local -id client-002
```

### 3. Запуск терминала оператора (TUI)
```bash
bin\terminal.exe -local
```
Ожидаемый вывод:
```
C2 Terminal Operator v2.0
Client: client-001
> Enter command...
```

## Использование

1. Запустите C2-сервер
2. Запустите одного или нескольких клиентов
3. Запустите терминал
4. В терминале введите команду и нажмите Enter

### Команды TUI

| Команда | Описание |
|---|---|
| `ls -la` | Выполнить Linux-команду |
| `/clients` | Показать список клиентов |
| `/select <id>` | Переключиться на клиента |
| `/clear` | Очистить историю |
| `/help` | Показать справку |
| `q` или `Ctrl+C` | Выйти из TUI |

### Примеры команд

**Linux (через Docker):**
- `ls -la` — список файлов
- `ps aux | head -10` — первые 10 процессов
- `echo "DEADBEEF" | base64` — кодирование в Base64
- `whoami` — имя пользователя
- `hostname` — имя компьютера

**Windows (локально):**
- `whoami && hostname && ver` — информация о системе
- `dir C:\ && echo OK || echo FAIL` — список файлов на диске C
- `ipconfig && systeminfo | findstr /i "OS Name"` — IP и версия ОС
- `tasklist | findstr /i "go"` — процессы с "go" в имени

## Пример работы

### Терминал оператора (TUI):
```
C2 Terminal Operator v2.0  Client: client-001  Command executed

История команд:
  > ls -la [client-001]
  > whoami [client-001]

Результат:
┌────────────────────────────────────────────────────┐
│ total 44                                           │
│ drwxr-xr-x 1 root root 4096 Aug 27 12:10 .         │
│ drwxr-xr-x 1 root root 4096 Aug 27 12:12 ..        │
└────────────────────────────────────────────────────┘

> Enter command...
```

### Логи сервера:
```
[OK] Client registered: client-001
[TASK] Task created: 17876693681484808600 -> ls -la
[TASK] Task 17876693681484808600 found for client client-001
[ENCRYPT] ADDITIONAL ENCRYPTION of task 17876693681484808600
[SEND] Encrypted task sent to client client-001
[RESULT] Result received: 17876693681484808600 -> completed
```

### Логи клиента:
```
[START] Client client-001 starting...
[OK] Registration successful
[DECRYPT] Task decrypted: ls -la
[EXEC] Executing: ls -la
[SEND] Encrypted result sent to server
```

## Протокол передачи данных
- Протокол: HTTP
- Шифрование: AES-256-GCM (симметричное)
- Маскировка: JWT (передача в заголовке `Authorization: Bearer <token>`)
- Фрагментация: чанки START/DATA/END (для данных > 1KB)
- Сжатие: gzip (для результатов длиннее 5000 байт)

## Безопасность
- Все данные шифруются AES-GCM
- Передача маскируется под JWT-аутентификацию
- Данные передаются в заголовках HTTP, а не в теле запроса
- Крупные данные фрагментируются на чанки

## Устранение неполадок

### Ошибка: "go: command not found"
Решение: Установите Go: https://go.dev/dl/

### Ошибка: "port 8080 already in use"
Решение: Закройте все окна с запущенным сервером и попробуйте снова.

### Ошибка: "Connection refused"
Решение: Убедитесь, что сервер запущен и конфигурации указывают на правильный адрес.

### TUI отображается некорректно (наслаивается)
Решение: Используйте Windows Terminal вместо обычного CMD. Он лучше поддерживает альтернативный экранный режим.

### Некорректные символы при выполнении Windows-команд
Команды выполняются корректно, статус всегда `completed`. Некорректные символы могут появляться из-за OEM-кодировки (CP866), которую Windows использует для консольного вывода. Это особенность отображения, а не ошибка программы. В Docker с Linux-командами такой проблемы нет.

### Docker: "port already in use"
Решение: Остановите локальный сервер или измените порт в `configs/server.json`.

## Важно
Данный проект имеет сугубо учебный характер. Не используйте в реальных сетях!
Разрабатывайте и тестируйте только в изолированном, виртуализированном пространстве.

## Автор
Хисматуллин Ильяс Рустемович
Группа: БББО-13-24
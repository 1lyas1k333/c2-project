# Базовый образ с Go
FROM golang:1.21-alpine

# Устанавливаем Git
RUN apk add --no-cache git

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем ВСЁ
COPY . .

# Скачиваем зависимости
RUN go env -w GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Собираем компоненты
RUN go build -o bin/c2-server ./cmd/c2-server
RUN go build -o bin/client ./cmd/client
RUN go build -o bin/terminal ./cmd/terminal

# Открываем порт
EXPOSE 8080

CMD ["./bin/c2-server"]
.PHONY: all build build-server build-client build-terminal run clean docker test

# Все компоненты
all: build

# Сборка всех бинарников
build: build-server build-client build-terminal
	@echo "Build complete!"

build-server:
	@echo "Building c2-server..."
	go build -o bin/c2-server.exe ./cmd/c2-server

build-client:
	@echo "Building client..."
	go build -o bin/client.exe ./cmd/client

build-terminal:
	@echo "Building terminal..."
	go build -o bin/terminal.exe ./cmd/terminal

# Очистка
clean:
	@echo "Cleaning..."
	rm -rf bin/

# Docker
docker:
	docker-compose down
	docker-compose build --no-cache
	docker-compose up

docker-down:
	docker-compose down

# Тесты
test:
	go test ./...

# Форматирование
fmt:
	go fmt ./...
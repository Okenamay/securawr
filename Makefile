# Переменные
PROTO_SRC = proto/securawr.proto
GEN_DIR = gen

# Определение версии и даты для ldflags
VERSION := $(shell git describe --tags --always --dirty || echo dev)
DATE := $(shell date -u +%Y-%m-%d)
LDFLAGS := -X github.com/Okenamay/securawr/internal/version.Version=$(VERSION) -X github.com/Okenamay/securawr/internal/version.BuildDate=$(DATE)

.PHONY: all clean generate build run-server

all: generate build

# Создает папки и генерирует Go код из Proto
generate:
	@echo "Generating gRPC code..."
	@mkdir -p $(GEN_DIR)
	protoc --proto_path=. \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_SRC)
	@echo "Done."

# Устанавливает зависимости (если нужно)
deps:
	go mod tidy

# Сборка бинарников с внедрением версии
build: deps
	@mkdir -p bin
	@echo "Building Server..."
	go build -ldflags "$(LDFLAGS)" -o bin/server cmd/server/main.go
	@echo "Building Client..."
	go build -ldflags "$(LDFLAGS)" -o bin/client cmd/client/main.go

# Запуск сервера (для теста)
run-server:
	go run -ldflags "$(LDFLAGS)" cmd/server/main.go --log-level=debug

clean:
	rm -rf bin $(GEN_DIR)
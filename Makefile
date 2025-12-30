# PhotoConverter Makefile
# Упрощает сборку, настройку и использование утилиты

# Переменные
APP_NAME := photoconverter
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w \
	-X 'github.com/artemshloyda/photoconverter/internal/cli.Version=$(VERSION)' \
	-X 'github.com/artemshloyda/photoconverter/internal/cli.BuildTime=$(BUILD_TIME)'

# Директории
BUILD_DIR := build
DIST_DIR := dist

# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: all build install clean test run help deps lint check-vips cross \
        build-linux build-darwin build-windows

# По умолчанию - сборка
all: build

## Сборка

# Сборка для текущей платформы
build:
	@echo "$(GREEN)Сборка $(APP_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(APP_NAME) ./cmd/photoconverter
	@echo "$(GREEN)Готово: ./$(APP_NAME)$(NC)"

# Сборка с отладочной информацией
build-debug:
	@echo "$(YELLOW)Сборка $(APP_NAME) (debug)...$(NC)"
	$(GO) build -o $(APP_NAME) ./cmd/photoconverter
	@echo "$(GREEN)Готово: ./$(APP_NAME)$(NC)"

# Установка в $GOPATH/bin
install:
	@echo "$(GREEN)Установка $(APP_NAME)...$(NC)"
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/photoconverter
	@echo "$(GREEN)Установлено в $(shell go env GOPATH)/bin/$(APP_NAME)$(NC)"

## Зависимости

# Установка зависимостей
deps:
	@echo "$(GREEN)Установка зависимостей...$(NC)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)Зависимости установлены$(NC)"

# Обновление зависимостей
deps-update:
	@echo "$(GREEN)Обновление зависимостей...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "$(GREEN)Зависимости обновлены$(NC)"

## Проверки

# Проверка наличия vips
check-vips:
	@echo "$(GREEN)Проверка vips...$(NC)"
	@which vips > /dev/null 2>&1 || (echo "$(YELLOW)vips не найден! Установите:$(NC)" && \
		echo "  macOS: brew install vips" && \
		echo "  Ubuntu: sudo apt install libvips-tools" && \
		echo "  Fedora: sudo dnf install vips-tools" && exit 1)
	@vips --version
	@echo "$(GREEN)vips доступен$(NC)"

# Линтер
lint:
	@echo "$(GREEN)Запуск линтера...$(NC)"
	@which golangci-lint > /dev/null 2>&1 || (echo "$(YELLOW)Установка golangci-lint...$(NC)" && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Форматирование кода
fmt:
	@echo "$(GREEN)Форматирование кода...$(NC)"
	$(GO) fmt ./...
	@echo "$(GREEN)Готово$(NC)"

# Проверка go vet
vet:
	@echo "$(GREEN)Запуск go vet...$(NC)"
	$(GO) vet ./...
	@echo "$(GREEN)Готово$(NC)"

## Тестирование

# Запуск тестов
test:
	@echo "$(GREEN)Запуск тестов...$(NC)"
	$(GO) test -v ./...

# Тесты с покрытием
test-coverage:
	@echo "$(GREEN)Запуск тестов с покрытием...$(NC)"
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Отчёт: coverage.html$(NC)"

## Запуск

# Запуск с примером (dry-run)
run: build
	@echo "$(GREEN)Запуск в режиме dry-run...$(NC)"
	./$(APP_NAME) --help

# Показать версию
version: build
	./$(APP_NAME) version

## Очистка

# Очистка артефактов сборки
clean:
	@echo "$(GREEN)Очистка...$(NC)"
	rm -f $(APP_NAME)
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Готово$(NC)"

## Кросс-компиляция

# Создание директории для сборок
$(DIST_DIR):
	mkdir -p $(DIST_DIR)

# Сборка для всех платформ
cross: $(DIST_DIR) build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo "$(GREEN)Все платформы собраны в $(DIST_DIR)/$(NC)"
	@ls -la $(DIST_DIR)/

# Linux AMD64
build-linux-amd64: $(DIST_DIR)
	@echo "$(GREEN)Сборка для Linux AMD64...$(NC)"
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./cmd/photoconverter

# Linux ARM64
build-linux-arm64: $(DIST_DIR)
	@echo "$(GREEN)Сборка для Linux ARM64...$(NC)"
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP_NAME)-linux-arm64 ./cmd/photoconverter || \
		echo "$(YELLOW)Требуется кросс-компилятор для ARM64$(NC)"

# macOS AMD64
build-darwin-amd64: $(DIST_DIR)
	@echo "$(GREEN)Сборка для macOS AMD64...$(NC)"
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/photoconverter

# macOS ARM64 (Apple Silicon)
build-darwin-arm64: $(DIST_DIR)
	@echo "$(GREEN)Сборка для macOS ARM64...$(NC)"
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/photoconverter

# Windows AMD64
build-windows-amd64: $(DIST_DIR)
	@echo "$(GREEN)Сборка для Windows AMD64...$(NC)"
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/photoconverter || \
		echo "$(YELLOW)Требуется mingw-w64 для Windows$(NC)"

## Docker

# Сборка Docker образа
docker-build:
	@echo "$(GREEN)Сборка Docker образа...$(NC)"
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest
	@echo "$(GREEN)Образ: $(APP_NAME):$(VERSION)$(NC)"

## Справка

# Показать справку
help:
	@echo ""
	@echo "$(GREEN)PhotoConverter - утилита для массовой конвертации изображений$(NC)"
	@echo ""
	@echo "Использование: make [цель]"
	@echo ""
	@echo "Сборка:"
	@echo "  build          - Сборка для текущей платформы"
	@echo "  build-debug    - Сборка с отладочной информацией"
	@echo "  install        - Установка в \$$GOPATH/bin"
	@echo "  cross          - Кросс-компиляция для всех платформ"
	@echo ""
	@echo "Зависимости:"
	@echo "  deps           - Установка зависимостей"
	@echo "  deps-update    - Обновление зависимостей"
	@echo "  check-vips     - Проверка наличия vips"
	@echo ""
	@echo "Проверки:"
	@echo "  lint           - Запуск линтера"
	@echo "  fmt            - Форматирование кода"
	@echo "  vet            - Запуск go vet"
	@echo ""
	@echo "Тестирование:"
	@echo "  test           - Запуск тестов"
	@echo "  test-coverage  - Тесты с отчётом покрытия"
	@echo ""
	@echo "Запуск:"
	@echo "  run            - Показать справку утилиты"
	@echo "  version        - Показать версию"
	@echo ""
	@echo "Очистка:"
	@echo "  clean          - Удалить артефакты сборки"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   - Собрать Docker образ"
	@echo ""
	@echo "Примеры использования утилиты:"
	@echo "  ./$(APP_NAME) --in ./photos --out ./converted"
	@echo "  ./$(APP_NAME) --in ./photos --out ./converted --out-format webp --quality 90"
	@echo "  ./$(APP_NAME) --in ./photos --out ./converted --mode dedup"
	@echo ""

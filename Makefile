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

# Минимальное покрытие тестами (%)
COVERAGE_MIN := 20

# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: all build install install-go uninstall clean test test-coverage coverage coverage-check \
        coverage-badge run help deps lint check-vips cross build-linux build-darwin build-windows \
        docker-build docker-run docker-push

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

# Установка в /usr/local/bin (требует sudo)
install: build
	@echo "$(GREEN)Установка $(APP_NAME) в /usr/local/bin...$(NC)"
	@sudo cp $(APP_NAME) /usr/local/bin/$(APP_NAME)
	@sudo chmod +x /usr/local/bin/$(APP_NAME)
	@echo "$(GREEN)Установлено: /usr/local/bin/$(APP_NAME)$(NC)"
	@echo "$(GREEN)Теперь можно использовать: photoconverter$(NC)"

# Установка в $GOPATH/bin (без sudo)
install-go:
	@echo "$(GREEN)Установка $(APP_NAME) в \$$GOPATH/bin...$(NC)"
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/photoconverter
	@echo "$(GREEN)Установлено в $(shell go env GOPATH)/bin/$(APP_NAME)$(NC)"
	@echo "$(YELLOW)Убедитесь, что $(shell go env GOPATH)/bin в вашем PATH$(NC)"

# Удаление из /usr/local/bin
uninstall:
	@echo "$(GREEN)Удаление $(APP_NAME)...$(NC)"
	@sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "$(GREEN)Удалено$(NC)"

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

# Показать процент покрытия
coverage:
	@echo "$(GREEN)Покрытие тестами:$(NC)"
	@$(GO) test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@$(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}'

# Проверка минимального покрытия (по умолчанию 50%)
coverage-check:
	@echo "$(GREEN)Проверка покрытия (минимум $(COVERAGE_MIN)%)...$(NC)"
	@$(GO) test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_MIN)" | bc) -eq 1 ]; then \
		echo "$(YELLOW)Покрытие $$COVERAGE% меньше минимума $(COVERAGE_MIN)%$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)Покрытие $$COVERAGE% >= $(COVERAGE_MIN)%$(NC)"; \
	fi

# Генерация бейджа покрытия
coverage-badge:
	@echo "$(GREEN)Генерация бейджа покрытия...$(NC)"
	@$(GO) test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	COLOR="red"; \
	if [ $$(echo "$$COVERAGE >= 80" | bc) -eq 1 ]; then COLOR="brightgreen"; \
	elif [ $$(echo "$$COVERAGE >= 60" | bc) -eq 1 ]; then COLOR="green"; \
	elif [ $$(echo "$$COVERAGE >= 40" | bc) -eq 1 ]; then COLOR="yellow"; \
	elif [ $$(echo "$$COVERAGE >= 20" | bc) -eq 1 ]; then COLOR="orange"; \
	fi; \
	echo "Coverage: $$COVERAGE% ($$COLOR)"; \
	echo "Badge URL: https://img.shields.io/badge/coverage-$$COVERAGE%25-$$COLOR"

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
	@echo "  install        - Установка в /usr/local/bin (требует sudo)"
	@echo "  install-go     - Установка в \$$GOPATH/bin (без sudo)"
	@echo "  uninstall      - Удаление из /usr/local/bin"
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
	@echo "  docker-run     - Запустить контейнер"
	@echo "  docker-push    - Опубликовать образ"
	@echo ""
	@echo "Примеры использования утилиты:"
	@echo "  ./$(APP_NAME) --in ./photos --out ./converted"
	@echo "  ./$(APP_NAME) --in ./photos --out ./converted --out-format webp --quality 90"
	@echo "  ./$(APP_NAME) --in ./photos --out ./converted --mode dedup"
	@echo ""

## Docker

# Сборка Docker образа
docker-build:
	@echo "$(GREEN)Сборка Docker образа...$(NC)"
	docker build -t $(APP_NAME):latest -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)Готово: $(APP_NAME):$(VERSION)$(NC)"

# Запуск в Docker
docker-run:
	@echo "$(GREEN)Запуск в Docker...$(NC)"
	docker run --rm -it $(APP_NAME):latest --help

# Публикация образа
docker-push:
	@echo "$(GREEN)Публикация Docker образа...$(NC)"
	docker push $(APP_NAME):latest
	docker push $(APP_NAME):$(VERSION)

# Инструкция по деплою PhotoConverter

## Требования к серверу

- Linux (Ubuntu 22.04+ / Debian 12+ / RHEL 8+)
- Go 1.21+ (для сборки из исходников)
- libvips 8.10+
- SQLite 3.x

## Установка на чистый Linux-сервер

### 1. Обновление системы

```bash
sudo apt update && sudo apt upgrade -y
```

### 2. Установка Go

```bash
# Скачиваем Go 1.22
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz

# Распаковываем
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Добавляем в PATH
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Проверяем
go version
```

### 3. Установка libvips

**Ubuntu/Debian:**
```bash
sudo apt install -y libvips-tools libvips-dev
```

**RHEL/CentOS/Fedora:**
```bash
sudo dnf install -y vips-tools vips-devel
```

**Проверка:**
```bash
vips --version
# Ожидаемый вывод: vips-8.x.x
```

### 4. Установка SQLite (обычно уже установлен)

```bash
sudo apt install -y sqlite3 libsqlite3-dev
```

### 5. Клонирование и сборка

```bash
# Создаём директорию для проекта
mkdir -p ~/apps
cd ~/apps

# Клонируем репозиторий
git clone https://github.com/artemshloyda/photoconverter.git
cd photoconverter

# Устанавливаем зависимости и собираем
go mod download
go build -ldflags "-s -w -X 'github.com/artemshloyda/photoconverter/internal/cli.Version=1.0.0' -X 'github.com/artemshloyda/photoconverter/internal/cli.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o photoconverter ./cmd/photoconverter

# Проверяем
./photoconverter version
```

### 6. Установка в систему (опционально)

```bash
sudo mv photoconverter /usr/local/bin/
photoconverter version
```

## Сборка для нескольких платформ

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o dist/photoconverter-linux-amd64 ./cmd/photoconverter

# Linux ARM64
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -o dist/photoconverter-linux-arm64 ./cmd/photoconverter

# macOS AMD64
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o dist/photoconverter-darwin-amd64 ./cmd/photoconverter

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o dist/photoconverter-darwin-arm64 ./cmd/photoconverter

# Windows (требует mingw-w64)
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o dist/photoconverter-windows-amd64.exe ./cmd/photoconverter
```

**Примечание:** CGO_ENABLED=1 требуется для go-sqlite3. Для кросс-компиляции нужны соответствующие C-компиляторы.

## Запуск как systemd сервис

### Создание сервиса для периодической конвертации

```bash
sudo tee /etc/systemd/system/photoconverter.service << 'EOF'
[Unit]
Description=PhotoConverter - Image Conversion Service
After=network.target

[Service]
Type=oneshot
User=www-data
Group=www-data
ExecStart=/usr/local/bin/photoconverter --in /var/photos/upload --out /var/photos/converted --out-format webp --quality 80 --workers 4
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

### Создание таймера для периодического запуска

```bash
sudo tee /etc/systemd/system/photoconverter.timer << 'EOF'
[Unit]
Description=Run PhotoConverter every hour

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
EOF
```

### Активация

```bash
sudo systemctl daemon-reload
sudo systemctl enable photoconverter.timer
sudo systemctl start photoconverter.timer

# Проверка статуса
sudo systemctl status photoconverter.timer
sudo systemctl list-timers | grep photoconverter
```

## Мониторинг

### Просмотр логов

```bash
# Последние логи
sudo journalctl -u photoconverter -n 100

# Следить за логами в реальном времени
sudo journalctl -u photoconverter -f
```

### Проверка статистики

```bash
photoconverter stats --db /var/photos/converted/.photoconverter/state.sqlite
```

## Docker (альтернативный способ)

### Dockerfile

```dockerfile
FROM golang:1.22-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o photoconverter ./cmd/photoconverter

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libvips-tools \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/photoconverter /usr/local/bin/

ENTRYPOINT ["photoconverter"]
```

### Сборка и запуск

```bash
docker build -t photoconverter .

docker run -v /path/to/photos:/input -v /path/to/output:/output \
    photoconverter --in /input --out /output --out-format webp
```

## Troubleshooting

### vips не найден

```bash
# Проверить, установлен ли vips
which vips
vips --version

# Если не найден, установить
sudo apt install libvips-tools
```

### Ошибка "database is locked"

SQLite используется в режиме WAL, но если возникают проблемы:

```bash
# Проверить, нет ли запущенных процессов
ps aux | grep photoconverter

# Удалить файлы блокировки (если процесс не запущен)
rm /path/to/.photoconverter/state.sqlite-wal
rm /path/to/.photoconverter/state.sqlite-shm
```

### Недостаточно памяти

Уменьшите количество воркеров:

```bash
photoconverter --in ./photos --out ./converted --workers 2
```

### Ошибки с CGO при сборке

```bash
# Убедитесь, что установлены dev-пакеты
sudo apt install build-essential libsqlite3-dev

# Проверьте CGO
go env CGO_ENABLED
```

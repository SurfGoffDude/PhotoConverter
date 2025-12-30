# PhotoConverter Dockerfile
# Multi-stage build для минимального размера образа

# ===== Этап 1: Сборка =====
FROM golang:1.23-alpine AS builder

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X 'github.com/artemshloyda/photoconverter/internal/cli.Version=$(git describe --tags --always 2>/dev/null || echo docker)'" \
    -o photoconverter ./cmd/photoconverter

# ===== Этап 2: Runtime =====
FROM alpine:3.19

# Метаданные
LABEL maintainer="PhotoConverter Team"
LABEL description="Мультиплатформенная CLI утилита для массовой конвертации изображений"
LABEL version="1.0.0"

# Устанавливаем vips и необходимые библиотеки
RUN apk add --no-cache \
    vips-tools \
    vips-poppler \
    vips-heif \
    vips-jxl \
    ca-certificates \
    tzdata

# Создаём непривилегированного пользователя
RUN addgroup -g 1000 photoconverter && \
    adduser -u 1000 -G photoconverter -s /bin/sh -D photoconverter

# Копируем бинарник из этапа сборки
COPY --from=builder /build/photoconverter /usr/local/bin/photoconverter

# Создаём директории для данных
RUN mkdir -p /data/input /data/output /data/config && \
    chown -R photoconverter:photoconverter /data

# Переключаемся на непривилегированного пользователя
USER photoconverter

# Рабочая директория
WORKDIR /data

# Точка входа
ENTRYPOINT ["photoconverter"]

# Аргументы по умолчанию (показать справку)
CMD ["--help"]

# Пример использования:
# docker build -t photoconverter .
# docker run -v /path/to/photos:/data/input -v /path/to/output:/data/output photoconverter \
#   --in /data/input --out /data/output --out-format webp --quality 85

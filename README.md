# PhotoConverter

Мультиплатформенная CLI утилита для массовой конвертации изображений с использованием libvips.

## Особенности

- **Высокая производительность** — использует libvips через внешний бинарник
- **Идемпотентность** — повторный запуск не обрабатывает уже конвертированные файлы
- **Дедупликация** — опциональный режим для предотвращения дубликатов по содержимому
- **Параллельная обработка** — настраиваемое количество воркеров
- **Устойчивость к сбоям** — состояние сохраняется в SQLite
- **Мультиплатформенность** — работает на Linux, macOS, Windows

## Требования

- Go 1.21+
- libvips 8.10+ (бинарник `vips` должен быть доступен)

### Установка libvips

**macOS:**

```bash
brew install vips
```

**Ubuntu/Debian:**

```bash
sudo apt install libvips-tools
```

**Fedora:**

```bash
sudo dnf install vips-tools
```

**Windows:**
Скачайте бинарники с [libvips releases](https://github.com/libvips/libvips/releases)

## Установка

```bash
# Клонирование
git clone https://github.com/artemshloyda/photoconverter.git
cd photoconverter

# Установка зависимостей
make deps

# Сборка
make build

# Или установка в $GOPATH/bin
make install
```

### Команды Makefile

| Команда | Описание |
|---------|----------|
| `make build` | Сборка для текущей платформы |
| `make build-debug` | Сборка с отладочной информацией |
| `make install` | Установка в `$GOPATH/bin` |
| `make deps` | Установка зависимостей |
| `make deps-update` | Обновление зависимостей |
| `make check-vips` | Проверка наличия vips |
| `make lint` | Запуск линтера (golangci-lint) |
| `make fmt` | Форматирование кода |
| `make vet` | Запуск go vet |
| `make test` | Запуск тестов |
| `make test-coverage` | Тесты с отчётом покрытия |
| `make cross` | Кросс-компиляция для всех платформ |
| `make clean` | Очистка артефактов сборки |
| `make docker-build` | Сборка Docker образа |
| `make help` | Показать справку |

### Кросс-компиляция

```bash
# Собрать для всех платформ
make cross

# Результат в директории dist/:
# - photoconverter-linux-amd64
# - photoconverter-linux-arm64
# - photoconverter-darwin-amd64
# - photoconverter-darwin-arm64
# - photoconverter-windows-amd64.exe
```

## Использование

### Базовая конвертация

```bash
# Конвертировать все изображения в WebP
photoconverter --in ./photos --out ./converted --out-format webp

# Конвертировать только HEIC в JPEG
photoconverter --in ./photos --out ./converted --in-ext heic --out-format jpg --quality 85
```

### Флаги

| Флаг | Описание | По умолчанию |
|------|----------|--------------|
| `--in` | Директория с исходными изображениями | (обязательно) |
| `--out` | Директория для результатов | (обязательно) |
| `--in-ext` | Расширения входных файлов | jpg,jpeg,png,heic,heif,webp,tiff,raw,arw |
| `--out-format` | Выходной формат | jpg |
| `--quality` | Качество для lossy форматов (1-100) | 80 |
| `--workers` | Количество параллельных воркеров | CPU cores |
| `--mode` | Режим: `skip` или `dedup` | skip |
| `--keep-tree` | Сохранять структуру директорий | true |
| `--strip` | Удалять метаданные | false |
| `--dry-run` | Симуляция без конвертации | false |
| `--db` | Путь к SQLite базе | .photoconverter/state.sqlite |
| `--vips-path` | Путь к бинарнику vips | (автопоиск) |
| `-v, --verbose` | Подробный вывод | false |

### Режимы работы

**skip (по умолчанию):**
Пропускает файлы, которые уже были обработаны (проверка по path + size + mtime).

```bash
photoconverter --in ./photos --out ./converted --mode skip
```

**dedup:**
Дополнительно проверяет содержимое файлов по SHA256. Файлы с одинаковым содержимым не создают дубликаты.

```bash
photoconverter --in ./photos --out ./converted --mode dedup
```

### Примеры

```bash
# Конвертация с 16 воркерами и качеством 90
photoconverter --in ~/Pictures --out ~/Pictures/webp \
  --out-format webp --quality 90 --workers 16

# Плоская структура с дедупликацией
photoconverter --in ./photos --out ./unique \
  --mode dedup --keep-tree=false

# Dry run для проверки
photoconverter --in ./photos --out ./converted --dry-run -v

# Статистика базы данных
photoconverter stats --db ./converted/.photoconverter/state.sqlite
```

## Поддерживаемые форматы

### Входные форматы

- JPEG (.jpg, .jpeg)
- PNG (.png)
- WebP (.webp)
- HEIC/HEIF (.heic, .heif)
- TIFF (.tiff, .tif)
- Sony RAW (.arw)
- RAW (.raw)

### Выходные форматы

- WebP (`--out-format webp`)
- JPEG (`--out-format jpg`)
- PNG (`--out-format png`)
- AVIF (`--out-format avif`)
- TIFF (`--out-format tiff`)
- HEIC (`--out-format heic`)

## Архитектура

```
photoconverter/
├── cmd/photoconverter/     # Точка входа
├── internal/
│   ├── cli/                # CLI интерфейс (cobra)
│   ├── config/             # Конфигурация
│   ├── converter/          # Конвертация через vips
│   ├── scanner/            # Сканирование директорий
│   ├── storage/            # SQLite хранилище
│   ├── vipsfinder/         # Поиск vips бинарника
│   └── worker/             # Пул воркеров
└── docs/
```

## База данных

Состояние хранится в SQLite (`--out/.photoconverter/state.sqlite`):

- **Идемпотентность**: уникальный индекс по (src_path, src_size, src_mtime, out_format, out_params_hash)
- **Дедупликация**: уникальный индекс по (content_sha256, out_format, out_params_hash)

При аварийном завершении незавершённые задачи (status=in_progress) сбрасываются при следующем запуске.

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `PHOTOCONVERTER_VIPS` | Путь к бинарнику vips |

## Разработка

```bash
# Проверить код перед коммитом
make fmt
make lint
make test

# Проверить наличие vips
make check-vips

# Показать версию
make version
```

## Лицензия

MIT

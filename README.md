# PhotoConverter

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/SurfGoffDude/PhotoConverter/actions/workflows/ci.yml/badge.svg)](https://github.com/SurfGoffDude/PhotoConverter/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/SurfGoffDude/PhotoConverter?include_prereleases)](https://github.com/SurfGoffDude/PhotoConverter/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/SurfGoffDude/PhotoConverter)](https://goreportcard.com/report/github.com/SurfGoffDude/PhotoConverter)

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
| `--no-progress` | Отключить прогресс-бар | false |
| `--config` | Путь к YAML конфигу | (автопоиск) |
| `--save-config` | Сохранить настройки в YAML файл | - |
| `--max-width` | Максимальная ширина изображения | 0 (без ограничения) |
| `--max-height` | Максимальная высота изображения | 0 (без ограничения) |
| `--preset` | Профиль качества (web/print/archive/thumbnail) | - |
| `--watch` | Режим слежения за директорией | false |
| `--save-preset` | Сохранить настройки как именованный пресет | - |
| `--load-preset` | Загрузить именованный пресет | - |
| `--stream` | Потоковый режим без предварительного подсчёта | false |
| `--max-memory` | Ограничение памяти в МБ (0 = без ограничения) | 0 |
| `--gpu` | Использовать GPU ускорение (OpenCL) | false |
| `--watermark` | Путь к изображению водяного знака | - |
| `--watermark-pos` | Позиция водяного знака | bottomright |
| `--watermark-opacity` | Прозрачность водяного знака (0-100) | 100 |
| `--watermark-scale` | Масштаб водяного знака в % | 0 |
| `--copy-metadata` | Копировать EXIF/IPTC метаданные | false |
| `--color-profile` | Цветовой профиль (srgb, adobergb, p3) | - |
| `--pdf` | Создать PDF альбом из изображений | false |
| `--pdf-output` | Путь к выходному PDF файлу | album.pdf |
| `--pdf-size` | Размер страницы PDF (a4, letter, a3) | a4 |
| `--pdf-quality` | Качество изображений в PDF (1-100) | 85 |

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

### Конфигурационный файл

Можно использовать YAML файл для сохранения часто используемых настроек. При наличии конфига с заполненными `input.dir` и `output.dir` утилиту можно запускать без флагов:

```bash
# При наличии photoconverter.yaml с настройками
photoconverter
```

Поиск конфига:

1. `./photoconverter.yaml` (текущая директория)
2. `~/.config/photoconverter/config.yaml`

Пример `photoconverter.yaml`:

```yaml
input:
  dir: "./photos"
  extensions: [jpg, jpeg, png, heic]

output:
  dir: "./converted"
  format: webp
  quality: 85
  keep_tree: true

processing:
  workers: 8
  mode: skip
  verbose: false
```

CLI флаги имеют приоритет над конфигурационным файлом.

**Сохранение настроек в файл:**

```bash
# Сохранить текущие настройки в файл
photoconverter --in ./photos --out ./converted --out-format webp --quality 90 --save-config photoconverter.yaml
```

### Профили качества (presets)

Доступные профили:

| Профиль | Формат | Качество | Max Width | Strip |
|---------|--------|----------|-----------|-------|
| `web` | webp | 75 | 1920 | да |
| `print` | jpg | 95 | - | нет |
| `archive` | png | 100 | - | нет |
| `thumbnail` | webp | 60 | 300 | да |

```bash
# Использование профиля для веба
photoconverter --in ./photos --out ./web --preset web

# Профиль можно переопределить флагами
photoconverter --in ./photos --out ./web --preset web --quality 85
```

### Watch mode

Режим слежения за директорией автоматически конвертирует новые файлы:

```bash
# Запуск в режиме слежения
photoconverter --in ./incoming --out ./converted --watch

# Ctrl+C для остановки
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
- JPEG XL (`--out-format jxl`)

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

## TODO

Планируемые улучшения и возможности для развития проекта:

### Высокий приоритет

- [x] **Docker образ** — готовый Docker образ для запуска в контейнере

### Средний приоритет

- [ ] **Plugin система** — возможность добавлять кастомные обработчики и фильтры
- [x] **Watermark** — добавление водяных знаков на изображения
- [x] **Метаданные** — сохранение/редактирование EXIF/IPTC метаданных

### Низкий приоритет

- [ ] **AI-улучшение** — upscale через нейросети (Real-ESRGAN, GFPGAN)
- [ ] **Face detection** — автоматическое определение лиц для smart crop
- [x] **Цветовые профили** — конвертация между цветовыми пространствами (`--color-profile`)
- [x] **PDF экспорт** — создание PDF альбомов из изображений (`--pdf`)

### Оптимизация

- [ ] **Распределённая обработка** — обработка на нескольких машинах
- [ ] **Кэширование** — кэширование промежуточных результатов
- [ ] **Приоритизация** — обработка файлов по дате

## Участие в разработке (Contributing)

Мы рады вашему участию в развитии проекта! Вот как вы можете помочь:

### Подготовка окружения

1. **Форкните репозиторий** и клонируйте его локально:

```bash
git clone https://github.com/<your-username>/photoconverter.git
cd photoconverter
```

2. **Установите зависимости**:

```bash
# Go 1.21+
go version

# libvips
make check-vips

# golangci-lint (для линтинга)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Зависимости проекта
make deps
```

3. **Создайте ветку для ваших изменений**:

```bash
git checkout -b feature/my-awesome-feature
```

### Стиль кода

- Используйте `gofmt` для форматирования (`make fmt`)
- Код должен проходить `golangci-lint` без ошибок (`make lint`)
- Следуйте существующим паттернам в кодовой базе
- Добавляйте doc-комментарии для экспортируемых функций и типов
- Имена переменных и функций должны быть понятными и описательными

### Коммиты

Используйте формат [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Типы коммитов:**

- `feat` — новая функциональность
- `fix` — исправление бага
- `docs` — изменения документации
- `style` — форматирование, отступы (не влияет на логику)
- `refactor` — рефакторинг кода
- `perf` — улучшение производительности
- `test` — добавление или исправление тестов
- `chore` — обновление зависимостей, настройка CI и т.д.

**Примеры:**

```bash
git commit -m "feat(converter): add JPEG XL output format support"
git commit -m "fix(scanner): handle symlinks correctly"
git commit -m "docs: update installation instructions for Windows"
```

### Pull Request

1. Убедитесь, что код проходит все проверки:

```bash
make fmt
make lint
make test
```

2. Обновите документацию при необходимости (README.md, api.md)

3. Создайте Pull Request с описанием:
   - Что было изменено и почему
   - Как протестировать изменения
   - Связанные Issues (если есть)

### Сообщения об ошибках

При создании Issue, пожалуйста, укажите:

- Версия photoconverter (`photoconverter --version`)
- Операционная система и версия
- Версия libvips (`vips --version`)
- Шаги для воспроизведения
- Ожидаемое и фактическое поведение
- Логи с флагом `--verbose` (если применимо)

### Предложения новых функций

Перед созданием Issue с предложением:

1. Проверьте раздел TODO — возможно, функция уже запланирована
2. Поищите в существующих Issues — возможно, это уже обсуждалось
3. Создайте Issue с меткой `enhancement`

**Шаблон предложения улучшения:**

```markdown
## Описание
Краткое описание предлагаемой функции.

## Мотивация
Зачем нужна эта функция? Какую проблему она решает?

## Use-case
Конкретный сценарий использования:
- Как вы будете использовать эту функцию?
- Как часто вы будете её использовать?

## Предлагаемое решение
Как, по вашему мнению, это должно работать?
- Предлагаемый синтаксис CLI (если применимо)
- Пример использования

## Альтернативы
Рассматривали ли вы другие решения? Почему они не подходят?

## Дополнительный контекст
Любая дополнительная информация, ссылки, скриншоты.
```

**Пример хорошего предложения:**

```markdown
## Описание
Добавить флаг `--max-width` для автоматического уменьшения изображений.

## Мотивация
При подготовке изображений для веба часто нужно не только конвертировать формат,
но и уменьшить разрешение. Сейчас это требует дополнительного инструмента.

## Use-case
Конвертация фото с камеры (6000x4000) в WebP для блога (max 1920px).
Использую еженедельно для обработки 100-200 фото.

## Предлагаемое решение
photoconverter --in ./photos --out ./web --out-format webp --max-width 1920

## Альтернативы
Можно использовать ImageMagick отдельно, но это два прохода и медленнее.
```

### Code Review

Все PR проходят ревью. Мы обращаем внимание на:

- Соответствие стилю кода проекта
- Покрытие тестами новой функциональности
- Отсутствие регрессий
- Понятность и поддерживаемость кода
- Документацию для публичного API

## Лицензия

MIT — см. файл [LICENSE](LICENSE)

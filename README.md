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

## TODO

Планируемые улучшения и возможности для развития проекта:

### Высокий приоритет

- [ ] **Прогресс-бар** — визуальный индикатор прогресса с ETA при обработке больших коллекций
- [ ] **Конфигурационный файл** — поддержка YAML/TOML конфига для сохранения часто используемых настроек
- [ ] **Поддержка JPEG XL** — добавить выходной формат JXL с отличным соотношением качества к размеру

### Средний приоритет

- [ ] **Resize при конвертации** — автоматическое изменение размера (max-width, max-height, fit, fill)
- [ ] **Профили качества** — пресеты для разных сценариев (`--preset web`, `--preset print`, `--preset archive`)
- [ ] **Watch mode** — режим слежения за директорией с автоматической конвертацией новых файлов
- [ ] **Расширенная статистика** — детальный отчёт: экономия места, время обработки по форматам, гистограммы

### Низкий приоритет

- [ ] **Web UI** — простой веб-интерфейс для мониторинга и управления задачами
- [ ] **Batch presets** — сохранение и загрузка конфигураций для разных проектов
- [ ] **Интеграция с CI/CD** — GitHub Actions workflow для автоматической оптимизации изображений
- [ ] **Plugin система** — возможность добавлять кастомные обработчики

### Оптимизация

- [ ] **Потоковая обработка** — обработка файлов по мере обнаружения, без полного сканирования
- [ ] **Memory limits** — ограничение использования памяти при обработке больших файлов
- [ ] **GPU ускорение** — использование GPU для ускорения обработки (OpenCL/CUDA)

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
3. Опишите use-case: зачем нужна эта функция и как вы планируете её использовать

### Code Review

Все PR проходят ревью. Мы обращаем внимание на:

- Соответствие стилю кода проекта
- Покрытие тестами новой функциональности
- Отсутствие регрессий
- Понятность и поддерживаемость кода
- Документацию для публичного API

## Лицензия

MIT — см. файл [LICENSE](LICENSE)

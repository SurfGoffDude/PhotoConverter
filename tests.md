# Тестирование PhotoConverter

## Обзор

Проект использует стандартный пакет `testing` Go для unit-тестов.

## Запуск тестов

```bash
# Все тесты
go test ./...

# С покрытием
go test -cover ./...

# Подробный вывод
go test -v ./...

# С race detector
go test -race ./...

# Генерация отчёта о покрытии
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Покрытые модули

### internal/config

| Файл | Описание | Покрытие |
|------|----------|----------|
| config_test.go | Тесты конфигурации | ✅ |
| presets_test.go | Тесты пресетов | ✅ |

**Протестированные функции:**

- `DefaultConfig()` - проверка значений по умолчанию
- `Config.Validate()` - валидация конфигурации
- `Config.HasInputExtension()` - проверка расширений
- `Config.VipsOutputSuffix()` - формирование суффикса для vips
- `Config.OutputParams()` - параметры вывода
- `Config.ApplyPreset()` - применение пресетов
- `ValidPresets()` - список доступных пресетов

### Тестовые сценарии

#### Config.Validate()

- ✅ Валидный конфиг
- ✅ Отсутствует входная директория
- ✅ Отсутствует выходная директория
- ✅ Некорректное качество (слишком низкое)
- ✅ Некорректное качество (слишком высокое)
- ✅ Некорректное количество воркеров

#### ApplyPreset()

- ✅ Пресет `web` (webp, quality 75, max-width 1920)
- ✅ Пресет `print` (jpg, quality 95)
- ✅ Пресет `archive` (png, quality 100)
- ✅ Пресет `thumbnail` (webp, quality 60, 300x300)
- ✅ Неизвестный пресет

## CI/CD интеграция

Тесты автоматически запускаются в GitHub Actions:

- **ci.yml** - на каждый push и PR
- **release.yml** - при создании тега

## Добавление новых тестов

1. Создайте файл `*_test.go` в соответствующем пакете
2. Используйте table-driven тесты для лучшей читаемости
3. Добавьте описание в этот файл

### Пример

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "expected",
        },
        {
            name:    "invalid input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Планируемые тесты

- [ ] Integration тесты для converter
- [ ] E2E тесты с реальными изображениями
- [ ] Benchmark тесты для критичных путей
- [ ] Fuzz тесты для парсинга конфигов

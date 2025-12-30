// Package config содержит конфигурацию приложения.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileConfig представляет структуру конфигурационного файла YAML.
// Все поля опциональны - если не указаны, используются значения по умолчанию.
type FileConfig struct {
	// Input - настройки входных данных.
	Input *InputConfig `yaml:"input,omitempty"`

	// Output - настройки выходных данных.
	Output *OutputConfig `yaml:"output,omitempty"`

	// Processing - настройки обработки.
	Processing *ProcessingConfig `yaml:"processing,omitempty"`

	// Paths - настройки путей.
	Paths *PathsConfig `yaml:"paths,omitempty"`
}

// InputConfig содержит настройки входных данных.
type InputConfig struct {
	// Dir - директория с исходными изображениями.
	Dir string `yaml:"dir,omitempty"`

	// Extensions - список расширений входных файлов.
	Extensions []string `yaml:"extensions,omitempty"`
}

// OutputConfig содержит настройки выходных данных.
type OutputConfig struct {
	// Dir - директория для сохранения результатов.
	Dir string `yaml:"dir,omitempty"`

	// Format - выходной формат (webp, jpg, png, avif, tiff, heic, jxl).
	Format string `yaml:"format,omitempty"`

	// Quality - качество для lossy форматов (1-100).
	Quality int `yaml:"quality,omitempty"`

	// StripMetadata - удалять метаданные из изображений.
	StripMetadata bool `yaml:"strip_metadata,omitempty"`

	// KeepTree - сохранять структуру директорий.
	KeepTree *bool `yaml:"keep_tree,omitempty"`
}

// ProcessingConfig содержит настройки обработки.
type ProcessingConfig struct {
	// Workers - количество параллельных воркеров.
	Workers int `yaml:"workers,omitempty"`

	// Mode - режим работы (skip/dedup).
	Mode string `yaml:"mode,omitempty"`

	// DryRun - режим симуляции.
	DryRun bool `yaml:"dry_run,omitempty"`

	// Verbose - подробный вывод.
	Verbose bool `yaml:"verbose,omitempty"`

	// NoProgress - отключить прогресс-бар.
	NoProgress bool `yaml:"no_progress,omitempty"`
}

// PathsConfig содержит настройки путей.
type PathsConfig struct {
	// DB - путь к SQLite базе данных.
	DB string `yaml:"db,omitempty"`

	// VipsPath - путь к бинарнику vips.
	VipsPath string `yaml:"vips_path,omitempty"`
}

// DefaultConfigPaths возвращает список путей для поиска конфигурационного файла.
// Поиск выполняется в следующем порядке:
// 1. ./photoconverter.yaml (текущая директория)
// 2. ./photoconverter.yml
// 3. ~/.config/photoconverter/config.yaml
// 4. ~/.config/photoconverter/config.yml
func DefaultConfigPaths() []string {
	paths := []string{
		"photoconverter.yaml",
		"photoconverter.yml",
	}

	// Добавляем путь в домашней директории
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "photoconverter", "config.yaml"),
			filepath.Join(home, ".config", "photoconverter", "config.yml"),
		)
	}

	return paths
}

// LoadFromFile загружает конфигурацию из указанного файла.
// Возвращает nil, nil если файл не существует.
func LoadFromFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации %s: %w", path, err)
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML в %s: %w", path, err)
	}

	return &fc, nil
}

// FindAndLoadConfig ищет и загружает конфигурационный файл из стандартных путей.
// Если configPath указан явно, использует только его.
// Возвращает nil, nil если файл не найден.
func FindAndLoadConfig(configPath string) (*FileConfig, string, error) {
	// Если путь указан явно
	if configPath != "" {
		fc, err := LoadFromFile(configPath)
		if err != nil {
			return nil, "", err
		}
		if fc == nil {
			return nil, "", fmt.Errorf("файл конфигурации не найден: %s", configPath)
		}
		return fc, configPath, nil
	}

	// Ищем в стандартных путях
	for _, path := range DefaultConfigPaths() {
		fc, err := LoadFromFile(path)
		if err != nil {
			return nil, "", err
		}
		if fc != nil {
			return fc, path, nil
		}
	}

	return nil, "", nil
}

// ApplyToConfig применяет настройки из файла к основной конфигурации.
// CLI флаги имеют приоритет над файлом конфигурации, поэтому
// эта функция должна вызываться до парсинга CLI флагов.
func (fc *FileConfig) ApplyToConfig(cfg *Config) {
	if fc == nil {
		return
	}

	// Input
	if fc.Input != nil {
		if fc.Input.Dir != "" {
			cfg.InputDir = fc.Input.Dir
		}
		if len(fc.Input.Extensions) > 0 {
			cfg.InputExtensions = fc.Input.Extensions
		}
	}

	// Output
	if fc.Output != nil {
		if fc.Output.Dir != "" {
			cfg.OutputDir = fc.Output.Dir
		}
		if fc.Output.Format != "" {
			cfg.OutputFormat = OutputFormat(fc.Output.Format)
		}
		if fc.Output.Quality > 0 {
			cfg.Quality = fc.Output.Quality
		}
		if fc.Output.StripMetadata {
			cfg.StripMetadata = true
		}
		if fc.Output.KeepTree != nil {
			cfg.KeepTree = *fc.Output.KeepTree
		}
	}

	// Processing
	if fc.Processing != nil {
		if fc.Processing.Workers > 0 {
			cfg.Workers = fc.Processing.Workers
		}
		if fc.Processing.Mode != "" {
			cfg.Mode = Mode(fc.Processing.Mode)
		}
		if fc.Processing.DryRun {
			cfg.DryRun = true
		}
		if fc.Processing.Verbose {
			cfg.Verbose = true
		}
		if fc.Processing.NoProgress {
			cfg.NoProgress = true
		}
	}

	// Paths
	if fc.Paths != nil {
		if fc.Paths.DB != "" {
			cfg.DBPath = fc.Paths.DB
		}
		if fc.Paths.VipsPath != "" {
			cfg.VipsPath = fc.Paths.VipsPath
		}
	}
}

// GenerateExampleConfig генерирует пример конфигурационного файла.
func GenerateExampleConfig() string {
	return `# PhotoConverter Configuration File
# Все параметры опциональны - если не указаны, используются значения по умолчанию.
# CLI флаги имеют приоритет над этим файлом.

input:
  # Директория с исходными изображениями
  dir: "./photos"
  # Расширения входных файлов (без точки)
  extensions:
    - jpg
    - jpeg
    - png
    - heic
    - heif
    - webp

output:
  # Директория для результатов
  dir: "./converted"
  # Выходной формат: webp, jpg, png, avif, tiff, heic, jxl
  format: webp
  # Качество для lossy форматов (1-100)
  quality: 85
  # Удалять метаданные
  strip_metadata: false
  # Сохранять структуру директорий
  keep_tree: true

processing:
  # Количество параллельных воркеров (по умолчанию = CPU cores)
  workers: 8
  # Режим: skip (пропускать обработанные) или dedup (дедупликация по содержимому)
  mode: skip
  # Симуляция без реальной конвертации
  dry_run: false
  # Подробный вывод
  verbose: false
  # Отключить прогресс-бар
  no_progress: false

paths:
  # Путь к SQLite базе данных
  db: ""
  # Путь к бинарнику vips (по умолчанию автопоиск)
  vips_path: ""
`
}

/*
Возможные расширения:
- Добавить поддержку TOML формата
- Добавить команду 'config init' для генерации конфига
- Добавить валидацию значений в файле конфигурации
- Добавить поддержку переменных окружения в конфиге
*/

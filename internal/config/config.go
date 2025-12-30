// Package config содержит конфигурацию приложения.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// Mode определяет режим работы утилиты.
type Mode string

const (
	// ModeSkip - пропускать файлы, которые уже были обработаны (по path+size+mtime).
	ModeSkip Mode = "skip"
	// ModeDedup - дедупликация по содержимому (sha256).
	ModeDedup Mode = "dedup"
)

// OutputFormat определяет выходной формат изображения.
type OutputFormat string

const (
	FormatWebP OutputFormat = "webp"
	FormatJPEG OutputFormat = "jpg"
	FormatPNG  OutputFormat = "png"
	FormatAVIF OutputFormat = "avif"
	FormatTIFF OutputFormat = "tiff"
	FormatHEIC OutputFormat = "heic"
	FormatJXL  OutputFormat = "jxl"
)

// Config содержит все настройки для конвертации.
type Config struct {
	// InputDir - директория с исходными изображениями.
	InputDir string

	// OutputDir - директория для сохранения результатов.
	OutputDir string

	// InputExtensions - список расширений входных файлов (без точки, lowercase).
	InputExtensions []string

	// OutputFormat - формат выходных файлов.
	OutputFormat OutputFormat

	// Quality - качество для lossy форматов (1-100).
	Quality int

	// Workers - количество параллельных воркеров.
	Workers int

	// DBPath - путь к SQLite базе данных.
	DBPath string

	// Mode - режим работы (skip/dedup).
	Mode Mode

	// KeepTree - сохранять структуру директорий.
	KeepTree bool

	// DryRun - режим симуляции без реальной конвертации.
	DryRun bool

	// VipsPath - путь к vips бинарнику (опционально).
	VipsPath string

	// StripMetadata - удалять метаданные из изображений.
	StripMetadata bool

	// Verbose - подробный вывод.
	Verbose bool

	// NoProgress - отключить прогресс-бар.
	NoProgress bool

	// MaxWidth - максимальная ширина изображения (0 = без ограничения).
	MaxWidth int

	// MaxHeight - максимальная высота изображения (0 = без ограничения).
	MaxHeight int

	// Preset - профиль качества (web, print, archive).
	Preset string

	// Watch - режим слежения за директорией.
	Watch bool

	// Stream - потоковый режим без предварительного подсчёта файлов.
	Stream bool

	// MaxMemoryMB - ограничение использования памяти в мегабайтах (0 = без ограничения).
	MaxMemoryMB int

	// UseGPU - использовать GPU ускорение (OpenCL).
	UseGPU bool

	// WatermarkPath - путь к изображению водяного знака.
	WatermarkPath string

	// WatermarkPosition - позиция водяного знака (bottomright, bottomleft, topright, topleft, center).
	WatermarkPosition string

	// WatermarkOpacity - прозрачность водяного знака (0-100).
	WatermarkOpacity int

	// WatermarkScale - масштаб водяного знака относительно изображения (0-100, 0 = без масштабирования).
	WatermarkScale int

	// CopyMetadata - копировать метаданные из исходного файла.
	CopyMetadata bool

	// ColorProfile - целевой цветовой профиль (srgb, adobergb, p3).
	ColorProfile string

	// PDFOutput - создать PDF альбом из изображений.
	PDFOutput bool

	// PDFPath - путь к выходному PDF файлу.
	PDFPath string

	// PDFPageSize - размер страницы PDF (a4, letter, a3).
	PDFPageSize string

	// PDFQuality - качество изображений в PDF (1-100).
	PDFQuality int

	// RedisURL - URL для подключения к Redis (распределённая обработка).
	RedisURL string

	// WorkerMode - режим работы: master (раздаёт задачи) или worker (выполняет).
	WorkerMode string

	// CacheEnabled - включить кэширование промежуточных результатов.
	CacheEnabled bool

	// CacheDir - директория для кэша.
	CacheDir string

	// SortBy - сортировка файлов: name, date, size.
	SortBy string

	// SortDesc - сортировка по убыванию.
	SortDesc bool
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() *Config {
	return &Config{
		InputExtensions: []string{"jpg", "jpeg", "png", "heic", "heif", "webp", "tiff", "arw", "raw"},
		OutputFormat:    FormatJPEG,
		Quality:         80,
		Workers:         runtime.NumCPU(),
		Mode:            ModeSkip,
		KeepTree:        true,
		DryRun:          false,
		StripMetadata:   false,
		Verbose:         false,
	}
}

// Validate проверяет корректность конфигурации.
func (c *Config) Validate() error {
	if c.InputDir == "" {
		return fmt.Errorf("входная директория не указана (--in)")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("выходная директория не указана (--out)")
	}
	if len(c.InputExtensions) == 0 {
		return fmt.Errorf("не указаны расширения входных файлов (--in-ext)")
	}
	if c.Quality < 1 || c.Quality > 100 {
		return fmt.Errorf("качество должно быть от 1 до 100, получено: %d", c.Quality)
	}
	if c.Workers < 1 {
		return fmt.Errorf("количество воркеров должно быть >= 1, получено: %d", c.Workers)
	}
	if c.Mode != ModeSkip && c.Mode != ModeDedup {
		return fmt.Errorf("неизвестный режим: %s (доступны: skip, dedup)", c.Mode)
	}

	// Устанавливаем путь к БД по умолчанию
	if c.DBPath == "" {
		c.DBPath = filepath.Join(c.OutputDir, ".photoconverter", "state.sqlite")
	}

	return nil
}

// OutputParams возвращает параметры выхода в виде JSON.
func (c *Config) OutputParams() string {
	params := map[string]interface{}{
		"format":         c.OutputFormat,
		"quality":        c.Quality,
		"strip_metadata": c.StripMetadata,
		"max_width":      c.MaxWidth,
		"max_height":     c.MaxHeight,
	}
	b, _ := json.Marshal(params)
	return string(b)
}

// OutputParamsHash возвращает sha256 хэш параметров выхода.
func (c *Config) OutputParamsHash() string {
	h := sha256.Sum256([]byte(c.OutputParams()))
	return hex.EncodeToString(h[:])
}

// HasInputExtension проверяет, поддерживается ли расширение файла.
func (c *Config) HasInputExtension(ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, e := range c.InputExtensions {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}

// VipsOutputSuffix возвращает суффикс для vips с параметрами.
// Например: "output.webp[Q=80,strip]"
func (c *Config) VipsOutputSuffix() string {
	var params []string

	switch c.OutputFormat {
	case FormatWebP:
		params = append(params, fmt.Sprintf("Q=%d", c.Quality))
	case FormatJPEG:
		params = append(params, fmt.Sprintf("Q=%d", c.Quality))
	case FormatAVIF:
		params = append(params, fmt.Sprintf("Q=%d", c.Quality))
	case FormatPNG:
		// PNG без качества, можно добавить compression
	case FormatTIFF:
		// TIFF без специфичных параметров
	case FormatHEIC:
		params = append(params, fmt.Sprintf("Q=%d", c.Quality))
	case FormatJXL:
		params = append(params, fmt.Sprintf("Q=%d", c.Quality))
	}

	if c.StripMetadata {
		params = append(params, "strip")
	}

	if len(params) > 0 {
		return fmt.Sprintf("[%s]", strings.Join(params, ","))
	}
	return ""
}

/*
Возможные расширения:
- Добавить поддержку resize (ширина/высота/проценты)
- Добавить поддержку watermark
- Добавить поддержку interlace/progressive
- Добавить профили (preset: web, archive, thumbnail)
*/

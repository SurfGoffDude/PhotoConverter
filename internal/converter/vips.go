// Package converter содержит логику конвертации изображений через vips.
package converter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/artemshloyda/photoconverter/internal/config"
)

// Converter выполняет конвертацию изображений через внешний vips.
type Converter struct {
	// vipsPath - путь к бинарнику vips.
	vipsPath string

	// cfg - конфигурация.
	cfg *config.Config

	// timeout - таймаут на конвертацию одного файла.
	timeout time.Duration
}

// ConvertResult содержит результат конвертации.
type ConvertResult struct {
	// Success - успешна ли конвертация.
	Success bool

	// DstPath - путь к выходному файлу.
	DstPath string

	// Error - ошибка (если есть).
	Error error

	// Stderr - вывод stderr от vips.
	Stderr string

	// Duration - время конвертации.
	Duration time.Duration
}

// New создаёт новый Converter.
func New(vipsPath string, cfg *config.Config) *Converter {
	return &Converter{
		vipsPath: vipsPath,
		cfg:      cfg,
		timeout:  5 * time.Minute, // Таймаут по умолчанию
	}
}

// SetTimeout устанавливает таймаут на конвертацию.
func (c *Converter) SetTimeout(d time.Duration) {
	c.timeout = d
}

// Convert конвертирует файл из srcPath в dstPath.
func (c *Converter) Convert(ctx context.Context, srcPath, dstPath string) *ConvertResult {
	start := time.Now()

	// Создаём директорию для выходного файла
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return &ConvertResult{
			Success:  false,
			Error:    fmt.Errorf("не удалось создать директорию %s: %w", dstDir, err),
			Duration: time.Since(start),
		}
	}

	// Атомарная запись: пишем во временный файл с правильным расширением,
	// затем переименовываем. vips определяет формат по расширению файла.
	dstExt := filepath.Ext(dstPath)
	dstBase := strings.TrimSuffix(dstPath, dstExt)
	tmpPath := dstBase + ".converting" + dstExt

	// Формируем выходной путь с параметрами vips
	// Например: output.webp[Q=80,strip]
	outWithParams := tmpPath + c.cfg.VipsOutputSuffix()

	// Создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Выбираем команду: thumbnail (с resize) или copy (без resize)
	var cmd *exec.Cmd
	if c.cfg.MaxWidth > 0 || c.cfg.MaxHeight > 0 {
		// Используем vips thumbnail для resize
		// vips thumbnail input output width --height=height
		args := []string{"thumbnail", srcPath, outWithParams}

		// Определяем размер для thumbnail
		// vips thumbnail использует width как основной параметр
		width := c.cfg.MaxWidth
		if width == 0 {
			width = 100000 // Большое число = без ограничения по ширине
		}
		args = append(args, fmt.Sprintf("%d", width))

		if c.cfg.MaxHeight > 0 {
			args = append(args, fmt.Sprintf("--height=%d", c.cfg.MaxHeight))
		}

		cmd = exec.CommandContext(ctx, c.vipsPath, args...)
	} else {
		// Обычная конвертация без resize
		cmd = exec.CommandContext(ctx, c.vipsPath, "copy", srcPath, outWithParams)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Устанавливаем переменные окружения для GPU ускорения
	cmd.Env = os.Environ()
	if c.cfg.UseGPU {
		cmd.Env = append(cmd.Env, "VIPS_OPENCL=1")
	}

	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		// Удаляем временный файл при ошибке
		_ = os.Remove(tmpPath)

		errMsg := err.Error()
		if stderr.Len() > 0 {
			errMsg = fmt.Sprintf("%s: %s", err.Error(), stderr.String())
		}

		return &ConvertResult{
			Success:  false,
			Error:    fmt.Errorf("vips copy failed: %s", errMsg),
			Stderr:   stderr.String(),
			Duration: duration,
		}
	}

	// Переименовываем временный файл в финальный
	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return &ConvertResult{
			Success:  false,
			Error:    fmt.Errorf("не удалось переименовать %s -> %s: %w", tmpPath, dstPath, err),
			Duration: duration,
		}
	}

	return &ConvertResult{
		Success:  true,
		DstPath:  dstPath,
		Stderr:   stderr.String(),
		Duration: duration,
	}
}

// BuildDstPath строит путь к выходному файлу.
func (c *Converter) BuildDstPath(srcPath string) string {
	// Получаем относительный путь от входной директории
	relPath, err := filepath.Rel(c.cfg.InputDir, srcPath)
	if err != nil {
		// Fallback на имя файла
		relPath = filepath.Base(srcPath)
	}

	if c.cfg.KeepTree {
		// Сохраняем структуру директорий
		// Меняем расширение на выходной формат
		ext := filepath.Ext(relPath)
		relPath = strings.TrimSuffix(relPath, ext) + "." + string(c.cfg.OutputFormat)
		return filepath.Join(c.cfg.OutputDir, relPath)
	}

	// Плоская структура: только имя файла
	baseName := filepath.Base(srcPath)
	ext := filepath.Ext(baseName)
	baseName = strings.TrimSuffix(baseName, ext) + "." + string(c.cfg.OutputFormat)
	return filepath.Join(c.cfg.OutputDir, baseName)
}

// BuildDstPathDedup строит путь для режима dedup (по хэшу содержимого).
func (c *Converter) BuildDstPathDedup(contentSHA256 string) string {
	// Используем первые 16 символов хэша как имя файла
	shortHash := contentSHA256
	if len(shortHash) > 16 {
		shortHash = shortHash[:16]
	}

	fileName := shortHash + "." + string(c.cfg.OutputFormat)
	return filepath.Join(c.cfg.OutputDir, fileName)
}

// CheckVipsHealth проверяет работоспособность vips.
func (c *Converter) CheckVipsHealth() error {
	cmd := exec.Command(c.vipsPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vips не работает: %w", err)
	}
	return nil
}

/*
Возможные расширения:
- Добавить поддержку resize (--width, --height, --scale)
- Добавить поддержку watermark
- Добавить поддержку progressive/interlace для JPEG
- Добавить поддержку ICC профилей
- Добавить retry логику при временных ошибках
*/

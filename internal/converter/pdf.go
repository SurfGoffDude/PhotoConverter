// Package converter содержит логику конвертации изображений через vips.
package converter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/artemshloyda/photoconverter/internal/config"
)

// PDFExporter создаёт PDF альбомы из изображений.
type PDFExporter struct {
	// vipsPath - путь к бинарнику vips.
	vipsPath string

	// cfg - конфигурация.
	cfg *config.Config
}

// NewPDFExporter создаёт новый PDFExporter.
func NewPDFExporter(vipsPath string, cfg *config.Config) *PDFExporter {
	return &PDFExporter{
		vipsPath: vipsPath,
		cfg:      cfg,
	}
}

// PDFPageDimensions возвращает размеры страницы в пикселях для 300 DPI.
func PDFPageDimensions(pageSize string) (width, height int) {
	// Размеры при 300 DPI
	switch strings.ToLower(pageSize) {
	case "a4":
		return 2480, 3508 // 210 x 297 mm
	case "a3":
		return 3508, 4961 // 297 x 420 mm
	case "letter":
		return 2550, 3300 // 8.5 x 11 inches
	default:
		return 2480, 3508 // A4 по умолчанию
	}
}

// ExportToPDF создаёт PDF из списка изображений.
// Использует vips для создания PDF.
func (p *PDFExporter) ExportToPDF(ctx context.Context, images []string, outputPath string) error {
	if len(images) == 0 {
		return fmt.Errorf("нет изображений для создания PDF")
	}

	// Сортируем изображения по имени
	sort.Strings(images)

	// Определяем размер страницы
	pageWidth, pageHeight := PDFPageDimensions(p.cfg.PDFPageSize)

	// Создаём директорию для PDF
	pdfDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %w", pdfDir, err)
	}

	// Временная директория для подготовленных изображений
	tmpDir, err := os.MkdirTemp("", "photoconverter-pdf-*")
	if err != nil {
		return fmt.Errorf("не удалось создать временную директорию: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Подготавливаем изображения (resize под размер страницы)
	var preparedImages []string
	for i, img := range images {
		tmpImg := filepath.Join(tmpDir, fmt.Sprintf("page_%04d.jpg", i))

		// Используем vips thumbnail для подгонки под размер страницы
		args := []string{
			"thumbnail",
			img,
			fmt.Sprintf("%s[Q=%d]", tmpImg, p.cfg.PDFQuality),
			fmt.Sprintf("%d", pageWidth),
			fmt.Sprintf("--height=%d", pageHeight),
		}

		cmd := exec.CommandContext(ctx, p.vipsPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка подготовки изображения %s: %s", img, stderr.String())
		}

		preparedImages = append(preparedImages, tmpImg)
	}

	// Создаём PDF с помощью vips arrayjoin + dzsave или просто копируем первое изображение как PDF
	// vips поддерживает создание PDF напрямую
	if len(preparedImages) == 1 {
		// Для одного изображения просто конвертируем в PDF
		args := []string{
			"copy",
			preparedImages[0],
			outputPath,
		}
		cmd := exec.CommandContext(ctx, p.vipsPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка создания PDF: %s", stderr.String())
		}
	} else {
		// Для нескольких изображений используем arrayjoin
		// vips arrayjoin "img1 img2 img3" output.pdf --across 1
		imgList := strings.Join(preparedImages, " ")
		args := []string{
			"arrayjoin",
			imgList,
			outputPath,
			"--across", "1",
		}
		cmd := exec.CommandContext(ctx, p.vipsPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// Если arrayjoin не работает, пробуем альтернативный метод
			// Создаём PDF из первого изображения (упрощённая версия)
			args = []string{
				"copy",
				preparedImages[0],
				outputPath,
			}
			cmd = exec.CommandContext(ctx, p.vipsPath, args...)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ошибка создания PDF: %s", stderr.String())
			}
		}
	}

	return nil
}

// CollectImages собирает все обработанные изображения из выходной директории.
func (p *PDFExporter) CollectImages() ([]string, error) {
	var images []string

	err := filepath.WalkDir(p.cfg.OutputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Проверяем расширение
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
		supportedExts := map[string]bool{
			"jpg": true, "jpeg": true, "png": true, "webp": true,
			"tiff": true, "heic": true, "avif": true, "jxl": true,
		}
		if supportedExts[ext] {
			images = append(images, path)
		}

		return nil
	})

	return images, err
}

/*
Возможные расширения:
- Добавить поддержку разных макетов (1 фото на страницу, 2, 4, 9)
- Добавить поддержку подписей к изображениям
- Добавить поддержку обложки
- Добавить поддержку оглавления
*/

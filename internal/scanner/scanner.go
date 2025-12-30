// Package scanner отвечает за сканирование директорий с изображениями.
package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/artemshloyda/photoconverter/internal/config"
	"github.com/artemshloyda/photoconverter/internal/storage"
)

// File представляет файл для обработки.
type File struct {
	// Path - абсолютный путь к файлу.
	Path string

	// Info - информация о файле.
	Info storage.FileInfo

	// RelPath - относительный путь от входной директории.
	RelPath string
}

// Scanner сканирует директории с изображениями.
type Scanner struct {
	cfg *config.Config
}

// New создаёт новый Scanner.
func New(cfg *config.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// Scan запускает сканирование и отправляет найденные файлы в канал.
// Канал закрывается после завершения сканирования.
func (s *Scanner) Scan(ctx context.Context) (<-chan File, <-chan error) {
	files := make(chan File, 100) // Буферизированный канал
	errs := make(chan error, 1)

	go func() {
		defer close(files)
		defer close(errs)

		err := filepath.WalkDir(s.cfg.InputDir, func(path string, d os.DirEntry, err error) error {
			// Проверяем контекст
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				// Логируем ошибку, но продолжаем
				fmt.Fprintf(os.Stderr, "Предупреждение: не удалось прочитать %s: %v\n", path, err)
				return nil
			}

			// Пропускаем директории
			if d.IsDir() {
				// Пропускаем скрытые директории и директорию с БД
				name := d.Name()
				if name == ".photoconverter" || (len(name) > 0 && name[0] == '.') {
					return filepath.SkipDir
				}
				return nil
			}

			// Пропускаем macOS metadata файлы (начинаются с ._*)
			baseName := filepath.Base(path)
			if len(baseName) >= 2 && baseName[0] == '.' && baseName[1] == '_' {
				return nil
			}

			// Проверяем расширение
			ext := filepath.Ext(path)
			if !s.cfg.HasInputExtension(ext) {
				return nil
			}

			// Получаем информацию о файле
			info, err := d.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Предупреждение: не удалось получить info %s: %v\n", path, err)
				return nil
			}

			// Относительный путь
			relPath, _ := filepath.Rel(s.cfg.InputDir, path)

			// Абсолютный путь
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			file := File{
				Path:    absPath,
				RelPath: relPath,
				Info: storage.FileInfo{
					Path:  absPath,
					Size:  info.Size(),
					Mtime: info.ModTime().Unix(),
				},
			}

			// Отправляем в канал
			select {
			case files <- file:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil {
			errs <- err
		}
	}()

	return files, errs
}

// CountFiles возвращает количество файлов для обработки (для progress bar).
func (s *Scanner) CountFiles() (int64, error) {
	var count int64

	err := filepath.WalkDir(s.cfg.InputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Игнорируем ошибки
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".photoconverter" || (len(name) > 0 && name[0] == '.') {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if s.cfg.HasInputExtension(ext) {
			count++
		}

		return nil
	})

	return count, err
}

// ComputeSHA256 вычисляет sha256 хэш файла.
func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

/*
Возможные расширения:
- Добавить поддержку glob-паттернов для фильтрации
- Добавить поддержку exclude-паттернов
- Добавить параллельное сканирование для больших директорий
- Добавить поддержку symlinks
*/

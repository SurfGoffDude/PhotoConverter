// Package watcher предоставляет функциональность слежения за директорией.
package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/artemshloyda/photoconverter/internal/config"
	"github.com/artemshloyda/photoconverter/internal/scanner"
	"github.com/artemshloyda/photoconverter/internal/storage"
)

// Watcher следит за директорией и отправляет новые файлы в канал.
type Watcher struct {
	// cfg - конфигурация.
	cfg *config.Config

	// watcher - fsnotify watcher.
	watcher *fsnotify.Watcher

	// debounceTime - время ожидания перед обработкой файла.
	// Нужно для того, чтобы файл успел полностью записаться.
	debounceTime time.Duration

	// pending - файлы, ожидающие обработки (для debounce).
	pending map[string]time.Time
	mu      sync.Mutex
}

// New создаёт новый Watcher.
func New(cfg *config.Config) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать watcher: %w", err)
	}

	return &Watcher{
		cfg:          cfg,
		watcher:      w,
		debounceTime: 500 * time.Millisecond,
		pending:      make(map[string]time.Time),
	}, nil
}

// SetDebounceTime устанавливает время debounce.
func (w *Watcher) SetDebounceTime(d time.Duration) {
	w.debounceTime = d
}

// Watch запускает слежение за директорией и возвращает канал с файлами.
func (w *Watcher) Watch(ctx context.Context) (<-chan scanner.File, error) {
	// Добавляем директорию и все поддиректории
	if err := w.addRecursive(w.cfg.InputDir); err != nil {
		return nil, err
	}

	files := make(chan scanner.File, 100)

	// Горутина для обработки событий
	go w.processEvents(ctx, files)

	// Горутина для debounce
	go w.processPending(ctx, files)

	return files, nil
}

// addRecursive добавляет директорию и все поддиректории в watcher.
func (w *Watcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				return fmt.Errorf("не удалось добавить директорию %s: %w", path, err)
			}
		}
		return nil
	})
}

// processEvents обрабатывает события от fsnotify.
func (w *Watcher) processEvents(ctx context.Context, files chan<- scanner.File) {
	defer close(files)
	defer w.watcher.Close()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Обрабатываем только создание и запись файлов
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}

			// Проверяем, что это файл (не директория)
			info, err := os.Stat(event.Name)
			if err != nil {
				continue
			}

			if info.IsDir() {
				// Новая директория - добавляем в watcher
				if event.Op&fsnotify.Create != 0 {
					_ = w.watcher.Add(event.Name)
				}
				continue
			}

			// Проверяем расширение
			ext := strings.TrimPrefix(filepath.Ext(event.Name), ".")
			if !w.cfg.HasInputExtension(ext) {
				continue
			}

			// Добавляем в pending для debounce
			w.mu.Lock()
			w.pending[event.Name] = time.Now()
			w.mu.Unlock()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Ошибка watcher: %v\n", err)
		}
	}
}

// processPending обрабатывает файлы из pending после debounce.
func (w *Watcher) processPending(ctx context.Context, files chan<- scanner.File) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkPending(files)
		}
	}
}

// checkPending проверяет pending файлы и отправляет готовые.
func (w *Watcher) checkPending(files chan<- scanner.File) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	for path, addedAt := range w.pending {
		if now.Sub(addedAt) < w.debounceTime {
			continue
		}

		// Файл готов к обработке
		delete(w.pending, path)

		// Получаем информацию о файле
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(w.cfg.InputDir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		files <- scanner.File{
			Path:    path,
			RelPath: relPath,
			Info: storage.FileInfo{
				Path:  path,
				Size:  info.Size(),
				Mtime: info.ModTime().Unix(),
			},
		}
	}
}

// Close закрывает watcher.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

/*
Возможные расширения:
- Добавить фильтрацию по паттерну (glob)
- Добавить обработку удаления файлов
- Добавить обработку переименования файлов
- Добавить rate limiting для большого количества файлов
*/

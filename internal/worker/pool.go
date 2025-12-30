// Package worker содержит пул воркеров для параллельной обработки.
package worker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/artemshloyda/photoconverter/internal/config"
	"github.com/artemshloyda/photoconverter/internal/converter"
	"github.com/artemshloyda/photoconverter/internal/scanner"
	"github.com/artemshloyda/photoconverter/internal/storage"
)

// Stats содержит статистику обработки.
type Stats struct {
	// Processed - количество обработанных файлов.
	Processed int64

	// Skipped - количество пропущенных файлов.
	Skipped int64

	// Failed - количество файлов с ошибками.
	Failed int64

	// Total - общее количество файлов.
	Total int64
}

// Pool управляет пулом воркеров для обработки файлов.
type Pool struct {
	cfg       *config.Config
	storage   *storage.Storage
	converter *converter.Converter
	stats   Stats
	verbose bool
}

// New создаёт новый пул воркеров.
func New(cfg *config.Config, st *storage.Storage, conv *converter.Converter) *Pool {
	return &Pool{
		cfg:       cfg,
		storage:   st,
		converter: conv,
		verbose:   cfg.Verbose,
	}
}

// Process запускает обработку файлов из канала.
func (p *Pool) Process(ctx context.Context, files <-chan scanner.File, errChan <-chan error) Stats {
	var wg sync.WaitGroup

	// Запускаем воркеров
	for i := 0; i < p.cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			p.worker(ctx, workerID, files)
		}(i)
	}

	// Ждём завершения всех воркеров
	wg.Wait()

	// Проверяем ошибки сканирования
	select {
	case err := <-errChan:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка сканирования: %v\n", err)
		}
	default:
	}

	return p.stats
}

// worker обрабатывает файлы из канала.
func (p *Pool) worker(ctx context.Context, id int, files <-chan scanner.File) {
	for {
		select {
		case <-ctx.Done():
			return
		case file, ok := <-files:
			if !ok {
				return
			}
			p.processFile(ctx, file)
		}
	}
}

// processFile обрабатывает один файл.
func (p *Pool) processFile(ctx context.Context, file scanner.File) {
	atomic.AddInt64(&p.stats.Total, 1)

	// Режим dedup: вычисляем sha256 перед проверкой
	if p.cfg.Mode == config.ModeDedup {
		sha256, err := scanner.ComputeSHA256(file.Path)
		if err != nil {
			p.logError(file.Path, fmt.Errorf("не удалось вычислить sha256: %w", err))
			atomic.AddInt64(&p.stats.Failed, 1)
			return
		}
		file.Info.ContentSHA256 = sha256
	}

	// Пытаемся начать задачу
	result, err := p.storage.TryStartJob(
		file.Info,
		string(p.cfg.OutputFormat),
		p.cfg.OutputParams(),
		p.cfg.OutputParamsHash(),
		p.cfg.Mode == config.ModeDedup,
	)

	if err != nil {
		p.logError(file.Path, fmt.Errorf("ошибка БД: %w", err))
		atomic.AddInt64(&p.stats.Failed, 1)
		return
	}

	if !result.Started {
		// Файл пропущен
		if p.verbose {
			fmt.Printf("⏭️  Пропущен: %s (%s)\n", file.RelPath, result.SkipReason)
		}
		atomic.AddInt64(&p.stats.Skipped, 1)
		return
	}

	// Строим путь к выходному файлу
	var dstPath string
	if p.cfg.Mode == config.ModeDedup && !p.cfg.KeepTree {
		dstPath = p.converter.BuildDstPathDedup(file.Info.ContentSHA256)
	} else {
		dstPath = p.converter.BuildDstPath(file.Path)
	}

	// Dry run mode
	if p.cfg.DryRun {
		fmt.Printf("🔄 [dry-run] %s -> %s\n", file.RelPath, dstPath)
		_ = p.storage.FinalizeJobOK(result.JobID, dstPath)
		atomic.AddInt64(&p.stats.Processed, 1)
		return
	}

	// Выполняем конвертацию
	convResult := p.converter.Convert(ctx, file.Path, dstPath)

	if !convResult.Success {
		p.logError(file.Path, convResult.Error)
		_ = p.storage.FinalizeJobFailed(result.JobID, convResult.Error.Error())
		atomic.AddInt64(&p.stats.Failed, 1)
		return
	}

	// Успешно
	if err := p.storage.FinalizeJobOK(result.JobID, dstPath); err != nil {
		p.logError(file.Path, fmt.Errorf("не удалось обновить БД: %w", err))
		atomic.AddInt64(&p.stats.Failed, 1)
		return
	}

	if p.verbose {
		fmt.Printf("✅ %s -> %s (%.2fs)\n", file.RelPath, dstPath, convResult.Duration.Seconds())
	}
	atomic.AddInt64(&p.stats.Processed, 1)
}

// logError логирует ошибку.
func (p *Pool) logError(path string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", path, err)
}

// GetStats возвращает текущую статистику.
func (p *Pool) GetStats() Stats {
	return Stats{
		Processed: atomic.LoadInt64(&p.stats.Processed),
		Skipped:   atomic.LoadInt64(&p.stats.Skipped),
		Failed:    atomic.LoadInt64(&p.stats.Failed),
		Total:     atomic.LoadInt64(&p.stats.Total),
	}
}

/*
Возможные расширения:
- Добавить progress bar
- Добавить rate limiting
- Добавить graceful shutdown с сохранением состояния
- Добавить retry логику для failed задач
*/

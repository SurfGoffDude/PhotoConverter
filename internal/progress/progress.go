// Package progress предоставляет прогресс-бар с ETA для отображения прогресса конвертации.
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Bar представляет прогресс-бар с поддержкой ETA.
type Bar struct {
	// bar - внутренний progressbar.
	bar *progressbar.ProgressBar

	// mu защищает доступ к bar.
	mu sync.Mutex

	// disabled - флаг отключения прогресс-бара.
	disabled bool

	// total - общее количество элементов.
	total int64

	// processed - обработанных элементов.
	processed int64

	// skipped - пропущенных элементов.
	skipped int64

	// failed - с ошибками.
	failed int64

	// startTime - время начала обработки.
	startTime time.Time

	// writer - куда выводить (по умолчанию os.Stderr).
	writer io.Writer
}

// Options содержит настройки для прогресс-бара.
type Options struct {
	// Total - общее количество элементов для обработки.
	Total int64

	// Description - описание задачи.
	Description string

	// Disabled - отключить прогресс-бар (только текстовый вывод).
	Disabled bool

	// Writer - куда выводить (по умолчанию os.Stderr).
	Writer io.Writer
}

// New создаёт новый прогресс-бар.
func New(opts Options) *Bar {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}

	b := &Bar{
		disabled:  opts.Disabled,
		total:     opts.Total,
		startTime: time.Now(),
		writer:    writer,
	}

	if !opts.Disabled && opts.Total > 0 {
		description := opts.Description
		if description == "" {
			description = "Обработка"
		}

		b.bar = progressbar.NewOptions64(
			opts.Total,
			progressbar.OptionSetWriter(writer),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("файл"),
			progressbar.OptionSetDescription(description),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]█[reset]",
				SaucerHead:    "[green]▓[reset]",
				SaucerPadding: "░",
				BarStart:      "[",
				BarEnd:        "]",
			}),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprintln(writer)
			}),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionFullWidth(),
		)
	}

	return b
}

// Increment увеличивает счётчик на 1 (обработан файл).
func (b *Bar) Increment() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.processed++

	if b.bar != nil {
		_ = b.bar.Add(1)
	}
}

// IncrementSkipped увеличивает счётчик пропущенных на 1.
func (b *Bar) IncrementSkipped() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.skipped++

	if b.bar != nil {
		_ = b.bar.Add(1)
	}
}

// IncrementFailed увеличивает счётчик ошибок на 1.
func (b *Bar) IncrementFailed() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failed++

	if b.bar != nil {
		_ = b.bar.Add(1)
	}
}

// SetTotal устанавливает общее количество элементов.
// Вызывается, когда становится известно точное количество файлов.
func (b *Bar) SetTotal(total int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.total = total

	if b.bar != nil {
		b.bar.ChangeMax64(total)
	}
}

// Finish завершает прогресс-бар.
func (b *Bar) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.bar != nil {
		_ = b.bar.Finish()
	}
}

// Clear очищает прогресс-бар (для вывода сообщений).
func (b *Bar) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.bar != nil {
		_ = b.bar.Clear()
	}
}

// Stats возвращает текущую статистику.
func (b *Bar) Stats() (processed, skipped, failed int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.processed, b.skipped, b.failed
}

// Duration возвращает время с начала обработки.
func (b *Bar) Duration() time.Duration {
	return time.Since(b.startTime)
}

// IsDisabled возвращает true, если прогресс-бар отключён.
func (b *Bar) IsDisabled() bool {
	return b.disabled
}

// WriteMessage выводит сообщение, временно скрывая прогресс-бар.
func (b *Bar) WriteMessage(format string, args ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.bar != nil {
		_ = b.bar.Clear()
	}

	fmt.Fprintf(b.writer, format, args...)

	if b.bar != nil {
		_ = b.bar.RenderBlank()
	}
}

/*
Возможные расширения:
- Добавить поддержку нескольких прогресс-баров (для разных этапов)
- Добавить историю скорости обработки
- Добавить поддержку pause/resume
- Добавить вывод в файл лога параллельно с прогресс-баром
*/

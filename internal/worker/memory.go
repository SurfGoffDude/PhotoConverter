// Package worker содержит пул воркеров для параллельной обработки.
package worker

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// MemoryLimiter ограничивает использование памяти при обработке файлов.
type MemoryLimiter struct {
	// maxMemoryBytes - максимальное использование памяти в байтах.
	maxMemoryBytes uint64

	// mu защищает доступ к текущему использованию.
	mu sync.Mutex

	// currentUsage - текущее зарезервированное использование памяти.
	currentUsage uint64

	// enabled - включено ли ограничение.
	enabled bool
}

// NewMemoryLimiter создаёт новый MemoryLimiter.
// maxMemoryMB - ограничение в мегабайтах (0 = без ограничения).
func NewMemoryLimiter(maxMemoryMB int) *MemoryLimiter {
	if maxMemoryMB <= 0 {
		return &MemoryLimiter{enabled: false}
	}

	return &MemoryLimiter{
		maxMemoryBytes: uint64(maxMemoryMB) * 1024 * 1024,
		enabled:        true,
	}
}

// Acquire пытается зарезервировать память для обработки файла.
// Блокирует выполнение, пока не будет достаточно памяти.
// Возвращает функцию для освобождения памяти.
func (ml *MemoryLimiter) Acquire(ctx context.Context, fileSize int64) (release func(), err error) {
	if !ml.enabled {
		return func() {}, nil
	}

	size := uint64(fileSize)
	// Оцениваем потребление памяти при обработке (примерно 3x от размера файла)
	estimatedUsage := size * 3

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ml.mu.Lock()
		// Проверяем текущее использование памяти системой
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		currentAlloc := memStats.Alloc

		// Если есть место, резервируем
		if ml.currentUsage+estimatedUsage <= ml.maxMemoryBytes &&
			currentAlloc+estimatedUsage <= ml.maxMemoryBytes {
			ml.currentUsage += estimatedUsage
			ml.mu.Unlock()

			return func() {
				ml.mu.Lock()
				ml.currentUsage -= estimatedUsage
				ml.mu.Unlock()
			}, nil
		}
		ml.mu.Unlock()

		// Ждём и пробуем снова
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Пробуем освободить память
			runtime.GC()
		}
	}
}

// IsEnabled возвращает true если ограничение включено.
func (ml *MemoryLimiter) IsEnabled() bool {
	return ml.enabled
}

// CurrentUsage возвращает текущее зарезервированное использование памяти.
func (ml *MemoryLimiter) CurrentUsage() uint64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return ml.currentUsage
}

// MaxMemory возвращает максимальное ограничение памяти.
func (ml *MemoryLimiter) MaxMemory() uint64 {
	return ml.maxMemoryBytes
}

/*
Возможные расширения:
- Добавить метрики использования памяти
- Добавить адаптивное ограничение на основе доступной памяти системы
- Добавить приоритеты для разных типов файлов
*/

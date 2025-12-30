// Package distributed реализует распределённую обработку через Redis.
package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/artemshloyda/photoconverter/internal/config"
	"github.com/artemshloyda/photoconverter/internal/scanner"
	"github.com/artemshloyda/photoconverter/internal/storage"
)

// Task представляет задачу на конвертацию.
type Task struct {
	// ID - уникальный идентификатор задачи.
	ID string `json:"id"`

	// FilePath - путь к исходному файлу.
	FilePath string `json:"file_path"`

	// RelPath - относительный путь от входной директории.
	RelPath string `json:"rel_path"`

	// Size - размер файла.
	Size int64 `json:"size"`

	// ModTime - время модификации.
	ModTime time.Time `json:"mod_time"`

	// Status - статус задачи (pending, processing, done, failed).
	Status string `json:"status"`

	// Error - ошибка (если есть).
	Error string `json:"error,omitempty"`

	// WorkerID - ID воркера, обрабатывающего задачу.
	WorkerID string `json:"worker_id,omitempty"`

	// StartedAt - время начала обработки.
	StartedAt time.Time `json:"started_at,omitempty"`

	// FinishedAt - время завершения обработки.
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// Queue управляет очередью задач.
// Это интерфейс для работы с Redis или in-memory очередью.
type Queue interface {
	// Push добавляет задачу в очередь.
	Push(ctx context.Context, task *Task) error

	// Pop извлекает задачу из очереди.
	Pop(ctx context.Context) (*Task, error)

	// Complete отмечает задачу как выполненную.
	Complete(ctx context.Context, taskID string) error

	// Fail отмечает задачу как неудачную.
	Fail(ctx context.Context, taskID string, err error) error

	// Stats возвращает статистику очереди.
	Stats(ctx context.Context) (*QueueStats, error)

	// Close закрывает соединение.
	Close() error
}

// QueueStats содержит статистику очереди.
type QueueStats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Done       int64 `json:"done"`
	Failed     int64 `json:"failed"`
}

// InMemoryQueue реализует очередь в памяти (для одной машины).
type InMemoryQueue struct {
	tasks   chan *Task
	done    map[string]bool
	failed  map[string]string
	pending int64
}

// NewInMemoryQueue создаёт новую in-memory очередь.
func NewInMemoryQueue(bufferSize int) *InMemoryQueue {
	return &InMemoryQueue{
		tasks:  make(chan *Task, bufferSize),
		done:   make(map[string]bool),
		failed: make(map[string]string),
	}
}

// Push добавляет задачу в очередь.
func (q *InMemoryQueue) Push(ctx context.Context, task *Task) error {
	select {
	case q.tasks <- task:
		q.pending++
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Pop извлекает задачу из очереди.
func (q *InMemoryQueue) Pop(ctx context.Context) (*Task, error) {
	select {
	case task := <-q.tasks:
		q.pending--
		return task, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Complete отмечает задачу как выполненную.
func (q *InMemoryQueue) Complete(ctx context.Context, taskID string) error {
	q.done[taskID] = true
	return nil
}

// Fail отмечает задачу как неудачную.
func (q *InMemoryQueue) Fail(ctx context.Context, taskID string, err error) error {
	q.failed[taskID] = err.Error()
	return nil
}

// Stats возвращает статистику очереди.
func (q *InMemoryQueue) Stats(ctx context.Context) (*QueueStats, error) {
	return &QueueStats{
		Pending: q.pending,
		Done:    int64(len(q.done)),
		Failed:  int64(len(q.failed)),
	}, nil
}

// Close закрывает очередь.
func (q *InMemoryQueue) Close() error {
	close(q.tasks)
	return nil
}

// TaskFromFile создаёт Task из scanner.File.
func TaskFromFile(file scanner.File) *Task {
	return &Task{
		ID:       fmt.Sprintf("%s-%d-%d", file.Path, file.Info.Size, file.Info.Mtime),
		FilePath: file.Path,
		RelPath:  file.RelPath,
		Size:     file.Info.Size,
		ModTime:  time.Unix(file.Info.Mtime, 0),
		Status:   "pending",
	}
}

// FileFromTask создаёт scanner.File из Task.
func FileFromTask(task *Task) scanner.File {
	return scanner.File{
		Path:    task.FilePath,
		RelPath: task.RelPath,
		Info: storage.FileInfo{
			Size:  task.Size,
			Mtime: task.ModTime.Unix(),
		},
	}
}

// Manager управляет распределённой обработкой.
type Manager struct {
	cfg   *config.Config
	queue Queue
	mode  string // "master" или "worker"
}

// NewManager создаёт новый Manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	var queue Queue

	if cfg.RedisURL != "" {
		// TODO: Реализовать RedisQueue
		// Пока используем in-memory
		queue = NewInMemoryQueue(10000)
	} else {
		queue = NewInMemoryQueue(10000)
	}

	return &Manager{
		cfg:   cfg,
		queue: queue,
		mode:  cfg.WorkerMode,
	}, nil
}

// IsMaster возвращает true если это master-узел.
func (m *Manager) IsMaster() bool {
	return m.mode == "master" || m.mode == ""
}

// IsWorker возвращает true если это worker-узел.
func (m *Manager) IsWorker() bool {
	return m.mode == "worker"
}

// Queue возвращает очередь задач.
func (m *Manager) Queue() Queue {
	return m.queue
}

// Close закрывает менеджер.
func (m *Manager) Close() error {
	return m.queue.Close()
}

// Serialize сериализует Task в JSON.
func (t *Task) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// DeserializeTask десериализует Task из JSON.
func DeserializeTask(data []byte) (*Task, error) {
	var task Task
	err := json.Unmarshal(data, &task)
	return &task, err
}

/*
Возможные расширения:
- Реализовать RedisQueue для настоящей распределённой обработки
- Добавить heartbeat для worker-ов
- Добавить автоматический retry неудачных задач
- Добавить балансировку нагрузки
- Добавить мониторинг и метрики
*/

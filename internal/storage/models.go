// Package storage содержит модели и логику работы с SQLite базой данных.
package storage

import "time"

// JobStatus определяет статус задачи конвертации.
type JobStatus string

const (
	// StatusInProgress - задача выполняется.
	StatusInProgress JobStatus = "in_progress"
	// StatusOK - задача успешно завершена.
	StatusOK JobStatus = "ok"
	// StatusFailed - задача завершилась с ошибкой.
	StatusFailed JobStatus = "failed"
)

// Job представляет задачу конвертации изображения.
type Job struct {
	// ID - уникальный идентификатор задачи.
	ID int64 `db:"id"`

	// SrcPath - абсолютный путь к исходному файлу.
	SrcPath string `db:"src_path"`

	// SrcSize - размер исходного файла в байтах.
	SrcSize int64 `db:"src_size"`

	// SrcMtime - время модификации исходного файла (unix timestamp).
	SrcMtime int64 `db:"src_mtime"`

	// OutFormat - выходной формат (webp, jpg, png, etc.).
	OutFormat string `db:"out_format"`

	// OutParams - JSON с параметрами выхода.
	OutParams string `db:"out_params"`

	// OutParamsHash - sha256 хэш параметров выхода.
	OutParamsHash string `db:"out_params_hash"`

	// ContentSHA256 - sha256 хэш содержимого исходного файла (nullable).
	ContentSHA256 *string `db:"content_sha256"`

	// DstPath - путь к выходному файлу.
	DstPath *string `db:"dst_path"`

	// Status - статус задачи.
	Status JobStatus `db:"status"`

	// Error - сообщение об ошибке (если есть).
	Error *string `db:"error"`

	// StartedAt - время начала обработки.
	StartedAt *time.Time `db:"started_at"`

	// FinishedAt - время завершения обработки.
	FinishedAt *time.Time `db:"finished_at"`
}

// FileInfo содержит информацию о файле для проверки.
type FileInfo struct {
	// Path - абсолютный путь к файлу.
	Path string

	// Size - размер файла в байтах.
	Size int64

	// Mtime - время модификации (unix timestamp).
	Mtime int64

	// ContentSHA256 - sha256 хэш содержимого (опционально).
	ContentSHA256 string
}

// JobResult содержит результат обработки задачи.
type JobResult struct {
	// JobID - ID задачи.
	JobID int64

	// Success - успешно ли завершена задача.
	Success bool

	// DstPath - путь к выходному файлу.
	DstPath string

	// Error - ошибка (если есть).
	Error error
}

// StartJobResult содержит результат попытки начать задачу.
type StartJobResult struct {
	// Started - была ли задача начата.
	Started bool

	// JobID - ID задачи (если начата).
	JobID int64

	// SkipReason - причина пропуска (если не начата).
	SkipReason string

	// ExistingDstPath - путь к существующему выходному файлу (для dedup).
	ExistingDstPath string
}

/*
Возможные расширения:
- Добавить поле для версии vips/параметров для инвалидации кэша
- Добавить поле для размера выходного файла (для статистики)
- Добавить поддержку тегов/категорий для группировки
*/

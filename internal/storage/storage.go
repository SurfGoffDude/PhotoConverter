// Package storage содержит логику работы с SQLite базой данных.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage предоставляет методы для работы с базой данных jobs.
type Storage struct {
	db *sql.DB
}

// New создаёт новое подключение к SQLite и выполняет миграции.
func New(dbPath string) (*Storage, error) {
	// Создаём директорию для БД, если не существует
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию для БД: %w", err)
	}

	// Открываем/создаём БД с параметрами для concurrent доступа
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(1) // SQLite не поддерживает concurrent writes
	db.SetMaxIdleConns(1)

	s := &Storage{db: db}

	// Выполняем миграции
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("не удалось выполнить миграции: %w", err)
	}

	return s, nil
}

// migrate выполняет все SQL-миграции.
func (s *Storage) migrate() error {
	for i, m := range GetMigrations() {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("миграция %d: %w", i+1, err)
		}
	}
	return nil
}

// Close закрывает подключение к БД.
func (s *Storage) Close() error {
	return s.db.Close()
}

// TryStartJob пытается начать обработку файла.
// Возвращает StartJobResult с информацией о том, была ли задача начата.
func (s *Storage) TryStartJob(info FileInfo, outFormat, outParams, outParamsHash string, dedupMode bool) (*StartJobResult, error) {
	now := time.Now().Unix()

	// Пытаемся вставить новую задачу
	query := `
		INSERT INTO jobs (src_path, src_size, src_mtime, out_format, out_params, out_params_hash, 
		                  content_sha256, status, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var contentSHA256 *string
	if dedupMode && info.ContentSHA256 != "" {
		contentSHA256 = &info.ContentSHA256
	}

	result, err := s.db.Exec(query,
		info.Path, info.Size, info.Mtime, outFormat, outParams, outParamsHash,
		contentSHA256, StatusInProgress, now,
	)

	if err != nil {
		// Проверяем, не конфликт ли уникального индекса
		if isUniqueConstraintError(err) {
			// Файл уже обработан или обрабатывается
			return s.checkExistingJob(info, outFormat, outParamsHash, dedupMode)
		}
		return nil, fmt.Errorf("не удалось создать задачу: %w", err)
	}

	jobID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить ID задачи: %w", err)
	}

	return &StartJobResult{
		Started: true,
		JobID:   jobID,
	}, nil
}

// checkExistingJob проверяет существующую задачу и возвращает причину пропуска.
func (s *Storage) checkExistingJob(info FileInfo, outFormat, outParamsHash string, dedupMode bool) (*StartJobResult, error) {
	// Сначала проверяем по source path
	var job Job
	query := `
		SELECT id, status, dst_path, error FROM jobs 
		WHERE src_path = ? AND src_size = ? AND src_mtime = ? 
		  AND out_format = ? AND out_params_hash = ?
		LIMIT 1
	`
	err := s.db.QueryRow(query, info.Path, info.Size, info.Mtime, outFormat, outParamsHash).
		Scan(&job.ID, &job.Status, &job.DstPath, &job.Error)

	if err == nil {
		switch job.Status {
		case StatusOK:
			dstPath := ""
			if job.DstPath != nil {
				dstPath = *job.DstPath
			}
			return &StartJobResult{
				Started:         false,
				SkipReason:      "уже успешно обработан",
				ExistingDstPath: dstPath,
			}, nil
		case StatusInProgress:
			return &StartJobResult{
				Started:    false,
				SkipReason: "уже обрабатывается",
			}, nil
		case StatusFailed:
			// Если failed - пробуем повторить, удаляя старую запись
			if _, err := s.db.Exec("DELETE FROM jobs WHERE id = ?", job.ID); err != nil {
				return nil, fmt.Errorf("не удалось удалить failed задачу: %w", err)
			}
			// Повторяем вставку
			return s.TryStartJob(info, outFormat, "", outParamsHash, dedupMode)
		}
	}

	// Если режим dedup, проверяем по content_sha256
	if dedupMode && info.ContentSHA256 != "" {
		query = `
			SELECT dst_path FROM jobs 
			WHERE content_sha256 = ? AND out_format = ? AND out_params_hash = ? AND status = 'ok'
			LIMIT 1
		`
		var dstPath *string
		err := s.db.QueryRow(query, info.ContentSHA256, outFormat, outParamsHash).Scan(&dstPath)
		if err == nil && dstPath != nil {
			return &StartJobResult{
				Started:         false,
				SkipReason:      "дубликат по содержимому",
				ExistingDstPath: *dstPath,
			}, nil
		}
	}

	return &StartJobResult{
		Started:    false,
		SkipReason: "неизвестная причина",
	}, nil
}

// FinalizeJobOK помечает задачу как успешно завершённую.
func (s *Storage) FinalizeJobOK(jobID int64, dstPath string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(
		"UPDATE jobs SET status = ?, dst_path = ?, finished_at = ? WHERE id = ?",
		StatusOK, dstPath, now, jobID,
	)
	if err != nil {
		return fmt.Errorf("не удалось обновить статус задачи: %w", err)
	}
	return nil
}

// FinalizeJobFailed помечает задачу как завершённую с ошибкой.
func (s *Storage) FinalizeJobFailed(jobID int64, errMsg string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(
		"UPDATE jobs SET status = ?, error = ?, finished_at = ? WHERE id = ?",
		StatusFailed, errMsg, now, jobID,
	)
	if err != nil {
		return fmt.Errorf("не удалось обновить статус задачи: %w", err)
	}
	return nil
}

// UpdateContentSHA256 обновляет sha256 хэш содержимого для задачи.
func (s *Storage) UpdateContentSHA256(jobID int64, sha256 string) error {
	_, err := s.db.Exec(
		"UPDATE jobs SET content_sha256 = ? WHERE id = ?",
		sha256, jobID,
	)
	if err != nil {
		return fmt.Errorf("не удалось обновить sha256: %w", err)
	}
	return nil
}

// GetStats возвращает статистику по задачам.
func (s *Storage) GetStats() (total, ok, failed, inProgress int64, err error) {
	err = s.db.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&total)
	if err != nil {
		return
	}
	_ = s.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = ?", StatusOK).Scan(&ok)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = ?", StatusFailed).Scan(&failed)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = ?", StatusInProgress).Scan(&inProgress)
	return
}

// CleanupInProgress сбрасывает задачи со статусом in_progress в failed.
// Вызывается при старте для очистки после аварийного завершения.
func (s *Storage) CleanupInProgress() (int64, error) {
	result, err := s.db.Exec(
		"UPDATE jobs SET status = ?, error = ? WHERE status = ?",
		StatusFailed, "прервано при предыдущем запуске", StatusInProgress,
	)
	if err != nil {
		return 0, fmt.Errorf("не удалось очистить in_progress: %w", err)
	}
	return result.RowsAffected()
}

// isUniqueConstraintError проверяет, является ли ошибка нарушением уникальности.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite возвращает "UNIQUE constraint failed" при конфликте
	return !errors.Is(err, sql.ErrNoRows) &&
		(contains(err.Error(), "UNIQUE constraint failed") ||
			contains(err.Error(), "constraint failed"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

/*
Возможные расширения:
- Добавить метод для экспорта статистики в JSON
- Добавить метод для очистки старых записей
- Добавить поддержку транзакций для batch-операций
*/

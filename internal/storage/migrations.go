// Package storage содержит миграции SQLite базы данных.
package storage

// migrations содержит SQL-миграции в порядке выполнения.
var migrations = []string{
	// Миграция 1: Создание таблицы jobs
	`CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		src_path TEXT NOT NULL,
		src_size INTEGER NOT NULL,
		src_mtime INTEGER NOT NULL,
		out_format TEXT NOT NULL,
		out_params TEXT NOT NULL,
		out_params_hash TEXT NOT NULL,
		content_sha256 TEXT,
		dst_path TEXT,
		status TEXT NOT NULL,
		error TEXT,
		started_at INTEGER,
		finished_at INTEGER
	);`,

	// Миграция 2: Уникальный индекс для идемпотентности по источнику
	// Гарантирует, что один и тот же файл (path+size+mtime) с теми же параметрами
	// не будет обработан дважды.
	`CREATE UNIQUE INDEX IF NOT EXISTS ux_jobs_src
	ON jobs (src_path, src_size, src_mtime, out_format, out_params_hash);`,

	// Миграция 3: Уникальный индекс для дедупликации по содержимому
	// Гарантирует, что файлы с одинаковым содержимым (sha256) и параметрами
	// не создадут дублирующиеся выходные файлы.
	`CREATE UNIQUE INDEX IF NOT EXISTS ux_jobs_dedup
	ON jobs (content_sha256, out_format, out_params_hash)
	WHERE content_sha256 IS NOT NULL AND status='ok';`,

	// Миграция 4: Индекс для быстрого поиска по статусу
	`CREATE INDEX IF NOT EXISTS ix_jobs_status ON jobs (status);`,

	// Миграция 5: Таблица метаданных для версионирования схемы
	`CREATE TABLE IF NOT EXISTS schema_info (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,

	// Миграция 6: Запись версии схемы
	`INSERT OR REPLACE INTO schema_info (key, value) VALUES ('version', '1');`,
}

// GetMigrations возвращает список SQL-миграций.
func GetMigrations() []string {
	return migrations
}

/*
Возможные расширения:
- Добавить таблицу для хранения статистики (общее время, количество файлов)
- Добавить таблицу для хранения ошибок отдельно
- Добавить поддержку отката миграций (down migrations)
*/

// Package cli содержит CLI интерфейс приложения.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/artemshloyda/photoconverter/internal/config"
	"github.com/artemshloyda/photoconverter/internal/converter"
	"github.com/artemshloyda/photoconverter/internal/scanner"
	"github.com/artemshloyda/photoconverter/internal/storage"
	"github.com/artemshloyda/photoconverter/internal/vipsfinder"
	"github.com/artemshloyda/photoconverter/internal/worker"
)

var (
	// Version будет установлена при сборке.
	Version = "dev"

	// BuildTime будет установлена при сборке.
	BuildTime = "unknown"
)

// cfg содержит глобальную конфигурацию.
var cfg = config.DefaultConfig()

// NewRootCmd создаёт корневую команду CLI.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "photoconverter",
		Short: "Утилита для массовой конвертации изображений",
		Long: `PhotoConverter - мультиплатформенная CLI утилита для массовой конвертации изображений.

Использует libvips для быстрой и качественной конвертации.
Поддерживает идемпотентность: повторный запуск не обрабатывает уже конвертированные файлы.

Примеры:
  # Конвертировать все JPEG/PNG в WebP
  photoconverter --in ./photos --out ./converted --out-format webp

  # Конвертировать только HEIC в JPEG с качеством 85
  photoconverter --in ./photos --out ./converted --in-ext heic --out-format jpg --quality 85

  # Режим дедупликации (одинаковые файлы не дублируются)
  photoconverter --in ./photos --out ./converted --mode dedup

  # Dry run (симуляция без реальной конвертации)
  photoconverter --in ./photos --out ./converted --dry-run`,
		RunE: runConvert,
	}

	// Флаги
	flags := rootCmd.Flags()

	// Входные параметры
	flags.StringVar(&cfg.InputDir, "in", "", "Директория с исходными изображениями (обязательно)")
	flags.StringVar(&cfg.OutputDir, "out", "", "Директория для сохранения результатов (обязательно)")
	flags.StringSliceVar(&cfg.InputExtensions, "in-ext", cfg.InputExtensions,
		"Расширения входных файлов через запятую (например: jpg,png,heic)")

	// Выходные параметры
	outFormat := flags.String("out-format", string(cfg.OutputFormat),
		"Выходной формат: webp, jpg, png, avif, tiff, heic")
	flags.IntVar(&cfg.Quality, "quality", cfg.Quality, "Качество для lossy форматов (1-100)")
	flags.BoolVar(&cfg.StripMetadata, "strip", cfg.StripMetadata, "Удалить метаданные из изображений")

	// Режим работы
	mode := flags.String("mode", string(cfg.Mode), "Режим: skip (по умолчанию) или dedup")
	flags.BoolVar(&cfg.KeepTree, "keep-tree", cfg.KeepTree, "Сохранять структуру директорий")
	flags.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Симуляция без реальной конвертации")

	// Производительность
	flags.IntVar(&cfg.Workers, "workers", cfg.Workers, "Количество параллельных воркеров")

	// Пути
	flags.StringVar(&cfg.DBPath, "db", cfg.DBPath, "Путь к SQLite базе данных")
	flags.StringVar(&cfg.VipsPath, "vips-path", cfg.VipsPath, "Путь к бинарнику vips")

	// Вывод
	flags.BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Подробный вывод")

	// Обязательные флаги
	_ = rootCmd.MarkFlagRequired("in")
	_ = rootCmd.MarkFlagRequired("out")

	// Парсинг enum-флагов
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		cfg.OutputFormat = config.OutputFormat(*outFormat)
		cfg.Mode = config.Mode(*mode)
		return nil
	}

	// Подкоманды
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newStatsCmd())

	return rootCmd
}

// runConvert выполняет основную логику конвертации.
func runConvert(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("ошибка конфигурации: %w", err)
	}

	// Создаём контекст с обработкой сигналов
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⚠️  Получен сигнал завершения, останавливаем...")
		cancel()
	}()

	// Ищем vips
	finder := vipsfinder.NewFinder(cfg.VipsPath)
	vipsInfo, err := finder.Find()
	if err != nil {
		return err
	}
	fmt.Printf("📦 Найден vips: %s (версия %s)\n", vipsInfo.Path, vipsInfo.Version)

	// Инициализируем хранилище
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("не удалось инициализировать БД: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Очищаем прерванные задачи
	cleaned, err := store.CleanupInProgress()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Не удалось очистить in_progress: %v\n", err)
	} else if cleaned > 0 {
		fmt.Printf("🧹 Очищено %d прерванных задач\n", cleaned)
	}

	// Создаём конвертер
	conv := converter.New(vipsInfo.Path, cfg)
	if err := conv.CheckVipsHealth(); err != nil {
		return err
	}

	// Создаём сканер
	scan := scanner.New(cfg)

	// Считаем файлы для отображения прогресса
	if cfg.Verbose {
		count, _ := scan.CountFiles()
		fmt.Printf("📁 Найдено файлов для обработки: %d\n", count)
	}

	// Запускаем сканирование
	files, errChan := scan.Scan(ctx)

	// Создаём пул воркеров
	pool := worker.New(cfg, store, conv)

	// Выводим параметры
	fmt.Printf("🚀 Запуск конвертации:\n")
	fmt.Printf("   Вход: %s\n", cfg.InputDir)
	fmt.Printf("   Выход: %s\n", cfg.OutputDir)
	fmt.Printf("   Формат: %s (качество: %d)\n", cfg.OutputFormat, cfg.Quality)
	fmt.Printf("   Режим: %s\n", cfg.Mode)
	fmt.Printf("   Воркеров: %d\n", cfg.Workers)
	if cfg.DryRun {
		fmt.Println("   ⚠️  Dry-run режим (без реальной конвертации)")
	}
	fmt.Println()

	// Запускаем обработку
	stats := pool.Process(ctx, files, errChan)

	// Выводим результаты
	duration := time.Since(startTime)
	fmt.Println()
	fmt.Printf("📊 Результаты:\n")
	fmt.Printf("   Обработано: %d\n", stats.Processed)
	fmt.Printf("   Пропущено: %d\n", stats.Skipped)
	fmt.Printf("   Ошибок: %d\n", stats.Failed)
	fmt.Printf("   Время: %s\n", duration.Round(time.Millisecond))

	if stats.Failed > 0 {
		return fmt.Errorf("завершено с %d ошибками", stats.Failed)
	}

	return nil
}

// newVersionCmd создаёт команду version.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("photoconverter %s (built %s)\n", Version, BuildTime)
		},
	}
}

// newStatsCmd создаёт команду stats.
func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Показать статистику из базы данных",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, _ := cmd.Flags().GetString("db")
			if dbPath == "" {
				return fmt.Errorf("укажите путь к БД через --db")
			}

			store, err := storage.New(dbPath)
			if err != nil {
				return fmt.Errorf("не удалось открыть БД: %w", err)
			}
			defer func() { _ = store.Close() }()

			total, ok, failed, inProgress, err := store.GetStats()
			if err != nil {
				return fmt.Errorf("не удалось получить статистику: %w", err)
			}

			fmt.Printf("📊 Статистика базы данных:\n")
			fmt.Printf("   Всего записей: %d\n", total)
			fmt.Printf("   Успешно: %d\n", ok)
			fmt.Printf("   Ошибок: %d\n", failed)
			fmt.Printf("   В процессе: %d\n", inProgress)

			return nil
		},
	}

	cmd.Flags().String("db", "", "Путь к SQLite базе данных")
	_ = cmd.MarkFlagRequired("db")

	return cmd
}

// Execute запускает CLI.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		// Не выводим ошибку, cobra уже вывела
		os.Exit(1)
	}
}


/*
Возможные расширения:
- Добавить команду clean для очистки БД
- Добавить команду retry для повторной обработки failed
- Добавить команду export для экспорта статистики в JSON
- Добавить интерактивный режим с progress bar
*/

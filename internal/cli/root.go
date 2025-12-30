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
	"github.com/artemshloyda/photoconverter/internal/progress"
	"github.com/artemshloyda/photoconverter/internal/scanner"
	"github.com/artemshloyda/photoconverter/internal/storage"
	"github.com/artemshloyda/photoconverter/internal/vipsfinder"
	"github.com/artemshloyda/photoconverter/internal/watcher"
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

// configPath содержит путь к файлу конфигурации.
var configPath string

// saveConfigPath содержит путь для сохранения конфигурации.
var saveConfigPath string

// savePresetName содержит имя для сохранения пресета.
var savePresetName string

// loadPresetName содержит имя пресета для загрузки.
var loadPresetName string

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
		"Выходной формат: webp, jpg, png, avif, tiff, heic, jxl")
	flags.IntVar(&cfg.Quality, "quality", cfg.Quality, "Качество для lossy форматов (1-100)")
	flags.BoolVar(&cfg.StripMetadata, "strip", cfg.StripMetadata, "Удалить метаданные из изображений")

	// Resize параметры
	flags.IntVar(&cfg.MaxWidth, "max-width", cfg.MaxWidth, "Максимальная ширина изображения (0 = без ограничения)")
	flags.IntVar(&cfg.MaxHeight, "max-height", cfg.MaxHeight, "Максимальная высота изображения (0 = без ограничения)")

	// Профиль качества
	preset := flags.String("preset", "", "Профиль качества: web, print, archive, thumbnail")

	// Режим работы
	mode := flags.String("mode", string(cfg.Mode), "Режим: skip (по умолчанию) или dedup")
	flags.BoolVar(&cfg.KeepTree, "keep-tree", cfg.KeepTree, "Сохранять структуру директорий")
	flags.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Симуляция без реальной конвертации")
	flags.BoolVar(&cfg.Watch, "watch", cfg.Watch, "Режим слежения за директорией")

	// Производительность
	flags.IntVar(&cfg.Workers, "workers", cfg.Workers, "Количество параллельных воркеров")

	// Пути
	flags.StringVar(&cfg.DBPath, "db", cfg.DBPath, "Путь к SQLite базе данных")
	flags.StringVar(&cfg.VipsPath, "vips-path", cfg.VipsPath, "Путь к бинарнику vips")

	// Вывод
	flags.BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Подробный вывод")
	flags.BoolVar(&cfg.NoProgress, "no-progress", cfg.NoProgress, "Отключить прогресс-бар")

	// Конфигурационный файл
	flags.StringVar(&configPath, "config", "", "Путь к файлу конфигурации (YAML)")
	flags.StringVar(&saveConfigPath, "save-config", "", "Сохранить текущие настройки в YAML файл и выйти")

	// Именованные пресеты
	flags.StringVar(&savePresetName, "save-preset", "", "Сохранить текущие настройки как именованный пресет")
	flags.StringVar(&loadPresetName, "load-preset", "", "Загрузить именованный пресет")

	// Флаги --in и --out НЕ обязательны, если есть конфиг файл
	// Валидация происходит в PreRunE после загрузки конфига

	// Парсинг конфигурации и enum-флагов
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Сохраняем значения CLI флагов ДО загрузки конфига
		// (Cobra уже применила их к cfg)
		cliInputDir := cfg.InputDir
		cliOutputDir := cfg.OutputDir
		cliInputExtensions := cfg.InputExtensions
		cliQuality := cfg.Quality
		cliStripMetadata := cfg.StripMetadata
		cliKeepTree := cfg.KeepTree
		cliWorkers := cfg.Workers
		cliDryRun := cfg.DryRun
		cliVerbose := cfg.Verbose
		cliNoProgress := cfg.NoProgress
		cliDBPath := cfg.DBPath
		cliVipsPath := cfg.VipsPath
		cliMaxWidth := cfg.MaxWidth
		cliMaxHeight := cfg.MaxHeight
		cliWatch := cfg.Watch

		// Загружаем именованный пресет (если указан)
		if loadPresetName != "" {
			fc, loadedPath, err := config.LoadPreset(loadPresetName)
			if err != nil {
				return err
			}
			fc.ApplyToConfig(cfg)
			if cfg.Verbose {
				fmt.Printf("📦 Загружен пресет '%s': %s\n", loadPresetName, loadedPath)
			}
		}

		// Загружаем конфигурацию из файла (если есть)
		fc, loadedPath, err := config.FindAndLoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
		}
		if fc != nil {
			// Применяем настройки из файла
			fc.ApplyToConfig(cfg)
			if cfg.Verbose {
				fmt.Printf("📄 Загружен конфиг: %s\n", loadedPath)
			}
		}

		// Применяем пресет (если указан) - он задаёт базовые настройки
		if cmd.Flags().Changed("preset") && *preset != "" {
			if !cfg.ApplyPreset(*preset) {
				return fmt.Errorf("неизвестный пресет: %s (доступны: %v)", *preset, config.ValidPresets())
			}
			cfg.Preset = *preset
		} else if cfg.Preset != "" {
			// Пресет из конфига
			if !cfg.ApplyPreset(cfg.Preset) {
				return fmt.Errorf("неизвестный пресет в конфиге: %s", cfg.Preset)
			}
		}

		// CLI флаги имеют приоритет над конфиг файлом
		// Восстанавливаем значения, если флаги были явно указаны
		// (проверяем что значение отличается от дефолтного)
		if cliInputDir != "" {
			cfg.InputDir = cliInputDir
		}
		if cliOutputDir != "" {
			cfg.OutputDir = cliOutputDir
		}
		if len(cliInputExtensions) > 0 && cmd.Flags().Changed("in-ext") {
			cfg.InputExtensions = cliInputExtensions
		}
		if cmd.Flags().Changed("quality") {
			cfg.Quality = cliQuality
		}
		if cmd.Flags().Changed("strip") {
			cfg.StripMetadata = cliStripMetadata
		}
		if cmd.Flags().Changed("keep-tree") {
			cfg.KeepTree = cliKeepTree
		}
		if cmd.Flags().Changed("workers") {
			cfg.Workers = cliWorkers
		}
		if cmd.Flags().Changed("dry-run") {
			cfg.DryRun = cliDryRun
		}
		if cmd.Flags().Changed("verbose") {
			cfg.Verbose = cliVerbose
		}
		if cmd.Flags().Changed("no-progress") {
			cfg.NoProgress = cliNoProgress
		}
		if cliDBPath != "" && cmd.Flags().Changed("db") {
			cfg.DBPath = cliDBPath
		}
		if cliVipsPath != "" && cmd.Flags().Changed("vips-path") {
			cfg.VipsPath = cliVipsPath
		}
		if cmd.Flags().Changed("max-width") {
			cfg.MaxWidth = cliMaxWidth
		}
		if cmd.Flags().Changed("max-height") {
			cfg.MaxHeight = cliMaxHeight
		}
		if cmd.Flags().Changed("watch") {
			cfg.Watch = cliWatch
		}

		// Обработка enum-флагов
		if cmd.Flags().Changed("out-format") {
			cfg.OutputFormat = config.OutputFormat(*outFormat)
		} else if fc != nil && fc.Output != nil && fc.Output.Format != "" {
			// Уже применено в ApplyToConfig
		} else if cfg.Preset == "" {
			cfg.OutputFormat = config.OutputFormat(*outFormat)
		}

		if cmd.Flags().Changed("mode") {
			cfg.Mode = config.Mode(*mode)
		} else if fc != nil && fc.Processing != nil && fc.Processing.Mode != "" {
			// Уже применено в ApplyToConfig
		} else {
			cfg.Mode = config.Mode(*mode)
		}

		// Проверяем обязательные поля после загрузки конфига
		// (--save-config не требует --in/--out заполненными)
		if saveConfigPath == "" {
			if cfg.InputDir == "" {
				return fmt.Errorf("входная директория не указана (--in или в конфиг файле)")
			}
			if cfg.OutputDir == "" {
				return fmt.Errorf("выходная директория не указана (--out или в конфиг файле)")
			}
		}

		return nil
	}

	// Подкоманды
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newStatsCmd())
	rootCmd.AddCommand(newPresetsCmd())

	return rootCmd
}

// runConvert выполняет основную логику конвертации.
func runConvert(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Сохранение конфигурации если указан флаг --save-config
	// (выполняется до валидации, т.к. не требует полной конфигурации)
	if saveConfigPath != "" {
		savedPath, err := config.SaveConfig(cfg, saveConfigPath)
		if err != nil {
			return fmt.Errorf("ошибка сохранения конфигурации: %w", err)
		}
		fmt.Printf("💾 Конфигурация сохранена в: %s\n", savedPath)
		return nil
	}

	// Сохранение именованного пресета если указан флаг --save-preset
	// (выполняется до валидации, т.к. не требует полной конфигурации)
	if savePresetName != "" {
		savedPath, err := config.SavePreset(savePresetName, cfg)
		if err != nil {
			return fmt.Errorf("ошибка сохранения пресета: %w", err)
		}
		fmt.Printf("📦 Пресет '%s' сохранён в: %s\n", savePresetName, savedPath)
		return nil
	}

	// Валидация конфигурации (только для реальной конвертации)
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

	// Создаём пул воркеров
	pool := worker.New(cfg, store, conv)

	// Выводим параметры
	fmt.Printf("🚀 Запуск конвертации:\n")
	fmt.Printf("   Вход: %s\n", cfg.InputDir)
	fmt.Printf("   Выход: %s\n", cfg.OutputDir)
	fmt.Printf("   Формат: %s (качество: %d)\n", cfg.OutputFormat, cfg.Quality)
	if cfg.MaxWidth > 0 || cfg.MaxHeight > 0 {
		fmt.Printf("   Resize: max %dx%d\n", cfg.MaxWidth, cfg.MaxHeight)
	}
	if cfg.Preset != "" {
		fmt.Printf("   Пресет: %s\n", cfg.Preset)
	}
	fmt.Printf("   Режим: %s\n", cfg.Mode)
	fmt.Printf("   Воркеров: %d\n", cfg.Workers)
	if cfg.DryRun {
		fmt.Println("   ⚠️  Dry-run режим (без реальной конвертации)")
	}
	if cfg.Watch {
		fmt.Println("   👁️  Watch режим (слежение за директорией)")
	}
	fmt.Println()

	// Watch mode или обычный режим
	if cfg.Watch {
		return runWatchMode(ctx, pool)
	}

	return runNormalMode(ctx, pool, startTime)
}

// runNormalMode выполняет обычную конвертацию.
func runNormalMode(ctx context.Context, pool *worker.Pool, startTime time.Time) error {
	// Создаём сканер
	scan := scanner.New(cfg)

	// Считаем файлы для отображения прогресса
	fileCount, _ := scan.CountFiles()
	if cfg.Verbose {
		fmt.Printf("📁 Найдено файлов для обработки: %d\n", fileCount)
	}

	// Запускаем сканирование
	files, errChan := scan.Scan(ctx)

	// Создаём прогресс-бар
	progressBar := progress.New(progress.Options{
		Total:       int64(fileCount),
		Description: "🔄 Конвертация",
		Disabled:    cfg.NoProgress || cfg.DryRun,
	})
	pool.SetProgressBar(progressBar)

	// Запускаем обработку
	stats := pool.Process(ctx, files, errChan)

	// Завершаем прогресс-бар
	progressBar.Finish()

	// Выводим результаты
	duration := time.Since(startTime)
	fmt.Println()
	fmt.Printf("📊 Результаты:\n")
	fmt.Printf("   Обработано: %d\n", stats.Processed)
	fmt.Printf("   Пропущено: %d\n", stats.Skipped)
	fmt.Printf("   Ошибок: %d\n", stats.Failed)
	fmt.Printf("   Время: %s\n", duration.Round(time.Millisecond))

	// Расширенная статистика размеров
	if stats.InputBytes > 0 {
		fmt.Printf("   Размер входных: %s\n", worker.FormatBytes(stats.InputBytes))
		fmt.Printf("   Размер выходных: %s\n", worker.FormatBytes(stats.OutputBytes))
		saved := stats.SavedBytes()
		if saved > 0 {
			fmt.Printf("   💾 Экономия: %s (%.1f%%)\n", worker.FormatBytes(saved), stats.SavedPercent())
		} else if saved < 0 {
			fmt.Printf("   ⚠️  Увеличение: %s (+%.1f%%)\n", worker.FormatBytes(-saved), -stats.SavedPercent())
		}
	}

	if stats.Failed > 0 {
		return fmt.Errorf("завершено с %d ошибками", stats.Failed)
	}

	return nil
}

// runWatchMode выполняет конвертацию в режиме слежения.
func runWatchMode(ctx context.Context, pool *worker.Pool) error {
	// Создаём watcher
	w, err := watcher.New(cfg)
	if err != nil {
		return fmt.Errorf("не удалось создать watcher: %w", err)
	}
	defer w.Close()

	// Запускаем слежение
	files, err := w.Watch(ctx)
	if err != nil {
		return fmt.Errorf("ошибка запуска watch: %w", err)
	}

	fmt.Println("👁️  Слежение запущено. Нажмите Ctrl+C для остановки.")

	// Прогресс-бар для watch mode (без общего счётчика)
	progressBar := progress.New(progress.Options{
		Total:       -1, // Бесконечный режим
		Description: "👁️ Watch",
		Disabled:    cfg.NoProgress,
	})
	pool.SetProgressBar(progressBar)

	// Канал для получения статистики
	statsChan := make(chan worker.Stats, 1)

	// Запускаем обработку в фоновой горутине
	go func() {
		stats := pool.Process(ctx, files, nil)
		statsChan <- stats
	}()

	// Ожидаем завершения контекста или обработки
	select {
	case <-ctx.Done():
		// Контекст отменён (Ctrl+C)
		fmt.Println("\n⏹️  Останавливаем слежение...")
	case stats := <-statsChan:
		// Обработка завершилась (не должно происходить в watch mode)
		progressBar.Finish()
		fmt.Println()
		fmt.Printf("📊 Результаты watch режима:\n")
		fmt.Printf("   Обработано: %d\n", stats.Processed)
		fmt.Printf("   Пропущено: %d\n", stats.Skipped)
		fmt.Printf("   Ошибок: %d\n", stats.Failed)
		return nil
	}

	// Ждём завершения обработки после отмены контекста
	stats := <-statsChan
	progressBar.Finish()

	fmt.Println()
	fmt.Printf("📊 Результаты watch режима:\n")
	fmt.Printf("   Обработано: %d\n", stats.Processed)
	fmt.Printf("   Пропущено: %d\n", stats.Skipped)
	fmt.Printf("   Ошибок: %d\n", stats.Failed)

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

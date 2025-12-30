// Package cli содержит CLI команды приложения.
package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/artemshloyda/photoconverter/internal/config"
)

// newPresetsCmd создаёт команду для управления пресетами.
func newPresetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "presets",
		Short: "Управление именованными пресетами конфигурации",
		Long: `Управление именованными пресетами конфигурации.

Пресеты хранятся в ~/.config/photoconverter/presets/ и позволяют
сохранять и загружать конфигурации для разных проектов.

Примеры:
  # Сохранить текущие настройки как пресет
  photoconverter --in ./photos --out ./web --preset web --save-preset my-project

  # Загрузить пресет и запустить конвертацию
  photoconverter --load-preset my-project

  # Список пресетов
  photoconverter presets list

  # Удалить пресет
  photoconverter presets delete my-project`,
	}

	cmd.AddCommand(newPresetsListCmd())
	cmd.AddCommand(newPresetsDeleteCmd())
	cmd.AddCommand(newPresetsShowCmd())

	return cmd
}

// newPresetsListCmd создаёт команду для списка пресетов.
func newPresetsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Показать список сохранённых пресетов",
		RunE: func(cmd *cobra.Command, args []string) error {
			presets, err := config.ListPresets()
			if err != nil {
				return fmt.Errorf("ошибка получения списка пресетов: %w", err)
			}

			if len(presets) == 0 {
				fmt.Println("Пресеты не найдены.")
				fmt.Println()
				fmt.Println("Сохраните пресет командой:")
				fmt.Println("  photoconverter --in ./photos --out ./web --save-preset my-project")
				return nil
			}

			fmt.Printf("📦 Сохранённые пресеты (%d):\n\n", len(presets))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ИМЯ\tФОРМАТ\tКАЧЕСТВО\tПУТЬ")
			fmt.Fprintln(w, "---\t------\t--------\t----")

			for _, p := range presets {
				format := "-"
				quality := "-"
				if p.Config != nil && p.Config.Output != nil {
					if p.Config.Output.Format != "" {
						format = p.Config.Output.Format
					}
					if p.Config.Output.Quality > 0 {
						quality = fmt.Sprintf("%d", p.Config.Output.Quality)
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, format, quality, p.Path)
			}
			w.Flush()

			return nil
		},
	}
}

// newPresetsDeleteCmd создаёт команду для удаления пресета.
func newPresetsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить пресет",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if !config.PresetExists(name) {
				return fmt.Errorf("пресет '%s' не найден", name)
			}

			if err := config.DeletePreset(name); err != nil {
				return fmt.Errorf("ошибка удаления пресета: %w", err)
			}

			fmt.Printf("✅ Пресет '%s' удалён\n", name)
			return nil
		},
	}
}

// newPresetsShowCmd создаёт команду для отображения пресета.
func newPresetsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Показать содержимое пресета",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			fc, path, err := config.LoadPreset(name)
			if err != nil {
				return err
			}

			fmt.Printf("📦 Пресет: %s\n", name)
			fmt.Printf("📁 Путь: %s\n\n", path)

			if fc.Input != nil {
				fmt.Println("Input:")
				if fc.Input.Dir != "" {
					fmt.Printf("  dir: %s\n", fc.Input.Dir)
				}
				if len(fc.Input.Extensions) > 0 {
					fmt.Printf("  extensions: %v\n", fc.Input.Extensions)
				}
			}

			if fc.Output != nil {
				fmt.Println("Output:")
				if fc.Output.Dir != "" {
					fmt.Printf("  dir: %s\n", fc.Output.Dir)
				}
				if fc.Output.Format != "" {
					fmt.Printf("  format: %s\n", fc.Output.Format)
				}
				if fc.Output.Quality > 0 {
					fmt.Printf("  quality: %d\n", fc.Output.Quality)
				}
				if fc.Output.MaxWidth > 0 {
					fmt.Printf("  max_width: %d\n", fc.Output.MaxWidth)
				}
				if fc.Output.MaxHeight > 0 {
					fmt.Printf("  max_height: %d\n", fc.Output.MaxHeight)
				}
			}

			if fc.Processing != nil {
				fmt.Println("Processing:")
				if fc.Processing.Workers > 0 {
					fmt.Printf("  workers: %d\n", fc.Processing.Workers)
				}
				if fc.Processing.Mode != "" {
					fmt.Printf("  mode: %s\n", fc.Processing.Mode)
				}
				if fc.Processing.Preset != "" {
					fmt.Printf("  preset: %s\n", fc.Processing.Preset)
				}
			}

			return nil
		},
	}
}

/*
Возможные расширения:
- Добавить команду 'presets export' для экспорта в файл
- Добавить команду 'presets import' для импорта из файла
- Добавить команду 'presets copy' для копирования пресета
*/

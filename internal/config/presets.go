// Package config содержит конфигурацию приложения.
package config

// Preset определяет профиль качества.
type Preset string

const (
	// PresetWeb - оптимизация для веба: webp, качество 75, max-width 1920, strip metadata.
	PresetWeb Preset = "web"
	// PresetPrint - высокое качество для печати: качество 95, без resize.
	PresetPrint Preset = "print"
	// PresetArchive - архивное качество: PNG, качество 100, без потерь.
	PresetArchive Preset = "archive"
	// PresetThumbnail - превью: webp, качество 60, max-width 300.
	PresetThumbnail Preset = "thumbnail"
)

// PresetConfig содержит настройки для пресета.
type PresetConfig struct {
	// Format - выходной формат.
	Format OutputFormat
	// Quality - качество (1-100).
	Quality int
	// MaxWidth - максимальная ширина (0 = без ограничения).
	MaxWidth int
	// MaxHeight - максимальная высота (0 = без ограничения).
	MaxHeight int
	// StripMetadata - удалять метаданные.
	StripMetadata bool
}

// Presets содержит все доступные пресеты.
var Presets = map[Preset]PresetConfig{
	PresetWeb: {
		Format:        FormatWebP,
		Quality:       75,
		MaxWidth:      1920,
		MaxHeight:     0,
		StripMetadata: true,
	},
	PresetPrint: {
		Format:        FormatJPEG,
		Quality:       95,
		MaxWidth:      0,
		MaxHeight:     0,
		StripMetadata: false,
	},
	PresetArchive: {
		Format:        FormatPNG,
		Quality:       100,
		MaxWidth:      0,
		MaxHeight:     0,
		StripMetadata: false,
	},
	PresetThumbnail: {
		Format:        FormatWebP,
		Quality:       60,
		MaxWidth:      300,
		MaxHeight:     300,
		StripMetadata: true,
	},
}

// ApplyPreset применяет пресет к конфигурации.
// Возвращает true, если пресет был применён.
func (c *Config) ApplyPreset(preset string) bool {
	p, ok := Presets[Preset(preset)]
	if !ok {
		return false
	}

	c.OutputFormat = p.Format
	c.Quality = p.Quality
	c.MaxWidth = p.MaxWidth
	c.MaxHeight = p.MaxHeight
	c.StripMetadata = p.StripMetadata

	return true
}

// ValidPresets возвращает список доступных пресетов.
func ValidPresets() []string {
	return []string{
		string(PresetWeb),
		string(PresetPrint),
		string(PresetArchive),
		string(PresetThumbnail),
	}
}

/*
Возможные расширения:
- Добавить пользовательские пресеты из конфигурационного файла
- Добавить пресет для социальных сетей (instagram, telegram)
- Добавить пресет для email (ограничение по размеру файла)
*/

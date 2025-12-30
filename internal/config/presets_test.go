package config

import (
	"testing"
)

func TestApplyPreset(t *testing.T) {
	tests := []struct {
		name       string
		preset     string
		wantOK     bool
		wantFormat OutputFormat
		wantQual   int
	}{
		{
			name:       "web preset",
			preset:     "web",
			wantOK:     true,
			wantFormat: FormatWebP,
			wantQual:   75,
		},
		{
			name:       "print preset",
			preset:     "print",
			wantOK:     true,
			wantFormat: FormatJPEG,
			wantQual:   95,
		},
		{
			name:       "archive preset",
			preset:     "archive",
			wantOK:     true,
			wantFormat: FormatPNG,
			wantQual:   100,
		},
		{
			name:       "thumbnail preset",
			preset:     "thumbnail",
			wantOK:     true,
			wantFormat: FormatWebP,
			wantQual:   60,
		},
		{
			name:   "unknown preset",
			preset: "unknown",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			ok := cfg.ApplyPreset(tt.preset)

			if ok != tt.wantOK {
				t.Errorf("ApplyPreset() = %v, want %v", ok, tt.wantOK)
			}

			if tt.wantOK {
				if cfg.OutputFormat != tt.wantFormat {
					t.Errorf("OutputFormat = %v, want %v", cfg.OutputFormat, tt.wantFormat)
				}
				if cfg.Quality != tt.wantQual {
					t.Errorf("Quality = %d, want %d", cfg.Quality, tt.wantQual)
				}
			}
		})
	}
}

func TestValidPresets(t *testing.T) {
	presets := ValidPresets()

	if len(presets) == 0 {
		t.Error("ValidPresets() returned empty slice")
	}

	expected := []string{"web", "print", "archive", "thumbnail"}
	if len(presets) != len(expected) {
		t.Errorf("ValidPresets() returned %d presets, want %d", len(presets), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, p := range presets {
			if p == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidPresets() missing %q", exp)
		}
	}
}

func TestPresetConfig(t *testing.T) {
	// Проверяем, что все пресеты имеют валидные значения
	for name, preset := range Presets {
		t.Run(string(name), func(t *testing.T) {
			if preset.Quality < 1 || preset.Quality > 100 {
				t.Errorf("Preset %s has invalid quality: %d", name, preset.Quality)
			}

			if preset.MaxWidth < 0 {
				t.Errorf("Preset %s has negative MaxWidth: %d", name, preset.MaxWidth)
			}

			if preset.MaxHeight < 0 {
				t.Errorf("Preset %s has negative MaxHeight: %d", name, preset.MaxHeight)
			}
		})
	}
}

func TestPresetWebSettings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ApplyPreset("web")

	if cfg.OutputFormat != FormatWebP {
		t.Errorf("Web preset format = %v, want webp", cfg.OutputFormat)
	}

	if cfg.MaxWidth != 1920 {
		t.Errorf("Web preset MaxWidth = %d, want 1920", cfg.MaxWidth)
	}

	if !cfg.StripMetadata {
		t.Error("Web preset should strip metadata")
	}
}

func TestPresetThumbnailSettings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ApplyPreset("thumbnail")

	if cfg.MaxWidth != 300 {
		t.Errorf("Thumbnail preset MaxWidth = %d, want 300", cfg.MaxWidth)
	}

	if cfg.MaxHeight != 300 {
		t.Errorf("Thumbnail preset MaxHeight = %d, want 300", cfg.MaxHeight)
	}
}

package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Проверяем значения по умолчанию
	if cfg.OutputFormat != FormatJPEG {
		t.Errorf("OutputFormat = %v, want %v", cfg.OutputFormat, FormatJPEG)
	}

	if cfg.Quality != 80 {
		t.Errorf("Quality = %d, want 80", cfg.Quality)
	}

	if cfg.Mode != ModeSkip {
		t.Errorf("Mode = %v, want %v", cfg.Mode, ModeSkip)
	}

	if cfg.Workers < 1 {
		t.Errorf("Workers = %d, want >= 1", cfg.Workers)
	}

	if !cfg.KeepTree {
		t.Error("KeepTree should be true by default")
	}

	if len(cfg.InputExtensions) == 0 {
		t.Error("InputExtensions should not be empty by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				InputDir:        "/input",
				OutputDir:       "/output",
				InputExtensions: []string{"jpg", "png"},
				OutputFormat:    FormatWebP,
				Quality:         85,
				Workers:         4,
				Mode:            ModeSkip,
			},
			wantErr: false,
		},
		{
			name: "missing input dir",
			cfg: &Config{
				OutputDir:    "/output",
				OutputFormat: FormatWebP,
				Quality:      85,
				Workers:      4,
			},
			wantErr: true,
		},
		{
			name: "missing output dir",
			cfg: &Config{
				InputDir:     "/input",
				OutputFormat: FormatWebP,
				Quality:      85,
				Workers:      4,
			},
			wantErr: true,
		},
		{
			name: "invalid quality low",
			cfg: &Config{
				InputDir:     "/input",
				OutputDir:    "/output",
				OutputFormat: FormatWebP,
				Quality:      0,
				Workers:      4,
			},
			wantErr: true,
		},
		{
			name: "invalid quality high",
			cfg: &Config{
				InputDir:     "/input",
				OutputDir:    "/output",
				OutputFormat: FormatWebP,
				Quality:      101,
				Workers:      4,
			},
			wantErr: true,
		},
		{
			name: "invalid workers",
			cfg: &Config{
				InputDir:     "/input",
				OutputDir:    "/output",
				OutputFormat: FormatWebP,
				Quality:      85,
				Workers:      0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_HasInputExtension(t *testing.T) {
	cfg := &Config{
		InputExtensions: []string{"jpg", "jpeg", "png"},
	}

	tests := []struct {
		ext  string
		want bool
	}{
		{"jpg", true},
		{"jpeg", true},
		{"png", true},
		{"JPG", true},  // case insensitive
		{"JPEG", true}, // case insensitive
		{"webp", false},
		{"gif", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := cfg.HasInputExtension(tt.ext); got != tt.want {
				t.Errorf("HasInputExtension(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestConfig_VipsOutputSuffix(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "webp with quality",
			cfg: &Config{
				OutputFormat: FormatWebP,
				Quality:      85,
			},
			want: "[Q=85]",
		},
		{
			name: "webp with quality and strip",
			cfg: &Config{
				OutputFormat:  FormatWebP,
				Quality:       85,
				StripMetadata: true,
			},
			want: "[Q=85,strip]",
		},
		{
			name: "png without quality",
			cfg: &Config{
				OutputFormat: FormatPNG,
				Quality:      85,
			},
			want: "",
		},
		{
			name: "png with strip",
			cfg: &Config{
				OutputFormat:  FormatPNG,
				Quality:       85,
				StripMetadata: true,
			},
			want: "[strip]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.VipsOutputSuffix(); got != tt.want {
				t.Errorf("VipsOutputSuffix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig_OutputParams(t *testing.T) {
	cfg := &Config{
		OutputFormat:  FormatWebP,
		Quality:       85,
		StripMetadata: true,
		MaxWidth:      1920,
		MaxHeight:     1080,
	}

	params := cfg.OutputParams()

	if params == "" {
		t.Error("OutputParams() returned empty string")
	}
}

func TestOutputFormat_String(t *testing.T) {
	tests := []struct {
		format OutputFormat
		want   string
	}{
		{FormatWebP, "webp"},
		{FormatJPEG, "jpg"},
		{FormatPNG, "png"},
		{FormatAVIF, "avif"},
		{FormatTIFF, "tiff"},
		{FormatHEIC, "heic"},
		{FormatJXL, "jxl"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := string(tt.format); got != tt.want {
				t.Errorf("OutputFormat string = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeSkip, "skip"},
		{ModeDedup, "dedup"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := string(tt.mode); got != tt.want {
				t.Errorf("Mode string = %q, want %q", got, tt.want)
			}
		})
	}
}

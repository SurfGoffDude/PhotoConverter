// Package vipsfinder отвечает за поиск бинарника vips в системе.
package vipsfinder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// VipsInfo содержит информацию о найденном vips.
type VipsInfo struct {
	// Path - абсолютный путь к бинарнику vips.
	Path string

	// Version - версия vips (например, "8.14.2").
	Version string
}

// Finder ищет бинарник vips.
type Finder struct {
	// CustomPath - пользовательский путь к vips (из флага --vips-path).
	CustomPath string

	// EnvVar - имя переменной окружения для пути к vips.
	EnvVar string
}

// NewFinder создаёт новый Finder.
func NewFinder(customPath string) *Finder {
	return &Finder{
		CustomPath: customPath,
		EnvVar:     "PHOTOCONVERTER_VIPS",
	}
}

// Find ищет vips в следующем порядке:
// 1. CustomPath (если задан)
// 2. Переменная окружения PHOTOCONVERTER_VIPS
// 3. PATH
// 4. Рядом с исполняемым файлом в ./bin/<os-arch>/vips
func (f *Finder) Find() (*VipsInfo, error) {
	var candidates []string

	// 1. Пользовательский путь
	if f.CustomPath != "" {
		candidates = append(candidates, f.CustomPath)
	}

	// 2. Переменная окружения
	if envPath := os.Getenv(f.EnvVar); envPath != "" {
		candidates = append(candidates, envPath)
	}

	// 3. PATH
	if pathVips, err := exec.LookPath("vips"); err == nil {
		candidates = append(candidates, pathVips)
	}

	// 4. Рядом с бинарником
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		platformDir := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

		// Проверяем несколько вариантов расположения
		localPaths := []string{
			filepath.Join(execDir, "bin", platformDir, vipsBinaryName()),
			filepath.Join(execDir, "bin", vipsBinaryName()),
			filepath.Join(execDir, vipsBinaryName()),
		}
		candidates = append(candidates, localPaths...)
	}

	// Проверяем каждого кандидата
	for _, path := range candidates {
		if info, err := f.checkVips(path); err == nil {
			return info, nil
		}
	}

	return nil, fmt.Errorf("vips не найден. Проверьте:\n"+
		"  1. Установлен ли vips в системе (apt install libvips-tools / brew install vips)\n"+
		"  2. Установлена ли переменная окружения %s\n"+
		"  3. Указан ли путь через флаг --vips-path\n"+
		"  4. Находится ли vips рядом с утилитой в ./bin/<os-arch>/", f.EnvVar)
}

// checkVips проверяет, является ли путь рабочим vips.
func (f *Finder) checkVips(path string) (*VipsInfo, error) {
	// Проверяем, существует ли файл
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("файл не найден: %w", err)
	}

	// Проверяем, что это исполняемый файл
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить абсолютный путь: %w", err)
	}

	// Пробуем получить версию
	cmd := exec.Command(absPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить vips --version: %w", err)
	}

	version := parseVersion(string(output))

	return &VipsInfo{
		Path:    absPath,
		Version: version,
	}, nil
}

// parseVersion извлекает версию из вывода "vips --version".
// Пример вывода: "vips-8.14.2"
func parseVersion(output string) string {
	output = strings.TrimSpace(output)

	// Формат: "vips-8.14.2" или "vips 8.14.2"
	if strings.HasPrefix(output, "vips-") {
		return strings.TrimPrefix(output, "vips-")
	}
	if strings.HasPrefix(output, "vips ") {
		return strings.TrimPrefix(output, "vips ")
	}

	// Возвращаем как есть
	return output
}

// vipsBinaryName возвращает имя бинарника vips для текущей ОС.
func vipsBinaryName() string {
	if runtime.GOOS == "windows" {
		return "vips.exe"
	}
	return "vips"
}

// GetSupportedFormats возвращает список поддерживаемых форматов vips.
func (v *VipsInfo) GetSupportedFormats() ([]string, error) {
	cmd := exec.Command(v.Path, "list", "classes")
	output, err := cmd.Output()
	if err != nil {
		// Fallback на стандартный список
		return []string{"jpeg", "png", "webp", "tiff", "heif", "avif"}, nil
	}

	var formats []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Ищем форматы для сохранения
		if strings.Contains(line, "save") {
			formats = append(formats, line)
		}
	}

	if len(formats) == 0 {
		return []string{"jpeg", "png", "webp", "tiff", "heif", "avif"}, nil
	}

	return formats, nil
}

/*
Возможные расширения:
- Кэширование результата поиска
- Проверка минимальной версии vips
- Автоматическое скачивание portable vips
*/

// Package config содержит конфигурацию приложения.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BatchPreset представляет именованный пресет конфигурации.
type BatchPreset struct {
	// Name - имя пресета.
	Name string
	// Path - путь к файлу пресета.
	Path string
	// Config - конфигурация пресета.
	Config *FileConfig
}

// GetPresetsDir возвращает директорию для хранения пресетов.
func GetPresetsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("не удалось получить домашнюю директорию: %w", err)
	}

	presetsDir := filepath.Join(homeDir, ".config", "photoconverter", "presets")
	return presetsDir, nil
}

// EnsurePresetsDir создаёт директорию для пресетов если она не существует.
func EnsurePresetsDir() (string, error) {
	presetsDir, err := GetPresetsDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(presetsDir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать директорию пресетов: %w", err)
	}

	return presetsDir, nil
}

// GetPresetPath возвращает путь к файлу пресета по имени.
func GetPresetPath(name string) (string, error) {
	presetsDir, err := GetPresetsDir()
	if err != nil {
		return "", err
	}

	// Очищаем имя от небезопасных символов
	safeName := sanitizePresetName(name)
	if safeName == "" {
		return "", fmt.Errorf("некорректное имя пресета: %s", name)
	}

	return filepath.Join(presetsDir, safeName+".yaml"), nil
}

// sanitizePresetName очищает имя пресета от небезопасных символов.
func sanitizePresetName(name string) string {
	// Разрешаем только буквы, цифры, дефисы и подчёркивания
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SavePreset сохраняет конфигурацию как именованный пресет.
func SavePreset(name string, cfg *Config) (string, error) {
	if _, err := EnsurePresetsDir(); err != nil {
		return "", err
	}

	presetPath, err := GetPresetPath(name)
	if err != nil {
		return "", err
	}

	fc := FromConfig(cfg)
	if err := fc.SaveToFile(presetPath); err != nil {
		return "", fmt.Errorf("не удалось сохранить пресет: %w", err)
	}

	return presetPath, nil
}

// LoadPreset загружает конфигурацию из именованного пресета.
func LoadPreset(name string) (*FileConfig, string, error) {
	presetPath, err := GetPresetPath(name)
	if err != nil {
		return nil, "", err
	}

	fc, err := LoadFromFile(presetPath)
	if err != nil {
		return nil, "", fmt.Errorf("не удалось загрузить пресет '%s': %w", name, err)
	}

	return fc, presetPath, nil
}

// ListPresets возвращает список всех сохранённых пресетов.
func ListPresets() ([]BatchPreset, error) {
	presetsDir, err := GetPresetsDir()
	if err != nil {
		return nil, err
	}

	// Проверяем существование директории
	if _, err := os.Stat(presetsDir); os.IsNotExist(err) {
		return []BatchPreset{}, nil
	}

	entries, err := os.ReadDir(presetsDir)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать директорию пресетов: %w", err)
	}

	var presets []BatchPreset
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		presetName := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		presetPath := filepath.Join(presetsDir, name)

		// Пробуем загрузить конфиг для проверки валидности
		fc, _ := LoadFromFile(presetPath)

		presets = append(presets, BatchPreset{
			Name:   presetName,
			Path:   presetPath,
			Config: fc,
		})
	}

	// Сортируем по имени
	sort.Slice(presets, func(i, j int) bool {
		return presets[i].Name < presets[j].Name
	})

	return presets, nil
}

// DeletePreset удаляет именованный пресет.
func DeletePreset(name string) error {
	presetPath, err := GetPresetPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(presetPath); os.IsNotExist(err) {
		return fmt.Errorf("пресет '%s' не найден", name)
	}

	if err := os.Remove(presetPath); err != nil {
		return fmt.Errorf("не удалось удалить пресет: %w", err)
	}

	return nil
}

// PresetExists проверяет существование пресета.
func PresetExists(name string) bool {
	presetPath, err := GetPresetPath(name)
	if err != nil {
		return false
	}

	_, err = os.Stat(presetPath)
	return err == nil
}

/*
Возможные расширения:
- Добавить описание к пресетам
- Добавить теги для группировки пресетов
- Добавить импорт/экспорт пресетов
- Добавить наследование пресетов (extends)
*/

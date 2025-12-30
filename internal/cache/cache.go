// Package cache реализует кэширование промежуточных результатов конвертации.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/artemshloyda/photoconverter/internal/config"
)

// Cache управляет кэшированием конвертированных изображений.
type Cache struct {
	// dir - директория для кэша.
	dir string

	// cfg - конфигурация.
	cfg *config.Config

	// enabled - включён ли кэш.
	enabled bool
}

// New создаёт новый Cache.
func New(cfg *config.Config) (*Cache, error) {
	if !cfg.CacheEnabled {
		return &Cache{enabled: false, cfg: cfg}, nil
	}

	dir := cfg.CacheDir
	if dir == "" {
		dir = filepath.Join(cfg.OutputDir, ".photoconverter", "cache")
	}

	// Создаём директорию кэша
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию кэша: %w", err)
	}

	return &Cache{
		dir:     dir,
		cfg:     cfg,
		enabled: true,
	}, nil
}

// IsEnabled возвращает true если кэш включён.
func (c *Cache) IsEnabled() bool {
	return c.enabled
}

// CacheKey генерирует ключ кэша на основе пути файла и параметров конвертации.
func (c *Cache) CacheKey(srcPath string, paramsHash string) string {
	h := sha256.New()
	h.Write([]byte(srcPath))
	h.Write([]byte(paramsHash))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// Get возвращает путь к кэшированному файлу, если он существует.
// Возвращает пустую строку если файл не найден в кэше.
func (c *Cache) Get(srcPath string, paramsHash string) string {
	if !c.enabled {
		return ""
	}

	key := c.CacheKey(srcPath, paramsHash)
	ext := "." + string(c.cfg.OutputFormat)
	cachePath := filepath.Join(c.dir, key+ext)

	if _, err := os.Stat(cachePath); err == nil {
		return cachePath
	}

	return ""
}

// Put сохраняет файл в кэш.
func (c *Cache) Put(srcPath string, paramsHash string, convertedPath string) error {
	if !c.enabled {
		return nil
	}

	key := c.CacheKey(srcPath, paramsHash)
	ext := filepath.Ext(convertedPath)
	cachePath := filepath.Join(c.dir, key+ext)

	// Копируем файл в кэш
	return copyFile(convertedPath, cachePath)
}

// CopyFromCache копирует файл из кэша в целевой путь.
func (c *Cache) CopyFromCache(cachePath string, dstPath string) error {
	// Создаём директорию для целевого файла
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	return copyFile(cachePath, dstPath)
}

// Clear очищает весь кэш.
func (c *Cache) Clear() error {
	if !c.enabled || c.dir == "" {
		return nil
	}

	return os.RemoveAll(c.dir)
}

// Size возвращает общий размер кэша в байтах.
func (c *Cache) Size() (int64, error) {
	if !c.enabled || c.dir == "" {
		return 0, nil
	}

	var size int64
	err := filepath.WalkDir(c.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// copyFile копирует файл из src в dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

/*
Возможные расширения:
- LRU eviction при превышении лимита размера
- TTL для записей кэша
- Сжатие кэшированных файлов
- Распределённый кэш (Redis, memcached)
*/

package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/samber/lo"

	"github.com/anatolyi0311/cupurl/internal/config"
)

type Repository interface {
	Get(hash string) (string, error)
	Set(url, hash string) error
}

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileStorage struct {
	cfg *config.Config
	s   []URLRecord
	mu  sync.RWMutex
}

func NewStorage(cfg *config.Config) (Repository, error) {

	fs := &FileStorage{
		cfg: cfg,
	}

	err := fs.loadFromFile()
	if err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *FileStorage) Get(hash string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	item, exist := lo.Find(fs.s, func(item URLRecord) bool {
		return item.ShortURL == hash
	})
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}

	return item.OriginalURL, nil
}

func (fs *FileStorage) Set(url, hash string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.s = append(fs.s, URLRecord{
		UUID:        strconv.Itoa(len(fs.s)),
		ShortURL:    hash,
		OriginalURL: url,
	})
	return fs.saveToFile()
}
func (fs *FileStorage) saveToFile() error {
	dir := filepath.Dir(fs.cfg.Opts.StorageFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(fs.s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(fs.cfg.Opts.StorageFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (fs *FileStorage) loadFromFile() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, err := os.Stat(fs.cfg.Opts.StorageFile); os.IsNotExist(err) {
		fmt.Println("File does not exist")
		return nil
	}

	data, err := os.ReadFile(fs.cfg.Opts.StorageFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &fs.s); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

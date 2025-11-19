package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	Ping() error
}

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage struct {
	cfg         *config.Config
	s           []URLRecord
	mu          sync.RWMutex
	db          *sql.DB
	memoryCache map[string]string
	hasFile     bool
}

func NewStorage(cfg *config.Config, db *sql.DB) (Repository, error) {
	s := &Storage{
		cfg:         cfg,
		db:          db,
		memoryCache: make(map[string]string),
		hasFile:     cfg.Opts.StorageFile != "",
	}
	err := s.loadFromFile()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) Get(hash string) (string, error) {
	if s.db != nil {
		return s.getPsql(hash)
	}
	if s.hasFile {
		return s.getFromFile(hash)
	}
	return s.getMemory(hash)
}

func (s *Storage) Set(url, hash string) error {
	if s.db != nil {
		return s.setPsql(url, hash)
	}
	if s.hasFile {
		return s.setInFile(url, hash)
	}
	return s.setMemory(url, hash)
}

func (s *Storage) saveToFile() error {
	dir := filepath.Dir(s.cfg.Opts.StorageFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(s.cfg.Opts.StorageFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *Storage) loadFromFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hasFile {
		return nil
	}

	if _, err := os.Stat(s.cfg.Opts.StorageFile); os.IsNotExist(err) {
		fmt.Println("File does not exist")
		return nil
	}

	data, err := os.ReadFile(s.cfg.Opts.StorageFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.s); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

func (s *Storage) Ping() error {
	if s.db == nil {
		return fmt.Errorf("db is not init")
	}
	return s.db.Ping()
}

func (s *Storage) setPsql(url, hash string) error {
	_, err := s.db.Exec(querySetURL, url, hash)
	return err
}

func (s *Storage) getPsql(hash string) (string, error) {
	var originalURL string
	err := s.db.QueryRow(queryGetURL, hash).Scan(&originalURL)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	return originalURL, nil
}

func (s *Storage) setInFile(url, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.s = append(s.s, URLRecord{
		UUID:        strconv.Itoa(len(s.s)),
		ShortURL:    hash,
		OriginalURL: url,
	})
	return s.saveToFile()
}

func (s *Storage) getFromFile(hash string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exist := lo.Find(s.s, func(item URLRecord) bool {
		return item.ShortURL == hash
	})
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}

	return item.OriginalURL, nil
}

func (s *Storage) getMemory(hash string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exist := s.memoryCache[hash]
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}
	return url, nil
}

func (s *Storage) setMemory(url, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.memoryCache[hash] = url
	return nil
}

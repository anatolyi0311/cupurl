package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/samber/lo"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
)

type Repository interface {
	Get(hash string) (string, error)
	Set(url, hash string) error
	Ping() error
	SetArrayURL(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error)
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

func NewStorage(cfg *config.Config, db *sql.DB, logger zap.SugaredLogger) (Repository, error) {
	s := &Storage{
		cfg:         cfg,
		db:          db,
		memoryCache: make(map[string]string),
		hasFile:     cfg.Opts.StorageFile != "",
	}
	err := s.loadFromFile(logger)
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

func (s *Storage) loadFromFile(logger zap.SugaredLogger) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hasFile {
		return nil
	}

	if _, err := os.Stat(s.cfg.Opts.StorageFile); os.IsNotExist(err) {
		logger.Warn("File does not exist")
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

	if _, exists := lo.Find(s.s, func(record URLRecord) bool {
		return record.OriginalURL == url
	}); exists {
		return nil
	}

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

func (s *Storage) SetArrayURL(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error) {
	if s.db != nil {
		return s.setArrayPsql(req)
	}
	if s.hasFile {
		return s.setArrayInFile(req)
	}
	return s.setArrayMemory(req)
}

func (s *Storage) setArrayPsql(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	resp := []model.SetArrayURLResponse{}
	for _, item := range req {
		_, err := tx.Exec(querySetURL, item.OriginalURL, item.ShortURL)
		if err != nil {
			return nil, err
		}
		resp = append(resp, model.SetArrayURLResponse{
			ID: item.ID,
			// URL: item.ShortURL,
			URL: s.cfg.Opts.BaseURL + "/" + item.ShortURL,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Storage) setArrayInFile(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := []model.SetArrayURLResponse{}

	for _, item := range req {
		if existingRecord, exists := lo.Find(s.s, func(record URLRecord) bool {
			return record.OriginalURL == item.OriginalURL
		}); exists {
			resp = append(resp, model.SetArrayURLResponse{
				ID:  item.ID,
				URL: existingRecord.ShortURL,
			})
		} else {
			newRecord := URLRecord{
				UUID:        strconv.Itoa(len(s.s)),
				ShortURL:    item.ShortURL,
				OriginalURL: item.OriginalURL,
			}
			s.s = append(s.s, newRecord)
			resp = append(resp, model.SetArrayURLResponse{
				ID:  item.ID,
				URL: item.ShortURL,
			})
		}
	}

	if err := s.saveToFile(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Storage) setArrayMemory(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error) {
	resp := []model.SetArrayURLResponse{}
	for _, item := range req {
		err := s.setMemory(item.OriginalURL, item.ShortURL)
		if err != nil {
			return resp, err
		}
		resp = append(resp, model.SetArrayURLResponse{
			ID:  item.ID,
			URL: item.ShortURL,
		})
	}
	return resp, nil
}

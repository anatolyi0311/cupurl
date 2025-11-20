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
	Get(hash string, logger zap.SugaredLogger) (string, error)
	Set(url, hash string, logger zap.SugaredLogger) (string, error)
	Ping() error
	SetArrayURL(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error)
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
		logger.Infow(
			"Storage.NewStorage.1",
			"err", err,
		)
		return nil, err
	}
	logger.Infow(
		"Storage.NewStorage.2",
		"s.hasFile", s.hasFile,
		"db", s.db,
	)
	return s, nil
}

func (s *Storage) Get(hash string, logger zap.SugaredLogger) (string, error) {
	logger.Infow(
		"Storage.Get",
		"hash", hash,
		"db", s.db,
		"file", s.hasFile,
	)
	if s.db != nil {
		logger.Infow(
			"Storage.Get.1",
			"hash", hash,
		)
		return s.getPsql(hash)
	}
	if s.hasFile {
		logger.Infow(
			"Storage.Get.2",
			"hash", hash,
		)
		return s.getFromFile(hash, logger)
	}
	return s.getMemory(hash, logger)
}

func (s *Storage) Set(url, hash string, logger zap.SugaredLogger) (string, error) {
	logger.Infow(
		"Storage.Set",
		"url", url,
		"hash", hash,
		"db", s.db,
		"file", s.hasFile,
	)
	if s.db != nil {
		logger.Infow(
			"Storage.Set.1",
			"hash", hash,
		)
		return s.setPsql(url, hash, logger)
	}
	if s.hasFile {
		logger.Infow(
			"Storage.Set.2",
			"hash", hash,
		)
		return s.setInFile(url, hash, logger)
	}
	logger.Infow(
		"Storage.Set.3",
		"hash", hash,
	)
	return s.setMemory(url, hash, logger)
}

func (s *Storage) saveToFile(logger zap.SugaredLogger) error {
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

	logger.Infow(
		"Storage.saveToFile",
		"data", string(data),
		"file", s.cfg.Opts.StorageFile,
	)

	return nil
}

func (s *Storage) loadFromFile(logger zap.SugaredLogger) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hasFile {
		return nil
	}

	logger.Infow(
		"Storage.loadFromFile",
		"file", s.cfg.Opts.StorageFile,
	)

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

func (s *Storage) setPsql(url, hash string, logger zap.SugaredLogger) (string, error) {
	result, err := s.db.Exec(querySetURL, url, hash)

	if err != nil {
		return "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	logger.Infow(
		"Storage.setPsql",
		"hash", hash,
		"rowsAffected", rowsAffected,
	)

	if rowsAffected == 0 {
		return hash, model.ErrURLAlreadyExists
	}

	return hash, nil
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

func (s *Storage) setInFile(url, hash string, logger zap.SugaredLogger) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := lo.Find(s.s, func(record URLRecord) bool {
		return record.OriginalURL == url
	}); exists {
		existingShortURL, _ := s.getFromFile(hash, logger)
		logger.Infow(
			"Storage.setInFile.1",
			"existingShortURL", existingShortURL,
		)
		return existingShortURL, model.ErrURLAlreadyExists
	}

	s.s = append(s.s, URLRecord{
		UUID:        strconv.Itoa(len(s.s)),
		ShortURL:    hash,
		OriginalURL: url,
	})
	err := s.saveToFile(logger)
	logger.Infow(
		"Storage.setInFile.2",
		"ShortURL", hash,
		"OriginalURL", url,
	)

	return "", err
}

func (s *Storage) getFromFile(hash string, logger zap.SugaredLogger) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger.Infow(
		"Storage.getFromFile",
		"hash ", hash,
	)

	if hash == "" {
		item, exist := lo.Find(s.s, func(item URLRecord) bool {
			return item.OriginalURL != ""
		})
		if !exist {
			return "", fmt.Errorf("%s not found", hash)
		}
		return item.OriginalURL, nil
	}

	item, exist := lo.Find(s.s, func(item URLRecord) bool {
		return item.ShortURL == hash
	})
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}

	return item.OriginalURL, nil
}

func (s *Storage) getMemory(hash string, logger zap.SugaredLogger) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if hash == "/" || hash == "" {
		for key, val := range s.memoryCache {
			if val != "" {
				// s.memoryCache[key] = ""
				delete(s.memoryCache, key)
				logger.Infow(
					"Storage.getMemory.1",
					"hash", hash,
					"key", key,
					"url", val,
					"s.memoryCache", s.memoryCache,
				)
				return val, nil
			}
		}
	}

	url, exist := s.memoryCache[hash]
	logger.Infow(
		"Storage.getMemory.2",
		"hash", hash,
		"url", url,
		"exist", exist,
		"s.memoryCache", s.memoryCache,
	)
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}

	return url, nil
}

func (s *Storage) setMemory(url, hash string, logger zap.SugaredLogger) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exist := s.memoryCache[hash]; exist {
		existingShortURL, _ := s.getMemory(hash, logger)

		logger.Infow(
			"Storage.SetMemory.1",
			"exist", exist,
			"existingShortURL", existingShortURL,
		)
		return existingShortURL, model.ErrURLAlreadyExists
	}

	s.memoryCache[hash] = url

	logger.Infow(
		"Storage.SetMemory",
		"hash", hash,
		"url", url,
	)

	return "", nil
}

func (s *Storage) SetArrayURL(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	if s.db != nil {
		return s.setArrayPsql(req, logger)
	}
	if s.hasFile {
		return s.setArrayInFile(req, logger)
	}
	return s.setArrayMemory(req, logger)
}

func (s *Storage) setArrayPsql(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
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

func (s *Storage) setArrayInFile(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
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

	if err := s.saveToFile(logger); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Storage) setArrayMemory(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	resp := []model.SetArrayURLResponse{}
	for _, item := range req {
		s.setMemory(item.OriginalURL, item.ShortURL, logger)
		resp = append(resp, model.SetArrayURLResponse{
			ID:  item.ID,
			URL: item.ShortURL,
		})
	}
	return resp, nil
}

package repository

import (
	"database/sql"
	"fmt"
	"sync"

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
		return nil, err
	}
	return s, nil
}

func (s *Storage) Get(hash string, logger zap.SugaredLogger) (string, error) {
	if s.db != nil {
		return s.getPsql(hash)
	}
	if s.hasFile {
		return s.getFromFile(hash, logger)
	}
	return s.getMemory(hash, logger)
}

func (s *Storage) Set(url, hash string, logger zap.SugaredLogger) (string, error) {
	if s.db != nil {
		return s.setPsql(url, hash, logger)
	}
	if s.hasFile {
		return s.setInFile(url, hash, logger)
	}
	return s.setMemory(url, hash, logger)
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

func (s *Storage) Ping() error {
	if s.db == nil {
		return fmt.Errorf("db is not init")
	}
	return s.db.Ping()
}

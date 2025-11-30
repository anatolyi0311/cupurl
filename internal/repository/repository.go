package repository

import (
	"context"
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
	GetArray(logger zap.SugaredLogger) ([]model.GetArrayURLRequest, error)
	DeleteArray(ctx context.Context, urls []string, logger zap.SugaredLogger) error
}

type URLRecord struct {
	UUID        string `json:"uuid" db:"uuid"`
	ShortURL    string `json:"short_url" db:"short_url"`
	OriginalURL string `json:"original_url" db:"original_url"`
	DeletedFlag bool   `json:"is_deleted" db:"is_deleted"`
	CreatedByID string
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
		return s.getPsql(hash, logger)
	}
	if s.hasFile {
		return s.getFromFile(hash, logger)
	}
	return s.getMemory(hash, logger)
}

func (s *Storage) GetArray(logger zap.SugaredLogger) ([]model.GetArrayURLRequest, error) {
	if s.db != nil {
		return s.getArrayPsql(logger)
	}
	if s.hasFile {
		return s.getArrayFromFile(logger)
	}
	return s.getArrayMemory(logger)
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

func (s *Storage) DeleteArray(ctx context.Context, toDel []string, logger zap.SugaredLogger) error {
	logger.Info("DeleteArray.urls: ", len(toDel), toDel)
	if len(toDel) == 0 {
		logger.Warn("DeleteArray.urls.1: toDel ", toDel, len(toDel))
		return nil
	}
	if s.db != nil {
		logger.Info("DeleteArray.urls.2: toDel ", toDel, len(toDel))
		// if len(toDel) == 0 {
		// 	return nil
		// }
		err := s.deleteArrayPsql(ctx, toDel, logger)
		// if err != nil {

		// }
		return err
		// s.deleteArrayPsql(ctx, toDel)
	}
	return nil
}

func (s *Storage) Ping() error {
	if s.db == nil {
		return fmt.Errorf("db is not init")
	}
	return s.db.Ping()
}

func (s *Storage) FormatURL(hash string) string {
	return s.cfg.Opts.BaseURL + "/" + hash
}

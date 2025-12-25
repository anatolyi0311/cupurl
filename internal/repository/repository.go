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
	Get(hash string, logger zap.SugaredLogger, userID int) (model.ShortURL, error)
	Set(shortURL model.ShortURL, logger zap.SugaredLogger, userID int) (model.ShortURL, error)
	Ping() error
	SetArrayURL(request []model.SetArrayURLRequest, logger zap.SugaredLogger, userID int) ([]model.ShortURL, error)
	GetArray(logger zap.SugaredLogger) ([]model.ShortURL, error)
	Delete(hash string, logger zap.SugaredLogger, userID int) error
	GetUsersAndUrlsCount(ctx context.Context) (int, int, error)
}

type URLRecord struct {
	UUID        string `json:"uuid" db:"uuid"`
	ShortURL    string `json:"short_url" db:"shortURL"`
	OriginalURL string `json:"original_url" db:"originalURL"`
	DeletedFlag bool   `json:"is_deleted" db:"deletedFlag"`
	CreatedByID string `json:"created_by" db:"createdBy"`
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

func (s *Storage) Get(hash string, logger zap.SugaredLogger, userID int) (model.ShortURL, error) {
	if s.db != nil {
		return s.getPsql(hash, logger, userID)
	}
	if s.hasFile {
		return s.getFromFile(hash, logger)
	}
	return s.getMemory(hash, logger)
}

func (s *Storage) GetArray(logger zap.SugaredLogger) ([]model.ShortURL, error) {
	if s.db != nil {
		res, err := s.getArrayPsql(logger)
		r := make([]model.ShortURL, 0, len(res))
		for i := range res {
			r = append(r, model.ShortURL{OriginalURL: res[i].Original, ShortURL: res[i].Short})
		}
		return r, err
	}
	if s.hasFile {
		return s.getArrayFromFile(logger)
	}
	return s.getArrayMemory(logger)
}

func (s *Storage) Set(shortURL model.ShortURL, logger zap.SugaredLogger, userID int) (model.ShortURL, error) {
	if s.db != nil {
		return s.setPsql(shortURL, logger, userID)
	}
	if s.hasFile {
		return s.setInFile(shortURL.OriginalURL, shortURL.ShortURL, logger)
	}
	return s.setMemory(shortURL.OriginalURL, shortURL.ShortURL, logger)
}

func (s *Storage) SetArrayURL(request []model.SetArrayURLRequest, logger zap.SugaredLogger, userID int) ([]model.ShortURL, error) {
	if s.db != nil {
		return s.setArrayPsql(request, logger, userID)
	}
	if s.hasFile {
		return s.setArrayInFile(request, logger)
	}
	return s.setArrayMemory(request, logger)
}

func (s *Storage) Delete(hash string, logger zap.SugaredLogger, userID int) error {
	if s.db != nil {
		err := s.DeleteDB(hash, userID)
		return err
	}
	if s.hasFile {
		return s.DeleteFile(hash, logger)
	}
	return s.DeleteMem(hash)
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

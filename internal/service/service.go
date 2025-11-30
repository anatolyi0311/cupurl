package service

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
	repo "github.com/anatolyi0311/cupurl/internal/repository"
	"go.uber.org/zap"
)

const (
	sizeHash = 16
)

type CaseURL interface {
	SetURL(url string, logger zap.SugaredLogger) (string, error)
	GetURL(hash string, logger zap.SugaredLogger) (string, error)
	SetArrayURL(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error)
	Ping() error
	GetArrayURL(logger zap.SugaredLogger) ([]model.GetArrayURLRequest, error)
}

type Service struct {
	repo repo.Repository
}

func NewService(cfg *config.Config, db *sql.DB, logger zap.SugaredLogger) (CaseURL, error) {
	repo, err := repo.NewStorage(cfg, db, logger)

	if err != nil {
		return nil, err
	}

	return &Service{
		repo: repo,
	}, nil
}

func (s *Service) SetURL(urlTo string, logger zap.SugaredLogger) (string, error) {
	// ...
	urlTo = strings.TrimSpace(urlTo)
	if urlTo == "" {
		logger.Warn("url.hash.empty")
		return "", fmt.Errorf("incorrect url")
	}

	hash := sha256.Sum256([]byte(urlTo))

	// To unique.
	now := time.Now()
	seconds := now.Unix() // Unix timestamp in seconds
	hash2 := [32]byte{byte(seconds)}
	combinedHash := append(hash[:], hash2[:]...)

	shortHash := fmt.Sprintf("%x", combinedHash[:sizeHash])
	shortHash, err := s.repo.Set(urlTo, shortHash, logger)
	if shortHash == "" {
		logger.Warn("url.hash.empty")
		return "", fmt.Errorf("incorrect id")
	}

	// logger.Info("Set.URL.hash: ", shortHash, " urlTo:", urlTo)
	return shortHash, err
}

func (s *Service) GetURL(hash string, logger zap.SugaredLogger) (string, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "", fmt.Errorf("incorrect id")
	}
	// logger.Info("Get.URL.hash: ", hash)
	return s.repo.Get(hash, logger)
}

func (s *Service) SetArrayURL(req []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	for i, item := range req {
		item.OriginalURL = strings.TrimSpace(item.OriginalURL)
		if item.OriginalURL == "" {
			return nil, fmt.Errorf("incorrect url")
		}
		hash := sha256.Sum256([]byte(item.OriginalURL))
		req[i].ShortURL = fmt.Sprintf("%x", hash[:8])
	}
	return s.repo.SetArrayURL(req, logger)
}

func (s *Service) GetArrayURL(logger zap.SugaredLogger) ([]model.GetArrayURLRequest, error) {
	return s.repo.GetArray(logger)
}

func (s *Service) Ping() error {
	return s.repo.Ping()
}

func (s *Service) FormatURL(baseURL, hash string) string {
	return baseURL + "/" + hash
}

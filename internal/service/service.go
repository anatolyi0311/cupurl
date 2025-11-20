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
}

type Service struct {
	repo repo.Repository
}

func NewService(cfg *config.Config, db *sql.DB, logger zap.SugaredLogger) (CaseURL, error) {
	repo, err := repo.NewStorage(cfg, db, logger)

	if err != nil {
		logger.Infow(
			"Service.New.1",
			"err", err.Error(),
		)

		return nil, err
	}
	logger.Infow(
		"Service.New.2",
		"db", db,
		"repo", repo,
	)

	return &Service{
		repo: repo,
	}, nil
}

func (s *Service) SetURL(urlTo string, logger zap.SugaredLogger) (string, error) {
	logger.Infow(
		"Service.SetURL",
		"url.hash", urlTo,
	)

	urlTo = strings.TrimSpace(urlTo)
	if urlTo == "" {
		logger.Infow(
			"Service.SetURL.1",
			"url.hash", urlTo,
		)
		// return "", fmt.Errorf("incorrect url")
	}

	hash := sha256.Sum256([]byte(urlTo))

	// To unique.
	now := time.Now()
	seconds := now.Unix() // Unix timestamp in seconds
	hash2 := [32]byte{byte(seconds)}
	combinedHash := append(hash[:], hash2[:]...)

	shortHash := fmt.Sprintf("%x", combinedHash[:sizeHash])
	shortHash, err := s.repo.Set(urlTo, shortHash, logger)
	logger.Infow(
		"Service.SetURL.2",
		"hash", hash,
		"shortHash", shortHash,
	)
	return shortHash, err
}

func (s *Service) GetURL(hash string, logger zap.SugaredLogger) (string, error) {
	logger.Infow(
		"Service.GetURL",
		"hash", hash,
	)

	hash = strings.TrimSpace(hash)
	// if hash == "" {
	// 	return "", fmt.Errorf("incorrect id")
	// }
	return s.repo.Get(hash, logger)
}

func (s *Service) Ping() error {
	return s.repo.Ping()
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

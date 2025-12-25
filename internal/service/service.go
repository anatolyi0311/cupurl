package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/anatolyi0311/cupurl/internal/repository"
)

const (
	sizeHash = 16
)

type CaseURL interface {
	SetURL(url string, userID int) (model.ShortURL, error)
	GetURL(hash string, userID int) (model.ShortURL, error)
	SetArrayURL(request []model.SetArrayURLRequest, userID int) ([]model.ShortURL, error)
	Ping() error
	GetArrayURL() ([]model.ShortURL, error)
	DeleteArrayURL(hash []string, userID int) // variant 2
	GetStats(ctx context.Context) (model.Stats, error)
}

type Service struct {
	repo   repository.Repository
	logger zap.SugaredLogger
}

func NewService(cfg *config.Config, db *sql.DB, logger zap.SugaredLogger) (CaseURL, error) {
	repo, err := repository.NewStorage(cfg, db, logger)

	if err != nil {
		return nil, err
	}

	return &Service{
		repo:   repo,
		logger: logger,
	}, nil
}

func (s *Service) SetURL(urlTo string, userID int) (model.ShortURL, error) {
	urlTo = strings.TrimSpace(urlTo)
	if urlTo == "" {
		s.logger.Warn("url.hash.empty")
		return model.ShortURL{}, fmt.Errorf("incorrect url")
	}
	hash := sha256.Sum256([]byte(urlTo))
	// To unique.
	now := time.Now()
	seconds := now.Unix() // Unix timestamp in seconds
	hash2 := [32]byte{byte(seconds)}
	combinedHash := append(hash[:], hash2[:]...)

	shortHashSize := fmt.Sprintf("%x", combinedHash[:sizeHash])
	shortHash, err := s.repo.Set(model.ShortURL{OriginalURL: urlTo, ShortURL: shortHashSize, UserID: userID}, s.logger, userID)
	if shortHash.ShortURL == "" {
		s.logger.Warn("url.hash.empty")
		return model.ShortURL{}, fmt.Errorf("incorrect id")
	}
	return shortHash, err
}

func (s *Service) GetURL(hash string, userID int) (model.ShortURL, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return model.ShortURL{}, fmt.Errorf("incorrect id")
	}
	originalURL, err := s.repo.Get(hash, s.logger, userID)
	return originalURL, err
}

func (s *Service) SetArrayURL(request []model.SetArrayURLRequest, userID int) ([]model.ShortURL, error) {
	for i, item := range request {
		item.OriginalURL = strings.TrimSpace(item.OriginalURL)
		if item.OriginalURL == "" {
			return nil, fmt.Errorf("incorrect url")
		}
		hash := sha256.Sum256([]byte(item.OriginalURL))
		request[i].ShortURL = fmt.Sprintf("%x", hash[:8])
		// req[i].UserID = userID
	}
	return s.repo.SetArrayURL(request, s.logger, userID)
}

func (s *Service) GetArrayURL() ([]model.ShortURL, error) {
	return s.repo.GetArray(s.logger)
}

func (s *Service) DeleteArrayURL(hashArray []string, userID int) {
	for _, hash := range hashArray {
		go func(hash string) {
			err := s.repo.Delete(hash, s.logger, userID)
			if err != nil {
				s.logger.Warn(err)
			}
		}(hash)
	}
}

func (s *Service) GetStats(ctx context.Context) (model.Stats, error) {
	usersCount, urlsCount, err := s.repo.GetUsersAndUrlsCount(ctx)
	if err != nil {
		return model.Stats{}, err
	}
	return model.Stats{UsersCount: usersCount, UrlsCount: urlsCount}, nil
}

func (s *Service) Ping() error {
	return s.repo.Ping()
}

func (s *Service) FormatURL(baseURL, hash string) string {
	return baseURL + "/" + hash
}

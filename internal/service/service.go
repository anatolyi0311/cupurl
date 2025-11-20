package service

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/anatolyi0311/cupurl/internal/config"
	repo "github.com/anatolyi0311/cupurl/internal/repository"
	"go.uber.org/zap"
)

const sizeHash = 16

type CaseURL interface {
	SetURL(url string) (string, error)
	GetURL(hash string) (string, error)
	Ping() error
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

func (s *Service) SetURL(urlTo string) (string, error) {
	urlTo = strings.TrimSpace(urlTo)
	if urlTo == "" {
		return "", fmt.Errorf("incorrect url")
	}

	hash := sha256.Sum256([]byte(urlTo))

	// To unique.
	now := time.Now()
	seconds := now.Unix() // Unix timestamp in seconds
	hash2 := [32]byte{byte(seconds)}
	combinedHash := append(hash[:], hash2[:]...)

	shortHash := fmt.Sprintf("%x", combinedHash[:sizeHash])
	err := s.repo.Set(urlTo, shortHash)

	return shortHash, err
}

func (s *Service) GetURL(hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "", fmt.Errorf("incorrect id")
	}
	return s.repo.Get(hash)
}

func (s *Service) Ping() error {
	return s.repo.Ping()
}

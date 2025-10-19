package service

import (
	"crypto/sha256"
	"fmt"
	"strings"

	repo "github.com/anatolyi0311/cupurl/internal/repository"
)

type CaseURL interface {
	SetURL(url string) (string, error)
	GetURL(hash string) (string, error)
}

type Service struct {
	repo repo.Repository
}

func NewService() CaseURL {
	return &Service{
		repo: repo.NewStorage(),
	}
}

func (s *Service) SetURL(urlTo string) (string, error) {
	// pUrl, err:= url.Parse(urlTo)
	// if err != nil {
	// 	return "", fmt.Errorf("incorrect url")
	// }
	urlTo = strings.TrimSpace(urlTo)
	if urlTo == "" {
		return "", fmt.Errorf("incorrect url")
	}

	hash := sha256.Sum256([]byte(urlTo))
	shortHash := fmt.Sprintf("%x", hash[:8])
	err := s.repo.Set(urlTo, shortHash)

	return shortHash, err
}

func (s *Service) GetURL(hash string) (string, error) {
	// pUrl, err:= url.Parse(urlTo)
	// if err != nil {
	// 	return "", fmt.Errorf("incorrect url")
	// }
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "", fmt.Errorf("incorrect id")
	}
	return s.repo.Get(hash)
}

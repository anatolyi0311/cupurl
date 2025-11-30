package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
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
	SetURL(url string) (string, error)
	GetURL(hash string) (model.ShortURL, error)
	SetArrayURL(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error)
	Ping() error
	GetArrayURL() ([]model.GetArrayURLRequest, error)
	DeleteArrayURL(ctx context.Context, urls []string, userID string)
}

type Service struct {
	repo   repo.Repository
	logger zap.SugaredLogger
}

func NewService(cfg *config.Config, db *sql.DB, logger zap.SugaredLogger) (CaseURL, error) {
	repo, err := repo.NewStorage(cfg, db, logger)

	if err != nil {
		return nil, err
	}

	return &Service{
		repo:   repo,
		logger: logger,
	}, nil
}

func (s *Service) SetURL(urlTo string) (string, error) {
	// ...
	urlTo = strings.TrimSpace(urlTo)
	if urlTo == "" {
		s.logger.Warn("url.hash.empty")
		return "", fmt.Errorf("incorrect url")
	}

	hash := sha256.Sum256([]byte(urlTo))

	// To unique.
	now := time.Now()
	seconds := now.Unix() // Unix timestamp in seconds
	hash2 := [32]byte{byte(seconds)}
	combinedHash := append(hash[:], hash2[:]...)

	shortHash := fmt.Sprintf("%x", combinedHash[:sizeHash])
	shortHash, err := s.repo.Set(urlTo, shortHash, s.logger)
	if shortHash == "" {
		s.logger.Warn("url.hash.empty")
		return "", fmt.Errorf("incorrect id")
	}

	// logger.Info("Set.URL.hash: ", shortHash, " urlTo:", urlTo)
	return shortHash, err
}

func (s *Service) GetURL(hash string) (model.ShortURL, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return model.ShortURL{}, fmt.Errorf("incorrect id")
	}
	// logger.Info("Get.URL.hash: ", hash)
	originalURL, err := s.repo.Get(hash, s.logger)
	// if originalURL == "" {

	// }
	return model.ShortURL{OriginalURL: originalURL}, err
}

func (s *Service) SetArrayURL(req []model.SetArrayURLRequest) ([]model.SetArrayURLResponse, error) {
	for i, item := range req {
		item.OriginalURL = strings.TrimSpace(item.OriginalURL)
		if item.OriginalURL == "" {
			return nil, fmt.Errorf("incorrect url")
		}
		hash := sha256.Sum256([]byte(item.OriginalURL))
		req[i].ShortURL = fmt.Sprintf("%x", hash[:8])
	}
	return s.repo.SetArrayURL(req, s.logger)
}

func (s *Service) GetArrayURL() ([]model.GetArrayURLRequest, error) {
	return s.repo.GetArray(s.logger)
}

func (s *Service) DeleteArrayURL(ctx context.Context, ids []string, userID string) {
	done := make(chan struct{})
	defer close(done)

	workersCount := runtime.NumCPU()
	inputCh := make(chan string)
	// modelsToDelete := make([]model.GetArrayURLRequest, 0, len(ids))
	toDel := make([]string, 0, len(ids))

	go func() {
		for _, id := range ids {
			inputCh <- id
		}

		close(inputCh)
	}()

	workerChs := make([]chan model.GetArrayURLRequest, 0, workersCount)
	for urlID := range inputCh {
		workerCh := make(chan model.GetArrayURLRequest)
		newWorker(urlID, userID, workerCh)
		workerChs = append(workerChs, workerCh)
	}

	for v := range fanIn(done, workerChs...) {
		// modelsToDelete = append(modelsToDelete, v)
		toDel = append(toDel, v.Hash) // v.ShortURL
	}
	// s.repo.DeleteArray(context.Background(), toDel, s.logger)
	err := s.repo.DeleteArray(ctx, toDel, s.logger)
	if err != nil {
		s.logger.Warn("Delete: ", toDel)
		fmt.Printf("couldn't delete urls: %v\n", err)
	}
}

func (s *Service) Ping() error {
	return s.repo.Ping()
}

func (s *Service) FormatURL(baseURL, hash string) string {
	return baseURL + "/" + hash
}

func newWorker(urlID string, userID string, out chan model.GetArrayURLRequest) {
	go func() {
		defer func() {
			if x := recover(); x != nil {
				newWorker(urlID, userID, out)
				log.Printf("run time panic: %v, %v", x, out)
			}
		}()

		out <- model.GetArrayURLRequest{ID: urlID, UserID: userID, Hash: urlID}
		close(out)
	}()
}

func fanIn(done <-chan struct{}, channels ...chan model.GetArrayURLRequest) chan model.GetArrayURLRequest {
	var wg sync.WaitGroup
	multiplexedStream := make(chan model.GetArrayURLRequest)

	multiplex := func(c <-chan model.GetArrayURLRequest) {
		defer wg.Done()
		for v := range c {
			select {
			case <-done:
				return
			case multiplexedStream <- v:
			}
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}

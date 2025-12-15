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

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/anatolyi0311/cupurl/internal/repository"
)

const (
	sizeHash = 16
)

type CaseURL interface {
	SetURL(url string) (model.ShortURL, error)
	GetURL(hash string) (model.ShortURL, error)
	SetArrayURL(req []model.SetArrayURLRequest) ([]model.ShortURL, error)
	Ping() error
	GetArrayURL() ([]model.ShortURL, error)
	DeleteArrayURL(hash []string)
	DeleteUrls(ctx context.Context, urls []string, userID string)
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

func (s *Service) SetURL(urlTo string) (model.ShortURL, error) {
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
	shortHash, err := s.repo.Set(model.ShortURL{OriginalURL: urlTo, ShortURL: shortHashSize}, s.logger)
	if shortHash.ShortURL == "" {
		s.logger.Warn("url.hash.empty")
		return model.ShortURL{}, fmt.Errorf("incorrect id")
	}
	return shortHash, err
}

func (s *Service) GetURL(hash string) (model.ShortURL, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return model.ShortURL{}, fmt.Errorf("incorrect id")
	}
	originalURL, err := s.repo.Get(hash, s.logger)
	return originalURL, err
}

func (s *Service) SetArrayURL(req []model.SetArrayURLRequest) ([]model.ShortURL, error) {
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

func (s *Service) GetArrayURL() ([]model.ShortURL, error) {
	return s.repo.GetArray(s.logger)
}

func (s *Service) DeleteArrayURL(hashArray []string) {
	for _, hash := range hashArray {
		go func(hash string) {
			err := s.repo.Delete(hash, s.logger)
			if err != nil {
				s.logger.Warn(err)
			}
		}(hash)
	}
}

func (s *Service) DeleteUrls(ctx context.Context, ids []string, userID string) {
	done := make(chan struct{})
	defer close(done)

	workersCount := runtime.NumCPU()
	inputCh := make(chan string)
	modelsToDelete := make([]model.ShortURL, 0, len(ids))
	// toDel := make([]string, 0, len(ids))

	go func() {
		for _, id := range ids {
			inputCh <- id
			s.logger.Info("..id: ", id)
		}
		close(inputCh)
	}()
	s.logger.Info("DeleteUrls.inputCh: ", len(inputCh))

	workerChs := make([]chan model.ShortURL, 0, workersCount)
	for urlID := range inputCh {
		workerCh := make(chan model.ShortURL)
		newWorker(urlID, userID, workerCh)
		workerChs = append(workerChs, workerCh)
		s.logger.Info("..urlID: ", urlID)
	}

	for v := range fanIn(done, workerChs...) {
		modelsToDelete = append(modelsToDelete, v)
	}
	err := s.repo.DeleteArray(ctx, modelsToDelete, s.logger)
	if err != nil {
		s.logger.Warn("Err.Delete: ", len(modelsToDelete))
		fmt.Printf("couldn't delete urls: %v\n", err)
	}
	s.logger.Info("DeleteUrls.workersCount: ", workersCount, " inputCh: ", len(inputCh))
	time.Sleep(time.Second)
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

func newWorker(urlID string, userID string, out chan model.ShortURL) {
	go func() {
		defer func() {
			if x := recover(); x != nil {
				newWorker(urlID, userID, out)
				log.Printf("run time panic: %v, %v", x, out)
			}
		}()
		out <- model.ShortURL{UserID: userID, ShortURL: urlID}
		close(out)
	}()
}

func fanIn(done chan struct{}, channels ...chan model.ShortURL) chan model.ShortURL {
	var wg sync.WaitGroup
	finalCh := make(chan model.ShortURL)

	multiplex := func(c chan model.ShortURL) {
		// defer wg.Done()
		for data := range c {
			select {
			case <-done:
				return
			case finalCh <- data:
			}
			time.Sleep(50 * time.Millisecond)
		}
		wg.Done()
	}

	wg.Add(len(channels))
	for _, c := range channels {
		chClosure := c
		// wg.Add(1)
		go multiplex(chClosure)
	}

	go func() {
		wg.Wait()
		close(finalCh)
	}()

	return finalCh
}

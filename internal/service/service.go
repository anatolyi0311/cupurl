package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/anatolyi0311/cupurl/internal/repository"
	"github.com/gin-gonic/gin"
)

const (
	sizeHash = 16
)

type CaseURL interface {
	SetURL(url string, userID int) (model.ShortURL, error)
	GetURL(userID string) (*model.ShortURL, error)
	SetArrayURL(request []model.SetArrayURLRequest, userID int) ([]model.ShortURL, error)
	Ping() error
	GetArrayURL() ([]model.ShortURL, error)
	DeleteArrayURL(hash []string, userID int)                      // variant 2
	DeleteUrls(context context.Context, hash []string, userID int) // variant 2
	GetStats(ctx context.Context) (model.Stats, error)
	MarkURLsAsDeleted(ctx context.Context, URLSToDel []string) error
}

type Service struct {
	repo    repository.Repository
	logger  zap.SugaredLogger
	storage *repository.MemCache
	baseURL string
}

func NewService(cfg *config.Config, db *sql.DB, logger zap.SugaredLogger) (CaseURL, error) {
	repo, err := repository.NewStorage(cfg, db, logger)

	if err != nil {
		return nil, err
	}

	cache := repository.NewCache()

	return &Service{
		repo:    repo,
		logger:  logger,
		storage: cache,
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

	s.storage.SaveHash(userID, shortHash.ShortURL)
	return *shortHash, err
}

func (s *Service) GetURL(hash string) (*model.ShortURL, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return &model.ShortURL{}, fmt.Errorf("incorrect id")
	}
	originalURL, err := s.repo.Get(hash, s.logger)
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
		request[i].UserID = userID
	}

	result, err := s.repo.SetArrayURL(request, s.logger, userID)
	if err != nil {
		for _, item := range result {
			s.storage.SaveHash(userID, item.ShortURL)
		}
	}

	return result, err
}

func (s *Service) GetArrayURL() ([]model.ShortURL, error) {
	return s.repo.GetArray(s.logger)
}

func (s *Service) DeleteArrayURL(hashArray []string, userID int) {
	for _, hash := range hashArray {
		go func(hash string) {
			if !s.storage.CanDeleteHash(userID, hash) {
				return
			}
			err := s.repo.Delete(hash, hashArray, s.logger, userID)
			if errors.Is(err, sql.ErrNoRows) {
				return
			}
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

func (s *Service) DeleteUrls(ctx context.Context, ids []string, userID int) {
	done := make(chan struct{})
	defer close(done)

	workersCount := runtime.NumCPU()
	inputCh := make(chan string)
	// modelsToDelete := make([]model.ShortURL, 0, len(ids))

	go func() {
		for _, id := range ids {
			inputCh <- id
		}
		close(inputCh)
	}()

	workerChs := make([]chan model.ShortURL, 0, workersCount)
	for urlID := range inputCh {
		workerCh := make(chan model.ShortURL, 1)
		newWorker(urlID, userID, workerCh)
		workerChs = append(workerChs, workerCh)
	}
	for v := range fanIn(done, workerChs...) {
		// modelsToDelete = append(modelsToDelete, v)
		go func(hash string) {
			if !s.storage.CanDeleteHash(userID, hash) {
				return
			}
			err := s.repo.Delete(hash, ids, s.logger, userID)
			if err != nil {
				s.logger.Warn(err)
			}
		}(v.ShortURL)

	}

	// err := s.repo.DeleteArray(ctx, modelsToDelete, s.logger, userID)
	// if err != nil {
	// 	s.logger.Warnf("couldn't delete urls: %v\n", err)
	// }
	time.Sleep(time.Second)
}

func newWorker(urlID string, userID int, out chan model.ShortURL) {
	go func() {
		defer func() {
			if x := recover(); x != nil {
				newWorker(urlID, userID, out)
				log.Printf("run time panic: %v, %v", x, out)
			}
		}()
		out <- model.ShortURL{ShortURL: urlID, UserID: userID}
		close(out)
	}()
}

func fanIn(done chan struct{}, channels ...chan model.ShortURL) chan model.ShortURL {
	var wg sync.WaitGroup
	finalCh := make(chan model.ShortURL, len(channels))
	multiplex := func(c chan model.ShortURL) {
		// defer wg.Done()
		for data := range c {
			select {
			case <-done:
				return
			case finalCh <- data:
			}
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
func (s *Service) MarkURLsAsDeleted(ctx context.Context, URLSToDel []string) error {
	return nil
}

func (s *Service) DelUserURLS(c *gin.Context) {
	ctx := c.Request.Context()
	var URLSToDel []string
	if err := c.ShouldBindJSON(&URLSToDel); err != nil {
		s.logger.Error(err)
		c.Status(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusAccepted)
	s.repo.AsyncDeleteUserURLs(ctx, URLSToDel)

}

package services

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"net/url"
	"strings"
	"time"

	"github.com/anatolyi0311/cupurl/internal/app/models"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks
type Repository interface {
	StoreURLInDB(ctx context.Context, originalURL, shortURL string) error
	GetShortURLFromDB(ctx context.Context, originalURL string) (string, error)
	GetOriginalURLFromDB(ctx context.Context, shortURL string) (string, error)
	StoreBatchURLInDB(ctx context.Context, batchURLtoStores map[string]string) error
	GetShortBatchURLFromDB(ctx context.Context, batchURLRequests []models.URLRequest) (map[string]string, error)
	GetUserURLSFromDB(ctx context.Context) ([]models.URL, error)
	MarkURLsAsDeleted(ctx context.Context, URLSToDel []string) error
}
type Encoder interface {
	CryptoBase62Encode() string
}

type ShortURLServices struct {
	repository Repository
	encoder    Encoder
	baseURL    string
}

type URLInMemoryRepository interface {
	SaveBatchToFile() error
}

func NewShortURLServices(repository Repository, encoder Encoder, baseURL string) *ShortURLServices {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		logrus.Error(err)
	}
	return &ShortURLServices{
		repository: repository,
		encoder:    encoder,
		baseURL:    parsedBaseURL.String(),
	}
}

func (s ShortURLServices) finalURLBuilder(shortURL string) string {
	resultURL, err := url.JoinPath(s.baseURL, shortURL)
	if err != nil {
		logrus.Error(err)
	}
	return resultURL
}

func (s ShortURLServices) GetBatchShortURL(ctx context.Context, batchURLRequests []models.URLRequest) ([]models.URLResponse, error) {
	shortsURL, err := s.repository.GetShortBatchURLFromDB(ctx, batchURLRequests)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	var batchURLtoStores = make(map[string]string, len(batchURLRequests))
	var batchURLResponses []models.URLResponse
	for _, value := range batchURLRequests {
		if shortURL, ok := shortsURL[value.OriginalURL]; ok {
			batchURLResponses = append(batchURLResponses,
				models.URLResponse{CorrelationID: value.CorrelationID, ShortURL: s.finalURLBuilder(shortURL)})
		} else {
			shortURL = s.encoder.CryptoBase62Encode()
			logrus.Info("shortURL:", len(shortURL), shortURL)
			batchURLtoStores[shortURL] = value.OriginalURL
			batchURLResponses = append(batchURLResponses,
				models.URLResponse{CorrelationID: value.CorrelationID, ShortURL: s.finalURLBuilder(shortURL)})
		}
	}

	err = s.repository.StoreBatchURLInDB(ctx, batchURLtoStores)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return batchURLResponses, nil
}

func (s ShortURLServices) GetShortURL(ctx context.Context, URL string) (string, error) {
	shortURL, err := s.repository.GetShortURLFromDB(ctx, URL)
	logrus.Info("shortURL:", len(shortURL), shortURL)
	if err != nil {
		shortURL = s.encoder.CryptoBase62Encode()
		err = s.repository.StoreURLInDB(ctx, URL, shortURL)
		if err != nil {
			return "", err
		}
		return s.finalURLBuilder(shortURL), nil
	}
	return s.finalURLBuilder(shortURL), models.ErrURLFound
}

func (s ShortURLServices) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	originURL, err := s.repository.GetOriginalURLFromDB(ctx, shortURL)
	if err != nil {
		return "", err
	}
	return originURL, nil
}

func (s ShortURLServices) GetUserURLS(ctx context.Context) ([]models.URL, error) {
	userURLS, err := s.repository.GetUserURLSFromDB(ctx)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	fullShortUserURLS := make([]models.URL, len(userURLS))
	for i, v := range userURLS {
		fullShortUserURLS[i].ShortURL = s.finalURLBuilder(v.ShortURL)
		fullShortUserURLS[i].OriginalURL = v.OriginalURL
	}
	return fullShortUserURLS, nil
}

func (s ShortURLServices) AsyncDeleteUserURLs(ctx context.Context, URLSToDel []string) {
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		defer cancel()
		if err := s.repository.MarkURLsAsDeleted(asyncCtx, URLSToDel); err != nil {
			logrus.Error(err)
		}
	}()
}

func (s ShortURLServices) CryptoBase62Encode() string {
	b := make([]byte, 8) // uint64 состоит из 8 байт, но мы будем использовать только 42 бита
	_, _ = rand.Read(b)
	num := binary.BigEndian.Uint64(b) & ((1 << 42) - 1) // Обнуление всех бит, кроме младших 42 бит
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var shortURL strings.Builder
	for num > 0 {
		remainder := num % 62
		shortURL.WriteString(string(chars[remainder]))
		num = num / 62
	}
	return shortURL.String()
}

package service

import (
	// "crypto/sha256"

	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const maskURL = "maskURL"

type MockRepo struct {
	mock.Mock
	logger *zap.SugaredLogger
}

func (m *MockRepo) Ping() error {
	return fmt.Errorf("")
}

func (m *MockRepo) Get(hash string, _ zap.SugaredLogger) (string, error) {
	args := m.Called(hash)
	return args.String(0), args.Error(1)
}

func (m *MockRepo) Set(url, hash string, _ zap.SugaredLogger) (string, error) {
	args := m.Called(url, hash)
	return args.String(0), args.Error(1)
}

func (m *MockRepo) SetArrayURL(_ []model.SetArrayURLRequest, _ zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	return make([]model.SetArrayURLResponse, 0), fmt.Errorf("")
}

func newWrapService() *Service {
	logger, _ := zap.NewDevelopment()
	r := MockRepo{
		logger: logger.Sugar(),
	}
	return &Service{
		repo: &r,
	}
}

func TestServiceSetURL(t *testing.T) {
	hashMask := sha256.Sum256([]byte(maskURL))
	hashEmpty := sha256.Sum256([]byte(""))
	tests := []struct {
		name     string
		url      string
		wantHash string
		wantErr  error
		repoIsOn bool
	}{
		{
			name:     "success set",
			url:      maskURL,
			wantHash: fmt.Sprintf("%x", hashMask[:sizeHash]),
			wantErr:  nil,
			repoIsOn: true,
		},
		{
			name:     "empty url",
			url:      "",
			wantHash: fmt.Sprintf("%x", hashEmpty[:sizeHash]),
			wantErr:  nil,
			repoIsOn: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ..1.
			s := newWrapService()
			// ..2.
			mockRepo := s.repo.(*MockRepo)
			if tt.repoIsOn {
				mockRepo.On("Set", tt.url, tt.wantHash).Return(tt.wantHash, tt.wantErr)
			}
			// ..3.
			gotHash, err := s.SetURL(tt.url, *mockRepo.logger)
			// ..4.
			assert.Equal(t, tt.wantHash, gotHash)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestServiceGetURL(t *testing.T) {
	hash := sha256.Sum256([]byte(maskURL))
	tests := []struct {
		name     string
		wantHash string
		wantURL  string
		wantErr  error
		repoIsOn bool
	}{
		{
			name:     "success url",
			wantHash: fmt.Sprintf("%x", hash[:sizeHash]),
			wantURL:  maskURL,
			wantErr:  nil,
			repoIsOn: true,
		},
		{
			name:     "empty url",
			wantHash: "",
			wantURL:  "",
			wantErr:  fmt.Errorf(" not found"),
			repoIsOn: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ..1.
			s := newWrapService()
			// ..2.
			mockRepo := s.repo.(*MockRepo)
			if tt.repoIsOn {
				mockRepo.On("Get", tt.wantHash).Return(tt.wantURL, tt.wantErr)
			}
			// ..3.
			gotURL, err := s.GetURL(tt.wantHash, *mockRepo.logger)
			// ..4.
			assert.Equal(t, tt.wantURL, gotURL)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

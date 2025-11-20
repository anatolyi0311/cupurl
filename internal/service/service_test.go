package service

import (
	// "crypto/sha256"

	"database/sql"
	"fmt"
	"testing"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/anatolyi0311/cupurl/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const maskURL = "maskURL"

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Ping() error {
	return fmt.Errorf("")
}

func newWrapService() *Service {
	// r := &MockRepo{}
	db := &sql.DB{}
	_ = db
	logger, _ := zap.NewDevelopment()

	rs, err := repository.NewStorage(&config.Config{Opts: &config.Options{StorageFile: ""}}, nil, *logger.Sugar())
	if err != nil {
		return &Service{
			&repository.Storage{},
		}
	}
	return &Service{
		repo: rs, // r
	}
}

func (m *MockRepo) Get(hash string, logger zap.SugaredLogger) (string, error) {
	args := m.Called(hash)
	return args.String(0), args.Error(1)
}

func (m *MockRepo) Set(url, hash string, logger zap.SugaredLogger) (string, error) {
	args := m.Called(url, hash)
	return "", args.Error(0)
}

func (m *MockRepo) SetArrayURL(_ []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	return make([]model.SetArrayURLResponse, 0), fmt.Errorf("")
}

func TestServiceSetURL(t *testing.T) {
	// hash := sha256.Sum256([]byte(maskURL))
	tests := []struct {
		name     string
		url      string
		wantHash string
		wantErr  error
		repoIsOn bool
		mockHash string
	}{
		// {
		// 	name:     "success set",
		// 	url:      maskURL,
		// 	wantHash: fmt.Sprintf("%x", hash[:sizeHash]),
		// 	wantErr:  nil,
		// 	repoIsOn: true,
		// 	mockHash: fmt.Sprintf("%x", hash[:sizeHash]),
		// },
		{
			name:     "empty url",
			url:      "",
			wantHash: "",
			wantErr:  nil, // fmt.Errorf("incorrect url"),
			repoIsOn: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newWrapService()
			// mockRepo := s.repo.(*MockRepo)
			// mockRepo := s.repo.(*repository.Storage)

			// if tt.repoIsOn {
			// 	mockRepo.On("Set", tt.url, tt.mockHash).Return(nil)
			// }
			logger, _ := zap.NewDevelopment()
			gotHash, err := s.SetURL(tt.url, *logger.Sugar())

			assert.Equal(t, tt.wantHash, gotHash)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestServiceGetURL(t *testing.T) {
	// hash := sha256.Sum256([]byte(maskURL))
	tests := []struct {
		name     string
		hash     string
		wantURL  string
		wantErr  error
		repoIsOn bool
		mockHash string
	}{
		// {
		// 	name:     "success get",
		// 	hash:     fmt.Sprintf("%x", hash[:sizeHash]),
		// 	wantURL:  maskURL,
		// 	wantErr:  nil,
		// 	repoIsOn: true,
		// 	mockHash: fmt.Sprintf("%x", hash[:sizeHash]),
		// },
		{
			name:     "empty url",
			hash:     "",
			wantURL:  "",
			wantErr:  fmt.Errorf(" not found"), // incorrect id.
			repoIsOn: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newWrapService()
			// mockRepo := s.repo.(*MockRepo)

			// if tt.repoIsOn {
			// 	mockRepo.On("Get", tt.mockHash).Return(tt.wantURL, nil)
			// }
			logger, _ := zap.NewDevelopment()
			gotURL, err := s.GetURL(tt.hash, *logger.Sugar())

			assert.Equal(t, tt.wantURL, gotURL)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/model"
)

// Mock для use case
type MockCaseURL struct {
	mock.Mock
}

func (m *MockCaseURL) Ping() error {
	return fmt.Errorf("")
}

func (m *MockCaseURL) GetArrayURL(_ zap.SugaredLogger) ([]model.GetArrayURLRequest, error) {
	args := m.Called()
	return []model.GetArrayURLRequest{model.GetArrayURLRequest{OriginalURL: args.String(0)}}, args.Error(1)
}

func (m *MockCaseURL) SetArrayURL(_ []model.SetArrayURLRequest, logger zap.SugaredLogger) ([]model.SetArrayURLResponse, error) {
	return make([]model.SetArrayURLResponse, 0), fmt.Errorf("")
}

func newWrapServer() *Server {
	logger, _ := zap.NewDevelopment()

	cfg := &config.Config{
		Opts: &config.Options{
			Addr:    "localhost:8080",
			BaseURL: "localhost:8080",
		},
	}
	cu := &MockCaseURL{}
	return &Server{
		cfg:    cfg,
		su:     cu,
		logger: *logger.Sugar(),
	}
}

func (m *MockCaseURL) GetURL(hash string, logger zap.SugaredLogger) (string, error) {
	args := m.Called(hash)
	return args.String(0), args.Error(1)
}

func (m *MockCaseURL) SetURL(url string, logger zap.SugaredLogger) (string, error) {
	args := m.Called(url)
	return args.String(0), args.Error(1)
}

func TestServerGetURL(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string

		isOn      bool
		mockHash  string
		mockURL   string
		mockError error

		expectedStatus   int
		expectedLocation string
	}{
		{
			name:   "successful redirect",
			method: http.MethodGet,
			path:   "/abc",

			isOn:      true,
			mockHash:  "abc",
			mockURL:   "https://practicum.yandex.ru/",
			mockError: nil,

			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://practicum.yandex.ru/",
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   "/abc",

			isOn:      true,
			mockHash:  "abc",
			mockURL:   "",
			mockError: fmt.Errorf("not found"),

			expectedStatus:   http.StatusBadRequest,
			expectedLocation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newWrapServer()

			mockUC := s.su.(*MockCaseURL)
			if tt.isOn {
				mockUC.On("GetURL", tt.mockHash).Return(tt.mockURL, tt.mockError)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			res := httptest.NewRecorder()

			s.GetURLHandler(res, req)
			assert.Equal(t, tt.expectedStatus, res.Code)
			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, res.Header().Get("Location"))
			}
			if tt.isOn {
				mockUC.AssertExpectations(t)
			}
		})
	}
}

func TestServerSetURL(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		contentType string

		isOn      bool
		mockURL   string
		mockHash  string
		mockError error

		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "successful post",
			method:      http.MethodPost,
			url:         "https://practicum.yandex.ru/",
			contentType: "text/plain",

			isOn:      true,
			mockURL:   "https://practicum.yandex.ru/",
			mockHash:  "abc",
			mockError: nil,

			expectedStatus: http.StatusCreated,
			expectedBody:   "localhost:8080/abc",
		},
		{
			name:        "bad content type",
			method:      http.MethodPost,
			url:         "https://practicum.yandex.ru/",
			contentType: "application/json",

			isOn:      false,
			mockURL:   "https://practicum.yandex.ru/",
			mockHash:  "abc",
			mockError: nil,

			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Content-Type must be text/plain\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newWrapServer()

			mockUC := s.su.(*MockCaseURL)
			if tt.isOn {
				mockUC.On("SetURL", tt.mockURL).Return(tt.mockHash, tt.mockError)
			}

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.url))
			req.Header.Set("Content-Type", tt.contentType)
			res := httptest.NewRecorder()

			s.SetURLHandler(res, req)
			assert.Equal(t, tt.expectedStatus, res.Code)
			assert.Equal(t, tt.expectedBody, res.Body.String())

			if tt.isOn {
				mockUC.AssertExpectations(t)
			}
		})
	}
}

func TestServerJSONHandler(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		contentType string

		isOn      bool
		mockURL   string
		mockHash  string
		mockError error

		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "successful post",
			method:      http.MethodPost,
			url:         "{\"url\":\"https://practicum.yandex.ru/\"}",
			contentType: "application/json",

			isOn:      true,
			mockURL:   "https://practicum.yandex.ru/",
			mockHash:  "abc",
			mockError: nil,

			expectedStatus: http.StatusCreated,
			expectedBody:   "{\"result\":\"localhost:8080/abc\"}",
		},
		{
			name:        "bad content type",
			method:      http.MethodPost,
			url:         "https://practicum.yandex.ru/",
			contentType: "text/plain",

			isOn:      false,
			mockURL:   "https://practicum.yandex.ru/",
			mockHash:  "abc",
			mockError: nil,

			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Content-Type must be application/json\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newWrapServer()

			mockUC := s.su.(*MockCaseURL)
			if tt.isOn {
				mockUC.On("SetURL", tt.mockURL).Return(tt.mockHash, tt.mockError)
			}

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.url))
			req.Header.Set("Content-Type", tt.contentType)
			res := httptest.NewRecorder()

			s.SetJSONHandler(res, req)
			assert.Equal(t, tt.expectedStatus, res.Code)
			assert.Equal(t, tt.expectedBody, res.Body.String())

			if tt.isOn {
				mockUC.AssertExpectations(t)
			}
		})
	}
}

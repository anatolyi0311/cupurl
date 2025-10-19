package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock для use case
type MockCaseURL struct {
	mock.Mock
}

func newWrapServer() *Server {
	cu := &MockCaseURL{}
	return &Server{
		su: cu,
	}
}

func (m *MockCaseURL) GetURL(hash string) (string, error) {
	args := m.Called(hash)
	return args.String(0), args.Error(1)
}

func (m *MockCaseURL) SetURL(url string) (string, error) {
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
			name:   "wrong method",
			method: http.MethodPost,
			path:   "/abc1234567890",

			expectedStatus: http.StatusBadRequest,
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

			s.GetURL(res, req)
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
			expectedBody:   "http://localhost:8080/abc",
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
		{
			name:        "invalid method",
			method:      http.MethodTrace,
			url:         "https://practicum.yandex.ru/",
			contentType: "text/plain",

			isOn:      false,
			mockURL:   "https://practicum.yandex.ru/",
			mockHash:  "abc",
			mockError: nil,

			expectedStatus: http.StatusBadRequest,
			expectedBody:   "method must be POST\n",
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

			s.SetURL(res, req)
			assert.Equal(t, tt.expectedStatus, res.Code)
			assert.Equal(t, tt.expectedBody, res.Body.String())

			if tt.isOn {
				mockUC.AssertExpectations(t)
			}
		})
	}
}

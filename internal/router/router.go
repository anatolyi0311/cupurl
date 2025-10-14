package router

import (
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// curl -X POST -H "Content-Type: text/plain" -d 'https://practicum.yandex.ru/' http://localhost:8080/
// curl -X GET -H "Content-Type: text/plain" -d '/EwHXdJfB' http://localhost:8080/{id}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // "0123456789"

func generateRandomString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	var result []byte
	for i := 0; i < length; i++ {
		index := seededRand.Intn(len(charset))
		result = append(result, charset[index])
	}
	return string(result)
}

// func GetOriginURL(key string) string {
// 	urls := map[string]string{
// 		"/EwHXdJfB": "https://practicum.yandex.ru/",
// 	}
// 	res, ok := urls[key]
// 	if !ok {
// 		urls[key] = "https://practicum.yandex.ru/"
// 		res = urls[key]
// 	}
// 	return res
// }

// func CopyModelURL()(int, error){
// 	return io.CopyBuffer())
// }

type ModelURL struct {
	s  map[string]string
	mu sync.Mutex
}

func (m *ModelURL) SetURL(customURL *url.URL, key string) {
	m.mu.Lock()
	m.s[key] = customURL.Host
	m.mu.Unlock()
}

func NewModelURL() *ModelURL {
	return &ModelURL{
		s: make(map[string]string),
	}
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	// продолжаем обработку запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	name := string(body) + ".txt"
	if len(string(body)) > 0 {
		name = string(body)[1:] + ".txt"
	}
	data, err := os.ReadFile(name)
	if err != nil {
		panic(err)
	}
	newPath, err := url.JoinPath(string(data), string(body))
	if err != nil {
		panic(err)
	}
	w.Header().Add("Location", newPath)
	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Write([]byte(newPath))
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	// продолжаем обработку запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	parseURL, err := r.URL.Parse(string(body))
	if err != nil {
		panic(err)
	}
	scheme := parseURL.Scheme
	if scheme == "https" || scheme == "" {
		scheme = "http"
	}
	newURL, err := url.Parse(scheme + ":/" + r.URL.JoinPath(r.Host, parseURL.Path).String())
	if err != nil {
		panic(err)
	}
	// data := make([]byte, 0)
	name := generateRandomString(6) + ".txt"
	err = os.WriteFile(name, []byte(newURL.String()), os.ModePerm)
	if err != nil {
		panic(err)
	}
	// modelURL := NewModelURL()
	// modelURL.SetURL(newURL, parseURL.Path)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(newURL.String()))
}

func Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, PostHandler)
	mux.HandleFunc(`/{id}`, GetHandler)
	return mux
}

package router

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
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
func GetOriginUrl(key string) string {
	urls := map[string]string{
		"/EwHXdJfB": "https://practicum.yandex.ru/",
	}
	res, ok := urls[key]
	if !ok {
		return ""
	}
	return res
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	// r.Header.Set("Content-Type", "text/plain")
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
	w.WriteHeader(307)
	w.Header().Add("Location", GetOriginUrl(r.URL.EscapedPath()))
	// w.Header().Set("Location", GetOriginUrl(r.URL.EscapedPath()))
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	res := fmt.Sprintf(
		"%s %d %s\r\nLocation: %s\r\n%s\n",
		r.Proto, http.StatusTemporaryRedirect, http.StatusText(307),
		GetOriginUrl(string(body)),
		r.URL.EscapedPath(),
	)
	w.Write([]byte(res))
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	// r.Header.Set("Content-Type", "text/plain")
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
	w.WriteHeader(201)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	// w.Write(body) // W([]byte(body))
	parseUrl, err := r.URL.Parse(string(body))
	_ = parseUrl
	if err != nil {
		panic(err)
	}
	scheme := parseUrl.Scheme
	if scheme == "https" {
		scheme = "http"
	}
	newUrl, err := url.Parse(scheme + ":/" + r.URL.JoinPath(r.Host, generateRandomString(len("EwHXdJfB"))).String())
	// w.Write([]byte(newUrl.String())) // W([]byte(body))
	w.Header().Set("Content-Length", strconv.Itoa(len(newUrl.String())))
	res := fmt.Sprintf(
		"%s %d %s\r\nContent-Type: %s\r\nContent-Length: %s\r\n\n%s\r\n\n",
		r.Proto, http.StatusCreated, http.StatusText(201),
		r.Header.Get("Content-Type"),
		w.Header().Get("Content-Length"),
		newUrl.String(),
	)
	w.Write([]byte(res))
}

func Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, PostHandler)
	mux.HandleFunc(`/{id}`, GetHandler)
	return mux
}

package router

import (
	"io"
	"net/http"
	"net/url"
)

// curl -X POST -H "Content-Type: text/plain" -d 'https://practicum.yandex.ru/' http://localhost:8080/
// curl -X GET -H "Content-Type: text/plain" -d '/EwHXdJfB' http://localhost:8080/{id}

// const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // "0123456789"

// func generateRandomString(length int) string {
// 	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	var result []byte
// 	for i := 0; i < length; i++ {
// 		index := seededRand.Intn(len(charset))
// 		result = append(result, charset[index])
// 	}
// 	return string(result)
// }

func GetOriginURL(key string) string {
	urls := map[string]string{
		"/EwHXdJfB": "https://practicum.yandex.ru/",
	}
	res, ok := urls[key]
	if !ok {
		urls[key] = "https://practicum.yandex.ru/"
		res = urls[key]
	}
	return res
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	// contentType := r.Header.Get("Content-Type")
	// if contentType != "text/plain" {
	// 	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	// 	return

	// }
	// продолжаем обработку запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	// w.Header().Set("Location", GetOriginURL(r.URL.EscapedPath()))
	w.Header().Add("Location", GetOriginURL(string(body)))
	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Write([]byte(GetOriginURL(string(body))))
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	if scheme == "https" {
		scheme = "http"
	}
	newURL, err := url.Parse(scheme + ":/" + r.URL.JoinPath(r.Host, parseURL.Path).String())
	if err != nil {
		panic(err)
	}
	// w.Header().Set("Content-Length", strconv.Itoa(len(newURL.String())))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(newURL.String()))
}

func Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, PostHandler)
	mux.HandleFunc(`/{id}`, GetHandler)
	return mux
}

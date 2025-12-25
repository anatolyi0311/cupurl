package handler

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

type responseData struct {
	status int
	size   int
}

func HandLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)
		logrus.Infoln(
			"uri:", r.RequestURI,
			"method:", r.Method,
			"duration:", duration,
			"status:", responseData.status,
			"size:", responseData.size,
		)
	})
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	if r.responseData.status == 0 {
		r.responseData.status = http.StatusOK
	}
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func (r *loggingResponseWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

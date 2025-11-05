package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	acceptsGzip bool
	wroteHeader bool
}

func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid gzip body", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			decompressed, err := io.ReadAll(gz)
			if err != nil {
				http.Error(w, "Error reading gzip body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decompressed))
			r.Header.Del("Content-Encoding")
			r.ContentLength = int64(len(decompressed))
		}

		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		if acceptsGzip {
			gw := &gzipResponseWriter{
				ResponseWriter: w,
				acceptsGzip:    acceptsGzip,
			}
			defer gw.close()

			next.ServeHTTP(gw, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	contentType := w.Header().Get("Content-Type")
	shouldCompress := w.acceptsGzip &&
		(contentType == "application/json" || contentType == "text/html")

	if shouldCompress {
		w.Header().Set("Content-Encoding", "gzip")
		w.writer = gzip.NewWriter(w.ResponseWriter)
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.writer != nil {
		return w.writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipResponseWriter) close() {
	if w.writer != nil {
		w.writer.Close()
	}
}

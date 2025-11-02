package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/samber/lo"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
	Reader io.Reader
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
}

func (w gzipWriter) Read(b []byte) (int, error) {
	// w.Reader будет отвечать за gzip-unpack, поэтому пишем в него
	return w.Reader.Read(b)
}

func GzipHandle(next http.Handler, sugar zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !lo.Contains([]string{"application/json", "text/plain"}, r.Header.Get("Content-Type")) {
				// если type не поддерживается, передаём управление
				// дальше без изменений
				sugar.Infoln(
					"no-zip-c-type",
				)
				next.ServeHTTP(w, r)
				return
			}
			// проверяем, что клиент поддерживает gzip-сжатие
			// это упрощённый пример. В реальном приложении следует проверять все
			// значения r.Header.Values("Accept-Encoding") и разбирать строку
			// на составные части, чтобы избежать неожиданных результатов
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				// если gzip не поддерживается, передаём управление
				// дальше без изменений
				sugar.Infoln(
					"no-zip-a-encoding",
				)
				next.ServeHTTP(w, r)
				return
			}

			// создаём gzip.Writer поверх текущего w
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				// io.WriteString(w, err.Error())
				sugar.Infoln(
					"no-zip-new",
				)
				next.ServeHTTP(w, r)
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")

			sugar.Infoln(
				"method", r.Method,
				"a-zip", r.Header.Get(`Accept-Encoding`),
				"c-zip", r.Header.Get(`Content-Encoding`),
				"content", r.Header.Get("Content-Type"),
			)

			// передаём обработчику страницы переменную типа gzipWriter для вывода данных
			next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
		}
		if r.Method == http.MethodGet {
			// // создаём *gzip.Reader, который будет читать тело запроса
			// // и распаковывать его
			// gz, err := gzip.NewReader(r.Body)
			// if err != nil {
			// 	http.Error(w, err.Error(), http.StatusInternalServerError)
			// 	return
			// }
			// // закрытие gzip-читателя опционально, так как все данные уже прочитаны и
			// // текущая реализация не требует закрытия, тем не менее лучше это делать -
			// // некоторые реализации могут рассчитывать на закрытие читателя
			// // gz.Close() не вызывает закрытия r.Body - это будет сделано позже, http-сервером
			// defer gz.Close()
			// // при чтении вернётся распакованный слайс байт
			// body, err := io.ReadAll(gz)
			// if err != nil {
			// 	http.Error(w, err.Error(), http.StatusInternalServerError)
			// 	return
			// }

			sugar.Infoln(
				"method", r.Method,
				"c-zip", r.Header.Get(`Content-Encoding`),
				"content", r.Header.Get("Content-Type"),
			)

			// var reader io.Reader
			if r.Header.Get(`Content-Encoding`) == `gzip` {
				gz, err := gzip.NewReader(r.Body)
				sugar.Infoln(
					"gzip.read", gz.Name,
				)
				if err != nil {
					next.ServeHTTP(w, r)
				}
				// reader = gz
				defer gz.Close()
				next.ServeHTTP(gzipWriter{ResponseWriter: w, Reader: gz}, r)
			} else {
				// reader = r.Body
				sugar.Infoln(
					"method", r.Method,
					"zip", nil,
				)
				next.ServeHTTP(w, r)
			}
		}
	})
}

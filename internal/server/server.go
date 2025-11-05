package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/handler"
	srv "github.com/anatolyi0311/cupurl/internal/service"
)

// const (
// 	addr = "localhost:8080"
// )

type ResultURL struct {
	Result string `json:"result" doc:"result"`
}

type URL struct {
	URL *string `json:"url"`
}

type Server struct {
	cfg   *config.Config
	route *chi.Mux
	su    srv.CaseURL
	sugar zap.SugaredLogger
}

func NewServer(cfg *config.Config, sugar zap.SugaredLogger) *Server {
	server := &Server{
		cfg:   cfg,
		route: chi.NewRouter(),
		su:    srv.NewService(),
		sugar: sugar,
	}
	server.router()
	return server
}

func (s *Server) router() {
	s.route.Post("/", handler.WithLogging(s.SetURL, s.sugar))
	s.route.Post("/api/shorten", handler.WithLogging(s.JSONHandler, s.sugar))
	s.route.Get("/{id}", handler.WithLogging(s.GetURL, s.sugar))
}

func (s *Server) Run() {
	s.sugar.Infow(
		"Starting server",
		"addr", s.cfg.Opts.Addr,
		"base", s.cfg.Opts.BaseURL,
	)
	if err := http.ListenAndServe(s.cfg.Opts.Addr, handler.Compress(s.route)); err != nil {
		log.Fatalln(err)
	}
}

func (s *Server) JSONHandler(w http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// id := req.URL.Query().Get("url")

	var addr URL
	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// десериализуем JSON в Visitor
	if err = json.Unmarshal(buf.Bytes(), &addr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := s.su.SetURL(string(*addr.URL))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := json.Marshal(ResultURL{Result: s.cfg.Opts.BaseURL + "/" + hash})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

func (s *Server) SetURL(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(res, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	hash, err := s.su.SetURL(string(body))
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(s.cfg.Opts.BaseURL + "/" + hash))
}

func (s *Server) GetURL(res http.ResponseWriter, req *http.Request) {
	pathURL := chi.URLParam(req, "id")
	if pathURL == "" {
		pathURL = req.URL.Path
	}
	hash := strings.TrimPrefix(pathURL, "/")

	url, err := s.su.GetURL(hash)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Location", url)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

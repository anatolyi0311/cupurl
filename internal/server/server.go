package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	// _ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/config/db"
	"github.com/anatolyi0311/cupurl/internal/handler"
	srv "github.com/anatolyi0311/cupurl/internal/service"
)

const (
	addr = "localhost:8080"
)

type ResultURL struct {
	Result string `json:"result" doc:"result"`
}

type URL struct {
	URL *string `json:"url"`
}

type Server struct {
	cfg    *config.Config
	route  *chi.Mux
	su     srv.CaseURL
	logger zap.SugaredLogger
	db     *sql.DB
}

func NewServer(cfg *config.Config, logger zap.SugaredLogger) (*Server, error) {
	// initial DB with sql.Open(driverName, dataSourceName string) (*DB, error)
	// addrDB := cfg.Opts.AddrDB
	// driverName := "pgx"
	// dataSourceName := fmt.Sprintf(
	// 	"host=%s user=%s password=%s dbname=%s sslmode=disable",
	// 	addrDB, `videos`, `userpassword`, `videos`,
	// )
	// db, err := sql.Open(driverName, dataSourceName)
	// if err != nil {
	// 	logger.Fatal(err)
	// }
	// defer db.Close()

	su, err := srv.NewService(cfg)
	if err != nil {
		return nil, err
	}
	server := &Server{
		cfg:    cfg,
		route:  chi.NewRouter(),
		su:     su,
		logger: logger,
		// db:     db,
	}
	server.router()
	return server, nil
}

func (s *Server) router() {
	s.route.Post("/", handler.WithLogging(s.SetURLHandler, s.logger))
	s.route.Post("/api/shorten", handler.WithLogging(s.JSONHandler, s.logger))
	s.route.Get("/{id}", handler.WithLogging(s.GetURLHandler, s.logger))
	s.route.Get("/ping", handler.WithLogging(s.PingDBHandler, s.logger))
}

func (s *Server) Run() {
	s.logger.Infow(
		"Starting server",
		"addr", s.cfg.Opts.Addr,
		"base", s.cfg.Opts.BaseURL,
		"addrDB", s.cfg.Opts.AddrDB,
		"hostDB", s.cfg.Opts.HostDB,
		"portDB", s.cfg.Opts.PortDB,
		"pathDB", s.cfg.Opts.PathDB,
		"sslmode", s.cfg.Opts.ParamsDB["sslmode"],
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

func (s *Server) SetURLHandler(res http.ResponseWriter, req *http.Request) {
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

func (s *Server) GetURLHandler(res http.ResponseWriter, req *http.Request) {
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

func (s *Server) PingDBHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "method must be Get", http.StatusBadRequest)
		return
	}

	db, err := db.InitPostgresDB(s.cfg, s.logger)
	if err != nil {
		s.logger.Infow(
			"PingDB",
			"addr", s.cfg.Opts.AddrDB,
			"msg", err.Error(),
		)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// // if err := s.db.PingContext(ctx); err != nil {
	// if err := db.PingContext(ctx); err != nil {
	// 	http.Error(res, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// // s.PingDBHandler(res, req)

	if err := db.Ping(); err != nil {
		s.logger.Infow(
			"PingDB",
			"addrDB", s.cfg.Opts.AddrDB,
			"msg", err.Error(),
		)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	s.logger.Infow(
		"PingDB",
		"addrDB", s.cfg.Opts.AddrDB,
	)
	res.WriteHeader(http.StatusOK)
}

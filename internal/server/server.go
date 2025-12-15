package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	// _ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"

	"github.com/anatolyi0311/cupurl/internal/config"
	"github.com/anatolyi0311/cupurl/internal/handler"
	"github.com/anatolyi0311/cupurl/internal/jwt"
	"github.com/anatolyi0311/cupurl/internal/model"
	srv "github.com/anatolyi0311/cupurl/internal/service"
	"github.com/anatolyi0311/cupurl/internal/service/crypto"
)

const (
	addr             = "localhost:8080"
	UserIDCookieName = "shortener-user-id"
)

type ServerHTTP interface {
	Run() error
	Shutdown() error
}

func New(config *config.Config, ipChecker srv.IPCheckerInterface, service *srv.Service, svr Server) (Server, error) {
	if config.Opts.EnableHTTPS {
		return Server{}, nil //NewHTTPS(config, ipChecker, service)
	} else {
		return svr, nil //NewHTTP(config, ipChecker, service, svr )
	}
}

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
	crypto crypto.Cryptographer // interface that we'll use to encrypt and decrypt values
}

func NewServer(cfg *config.Config, logger zap.SugaredLogger, db *sql.DB) (*Server, error) {
	su, err := srv.NewService(cfg, db, logger)
	if err != nil {
		return nil, err
	}
	server := &Server{
		cfg:    cfg,
		route:  chi.NewRouter(),
		su:     su,
		logger: logger,
	}
	server.router()
	return server, nil
}

func (s *Server) router() {
	// s.route.Use(handler.Compress)
	// s.route.Use(jwt.Cookies)

	s.route.Post("/", handler.WithLogging(s.SetURLHandler, s.logger))
	s.route.Post("/api/shorten", handler.WithLogging(s.SetJSONHandler, s.logger))
	s.route.Post("/api/shorten/batch", handler.WithLogging(s.SetArrayURLJson, s.logger))
	s.route.Get("/{id}", handler.WithLogging(s.GetURLHandler, s.logger))
	s.route.Get("/ping", handler.WithLogging(s.PingDB, s.logger))
	s.route.Get("/api/user/urls", handler.WithLogging(s.GetArrayURLJson, s.logger))
	// s.route.Get("/api/internal/stats", handler.WithLogging(s.Stats, s.logger))
	s.route.Delete("/api/user/urls", handler.WithLogging(s.DeleteArrayURLJson, s.logger))
}

func (s *Server) Run() {
	s.logger.Infow(
		"Starting server",
		"addr", s.cfg.Opts.Addr,
		"base", s.cfg.Opts.BaseURL,
		"addrDB", s.cfg.Opts.AddrDB,
		"key", s.cfg.Opts.EncryptionKey,
	)
	// httpServer := &http.Server{
	// 	Addr:              s.cfg.Opts.Addr,
	// 	Handler:           s.route,
	// 	ReadHeaderTimeout: 1 * time.Second,
	// }
	if err := http.ListenAndServe(s.cfg.Opts.Addr, handler.Compress(jwt.Cookies(s.route))); err != nil {
		s.logger.Warn("err", err.Error())
		s.logger.Fatalln(err)
	}
}

func (s *Server) SetJSONHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "method must be POST", http.StatusBadRequest)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var request model.SetURLJsonRequest
	if err = json.Unmarshal(body, &request); err != nil {
		s.logger.Errorln(err)
		http.Error(res, "cannot unmarshal body", http.StatusBadRequest)
		return
	}

	hash, err := s.su.SetURL(request.URL)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			hashJSON := model.SetURLJsonResponse{
				URL: s.cfg.Opts.BaseURL + "/" + hash.ShortURL,
			}
			response, _ := json.Marshal(hashJSON)
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusConflict)
			res.Write(response)
			return
		}
		s.logger.Errorln(err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	hashJSON := model.SetURLJsonResponse{
		URL: s.cfg.Opts.BaseURL + "/" + hash.ShortURL,
	}

	response, err := json.Marshal(hashJSON)
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, "cannot marshal body", http.StatusBadRequest)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(response)
}

func (s *Server) SetURLHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "method must be POST", http.StatusBadRequest)
		return
	}

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
		if errors.Is(err, model.ErrURLAlreadyExists) {
			res.Header().Set("Content-Type", "text/plain")
			res.WriteHeader(http.StatusConflict)
			res.Write([]byte(s.cfg.Opts.BaseURL + "/" + hash.ShortURL))
			return
		}
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(s.cfg.Opts.BaseURL + "/" + hash.ShortURL))
}

func (s *Server) GetURLHandler(res http.ResponseWriter, req *http.Request) {
	pathURL := chi.URLParam(req, "id")
	if pathURL == "" {
		pathURL = req.URL.Path
	}
	hash := strings.TrimPrefix(pathURL, "/")

	url, err := s.su.GetURL(hash)
	if err != nil {
		if errors.Is(err, model.ErrDeletedURL) {
			http.Error(res, err.Error(), http.StatusGone)
			return
		}
		s.logger.Error(err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	// s.logger.Info("GetURL.url: ", url)

	res.Header().Set("Location", url.OriginalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (s *Server) SetArrayURLJson(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "method must be POST", http.StatusBadRequest)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var request []model.SetArrayURLRequest
	if err = json.Unmarshal(body, &request); err != nil {
		s.logger.Errorln(err)
		http.Error(res, "cannot unmarshal body", http.StatusBadRequest)
		return
	}

	result, err := s.su.SetArrayURL(request)
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := json.Marshal(result)
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, "cannot marshal body", http.StatusBadRequest)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(response)
}

func (s *Server) GetArrayURLJson(res http.ResponseWriter, req *http.Request) {
	s.logger.Info("GetArrayURLJson: ", req.Method)

	if req.Method != http.MethodGet {
		http.Error(res, "method must be GET", http.StatusBadRequest)
		return
	}

	_, err := jwt.GetUserID(req)
	if err != nil {
		s.logger.Warn("GetArrayURLJson.err.ID: ", req.Method)
		http.Error(res, err.Error(), http.StatusNoContent)
		return
	}

	contentType := req.Header.Get("Content-Type")
	// if contentType != "application/json" {
	// 	http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
	// 	return
	// }
	s.logger.Info("GetArrayURLJson.contentType: ", contentType)

	result, err := s.su.GetArrayURL()
	s.logger.Info("GetArrayURLJson.result: ", result)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(res, err.Error(), http.StatusNoContent)
			return
		}
		s.logger.Errorln(err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := json.Marshal(result)
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, "cannot marshal body", http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(response)
}

func (s *Server) DeleteArrayURLJson(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		http.Error(res, "method must be DELETE", http.StatusBadRequest)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var hashArray []string
	if err = json.Unmarshal(body, &hashArray); err != nil {
		http.Error(res, "cannot unmarshal body", http.StatusBadRequest)
		return
	}

	_, err = jwt.GetUserID(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusNoContent)
		return
	}

	s.su.DeleteArrayURL(hashArray)

	res.WriteHeader(http.StatusAccepted)
}

// func (s *Server) Stats(w http.ResponseWriter, r *http.Request) {
// 	stats, err := s.su.GetStats(r.Context())
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	s.logger.Info("start.Stats", stats)
// 	out, err := json.Marshal(stats)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	if _, err = w.Write(out); err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

func (s *Server) PingDB(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "method must be Get", http.StatusBadRequest)
		return
	}

	status := http.StatusOK
	err := s.su.Ping()
	if err != nil {
		status = http.StatusInternalServerError
	}
	res.WriteHeader(status)
}

func (s *Server) FormatShortURL(hash string) string {
	return s.cfg.Opts.BaseURL + "/" + hash
}

// func validateRequest(req *http.Request) (int, error) {
// 	if req.Method != http.MethodDelete {
// 		return http.StatusBadRequest, errors.New("method must be DELETE")
// 	}
// 	if req.Header.Get("Content-Type") != "application/json" {
// 		return http.StatusBadRequest, errors.New("Content-Type must be application/json")
// 	}
// 	return http.StatusOK, nil
// }

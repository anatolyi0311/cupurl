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
)

const (
	addr             = "localhost:8080"
	UserIDCookieName = "shortener-user-id"
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
	db     *sql.DB
	logger zap.SugaredLogger
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
		db:     db,
		logger: logger,
	}
	server.router()
	return server, nil
}

func (s *Server) router() {
	s.route.Use(handler.HandLogger)
	s.route.Use(handler.Compress)
	s.route.Use(jwt.Cookies)
	// s.route.Use(handler.MiddlewareAuthPublic)

	s.route.Post("/", s.SetURLHandler)
	s.route.Post("/api/shorten", s.SetJSONHandler)
	s.route.Post("/api/shorten/batch", s.SetArrayURLJson)
	s.route.Get("/{id}", s.GetURLHandler)
	s.route.Get("/ping", s.PingDB)
	s.route.Get("/api/user/urls", s.GetArrayURLJson)
	// s.route.Get("/api/internal/stats", handler.WithLogging(s.Stats, s.logger))
	s.route.Delete("/api/user/urls", s.DeleteArrayURLJson)
	// privateRoutes := s.route.Group(func(r chi.Router) {
	// 	return chi.NewRouter()
	// })
	// privateRoutes.DELETE("/api/user/urls", myHandler.DelUserURLS)
}

func (s *Server) Run() {
	s.logger.Infow(
		"Starting server",
		"addr", s.cfg.Opts.Addr,
		"base", s.cfg.Opts.BaseURL,
		"addrDB", s.cfg.Opts.AddrDB,
	)
	// httpServer := &http.Server{
	// 	Addr:              s.cfg.Opts.Addr,
	// 	Handler:           s.route,
	// 	ReadHeaderTimeout: 1 * time.Second,
	// }
	if err := http.ListenAndServe(s.cfg.Opts.Addr, s.route); err != nil {
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

	userID, err := jwt.GetUserID(req)
	if err != nil {
		userID = 1
	}
	// s.logger.Info("SetJSONHandler.userID", userID, " body: ", string(body))

	hash, err := s.su.SetURL(request.URL, userID)
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

	userID, err := jwt.GetUserID(req)
	if err != nil {
		userID = 1
	}
	// s.logger.Info("SetURLHandler.userID", userID, " body: ", string(body))

	hash, err := s.su.SetURL(string(body), userID)
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
	// s.logger.Info("get.url")

	pathURL := chi.URLParam(req, "id")
	if pathURL == "" {
		pathURL = req.URL.Path
	}
	hash := strings.TrimPrefix(pathURL, "/")

	// userID, err := jwt.GetUserID(req)
	// if err != nil {
	// 	userID = 1
	// }
	// s.logger.Info("GetURLHandler.userID: ", userID, " hash: ", hash)
	req.AddCookie(&http.Cookie{Name: "jwt_token"})
	userID, err := jwt.GetUserID(req)
	if err != nil {
		s.logger.Warn("not user", err.Error())
		// http.Error(res, err.Error(), http.StatusNoContent)
		// return
	}

	url, err := s.su.GetURL(hash, userID)
	if err != nil {
		if errors.Is(err, model.ErrDeletedURL) {
			http.Error(res, err.Error(), http.StatusGone)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(res, err.Error(), http.StatusGone)
			return
		}
		s.logger.Error(err)
		http.Error(res, err.Error(), http.StatusGone)
		return
	}
	// s.logger.Info("get.url: ", url)

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

	userID, err := jwt.GetUserID(req)
	if err != nil {
		userID = 1
	}
	// s.logger.Info("SetArrayURLJson.userID", userID)

	result, err := s.su.SetArrayURL(request, userID)
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
	if req.Method != http.MethodGet {
		http.Error(res, "method must be GET", http.StatusBadRequest)
		return
	}
	// contentType := req.Header.Get("Content-Type")
	// if contentType != "application/json" {
	// 	http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
	// 	return
	// }

	req.AddCookie(&http.Cookie{Name: "jwt_token"})
	userID, err := jwt.GetUserID(req)
	if err != nil {
		s.logger.Warn("not user ", err.Error())
		// http.Error(res, err.Error(), http.StatusNoContent)
		// return
	}
	s.cfg.Opts.User = userID
	// s.logger.Info("GetArrayURLJson.userID: ", userID)

	result, err := s.su.GetArrayURL()

	if err != nil {
		if errors.Is(err, model.ErrDeletedURL) {
			http.Error(res, err.Error(), http.StatusNoContent)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(res, err.Error(), http.StatusNoContent)
			return
		}
		s.logger.Errorln(err)
		http.Error(res, err.Error(), http.StatusNoContent)
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

	userID, err := jwt.GetUserID(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusNoContent)
		return
	}
	s.cfg.Opts.User = userID
	s.logger.Info("DeleteArrayURLJson.userID: ", userID, "...", s.cfg.Opts.User)

	// go
	s.su.DeleteArrayURL(hashArray, userID)
	// s.su.DeleteUrls(context.Background(), hashArray, userID)

	res.WriteHeader(http.StatusAccepted)
}

func (s *Server) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.su.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.logger.Info("start.Stats", stats)
	out, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(out); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

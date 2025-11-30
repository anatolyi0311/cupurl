package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
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
	// db     *sql.DB
	cookie string
}

func NewServer(cfg *config.Config, logger zap.SugaredLogger, db *sql.DB) (*Server, error) {
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

	su, err := srv.NewService(cfg, db, logger)
	if err != nil {
		return nil, err
	}
	userID := 1
	setJWT, err := jwt.SetJWT(userID, logger)
	if err != nil {
		return nil, err
	}
	server := &Server{
		cfg:    cfg,
		route:  chi.NewRouter(),
		su:     su,
		logger: logger,
		// db:     db,
		cookie: setJWT,
	}
	server.router()
	return server, nil
}

func (s *Server) router() {
	s.route.Post("/", handler.WithLogging(s.SetURLHandler, s.logger))
	s.route.Post("/api/shorten", handler.WithLogging(s.SetJSONHandler, s.logger))
	s.route.Get("/{id}", handler.WithLogging(s.GetURLHandler, s.logger))
	s.route.Get("/ping", handler.WithLogging(s.PingDB, s.logger))
	s.route.Post("/api/shorten/batch", handler.WithLogging(s.SetArrayURLJson, s.logger))
	s.route.Get("/api/user/urls", handler.WithLogging(s.GetArrayURLJson, s.logger))
}

func (s *Server) Run() {
	s.logger.Infow(
		"Starting server",
		"addr", s.cfg.Opts.Addr,
		"base", s.cfg.Opts.BaseURL,
		"addrDB", s.cfg.Opts.AddrDB,
		"hostDB", s.cfg.Opts.HostDB,
		"portDB", s.cfg.Opts.PortDB,
		"userID", jwt.GetUserID(s.cookie, s.logger),
	)
	if err := http.ListenAndServe(s.cfg.Opts.Addr, handler.Compress(s.route)); err != nil {
		log.Fatalln(err)
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

	// id := req.URL.Query().Get("url")

	// var addr URL
	// var buf bytes.Buffer
	// // читаем тело запроса
	// _, err := buf.ReadFrom(req.Body)
	// if err != nil {
	// 	http.Error(res, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	// // десериализуем JSON в Visitor
	// if err = json.Unmarshal(buf.Bytes(), &addr); err != nil {
	// 	http.Error(res, err.Error(), http.StatusBadRequest)
	// 	return
	// }

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

	// hash, err := s.su.SetURL(string(*addr.URL), s.logger)
	hash, err := s.su.SetURL(request.URL, s.logger)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			hashJSON := model.SetURLJsonResponse{
				URL: s.cfg.Opts.BaseURL + "/" + hash,
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

	resp, err := json.Marshal(ResultURL{Result: s.cfg.Opts.BaseURL + "/" + hash})
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(resp)
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

	status := http.StatusCreated

	hash, err := s.su.SetURL(string(body), s.logger)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			status = http.StatusConflict
		}
		if status != http.StatusConflict {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// s.logger.Info("SetURL.shortHash: ", hash, string(body))
	// cookie := &http.Cookie{
	// 	HttpOnly: true,
	// 	Value:    s.cookie,
	// 	Name:     "access_token",
	// }
	// http.SetCookie(res, cookie)
	res.Header().Set("Authorization", "Bearer "+s.cookie)

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(status)
	res.Write([]byte(s.cfg.Opts.BaseURL + "/" + hash))
}

func (s *Server) GetURLHandler(res http.ResponseWriter, req *http.Request) {
	// authorization := req.Header.Get("Authorization")
	// if authorization != "" && authorization != "Bearer "+s.cookie {
	// 	http.Error(res, "authorization fail", http.StatusUnauthorized)
	// 	return
	// }

	pathURL := chi.URLParam(req, "id")
	if pathURL == "" {
		pathURL = req.URL.Path
	}
	hash := strings.TrimPrefix(pathURL, "/")

	url, err := s.su.GetURL(hash, s.logger)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// coocies := req.Cookies()
	// for _, coocie := range coocies {
	// 	if coocie.Name == "access_token" && coocie.Value != s.cookie {
	// 		http.Error(res, strconv.Itoa(jwt.GetUserID(s.cookie, s.logger)), http.StatusUnauthorized)
	// 		return
	// 	}
	// }
	// s.logger.Info("GetURL.url: ", url)

	res.Header().Set("Location", url)
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
		// logger.Errorln(err)
		http.Error(res, "cannot unmarshal body", http.StatusBadRequest)
		return
	}

	result, err := s.su.SetArrayURL(request, s.logger)
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

	// s.logger.Info("SetArrayURLJson.result: ", result)
	// cookie := &http.Cookie{
	// 	HttpOnly: true,
	// 	Value:    s.cookie,
	// 	Name:     "access_token",
	// }
	// http.SetCookie(res, cookie)
	// res.Header().Set("Authorization", "Bearer "+s.cookie)

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

	result, err := s.su.GetArrayURL(s.logger)
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

	cookie := &http.Cookie{
		HttpOnly: true,
		Value:    s.cookie,
		Name:     "access_token",
	}
	http.SetCookie(res, cookie)
	// res.Header().Set("Authorization", "Bearer "+s.cookie)

	// coocies := req.Cookies()
	// for _, coocie := range coocies {
	// 	if coocie.Name == "access_token" && coocie.Value != s.cookie {
	// 		http.Error(res, strconv.Itoa(jwt.GetUserID(s.cookie, s.logger)), http.StatusUnauthorized)
	// 		return
	// 	}
	// }

	status := http.StatusOK
	authorization := req.Header.Get("Authorization")
	if authorization != "" && authorization != "Bearer "+s.cookie || len(response) == 0 {
		status = http.StatusNoContent
	}
	cooc := req.Cookies()
	if len(cooc) == 0 {
		status = http.StatusNoContent
	}
	for _, c := range cooc {
		if c.Value != s.cookie {
			status = http.StatusNoContent
		}
	}

	s.logger.Info("GetArrayURLJson.result: ", result)
	s.logger.Info("GetArrayURLJson.resCookiesult: ", cooc)

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(response)
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

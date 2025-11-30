package server

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	jwt string
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
	setJWT, err := jwt.SetJWT(cfg.Opts.SecretKey, userID, logger)
	if err != nil {
		return nil, err
	}
	server := &Server{
		cfg:    cfg,
		route:  chi.NewRouter(),
		su:     su,
		logger: logger,
		// db:     db,
		jwt: setJWT,
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
	s.route.Delete("/api/user/urls", handler.WithLogging(s.DeleteArrayURLJson, s.logger))
}

func (s *Server) Run() {
	s.logger.Infow(
		"Starting server",
		"addr", s.cfg.Opts.Addr,
		"base", s.cfg.Opts.BaseURL,
		"addrDB", s.cfg.Opts.AddrDB,
		// "hostDB", s.cfg.Opts.HostDB,
		// "portDB", s.cfg.Opts.PortDB,
		"userID", jwt.GetUserID(s.cfg.Opts.SecretKey, s.jwt, s.logger),
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
	hash, err := s.su.SetURL(request.URL)
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

	hash, err := s.su.SetURL(string(body))
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			status = http.StatusConflict
		}
		if status != http.StatusConflict {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// DEBUG.
	if s.cfg.Opts.Debug {
		authorization := req.Header.Get("Authorization")
		coocies := req.Cookies()
		s.logger.Info("SetURL.hash: ", hash, " body:", string(body), " coocies:", coocies, " authorization:", authorization)
	}

	// ...
	cookie := &http.Cookie{
		HttpOnly: true,
		Value:    s.jwt,
		Name:     "access_token",
	}
	http.SetCookie(res, cookie)

	res.Header().Set("Authorization", "Bearer "+s.jwt)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(status)
	res.Write([]byte(s.FormatURL(hash)))
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

	url, err := s.su.GetURL(hash)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	status := http.StatusTemporaryRedirect
	if url.DeletedFlag {
		status = http.StatusGone
	}
	// MY-FIX
	// if url.is_deleted == "" {
	// 	status = http.StatusGone
	// }

	// coocies := req.Cookies()
	// for _, coocie := range coocies {
	// 	if coocie.Name == "access_token" && coocie.Value != s.cookie {
	// 		http.Error(res, strconv.Itoa(jwt.GetUserID(s.cookie, s.logger)), http.StatusUnauthorized)
	// 		return
	// 	}
	// }
	// s.logger.Info("GetURL.url: ", url)

	res.Header().Set("Location", url.OriginalURL)
	res.WriteHeader(status)
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
	contentType := req.Header.Get("Content-Type")
	// if contentType != "application/json" {
	// 	http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
	// 	return
	// }

	result, err := s.su.GetArrayURL()
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	// s.logger.Info("GetArrayURLJson.result: ", result)

	response, err := json.Marshal(result)
	if err != nil {
		s.logger.Errorln(err)
		http.Error(res, "cannot marshal body", http.StatusBadRequest)
		return
	}

	status := http.StatusOK
	// for _, r := range result {
	// 	s.logger.Info("hash: ", r.Hash)
	// 	if r.Hash == "" || r.ShortURL == "" {
	// 		status = http.StatusNoContent
	// 	}
	// }

	// AUTH.
	authorization := req.Header.Get("Authorization")
	if authorization == "" && authorization != "Bearer "+s.jwt {
		status = http.StatusNoContent
	}
	// DEBUG.
	if s.cfg.Opts.Debug {
		coocies := req.Cookies()
		if len(coocies) > 0 {
			for _, coocie := range coocies {
				userID := jwt.GetUserID(s.cfg.Opts.SecretKey, s.jwt, s.logger)
				if coocie.Name == "access_token" && coocie.Value != s.jwt && userID != 1 {
					http.Error(res, strconv.Itoa(userID), http.StatusUnauthorized)
				}
			}
		}
		s.logger.Info("GetArrayURLJson.coocies: ", coocies, " authorization:", authorization, " contentType:", contentType)
	}

	// ...
	cookie := &http.Cookie{
		HttpOnly: true,
		Value:    s.jwt,
		Name:     "access_token",
	}
	http.SetCookie(res, cookie)

	// ...
	res.Header().Set("Authorization", "Bearer "+s.jwt)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(response)
}

func (s *Server) DeleteArrayURLJson(w http.ResponseWriter, r *http.Request) {
	if status, err := validateRequest(r); err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	var ids []string

	reader, err := getDecompressedReader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if errDecode := json.NewDecoder(reader).Decode(&ids); errDecode != nil {
		http.Error(w, "cannot decode json", http.StatusBadRequest)
		return
	}

	status := http.StatusAccepted

	// go s.su.DeleteArrayURL(context.Background(), ids, strconv.Itoa(userID))
	authorization := r.Header.Get("Authorization")
	go func(authorization string) {
		if authorization != "" && authorization == "Bearer "+s.jwt {
			userID := jwt.GetUserID(s.cfg.Opts.SecretKey, s.jwt, s.logger)
			s.logger.Info("authorization: ", authorization)
			// s.logger.Info("userID: ", userID)
			// s.logger.Info("ids: ", len(ids), ids)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			s.su.DeleteArrayURL(ctx, ids, strconv.Itoa(userID))
			status = http.StatusAccepted
		}
	}(authorization)
	// s.logger.Info("GetArrayURLJson.result: ", result)

	// AUTH.
	// authorization := req.Header.Get("Authorization")
	// if authorization == "" && authorization != "Bearer "+s.jwt {
	// }

	w.WriteHeader(status)
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

func (s *Server) FormatURL(hash string) string {
	return s.cfg.Opts.BaseURL + "/" + hash
}

func validateRequest(req *http.Request) (int, error) {
	if req.Method != http.MethodDelete {
		return http.StatusBadRequest, errors.New("method must be DELETE")
	}
	if req.Header.Get("Content-Type") != "application/json" {
		return http.StatusBadRequest, errors.New("Content-Type must be application/json")
	}
	return http.StatusOK, nil
}

func getDecompressedReader(r *http.Request) (io.Reader, error) {
	if r.Header.Get("Content-Encoding") == "gzip" {
		return gzip.NewReader(r.Body)
	}
	return r.Body, nil
}
